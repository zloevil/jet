package jet

const (
	ErrCodeWatchdogWorkerStuck = "WDG-001"
)

var ErrWatchdogWorkerStuck = func(workerName string) error {
	return NewAppErrBuilder(ErrCodeWatchdogWorkerStuck, "worker is stuck").
		F(KV{"worker": workerName}).
		Business().
		Err()
}
