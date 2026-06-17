# jet capability catalog — "don't build it, jet has it"

The whole point of this file: before writing any infrastructure, plumbing, or
utility code in a `jet` service, scan here first. If the need is in this list,
use the jet component instead of hand-rolling one.

**Signatures are a map, not gospel.** `jet` is a readable dependency — confirm the
exact constructor/params with `go doc github.com/zloevil/jet/<pkg>` or by opening
the package source (every package has a `doc.go` + `Example` tests). Names here are
correct; argument order may have evolved. Verify before you call.

## Quick "I'm about to write…" → use this instead

| If you're about to hand-roll… | Use | Package |
|---|---|---|
| a slog/zap/logrus setup, context-tagged logging | `jet.InitLogger` + `jet.L` → `CLogger` | `jet` |
| a custom error struct, error codes, http/grpc status mapping | `jet.NewAppErrBuilder` / `AppError` | `jet` |
| request-id / trace-id threading across services | `jet.RequestContext` (auto over grpc/kafka) | `jet` |
| a Viper/env config loader | `jet.NewConfigLoader[T]` | `jet` |
| `main` with signal handling + graceful shutdown + CLI | `cluster.New[Cfg](…).Execute()` | `cluster` |
| `go func(){ recover() … }()` | `goroutine.New()….Go(ctx, fn)` | `goroutine` |
| a parallel fan-out / errgroup | `goroutine.NewGroup(ctx)` | `goroutine` |
| a retry/backoff loop | `retry.Do` + `retry.DefaultConfig`/`RPCConfig` | `retry` |
| Kafka producer/consumer wiring, SASL, workers | `kafka.NewBroker` (+ cfg builders) | `kafka` |
| an in-process pub/sub between components | `event.New` bus | `event` |
| a "buffer items, flush by size/interval" accumulator | `batch.NewBatchWorker[T]` | `batch` |
| a gorilla/mux HTTP server with CORS/tracing/graceful stop | `http.NewHttpServer` + `BaseController` | `http` |
| a gRPC server/client (auth, health, recovery, ctx propagation) | `grpc.NewServer` / `grpc.NewClient` | `grpc` |
| a GORM/pgx pool, paging, JSONB, soft-delete columns | `pg.Open`, `pg.GormDto`, scopes | `storages/pg` |
| Redis client, distributed lock, priority queue | `redis.Open` (+ lock/queue helpers) | `storages/redis` |
| SQL schema migrations (with single-writer locking) | `migration` (goose-based) | `storages/migration` |
| k8s liveness/readiness HTTP probes | `jet.NewHealthCheck` | `jet` |
| Prometheus `/metrics`, business/system/panic error counters | `monitoring.NewMetricsServer` / `NewErrorMonitoring` | `monitoring` |
| a pprof endpoint | `profile` | `profile` |
| HS256 JWT generate/verify, internal service tokens | `jet.GenJwtToken` / `VerifyJwtToken` / `GenerateInternalAccessToken` | `jet` |
| AES encrypt/decrypt, object hashing | `jet.EncryptString` / `DecryptString` / `HashObj` | `jet` |
| UUID / nanoid / numeric-code generation | `jet.NewId` / `NanoId` / `NumCode` | `jet` |
| map/filter/reduce/group over slices (generics) | `jet.Map`/`Filter`/`Reduce`/`GroupBy`/`SliceToMap`/`ConvertSlice` | `jet` |
| millis↔time, HH:MM parsing, time ranges | `jet` datetime helpers (`Now`, `MillisFromTime`, `HourMinTime`) | `jet` |
| fluent input validation with business errors | `jet.NewValidator(ctx)` | `jet` |
| Mongo / ClickHouse / MinIO / Aerospike / Memcache clients | the matching adapter | `storages/*`, `memcache` |
| S3 / SQS / Elasticsearch / Centrifugo / reCAPTCHA / Excel | the matching package | `aws/*`, `elasticsearch`, `centrifugo`, `google`, `excel` |
| request/response RPC over Kafka | `rpc` + `rpc/client` + `rpc/server` | `rpc` |

## Root package `jet`

Core primitives every service uses. Sub-packages depend on the root only through
three of them: the **logger** (`CLoggerFunc`), **errors** (`AppErrBuilder`), and
**context** (`RequestContext`).

- **Logger:** `InitLogger(*LogConfig) *Logger`, `L(*Logger) CLogger`, type
  `CLoggerFunc = func() CLogger`. Chain: `Cmp/Mth/C/F` + `Inf/Warn/Err/Dbg`. `Clone()`
  before crossing a goroutine.
- **Errors:** `NewAppErrBuilder(code, fmt, args…)` → `.C/.F/.GrpcSt/.HttpSt/.Business/.System/.Panic/.Wrap/.Err`;
  `IsAppErr`, `IsAppErrCode`, `ErrPanic`.
- **Context:** `NewRequestCtx()` → `.WithNewRequestId/.WithSessionId/.WithUser/.WithRoles/.WithLang/.ToContext`;
  `Request(ctx)`, `MustRequest(ctx)`, grpc-metadata bridges.
