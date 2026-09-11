# Go code review

Linters in `golangci.yml` catch the mechanical mistakes: unchecked errors, `%v` wrapping,
unclosed bodies, lost `cancel`, shadowing, fat interfaces, mixed receivers, requests without context. Run them first. Review the judgment calls below by hand.
Numbers refer to chapters of "100 Go Mistakes and How to Avoid Them".

## Severity

A finding names the triggering condition, the observable impact, the evidence in the diff, and
a concrete fix. Blocker: a demonstrated path to severe harm (double charging, data loss, a
security breach). Major: a material correctness or reliability defect. Minor: a bounded
maintainability issue. Nit: optional polish. Missing context becomes a question or an explicitly
conditional finding. A deviation from an owner default without a failure mode is labelled
`Convention` and listed after the correctness findings; an established project choice gets a
`Convention` note, and a migration recommendation only when the task asks for one.

## Stances

- Money as `float64` is a Blocker; the fix names `shopspring/decimal` and the currency field.
  Money as checked `int64` minor units is a `Convention` finding recommending decimal.
- A required side effect after a database write (notify, publish, email) started from a
  goroutine is a Blocker; the fix names the outbox table. Best-effort telemetry may use an owned
  goroutine.
- A retriable external call with a non-idempotent effect and no idempotency key is a Blocker;
  the fix names the provider's key or a reconciliation path. Reads need no key.
- A change to a DB or broker adapter without a testcontainers test is a Major. Pure mapping code
  and HTTP transport use unit tests and `httptest`.
- An in-process map used as a cache for shared mutable data is a Major when the service runs
  more than one replica; the fix names distributed memcached. An immutable or versioned local
  cache is correct across replicas.
- An interface declared next to its only implementation is a Minor; interfaces live on the
  consumer side with 1–3 methods.

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
- Do not rely on map deletion to release backing capacity. Measure churn and retained memory;
  rebuild or bound the map if needed. Pointer values do not shrink its backing allocation.
- Structs containing slices/maps cannot use `==`; choose domain equality, `slices.Equal`,
  `maps.Equal`, or an appropriate deep comparison, including explicit nil/empty semantics.

## Control flow (30–35)

- `for _, v := range` copies `v`; mutate through the index or a pointer slice.
- A range expression is evaluated once; appending to the ranged slice inside the loop is a bug
  or a deliberate trick that needs a comment.
- `break` inside `select` inside `for` exits the `select`; use a label.
- `defer` inside a loop runs at function exit; wrap the body in a function.

## Strings (36–41)

- Length and indexing are bytes; iterate runes for Unicode code points; user-perceived characters may span multiple runes.
- `strings.Builder` for loops, `TrimPrefix` vs `TrimLeft`, and copy a substring you keep from
  a large string.

## Functions and methods (42–48)

- Pointer receiver when the method mutates, or the type holds a mutex or is large; prefer
  consistent receivers, but assess method sets and intentional exceptions before reporting a defect.
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
- Pass request-scoped `context.Context` first; avoid storing it in a long-lived struct.
  Assess compatibility wrappers or explicit task-lifetime objects by their actual lifetime.
- Unbuffered channels by default; a buffer size comes with a reason (worker pool, batch size).
- `chan struct{}` for signals; close to broadcast.
- For explicit `WaitGroup.Add`, call it before `go`; `WaitGroup.Go` is available from Go 1.25.
- `errgroup` for fan-out with cancellation.
- Sync types are never copied: pointer receivers, no struct copies, no passing by value.
- Slices and maps shared between goroutines are guarded by a mutex; an append-only cache may
  use `sync.Map`.
- `-race` runs in CI for every package.

## Standard library (75–84)

- `time.Duration` arguments: `time.Sleep(100)` is 100ns.
- Check the target Go version and timer mode. With Go 1.23+ timer semantics, GC can recover
  unreferenced timers: `time.After` in a loop is not by itself a leak. Older semantics (including
  `GODEBUG=asynctimerchan=1`) retain timers until firing. Reuse timers for measured allocation
  pressure; distinguish an idle timeout from a total deadline or periodic tick.
- JSON: embedded types hijack marshalling, `time.Time` monotonic readings break `==`,
  numbers into `any` become `float64`; use typed structs or `json.Number`.
- `database/sql`: `sql.Open` does not connect, call `PingContext`; set `SetMaxOpenConns`,
  `SetMaxIdleConns`, `SetConnMaxLifetime`; nullable columns are `sql.NullX` or pointers;
  always `rows.Close()` and `rows.Err()`.
- Bound HTTP operations with appropriate client/server timeouts or request deadlines; account
  for long-lived streaming. A zero client-wide timeout is not a defect if deadlines bound calls.
  `return` after `http.Error`. Translate internal/driver/provider errors to the public error
  contract; do not expose raw `err.Error()` to clients. Handle response encoding/write errors
  without attempting a second response after headers/body have already been sent.
- Close the body, the file, the rows; check the error on writes.

## Testing (82–90)

- `//go:build integration` on testcontainers suites; unit tests run without Docker.
- `-race` and `-shuffle=on` always; `t.Parallel()` where the test is isolated.
- Table-driven tests; synchronization instead of sleeps; an injected clock for time; `httptest`
  for HTTP.
- For Go 1.24+, prefer `for b.Loop()` where applicable; otherwise reset setup time and keep
  observable results so the compiler cannot remove the measured work.

## Optimization (91–100)

- Profile with `pprof` before touching a hot path.
- Reduce allocations first: reuse buffers, `sync.Pool`, `Builder.Grow`; then consider layout
  (false sharing padding, struct alignment).
- In containers, Go before 1.25 needs `GOMAXPROCS` set to the CPU quota (`uber-go/automaxprocs`);
  set `GOMEMLIMIT` to the memory limit minus headroom.

Done when: applicable checks above are assessed against the diff, linters ran or their absence is
stated, and each finding carries a severity and a concrete fix.

Version references: [time.After](https://pkg.go.dev/time#After),
[Go 1.23 timer compatibility](https://go.dev/wiki/Go123Timer),
[WaitGroup.Go](https://pkg.go.dev/sync#WaitGroup.Go), [B.Loop](https://pkg.go.dev/testing#B.Loop).
