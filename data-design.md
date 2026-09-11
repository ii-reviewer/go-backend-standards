# Data design

## 1. Store choice

| Data | Store |
|---|---|
| entities with relations, transactions, reporting | Postgres |
| document-shaped records with per-record schema and no joins | MongoDB |
| both inside one service | Postgres owns the relational core; Mongo holds only the document part; cross-store consistency is stated |

Done when: each aggregate has exactly one owning store.

## 2. Postgres schema

- Normalize to 5NF: model keys and functional dependencies (3NF/BCNF), then split independent
  multivalued facts (4NF) and nontrivial join dependencies (5NF), each split with a lossless-join
  argument; the "whole key" mnemonic covers only 3NF/BCNF. Denormalized read models live in
  separate tables fed from the normalized source, with the refresh path documented.
- Types: `uuid` ids generated app-side as UUIDv7; `timestamptz` everywhere; money as
  `NUMERIC(19,4)` with a `char(3)` currency column, with range, scale, and rounding defined at
  the domain boundary; statuses
  as `text` with a CHECK or a lookup table; `jsonb` for opaque payloads or justified document
  access patterns with explicit validation and indexes.
- Constraints in the database: NOT NULL, CHECK, FK, UNIQUE. Application validation adds to
  them and never replaces them.
- `version int NOT NULL DEFAULT 1` on aggregates edited concurrently, for optimistic locking.

Done when: each table names its key, its normal-form justification for any repeated group, and
its constraints.

## 3. Indexes and query review

- Every repository query gets `EXPLAIN (ANALYZE, BUFFERS)` on representative volume before
  merge; the plan goes into the PR for anything beyond a primary-key lookup. `ANALYZE` executes
  the statement, so mutating queries run against disposable test data.
- Index the access paths the plan shows. For B-tree indexes, equality prefixes are a starting
  point; choose the remaining order from
  filtering, ordering, selectivity and the plan. A range column can prevent a later column from
  satisfying the requested global sort. Partial indexes for queues (`WHERE status = 'pending'`).
  `INCLUDE` columns for hot reads that would otherwise touch the heap.
- A sequential scan on a large table in a request path is a Major finding; when the plan shows
  the scan optimal for a broad result, the finding becomes a note with the plan attached.
- `pg_stat_statements` enabled; top queries by total time reviewed on a schedule.

Done when: each query has a plan and each index has a query that needs it.

## 4. Partitioning

Outbox and other append-only, time-bounded tables are range-partitioned by quarter from the
first migration. A quarter is dropped once every event in it is published and the replay/audit
horizon has passed, so vacuum never chases its dead tuples. Publishing updates rows, so active
partitions still get vacuumed; the partial index on unpublished rows stays small.

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

Done when: the outbox DDL is partitioned, and partition creation, safe retention, and
monitoring for missing partitions or unresolved old events have named owners.

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
