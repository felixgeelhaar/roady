package webhook

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

func TestDeadLetterStore_AppendAndRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deadletters.jsonl")
	store := NewDeadLetterStore(path)

	dl := events.DeadLetter{
		Timestamp:   time.Now(),
		WebhookName: "test",
		URL:         "https://example.com/hook",
		EventType:   "task.started",
		Payload:     `{"event_type":"task.started"}`,
		Error:       "connection refused",
		Attempts:    3,
	}

	if err := store.Append(dl); err != nil {
		t.Fatal(err)
	}

	entries, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].WebhookName != "test" {
		t.Errorf("expected webhook name test, got %s", entries[0].WebhookName)
	}
}

func TestDeadLetterStore_ReadAll_MissingFile(t *testing.T) {
	store := NewDeadLetterStore(filepath.Join(t.TempDir(), "nonexistent.jsonl"))

	entries, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if entries != nil {
		t.Errorf("expected nil entries for missing file, got %v", entries)
	}
}
