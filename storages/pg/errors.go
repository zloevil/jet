package pg

import "github.com/zloevil/jet"

const (
	ErrCodePostgresOpen = "PG-001"
	ErrCodePgSetJsonb   = "PG-003"
	ErrCodePgGetJsonb   = "PG-004"
)

var (
	ErrPostgresOpen = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePostgresOpen, "").Wrap(cause).Err()
	}
	ErrPgSetJsonb = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePgSetJsonb, "set JSONB").Wrap(cause).Err()
	}
	ErrPgGetJsonb = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePgGetJsonb, "get JSONB").Wrap(cause).Err()
	}
)
