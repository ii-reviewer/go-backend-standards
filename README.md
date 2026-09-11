# go-backend-standards

Go backend engineering standards packaged as a [Claude Code skill](https://docs.claude.com/en/docs/claude-code/skills).
Once installed, Claude applies these defaults whenever it writes, reviews, or designs Go backend
code: clean layout, decimal money, spec-first contracts, outbox and inbox, idempotency keys,
Kafka partitioning by entity, goose migrations, memcached, TDD with testcontainers.

## Install

The repo root is the skill directory. Symlink it into your personal skills folder:

```bash
git clone https://github.com/ii-reviewer/go-backend-standards ~/git/own/go-backend-standards
ln -s ~/git/own/go-backend-standards ~/.claude/skills/go-backend-standards
```

Claude Code picks it up on the next session. For one project only, clone into
`<repo>/.claude/skills/go-backend-standards` instead.

## Layout

| File | Purpose |
|---|---|
| `SKILL.md` | entry point: hard defaults table and which reference to read for which task |
| `go-review.md` | judgment-call checklist distilled from "100 Go Mistakes"; linters cover the rest |
| `service-design.md` | infra discovery, contracts, outbox/inbox, idempotency, Kafka, gRPC, media |
| `data-design.md` | Postgres vs Mongo, 5NF, indexes and `EXPLAIN ANALYZE`, partitioning, goose, cache |
| `testing.md` | spec first, TDD, testcontainers, autotests repo |
| `golangci.yml` | golangci-lint v2 config automating the mechanical half of the 100 mistakes |
| `docs/specs/` | design of the skill, baseline results, and test scenarios |

Copy `golangci.yml` into a service repo as `.golangci.yml`.

## Changing the skill

Skills are tested like code. Before editing guidance, run the scenarios in
`docs/specs/scenarios.md` with a fresh subagent and without the change, record what it does,
then edit, then run again. `docs/specs/baseline.md` holds the recorded behaviour without the
skill; append new baselines there.
