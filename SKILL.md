---
name: go-backend-standards
description: Use when writing, refactoring, or reviewing Go backend code, or when designing a Go service or its API, gRPC contracts, database schema, migrations, messaging (Kafka, NATS, ZMQ), cache, money handling, or tests. Also when choosing infrastructure for a Go service. Go backend only.
---

# Go Backend Standards

The owner's defaults for Go services. Apply them without being asked. When the task or the repo
forces a different choice, name the deviation and the reason in your answer.

## Hard defaults

| Concern | Default | Why |
|---|---|---|
| Layout | `cmd/<svc>`, `internal/domain`, `internal/usecase`, `internal/port`, `internal/adapter/{grpc,http,postgres,kafka,cache}` | domain imports nothing from adapters; adapters are swappable |
| Money | `shopspring/decimal` in Go, `NUMERIC(19,4)` in Postgres, `Decimal128` in Mongo, ISO-4217 `currency` beside it | exact arithmetic for rates, tax, splits; matches DB types |
| Contracts | spec-first: proto via `buf`, OpenAPI 3 via `oapi-codegen`; contracts committed and reviewed before code | tests and clients derive from the contract |
| Services | one service per bounded domain; gRPC between services; server streaming for large or long-lived responses | ownership follows data |
| HTTP lists | batch endpoints where a client would loop; cursor pagination for user-facing lists; `limit`/`offset` for statistics, exports, and admin scans | scans cover a fixed sorted range, offset restarts are trivial |
| Events | outbox table written in the same transaction; in-service relay; inbox table on every consumer; at-least-once delivery, idempotent effect | no lost or duplicated side effects |
| Idempotency | every mutating request and every external call carries an idempotency key | retries are safe |
| Kafka | `twmb/franz-go`; partition key = entity id (user, order); partition count derived from measured throughput plus headroom; transactions only for consume-transform-produce | per-entity ordering |
| Native stack | NATS JetStream when self-hosted without Kafka; ZeroMQ for brokerless point-to-point | fits the infra that exists |
| Database | Postgres when relations exist, MongoDB when they do not; 5NF; `EXPLAIN (ANALYZE, BUFFERS)` on every repository query | plans, not guesses |
| Outbox table | range-partitioned by quarter | drop old partitions instead of vacuuming them |
| Migrations | `goose`, SQL files with Up and Down, run as a deploy step | reversible, reviewable |
| Cache | distributed `memcached`, cache-aside, invalidate after commit; `hazelcast` when memcached lacks the needed structures | one cache tier, no extra datastore |
| Media | `ffmpeg`/`ffprobe` in a worker, frames and transcodes to object storage | CPU-bound work off the API path |
| Tests | TDD; integration tests via `testcontainers-go`; black-box autotests in a sibling `<svc>-autotests` repo for externally consumed services | infra bugs surface before deploy |

## Branches

Read the reference for the branch you are in; each ends with a done-when check.

- Writing or reviewing Go code: [go-review.md](go-review.md). Linters from [golangci.yml](golangci.yml) cover the mechanical mistakes; the reference covers the judgment calls.
- Designing a service, API, or messaging flow: [service-design.md](service-design.md). The first step is the infrastructure question.
- Designing a schema, query, partition, migration, or cache: [data-design.md](data-design.md).
- Writing tests or setting up a repo's test layout: [testing.md](testing.md).

## Design output

A design answer lists, in this order: infra assumptions and the questions still open, the
contract, the schema, the event flow with its guarantee, the idempotency points, the test plan.
