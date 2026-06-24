# jet

[![Go Reference](https://pkg.go.dev/badge/github.com/zloevil/jet.svg)](https://pkg.go.dev/github.com/zloevil/jet)
[![CI](https://github.com/zloevil/jet/actions/workflows/ci.yml/badge.svg)](https://github.com/zloevil/jet/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zloevil/jet)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A pragmatic Go toolkit for building microservices — so you stop rewriting the same
logger setup, graceful shutdown, database pool and Kafka consumer in every service.

`jet` is extracted from an internal toolkit that has been running 20+ production
services for several years. It favors boring, explicit building blocks over magic.

## Install

```bash
go get github.com/zloevil/jet
```

Requires Go 1.26+.

## What's inside

| Package | What it gives you |
|---|---|
| `jet` (root) | Structured logger (`log/slog`-based), typed config loader, request context, `AppError` model, healthcheck, JWT, crypto, validators, watchdog, generics helpers |
| `jet/cluster` | Service lifecycle (`Bootstrap` + CLI), config loading, DB migrations |
| `jet/goroutine` | Panic-safe goroutines and error groups |
| `jet/retry` | Bounded retry with exponential backoff + jitter |
| `jet/kafka` | Kafka producer/subscriber (segmentio/kafka-go) with SASL, workers, context propagation, at-most/at-least-once delivery |
| `jet/http`, `jet/grpc` | HTTP (gorilla/mux) and gRPC servers with middleware |
| `jet/storages/pg` | PostgreSQL via GORM + JSONB/paging helpers |
| `jet/storages/{redis,mongodb,clickhouse,migration,minio,aerospike}` | Storage adapters |
| `jet/event`, `jet/batch` | In-process event bus, batch writer |
| `jet/monitoring`, `jet/profile` | Prometheus metrics, pprof server |
| `jet/aws/{s3,sqs}`, `jet/elasticsearch`, … | Additional integrations |

## Quick start

A minimal service. `cluster` wires config loading, signal handling and ordered
shutdown around your `Bootstrap`.

```go
package main

import (
	"context"
	"log"

	"github.com/zloevil/jet/cluster"
)

type Config struct {
	HTTP struct{ Port string }
}

type App struct{}

func (a *App) Init(ctx context.Context, cfg any) error {
	c := cfg.(*Config)
	_ = c // build dependencies here
	return nil
}

func (a *App) Start(ctx context.Context) error {
	// start background processes (servers, consumers, …)
	return nil
}

func (a *App) Close(ctx context.Context) {
	// release resources
}

func main() {
	svc := cluster.New[Config]("my-service", &App{})
	if err := svc.Execute(); err != nil { // runs the service CLI (db-up/db-down added when migrations are configured)
		log.Fatal(err)
	}
}
```

## Building blocks

### Logging

```go
logger := jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel, Format: jet.FormatterJson})

log := jet.L(logger)
log.Cmp("orders").Mth("Create").C(ctx).F(jet.KV{"id": id}).Inf("order created")
```

`CLogger` is a chainable, context-aware wrapper over `log/slog`. Many `jet`
components accept a `jet.CLoggerFunc` (`func() jet.CLogger`):

```go
logFn := func() jet.CLogger { return jet.L(logger) }
```

### Config

YAML files with environment-variable overrides, into your own typed struct:

```go
cfg, err := jet.NewConfigLoader[Config]().WithPath("./config.yml").WithPrefix("MYSVC").Load()
```

### Errors

`AppError` carries a code, type (business/system), request context, HTTP/gRPC
status hints and structured fields:

```go
return jet.NewAppErrBuilder("ORD-001", "order not found: %s", id).
	Business().C(ctx).F(jet.KV{"id": id}).Err()

if appErr, ok := jet.IsAppErr(err); ok {
	_ = appErr.Code() // "ORD-001"
}
```

### PostgreSQL

```go
db, err := pg.Open(&pg.DbConfig{
	Host: "localhost", Port: "5432", User: "app", Password: "secret", DBName: "app",
}, logFn)
```

### Kafka

```go
broker := kafka.NewBroker(logFn)
_ = broker.Init(ctx, &kafka.BrokerConfig{Url: "localhost:9092"})

_ = broker.AddSubscriber(ctx,
	kafka.NewTopicCfgBuilder("orders").Build(),
	kafka.NewSubscriberCfgBuilder().GroupId("my-service").Build(), // at-most-once; or .DeliveryGuarantee(kafka.AtLeastOnce)
	func(payload []byte) error { /* handle message */ return nil },
)

_ = broker.Start(ctx)
defer broker.Close(ctx)
```

### HTTP / gRPC

```go
httpSrv := http.NewHttpServer(&http.Config{Port: "8080"}, logFn)
httpSrv.Listen()

grpcSrv, _ := grpc.NewServer("my-service", logFn, &grpc.ServerConfig{Host: "0.0.0.0", Port: "50051"})
_ = grpcSrv.Listen(ctx)
```

## Testing

```bash
go test ./...                      # unit tests
go test -tags integration ./...    # integration tests (need real Postgres/Kafka/etc.)
```

Integration tests are guarded by the `integration` build tag and require the
corresponding services to be running.

## AI agents

The [`.agents/`](.agents/) directory ships two expert subagent definitions for building
services *on top of* `jet`. They encode the toolkit's API (verified against this source) and
its conventions — layering, error model, lifecycle, observability — so an agent can scaffold
and extend a service without rediscovering the patterns each time.

| Agent | Use it for |
|---|---|
| [`jet-service-agent`](.agents/jet-service-agent.md) | Domain / business microservices: a gRPC/HTTP service owning business logic and relational data, in the layered `cmd → bootstrap → transport / usecase / domain / repository` style. |
| [`jet-gateway-agent`](.agents/jet-gateway-agent.md) | Gateway / external-integration services: fronting an external protocol behind an internal gRPC/HTTP facade, managing a bounded pool of long-lived client sessions with reconnection and graceful drain. |

Each file is a self-contained system prompt (Markdown with `name`/`description` frontmatter):
point your agent harness at it, or read it as a hands-on guide to building a service with `jet`.

## Claude Code skill

The [`skills/jet-toolkit/`](skills/jet-toolkit/) directory is a [Claude Code](https://claude.com/claude-code)
skill that keeps an agent from reinventing what `jet` already provides. Where the `.agents/` specs
are heavyweight personas you hand a whole service to, the skill is a lightweight, always-available
reflex: whenever the agent is about to write infrastructure (logging, errors, config, retries,
goroutines, a Kafka consumer, a DB pool, …) in a `jet`-based service, it first checks the toolkit —
and when reviewing or refactoring, it hunts down hand-rolled boilerplate and replaces it with the
`jet` equivalent.

| File | What it holds |
|---|---|
| [`SKILL.md`](skills/jet-toolkit/SKILL.md) | Trigger rules, the "check the catalog before writing infra" reflex, and the build-new / refactor-out-boilerplate workflows. |
| [`references/catalog.md`](skills/jet-toolkit/references/catalog.md) | The "don't build it, `jet` has it" map: an *I'm-about-to-hand-roll-X → use `jet`'s Y* table plus a package-by-package capability list. |
| [`references/conventions.md`](skills/jet-toolkit/references/conventions.md) | How to use the core primitives idiomatically — the `AppError` builder, the `CLoggerFunc` pattern, request context, the `cluster` lifecycle, layering, storage conventions, concurrency, config and testing. |

To use it, copy `skills/jet-toolkit/` into your Claude Code skills directory (e.g.
`~/.claude/skills/`). It activates automatically when you work on a Go service that imports
`github.com/zloevil/jet`.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © Kukhtin Vasiliy
