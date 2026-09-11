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
| `SKILL.md` | entry point: correctness requirements, owner defaults and which reference to read for which task |
| `go-review.md` | judgment-call checklist distilled from "100 Go Mistakes"; linters cover the rest |
| `service-design.md` | infra discovery, contracts, outbox/inbox, idempotency, Kafka, gRPC, media |
| `data-design.md` | Postgres vs Mongo, normalization, indexes and `EXPLAIN ANALYZE`, partitioning, goose, cache |
| `testing.md` | spec first, TDD, testcontainers, autotests repo |
| `golangci.yml` | golangci-lint v2 config automating the mechanical half of the 100 mistakes |

Copy `golangci.yml` into a service repo as `.golangci.yml`.

## Maintaining the skill

Validate the reusable linter configuration with `golangci-lint config verify --config golangci.yml`.
For guidance changes, compare behavior on realistic tasks before and after the change using an
isolated workspace. Check correctness, missed defects, false positives, unnecessary architecture,
and preservation of project choices; matching library names alone is not a correctness measure.
Keep evaluation runs, reports, test projects and generated artifacts outside this repository.
