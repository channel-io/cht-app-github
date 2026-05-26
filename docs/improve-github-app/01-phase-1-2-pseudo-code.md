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
            resolved, err := w.tryResolveExistingRoot(ctx, subjectKey, pkt)
            if err != nil {
                return err
            }
            switch resolved.Kind {
            case ExistingRootFound:
                return w.dispatch.DispatchCurrent(ctx, msg, resolved.State, pkt)
            case ExistingRootRetryLater:
                return ErrRetryLater
            case ExistingRootFinalMiss:
                return w.queue.Ack(ctx, msg.StreamID)
            }
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

            waitingPatch := buildWaitingRootPatch(subjectKey, state, w.clock.Now())
            if _, err := w.state.Patch(ctx, subjectKey, state.Version, waitingPatch); err != nil {
                if errors.Is(err, ErrVersionConflict) {
                    return ErrRetryLater
                }
                return err
            }
            return w.queue.Ack(ctx, msg.StreamID)
        }

        if shouldMarkWaitingRootExpired(state, w.clock.Now()) {
            expiredPatch := buildWaitingRootExpiredPatch(subjectKey, state, w.clock.Now())
            expired, err := w.state.Patch(ctx, subjectKey, state.Version, expiredPatch)
            if err != nil {
                if errors.Is(err, ErrVersionConflict) {
                    return ErrRetryLater
                }
                return err
            }
            state = expired
        }

        root, ok, err := w.tryClaimAndCreateRootIfEligible(ctx, subjectKey, state, pkt)
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

        if err := w.state.MarkOpenedProcessed(ctx, root.SubjectKey, root.Version, pkt.DeliveryID, w.clock.Now()); err != nil {
            if errors.Is(err, ErrVersionConflict) {
                return ErrRetryLater
            }
            return err
        }
        return w.queue.Ack(ctx, msg.StreamID)
    })
}
```

- dispatchable root가 있으면 바로 root-ready dispatch path로 보낸다.
- `computeSubjectKey`는 final subject를 정할 수 없는 packet에 대해 `(subjectKey, false)` 를 반환할 수 있다.
- 현재 기준으로 `status` / `check_run`은 merged PR association miss 시 여기서 바로 종료한다.
- root가 없고 기존 root가 반드시 필요한 packet이면 current behavior와 동일한 existing-root lookup / bounded retry를 먼저 시도한다.
- existing-root lookup 결과는 `Found`, `RetryLater`, `FinalMiss` 로 구분한다. `FinalMiss` 만 ack/drop 한다.
- root가 없으면 verified Anchor -> pre-root queue -> `waiting_root_expired` -> eligible root 생성 순서로 내려간다.
- pre-root append는 delivery 단위로 idempotent 해야 한다.
- 같은 stream message가 worker retry로 재처리돼도 동일 `deliveryID` 는 한 번만 buffer append 되고, `waiting_root` state는 재시도에서 항상 복구 가능해야 한다.
- subject state update는 blind `Save` 가 아니라 `version` 기반 patch로만 수행한다.
- `ErrVersionConflict` 는 정상적인 stale worker 신호이며, stream ack 없이 retry/backoff 한다.
- `DispatchCurrent` 는 Phase 1/2에서 downstream write-side ledger를 두지 않는 best-effort dispatch seam이다.
- 따라서 downstream thread write 성공 후 ack/state 저장 전에 worker가 중단되면 동일 `deliveryID` 의 thread message가 중복 전송될 수 있다.
- `opened`가 wait window 안에 오면 그 packet으로 normal root를 만든다.
- wait window가 지나면 즉시 synthetic fallback root를 만들지 않고 `waiting_root_expired` 로 전이하되, 그 상태 전이를 만든 현재 packet은 ack하지 않고 이어서 rebuild/fallback 판단에 사용한다.
- 이후 같은 subject의 새 packet이 오면 lazy rebuild를 먼저 시도하고, 그마저 실패한 경우에만 fallback root를 한 번 생성한다.
- root가 생기면 buffered packet과 current packet을 같은 dispatch seam으로 흘린다.

## 2. Root claim CAS

`lock:subject:{subjectKey}` 는 동시 실행을 줄이는 lease다.
root 확정은 아래 root claim CAS가 담당한다.

```go
func (w *EventWorker) tryClaimAndCreateRootIfEligible(
    ctx context.Context,
    subjectKey string,
    state SubjectState,
    pkt WebhookPacket,
) (SubjectState, bool, error) {
    if !isRootCreationEligible(state, pkt, w.clock.Now()) {
        return SubjectState{}, false, nil
    }

    token := w.idgen.New()
    claim, err := w.state.BeginRootClaim(ctx, subjectKey, token, w.clock.Now().Add(w.rootCreateLease))
    if err != nil {
        return SubjectState{}, false, err
    }

    switch claim.Kind {
    case RootAlreadyExists:
        return claim.State, true, nil
    case RootClaimBusy:
        return SubjectState{}, false, ErrRetryLater
    case RootClaimAcquired:
        // continue
    }

    created, err := w.channel.CreateRootMessage(ctx, pkt)
    if err != nil {
        _ = w.state.ReleaseRootClaim(ctx, subjectKey, token)
        return SubjectState{}, false, err
    }

    committed, err := w.state.CommitRootClaim(ctx, subjectKey, token, created.RootMessageID, created.ChannelID, created.GroupID)
    if err != nil {
        w.metrics.RootCommitAfterCreateFailed(subjectKey)
        return SubjectState{}, false, err
    }

    switch committed.Kind {
    case RootCommitSucceeded:
        return committed.State, true, nil
    case RootAlreadyExists:
        return committed.State, true, nil
    case RootClaimLost:
        return SubjectState{}, false, ErrRetryLater
    }

    return SubjectState{}, false, nil
}
```

- `BeginRootClaim` 과 `CommitRootClaim` 은 ValKey Lua/function 또는 optimistic transaction으로 실행한다.
- `BeginRootClaim` 은 기존 `rootMessageId` 가 있으면 새 claim을 만들지 않고 existing root를 반환한다.
- `root_creating` claim이 만료되지 않았으면 다른 worker는 root를 만들지 않고 retry/backoff 한다.
- `CommitRootClaim` 은 `rootClaimToken` 이 일치하고 기존 `rootMessageId` 가 없을 때만 root pointer를 기록한다.
- `ErrRetryLater` 는 stream ack 없이 retry/backoff 해야 하는 오류로 취급한다.
- lock TTL이 만료되어 두 worker가 같은 subject를 처리해도 `rootMessageId` 를 두 번 기록할 수 없다.
- claim lease는 ChannelTalk root 생성 timeout보다 길게 잡고, 장시간 처리에는 heartbeat로 연장한다.
- root 생성 이후 `CommitRootClaim` 이 실패하면 claim을 즉시 해제하지 않고 retry/backoff 한다.
- claim을 잃은 worker가 뒤늦게 만든 downstream root는 canonical state에 들어가지 않는다. cleanup 대상이 될 수는 있지만 `first root wins forever` invariant는 깨지지 않는다.
- ChannelTalk root 생성 성공 후 `CommitRootClaim` 전 crash의 exactly-once root creation은 Phase 1/2의 non-goal이다.
