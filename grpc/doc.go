// Package grpc provides a gRPC server and client preconfigured with middleware.
//
// Interceptors handle request-context propagation through metadata, optional
// JWT authentication, panic recovery and tracing. The server also registers
// the standard gRPC health service. AppError is mapped to the appropriate gRPC
// status.
package grpc
