package jet

import (
	"context"
	stdErr "errors"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

const (
	ErrTypeBusiness = "business"
	ErrTypeSystem   = "system"
	ErrTypePanic    = "panic"

	ErrCodePanic = "PANIC-001"
)

var (
	ErrPanic = func(ctx context.Context, cause interface{}) error {
		var causeErr error
		switch v := cause.(type) {
		case nil:
			causeErr = fmt.Errorf("panic")
		case string:
			causeErr = fmt.Errorf("%s", v)
		case error:
			causeErr = v
		default:
			causeErr = fmt.Errorf("%v", v)
		}
		return NewAppErrBuilder(ErrCodePanic, "").Wrap(causeErr).C(ctx).Panic().Err()
	}
)

// AppError specifies application error object
type AppError struct {
	error
	grpcStatus *uint32
	httpStatus *uint32
	code       string
	fields     KV
	errType    string
	cause      error
}

// AppErrBuilder allows building AppError object
type AppErrBuilder interface {
	// C attaches a request context to AppError
	C(ctx context.Context) AppErrBuilder
	// F attaches additional fields to AppError object
	// if type of passed field isn't valid, it's just silently ignored
	F(fields KV) AppErrBuilder
	// GrpcSt attaches gRPC status
	// when converting to grpc error it will be checked and if populated, corresponding grpc status is set
	GrpcSt(status uint32) AppErrBuilder
	// HttpSt attaches HTTP status
	// it gives some hint to API gateway layer what HTTP status to return client
	HttpSt(status uint32) AppErrBuilder
	// Business marks error as business validation error
	Business() AppErrBuilder
	// System marks error as system error (this is by default)
	System() AppErrBuilder
	// Panic marks error as panic error
	Panic() AppErrBuilder
	// Type sets type of error
	Type(t string) AppErrBuilder
	// Err builds error with all specified attributes
	Err() error
	// Wrap wraps another error
	Wrap(cause error) AppErrBuilder
}

// appErrBuildImpl implements AppErrBuilder
type appErrBuildImpl struct {
	appErr *AppError
	format string
	args   []interface{}
}

func (b *appErrBuildImpl) C(ctx context.Context) AppErrBuilder {
	if rCtx, ok := Request(ctx); ok {
		for k, v := range rCtx.ToMap() {
			if !IsEmpty(v) {
				b.appErr.fields[k] = v
			}
		}
	}
	return b
}

func (b *appErrBuildImpl) F(fields KV) AppErrBuilder {
	for k, v := range fields {
		if !IsEmpty(v) {
			b.appErr.fields[k] = v
		}
	}
	return b
}

func (b *appErrBuildImpl) GrpcSt(status uint32) AppErrBuilder {
	b.appErr.grpcStatus = &status
	return b
}

func (b *appErrBuildImpl) HttpSt(status uint32) AppErrBuilder {
	b.appErr.httpStatus = &status
	return b
}

func (b *appErrBuildImpl) Business() AppErrBuilder {
	b.appErr.errType = ErrTypeBusiness
	return b
}

func (b *appErrBuildImpl) System() AppErrBuilder {
	b.appErr.errType = ErrTypeSystem
	return b
}

func (b *appErrBuildImpl) Panic() AppErrBuilder {
	b.appErr.errType = ErrTypePanic
	return b
}

func (b *appErrBuildImpl) Type(t string) AppErrBuilder {
	b.appErr.errType = t
	return b
}

func (b *appErrBuildImpl) Err() error {
	if b.appErr.httpStatus == nil {
		// set default http status depending on error type
		if b.appErr.errType == ErrTypeBusiness {
			b.appErr.httpStatus = UInt32Ptr(http.StatusBadRequest)
		} else {
			b.appErr.httpStatus = UInt32Ptr(http.StatusInternalServerError)
		}
	}
	return b.appErr
}

func (b *appErrBuildImpl) Wrap(cause error) AppErrBuilder {
	if appErr, ok := IsAppErr(cause); ok {
		b.F(appErr.fields)
	}
	b.appErr.error = errors.Wrapf(cause, b.format, b.args...)
	b.appErr.cause = cause
	return b
}

// NewAppErrBuilder creates a new AppErrBuilder and default AppError object
func NewAppErrBuilder(code string, format string, args ...interface{}) AppErrBuilder {
	b := &appErrBuildImpl{
		args:   args,
		format: format,
		appErr: newAppErr(code, format, ErrTypeSystem, args...),
	}
	return b
}

// newAppErr creates a new AppError
func newAppErr(code string, format string, errType string, args ...interface{}) *AppError {
	return &AppError{
		error:   errors.Errorf(format, args...),
		code:    code,
		fields:  make(KV),
		errType: errType,
	}
}

// NewAppError creates a new AppError and returns error interface
func NewAppError(code string, format string, args ...interface{}) error {
	return newAppErr(code, format, ErrTypeSystem, args...)
}

// Error returns default error message
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.code, e.error)
}

// WithStack return error message with stack trace attached
// if you need split fields, assert to *AppError
func (e *AppError) WithStack() string {
	return fmt.Sprintf("%s: %+v", e.code, e.error)
}

func (e *AppError) WithStackErr() error {
	return &withStackAppErr{AppError: e}
}

// Code returns error code
func (e *AppError) Code() string {
	return e.code
}

// Message returns error message
func (e *AppError) Message() string {
	if e.error != nil {
		return e.error.Error()
	} else {
		return ""
	}
}

// Fields returns fields
func (e *AppError) Fields() KV {
	return e.fields
}

// GrpcStatus returns grpc status
func (e *AppError) GrpcStatus() *uint32 {
	return e.grpcStatus
}

// HttpStatus returns http status
func (e *AppError) HttpStatus() *uint32 {
	return e.httpStatus
}

// Type returns error type
func (e *AppError) Type() string {
	return e.errType
}

// Cause returns wrapped error
func (e *AppError) Cause() error {
	return e.cause
}

// IsAppErr checks if error interface is asserted to *AppError
// if true, it returns *AppError
func IsAppErr(e error) (*AppError, bool) {
	var appErr *AppError
	ok := stdErr.As(e, &appErr)
	return appErr, ok
}

// IsAppErrCode checks if error interface is asserted to *AppError and contains specific code
func IsAppErrCode(e error, code string) bool {
	err, ok := IsAppErr(e)
	if !ok {
		return false
	}
	return err.code == code
}

type withStackAppErr struct {
	*AppError
}

func (s *withStackAppErr) Error() string {
	return s.AppError.WithStack()
}
