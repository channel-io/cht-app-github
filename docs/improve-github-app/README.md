# GitHub App 개선 ValKey Schema 요약

용어 -> Goals / Non-goals / Semantics -> Design Decisions -> ValKey Schema.

## 1. 용어 정의

### 1-1. deliveryID

`deliveryID`: GitHub webhook request의 `X-GitHub-Delivery` header 값.

- transport-level identifier
- packet 저장과 idempotency check 기준값
- header가 없으면 정상 webhook으로 취급하지 않음

Phase 1 dedup key 정의:

```text
deliveryID := req.Header["X-GitHub-Delivery"]
dedupKey   := "dedup:delivery:" + deliveryID
```

### 1-2. subject

`subject`: Phase 1/2에서 하나의 채널톡 루트메시지와 채널톡 스레드 상태를 공유하는 canonical 처리 단위.

이 문서의 `subject/root/anchor` 모델은 issue / PR 계열에만 적용한다.
release event는 현재 구현처럼 standalone group message로만 dispatch하며, Phase 1/2의 shared root state / Anchor recovery 범위에 포함하지 않는다.

canonical subject는 아래 두 종류로 고정.

```text
issue:{org}/{repo}#{number}
pr:{org}/{repo}#{number}
```

- issue event -> `issue:{org}/{repo}#{number}`
- pull_request 계열 event -> `pr:{org}/{repo}#{number}`
- issue_comment on issue -> `issue:{org}/{repo}#{number}`
- issue_comment on pull request -> `pr:{org}/{repo}#{number}`
- status/check_run -> merged된 PR association hit 시 기존 `pr:{org}/{repo}#{number}` subject에만 매핑

worker가 이 값을 계산해 lock/state key 생성.

GitHub의 `issue_comment`는 issue/PR 공용 이벤트다.
`Issue.IsPullRequest()==true` 인 경우 canonical subject는 항상 `pr:{org}/{repo}#{number}` 로 본다.

status/check_run 은 독립 canonical subject를 만들지 않는다.
worker는 sha로 merged된 PR association을 조회하고, 연결된 PR이 있으면 해당 `pr` subject로만 처리한다.
association miss 시 packet은 drop/skip 한다.

### 1-3. Anchor

`Anchor`: GitHub issue / PR에 앱이 남기는 코멘트.

- 채널톡 루트메시지 위치를 가리키는 보조 힌트
- root state source-of-truth는 아니고, Phase 2 lazy rebuild와 retry 기준값으로 사용

## 2. Goals

- delivery dedup을 도입해 같은 webhook delivery의 중복 처리를 막는다.
- issue / PR subject 단위 lock/shared state를 도입해 canonical root를 한 번만 정한다.
- out-of-order event를 pre-root queue와 lazy rebuild로 흡수한다.
- root pointer commit 이후 Anchor 생성/갱신 실패는 state 기반으로 retry/resume 가능하게 만든다.
- `status` / `check_run`은 기존 merged PR thread에만 연결되게 유지한다.

## 3. Non-goals

- end-to-end exactly-once dispatch를 보장하지 않는다.
- downstream thread write 성공 후 ack/state 저장 전 crash로 인한 duplicate thread message는 Phase 1/2 범위에서 제거하지 않는다.
- late `opened`를 위해 root merge나 root replacement를 하지 않는다.
- `status` / `check_run`용 독립 subject나 fallback root를 만들지 않는다.
- release event를 Phase 1/2의 subject/root/anchor state machine에 포함하지 않는다.
- ingress stream을 장기 source-of-truth 저장소로 쓰지 않는다.
- ChannelTalk root 생성 성공 후 subject state commit 전 crash의 exactly-once root creation을 보장하지 않는다.
- orphan ChannelTalk root 자동 삭제/병합은 하지 않고 metric과 수동 cleanup 대상으로 둔다.

## 4. Processing Semantics

