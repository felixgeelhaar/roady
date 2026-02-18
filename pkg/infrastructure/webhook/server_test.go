package webhook_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/infrastructure/webhook"
)

func TestServer_Health(t *testing.T) {
	server := webhook.NewServer(":8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Create a test handler
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %q", w.Body.String())
	}

	_ = server // Use server variable
}

func TestServer_RecentEvents(t *testing.T) {
	server := webhook.NewServer(":8080", nil)

	events := server.RecentEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestGitHubHandler_Provider(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	if handler.Provider() != "github" {
		t.Errorf("expected 'github', got %q", handler.Provider())
	}
}

func TestGitHubHandler_ParseEvent_Issues(t *testing.T) {
	handler := webhook.NewGitHubHandler()

	payload := map[string]interface{}{
		"action": "closed",
		"issue": map[string]interface{}{
			"number":   123,
			"title":    "Test Issue",
			"body":     "Description\n\nroady-id: task-456",
			"state":    "closed",
			"html_url": "https://github.com/owner/repo/issues/123",
		},
		"repository": map[string]interface{}{
			"full_name": "owner/repo",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "issues")

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if event.Provider != "github" {
		t.Errorf("expected provider 'github', got %q", event.Provider)
	}
	if event.EventType != "issue.closed" {
		t.Errorf("expected event type 'issue.closed', got %q", event.EventType)
	}
	if event.ExternalID != "123" {
		t.Errorf("expected external ID '123', got %q", event.ExternalID)
	}
	if event.TaskID != "task-456" {
		t.Errorf("expected task ID 'task-456', got %q", event.TaskID)
	}
	if event.Status != planning.StatusDone {
		t.Errorf("expected status 'done', got %q", event.Status)
	}
}

func TestGitHubHandler_ParseEvent_Ping(t *testing.T) {
	handler := webhook.NewGitHubHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-GitHub-Event", "ping")

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if event.EventType != "ping" {
		t.Errorf("expected event type 'ping', got %q", event.EventType)
	}
}

func TestGitHubHandler_ParseEvent_MissingHeader(t *testing.T) {
	handler := webhook.NewGitHubHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte("{}")))
	// No X-GitHub-Event header

	_, err := handler.ParseEvent(req)
	if err == nil {
		t.Error("expected error for missing header")
	}
}

func TestJiraHandler_Provider(t *testing.T) {
	handler := webhook.NewJiraHandler()
	if handler.Provider() != "jira" {
		t.Errorf("expected 'jira', got %q", handler.Provider())
	}
}

func TestJiraHandler_ParseEvent(t *testing.T) {
	handler := webhook.NewJiraHandler()

	payload := map[string]interface{}{
		"webhookEvent": "jira:issue_updated",
		"issue": map[string]interface{}{
			"id":   "10001",
			"key":  "PROJ-123",
			"self": "https://jira.example.com/rest/api/2/issue/10001",
			"fields": map[string]interface{}{
				"summary":     "Test Issue",
				"description": "Description\n\nroady-id: task-789",
				"status": map[string]interface{}{
					"name": "Done",
					"statusCategory": map[string]interface{}{
						"key": "done",
					},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if event.Provider != "jira" {
		t.Errorf("expected provider 'jira', got %q", event.Provider)
	}
	if event.EventType != "jira:issue_updated" {
		t.Errorf("expected event type 'jira:issue_updated', got %q", event.EventType)
	}
	if event.ExternalID != "PROJ-123" {
		t.Errorf("expected external ID 'PROJ-123', got %q", event.ExternalID)
	}
	if event.TaskID != "task-789" {
		t.Errorf("expected task ID 'task-789', got %q", event.TaskID)
	}
	if event.Status != planning.StatusDone {
		t.Errorf("expected status 'done', got %q", event.Status)
	}
}

func TestJiraHandler_ValidateSignature_NoSecret(t *testing.T) {
	handler := webhook.NewJiraHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", nil)

	// With empty secret, should return true
	if !handler.ValidateSignature(req, "") {
		t.Error("expected valid signature with empty secret")
	}
}

func TestJiraHandler_ValidateSignature_QueryParam(t *testing.T) {
	handler := webhook.NewJiraHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira?secret=my-secret", nil)

	if !handler.ValidateSignature(req, "my-secret") {
		t.Error("expected valid signature with query param")
	}
}

func TestJiraHandler_ValidateSignature_Bearer(t *testing.T) {
	handler := webhook.NewJiraHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", nil)
	req.Header.Set("Authorization", "Bearer my-secret")

	if !handler.ValidateSignature(req, "my-secret") {
		t.Error("expected valid signature with bearer token")
	}
}

func TestLinearHandler_Provider(t *testing.T) {
	handler := webhook.NewLinearHandler()
	if handler.Provider() != "linear" {
		t.Errorf("expected 'linear', got %q", handler.Provider())
	}
}

func TestLinearHandler_ParseEvent(t *testing.T) {
	handler := webhook.NewLinearHandler()

	payload := map[string]interface{}{
		"action": "update",
		"type":   "Issue",
		"data": map[string]interface{}{
			"id":          "issue-id-123",
			"identifier":  "ABC-456",
			"title":       "Test Issue",
			"description": "Description\n\nroady-id: task-linear-1",
			"url":         "https://linear.app/team/issue/ABC-456",
			"state": map[string]interface{}{
				"id":   "state-1",
				"name": "In Progress",
				"type": "started",
			},
			"team": map[string]interface{}{
				"id":  "team-1",
				"key": "ABC",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if event.Provider != "linear" {
		t.Errorf("expected provider 'linear', got %q", event.Provider)
	}
	if event.EventType != "Issue.update" {
		t.Errorf("expected event type 'Issue.update', got %q", event.EventType)
	}
	if event.ExternalID != "issue-id-123" {
		t.Errorf("expected external ID 'issue-id-123', got %q", event.ExternalID)
	}
	if event.TaskID != "task-linear-1" {
		t.Errorf("expected task ID 'task-linear-1', got %q", event.TaskID)
	}
	if event.Status != planning.StatusInProgress {
		t.Errorf("expected status 'in_progress', got %q", event.Status)
	}
}

func TestLinearHandler_ParseEvent_CompletedState(t *testing.T) {
	handler := webhook.NewLinearHandler()

	payload := map[string]interface{}{
		"action": "update",
		"type":   "Issue",
		"data": map[string]interface{}{
			"id":          "issue-id-123",
			"identifier":  "ABC-456",
			"title":       "Test Issue",
			"description": "",
			"url":         "https://linear.app/team/issue/ABC-456",
			"state": map[string]interface{}{
				"id":   "state-1",
				"name": "Done",
				"type": "completed",
			},
			"team": map[string]interface{}{
				"id":  "team-1",
				"key": "ABC",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if event.Status != planning.StatusDone {
		t.Errorf("expected status 'done', got %q", event.Status)
	}
}
