// Package cluster wires a service's lifecycle.
//
// It loads configuration, builds the CLI (the run command plus optional
// database-migration commands) and drives a user-provided Bootstrap through
// Init/Start/Close with signal handling and ordered shutdown.
//
// A service implements Bootstrap and starts with:
//
//	cluster.New[Config]("my-service", &App{}).Execute()
package cluster
