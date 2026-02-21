package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jira "github.com/felixgeelhaar/jirasdk"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestJiraSyncer_InitMissing(t *testing.T) {
	s := &JiraSyncer{}
	if err := s.Init(map[string]string{}); err == nil {
		t.Fatal("expected error for missing Jira config")
	}
}

func TestJiraSyncer_InitConfigPrefix(t *testing.T) {
	s := &JiraSyncer{}
	cfg := map[string]string{
		"domain":      "example.atlassian.net",
		"project_key": "RD",
		"email":       "test@example.com",
		"api_token":   "token",
	}
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.baseURL != "https://example.atlassian.net" {
		t.Fatalf("expected https prefix, got %q", s.baseURL)
	}
}

func TestJiraSyncer_InitWithHttps(t *testing.T) {
	s := &JiraSyncer{}
	cfg := map[string]string{
		"domain":      "https://example.atlassian.net",
		"project_key": "RD",
		"email":       "test@example.com",
		"api_token":   "token",
	}
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.baseURL != "https://example.atlassian.net" {
		t.Fatalf("expected unchanged https URL, got %q", s.baseURL)
	}
}

func TestJiraHelpers(t *testing.T) {
	// Test extractRoadyID
	id := extractRoadyID("hello\nroady-id: task-123")
	if id != "task-123" {
		t.Fatalf("expected roady id, got %q", id)
	}

	// Test extractRoadyID with trailing content
	id = extractRoadyID("hello\nroady-id: task-456\nmore content")
	if id != "task-456" {
		t.Fatalf("expected roady id with trailing content, got %q", id)
	}

	// Test extractRoadyID with no marker
	id = extractRoadyID("no roady id here")
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}

	// Test mapJiraStatus - returns planning.TaskStatus values
	if got := mapJiraStatus("Done"); got != planning.StatusDone {
		t.Fatalf("expected StatusDone, got %v", got)
	}
	if got := mapJiraStatus("resolved"); got != planning.StatusDone {
		t.Fatalf("expected StatusDone for resolved, got %v", got)
	}
	if got := mapJiraStatus("In Progress"); got != planning.StatusInProgress {
		t.Fatalf("expected StatusInProgress, got %v", got)
	}
	if got := mapJiraStatus("started"); got != planning.StatusInProgress {
		t.Fatalf("expected StatusInProgress for started, got %v", got)
	}
	if got := mapJiraStatus("Blocked"); got != planning.StatusBlocked {
		t.Fatalf("expected StatusBlocked, got %v", got)
	}
	if got := mapJiraStatus("on hold"); got != planning.StatusBlocked {
		t.Fatalf("expected StatusBlocked for on hold, got %v", got)
	}
	if got := mapJiraStatus("unknown"); got != planning.StatusPending {
		t.Fatalf("expected StatusPending, got %v", got)
	}
	if got := mapJiraStatus("To Do"); got != planning.StatusPending {
		t.Fatalf("expected StatusPending for To Do, got %v", got)
	}
}

