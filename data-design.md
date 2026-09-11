# Data design

## 1. Store choice

| Data | Store |
|---|---|
| entities with relations, transactions, reporting | Postgres |
| document-shaped records with per-record schema and no joins | MongoDB |
| both inside one service | Postgres owns the relational core; Mongo holds only the document part |

Done when: each aggregate has exactly one owning store.

## 2. Postgres schema

- Normalize to 5NF: every non-key column depends on the key, the whole key, and nothing but the
  key; no multi-valued fact in one row; a join dependency becomes its own table. Denormalized
  read models live in separate tables fed from the normalized source.
- Types: `uuid` ids generated app-side as UUIDv7; `timestamptz` everywhere; money as
  `NUMERIC(19,4)` with a `char(3)` currency column; statuses as `text` with a CHECK or a
  lookup table; `jsonb` only for opaque payloads.
- Constraints in the database: NOT NULL, CHECK, FK, UNIQUE. Application validation adds to
  them and never replaces them.
- `version int NOT NULL DEFAULT 1` on aggregates edited concurrently, for optimistic locking.

Done when: each table names its key, its normal-form justification for any repeated group, and
its constraints.

## 3. Indexes and query review

- Every repository query gets `EXPLAIN (ANALYZE, BUFFERS)` on production-like volume before
  merge; the plan goes into the PR for anything beyond a primary-key lookup.
- Index the access paths the plan shows. Composite column order: equality columns, then the
  range column, then the sort. Partial indexes for queues (`WHERE status = 'pending'`).
  `INCLUDE` columns for hot reads that would otherwise touch the heap.
- A sequential scan on a large table in a request path is a Major finding.
- `pg_stat_statements` enabled; top queries by total time reviewed on a schedule.

Done when: each query has a plan and each index has a query that needs it.

## 4. Partitioning

Outbox and other append-only, time-bounded tables are range-partitioned by quarter. Old
quarters are dropped, so vacuum never chases their dead tuples, and the partial index on
unpublished rows stays small.

```sql
CREATE TABLE outbox (
    id            bigint GENERATED ALWAYS AS IDENTITY,
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

Done when: the outbox DDL is partitioned and the partition-creation job is named.

## 5. Migrations with goose

- `migrations/NNNNN_<verb>_<object>.sql`, with `-- +goose Up` and a working `-- +goose Down`.
- Run as a deploy step (`goose up` in a job before the new version starts), never on service
  boot in production.
- Expand and contract for column changes; `CREATE INDEX CONCURRENTLY` inside
  `-- +goose NO TRANSACTION`.
- CI runs up, down, up against a testcontainers Postgres.

Done when: the migration set applies cleanly forward and backward in CI.

## 6. MongoDB

- Model for the read pattern: embed what is read together, reference what changes independently.
- Indexes declared in code at startup; every query checked with `explain("executionStats")`.
- Money as `Decimal128`.

## 7. Cache tier

- Distributed memcached, cache-aside: read the cache, on miss read the database and set with a
  TTL per key class; on write, invalidate after the transaction commits, and let the outbox
  carry the invalidation when a crash between commit and delete would matter.
- `singleflight` per key against stampedes.
- Hazelcast when the cache needs structures memcached lacks: near-cache, distributed locks,
  entry processors, or survival across restarts.
- The cache is never a source of truth; a miss always reaches the database.

Done when: each cached key class has a TTL, an invalidation trigger, and a named owner.
