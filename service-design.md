# Service design

Work the sections in order. Each ends with a done-when check; the design answer is not complete
until every check holds.

## 1. Infrastructure discovery

The first question in any design, before a diagram exists. Ask, and record the answers as
assumptions when nobody can answer yet:

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
  - user-facing lists: cursor on `(created_at, id)`, `limit` capped, `next_cursor` in the response.
  - statistics, exports, admin scans: `limit`/`offset` with a fixed `ORDER BY` over a bounded
    time range; the job resumes from its last offset and the range does not move.
- Every mutating request carries an idempotency key, scoped per caller
  (`Idempotency-Key` header, `idempotency_key` field in proto).

Done when: proto and OpenAPI files exist, generated stubs compile, review comments are on the
contract rather than on code.

## 3. Reliability: outbox, inbox, idempotency

- State change and the event it produces commit in one transaction: the outbox row is inserted
  next to the update.
- Relay: an in-service worker, one or two replicas, polling
  `SELECT ... FROM outbox WHERE published_at IS NULL ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`,
  publishing, then marking `published_at`. Table shape in `data-design.md`.
- Inbox on every consumer: `inbox(event_id PRIMARY KEY, received_at, processed_at)`. The handler
  runs `INSERT ... ON CONFLICT DO NOTHING` and the business change in one transaction; zero rows
  inserted means duplicate, acknowledge and move on.
- Guarantee to state in the design: at-least-once delivery, exactly-once effect through the inbox.
- Command idempotency store: `idempotency_keys(scope, key) PRIMARY KEY, request_hash, response_ref`.
  Same key and same hash replays the stored response; same key and a different hash returns 422.
- External calls (PSP, partner APIs) carry the entity id or the stored key as their idempotency
  key, so a retry after a crash charges once.

Done when: outbox, inbox, and idempotency tables are in the migration set, the relay is a named
component, and the guarantee sentence appears in the design.

## 4. Kafka

- Client: `twmb/franz-go`.
- Partition key is the entity whose events must stay ordered: `user_id`, `order_id`,
  `account_id`. A key is never empty and never random.
- Partition count: measure per-partition throughput on the actual cluster for both the producer
  and the slowest consumer, take `ceil(target_throughput / min(producer_pp, consumer_pp))`, add
  headroom for growth (2x is a sane start). Counts only grow, and growing rehashes keys, so size
  once with the peak in mind.
- Topic name `<domain>.<entity>.<event>.v<N>`; payload is proto from the same buf module.
- One consumer group per consuming service; commit offsets after the inbox transaction commits.
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

Contains, in order: infra assumptions and open questions; bounded context and service list;
contracts; schema; event flow with guarantee; idempotency points; cache plan; test plan;
rollout and migration steps.
