# go-backend-standards skill: design

Date: 2026-09-11

## Goal

One Claude Code skill that encodes the owner's Go backend engineering standards so that
every code-writing, code-review and service-design task applies them without being asked.
Replaces the current habit of manually invoking assorted agents and skills per task.

## Packaging

- Repo `ii-reviewer/go-backend-standards` is the skill. `SKILL.md` sits at the repo root.
- Installed by symlink: `~/.claude/skills/go-backend-standards -> ~/git/own/go-backend-standards`.
- Model-invoked (has a `description`). Fires on its own when the task is Go backend work.
- Language: English throughout.

## Structure

```
SKILL.md            router + hard defaults; < 500 words
go-review.md        judgment-call checklist distilled from "100 Go Mistakes"; linters cover the rest
service-design.md   DDIA-style checklist: infra discovery, outbox/inbox, idempotency, delivery
                    guarantees, transport choice, Kafka partitioning
data-design.md      Postgres vs MongoDB, normalization, indexes, EXPLAIN ANALYZE, partitioning
testing.md          spec-first, TDD, testcontainers, autotest repo
golangci.yml        reusable linter config that automates the mechanical part of the 100 mistakes
docs/specs/         design notes (this file)
```

`SKILL.md` inlines what every branch needs (hard defaults) and points to one reference file
per branch. Reference files load only when their branch is active.

## Hard defaults (inlined in SKILL.md)

| Concern | Default |
|---|---|
| Layout | clean/hexagonal: `cmd/<svc>`, `internal/domain`, `internal/usecase`, `internal/port`, `internal/adapter/{grpc,http,postgres,kafka}` |
| Money | `shopspring/decimal`; `NUMERIC(19,4)` in Postgres, `Decimal128` in Mongo; currency as separate ISO-4217 field |
| Contracts | spec-first: proto via `buf`, OpenAPI 3 via `oapi-codegen`; contracts committed before code |
| Sync RPC | gRPC between services; server streaming when a response is large or long-lived |
| HTTP API | batch endpoints where the client would otherwise loop; cursor pagination for user-facing lists; limit/offset for statistics scans |
| Events | outbox + inbox tables, idempotency keys, at-least-once delivery; relay = in-service poller |
| Kafka | `twmb/franz-go`; partition key = entity id (user/order); partition count derived from load; transactions optional |
| Native infra | NATS JetStream when the stack is self-hosted; ZMQ when a broker is overkill |
| Media | `ffmpeg` for frame extraction and transcoding |
| DB | Postgres when relations exist, MongoDB when they do not; 5NF; every query `EXPLAIN ANALYZE`d |
| Outbox table | range-partitioned by quarter so old partitions drop instead of vacuum |
| Migrations | `goose` |
| Cache | distributed `memcached`; `hazelcast` when memcached is not enough; Redis is not the default |
| Tests | TDD; integration via `testcontainers-go`; autotests in a sibling repo when the service is externally consumed |

## Triggers (description branches)

- writing or refactoring Go backend code
- reviewing a Go pull request or diff
- designing a service, API, database schema, or messaging flow
- choosing infrastructure (broker, cache, database)

## Verification

RED: two subagents ran without the skill on (a) a billing-service design task and (b) a Go PR review.
Their deviations from the table above are recorded in `docs/specs/baseline.md` and drive the
wording of the skill.

GREEN: the same scenarios with the skill present must produce: decimal money, goose, outbox +
idempotency key, memcached, testcontainers, limit/offset for the statistics scan, partition key
by entity, and an explicit infra-discovery question.

## Out of scope

- Frontend, Python, and project-specific configuration.
- Anything a linter enforces mechanically; that lives in `golangci.yml`.
