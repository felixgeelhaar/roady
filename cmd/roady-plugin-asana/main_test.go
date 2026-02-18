package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestAsanaSyncer_Init(t *testing.T) {
	syncer := &AsanaSyncer{}

	// Missing token
	err := syncer.Init(map[string]string{"project_id": "test-project"})
	if err == nil {
		t.Error("Expected error for missing token")
	}

	// Missing project_id
	err = syncer.Init(map[string]string{"token": "test-token"})
	if err == nil {
		t.Error("Expected error for missing project_id")
	}

	// Valid config
	err = syncer.Init(map[string]string{
		"token":      "test-token",
		"project_id": "test-project-id",
	})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestExtractRoadyIDFromNotes(t *testing.T) {
	tests := []struct {
		name     string
		notes    string
		expected string
	}{
		{
			name:     "with roady-id",
			notes:    "Task description\n\nroady-id: task-123",
			expected: "task-123",
		},
		{
			name:     "roady-id with newline after",
			notes:    "Description\n\nroady-id: task-456\nMore text",
			expected: "task-456",
		},
		{
			name:     "no roady-id",
			notes:    "Just a regular task description",
			expected: "",
		},
		{
			name:     "empty notes",
			notes:    "",
			expected: "",
		},
		{
			name:     "roady-id with spaces",
			notes:    "roady-id:   task-789  \n",
			expected: "task-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRoadyIDFromNotes(tt.notes)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMapAsanaStatus(t *testing.T) {
	tests := []struct {
		name     string
		task     AsanaTask
		expected planning.TaskStatus
	}{
		{
			name: "completed task",
			task: AsanaTask{
				Completed: true,
			},
			expected: planning.StatusDone,
		},
		{
			name: "task with assignee",
			task: AsanaTask{
				Completed: false,
				Assignee: &struct {
					GID string `json:"gid"`
				}{GID: "user-123"},
			},
			expected: planning.StatusInProgress,
		},
		{
			name: "unassigned incomplete task",
			task: AsanaTask{
				Completed: false,
				Assignee:  nil,
			},
			expected: planning.StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapAsanaStatus(tt.task)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestShortenGID(t *testing.T) {
	tests := []struct {
		gid      string
		expected string
	}{
		{"1234567890123456", "12345678"},
		{"abc", "abc"},
		{"12345678", "12345678"},
		{"123456789", "12345678"},
	}

	for _, tt := range tests {
		result := shortenGID(tt.gid)
		if result != tt.expected {
			t.Errorf("shortenGID(%q) = %q, want %q", tt.gid, result, tt.expected)
		}
	}
}

func TestAsanaTask_CustomFields(t *testing.T) {
	task := AsanaTask{
		GID:       "123",
		Name:      "Test Task",
		Completed: false,
		CustomFields: []AsanaCustomField{
			{
				GID:       "field-1",
				Name:      "Priority",
				TextValue: "High",
			},
		},
	}

	if len(task.CustomFields) != 1 {
		t.Errorf("Expected 1 custom field, got %d", len(task.CustomFields))
	}
	if task.CustomFields[0].Name != "Priority" {
		t.Errorf("Expected Priority field, got %s", task.CustomFields[0].Name)
	}
}

func TestAsanaSyncer_Sync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/projects/"):
			_ = json.NewEncoder(w).Encode(AsanaTasksResponse{
				Data: []AsanaTask{
					{GID: "111", Name: "Task 1", Notes: "roady-id: t1", Completed: false, PermalinkURL: "https://asana.com/111"},
				},
			})
		case r.Method == "POST" && r.URL.Path == "/tasks":
			_ = json.NewEncoder(w).Encode(AsanaTaskResponse{
				Data: AsanaTask{GID: "222", Name: "Task 2", Notes: "roady-id: t2", PermalinkURL: "https://asana.com/222"},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := asanaBaseURL
	asanaBaseURL = server.URL
	defer func() { asanaBaseURL = orig }()

	syncer := &AsanaSyncer{
		token:     "test-token",
		projectID: "test-project",
		client:    server.Client(),
	}

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if _, ok := result.LinkUpdates["t2"]; !ok {
		t.Error("expected link update for created task t2")
	}
	if _, ok := result.LinkUpdates["t1"]; !ok {
		t.Error("expected link update for existing task t1")
	}
}

func TestAsanaSyncer_Push(t *testing.T) {
	var receivedMethod, receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			_ = json.NewEncoder(w).Encode(AsanaTasksResponse{
				Data: []AsanaTask{
					{GID: "111", Name: "Task 1", Notes: "roady-id: t1", Completed: false},
				},
			})
		case "PUT":
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(AsanaTaskResponse{
				Data: AsanaTask{GID: "111", Completed: true},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := asanaBaseURL
	asanaBaseURL = server.URL
	defer func() { asanaBaseURL = orig }()

	syncer := &AsanaSyncer{
		token:     "test-token",
		projectID: "test-project",
		client:    server.Client(),
	}

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if receivedMethod != "PUT" {
		t.Errorf("expected PUT request, got %s", receivedMethod)
	}
	if !strings.Contains(receivedPath, "/tasks/111") {
		t.Errorf("expected PUT to /tasks/111, got %s", receivedPath)
	}
}

func TestAsanaSyncer_Push_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(AsanaTasksResponse{Data: []AsanaTask{}})
	}))
	defer server.Close()

	orig := asanaBaseURL
	asanaBaseURL = server.URL
	defer func() { asanaBaseURL = orig }()

	syncer := &AsanaSyncer{
		token:     "test-token",
		projectID: "test-project",
		client:    server.Client(),
	}

	err := syncer.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for task not found")
	}
}

func TestAsanaSyncer_Push_AlreadyCorrectState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			t.Error("should not PUT when already in correct state")
		}
		_ = json.NewEncoder(w).Encode(AsanaTasksResponse{
			Data: []AsanaTask{
				{GID: "111", Notes: "roady-id: t1", Completed: true},
			},
		})
	}))
	defer server.Close()

	orig := asanaBaseURL
	asanaBaseURL = server.URL
	defer func() { asanaBaseURL = orig }()

	syncer := &AsanaSyncer{
		token:     "test-token",
		projectID: "test-project",
		client:    server.Client(),
	}

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}