- ingress stream consumption과 worker retry 모델은 `at least once`를 전제로 한다.
- `dedup:delivery:{deliveryID}` 는 동일 delivery의 중복 진입을 줄이기 위한 guard이지, 모든 downstream side effect의 exactly-once를 의미하지 않는다.
- ingress는 `dedup check + stream append + dedup record` 를 ValKey Lua/function 한 번으로 원자적으로 수행한다.
- ingress는 atomic enqueue 결과를 받기 전에는 delivery를 성공 처리하지 않는다. atomic step 실패 시 request는 실패로 남기고 GitHub redelivery에 의존한다.
- subject root selection은 strong invariant로 본다. 즉 `rootMessageId` 가 한 번 저장되면 그 root가 canonical root다.
- pre-root queue는 bounded hold 용도다. packet은 짧은 window 동안만 유지하고, timeout 이후에는 `waiting_root_expired` 와 lazy rebuild로 복구를 시도한다.
- downstream dispatch는 best-effort retry 대상이다. worker가 downstream thread write 성공 후 ack/state 저장 전에 중단되면 동일 delivery의 thread message가 중복 관찰될 수 있다.
- 따라서 Phase 1/2의 dedup 범위는 ingress 진입, pre-root append, canonical root selection까지로 한정한다.
- ChannelTalk side effect는 exactly-once 보장 범위 밖이다. 특히 root 생성 성공 후 `commitRootClaim` 전 crash가 나면 user-visible orphan root가 생길 수 있다.
- 이 경우 `subject:{subjectKey}` 에는 root claim CAS로 하나의 canonical root만 기록하고, orphan root는 운영 지표와 수동 cleanup으로 다룬다.

## 5. Design Decisions

### 5-1. ValKey lock vs Kafka partition-by-subject

Kafka에서 subject key로 partition을 나누면 같은 subject의 worker 경쟁을 구조적으로 줄일 수 있다.
다만 Phase 1/2에서는 Kafka partition을 correctness boundary로 두지 않고 ValKey 기반 ingress/state를 먼저 둔다.

이유:

- 현재 webhook ingress는 HTTP request를 즉시 ack해야 하므로 delivery dedup, raw packet enqueue, retry/backlog 상태가 먼저 필요하다.
- root pointer, pending Anchor, pre-root buffer, lazy rebuild 상태는 Kafka를 쓰더라도 별도 shared state가 필요하다.
- Kafka 전환 시점과 GitHub App 정합성 개선 시점을 강하게 묶지 않는다. Kafka 도입 전에도 중복 delivery와 root lookup miss를 줄일 수 있어야 한다.
- partition-by-subject는 좋은 worker scheduling 전략이지만, `first root wins forever` 같은 root 확정 불변식은 여전히 state machine에서 보장해야 한다.

따라서 Phase 1/2의 선택은 "ValKey로 영구 불변식을 보장하고, lock은 동일 subject의 동시 실행을 줄이는 fast path로 사용"이다.
Kafka가 event backbone으로 안정화되면 `PacketQueue` backend는 Kafka partition-by-subject로 바꿀 수 있지만, `subject:{subjectKey}` CAS 규약은 유지한다.

### 5-2. Lock TTL and root claim

`lock:subject:{subjectKey}` TTL은 worker mutual exclusion을 돕는 lease일 뿐, root 확정의 단독 보장 장치가 아니다.
worker가 downstream write 중 멈추거나 network timeout이 길어져 lock TTL이 만료되면 다른 worker가 같은 subject를 처리할 수 있다.

`first root wins forever` 는 `subject:{subjectKey}` 에 대한 원자적 root claim/commit으로 보장한다.

```text
beginRootClaim(subjectKey, claimToken, claimExpiresAt):
  state = HGETALL subject:{subjectKey}

  if state.rootMessageId exists:
    return existing_root(state)

  if state.rootClaimToken exists and state.rootClaimExpiresAt > now:
    return claim_busy(state.rootClaimToken)

  HSET subject:{subjectKey}
    rootClaimToken claimToken
    rootClaimExpiresAt claimExpiresAt
    rootState root_creating
    updatedAt now
  EXPIRE subject:{subjectKey} 90d
  return claim_acquired(claimToken)

commitRootClaim(subjectKey, claimToken, rootMessageId, channelId, groupId):
  state = HGETALL subject:{subjectKey}

  if state.rootMessageId exists:
    return existing_root(state)

  if state.rootClaimToken != claimToken:
    return claim_lost(state)

  HSET subject:{subjectKey}
    rootMessageId rootMessageId
    channelId channelId
    groupId groupId
    rootState pending_anchor
    updatedAt now
  HDEL subject:{subjectKey} rootClaimToken rootClaimExpiresAt
  EXPIRE subject:{subjectKey} 90d
  return committed(rootMessageId)
```

