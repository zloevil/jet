# jet conventions & idioms

How to use `jet` the way it's meant to be used. These patterns come from the
toolkit's own agent specs, verified against source. When a snippet here and the
actual source disagree, **the source wins** — `jet` is a normal dependency, so
`go doc github.com/zloevil/jet[/subpkg]` and reading the package's `*.go` /
`doc.go` is always the tiebreaker. Don't trust a remembered signature; confirm it.

## Table of contents
1. Errors — `AppError`
2. Logging — `CLogger`
3. Request context
4. Service lifecycle — `cluster`
5. Layered architecture
6. Storage repositories
7. Concurrency — `goroutine` / `retry`
8. Config
9. Testing

---

## 1. Errors — `AppError`

Never return `errors.New`, `fmt.Errorf`, or `status.Errorf` from service code.
Build an `AppError`: it carries a code, a type (business/system/panic), request
context, structured fields, and HTTP/gRPC status hints, and the transports know
how to translate it.

```go
return jet.NewAppErrBuilder("ORD-001", "order not found: %s", id).
    C(ctx).F(jet.KV{"id": id}).
    GrpcSt(uint32(codes.NotFound)).
    Business().
    Err()
```

Rules that matter:
- **Create errors deep** (domain/usecase/repository), **log once at the edge** —
  the http/grpc interceptor logs with stack and maps the status. Don't log-and-return.
- `.Business()` → a caller/input problem (defaults to HTTP 400). Set `GrpcSt` to a
  semantic code (NotFound, InvalidArgument, AlreadyExists, FailedPrecondition…).
- `.System()` → an internal failure (defaults to 500/Unknown). `.Wrap(cause)` the
  underlying error.
- `.C(ctx)` folds request-context fields in; `.F(jet.KV{...})` adds structured fields.
- There is **no `.Mth()` on the error builder** — `Mth` is logger-only.
- Per-service codes as constants: `ErrCodeXxx = "SVC-NNN"`, wrapped in small
  constructor funcs (`errors.ErrOrderNotFound(ctx, id)`).
- Inspect with `jet.IsAppErr(err) (*AppError, bool)` / `jet.IsAppErrCode(err, code)`.

## 2. Logging — `CLogger`

`CLogger` is a chainable, context-aware wrapper over `log/slog`. It is **not safe
for concurrent use**. The idiom is to pass a factory `jet.CLoggerFunc`
(`func() jet.CLogger`), never a bare `CLogger`, so each call site / goroutine gets
its own instance.

```go
logger := jet.InitLogger(&cfg.Log)            // *jet.Logger, once at boot
logFn := func() jet.CLogger { return jet.L(logger) }   // jet.CLoggerFunc — pass this around

logFn().Cmp("orders").Mth("Create").C(ctx).F(jet.KV{"id": id}).Inf("order created")
```

Chain: `.Cmp(component).Mth(method).C(ctx).F(jet.KV{...})` then a level
(`.Inf/.Warn/.Err/.Dbg`). If you must cache a `CLogger` across a goroutine
boundary, `Clone()` it first. Every `jet` component constructor takes a
`CLoggerFunc` — wire the same one through.

## 3. Request context

`jet.RequestContext` carries request id, session, user, roles, language, etc., and
**propagates across HTTP, gRPC and Kafka** so logs and errors correlate end-to-end.

```go
ctx = jet.NewRequestCtx().WithNewRequestId().ToContext(parent)
if rc, ok := jet.Request(ctx); ok { _ = rc.RequestId() }
```

You rarely wire propagation by hand: the grpc interceptors put it in/read it from
metadata, and `kafka.Decode[T]` reconstructs it from the message envelope (the
producer's `Send(ctx, …)` puts it there). Don't invent your own trace-id plumbing.

## 4. Service lifecycle — `cluster`

A service's `main` is one line; `cluster` loads config, builds the CLI (run +
`db-up`/`db-down` when migrations are configured), handles signals and ordered
shutdown, and drives your `Bootstrap`.

```go
func main() {
    cluster.New[config.Config]("orders", &app.App{}).
        WithDbMigration(func(c *config.Config) (any, error) { return c.DB.Master, nil }).
        Execute()
}
```

`Bootstrap` is three methods:
- `Init(ctx, cfgAny any) error` — type-assert the config, then **build dependencies
  inner-out**: repository → domain → usecase → transport. Returning an error aborts boot.
