# Contributing to jet

Thanks for your interest in contributing! `jet` is a pragmatic toolkit for Go
microservices — contributions that keep it simple, explicit and well-tested are
very welcome.

## Getting started

Requires Go 1.26+.

```bash
git clone git@github.com:zloevil/jet.git
cd jet
go build ./...
go test ./...
```

A `Makefile` wraps the common tasks:

```bash
make build              # go build ./...
make test               # unit tests
make test-integration   # integration tests (require real services)
make vet                # go vet ./...
make lint               # vet + gofmt
```

## Conventions

- **English only.** All repository content — code, comments, identifiers, docs
  and commit messages — is in English.
- **Format and vet.** Run `gofmt` and `go vet` before pushing (`make lint`).
- **Errors.** Return errors built with `jet.NewAppErrBuilder` so they carry a
  code, a type (business/system) and HTTP/gRPC status hints.
- **Logging.** Use `CLogger`; clone it (`logger.Clone()`) before passing it into
  a goroutine — it is not safe for concurrent use.
- **Keep it boring.** Prefer the standard library and well-established
  dependencies; avoid premature abstraction.

## Tests

- Write tests alongside the code. Pure logic gets table-driven unit tests with
  `testify`; components use the `jet.Suite` testify suite.
- Tests that need real infrastructure (PostgreSQL, Kafka, Redis, …) go behind
  the `integration` build tag:

  ```go
  //go:build integration
  ```

  Run them with `go test -tags integration ./...`.
- Add runnable [examples](https://pkg.go.dev/testing#hdr-Examples)
  (`func ExampleXxx`) for public APIs where they help — they appear on
  pkg.go.dev and run as tests.

## Documentation

Every package has a doc comment (`doc.go`), and exported identifiers are
documented following the [Go doc conventions](https://go.dev/doc/comment).
Update the docs and examples whenever you change a public API.

## Pull requests

1. Branch off `master` (e.g. `feat/...`, `fix/...`).
2. Ensure `go build ./...`, `go vet ./...` and `go test ./...` pass.
3. Open a PR describing the change and its motivation.
