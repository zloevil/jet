// Package migration runs SQL schema migrations via goose for PostgreSQL and
// ClickHouse.
//
// On PostgreSQL it uses advisory locks so only one instance applies migrations
// at a time.
package migration
