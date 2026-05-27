package monitoring

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zloevil/jet"
)

func testLogger() jet.CLoggerFunc {
	l := jet.InitLogger(&jet.LogConfig{Level: jet.ErrorLevel})
	return func() jet.CLogger { return jet.L(l) }
}

func Test_ErrorMonitoring_ClassifiesByType(t *testing.T) {
	m := NewErrorMonitoring().(*errorMonitoring)

	m.Error(jet.NewAppErrBuilder("B-1", "biz").Business().Err())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.businessErrorCounter.WithLabelValues("B-1")))

	m.Error(jet.NewAppErrBuilder("S-1", "sys").System().Err())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.systemErrorCounter.WithLabelValues("S-1")))

	m.Error(jet.NewAppErrBuilder("P-1", "panic").Panic().Err())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.panicCounter.WithLabelValues()))

	m.Error(errors.New("raw non-app error"))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.systemErrorCounter.WithLabelValues("unhandled")))
}

func Test_ErrorMonitoring_IncAccumulates(t *testing.T) {
	m := NewErrorMonitoring().(*errorMonitoring)
	m.BusinessErrorInc("X")
	m.BusinessErrorInc("X")
	assert.Equal(t, 2.0, testutil.ToFloat64(m.businessErrorCounter.WithLabelValues("X")))
}

func Test_ErrorMonitoring_GetCollector(t *testing.T) {
	assert.Len(t, NewErrorMonitoring().GetCollector()(), 3)
}

func Test_RegexpMonitoring_Disabled_ReturnsMock(t *testing.T) {
	m := NewRegexpMonitoring(&Config{Enabled: false}, "src")

	_, ok := m.(*regexpMonitoringMock)
	assert.True(t, ok, "disabled config must yield the no-op mock")
	assert.NotPanics(t, func() {
		m.AddRegexps(&Regexp{Code: "c", Regexp: "x"})
		m.Text("x")
	})
	assert.Empty(t, m.GetCollector()())
}

func Test_RegexpMonitoring_CountsMatches(t *testing.T) {
	m := NewRegexpMonitoring(&Config{Enabled: true}, "src").(*regexpMonitoring)
	m.AddRegexps(&Regexp{Code: "greeting", Regexp: "hello"})

	m.Text("well hello there")
	m.Text("no match here")

	assert.Equal(t, 1.0, testutil.ToFloat64(m.regexpMatchCounter.WithLabelValues("src", "greeting")))
}

func Test_MetricsServer_Init_InvalidPort(t *testing.T) {
	err := NewMetricsServer(testLogger()).Init(&Config{Port: "not-a-port"})
	appErr, ok := jet.IsAppErr(err)
	require.True(t, ok)
	assert.Equal(t, ErrCodePrometheusInvalidPort, appErr.Code())
}

func Test_MetricsServer_Init_Valid(t *testing.T) {
	err := NewMetricsServer(testLogger()).Init(&Config{Enabled: true, Port: "19011"}, NewErrorMonitoring())
	assert.NoError(t, err)
}

func Test_MetricsServer_ServesMetrics(t *testing.T) {
	srv := NewMetricsServer(testLogger())
	require.NoError(t, srv.Init(&Config{Enabled: true, Port: "19012"}, NewErrorMonitoring()))

	srv.Listen()
	defer srv.Close()

	// give the async server time to start
	time.Sleep(150 * time.Millisecond)

	resp, err := http.Get("http://localhost:19012/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
