// Package http provides an HTTP server built on gorilla/mux.
//
// The server adds configurable CORS, request/response tracing and graceful
// shutdown. BaseController offers helpers for parsing request variables, decoding
// JSON bodies, pagination and writing JSON or error responses (AppError is
// translated to the right HTTP status).
package http
