package domain

// AuditRepository handles persistence of events and usage statistics.
type AuditRepository interface {
	RecordEvent(event Event) error
	LoadEvents() ([]Event, error)
	UpdateUsage(stats UsageStats) error
	LoadUsage() (*UsageStats, error)
}