위 두 함수는 ValKey Lua/function 또는 optimistic transaction으로 구현한다.
worker는 root 생성 전 `beginRootClaim` 을 먼저 통과해야 하고, ChannelTalk root 생성 후 `commitRootClaim` 이 성공한 root만 canonical root로 사용한다.
lock TTL이 만료되어 다른 worker가 들어오더라도 이미 `rootMessageId` 가 있으면 그 값을 따른다.
아직 root 생성 중이면 claim expiry 전까지 새 root를 만들지 않는다.

claim lease는 ChannelTalk root 생성 timeout보다 길게 잡고 처리 중 heartbeat로 연장한다.
worker가 claim을 잃은 뒤 뒤늦게 downstream root를 만들었더라도 `commitRootClaim` 에 실패하므로 그 root는 canonical state에 들어가지 않는다.
이 경우 운영상 orphan root가 남을 수 있지만, subject의 canonical root가 바뀌거나 두 root가 동시에 state에 기록되지는 않는다.
root 생성 이후 `commitRootClaim` 이 실패하면 worker는 claim을 즉시 해제하지 않고 retry/backoff 한다.
그래도 claim lease가 만료되면 새 worker가 다시 root를 만들 수 있으며, 이 위험은 Phase 1/2에서 accepted trade-off로 둔다.

운영 지표:

- `root_claim_expired_total`
- `root_commit_after_create_failed_total`
- `root_orphan_suspected_total`

### 5-3. Subject state update contract

`subject:{subjectKey}` 에 대한 일반 update는 blind overwrite를 금지한다.
`lock:subject:{subjectKey}` 는 lease이므로, lock을 잡았던 worker도 stale state를 저장할 수 있다.

모든 state update는 아래 둘 중 하나로만 수행한다.

1. ValKey Lua/function으로 조건과 patch를 한 번에 적용한다.
2. optimistic transaction으로 `version` 을 확인한 뒤 patch를 적용한다.

```text
patchSubjectState(subjectKey, expectedVersion, patch):
  state = HGETALL subject:{subjectKey}

  if state.version != expectedVersion:
    return version_conflict

  if patch clears rootMessageId/channelId/groupId:
    return invalid_patch

  if state.rootMessageId exists and patch.rootState is waiting_root:
    return invalid_patch

  HSET subject:{subjectKey} patch
  HINCRBY subject:{subjectKey} version 1
  HSET subject:{subjectKey} updatedAt now
  EXPIRE subject:{subjectKey} 90d
  return patched
```

Monotonic fields:

- `rootMessageId`, `channelId`, `groupId` 는 한 번 set 되면 삭제/교체하지 않는다.
- `rootClaimToken`, `rootClaimExpiresAt` 은 claim owner, expired claim takeover, 또는 successful commit만 변경한다.
- `anchorCommentId` 는 verified Anchor 또는 pending Anchor retry 성공으로만 set 한다.
- `rootState=ready/pending_anchor` 인 state를 stale `waiting_root` 로 되돌리지 않는다.

version conflict는 정상적인 동시성 신호다.
worker는 최신 state를 reload하고 다시 판단하거나, retryable packet이면 stream ack 없이 retry/backoff 한다.

### 5-4. Stream retry and reclaim

`stream:github-events` 는 ValKey Streams consumer group으로 소비한다.

```text
consumer group: github-event-workers
consumer name : {workerId}
read          : XREADGROUP GROUP github-event-workers {workerId}
ack           : XACK stream:github-events github-event-workers {streamEntryID}
reclaim       : XAUTOCLAIM/XCLAIM idle > 60s
max attempts  : 10
dead stream   : stream:github-events:dead
```

Worker ack 규칙:

