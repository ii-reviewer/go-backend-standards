---
name: go-backend-standards
description: Use when writing, refactoring, or reviewing Go backend code, or when designing a Go service or its API, gRPC contracts, database schema, migrations, messaging (Kafka, NATS, ZMQ), cache, money handling, or tests. Also when choosing infrastructure for a Go service. Go backend only.
---

# Go Backend Standards

The owner's defaults for Go services. Apply them to new work without being asked. Preserve
explicit user choices and established repository conventions, and name the deviation and its
reason when the task or the repo forces one. Apply only the branches the task touches; a small
fix does not redesign the service.

## Correctness first

Exact monetary values with explicit currency and rounding; retries that cannot repeat an
effect; stated transaction, delivery, ordering, and consistency boundaries. A finding carries a
failure mode and evidence; a deviation from a default without a failure mode is labelled
`Convention`. Severity rules live in [go-review.md](go-review.md).

## Hard defaults

| Concern | Default | Why |
|---|---|---|
| Layout | `cmd/<svc>`, `internal/domain`, `internal/usecase`, `internal/port`, `internal/adapter/{grpc,http,postgres,kafka,cache}` | domain imports nothing from adapters; adapters are swappable |
| Money | `shopspring/decimal` in Go, `NUMERIC(19,4)` in Postgres, `Decimal128` in Mongo, ISO-4217 `currency` beside it; scale and rounding defined at domain boundaries | exact arithmetic for rates, tax, splits; matches DB types |
| Contracts | spec-first: proto via `buf`, OpenAPI 3 via `oapi-codegen`; contracts committed and reviewed before code | tests and clients derive from the contract |
| Services | one service per bounded domain; gRPC between services; server streaming for large or long-lived responses | ownership follows data |
| HTTP lists | batch endpoints where a client would loop; cursor pagination for user-facing lists; `limit`/`offset` with a fixed unique `ORDER BY` for statistics, exports, and admin scans over a closed past range | scans cover a fixed sorted range; offset restarts are trivial; an exact snapshot uses one `REPEATABLE READ` transaction |
| Events | outbox table written in the same transaction; in-service relay; inbox table on every consumer, business change in the inbox transaction; at-least-once delivery, single committed effect | no lost or duplicated side effects |
| Idempotency | every mutating request and every external side effect carries an idempotency key scoped per caller and operation | retries are safe; provider support, scope, and retention are verified |
| Kafka | `twmb/franz-go`; partition key = entity id (user, order); partition count derived from measured throughput plus headroom; transactions only for consume-transform-produce | per-entity ordering, given per-aggregate dispatch |
| Native stack | NATS JetStream when self-hosted without Kafka; ZeroMQ for brokerless point-to-point | fits the infra that exists |
| Database | Postgres when relations exist, MongoDB when they do not; 5NF; `EXPLAIN (ANALYZE, BUFFERS)` on every repository query | plans, not guesses |
| Outbox table | range-partitioned by quarter from the first migration | drop resolved quarters instead of vacuuming them |
| Migrations | `goose`, SQL files with Up and Down, run as a deploy step, expand/contract | reversible where possible, documented recovery where not |
| Cache | distributed `memcached`, cache-aside, invalidate after commit; `hazelcast` when memcached lacks the needed structures | one cache tier shared by all replicas |
| Media | `ffmpeg`/`ffprobe` in a worker, frames and transcodes to object storage | CPU-bound work off the API path |
| Tests | TDD; integration tests via `testcontainers-go`; black-box autotests in a sibling `<svc>-autotests` repo for externally consumed services | infra bugs surface before deploy |

## Branches

Read the reference for the branch you are in; each ends with a done-when check.

- Writing or reviewing Go code: [go-review.md](go-review.md). Linters from [golangci.yml](golangci.yml) cover the mechanical mistakes; the reference covers the judgment calls.
- Designing a service, API, or messaging flow: [service-design.md](service-design.md). Start with the infrastructure facts you have and state the missing ones as assumptions.
- Designing a schema, query, partition, migration, or cache: [data-design.md](data-design.md).
- Writing tests or setting up a repo's test layout: [testing.md](testing.md).

## Design output

A full service design lists, in this order: infra assumptions and open questions, the
contract, the schema, the event flow with its guarantee, the idempotency points, the test plan.
A focused task includes only the parts it touches.
