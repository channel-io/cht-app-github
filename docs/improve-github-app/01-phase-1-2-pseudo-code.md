# GitHub App 개선 Phase 1 + 2 Pseudo Code

리뷰용 문서.
문서에서 Anchor는 GitHub에 달리는 코멘트를 뜻한다.

## 1. EventWorker

```go
func (w *EventWorker) ProcessPacket(ctx context.Context, msg QueuedPacket) error {
    pkt := msg.Packet
    subjectKey, ok := computeSubjectKey(pkt)
    if !ok {
        return w.queue.Ack(ctx, msg.StreamID)
    }

    return w.locker.WithSubjectLock(ctx, subjectKey, func() error {
        state, err := w.state.Load(ctx, subjectKey)
        if err != nil {
            return err
        }

        if hasDispatchableRoot(state) {
            return w.dispatch.DispatchCurrent(ctx, msg, state, pkt)
        }

        if requiresExistingRoot(pkt) {
            rebuilt, ok, err := w.tryResolveExistingRoot(ctx, subjectKey, pkt)
            if err != nil {
                return err
            }
            if ok {
                return w.dispatch.DispatchCurrent(ctx, msg, rebuilt, pkt)
            }
            return w.queue.Ack(ctx, msg.StreamID)
        }

        if rebuilt, ok, err := w.tryLazyRebuildFromVerifiedAnchor(ctx, subjectKey, pkt); err != nil {
            return err
        } else if ok {
            return w.dispatch.DispatchCurrent(ctx, msg, rebuilt, pkt)
        }

        if shouldBufferUntilRoot(pkt, state, w.clock.Now()) {
            _, err := w.preRoot.AppendIfNewDelivery(ctx, subjectKey, pkt.DeliveryID, pkt)
            if err != nil {
                return err
            }

            waiting := ensureWaitingRootState(subjectKey, state, w.clock.Now())
            if err := w.state.Save(ctx, waiting); err != nil {
                return err
            }
            return w.queue.Ack(ctx, msg.StreamID)
        }

        if shouldMarkWaitingRootExpired(state, w.clock.Now()) {
            expired := ensureWaitingRootExpiredState(subjectKey, state, w.clock.Now())
            if err := w.state.Save(ctx, expired); err != nil {
                return err
            }
            state = expired
        }

        root, ok, err := w.tryCreateRootIfEligible(ctx, subjectKey, state, pkt)
        if err != nil {
            return err
        }
        if !ok {
            return w.queue.Ack(ctx, msg.StreamID)
        }

        if err := w.flushBufferedPreRootPackets(ctx, root); err != nil {
            return err
        }

        if !isOpened(pkt) {
            return w.dispatch.DispatchCurrent(ctx, msg, root, pkt)
        }

        root.LastProcessedAt = w.clock.Now()
        root.LastProcessedID = pkt.DeliveryID
        if err := w.state.Save(ctx, root); err != nil {
            return err
        }
        return w.queue.Ack(ctx, msg.StreamID)
    })
}
```

- dispatchable root가 있으면 바로 root-ready dispatch path로 보낸다.
- `computeSubjectKey`는 final subject를 정할 수 없는 packet에 대해 `(subjectKey, false)` 를 반환할 수 있다.
- 현재 기준으로 `status` / `check_run`은 merged PR association miss 시 여기서 바로 종료한다.
- root가 없고 기존 root가 반드시 필요한 packet이면 current behavior와 동일한 existing-root lookup / bounded retry를 먼저 시도하고, 그 lookup이 실패한 경우에만 종료한다.
- root가 없으면 verified Anchor -> pre-root queue -> `waiting_root_expired` -> eligible root 생성 순서로 내려간다.
- pre-root append는 delivery 단위로 idempotent 해야 한다.
- 같은 stream message가 worker retry로 재처리돼도 동일 `deliveryID` 는 한 번만 buffer append 되고, `waiting_root` state는 재시도에서 항상 복구 가능해야 한다.
- `DispatchCurrent` 는 Phase 1/2에서 downstream write-side ledger를 두지 않는 best-effort dispatch seam이다.
- 따라서 downstream thread write 성공 후 ack/state 저장 전에 worker가 중단되면 동일 `deliveryID` 의 thread message가 중복 전송될 수 있다.
- `opened`가 wait window 안에 오면 그 packet으로 normal root를 만든다.
- wait window가 지나면 즉시 synthetic fallback root를 만들지 않고 `waiting_root_expired` 로 전이하되, 그 상태 전이를 만든 현재 packet은 ack하지 않고 이어서 rebuild/fallback 판단에 사용한다.
- 이후 같은 subject의 새 packet이 오면 lazy rebuild를 먼저 시도하고, 그마저 실패한 경우에만 fallback root를 한 번 생성한다.
- root가 생기면 buffered packet과 current packet을 같은 dispatch seam으로 흘린다.
