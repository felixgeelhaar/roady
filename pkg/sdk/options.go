package sdk

import "time"

type options struct {
	timeout      time.Duration
	maxAttempts  int
	initialDelay time.Duration
}

func defaultOptions() options {
	return options{
		timeout:      30 * time.Second,
		maxAttempts:  3,
		initialDelay: 500 * time.Millisecond,
	}
}

// Option configures the SDK client.
type Option func(*options)

// WithTimeout sets the per-call timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithRetry configures retry behaviour.
func WithRetry(maxAttempts int, initialDelay time.Duration) Option {
	return func(o *options) {
		o.maxAttempts = maxAttempts
		o.initialDelay = initialDelay
	}
}
