package jet

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustAppErr(t *testing.T, err error) *AppError {
	t.Helper()
	ae, ok := IsAppErr(err)
	require.True(t, ok)
	return ae
}

func Test_Builder_Statuses(t *testing.T) {
	ae := mustAppErr(t, NewAppErrBuilder("C", "m").GrpcSt(5).HttpSt(404).Err())

	require.NotNil(t, ae.GrpcStatus())
	assert.EqualValues(t, 5, *ae.GrpcStatus())
	require.NotNil(t, ae.HttpStatus())
	assert.EqualValues(t, 404, *ae.HttpStatus())
}

func Test_Builder_Types(t *testing.T) {
	assert.Equal(t, ErrTypeBusiness, mustAppErr(t, NewAppErrBuilder("C", "m").Business().Err()).Type())
	assert.Equal(t, ErrTypeSystem, mustAppErr(t, NewAppErrBuilder("C", "m").System().Err()).Type())
	assert.Equal(t, ErrTypePanic, mustAppErr(t, NewAppErrBuilder("C", "m").Panic().Err()).Type())
	assert.Equal(t, "custom", mustAppErr(t, NewAppErrBuilder("C", "m").Type("custom").Err()).Type())
	assert.Equal(t, ErrTypeSystem, mustAppErr(t, NewAppErrBuilder("C", "m").Err()).Type(), "default type is system")
}

func Test_Builder_DefaultHttpStatus(t *testing.T) {
	biz := mustAppErr(t, NewAppErrBuilder("C", "m").Business().Err())
	require.NotNil(t, biz.HttpStatus())
	assert.EqualValues(t, http.StatusBadRequest, *biz.HttpStatus(), "business defaults to 400")

	sys := mustAppErr(t, NewAppErrBuilder("C", "m").System().Err())
	require.NotNil(t, sys.HttpStatus())
	assert.EqualValues(t, http.StatusInternalServerError, *sys.HttpStatus(), "system defaults to 500")
}

func Test_Cause(t *testing.T) {
	cause := errors.New("root cause")
	ae := mustAppErr(t, NewAppErrBuilder("C", "m").Wrap(cause).Err())
	assert.Equal(t, cause, ae.Cause())
}

func Test_IsAppErrCode(t *testing.T) {
	err := NewAppError("MY-1", "m")
	assert.True(t, IsAppErrCode(err, "MY-1"))
	assert.False(t, IsAppErrCode(err, "OTHER"), "wrong code")
	assert.False(t, IsAppErrCode(errors.New("plain"), "MY-1"), "non-AppError")
	assert.False(t, IsAppErrCode(nil, "MY-1"), "nil error")
}