- **Config:** `NewConfigLoader[T]()` → `.WithPath/.WithEnv/.WithPrefix/.Load`.
- **Health:** `NewHealthCheck(*HealthcheckConfig)` → `AddLivenessCheck/AddReadinessCheck/Start/Stop` (`/live`, `/ready`).
- **JWT:** `GenJwtToken`, `VerifyJwtToken`, `GenerateInternalAccessToken`, `ParseInternalAccessToken`.
- **Crypto:** `EncryptString`, `DecryptString` (AES-256-CFB, base64), `HashObj` (xxHash64).
- **Validation:** `NewValidator(ctx).Mth(…).NotEmptyString(attr,val).E()`.
- **IDs / strings:** `NewId` (uuid), `NanoId`, `NumCode`, `NewRandString`, `UUID`, `ValidateUUIDs`.
- **Datetime:** `Now`, `MillisFromTime`, `TimeFromMillis`, `HourMinTime`/`TimeRange`.
- **Generics:** `Map`, `Filter`, `Reduce`, `GroupBy`, `SliceToMap`, `ConvertSlice`,
  `MapValues`, `MapKeys`, `First`, `ToSet`, `ForAll`, `GetDefault`.
- **Common types:** `KV` (`map[string]any`), `PagingRequest`/`PagingResponse`
  (+ generic `…G[T]`), `SortRequest`, `Adapter[T]`, `AdapterListener[T]`, `Searchable`.

## Lifecycle & concurrency

- **`cluster`** — `New[TCfg](code, bootstrap)` → `.WithDbMigration(fn)` /
  `.WithConfigPathEnv` / `.WithMigrationSourceEnv` → `.Execute()`. `Bootstrap` =
  `Init/Start/Close`. CLI: run + `db-up`/`db-down`.
- **`goroutine`** — `New().WithLoggerFn(fn).Cmp(c).Mth(m).WithRetry(n).WithRetryDelay(d).Go(ctx, func())`;
  `NewGroup(ctx)…Go(func() error)…Wait()`. `Unrestricted` = retry forever.
- **`retry`** — `Do(ctx, cfg, fn)`, `DefaultConfig()`, `RPCConfig()`, `NonRetryable(err)`, `IsRetryable(err)`.

## Messaging

- **`kafka`** — `NewBroker(CLoggerFunc)` → `Init/AddProducer/AddSubscriber/DeclareTopics/Start/Close`;
  cfg builders `NewProducerCfgBuilder/NewSubscriberCfgBuilder/NewTopicCfgBuilder`; SASL plain/sha256/sha512;
  `Encode[T]`/`Decode[T]` carry the request context. `Producer.Send(ctx, key, msg)`.
- **`event`** — `New(CLoggerFunc)` bus: `Subscribe/SubscribeAsync/SubscribeOnce/Publish/WaitAsync`. In-process only.
- **`rpc`** (+ `rpc/client`, `rpc/server`) — request/response correlated over Kafka, with timeouts and a request pool.
- **`aws/sqs`** — SQS broker/subscriber with context propagation.

## Transport

- **`http`** — `NewHttpServer(*Config, CLoggerFunc)` (CORS, tracing, graceful, WS upgrade);
  `BaseController` for parsing vars/body, pagination, JSON/error responses (AppError → HTTP status); `RouteSetter`.
- **`grpc`** — `NewServer(service, CLoggerFunc, *ServerConfig)` (interceptors: ctx propagation, optional JWT auth,
  panic recovery, tracing, health service, AppError→status) and `NewClient(*ClientConfig)` (ctx + auth, `.Conn`).

## Storage (`storages/*`)

- **`pg`** — `Open(*DbConfig, CLoggerFunc) (*Storage, error)`; `GormDto` (audit cols); JSONB
  `ToJsonb/FromJsonb/MapToJsonb`; scopes `Paging/PagingLimit/Single/Update/Merge/OrderByCreatedAt/OrderByUpdatedAt/WhereStrings`;
  `StringToNull/NullToString`. Not-found = `(nil, nil)`.
- **`redis`** — `Open(ctx, *Config, CLoggerFunc)`; `NotFound` sentinel; distributed lock + priority queue helpers.
- **`migration`** — goose runner for Postgres/ClickHouse; advisory lock so one instance migrates at a time.
- **`mongodb` / `clickhouse` / `minio` / `aerospike`** — `Open(...)` → `*Storage` wrapping the native client.

## Observability

- **`monitoring`** — `NewMetricsServer(CLoggerFunc)` (`/metrics` from a private registry);
  `NewErrorMonitoring()` classifies `AppError` into business/system/panic counters.
- **`profile`** — pprof over a dedicated HTTP server.

## Other integrations

- **`batch`** — `NewBatchWorker[T](writer, *Options, CLoggerFunc)`; flush by `MaxItems` or `Interval`.
- **`elasticsearch`** — `NewEs(*Config, CLoggerFunc)`: index/bulk/search + mapping builder.
- **`aws/s3`** — presigned upload links, get/delete objects.
- **`centrifugo`** — real-time publish/subscribe.
- **`google`** — OAuth2 token handling + reCAPTCHA verification.
- **`memcache`** — TTL cache over patrickmn/go-cache.
- **`excel`** — read rows from XLSX.
- **`notification`** — resolve receivers from permission/resource policies into a typed `Notification`.
