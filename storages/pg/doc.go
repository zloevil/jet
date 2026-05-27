// Package pg provides a PostgreSQL connection via GORM.
//
// Open returns a *Storage wrapping a configured *gorm.DB. The package also
// offers helpers for JSONB conversion (ToJsonb/FromJsonb/MapToJsonb), null
// string handling and reusable query scopes (Paging, OrderBy*, WhereStrings,
// Merge, …).
package pg
