# Go code review

Linters in `golangci.yml` catch the mechanical mistakes: unchecked errors, `%v` wrapping,
unclosed bodies, lost `cancel`, shadowing, fat interfaces, mixed receivers, `time.After` in
loops, requests without context. Run them first. Review the judgment calls below by hand.
Numbers refer to chapters of "100 Go Mistakes and How to Avoid Them".

## Stances

These decide severity regardless of what the linter says.

- `float64` or `int64` for money is a Blocker. Fix names `shopspring/decimal` and the currency field.
- A side effect after a database write (notify, publish, email) started from a goroutine is a
  Blocker. Fix names the outbox table.
- An external call that can be retried without an idempotency key is a Blocker.
- A change to a repository or adapter without a testcontainers test is a Major.
- An in-process map used as a cache is a Major when the service runs more than one replica.
  Fix names distributed memcached.
- An interface declared next to its implementation with one implementer is a Minor; interfaces
  live on the consumer side.

## Organization (1–17)

- Happy path left-aligned, early return; nested `else` after `return` is a smell.
- `init()` only for work that cannot fail; connections and config belong in `main` or constructors.
- Interfaces on the consumer side, 1–3 methods, introduced when a second implementation or a
  test needs them. Return concrete types, accept interfaces.
- `any` only for truly untyped data. Generics when the same logic serves several types.
- Embedding only when promoting methods is the point; a mutex is never promoted.
- Functional options for constructors with several optional parameters.
- Packages by domain and layer, never `utils`, `common`, `helpers`, `base`.

## Data types (17–29)

- `append` on a sub-slice can write into the parent; use the full slice expression `s[a:b:b]`
  or copy.
- Preallocate with `make([]T, 0, n)` when `n` is known.
- Return nil for "nothing"; encode `[]` only where the JSON contract requires it.
- Maps never shrink; a hot map with churn holds pointers or gets recreated.
- Comparing structs that hold slices or maps needs an `Equal` method, not `==` or `reflect.DeepEqual`.

## Control flow (30–35)

- `for _, v := range` copies `v`; mutate through the index or a pointer slice.
- A range expression is evaluated once; appending to the ranged slice inside the loop is a bug
  or a deliberate trick that needs a comment.
- `break` inside `select` inside `for` exits the `select`; use a label.
- `defer` inside a loop runs at function exit; wrap the body in a function.

## Strings (36–41)

- Length and indexing are bytes; iterate runes when characters matter.
- `strings.Builder` for loops, `TrimPrefix` vs `TrimLeft`, and copy a substring you keep from
  a large string.

## Functions and methods (42–48)

- Pointer receiver when the method mutates, or the type holds a mutex or is large; never mixed.
- Named results only when they add meaning or a deferred function must modify them.
- A nil pointer returned as an interface is not nil.
- `defer` evaluates its arguments immediately; wrap in a closure for late binding.

## Errors (48–54)

- Wrap once per layer with context: `fmt.Errorf("charge invoice %s: %w", id, err)`.
- Compare with `errors.Is`, extract with `errors.As`.
- Handle an error once: log or return, never both.
- Ignore explicitly: `_ = f()` with the reason on the same line.
- Errors from `defer`: `defer func() { err = errors.Join(err, rows.Close()) }()`.
- Sentinel errors live in the domain package; adapters translate driver errors
  (`sql.ErrNoRows` becomes `domain.ErrNotFound`).
- Panic only for programmer errors at startup.

## Concurrency (55–75)

- Every goroutine has an owner that knows when it stops: context, `errgroup`, or a done channel.
- `context.Context` is the first parameter and is never stored in a struct.
- Unbuffered channels by default; a buffer size comes with a reason (worker pool, batch size).
- `chan struct{}` for signals; close to broadcast.
- `wg.Add` before `go`, not inside the goroutine.
- `errgroup` for fan-out with cancellation.
- Sync types are never copied: pointer receivers, no struct copies, no passing by value.
- Slices and maps shared between goroutines are guarded by a mutex; an append-only cache may
  use `sync.Map`.
- `-race` runs in CI for every package.

## Standard library (75–84)

- `time.Duration` arguments: `time.Sleep(100)` is 100ns.
- `time.After` in a loop leaks until it fires; use `time.NewTimer` with `Stop` or a `Ticker`.
- JSON: embedded types hijack marshalling, `time.Time` monotonic readings break `==`,
  numbers into `any` become `float64`; use typed structs or `json.Number`.
- `database/sql`: `sql.Open` does not connect, call `PingContext`; set `SetMaxOpenConns`,
  `SetMaxIdleConns`, `SetConnMaxLifetime`; nullable columns are `sql.NullX` or pointers;
  always `rows.Close()` and `rows.Err()`.
- `http.Client` and `http.Server` need explicit timeouts; `return` after `http.Error`.
- Close the body, the file, the rows; check the error on writes.

## Testing (82–90)

- `//go:build integration` on testcontainers suites; unit tests run without Docker.
- `-race` and `-shuffle=on` always; `t.Parallel()` where the test is isolated.
- Table-driven tests; synchronization instead of sleeps; an injected clock for time; `httptest`
  for HTTP.
- Benchmarks reset the timer and sink results so the compiler keeps the work.

## Optimization (91–100)

- Profile with `pprof` before touching a hot path.
- Reduce allocations first: reuse buffers, `sync.Pool`, `Builder.Grow`; then consider layout
  (false sharing padding, struct alignment).
- In containers, Go before 1.25 needs `GOMAXPROCS` set to the CPU quota (`uber-go/automaxprocs`);
  set `GOMEMLIMIT` to the memory limit minus headroom.

Done when: every stance above is checked against the diff, linters ran or their absence is
stated, and each finding carries a severity and a concrete fix.