func TestJiraSyncer_InitPartialConfig(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "missing domain",
			config: map[string]string{
				"project_key": "RD",
				"email":       "test@example.com",
				"api_token":   "token",
			},
		},
		{
			name: "missing project_key",
			config: map[string]string{
				"domain":    "example.atlassian.net",
				"email":     "test@example.com",
				"api_token": "token",
			},
		},
		{
			name: "missing email",
			config: map[string]string{
				"domain":      "example.atlassian.net",
				"project_key": "RD",
				"api_token":   "token",
			},
		},
		{
			name: "missing api_token",
			config: map[string]string{
				"domain":      "example.atlassian.net",
				"project_key": "RD",
				"email":       "test@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &JiraSyncer{}
			if err := s.Init(tt.config); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestJiraSyncer_ProjectKey(t *testing.T) {
	s := &JiraSyncer{}
	cfg := map[string]string{
		"domain":      "example.atlassian.net",
		"project_key": "MYPROJECT",
		"email":       "test@example.com",
		"api_token":   "token",
	}
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.projectKey != "MYPROJECT" {
		t.Fatalf("expected project key MYPROJECT, got %q", s.projectKey)
	}
}

// newTestSyncer creates a JiraSyncer backed by an httptest server.
// The caller provides a handler mux for routing API endpoints.
// Returns the syncer and a cleanup function.
func newTestSyncer(t *testing.T, mux *http.ServeMux) (*JiraSyncer, func()) {
	t.Helper()
	server := httptest.NewServer(mux)

	client, err := jira.NewClient(
		jira.WithBaseURL(server.URL),
		jira.WithAPIToken("test@example.com", "test-token"),
		jira.WithMaxRetries(0),
		jira.WithTimeout(5*time.Second),
	)
	if err != nil {
		server.Close()
		t.Fatalf("failed to create jira client: %v", err)
	}

	syncer := &JiraSyncer{
		client:     client,
		projectKey: "RD",
		baseURL:    server.URL,
	}

	return syncer, server.Close
}

// adfJSON returns a JSON-encoded ADF document containing the given text.
// Each line becomes a separate paragraph in the ADF structure.
func adfJSON(text string) json.RawMessage {
	lines := strings.Split(text, "\n")
	var content []map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		content = append(content, map[string]interface{}{
			"type": "paragraph",
			"content": []map[string]interface{}{
				{"type": "text", "text": line},
			},
		})
	}
	doc := map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
	b, _ := json.Marshal(doc)
	return b
}

// jiraIssueJSON builds a JSON object for a single Jira issue response.
func jiraIssueJSON(id, key, summary, descText, statusName string) map[string]interface{} {
	return map[string]interface{}{
		"id":  id,
		"key": key,
		"fields": map[string]interface{}{
			"summary":     summary,
			"description": json.RawMessage(adfJSON(descText)),
			"status": map[string]interface{}{
				"name": statusName,
			},
		},
	}
}

func TestJiraSyncer_FetchProjectIssues(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "Some description\nroady-id: t1", "To Do"),
				jiraIssueJSON("101", "RD-2", "Task 2", "Another task\nroady-id: t2", "In Progress"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	ctx := t
	_ = ctx
	issues, err := syncer.fetchProjectIssues(t.Context())
	if err != nil {
		t.Fatalf("fetchProjectIssues failed: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Key != "RD-1" {
		t.Errorf("expected first issue key RD-1, got %q", issues[0].Key)
	}
	if issues[1].Key != "RD-2" {
		t.Errorf("expected second issue key RD-2, got %q", issues[1].Key)
	}
}

func TestJiraSyncer_CreateIssue(t *testing.T) {
	var createCalled bool
	mux := http.NewServeMux()

	// POST /rest/api/3/issue - Create endpoint
	mux.HandleFunc("/rest/api/3/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		createCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "200",
			"key":  "RD-10",
			"self": "https://example.atlassian.net/rest/api/3/issue/200",
		})
	})

	// GET /rest/api/3/issue/RD-10 - Get endpoint (called after create to fetch full issue)
	mux.HandleFunc("/rest/api/3/issue/RD-10", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jiraIssueJSON("200", "RD-10", "New Task", "New task desc\nroady-id: t-new", "To Do"))
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	task := planning.Task{
		ID:          "t-new",
		Title:       "New Task",
		Description: "New task desc",
	}

	iss, err := syncer.createIssue(t.Context(), task)
	if err != nil {
		t.Fatalf("createIssue failed: %v", err)
	}
	if !createCalled {
		t.Error("expected create endpoint to be called")
	}
	if iss.Key != "RD-10" {
		t.Errorf("expected issue key RD-10, got %q", iss.Key)
	}
	if iss.GetStatusName() != "To Do" {
		t.Errorf("expected status 'To Do', got %q", iss.GetStatusName())
	}
}

