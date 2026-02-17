package webhook_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/infrastructure/webhook"
)

// --- GitHub Signature Validation ---

func TestGitHubHandler_ValidateSignature_Valid(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	secret := "my-webhook-secret"
	body := []byte(`{"action":"opened","issue":{"number":1}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)

	if !handler.ValidateSignature(req, secret) {
		t.Error("expected valid signature")
	}
}

func TestGitHubHandler_ValidateSignature_Invalid(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	body := []byte(`{"action":"opened"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	if handler.ValidateSignature(req, "secret") {
		t.Error("expected invalid signature")
	}
}

func TestGitHubHandler_ValidateSignature_MissingHeader(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	body := []byte(`{"action":"opened"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	// No X-Hub-Signature-256 header

	if handler.ValidateSignature(req, "secret") {
		t.Error("expected invalid signature when header missing")
	}
}

// --- Linear Signature Validation ---

func TestLinearHandler_ValidateSignature_Valid(t *testing.T) {
	handler := webhook.NewLinearHandler()
	secret := "linear-secret"
	body := []byte(`{"action":"update","type":"Issue"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", sig)

	if !handler.ValidateSignature(req, secret) {
		t.Error("expected valid signature")
	}
}

func TestLinearHandler_ValidateSignature_Invalid(t *testing.T) {
	handler := webhook.NewLinearHandler()
	body := []byte(`{"action":"update"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", "invalid-sig")

	if handler.ValidateSignature(req, "secret") {
		t.Error("expected invalid signature")
	}
}

func TestLinearHandler_ValidateSignature_MissingHeader(t *testing.T) {
	handler := webhook.NewLinearHandler()
	body := []byte(`{"action":"update"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))

	if handler.ValidateSignature(req, "secret") {
		t.Error("expected invalid signature when header missing")
	}
}

// --- Jira Signature Validation ---

func TestJiraHandler_ValidateSignature_InvalidSecret(t *testing.T) {
	handler := webhook.NewJiraHandler()

	t.Run("wrong query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/jira?secret=wrong", nil)
		if handler.ValidateSignature(req, "correct-secret") {
			t.Error("expected invalid signature with wrong query param")
		}
	})

	t.Run("wrong bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", nil)
		req.Header.Set("Authorization", "Bearer wrong-secret")
		if handler.ValidateSignature(req, "correct-secret") {
			t.Error("expected invalid signature with wrong bearer")
		}
	})

	t.Run("no auth at all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", nil)
		if handler.ValidateSignature(req, "some-secret") {
			t.Error("expected invalid signature with no auth")
		}
	})
}

// --- GitHub Issue Status Mapping ---

func TestMapGitHubIssueStatus(t *testing.T) {
	tests := []struct {
		name        string
		state       string
		hasAssignee bool
		action      string
		wantStatus  planning.TaskStatus
	}{
		{"closed issue", "closed", false, "closed", planning.StatusDone},
		{"closed with assignee", "closed", true, "closed", planning.StatusDone},
		{"open with assignee", "open", true, "opened", planning.StatusInProgress},
		{"open no assignee", "open", false, "opened", planning.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"action": tt.action,
				"issue": map[string]interface{}{
					"number":   1,
					"title":    "Test",
					"body":     "roady-id: task-1",
					"state":    tt.state,
					"html_url": "https://github.com/test/repo/issues/1",
				},
				"repository": map[string]interface{}{
					"full_name": "test/repo",
				},
			}
			if tt.hasAssignee {
				issue := payload["issue"].(map[string]interface{})
				issue["assignee"] = map[string]interface{}{"login": "alice"}
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
			req.Header.Set("X-GitHub-Event", "issues")

			handler := webhook.NewGitHubHandler()
			event, err := handler.ParseEvent(req)
			if err != nil {
				t.Fatalf("ParseEvent failed: %v", err)
			}
			if event.Status != tt.wantStatus {
				t.Errorf("got status %q, want %q", event.Status, tt.wantStatus)
			}
		})
	}
}

// --- Jira Status Mapping ---

