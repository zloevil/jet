---
name: jet-toolkit
description: >-
  Use this skill whenever working on a Go service that is (or should be) built on
  the `jet` toolkit — imported as `github.com/zloevil/jet` or, in the monorepo, as
  `gitlab.monowork.tech/back/kit`. jet ships production-grade building blocks
  (structured logger, AppError model, request context, typed config, service
  lifecycle/`cluster`, panic-safe goroutines, retry, Kafka, gRPC/HTTP servers,
  Postgres/Redis/Mongo/etc. adapters, healthcheck, metrics) so a service should
  almost never hand-roll this infrastructure. Trigger when IMPLEMENTING a new
  feature, endpoint, repository, worker, consumer, or whole service in such a repo
  (so it's built on jet from the start), and especially when REFACTORING or
  REVIEWING — when the user wants to "убрать велосипеды / выпилить бойлерплейт /
  использовать jet (kit) / переписать на jet / почему руками, есть же jet",
  replace custom logging/error/config/retry/goroutine/kafka/db plumbing with jet
  equivalents, or asks why hand-written infra exists where jet already provides it.
  Fires on jet/kit component names (CLogger, AppError, AppErrBuilder, cluster,
  Bootstrap, CLoggerFunc, goroutine.New, NewAppErrBuilder, pg.Open, kafka.NewBroker)
  and on Go services in the fin domain (killjoy, ton-service, payouts, swapper,
  scavenger). This is the source of truth for using jet idiomatically and for
  spotting reinvented wheels. NOT for developing the jet library itself, and NOT
  for Go code that has nothing to do with jet/kit.
---

# Working with the jet toolkit

`jet` (public mirror `github.com/zloevil/jet`; internal `gitlab.monowork.tech/back/kit`)
is a pragmatic Go microservice toolkit that already implements the things most
services rewrite from scratch: structured logging, a structured error model,
request-context propagation, typed config, service lifecycle, panic-safe
concurrency, retries, Kafka/gRPC/HTTP, storage adapters, healthchecks and metrics.

The job of this skill is simple to state and important to get right: **in a jet
service, lean on jet instead of reinventing it.** Hand-written logger setups,
bespoke error structs, raw `go` routines, manual retry loops, custom config
parsers, ad-hoc Kafka wiring — these are the "велосипеды" the user wants gone.
Every one of them is a maintenance liability that diverges from how the other 20+
services behave, and jet already has the battle-tested version.

## First move: confirm you're in a jet project

Before applying any of this, make sure jet is actually in play — otherwise these
conventions don't apply and you'd be forcing a dependency:

```
rg 'zloevil/jet|back/kit' go.mod
```

If it's there (or the user is explicitly asking to adopt jet), proceed. If a Go
repo has nothing to do with jet/kit, this skill doesn't apply — don't push it.

## The reflex: check the catalog before writing infra

Whenever you're about to write code that *isn't* this service's business logic —
logging, errors, config, an HTTP/gRPC server, a DB pool, a Kafka consumer, a
goroutine, a retry loop, UUID/crypto/slice helpers — **stop and check
[references/catalog.md](references/catalog.md) first.** It's an "I'm about to
hand-roll X → use jet's Y" map covering the whole toolkit. The overwhelming
majority of plumbing is already there; writing your own is the exception that
needs justifying, not the default.

Read [references/conventions.md](references/conventions.md) for *how* to use the
core primitives idiomatically — the `AppError` builder chain, the `CLoggerFunc`
pattern and its concurrency caveat, request-context propagation, the
`cluster`/`Bootstrap` lifecycle, the layered architecture, storage repository
conventions (not-found = `(nil, nil)`), the `goroutine`/`retry` rules, config, and
testing with `jet.Suite`.

**Signatures: trust the source, not memory.** jet is a readable dependency. When
you need an exact constructor or argument order, run `go doc
github.com/zloevil/jet/<pkg>` or open the package (every one has a `doc.go` and
`Example` tests). The catalog tells you *what exists and what to reach for*; the
source confirms *how to call it*. Don't emit a jet call you haven't verified.

## Two modes

### Building something new

When adding a feature, endpoint, repository, worker, consumer, or a whole service,
build it on jet from the first line:

1. Glance at the catalog for the components this feature touches (e.g. a new
   consumer → `kafka`; a new stored entity → `storages/pg`; background work →
   `goroutine`).
2. Follow the conventions: errors via `NewAppErrBuilder`, logging via the injected
   `CLoggerFunc`, deps wired in `bootstrap`, layering respected, not-found as
   `(nil, nil)`, concurrency through `goroutine`.
3. Verify signatures against the source as you go.

For larger, multi-layer work (a new service, a cross-domain feature, a gateway
integration), the repo also ships two expert subagents — `jet-service-agent` (domain/
business services) and `jet-gateway-agent` (external-integration gateways) — and a
`kit-migration-architect` for systematic audits. Hand off to those for heavy
scaffolding; use this skill for the in-the-moment "use jet, not a велосипед" reflex.

### Refactoring out велосипеды

When the user wants reinvented infra replaced (or you spot it while doing other
work), the loop is: **find → map → replace → verify.**

1. **Find.** Search the project for the usual reinvention smells:
   ```
   rg -n 'errors\.New|fmt\.Errorf|status\.Errorf'   # custom errors instead of AppError
   rg -n '\bgo func'                                  # raw goroutines instead of goroutine.New
   rg -n 'logrus|zap\.|slog\.New|log\.Print'          # bespoke logging instead of CLogger
   rg -n 'for .*retr|time\.Sleep.*retry|backoff'      # hand-rolled retry instead of retry.Do
   rg -n 'viper\.|os\.Getenv'                          # ad-hoc config instead of ConfigLoader
   rg -n 'AutoMigrate'                                 # GORM automigrate instead of goose
   ```
2. **Map** each finding to its jet replacement via the catalog.
3. **Replace** following the conventions — preserving behavior. Don't just swap the
   call; adopt the idiom (e.g. an `errors.New` becomes a coded `AppError` with the
   right business/system type and status hint, created where the failure originates).
4. **Verify**: `go build ./...`, `go vet ./...`, and the package's tests. A
   refactor that doesn't compile or changes behavior is worse than the велосипед.

Don't rip out a custom implementation if it genuinely does something jet doesn't —
flag the gap and explain it instead of forcing a lossy swap. The goal is less
divergent boilerplate, not jet-for-its-own-sake.

## How to report

When you replace велосипеды, give the user a short ledger, not a wall of diff:
what was reinvented, what jet component replaced it, and anything you deliberately
left alone (with the reason). Lead with the summary; keep the build/test result at
the end so they can see it's still green.
