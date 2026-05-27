package migration

import "github.com/zloevil/jet"

const (
	ErrCodeGooseFolderNotFound     = "GS-001"
	ErrCodeGooseFolderOpen         = "GS-002"
	ErrCodeGooseMigrationLock      = "GS-003"
	ErrCodeGooseMigrationUnLock    = "GS-004"
	ErrCodeGooseMigrationDown      = "GS-005"
	ErrCodeGooseMigrationUp        = "GS-006"
	ErrCodeGooseMigrationGetVer    = "GS-007"
	ErrCodeGooseUnsupportedDialect = "GS-008"
	ErCodeGoosePing                = "GS-009"
)

var (
	ErrGooseMigrationUp = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseMigrationUp, "").Wrap(cause).Err()
	}
	ErrGooseMigrationDown = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseMigrationDown, "").Wrap(cause).Err()
	}
	ErrGooseMigrationGetVer = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseMigrationGetVer, "").Wrap(cause).Err()
	}
	ErrGooseFolderNotFound = func(path string) error {
		return jet.NewAppErrBuilder(ErrCodeGooseFolderNotFound, "folder not found %s", path).Err()
	}
	ErrGooseFolderOpen = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseFolderOpen, "folder open").Wrap(cause).Err()
	}
	ErrGooseMigrationLock = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseMigrationLock, "locking before migration").Wrap(cause).Err()
	}
	ErrGooseMigrationUnLock = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeGooseMigrationUnLock, "unlocking after migration").Wrap(cause).Err()
	}
	ErrGooseUnsupportedDialect = func(dialect string) error {
		return jet.NewAppErrBuilder(ErrCodeGooseUnsupportedDialect, "unsupported dialect: %s", dialect).Err()
	}
	ErGoosePing = func(cause error) error {
		return jet.NewAppErrBuilder(ErCodeGoosePing, "ping").Wrap(cause).Err()
	}
)
