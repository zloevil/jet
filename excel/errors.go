package excel

import (
	"github.com/zloevil/jet"
)

const (
	ErrCodeExcelOpenFile  = "XLS-001"
	ErrCodeExcelOpenSheet = "XLS-002"
	ErrCodeExcelReadRows  = "XLS-003"
)

var (
	ErrExcelOpenFile = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeExcelOpenFile, "").Wrap(cause).Err()
	}
	ErrExcelOpenSheet = func() error {
		return jet.NewAppErrBuilder(ErrCodeExcelOpenSheet, "unable to find excel sheet to open").Err()
	}
	ErrExcelReadRows = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeExcelReadRows, "").Wrap(cause).Err()
	}
)
