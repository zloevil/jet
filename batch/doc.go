// Package batch provides a generic worker that accumulates items and flushes
// them in batches.
//
// A batch is flushed when it reaches the configured size or when the flush
// interval elapses, whichever comes first. The caller supplies a Writer that
// persists each flushed batch (bulk insert, API call, …).
package batch
