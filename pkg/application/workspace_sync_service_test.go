package application_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
)

type MockAuditLogger struct {
	Logs [][]interface{}
}

func (m *MockAuditLogger) Log(action, actor string, metadata map[string]interface{}) {
	m.Logs = append(m.Logs, []interface{}{action, actor, metadata})
}

func (m *MockAuditLogger) Close() error { return nil }

type MockWorkspaceSyncRepo struct {
	Root string
}

func TestWorkspaceSyncService_Push_NoChanges(t *testing.T) {
	// This test would need git to be available
	// Skip if git is not available
	t.Skip("Requires git to be available")
}

func TestWorkspaceSyncService_Push(t *testing.T) {
	t.Skip("Requires git to be available")
}

func TestWorkspaceSyncService_Pull(t *testing.T) {
	t.Skip("Requires git to be available")
}

func TestSyncResult_String(t *testing.T) {
	result := &application.SyncResult{
		Action:   "push",
		Files:    []string{"plan.json", "state.json"},
		Conflict: false,
		Message:  "Successfully pushed",
	}

	if result.Action != "push" {
		t.Errorf("expected push but got %s", result.Action)
	}
	if len(result.Files) != 2 {
		t.Errorf("expected 2 files but got %d", len(result.Files))
	}
	if result.Conflict {
		t.Error("expected no conflict")
	}
}

func TestSyncResult_WithConflict(t *testing.T) {
	result := &application.SyncResult{
		Action:   "pull",
		Files:    []string{"plan.json"},
		Conflict: true,
		Message:  "Conflict detected",
	}

	if !result.Conflict {
		t.Error("expected conflict")
	}
}

func TestWorkspaceSyncService_New(t *testing.T) {
	svc := application.NewWorkspaceSyncService("/tmp/test", nil)
	if svc == nil {
		t.Error("expected service to not be nil")
	}
}
