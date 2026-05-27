package profile

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zloevil/jet"
)

func testLogger() jet.CLoggerFunc {
	l := jet.InitLogger(&jet.LogConfig{Level: jet.ErrorLevel})
	return func() jet.CLogger { return jet.L(l) }
}

func Test_New(t *testing.T) {
	assert.NotNil(t, New(testLogger()))
}

func Test_Init(t *testing.T) {
	srv := New(testLogger()).(*profileServerImpl)
	cfg := &Config{Enabled: true, Port: "16061"}

	require.NoError(t, srv.Init(cfg))
	assert.Equal(t, cfg, srv.config)
}

func Test_Listen_ServesPprof(t *testing.T) {
	srv := New(testLogger())
	require.NoError(t, srv.Init(&Config{Enabled: true, Port: "16062"}))

	srv.Listen()
	defer srv.Close()

	// give the async server time to start
	time.Sleep(150 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/debug/pprof/", "16062"))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func Test_Close_NoPanic(t *testing.T) {
	srv := New(testLogger())
	require.NoError(t, srv.Init(&Config{Port: "16063"}))
	assert.NotPanics(t, func() { srv.Close() })
}

func Test_ErrProfileHttpError(t *testing.T) {
	err := ErrProfileHttpError(errors.New("boom"))
	appErr, ok := jet.IsAppErr(err)
	require.True(t, ok)
	assert.Equal(t, ErrCodeProfileHttpError, appErr.Code())
}
