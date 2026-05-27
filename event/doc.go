// Package event provides an in-process publish/subscribe event bus.
//
// Handlers can be registered synchronously or asynchronously, as one-shot
// (once) subscriptions, and in a transactional mode that serializes async
// delivery. It is meant for decoupling components within a single process, not
// for cross-service messaging (use the kafka package for that).
package event
