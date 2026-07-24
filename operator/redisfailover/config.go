package redisfailover

import "time"

// Config is the configuration for the redis operator.
type Config struct {
	ListenAddress            string
	MetricsPath              string
	Concurrency              int
	SupportedNamespacesRegex string
	// ResyncPeriod is how often every RedisFailover is re-reconciled even
	// without a change event. Zero falls back to the built-in default.
	ResyncPeriod time.Duration
}
