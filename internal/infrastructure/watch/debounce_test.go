package watch

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncer_CoalescesRapidTriggers(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(50*time.Millisecond, func() {
		count.Add(1)
	})
	defer d.Stop()

	for i := 0; i < 10; i++ {
		d.Trigger()
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce window to expire
	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 callback invocation, got %d", got)
	}
}

func TestDebouncer_Stop(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(50*time.Millisecond, func() {
		count.Add(1)
	})

	d.Trigger()
	d.Stop()

	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 0 {
		t.Errorf("expected 0 callback invocations after stop, got %d", got)
	}
}