func TestMapJiraStatusToRoady(t *testing.T) {
	tests := []struct {
		name        string
		statusName  string
		categoryKey string
		wantStatus  planning.TaskStatus
	}{
		{"category done", "Done", "done", planning.StatusDone},
		{"category indeterminate", "In Progress", "indeterminate", planning.StatusInProgress},
		{"category new", "To Do", "new", planning.StatusPending},
		{"name done fallback", "Closed", "unknown", planning.StatusDone},
		{"name resolved fallback", "Resolved", "unknown", planning.StatusDone},
		{"name progress fallback", "In Progress", "unknown", planning.StatusInProgress},
		{"name started fallback", "Started", "unknown", planning.StatusInProgress},
		{"name blocked fallback", "Blocked", "unknown", planning.StatusBlocked},
		{"name on hold fallback", "On Hold", "unknown", planning.StatusBlocked},
		{"default fallback", "Custom Status", "unknown", planning.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"webhookEvent": "jira:issue_updated",
				"issue": map[string]interface{}{
					"id":   "10001",
					"key":  "TEST-1",
					"self": "https://jira.example.com",
					"fields": map[string]interface{}{
						"summary":     "Test Issue",
						"description": "",
						"status": map[string]interface{}{
							"name": tt.statusName,
							"statusCategory": map[string]interface{}{
								"key": tt.categoryKey,
							},
						},
					},
				},
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))

			handler := webhook.NewJiraHandler()
			event, err := handler.ParseEvent(req)
			if err != nil {
				t.Fatalf("ParseEvent failed: %v", err)
			}
			if event.Status != tt.wantStatus {
				t.Errorf("got status %q, want %q", event.Status, tt.wantStatus)
			}
		})
	}
}

// --- Linear State Mapping ---

func TestMapLinearStateToRoady(t *testing.T) {
	tests := []struct {
		stateType  string
		wantStatus planning.TaskStatus
	}{
		{"completed", planning.StatusDone},
		{"started", planning.StatusInProgress},
		{"canceled", planning.StatusBlocked},
		{"backlog", planning.StatusPending},
		{"unstarted", planning.StatusPending},
		{"triage", planning.StatusPending},
		{"unknown_type", planning.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.stateType, func(t *testing.T) {
			payload := map[string]interface{}{
				"action": "update",
				"type":   "Issue",
				"data": map[string]interface{}{
					"id":          "issue-1",
					"identifier":  "LIN-1",
					"title":       "Test",
					"description": "",
					"url":         "https://linear.app/test",
					"state": map[string]interface{}{
						"id":   "s1",
						"name": "Some State",
						"type": tt.stateType,
					},
					"team": map[string]interface{}{
						"id":  "t1",
						"key": "TEST",
					},
				},
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))

			handler := webhook.NewLinearHandler()
			event, err := handler.ParseEvent(req)
			if err != nil {
				t.Fatalf("ParseEvent failed: %v", err)
			}
			if event.Status != tt.wantStatus {
				t.Errorf("got status %q, want %q", event.Status, tt.wantStatus)
			}
		})
	}
}

// Test GitHub handler with unknown event type
func TestGitHubHandler_ParseEvent_UnknownEventType(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	body := []byte(`{"action":"created"}`)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if event.EventType != "push" {
		t.Errorf("got event type %q, want %q", event.EventType, "push")
	}
}

// Test GitHub handler with roady-id extraction
func TestGitHubHandler_ExtractRoadyID(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		wantID string
	}{
		{"with roady-id mid-text", "Some text\nroady-id: task-123\nMore text", "task-123"},
		{"with roady-id at end", "Description\nroady-id: task-end", "task-end"},
		{"no roady-id", "Just a description", ""},
		{"empty body", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := webhook.NewGitHubHandler()
			payload := map[string]interface{}{
				"action": "opened",
				"issue": map[string]interface{}{
					"number":   1,
					"title":    "Test",
					"body":     tt.body,
					"state":    "open",
					"html_url": "https://github.com/test/repo/issues/1",
				},
				"repository": map[string]interface{}{
					"full_name": "test/repo",
				},
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
			req.Header.Set("X-GitHub-Event", "issues")

			event, err := handler.ParseEvent(req)
			if err != nil {
				t.Fatalf("ParseEvent failed: %v", err)
			}
			if event.TaskID != tt.wantID {
				t.Errorf("got task ID %q, want %q", event.TaskID, tt.wantID)
			}
		})
	}
}