- `Start(ctx) error` — start background work (servers, consumers, health, metrics)
  **non-blocking**. `cluster` blocks on the signal; `Start` must return promptly. Use
  the `ListenAsync` / non-blocking `Listen()` forms.
- `Close(ctx)` — graceful shutdown in order: stop inbound traffic → drain workers →
  flush async messaging → observability → infra (Redis, DB, Mongo). `ctx` is still
  live; `cluster` cancels it after `Close` returns.

## 5. Layered architecture (dependency rule: everything points inward)

```
cmd/main          cluster.New + Execute
bootstrap         the ONLY place that imports concrete impls and wires the graph
transport/{grpc,http,kafka}   handlers only — return AppError, no business logic
usecase[/impl]    cross-domain orchestration; imports only domain (+ usecase) interfaces
domain[/impl]     business logic; domain holds entities, value objects, status consts, ALL interfaces
repository/storage    GORM DTOs + converters, implements domain.*Storage
repository/adapters   Kafka producers, gRPC clients to other services
```

- `domain` imports nothing internal (only `jet`, ctx, time).
- Never import `repository` or `transport` into `domain`/`usecase` — it breaks testability.
- Business logic lives in `domain/impl`; `usecase` only orchestrates across domains.

## 6. Storage repositories

- **Not-found returns `(nil, nil)`, never an error.** A decision layer above
  (`MustGet` in the domain service) turns `nil` into a business `AppError`.
- Repository works with **DTOs** (GORM structs embedding `pg.GormDto` for
  CreatedAt/UpdatedAt/DeletedAt); domain works with **entities**. Convert in sibling
  `*_converter.go` files.
- Use `pg` query scopes instead of raw GORM: `pg.Paging(req)`, `pg.Single()`,
  `pg.Update()`, `pg.Merge()`, `pg.WhereStrings(...)`; nullable strings via
  `pg.StringToNull` / `pg.NullToString`; JSONB via `pg.ToJsonb[T]` / `pg.FromJsonb[T]`.
- Schema is owned by **goose migrations** (`db/migrations/*.sql`), not GORM AutoMigrate.
- Compose one `Adapter` interface embedding every `domain.*Storage` over shared
  connections.

## 7. Concurrency — `goroutine` / `retry`

**Never write a raw `go func(){…}()`.** A panic in a bare goroutine crashes the
process and is invisible. Use the panic-safe, logged wrapper:

```go
goroutine.New().
    WithLoggerFn(logFn).Cmp("worker").Mth("run").
    WithRetry(goroutine.Unrestricted).WithRetryDelay(time.Second).
    Go(ctx, func() { /* body */ })
```

For bounded parallel work use the error group: `goroutine.NewGroup(ctx).WithLoggerFn(fn)…`
then `.Go(func() error)` × N and `.Wait()`.

For retrying a fallible operation (not a goroutine), use `retry.Do(ctx, cfg, fn)`
with `retry.DefaultConfig()` / `retry.RPCConfig()` — exponential backoff + jitter.
Mark an error `retry.NonRetryable(err)` to stop early. Don't hand-roll backoff loops.

## 8. Config

Compose your typed `Config` from jet component configs and let `cluster` load it
(YAML + env). Keep **secrets out of YAML** — pass via env, where a key like
`db.master.password` is overridden by `DB_MASTER_PASSWORD` (`.`→`_`, uppercased).

```go
type Config struct {
    Log         jet.LogConfig         `mapstructure:"log"`
    Grpc        grpc.ServerConfig     `mapstructure:"grpc"`
    DB          pg.DbClusterConfig    `mapstructure:"db"`
    Kafka       kafka.BrokerConfig    `mapstructure:"kafka"`
    Healthcheck jet.HealthcheckConfig `mapstructure:"healthcheck"`
}
```

Standalone (outside `cluster`): `jet.NewConfigLoader[Config]().WithPath(p).WithPrefix(px).Load()`.

## 9. Testing

- Pure logic: table-driven `testify/assert`.
- Components with deps (domain/usecase): the `jet.Suite` testify suite — call
  `s.Suite.Init(nil)` (or `Init(loggerFn)`) in `SetupSuite`, build the unit under
  test with mocks in `SetupTest`. Assert typed errors with the suite's
  `AssertAppErr(err, code)`.
- Mocks are generated with mockery into `internal/mocks`.
- Adapter tests that need real infra go behind `//go:build integration` and run with
  `go test -tags integration ./...`.
