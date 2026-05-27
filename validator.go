package jet

import "context"

const (
	ErrCodeValidatorNotEmpty = "VAL-001"
)

var (
	ErrValidatorNotEmpty = func(ctx context.Context, method, attr string) error {
		return NewAppErrBuilder(ErrCodeValidatorNotEmpty, "empty: %s", method, attr).F(KV{"method": method}).C(ctx).Business().Err()
	}
)

type Validator struct {
	err    error
	ctx    context.Context
	method string
}

func NewValidator(ctx context.Context) *Validator {
	return &Validator{
		ctx: ctx,
	}
}

func (v *Validator) Mth(m string) *Validator {
	v.method = m
	return v
}

func (v *Validator) NotEmptyString(attr string, val string) *Validator {
	if v.err != nil {
		return v
	}
	if val == "" {
		v.err = ErrValidatorNotEmpty(v.ctx, v.method, attr)
		return v
	}
	return v
}

func (v *Validator) E() error {
	return v.err
}