- 처리 성공, duplicate, unsupported event, final abandon 만 `XACK` 한다.
- retryable error, busy root claim, retryable existing-root lookup miss는 ack하지 않는다.
- attempt count가 10회를 넘으면 original packet과 last error를 dead stream에 기록하고 original entry를 `XACK` 한다.

Stream trim 규칙:

- stream은 최근 backlog 보관용이며 기본 retention은 1h다.
- pending entry는 trim 대상에서 제외한다.
- trim cutoff는 `XPENDING` 의 가장 작은 pending ID보다 오래된 acked entry에만 적용한다.
- dead stream은 별도 retention을 두고 운영 확인 후 삭제한다.

운영 지표:

- `github_event_stream_pending`
- `github_event_stream_reclaimed_total`
- `github_event_stream_dead_total`
- `github_event_stream_attempts_total`

### 5-5. Verified Anchor

Phase 2 lazy rebuild는 ChannelTalk URL 문자열만으로 root를 복구하지 않는다.
앱이 작성한 GitHub comment 안의 hidden marker와 URL을 함께 검증한다.

Anchor comment format:

```text
<!-- cht-app-github:anchor v1 subject=pr:org/repo#123 rootMessageId={rootMessageId} channelId={channelId} groupId={groupId} -->
https://desk.channel.io/#/channels/{channelId}/team_chats/groups/{groupId}/{rootMessageId}
```

Verified Anchor 조건:

- comment author가 현재 GitHub App bot이다.
- marker version이 지원 범위 안이다.
- marker subject가 worker가 계산한 `subjectKey` 와 정확히 일치한다.
- URL에서 파싱한 `channelId/groupId/rootMessageId` 가 marker 값과 일치한다.
- state에 이미 `rootMessageId` 가 있으면 Anchor 값으로 state를 교체하지 않는다.

Pagination / conflict 규칙:

- GitHub comments는 모든 page를 조회한다.
- valid marker가 정확히 하나면 lazy rebuild에 사용한다.
- 서로 다른 `rootMessageId` 의 valid marker가 여러 개면 자동 rebuild하지 않고 `anchor_conflict_total` 을 기록한다.
- marker 없는 기존 URL-only comment는 Phase 2 rebuild source로 쓰지 않는다.

### 5-6. Complexity and rollout gate

이 설계는 처음부터 모든 복잡도를 켜는 계획이 아니다.
운영 복잡도는 아래 순서로 점진 도입한다.

1. 단일 worker + delivery dedup + root claim CAS를 먼저 적용한다.
2. queue lag 또는 subject contention이 관찰될 때만 multi-worker와 `lock:subject:{subjectKey}` 를 켠다.
3. root lookup miss, pending Anchor, pre-root abandon이 실제 지표로 확인될 때 Phase 2 lazy rebuild/retry를 켠다.
4. Kafka partition-by-subject가 플랫폼 기본 경로가 되면 `PacketQueue` backend를 교체하고 ValKey stream 의존도를 낮춘다.

복잡도 확대 기준:

- duplicate delivery 또는 redelivery로 같은 webhook이 여러 번 처리된다.
- 같은 issue/PR subject에서 root lookup miss 또는 duplicate root가 관찰된다.
- webhook ack 지연을 줄이기 위해 ingress와 worker를 분리해야 한다.
- 단일 worker의 queue lag가 운영 SLO를 넘는다.

이 기준을 만족하지 않으면 단일 worker + dedup + root claim CAS까지만 적용한다.
ValKey stream, subject lock, state machine은 같은 인터페이스 뒤에 두어 단계적으로 켜고 끌 수 있게 둔다.

## 6. ValKey Schema

Schema는 phase별 책임과 필요한 키만 남김.

### 6-1. Phase 1 목표와 필수 키

목표: delivery dedup, subject lock/shared state, root claim CAS, pre-root queue 도입.

```text
# delivery dedup
dedup:delivery:{deliveryID}        -> "{streamEntryID}"                TTL 7d

# ingress queue
stream:github-events               -> WebhookPacket                    retain acked entries 1h, pending-safe trim

# dead queue
stream:github-events:dead          -> WebhookPacket + lastError         retain 7d, trim by age/size

# subject lock
lock:subject:{subjectKey}          -> "{workerId}"                     TTL 30s

# subject state
subject:{subjectKey}               -> Hash / JSON                      TTL 90d, refresh on write

# pre-root queue
pre-root:subject:{subjectKey}      -> List(WebhookPacket)              TTL 5m

# pre-root idempotency guard
pre-root:buffered-delivery:{subjectKey} -> Set(deliveryID)             TTL 5m
```

