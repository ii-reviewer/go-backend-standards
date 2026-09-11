# Service design

Use the sections relevant to the requested design. Done-when checks apply to implementation
when implementation is requested; a design-only answer describes the artifacts without claiming
to have created or compiled them.

## 1. Infrastructure discovery

Start with infrastructure facts already in the request or repository. Ask only for missing
facts that materially change the design; otherwise state assumptions:

- Cloud or self-hosted: AWS, GCP, on-prem?
- Broker: managed Kafka (MSK, Confluent), an own Kafka cluster (version, who owns topics and
  ACLs), NATS, or nothing yet?
- Stores available: Postgres version, MongoDB, memcached cluster, object storage?
- CDC (Debezium) or a shared schema registry already in use?
- Which services already consume events, and with what envelope?

Transport per interaction:

| Interaction | Transport |
|---|---|
| durable events, replay, many consumers, per-key ordering | Kafka |
| durable events on a self-hosted stack without Kafka | NATS JetStream |
| brokerless point-to-point or pipeline between processes, app handles loss | ZeroMQ |
| request/response between services | gRPC unary |
| large or long-lived response: export, feed, progress | gRPC server streaming |
| bidirectional realtime | gRPC bidi streaming |
| edge traffic from browsers and gateways | HTTP via OpenAPI |

Done when: every interaction in the design names its transport and the infra fact behind it.

## 2. Contracts, spec first

- One service owns one bounded domain. Split when data ownership splits, never earlier.
- gRPC: `api/proto/<svc>/v1/*.proto` in a `buf` module; `buf lint` and `buf breaking` in CI.
  Generated code is committed or generated in CI, never edited by hand.
- HTTP for the edge: `api/openapi.yaml`, `oapi-codegen` strict server, handlers in
  `internal/adapter/http`.
- Batch endpoints wherever a client would otherwise loop: `POST /invoices:batchGet` with a
  bounded id list.
- Pagination:
  - user-facing lists: cursor on `(created_at, id)`, `limit` capped, `next_cursor` in the
    response; the cursor binds filters, tenant scope, and sort direction.
  - statistics, exports, admin scans: `limit`/`offset` with a fixed unique `ORDER BY` over a
    closed past range; the job resumes from its last offset. Late writes into a closed range
    shift offsets, so the range starts after the write grace period and a backfill reruns it.
  - exact snapshot required: run the scan in one `REPEATABLE READ` transaction, or materialize
    the rows into an export table first; a snapshot cannot be resumed after its transaction ends.
- Mutations that can repeat a non-idempotent effect need an idempotency key, scoped per caller
  and operation (`Idempotency-Key` header, `idempotency_key` field in proto), unless natural
  idempotency gives the same guarantee. Read-only calls do not need such a key.

Done when: applicable HTTP/proto contracts specify pagination consistency, error and retry
semantics. For implementation tasks, generated stubs compile; one transport does not require
introducing the other.

## 3. Reliability: outbox, inbox, idempotency

### Database and delivery boundaries

- State change and its outbox event commit in one transaction. Assign a stable event ID once;
  retain it across every publish retry. Include aggregate ID, event type/schema version, and an
  aggregate sequence if ordered application is required.
- A polling relay may claim pending rows with `FOR UPDATE SKIP LOCKED`. Keep row locks until
  publish acknowledgement and the `published_at` update commit, using bounded network calls
  and batches. A lease-based implementation instead needs expiry and stale-owner handling.
  Neither a lock nor a lease is a distributed transaction with the broker.
- Mark published only after a successful broker acknowledgement. A timeout is ambiguous;
  retry with the same event ID. A crash after publish but before DB commit produces a duplicate.
  This is at-least-once delivery, assuming retained pending rows and continued retries/recovery.
- Consumer inbox: `PRIMARY KEY (consumer_name, event_id)` when multiple logical handlers share
  a table, or `event_id PRIMARY KEY` in a handler-specific table. Insert with
  `ON CONFLICT DO NOTHING` and apply the business change in the SAME DB transaction. Check rows
  affected; a conflict means already committed, while failure rolls back both. Acknowledge only
  after commit. Keep inbox entries at least as long as the allowed replay/redelivery horizon.
- Guarantee: duplicate deliveries have a single committed DB effect within that transaction
  and deduplication horizon. This does not make HTTP calls, email, or payments exactly-once.
  Enqueue those intents in a downstream outbox in the consumer transaction.

### Ordering

- `ORDER BY id ... SKIP LOCKED` with two workers can publish event 2 while event 1 is locked.
  Kafka's aggregate key preserves partition append order, not the original DB business order;
  identity IDs also do not prove transaction commit order.
