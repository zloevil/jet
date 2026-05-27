package jet

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// HealthcheckSuite is the test suite for the healthcheck module.
type HealthcheckSuite struct {
	Suite
	hc   *Healthcheck
	port string
}

// SetupSuite initializes the test suite.
func (s *HealthcheckSuite) SetupSuite() {
	s.Suite.Init(nil)
	s.port = "19876" // use a non-standard port for tests
}

// SetupTest creates a new healthcheck before each test.
func (s *HealthcheckSuite) SetupTest() {
	s.hc = NewHealthCheck(&HealthcheckConfig{Port: s.port})
}

// TearDownTest stops the healthcheck after each test.
func (s *HealthcheckSuite) TearDownTest() {
	if s.hc != nil {
		s.hc.Stop()
	}
}

// TestNewHealthCheck verifies creation of a new Healthcheck.
func (s *HealthcheckSuite) TestNewHealthCheck() {
	hc := NewHealthCheck(&HealthcheckConfig{Port: "8081"})

	s.NotNil(hc)
	s.NotNil(hc.hcHandler)
	s.Equal("8081", hc.port)
	s.Nil(hc.srv)
}

// TestStartAndStop verifies starting and stopping the server.
func (s *HealthcheckSuite) TestStartAndStop() {
	s.hc.Start()

	// give the server time to start
	time.Sleep(100 * time.Millisecond)

	// verify the server responds
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// stop
	s.hc.Stop()

	// give the server time to stop
	time.Sleep(100 * time.Millisecond)

	// verify the server no longer responds
	_, err = http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.Error(err)
}

// TestLivenessCheckSuccess verifies a successful liveness check.
func (s *HealthcheckSuite) TestLivenessCheckSuccess() {
	s.hc.AddLivenessCheck("test-check", func() error {
		return nil
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestLivenessCheckFailure verifies a failing liveness check.
func (s *HealthcheckSuite) TestLivenessCheckFailure() {
	s.hc.AddLivenessCheck("failing-check", func() error {
		return errors.New("check failed")
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()
}

// TestReadinessCheckSuccess verifies a successful readiness check.
func (s *HealthcheckSuite) TestReadinessCheckSuccess() {
	s.hc.AddReadinessCheck("test-check", func() error {
		return nil
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/ready", s.port))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestReadinessCheckFailure verifies a failing readiness check.
func (s *HealthcheckSuite) TestReadinessCheckFailure() {
	s.hc.AddReadinessCheck("failing-check", func() error {
		return errors.New("not ready")
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/ready", s.port))
	s.NoError(err)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()
}

// TestMultipleLivenessChecks verifies multiple liveness checks.
func (s *HealthcheckSuite) TestMultipleLivenessChecks() {
	s.hc.AddLivenessCheck("check-1", func() error {
		return nil
	})
	s.hc.AddLivenessCheck("check-2", func() error {
		return nil
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestMultipleLivenessChecksOneFails verifies that one failing check makes the whole endpoint unhealthy.
func (s *HealthcheckSuite) TestMultipleLivenessChecksOneFails() {
	s.hc.AddLivenessCheck("check-ok", func() error {
		return nil
	})
	s.hc.AddLivenessCheck("check-fail", func() error {
		return errors.New("failed")
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()
}

// TestLivenessCheckIncludedInReadiness verifies that liveness checks are included in readiness.
func (s *HealthcheckSuite) TestLivenessCheckIncludedInReadiness() {
	s.hc.AddLivenessCheck("liveness-check", func() error {
		return errors.New("liveness failed")
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	// readiness must fail too, since liveness checks are included in readiness
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/ready", s.port))
	s.NoError(err)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()
}

// TestDynamicCheck verifies dynamic change of the check state.
func (s *HealthcheckSuite) TestDynamicCheck() {
	healthy := true

	s.hc.AddLivenessCheck("dynamic-check", func() error {
		if !healthy {
			return errors.New("unhealthy")
		}
		return nil
	})

	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	// healthy at first
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// change the state
	healthy = false

	// now unhealthy
	resp, err = http.Get(fmt.Sprintf("http://localhost:%s/live", s.port))
	s.NoError(err)
	s.Equal(http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()
}

// TestStopIdempotent verifies that Stop can be called multiple times without panicking.
func (s *HealthcheckSuite) TestStopIdempotent() {
	s.hc.Start()
	time.Sleep(100 * time.Millisecond)

	// call Stop several times
	s.NotPanics(func() {
		s.hc.Stop()
		s.hc.Stop()
		s.hc.Stop()
	})
}

// TestStopBeforeStart verifies that Stop before Start does not panic.
func (s *HealthcheckSuite) TestStopBeforeStart() {
	s.NotPanics(func() {
		s.hc.Stop()
	})
}

func TestHealthcheckSuite(t *testing.T) {
	suite.Run(t, new(HealthcheckSuite))
}