키 역할:

- `dedup:delivery:{deliveryID}`
  - 같은 delivery를 두 번 처리하지 않기 위한 idempotency key
  - value는 `"1"` 고정값이 아니라 ingress가 enqueue에 성공했을 때 받은 `streamEntryID`
  - duplicate redelivery는 key 존재만으로 판별하고, 저장된 `streamEntryID` 는 운영 디버깅/추적에 사용
- `stream:github-events`
  - ingress가 raw webhook packet을 적재하는 단일 stream
  - `PacketQueue`의 ValKey Streams backend
  - source-of-truth 저장소가 아니라 짧은 transport backlog이므로 최근 1시간만 보관
  - consumer group / pending reclaim / dead stream 규약은 Design Decisions의 stream retry contract를 따른다
- `stream:github-events:dead`
  - max attempts를 초과한 packet과 last error를 남기는 dead stream
  - 운영자가 원인을 확인하고 수동 replay 또는 abandon 여부를 결정한다

Ingress write contract:

```text
atomicEnqueue(packet):
  if EXISTS dedup:delivery:{deliveryID}:
    return duplicate

  streamEntryID = XADD stream:github-events * packet
  SET dedup:delivery:{deliveryID} streamEntryID EX 7d
  return enqueued(streamEntryID)
```

- 위 세 단계는 ingress에서 분리 수행하지 않고 ValKey Lua/function 한 번으로 실행한다.
- `SET dedup` 이 먼저 성공하고 `XADD` 전에 중단되는 경우, 혹은 `XADD` 후 `SET dedup` 전에 중단되는 경우를 허용하지 않는다.
- 즉 ingress crash/retry에 대해 `deliveryID` 당 `stream:github-events` enqueue는 0회 또는 1회만 관찰되게 만든다.
- `lock:subject:{subjectKey}`
  - 같은 subject를 동시에 두 worker가 처리하지 못하게 하는 distributed lock
  - correctness boundary가 아니라 contention을 줄이는 lease
  - TTL 만료 후 동시 worker가 생겨도 root 확정은 `subject:{subjectKey}` root claim CAS가 보장
- `subject:{subjectKey}`
  - root를 다시 찾기 위한 minimal root pointer state를 저장하는 shared state
  - `rootMessageId` 는 ValKey Lua/function 또는 optimistic transaction으로만 최초 기록
  - `rootClaimToken/rootClaimExpiresAt` 으로 root 생성 중 상태를 표현하고 stale claim만 재획득 허용
  - 일반 update는 `version` 기반 patch 또는 field-level monotonic merge로만 수행
  - write/update 때 TTL을 90일로 갱신
  - 장기간 닫힌 subject는 자연 만료되고, state miss는 Phase 2 lazy rebuild로 복구
- `pre-root:subject:{subjectKey}`
  - root가 아직 없는 동안 subject의 모든 packet을 arrival order대로 잠깐 보관
  - wait window 30s보다 길게 5m 유지해 timeout sweep가 늦어도 queue가 먼저 사라지지 않게 함
  - subject당 최대 100개 packet만 보관하고 초과분은 `pre_root_overflow_total` 기록 후 final abandon
  - `opened` 도착 시 채널톡 루트메시지 생성 후 drain
  - `opened`가 wait window 안에 오지 않아도 즉시 synthetic fallback root를 만들지 않음
  - timeout 시 subject를 `waiting_root_expired` 로만 표시하고, 이후 같은 subject의 새 packet이 올 때 지연 복구를 시도
  - TTL 만료 전 복구되지 않은 buffered packet은 abandoned로 보고 `pre_root_abandoned_total` 을 기록
