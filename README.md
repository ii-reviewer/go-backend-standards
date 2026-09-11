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
| `go-review.md` | severity rules, owner stances, and the judgment-call checklist distilled from "100 Go Mistakes"; linters cover the rest |
| `service-design.md` | infra discovery, contracts, outbox/inbox, ordering, idempotency, Kafka, gRPC, media |
| `data-design.md` | Postgres vs Mongo, normalization, indexes and `EXPLAIN ANALYZE`, partitioning, goose, cache |
| `testing.md` | spec first, TDD, testcontainers, failure scenarios, autotests repo |
| `golangci.yml` | golangci-lint v2 config automating the mechanical half of the 100 mistakes |
| `docs/specs/` | design of the skill, test scenarios, and recorded runs without and with the skill |

Copy `golangci.yml` into a service repo as `.golangci.yml`.

## Maintaining the skill

- Validate the linter config with `golangci-lint config verify --config golangci.yml`.
- Guidance changes are tested like code: run the scenarios in `docs/specs/scenarios.md` with a
  fresh subagent before the change and after it, and compare the choices against the hard
  defaults table. `docs/specs/baseline.md` and `docs/specs/green.md` hold the recorded runs.
- Judge a run on correctness, missed defects, false positives, and preserved project choices;
  a matching library name alone is not a pass.
- Evaluation workspaces, generated artifacts, and test projects stay outside this repository
  (see `.gitignore`).
