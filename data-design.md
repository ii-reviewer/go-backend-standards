# Data design

## 1. Store choice

| Data | Store |
|---|---|
| entities with relations, transactions, reporting | Postgres |
| document-shaped records whose access patterns/operational needs justify a document store | MongoDB; compare with Postgres JSONB first |
| both inside one service | justify the second store; define ownership and cross-store consistency |

Done when: each aggregate has exactly one owning store.

## 2. Postgres schema

- Start with explicit keys and functional dependencies, usually 3NF/BCNF. Check 4NF for
  independent multivalued facts and 5NF for nontrivial join dependencies; the usual "whole key"
  mnemonic is not a definition of 5NF. Decompose only with a lossless reconstruction argument.
  Document deliberate denormalization and how its copies stay consistent.
- Types: `uuid` ids generated app-side as UUIDv7; `timestamptz` everywhere; money as
  `NUMERIC(19,4)` with a `char(3)` currency column by default (validate required range/scale,
  currency and rounding; checked integer minor units are valid for suitable domains); statuses
  as `text` with a CHECK or a lookup table; `jsonb` for opaque payloads or justified document
  access patterns with explicit validation and indexes.
- Constraints in the database: NOT NULL, CHECK, FK, UNIQUE. Application validation adds to
  them and never replaces them.
- `version int NOT NULL DEFAULT 1` on aggregates edited concurrently, for optimistic locking.

Done when: each table names its key, its normal-form justification for any repeated group, and
its constraints.

## 3. Indexes and query review

- Review query access paths; capture `EXPLAIN (ANALYZE, BUFFERS)` for new or materially changed
  nontrivial queries on representative volume. `ANALYZE` executes the statement: run mutating
  queries in disposable test data, not production merely to obtain a plan.
- Index the access paths the plan shows. For B-tree indexes, equality prefixes are a starting
  point; choose the remaining order from
  filtering, ordering, selectivity and the plan. A range column can prevent a later column from
  satisfying the requested global sort. Partial indexes for queues (`WHERE status = 'pending'`).
  `INCLUDE` columns for hot reads that would otherwise touch the heap.
- A sequential scan can be optimal for a broad result. Report it as a performance defect only
  when representative plans/latency show material cost against the request budget.
- `pg_stat_statements` enabled; top queries by total time reviewed on a schedule.

Done when: changed nontrivial queries have representative plans and each added index has a
justified access path.

## 4. Partitioning

Use quarterly range partitions when outbox volume and retention justify their operational cost.
An unpartitioned table with a pending-row index is a valid starting point. Outbox rows are updated
when published, so active partitions still need vacuum. Drop a partition only after all its events
are resolved and the replay/audit horizon has expired; age alone is insufficient.

For a partitioned design, the following DDL illustrates retention structure, not a complete relay.
Generate `event_id` once in the application and preserve it on every retry. The composite key is
required by Postgres partitioning and does not enforce global uniqueness of `event_id` alone.
If that uniqueness must be DB-enforced, use an unpartitioned identity registry or unpartitioned
outbox. Allocate `aggregate_sequence` under the aggregate lock; it is not the identity counter.

```sql
CREATE TABLE outbox (
    id            bigint GENERATED ALWAYS AS IDENTITY,
    event_id      uuid        NOT NULL,
    aggregate_sequence bigint NOT NULL,
    aggregate_id  uuid        NOT NULL,
    event_type    text        NOT NULL,
    payload       bytea       NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    published_at  timestamptz,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE outbox_2026q3 PARTITION OF outbox
    FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');
CREATE TABLE outbox_2026q4 PARTITION OF outbox
    FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
```

Partitions for the next two quarters are created ahead by a migration or `pg_partman`.
Business tables partition only when a plan or vacuum lag proves the need; a unique constraint
on a partitioned table must include the partition key.

Done when: the partitioning choice is justified; if partitioned, creation, safe retention,
and monitoring for missing partitions/unresolved old events have named owners.

## 5. Migrations with goose

- `migrations/NNNNN_<verb>_<object>.sql`, with `-- +goose Up` and a `-- +goose Down` for reversible changes.
  A destructive data change is not made reversible by recreating empty columns: document a
  forward fix or restore procedure and explicitly fail unsupported downs rather than faking one.
- Run as a deploy step (`goose up` in a job before the new version starts), never on service
  boot in production.
- Expand and contract for column changes; `CREATE INDEX CONCURRENTLY` inside
  `-- +goose NO TRANSACTION`.
- CI runs up/down/up for reversible migrations against testcontainers Postgres; test the stated
  recovery path and expected down refusal for irreversible migrations.

Done when: forward migration and its documented rollback/forward-recovery behavior are verified in CI.

## 6. MongoDB

- Model for the read pattern: embed what is read together, reference what changes independently.
- Indexes declared in code at startup; every query checked with `explain("executionStats")`.
- Money as `Decimal128`.

## 7. Cache tier

- Distributed memcached, cache-aside: read the cache, on miss read the database and set with a
  TTL per key class; on write, invalidate after the transaction commits, and let the outbox
  carry the invalidation when a crash between commit and delete would matter.
- `singleflight` per key coalesces requests only within one process, not across replicas.
  Bound cache size, use TTL jitter where helpful, and assess miss load against DB capacity.
- Hazelcast when the cache needs structures memcached lacks: near-cache, distributed locks,
  entry processors, or survival across restarts.
- A post-commit delete can race with an earlier reader that fills stale data after the delete.
  State the allowed staleness; use versioned keys or a coordinated version check where stronger
  consistency is needed. Outbox invalidation closes the crash window, not every fill/delete race.
- The cache is never a source of truth; define miss/failure fallback and protect DB capacity.
  Local immutable/versioned caches are valid when consistency and memory bounds allow them.

Done when: each cached key class has a TTL, an invalidation trigger, and a named owner.
