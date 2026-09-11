---
name: go-backend-standards
description: Use when writing, refactoring, or reviewing Go backend code, or when designing a Go service or its API, gRPC contracts, database schema, migrations, messaging (Kafka, NATS, ZMQ), cache, money handling, or tests. Also when choosing infrastructure for a Go service. Go backend only.
---

# Go Backend Standards

The owner's defaults for Go services. Use them for new work; preserve explicit user choices
and established repository conventions. Explain material deviations briefly. A tool preference
is not evidence of a correctness defect. Apply only the branches relevant to the task; a small
fix does not require redesigning the service.

## Correctness requirements

Preserve exact monetary values and explicit currency/rounding semantics; make retries safe;
state transaction, delivery, ordering, and consistency boundaries. Findings need a concrete
failure mode and evidence. Rate their impact using [go-review.md](go-review.md).

## Owner defaults and exceptions

The table guides choices where the project has no established answer. References describe
conditions and exceptions; do not manufacture infrastructure or review blockers to match it.

| Concern | Default | Why |
|---|---|---|
| Layout | `cmd/<svc>`, `internal/domain`, `internal/usecase`, `internal/port`, `internal/adapter/{grpc,http,postgres,kafka,cache}` | domain imports nothing from adapters; adapters are swappable |
| Money | `shopspring/decimal` in Go, `NUMERIC(19,4)` in Postgres, `Decimal128` in Mongo, ISO-4217 `currency` beside it | choose precision and rounding for the domain; checked minor units are a valid exception |
| Contracts | spec-first: proto via `buf`, OpenAPI 3 via `oapi-codegen`; contracts committed and reviewed before code | tests and clients derive from the contract |
| Services | one service per bounded domain; gRPC between services; server streaming for large or long-lived responses | ownership follows data |
| HTTP lists | batch endpoints where a client would loop; cursor pagination for user-facing lists; keyset for large scans; offset for small or frozen datasets | exact exports require a snapshot or materialized dataset |
| Events | outbox table written in the same transaction; in-service relay; transactional inbox for DB effects; at-least-once delivery | duplicates and external effects need explicit handling |
| Idempotency | keys for retriable non-idempotent mutations and external side effects | verify provider support, scope, concurrency, and retention |
| Kafka | `twmb/franz-go`; partition key = entity id (user, order); partition count derived from measured throughput plus headroom; transactions only for consume-transform-produce | ordering also depends on publication and consumer processing |
| Native stack | NATS JetStream when self-hosted without Kafka; ZeroMQ for brokerless point-to-point | fits the infra that exists |
| Database | Postgres by default; MongoDB for justified document access patterns; normalization and measured query plans | existing infrastructure and workloads decide |
| Outbox table | quarterly partitions when volume/retention justify them | retain pending events; an indexed unpartitioned table is a valid start |
| Migrations | `goose`, SQL files, deploy step, expand/contract | document irreversible changes and recovery |
| Cache | distributed `memcached`, cache-aside, invalidate after commit; `hazelcast` when memcached lacks the needed structures | local immutable/versioned caches are valid when consistency allows |
| Media | `ffmpeg`/`ffprobe` in a worker, frames and transcodes to object storage | CPU-bound work off the API path |
| Tests | TDD; integration tests via `testcontainers-go`; black-box autotests in a sibling `<svc>-autotests` repo for externally consumed services | infra bugs surface before deploy |

## Branches

Read the reference for the branch you are in; each ends with a done-when check.

- Writing or reviewing Go code: [go-review.md](go-review.md). Linters from [golangci.yml](golangci.yml) cover the mechanical mistakes; the reference covers the judgment calls.
- Designing a service, API, or messaging flow: [service-design.md](service-design.md). Start with known infrastructure and state missing assumptions.
- Designing a schema, query, partition, migration, or cache: [data-design.md](data-design.md).
- Writing tests or setting up a repo's test layout: [testing.md](testing.md).

## Design output

For a full service design, cover infra assumptions and open questions, contracts, schema, event
flow and guarantees, idempotency points, and tests. For focused tasks include only relevant parts.
