// Package watch provides filesystem watching with debounce support.
package watch

import (
	"sync"
	"time"
)

// Debouncer coalesces rapid events into a single callback invocation.
type Debouncer struct {
	window   time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	callback func()
}

// NewDebouncer creates a debouncer with the given window duration.
func NewDebouncer(window time.Duration, callback func()) *Debouncer {
	return &Debouncer{
		window:   window,
		callback: callback,
	}
}

// Trigger resets the debounce timer. The callback fires after the window
// elapses with no further triggers.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.window, d.callback)
}

// Stop cancels any pending callback.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
}
