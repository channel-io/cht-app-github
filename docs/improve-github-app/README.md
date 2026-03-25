# GitHub App 개선 ValKey Schema 요약

용어 -> Goals / Non-goals / Semantics -> ValKey Schema.

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
- partial failure after root write 상황에서도 state/Anchor 기반으로 resume 가능하게 만든다.
- `status` / `check_run`은 기존 merged PR thread에만 연결되게 유지한다.

## 3. Non-goals

- end-to-end exactly-once dispatch를 보장하지 않는다.
- downstream thread write 성공 후 ack/state 저장 전 crash로 인한 duplicate thread message는 Phase 1/2 범위에서 제거하지 않는다.
- late `opened`를 위해 root merge나 root replacement를 하지 않는다.
- `status` / `check_run`용 독립 subject나 fallback root를 만들지 않는다.
- release event를 Phase 1/2의 subject/root/anchor state machine에 포함하지 않는다.
- ingress stream을 장기 source-of-truth 저장소로 쓰지 않는다.

## 4. Processing Semantics

- ingress stream consumption과 worker retry 모델은 `at least once`를 전제로 한다.
- `dedup:delivery:{deliveryID}` 는 동일 delivery의 중복 진입을 줄이기 위한 guard이지, 모든 downstream side effect의 exactly-once를 의미하지 않는다.
- ingress는 `dedup check + stream append + dedup record` 를 ValKey Lua/function 한 번으로 원자적으로 수행한다.
- ingress는 atomic enqueue 결과를 받기 전에는 delivery를 성공 처리하지 않는다. atomic step 실패 시 request는 실패로 남기고 GitHub redelivery에 의존한다.
- subject root selection은 strong invariant로 본다. 즉 `rootMessageId` 가 한 번 저장되면 그 root가 canonical root다.
- pre-root queue는 bounded hold 용도다. packet은 짧은 window 동안만 유지하고, timeout 이후에는 `waiting_root_expired` 와 lazy rebuild로 복구를 시도한다.
- downstream dispatch는 best-effort retry 대상이다. worker가 downstream thread write 성공 후 ack/state 저장 전에 중단되면 동일 delivery의 thread message가 중복 관찰될 수 있다.
- 따라서 Phase 1/2의 dedup 범위는 ingress 진입, pre-root append, canonical root selection까지로 한정한다.

## 5. ValKey Schema

Schema는 phase별 책임과 필요한 키만 남김.

### 5-1. Phase 1 목표와 필수 키

목표: delivery dedup, subject lock/shared state, pre-root queue 도입.

```text
# delivery dedup
dedup:delivery:{deliveryID}        -> "{streamEntryID}"                TTL 7d

# ingress queue
stream:github-events               -> WebhookPacket                    retain 1h, trim by age/size

# subject lock
lock:subject:{subjectKey}          -> "{workerId}"                     TTL 30s

# subject state
subject:{subjectKey}               -> Hash / JSON                      TTL 90d, refresh on write

# pre-root queue
pre-root:subject:{subjectKey}      -> List(WebhookPacket)              TTL 30s

# pre-root idempotency guard
pre-root:buffered-delivery:{subjectKey} -> Set(deliveryID)             TTL 30s
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
- `subject:{subjectKey}`
  - root를 다시 찾기 위한 minimal root pointer state를 저장하는 shared state
  - write/update 때 TTL을 90일로 갱신
  - 장기간 닫힌 subject는 자연 만료되고, state miss는 Phase 2 lazy rebuild로 복구
- `pre-root:subject:{subjectKey}`
  - root가 아직 없는 동안 subject의 모든 packet을 arrival order대로 잠깐 보관
  - wait window보다 길게 유지해 timeout sweep가 늦어도 queue가 먼저 사라지지 않게 함
  - `opened` 도착 시 채널톡 루트메시지 생성 후 drain
  - `opened`가 wait window 안에 오지 않아도 즉시 synthetic fallback root를 만들지 않음
  - timeout 시 subject를 `waiting_root_expired` 로만 표시하고, 이후 같은 subject의 새 packet이 올 때 지연 복구를 시도
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
rootState               # waiting_root | waiting_root_expired | pending_anchor | ready
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

### 5-2. Phase 2 목표와 확장 키

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
