package wiring

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/events"
)

func TestNewWorkspaceProvidesRepoAndAudit(t *testing.T) {
	tempDir := t.TempDir()
	ws := NewWorkspace(tempDir)
	if ws.Repo == nil {
		t.Fatal("expected repository instance")
	}
	if ws.Audit == nil {
		t.Fatal("expected audit service instance")
	}
	if err := ws.Repo.Initialize(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}
	if !ws.Repo.IsInitialized() {
		t.Fatal("expected repository to be initialized")
	}
	if err := ws.Audit.Log("test.workspace", "tester", nil); err != nil {
		t.Fatalf("audit log failed: %v", err)
	}
}

func TestNewWorkspaceWithWebhook(t *testing.T) {
	tempDir := t.TempDir()
	ws := NewWorkspace(tempDir)
	if err := ws.Repo.Initialize(); err != nil {
		t.Fatalf("failed to initialize repo: %v", err)
	}

	// Save webhook config
	config := &events.WebhookConfig{
		Webhooks: []events.WebhookEndpoint{
			{Name: "test", URL: "https://example.com/webhook", Enabled: true},
		},
	}
	if err := ws.Repo.SaveWebhookConfig(config); err != nil {
		t.Fatalf("failed to save webhook config: %v", err)
	}

	// Create new workspace - should load webhook config
	ws2 := NewWorkspace(tempDir)
	if ws2.Notifier == nil {
		t.Fatal("expected notifier to be created when webhook config exists")
	}
}
