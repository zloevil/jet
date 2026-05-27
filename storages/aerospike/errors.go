package aerospike

import (
	"context"
	"github.com/zloevil/jet"
)

var (
	ErrCodeAeroConn           = "AERO-001"
	ErrCodeAeroClosed         = "AERO-002"
	ErrCodeAeroNewKey         = "AERO-003"
	ErrCodeAeroInvalidBinType = "AERO-004"
)

var (
	ErrAeroConn = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeAeroConn, "").Wrap(cause).C(ctx).Err()
	}
	ErrAeroNewKey = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeAeroNewKey, "").Wrap(cause).C(ctx).Err()
	}
	ErrAeroClosed = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeAeroClosed, "dealing with closed instance").C(ctx).Err()
	}
	ErrAeroInvalidBinType = func(ctx context.Context, bin string) error {
		return jet.NewAppErrBuilder(ErrCodeAeroInvalidBinType, "invalid bin type").F(jet.KV{"bin": bin}).C(ctx).Err()
	}
)