func TestJiraSyncer_Sync(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "Description\nroady-id: t1", "In Progress"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// t1 matched by roady-id, status "In Progress" maps to StatusInProgress
	if result.StatusUpdates["t1"] != planning.StatusInProgress {
		t.Errorf("expected t1 status in_progress, got %q", result.StatusUpdates["t1"])
	}
}

func TestJiraSyncer_Sync_CreatesMissing(t *testing.T) {
	mux := http.NewServeMux()

	// Search returns one existing issue for t1
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "Desc\nroady-id: t1", "Done"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Create endpoint for t2
	mux.HandleFunc("/rest/api/3/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "201",
				"key":  "RD-2",
				"self": "https://example.atlassian.net/rest/api/3/issue/201",
			})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	// Get endpoint for fetching the newly created issue
	mux.HandleFunc("/rest/api/3/issue/RD-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jiraIssueJSON("201", "RD-2", "Task 2", "task 2 desc\nroady-id: t2", "To Do"))
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2", Description: "task 2 desc"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// t1 is Done, current state is pending, so we get a status update
	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("expected t1 done, got %q", result.StatusUpdates["t1"])
	}

	// t2 was created, so we should have a link update
	if ref, ok := result.LinkUpdates["t2"]; !ok {
		t.Error("expected link update for created t2")
	} else {
		if ref.ID != "201" {
			t.Errorf("expected link ID '201', got %q", ref.ID)
		}
		if ref.Identifier != "RD-2" {
			t.Errorf("expected link identifier 'RD-2', got %q", ref.Identifier)
		}
	}
}

func TestJiraSyncer_Sync_WithExistingRef(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "Desc\nroady-id: t1", "Done"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	state := planning.NewExecutionState("p1")
	// Pre-populate with an existing external ref matching by ID
	state.TaskStates["t1"] = planning.TaskResult{
		Status: planning.StatusPending,
		ExternalRefs: map[string]planning.ExternalRef{
			"jira": {ID: "100", Identifier: "RD-1"},
		},
	}

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Should match by external ref (ID=100), status Done maps to StatusDone
	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("expected t1 done via external ref, got %q", result.StatusUpdates["t1"])
	}
}

func TestJiraSyncer_Sync_ExistingRefByKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "no roady marker", "In Progress"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	state := planning.NewExecutionState("p1")
	// Match by Key via external ref
	state.TaskStates["t1"] = planning.TaskResult{
		Status: planning.StatusPending,
		ExternalRefs: map[string]planning.ExternalRef{
			"jira": {ID: "different-id", Identifier: "RD-1"},
		},
	}

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.StatusUpdates["t1"] != planning.StatusInProgress {
		t.Errorf("expected t1 in_progress via key ref, got %q", result.StatusUpdates["t1"])
	}
}

func TestJiraSyncer_Sync_NoStatusChange(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	// State already pending (same as Jira "To Do" -> StatusPending)
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// No status change expected since both are pending
	if _, ok := result.StatusUpdates["t1"]; ok {
		t.Errorf("expected no status update when status matches, got %v", result.StatusUpdates["t1"])
	}
}

func TestJiraSyncer_Sync_CreateFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues":        []interface{}{},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Create fails with server error
	mux.HandleFunc("/rest/api/3/issue", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"server error"},
		})
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1", Description: "desc"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync should not fail entirely on create error, got: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("expected errors in result when create fails")
	}
}

func TestJiraSyncer_Push(t *testing.T) {
	var transitionCalled bool
	mux := http.NewServeMux()

	// Search returns one issue matching the task
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "Desc\nroady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Transition endpoint
	mux.HandleFunc("/rest/api/3/issue/RD-1/transitions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			transitionCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !transitionCalled {
		t.Error("expected transition endpoint to be called")
	}
}

