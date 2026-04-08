// Package watchdog implements a background process that monitors
// the gateway proxy for health and restarts it if necessary.
package watchdog

// Watchdog monitors the gateway process health.
type Watchdog struct {
	interval int // check interval in seconds
}

// New creates a new watchdog with the given check interval.
func New(intervalSeconds int) *Watchdog {
	return &Watchdog{interval: intervalSeconds}
}

// Start begins the watchdog monitoring loop.
func (w *Watchdog) Start() error {
	// TODO: implement health monitoring
	return nil
}

// Stop halts the watchdog.
func (w *Watchdog) Stop() {
	// TODO: implement graceful shutdown
}
