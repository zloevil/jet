// Package jet is a pragmatic toolkit for building Go microservices.
//
// It provides the building blocks most services re-implement from scratch:
//
//   - CLogger — a structured, context-aware logger on top of log/slog.
//   - ConfigLoader — a typed YAML + environment-variable configuration loader.
//   - RequestContext — a request context (request id, user, session, roles, …)
//     that propagates across HTTP, gRPC and Kafka boundaries.
//   - AppError — a structured error model with codes, business/system/panic
//     types and HTTP/gRPC status hints.
//   - Healthcheck, JWT, crypto, validation, watchdog and generics helpers.
//
// Sub-packages add the rest of a typical service: storage adapters
// (storages/pg, storages/redis, …), messaging (kafka, event), transport
// servers (http, grpc), service lifecycle (cluster), concurrency helpers
// (goroutine, retry) and observability (monitoring, profile).
//
// The toolkit favors boring, explicit building blocks over magic, and is
// extracted from an internal kit that has run 20+ production services.
package jet
