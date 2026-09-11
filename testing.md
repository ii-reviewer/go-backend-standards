# Testing

## Spec first

For new public behavior, define the applicable contract (proto or OpenAPI) and domain rules
before implementation. Tests cover observable behavior, invariants, and failure boundaries.
Regression, concurrency, and resource-lifetime tests need not map to a protocol field.
A focused fix can use the existing contract without creating a new design document.

## TDD loop

Red: one failing test for the smallest next behaviour. Green: the minimal code that passes.
Refactor with the test green. Keep commits coherent with the project workflow. A bug fix starts with the test that
reproduces the bug.

| Layer | Test | Infrastructure |
|---|---|---|
| `internal/domain` | table-driven unit tests | none |
| `internal/usecase` | unit tests with hand-written fakes for ports | none |
| DB/broker adapters | integration tests behind `//go:build integration` | testcontainers-go |
| pure adapter mapping / HTTP transport | unit tests / `httptest` | none unless external behavior is under test |
| service as a whole | black-box tests through the public contract | deployed environment |

Prefer small hand-written fakes; existing generated mocks are acceptable when they test
observable contracts without brittle call-order expectations.

## testcontainers

- A container per package in `TestMain` is a useful default. Bound startup and cleanup, close
  database pools, and ensure cleanup happens before `os.Exit` even on setup failure.
- Isolate each test using rollback only when all application queries use that same transaction.
  A separate HTTP handler/pool connection does not join it automatically. For committed or
  concurrent work, use a database/schema per test, or serialize tests with explicit cleanup.
  Shared `TRUNCATE` is incompatible with `t.Parallel()` against the same tables.
- Use matching testcontainers modules for Postgres, Kafka-compatible Redpanda, NATS, and MongoDB;
  memcached can use `GenericContainer`. Pin Go module versions and choose container versions
  that match the service's compatibility targets; update them deliberately.
- Ensure transaction rollback happens before canceling its lifetime context. `t.Context()` is
  canceled before `t.Cleanup` callbacks run; cleanup may need its own bounded context.
- Inspect the target service's Go and infrastructure versions when \1
```go
//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var dsn string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("billing"),
		postgres.WithUsername("billing"),
		postgres.WithPassword("billing"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = pg.Terminate(ctx) }()
	dsn, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	db := mustOpen(dsn)
	defer db.Close()
	if err := goose.Up(db, "../../../migrations"); err != nil {
		panic(err)
	}
	return m.Run()
}
```

Each test opens its own connection from `dsn` and isolates itself as described above.

## Failure scenarios

Check the boundary that supplies the guarantee: transaction rollback before effect/ack;
redelivery after commit; concurrent identical command keys; ambiguous external success;
provider dedupe expiry; two relay workers handling one aggregate; poison events and sequence
gaps; mutable source rows during export; cache fill racing invalidation. Use only scenarios
relevant to the changed behavior. Design review does not replace the service's executable
broker/PSP integration tests.

## Suggested service Make targets

- `make test`: `go test -race -shuffle=on ./...`
- `make test-integration`: `go test -race -shuffle=on -tags integration ./...`, Docker required on the runner
- `make lint`: `golangci-lint run`, `buf lint`, `buf breaking --against '.git#branch=main'`

Applicable checks block merge. Run `buf` only for services with proto contracts; compare breaking
changes against the actual PR base with the required git history. These targets belong in the
consuming service repository.

## Autotests repo

A service consumed by other teams or exposed externally gets a sibling repo
`<service>-autotests`. It talks to a deployed environment only through the public contract,
using clients generated from the same proto or OpenAPI, owns its fixtures and test accounts, and
runs on a schedule and on the service's release pipeline. It imports nothing from the service's
`internal` packages.

Done when: changed behavior has tests at the relevant layer, the integration suite runs in CI with
Docker, and an externally consumed service has its autotests repo named in the design.
