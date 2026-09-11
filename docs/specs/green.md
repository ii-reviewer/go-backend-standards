# GREEN run: same scenarios with the skill installed

Recorded 2026-09-11. Prompts identical to the RED run; the skill was not named in either
prompt. Both agents reported `Skills used: go-backend-standards`, so discovery through the
description works.

## Scenario A: design the billing service

| Topic | RED (no skill) | GREEN (skill) |
|---|---|---|
| Money | `int64` minor units | `shopspring/decimal`, `NUMERIC(19,4)` + `char(3)`, string on the wire |
| Cache | Redis | memcached cache-aside, TTL per key class, invalidate after commit, singleflight |
| Statistics scan | keyset cursor, "no offset" | `limit`/`offset` over a closed past range with fixed `ORDER BY` |
| Outbox table | unpartitioned | range-partitioned by quarter, pg_partman two ahead |
| Consumer dedupe | "dedupe on event_id" | `inbox(event_id PK)` per consumer, guarantee sentence stated |
| Test process | per-layer list | TDD, testcontainers per adapter, `billing-autotests` sibling repo |
| Schema review | none | `EXPLAIN (ANALYZE, BUFFERS)` in every PR |
| Infra discovery | last | first, with NATS JetStream named as the self-hosted fallback |

Unchanged and correct in both runs: clean/hexagonal layout, Postgres, goose, transactional
outbox with `SKIP LOCKED` relay, franz-go, partition key by entity, idempotency key on create,
buf + oapi-codegen, gRPC for sync calls.

## Scenario B: review billing.go

All seeded mistakes found in both runs. Differences: money fix names `shopspring/decimal`
(RED: `int64`); goroutine notify is a Blocker with the outbox as the fix (RED: Major, outbox
as one option); in-process map cache is replaced by memcached (RED: hardened map); tests name
testcontainers (RED: unit only); fat interface gets the consumer-side rule. The agent stated
that linters were not run and why.

## Verdict

Every expected outcome in `scenarios.md` holds. No wording changes needed after the first GREEN
run.

## PR #1 check (2026-09-11)

PR #1 "Refine Go backend guidance" rewrote defaults as preferences with exceptions and removed
`docs/specs`. Scenario A against the PR branch as submitted regressed three rows: statistics
scan went back to keyset over a gRPC stream, the outbox started unpartitioned "until volume
proves the need", normalization stopped at 3NF; the infra questions added "Redis fine, same
cache-aside". The technical corrections in the PR (Go 1.23 timers, `WaitGroup.Go`, `b.Loop`,
`EXPLAIN ANALYZE` executes, singleflight scope, ordering under `SKIP LOCKED`, partitioned
uniqueness, cache fill/delete race) were correct and kept.

After restoring the hard defaults on the same branch (commit "Keep hard defaults while adopting
the correctness fixes"), both scenarios pass again: scenario A gives `limit`/`offset` inside one
`REPEATABLE READ` transaction for the statistics scan, quarterly outbox partitions in the first
migration, decimal money, memcached, inbox per consumer, testcontainers and `billing-autotests`;
scenario B keeps every stance at its severity and the agent ran `golangci-lint` with the skill
config on its own. Lesson recorded: an exception clause on a default row reopens the choice the
baseline got wrong; state exceptions as conditions on observable facts, never as "also valid".
