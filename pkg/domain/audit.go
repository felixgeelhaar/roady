package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Event represents a single auditable action in the system.
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Actor     string                 `json:"actor"` // "human" or "ai"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	PrevHash  string                 `json:"prev_hash,omitempty"` // Hash of the preceding event
	Hash      string                 `json:"hash,omitempty"`      // Deterministic hash of this event
}

// CalculateHash generates a deterministic SHA256 hash of the event data.
func (e *Event) CalculateHash() string {
	h := sha256.New()
	// Deterministic sequence: PrevHash + ID + Timestamp + Action + Actor
	h.Write([]byte(e.PrevHash))
	h.Write([]byte(e.ID))
	h.Write([]byte(e.Timestamp.Format(time.RFC3339Nano)))
	h.Write([]byte(e.Action))
	h.Write([]byte(e.Actor))
	// We skip Metadata for hashing simplicity in this prototype,
	// but a production-grade implementation would hash a canonical JSON of Metadata too.
	return hex.EncodeToString(h.Sum(nil))
}

// UsageStats tracks the "cost" and telemetry of operations.
type UsageStats struct {
	TotalCommands int            `json:"total_commands"`
	LastCommandAt time.Time      `json:"last_command_at"`
	ProviderStats map[string]int `json:"provider_stats"` // e.g., "gemini-tokens": 500
}