- Decide whether the domain needs ordering. For ordered state transitions, allocate a monotonic
  per-aggregate sequence while holding the aggregate lock in the business transaction. Serialize
  dispatch per aggregate (or stable shard) and address stale publishers during ownership changes;
  a bare expiring lease does not fence broker writes.
- Enforce sequence at the consumer in the same transaction as the inbox/business change:
  defer or durably buffer gaps; reject already-applied versions as appropriate to the domain.
  Do not mark a gap event processed unless a durable buffer owns its later application. If events
  are deltas, do not skip gaps; versioned full snapshots may allow replacement if specified.
  This gives ordered application even when arrival order or stale retries cannot be controlled.
  Define sequence numbers for the subscribed stream; filtered event types must not leave gaps
  that a consumer can never receive.
- Retry/DLQ handling must preserve that contract: quarantine and alert on poison events; do not
  silently advance past a required transition. Specify how an operator repairs and replays it.

### Commands and external effects

- Command store: unique `(scope, operation, key)`, canonical request hash, state (`pending`,
  `completed`, or a domain-specific terminal result), stored response/status, timestamps, expiry.
  Atomically reserve the key with the business change or durable work intent. Concurrent requests
  must not both execute; an in-flight duplicate waits within a deadline or returns a documented
  retryable status. Same key/hash replays a completed result; different hash returns the contract's
  conflict response (owner HTTP default: 422). Do not store only a response reference without
  defining how it stays available and unchanged for retries.
- External effects use a stable key per logical operation, not just entity ID when an entity
  can have several charges/refunds. Verify the provider's key scope, retention window and replay
  behavior. A header the provider ignores provides no safety.
- On ambiguous PSP timeout, retain the pending operation and retry the same supported key or
  reconcile provider status/webhooks. Do not start a new charge with a new key. If the provider
  offers no dedupe/status mechanism, document the duplicate/loss tradeoff and require an explicit
  recovery decision before an unsafe retry; generic exactly-once claims are invalid.
- Retention/operations: bound retries with backoff and jitter, alert on oldest pending event,
  relay failures and sequence gaps, and provide replay/reconciliation procedures. Never drop a
  partition containing pending or unresolved events just because its quarter ended.

Done when: the design names transaction boundaries, stable IDs, concurrency handling, ordering
requirements, external-effect limits, retention and crash recovery. Implementation includes the
applicable migrations and tests for duplicate delivery and ambiguous outcomes.

## 4. Kafka

- Client: `twmb/franz-go`.
- Partition key is the entity whose events must stay ordered: `user_id`, `order_id`,
  `account_id`. A key is never empty and never random.
- Partition count: measure per-partition throughput on the actual cluster for both the producer
  and the slowest consumer, take `ceil(target_throughput / min(producer_pp, consumer_pp))`, add
  headroom for growth (2x is a sane start). Counts only grow, and growing rehashes keys, so size
  once with the peak in mind.
- Topic name `<domain>.<entity>.v<N>` for events that must be ordered together; event type is
  in the envelope. Separate event-type topics only when cross-type ordering is unnecessary or
  application sequencing handles it; Kafka has no ordering guarantee across topics. Payload is
  proto from the same buf module.
- One consumer group per independent logical subscription; commit offsets after durable handling.
  With parallel handlers, advance only the contiguous completed offset per partition, never past
  unfinished work. Preserve per-aggregate application ordering if required.
- Transactions (`transactional.id`, `read_committed` consumers) only for consume-transform-produce
  chains that the outbox cannot express; the design says why.

Done when: key, partition count with its arithmetic, topic names, and consumer groups are listed.

## 5. Synchronous calls

- gRPC with a deadline per call, retries only on idempotent RPCs, a circuit breaker, otel
  interceptors.
- Prefer carrying the needed snapshot in the event over calling back at runtime; the sync call is
  the fallback, not the design.

## 6. Media

- Frame extraction, thumbnails, transcodes: `ffmpeg` and `ffprobe` through `os/exec` with a
  context timeout, in a worker process separate from the API, results to object storage, job
  state in the database.

## 7. Design document

For a full service design: infra assumptions and open questions; bounded context and service
list; contracts; schema; event flow with guarantee; idempotency points; cache plan; test plan;
rollout and migration steps. Focused designs include only relevant sections.

Sources: [Postgres locking](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE),
[transaction snapshots](https://www.postgresql.org/docs/current/transaction-iso.html),
[LIMIT/OFFSET](https://www.postgresql.org/docs/current/queries-limit.html),
[Kafka delivery semantics](https://kafka.apache.org/41/design/design/).
