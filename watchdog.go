package jet

import (
	"sync"
	"time"
)

// Watchdog tracks worker activity.
// Each worker must call Ping() periodically to confirm it is alive.
// If a worker does not report for longer than timeout, Check() returns an error.
type Watchdog struct {
	mu       sync.RWMutex
	workers  map[string]time.Time
	timeout  time.Duration
	disabled bool
}

// NewWatchdog creates a new Watchdog with the given timeout.
// A worker that does not call Ping() within timeout is considered stuck.
func NewWatchdog(timeout time.Duration) *Watchdog {
	return &Watchdog{
		workers: make(map[string]time.Time),
		timeout: timeout,
	}
}

// Register starts tracking a worker.
// After registration the worker must call Ping() periodically.
func (w *Watchdog) Register(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.workers[name] = time.Now()
}

// Unregister stops tracking a worker.
// Used during a worker's graceful shutdown.
func (w *Watchdog) Unregister(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.workers, name)
}

// Ping updates the worker's last-activity time.
// A worker should call this on every iteration of its loop.
func (w *Watchdog) Ping(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.workers[name] = time.Now()
}

// Check inspects all registered workers.
// Returns an error if at least one worker is stuck (has not reported within timeout).
func (w *Watchdog) Check() error {
	if w.disabled {
		return nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	now := time.Now()
	for name, lastPing := range w.workers {
		if now.Sub(lastPing) > w.timeout {
			return ErrWatchdogWorkerStuck(name)
		}
	}
	return nil
}

// Disable turns off the watchdog check.
// Used during graceful shutdown so liveness does not fail while stopping.
func (w *Watchdog) Disable() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.disabled = true
}

// Enable turns the watchdog check back on.
func (w *Watchdog) Enable() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.disabled = false
}

// Workers returns the registered workers and their last ping time.
func (w *Watchdog) Workers() map[string]time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]time.Time, len(w.workers))
	for k, v := range w.workers {
		result[k] = v
	}
	return result
}
