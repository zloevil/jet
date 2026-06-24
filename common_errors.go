package jet

import "context"

const (
	ErrCodeAdapterConfigInvalid = "JET-001"
)

var (
	ErrAdapterConfigInvalid = func(ctx context.Context) error {
		return NewAppErrBuilder(ErrCodeAdapterConfigInvalid, "invalid config").Err()
	}
)