- `pre-root:buffered-delivery:{subjectKey}`
  - pre-root queue에 이미 적재한 `deliveryID` 를 기록하는 idempotency guard
  - worker retry로 같은 packet이 재처리돼도 동일 `deliveryID` 는 한 번만 append 한다
  - `pre-root:subject:{subjectKey}` 와 같은 TTL로 관리한다

`subject:{subjectKey}` 최소 필드는 아래와 같이 고정.

```text
subjectKey
rootMessageId
channelId
groupId
rootState               # waiting_root | waiting_root_expired | root_creating | pending_anchor | ready
rootClaimToken          # optional, root_creating lease owner
rootClaimExpiresAt      # optional, root_creating lease expiry
version
updatedAt
```

`groupId`는 해당 채널톡 스레드 생성 시점의 groupId를 본다.
config가 바뀌어도 기존 subject는 기존 `channelId/groupId/rootMessageId`를 계속 사용하고, 새 subject만 새 config 적용.

`stream:github-events`에 적재하는 packet는 ingress에서 final subject key를 확정하지 않음.
worker가 final subject key를 계산할 수 있도록 routing hint와 raw payload 중심으로 저장.

```text
deliveryID
eventName
action
org
repo
number
sha
receivedAt
headers
payload
```

`status` / `check_run`은 ingress 시점에 아직 merged PR association이 없을 수 있다.
이 경우 별도 fallback subject를 만들지 않고 packet을 종료한다.

다만 merged PR association이 확인된 뒤에는 current behavior와 동일하게 기존 root comment(Anchor) 기반 existing-root lookup을 먼저 시도한다.
이 경로의 root lookup miss는 immediate drop 대상이 아니고, 짧은 bounded retry/backoff 이후에도 root를 찾지 못한 경우에만 종료한다.

`opened` 누락 시 root 생성 정책은 아래와 같이 고정한다.

1. wait window 안에 `opened`가 오면 그 packet으로 normal root를 만든다.
2. wait window가 지나도 `opened`가 없으면 즉시 synthetic fallback root를 만들지 않는다.
3. 대신 `subject:{subjectKey}` 를 `waiting_root_expired` 상태로 남기고 pre-root packet은 TTL 동안 유지한다.
4. 이후 같은 subject의 새 packet이 오면 worker는 먼저 existing root/state를 확인하고, 없으면 Anchor 기반 lazy rebuild를 시도한다.
5. rebuild도 실패한 경우에만 그 시점의 packet을 기준으로 fallback root를 한 번 생성한다.
6. `rootMessageId` 가 한 번 저장되면 그 root가 canonical root다. late `opened` 는 새 root를 만들지 않고 기존 root에 대한 일반 packet처럼 처리하거나 noop 한다.

즉 정책은 `first root wins forever` 이다.
late `opened` reconciliation을 위해 root merge나 root 교체는 하지 않는다.

Pre-root constants:

```text
waitWindow          = 30s
preRootTTL          = 5m
maxBufferedPackets  = 100
```

### 6-2. Phase 2 목표와 확장 키

목표: pending Anchor retry, verified Anchor lazy rebuild, bounded abandon.

`status` / `check_run`은 root 생성 트리거가 아니다.
기존 merged PR root와 연결된 경우에만 thread message dispatch 대상으로 본다.
merged PR association이 없으면 buffer/rebuild/fallback 없이 종료한다.
다만 merged PR association hit 이후에는 current behavior와 동일하게 기존 root comment(Anchor) 기반 existing-root lookup을 먼저 시도한다.
이 경로의 root lookup miss는 immediate drop 대상이 아니고, bounded retry/backoff 이후에도 root를 찾지 못한 경우에만 종료한다.

```text
# pending anchor retry index
anchor:pending-retry               -> ZSET(subjectKey, retryAt)        no TTL, remove on success + stale cleanup
```

키 역할:

- `anchor:pending-retry`
  - GitHub에 달리는 코멘트(Anchor) 재시도가 필요한 subject 목록
- `subject:{subjectKey}`
  - Anchor retry metadata와 마지막 실패 원인 저장
  - retry 성공 시 관련 필드 정리

권장 필드:

```text
anchorRetryCount
anchorLastError
anchorLastTriedAt
anchorNextRetryAt
anchorCommentId
```

이 키들은 Anchor retry worker가 소비한다.
