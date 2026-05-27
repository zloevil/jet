package jet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// WatchdogSuite is the test suite for the watchdog module.
type WatchdogSuite struct {
	Suite
}

// SetupSuite initializes the test suite.
func (s *WatchdogSuite) SetupSuite() {
	s.Suite.Init(nil)
}

// TestNewWatchdog verifies creation of a new Watchdog.
func (s *WatchdogSuite) TestNewWatchdog() {
	wd := NewWatchdog(time.Second * 5)

	s.NotNil(wd)
	s.NotNil(wd.workers)
	s.Equal(time.Second*5, wd.timeout)
	s.False(wd.disabled)
}

// TestRegisterAndPing verifies worker registration and ping.
func (s *WatchdogSuite) TestRegisterAndPing() {
	wd := NewWatchdog(time.Second * 5)

	wd.Register("worker-1")
	wd.Register("worker-2")

	workers := wd.Workers()
	s.Len(workers, 2)
	s.Contains(workers, "worker-1")
	s.Contains(workers, "worker-2")

	// Ping should update the timestamp
	time.Sleep(10 * time.Millisecond)
	wd.Ping("worker-1")

	workers = wd.Workers()
	s.True(workers["worker-1"].After(workers["worker-2"]))
}

// TestUnregister verifies removing a worker from tracking.
func (s *WatchdogSuite) TestUnregister() {
	wd := NewWatchdog(time.Second * 5)

	wd.Register("worker-1")
	wd.Register("worker-2")

	wd.Unregister("worker-1")

	workers := wd.Workers()
	s.Len(workers, 1)
	s.NotContains(workers, "worker-1")
	s.Contains(workers, "worker-2")
}

// TestCheckHealthy verifies Check passes for healthy workers.
func (s *WatchdogSuite) TestCheckHealthy() {
	wd := NewWatchdog(time.Second * 5)

	wd.Register("worker-1")
	wd.Register("worker-2")

	err := wd.Check()
	s.NoError(err)
}

// TestCheckStuck verifies Check fails for a stuck worker.
func (s *WatchdogSuite) TestCheckStuck() {
	wd := NewWatchdog(50 * time.Millisecond)

	wd.Register("worker-1")

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	err := wd.Check()
	s.Error(err)

	// verify the error is an AppError with the correct code
	appErr, ok := IsAppErr(err)
	s.True(ok)
	s.Equal(ErrCodeWatchdogWorkerStuck, appErr.Code())
	s.Equal("worker-1", appErr.Fields()["worker"])
}

// TestCheckPartiallyStuck verifies Check fails if at least one worker is stuck.
func (s *WatchdogSuite) TestCheckPartiallyStuck() {
	wd := NewWatchdog(50 * time.Millisecond)

	wd.Register("worker-1")
	wd.Register("worker-2")

	// Wait and ping only one worker
	time.Sleep(100 * time.Millisecond)
	wd.Ping("worker-1")

	err := wd.Check()
	s.Error(err)

	// verify the error carries information about the stuck worker
	appErr, ok := IsAppErr(err)
	s.True(ok)
	s.Equal(ErrCodeWatchdogWorkerStuck, appErr.Code())
	s.Equal("worker-2", appErr.Fields()["worker"])
}

// TestDisable verifies that Disable turns off the check.
func (s *WatchdogSuite) TestDisable() {
	wd := NewWatchdog(50 * time.Millisecond)

	wd.Register("worker-1")

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should fail before disable
	err := wd.Check()
	s.Error(err)

	// Disable watchdog
	wd.Disable()

	// Should pass after disable
	err = wd.Check()
	s.NoError(err)
}

// TestEnable verifies that Enable turns the check back on.
func (s *WatchdogSuite) TestEnable() {
	wd := NewWatchdog(50 * time.Millisecond)

	wd.Register("worker-1")
	wd.Disable()

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should pass while disabled
	err := wd.Check()
	s.NoError(err)

	// Enable watchdog
	wd.Enable()

	// Should fail after enable
	err = wd.Check()
	s.Error(err)
}

// TestEmptyWorkers verifies Check passes with no registered workers.
func (s *WatchdogSuite) TestEmptyWorkers() {
	wd := NewWatchdog(time.Second * 5)

	// No workers registered - should pass
	err := wd.Check()
	s.NoError(err)
}

// TestWorkersReturnsSnapshot verifies Workers returns a copy of the data.
func (s *WatchdogSuite) TestWorkersReturnsSnapshot() {
	wd := NewWatchdog(time.Second * 5)

	wd.Register("worker-1")

	workers1 := wd.Workers()
	workers1["worker-1"] = time.Time{} // Modify returned map

	workers2 := wd.Workers()
	s.NotEqual(workers1["worker-1"], workers2["worker-1"]) // Original should be unchanged
}

func TestWatchdogSuite(t *testing.T) {
	suite.Run(t, new(WatchdogSuite))
}
