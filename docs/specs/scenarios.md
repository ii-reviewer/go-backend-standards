# Test scenarios

Run each as a fresh general-purpose subagent. RED = without the skill (symlink removed or
`SKILL.md` absent). GREEN = with the skill installed. Compare the choices against the hard
defaults table in `2026-09-11-skill-design.md`.

## Scenario A: design a service

> You are helping a backend team. Do NOT create or edit any files. Do NOT ask clarifying
> questions; make reasonable assumptions and state them. Answer in English, concise, in one
> message.
>
> Task: design and scaffold a new Go microservice `billing` for a marketplace.
>
> Requirements: accepts "create invoice" requests from an API gateway; persists invoices;
> charges via an external PSP whose result arrives asynchronously via a webhook; when an
> invoice becomes paid, the `orders` and `notifications` services must be informed; 2k
> invoices/min at peak; a "list invoices" endpoint serves an admin dashboard and a monthly
> statistics job that scans everything; hot invoice reads (status polling) hit the service hard.
>
> Deliver, concretely, naming libraries/tools: (1) directory layout, (2) database choice and
> invoices schema with column types, (3) migration tooling, (4) how "invoice paid" reaches the
> other services and the delivery guarantee, (5) handling of duplicate PSP webhooks and client
> retries of "create invoice", (6) API shape for listing invoices for the dashboard and for the
> statistics job, (7) caching for hot reads, (8) test organisation and infrastructure,
> (9) Go type for money, (10) how `billing` calls `orders` synchronously if it ever needs to.
>
> Also: the team is not sure what messaging infra they already have. Say what you would ask
> them before finalizing, if anything.

Expected with the skill: clean/hexagonal layout; Postgres, `NUMERIC(19,4)` + currency column;
goose; outbox table + relay + inbox on consumers, at-least-once; idempotency key on create and
on webhook (PSP event id); cursor pagination for the dashboard, limit/offset for the statistics
scan; distributed memcached; TDD + testcontainers; `shopspring/decimal`; gRPC with proto via
buf; an explicit question about existing broker (AWS/GCP managed Kafka, own Kafka, or NATS).

## Scenario B: review a Go PR

Fixture: `docs/specs/fixtures/billing.go` (seeded mistakes: float money, no idempotency key,
unsynchronized map read, client without timeout, request without ctx, body not closed, `defer`
in loop, lost `cancel`, bare goroutine notify, `%v` wrapping, `time.After` in loop, error text
leaked to client, fat interface).

> You are reviewing a pull request for a Go billing microservice. Read the file
> `<fixture path>` and review it as a senior Go engineer would before approving. Do NOT create
> or edit any files. Do NOT ask clarifying questions. Answer in English in one message.
> Output: a list of findings `SEVERITY | line | issue | fix`, severity Blocker/Major/Minor/Nit,
> ordered by severity, then a one-line verdict. The PR is going to production for a payments
> system.

Expected with the skill: all seeded mistakes found; money fix names `shopspring/decimal`;
notify fix names outbox as the default; test fix names testcontainers; cache fix names
distributed memcached rather than a hardened in-process map.
