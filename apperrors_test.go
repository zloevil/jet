package jet

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// by default prints in format "code: message"
func Test_Error(t *testing.T) {
	e := NewAppError("ERR-123", "%s happened", "shit")
	fmt.Println(e)
}

// use IsAppErr function to assert to *AppError
func Test_WithStack(t *testing.T) {
	e := NewAppError("ERR-123", "%s happened", "shit")
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr.WithStack())
		return
	}
	t.Fatal()
}

// you can get error interface with stack
func Test_WithStackError(t *testing.T) {
	e := NewAppError("ERR-123", "%s happened", "shit")
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr.WithStackErr())
		return
	}
	t.Fatal()
}

// use IsAppErr function to split code and message
func Test_CodeAndMessageSplit(t *testing.T) {
	e := NewAppError("ERR-123", "%s happened", "shit")
	if appErr, ok := IsAppErr(e); ok {
		fmt.Printf("%s %s", appErr.Code(), appErr.Message())
		return
	}
	t.Fatal()
}

func Test_Wrap(t *testing.T) {
	originalErr := fmt.Errorf("original issue")
	e := NewAppErrBuilder("ERR-123", "%s happened", "shit").Wrap(originalErr).Err()
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr.WithStackErr())
		return
	}
	t.Fatal()
}

func Test_TwoWrappers(t *testing.T) {
	originalErr := fmt.Errorf("original issue")
	e := NewAppErrBuilder("ERR-123", "%s happened", "shit").Wrap(originalErr).Err()
	e2 := NewAppErrBuilder("ERR-124", "very bad %s happened", "shit").Wrap(e).Err()
	if appErr, ok := IsAppErr(e2); ok {
		fmt.Println(appErr.WithStackErr())
		return
	}
	t.Fatal()
}

func Test_NewWithBuilder_WhenContext(t *testing.T) {
	e := NewAppErrBuilder("ERR-123", "%s happens", "shit").
		F(KV{"val": "key"}).
		C(NewRequestCtx().WithNewRequestId().ToContext(context.Background())).
		Err()
	if appErr, ok := IsAppErr(e); ok {
		assert.NotEmpty(t, appErr.Fields()["_ctx.rid"])
		assert.NotEmpty(t, appErr.Fields()["val"])
		return
	}
	t.Fatal()
}

func Test_NewWithBuilder_WhenEmptyContext(t *testing.T) {
	e := NewAppErrBuilder("ERR-123", "%s happens", "shit").C(context.Background()).Err()
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr)
		return
	}
	t.Fatal()
}

func Test_NewWithBuilder_WhenFields(t *testing.T) {
	e := NewAppErrBuilder("ERR-123", "%s happens", "shit").
		F(KV{"f": "v"}).
		Err()
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr.WithStack())
		assert.NotEmpty(t, appErr.fields)
		assert.Equal(t, appErr.fields["f"], "v")
		return
	}
	t.Fatal()
}

func Test_NewWithBuilder_WrapWithFields(t *testing.T) {
	originalErr := NewAppErrBuilder("ERR-123", "%s happened", "shit").
		F(KV{"f": "v", "f2": "v2"}).
		Err()

	e := NewAppErrBuilder("ERR-124", "%s happens", "shit2").Wrap(originalErr).Err()
	if appErr, ok := IsAppErr(e); ok {
		fmt.Println(appErr.WithStack())
		assert.True(t, ok)
		assert.NotEmpty(t, appErr.fields)
		assert.Equal(t, appErr.fields["f"], "v")
		assert.Equal(t, appErr.fields["f2"], "v2")
		return
	}
	t.Fatal()
}