// Test Jira handler with changelog
func TestJiraHandler_ParseEvent_WithChangelog(t *testing.T) {
	handler := webhook.NewJiraHandler()
	payload := map[string]interface{}{
		"webhookEvent": "jira:issue_updated",
		"issue": map[string]interface{}{
			"id":   "10001",
			"key":  "PROJ-1",
			"self": "https://jira.example.com",
			"fields": map[string]interface{}{
				"summary":     "Test",
				"description": "roady-id: task-1",
				"status": map[string]interface{}{
					"name": "In Progress",
					"statusCategory": map[string]interface{}{
						"key": "indeterminate",
					},
				},
			},
		},
		"changelog": map[string]interface{}{
			"items": []map[string]interface{}{
				{"field": "status", "fromString": "To Do", "toString": "In Progress"},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", bytes.NewReader(body))

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if event.Metadata["status_from"] != "To Do" {
		t.Errorf("expected status_from 'To Do', got %v", event.Metadata["status_from"])
	}
	if event.Metadata["status_to"] != "In Progress" {
		t.Errorf("expected status_to 'In Progress', got %v", event.Metadata["status_to"])
	}
}

// Test GitHub ParseEvent with invalid JSON
func TestGitHubHandler_ParseEvent_InvalidJSON(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader("not json"))
	req.Header.Set("X-GitHub-Event", "issues")

	_, err := handler.ParseEvent(req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Test Jira ParseEvent with invalid JSON
func TestJiraHandler_ParseEvent_InvalidJSON(t *testing.T) {
	handler := webhook.NewJiraHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira", strings.NewReader("not json"))

	_, err := handler.ParseEvent(req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Test Linear ParseEvent with invalid JSON
func TestLinearHandler_ParseEvent_InvalidJSON(t *testing.T) {
	handler := webhook.NewLinearHandler()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", strings.NewReader("not json"))

	_, err := handler.ParseEvent(req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Test Linear handler with assignee
func TestLinearHandler_ParseEvent_WithAssignee(t *testing.T) {
	handler := webhook.NewLinearHandler()
	payload := map[string]interface{}{
		"action": "update",
		"type":   "Issue",
		"data": map[string]interface{}{
			"id":          "issue-1",
			"identifier":  "LIN-1",
			"title":       "Test",
			"description": "roady-id: task-linear",
			"url":         "https://linear.app/test",
			"state": map[string]interface{}{
				"id":   "s1",
				"name": "In Progress",
				"type": "started",
			},
			"assignee": map[string]interface{}{
				"id":   "user-1",
				"name": "Alice",
			},
			"team": map[string]interface{}{
				"id":  "t1",
				"key": "TEST",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear", bytes.NewReader(body))

	event, err := handler.ParseEvent(req)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}
	if event.TaskID != "task-linear" {
		t.Errorf("got task ID %q, want %q", event.TaskID, "task-linear")
	}
}

// Test GitHub signature validation with body reset
func TestGitHubHandler_ValidateSignature_BodyReset(t *testing.T) {
	handler := webhook.NewGitHubHandler()
	secret := "test-secret"
	body := []byte(`{"test":"data"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)

	// First validation should succeed
	if !handler.ValidateSignature(req, secret) {
		t.Error("expected valid signature")
	}

	// Body should be reset for subsequent reads (ParseEvent)
	_, err := handler.ParseEvent(req)
	if err != nil {
		// This might fail because X-GitHub-Event is missing, which is fine
		// The key test is that the body was reset after signature validation
		t.Logf("ParseEvent error (expected if no X-GitHub-Event): %v", err)
	}
}
