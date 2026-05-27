// Package goroutine provides panic-safe goroutines and error groups.
//
// Goroutines started here recover from panics, log them through the provided
// logger and can be retried. Pass a cloned logger (CLogger is not safe for
// concurrent use) when starting work in the background.
package goroutine
