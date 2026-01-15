package domain

// AuditLogger provides a simple interface for logging audit events.
// Services should depend on this interface rather than concrete implementations.
type AuditLogger interface {
	Log(action string, actor string, metadata map[string]interface{}) error
}
