// Package retry runs functions with bounded retries using exponential backoff
// with jitter.
//
// By default every error is retried; wrap an error with NonRetryable to stop
// immediately. Use DefaultConfig or RPCConfig for ready-made settings, or the
// DoWithLogger variants to log each attempt.
package retry
