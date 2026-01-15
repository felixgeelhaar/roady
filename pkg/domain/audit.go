package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
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
	// Deterministic sequence: PrevHash + ID + Timestamp + Action + Actor + Metadata
	h.Write([]byte(e.PrevHash))
	h.Write([]byte(e.ID))
	h.Write([]byte(e.Timestamp.Format(time.RFC3339Nano)))
	h.Write([]byte(e.Action))
	h.Write([]byte(e.Actor))
	h.Write([]byte(canonicalJSON(e.Metadata)))
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSON produces a deterministic JSON representation of metadata.
// Keys are sorted alphabetically to ensure consistent hashing.
func canonicalJSON(m map[string]interface{}) string {
	if len(m) == 0 {
		return ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build ordered map representation
	ordered := make([]byte, 0, 256)
	ordered = append(ordered, '{')
	for i, k := range keys {
		if i > 0 {
			ordered = append(ordered, ',')
		}
		keyJSON, _ := json.Marshal(k)
		valJSON, _ := json.Marshal(m[k])
		ordered = append(ordered, keyJSON...)
		ordered = append(ordered, ':')
		ordered = append(ordered, valJSON...)
	}
	ordered = append(ordered, '}')

	return string(ordered)
}

// UsageStats tracks the "cost" and telemetry of operations.
type UsageStats struct {
	TotalCommands int            `json:"total_commands"`
	LastCommandAt time.Time      `json:"last_command_at"`
	ProviderStats map[string]int `json:"provider_stats"` // e.g., "gemini-tokens": 500
}