func TestJiraSyncer_Push_InProgress(t *testing.T) {
	var transitionCalled bool
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/rest/api/3/issue/RD-1/transitions", func(w http.ResponseWriter, r *http.Request) {
		transitionCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusInProgress)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !transitionCalled {
		t.Error("expected transition to In Progress")
	}
}

func TestJiraSyncer_Push_Pending(t *testing.T) {
	var transitionCalled bool
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "In Progress"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/rest/api/3/issue/RD-1/transitions", func(w http.ResponseWriter, r *http.Request) {
		transitionCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusPending)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !transitionCalled {
		t.Error("expected transition to To Do")
	}
}

func TestJiraSyncer_Push_NotFound(t *testing.T) {
	mux := http.NewServeMux()

	// Search returns no matching issues
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues":        []interface{}{},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for issue not found")
	}
	if !strings.Contains(err.Error(), "issue not found") {
		t.Errorf("expected 'issue not found' in error, got: %v", err)
	}
}

func TestJiraSyncer_Push_Blocked(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusBlocked)
	if err == nil {
		t.Error("expected error for blocked status")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error message, got: %v", err)
	}
}

func TestJiraSyncer_Push_UnsupportedStatus(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusVerified)
	if err == nil {
		t.Error("expected error for unsupported status")
	}
	if !strings.Contains(err.Error(), "unsupported status") {
		t.Errorf("expected 'unsupported status' in error, got: %v", err)
	}
}

func TestJiraSyncer_Push_TransitionFails(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "To Do"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/rest/api/3/issue/RD-1/transitions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"transition not valid"},
		})
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusDone)
	if err == nil {
		t.Error("expected error when transition fails")
	}
}

func TestJiraSyncer_Sync_FetchError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"internal error"},
		})
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	state := planning.NewExecutionState("p1")

	_, err := syncer.Sync(plan, state)
	if err == nil {
		t.Error("expected error when fetch fails")
	}
}

func TestJiraSyncer_Push_FetchError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"internal error"},
		})
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	err := syncer.Push("t1", planning.StatusDone)
	if err == nil {
		t.Error("expected error when fetch fails in Push")
	}
	if !strings.Contains(err.Error(), "fetch issues") {
		t.Errorf("expected 'fetch issues' in error, got: %v", err)
	}
}

func TestJiraSyncer_InitEnvFallback(t *testing.T) {
	t.Setenv("JIRA_DOMAIN", "env.atlassian.net")
	t.Setenv("JIRA_PROJECT_KEY", "ENV")
	t.Setenv("JIRA_EMAIL", "env@example.com")
	t.Setenv("JIRA_API_TOKEN", "env-token")

	s := &JiraSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init with env vars failed: %v", err)
	}
	if s.baseURL != "https://env.atlassian.net" {
		t.Errorf("expected env domain, got %q", s.baseURL)
	}
	if s.projectKey != "ENV" {
		t.Errorf("expected env project key, got %q", s.projectKey)
	}
}

func TestJiraSyncer_Sync_MultipleTasks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"issues": []interface{}{
				jiraIssueJSON("100", "RD-1", "Task 1", "roady-id: t1", "Done"),
				jiraIssueJSON("101", "RD-2", "Task 2", "roady-id: t2", "Blocked"),
				jiraIssueJSON("102", "RD-3", "Task 3", "roady-id: t3", "In Progress"),
			},
			"nextPageToken": "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	syncer, cleanup := newTestSyncer(t, mux)
	defer cleanup()

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
			{ID: "t3", Title: "Task 3"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("expected t1 done, got %q", result.StatusUpdates["t1"])
	}
	if result.StatusUpdates["t2"] != planning.StatusBlocked {
		t.Errorf("expected t2 blocked, got %q", result.StatusUpdates["t2"])
	}
	if result.StatusUpdates["t3"] != planning.StatusInProgress {
		t.Errorf("expected t3 in_progress, got %q", result.StatusUpdates["t3"])
	}
}
