package jet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func Test_RequestContext_Getters(t *testing.T) {
	r := NewRequestCtx().
		WithRequestId("rid").
		WithSessionId("sid").
		WithUser("uid", "un").
		WithApp("app").
		WithClientIp("1.2.3.4").
		WithLang(language.English).
		WithRoles("admin", "user").
		WithKv("k", "v")

	assert.Equal(t, "rid", r.GetRequestId())
	assert.Equal(t, "sid", r.GetSessionId())
	assert.Equal(t, "uid", r.GetUserId())
	assert.Equal(t, "un", r.GetUsername())
	assert.Equal(t, "app", r.GetApp())
	assert.Equal(t, "1.2.3.4", r.GetClientIp())
	assert.Equal(t, language.English, r.GetLang())
	assert.Equal(t, []string{"admin", "user"}, r.GetRoles())
	assert.Equal(t, "v", r.GetKv()["k"])
}

func Test_RequestContext_Empty(t *testing.T) {
	r := NewRequestCtx().WithRequestId("x")
	assert.Empty(t, r.Empty().GetRequestId())
}

func Test_Request_And_MustRequest(t *testing.T) {
	ctx := NewRequestCtx().WithRequestId("rid").ToContext(context.Background())

	got, ok := Request(ctx)
	assert.True(t, ok)
	assert.Equal(t, "rid", got.GetRequestId())

	mr, err := MustRequest(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "rid", mr.GetRequestId())

	_, err = MustRequest(context.Background())
	assert.Error(t, err, "no request context -> error")
}

func Test_GrpcMD_RoundTrip(t *testing.T) {
	ctx := NewRequestCtx().WithRequestId("rid").WithUser("uid", "un").ToContext(context.Background())

	md, ok := ContextToGrpcMD(ctx)
	require.True(t, ok)

	restored := FromGrpcMD(context.Background(), md)
	rc, ok := Request(restored)
	require.True(t, ok)
	assert.Equal(t, "rid", rc.GetRequestId())
	assert.Equal(t, "uid", rc.GetUserId())
}

func Test_FromMap(t *testing.T) {
	src := NewRequestCtx().WithRequestId("rid").WithUser("uid", "un")

	ctx, err := FromMap(context.Background(), src.ToMap())
	require.NoError(t, err)

	rc, ok := Request(ctx)
	require.True(t, ok)
	assert.Equal(t, "rid", rc.GetRequestId())
	assert.Equal(t, "uid", rc.GetUserId())
}

func Test_Copy(t *testing.T) {
	ctx := NewRequestCtx().WithRequestId("rid").WithUser("uid", "un").ToContext(context.Background())

	rc, ok := Request(Copy(ctx))
	require.True(t, ok)
	assert.Equal(t, "rid", rc.GetRequestId())
	assert.Equal(t, "uid", rc.GetUserId())
}
