# Testing

## Spec first

The contract (proto, OpenAPI) and a short design note exist before implementation. Tests derive
from the contract, so a test that cannot be traced to a contract line or a domain rule is
questioned in review.

## TDD loop

Red: one failing test for the smallest next behaviour. Green: the minimal code that passes.
Refactor with the test green. Commit per green step. A bug fix starts with the test that
reproduces the bug.

| Layer | Test | Infrastructure |
|---|---|---|
| `internal/domain` | table-driven unit tests | none |
| `internal/usecase` | unit tests with hand-written fakes for ports | none |
| `internal/adapter/*` | integration tests behind `//go:build integration` | testcontainers-go |
| service as a whole | black-box tests through the public contract | deployed environment |

Fakes are written by hand; a mock generator hides the port's shape and produces brittle
expectations.

## testcontainers

- One container per package, started in `TestMain`, goose migrations applied once.
- Each test isolates itself: a transaction rolled back in cleanup, or
  `TRUNCATE ... RESTART IDENTITY CASCADE`.
- Modules: `testcontainers-go/modules/postgres`, `modules/redpanda` for the Kafka API,
  `modules/nats`, `modules/mongodb`; memcached through `GenericContainer` with `memcached:alpine`.

```go
//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var dsn string

func TestMain(m *testing.M) {
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
	dsn, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	db := mustOpen(dsn)
	if err := goose.Up(db, "../../../migrations"); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = pg.Terminate(ctx)
	os.Exit(code)
}
```

Each test opens its own connection from `dsn`, runs inside a transaction, and rolls back in
`t.Cleanup`.

## Make targets

- `make test`: `go test -race -shuffle=on ./...`
- `make test-integration`: `go test -race -tags integration ./...`, Docker required on the runner
- `make lint`: `golangci-lint run`, `buf lint`, `buf breaking --against '.git#branch=main'`

All three block merge.

## Autotests repo

A service consumed by other teams or exposed externally gets a sibling repo
`<service>-autotests`. It talks to a deployed environment only through the public contract,
using clients generated from the same proto or OpenAPI, owns its fixtures and test accounts, and
runs on a schedule and on the service's release pipeline. It imports nothing from the service's
`internal` packages.

Done when: every layer in the table has tests of its kind, the integration suite runs in CI with
Docker, and an externally consumed service has its autotests repo named in the design.
