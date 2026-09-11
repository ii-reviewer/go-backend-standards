# Baseline (RED phase): agent behaviour without the skill

Recorded 2026-09-11. Scenarios in `docs/specs/scenarios.md`.

## Scenario B: Go PR review (billing.go with seeded mistakes)

Caught without the skill: float money, missing idempotency key, double charge on retry,
unsynchronized map read, no `http.Client` timeout, `NewRequest` without ctx, body not closed,
`defer` in loop, lost `cancel`, fire-and-forget goroutine, `%v` instead of `%w`,
`time.After` in loop, error leak to client, fat `Repo` interface.

Deviations from the standards:

| Topic | Baseline choice | Standard |
|---|---|---|
| Money fix | `int64` minor units, decimal only as an aside | `shopspring/decimal` + `NUMERIC(19,4)` |
| Tests | "add unit tests" and a race test | TDD, integration via testcontainers |
| Notify fix | outbox mentioned as one option among others | outbox + inbox is the default |
| Cache | invalidate + TTL on the in-process map | distributed memcached; in-process map is not a cache tier |

Conclusion: the base model already finds mechanical and most logic bugs. `go-review.md`
needs only the owner's stances and the judgment-call items, not the full list of 100 mistakes.

## Scenario A: design the billing service

Matched the standards without the skill: clean/hexagonal layout, Postgres, goose, transactional
outbox with `FOR UPDATE SKIP LOCKED` relay, franz-go, partition key = invoice id, idempotency
key on create, PSP event id dedupe, buf + oapi-codegen, gRPC for sync calls, testcontainers
for adapters, an infra-discovery question list at the end.

Deviations from the standards:

| Topic | Baseline choice | Standard |
|---|---|---|
| Money | `int64` minor units + `bigint`; decimal only for fee math | `shopspring/decimal` + `NUMERIC(19,4)` |
| Cache | Redis (rueidis, client-side cache) | distributed memcached; hazelcast when that is not enough |
| Statistics scan | keyset cursor over gRPC stream, "no offset pagination" | limit/offset for statistics and other full scans |
| Outbox table | unpartitioned, partial index on unpublished | range-partitioned by quarter |
| Consumer dedupe | "consumers dedupe on event_id", no table named | inbox table on every consumer |
| Test process | tests listed per layer, not test-first | TDD; sibling autotests repo for externally consumed services |
| Schema review | none | 5NF check, `EXPLAIN ANALYZE` every query |
| Infra discovery | asked last, as a list | asked first; answer picks Kafka (managed or own) vs NATS JetStream vs ZMQ |

Conclusion: the skill earns its load on stances, not on generic architecture. Every deviation
above becomes a row in the hard-defaults table in `SKILL.md`, phrased as a positive default
with its reason, since the baseline argued the opposite on pagination and cache.
