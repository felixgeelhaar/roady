package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestNotionSyncer_Init(t *testing.T) {
	syncer := &NotionSyncer{}

	// Missing token
	err := syncer.Init(map[string]string{"database_id": "test-db"})
	if err == nil {
		t.Error("Expected error for missing token")
	}

	// Missing database_id
	err = syncer.Init(map[string]string{"token": "test-token"})
	if err == nil {
		t.Error("Expected error for missing database_id")
	}

	// Valid config
	err = syncer.Init(map[string]string{
		"token":       "test-token",
		"database_id": "test-db-id",
	})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNotionSyncer_Sync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/databases/"):
			// Query database - return one existing page
			_ = json.NewEncoder(w).Encode(NotionQueryResult{
				Results: []NotionPage{
					{
						ID:  "page-1",
						URL: "https://notion.so/page-1",
						Properties: map[string]interface{}{
							"Roady ID": map[string]interface{}{
								"rich_text": []interface{}{
									map[string]interface{}{"plain_text": "t1"},
								},
							},
							"Name": map[string]interface{}{
								"title": []interface{}{
									map[string]interface{}{"plain_text": "Task 1"},
								},
							},
							"Status": map[string]interface{}{
								"status": map[string]interface{}{"name": "In Progress"},
							},
						},
					},
				},
				HasMore: false,
			})
		case r.Method == "POST" && r.URL.Path == "/pages":
			// Create page
			_ = json.NewEncoder(w).Encode(NotionPage{
				ID:         "page-2",
				URL:        "https://notion.so/page-2",
				Properties: map[string]interface{}{},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := notionBaseURL
	notionBaseURL = server.URL
	defer func() { notionBaseURL = orig }()

	syncer := &NotionSyncer{
		token:      "test-token",
		databaseID: "test-db",
		client:     server.Client(),
	}

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2", Description: "New task"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if _, ok := result.LinkUpdates["t1"]; !ok {
		t.Error("expected link update for existing t1")
	}
	if _, ok := result.LinkUpdates["t2"]; !ok {
		t.Error("expected link update for created t2")
	}
	if result.StatusUpdates["t1"] != planning.StatusInProgress {
		t.Errorf("expected t1 status in_progress, got %q", result.StatusUpdates["t1"])
	}
}

func TestExtractRoadyIDFromPage(t *testing.T) {
	tests := []struct {
		name     string
		page     NotionPage
		expected string
	}{
		{
			name: "with roady-id property",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Roady ID": map[string]interface{}{
						"rich_text": []interface{}{
							map[string]interface{}{
								"plain_text": "task-123",
							},
						},
					},
				},
			},
			expected: "task-123",
		},
		{
			name: "without roady-id property",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Name": map[string]interface{}{},
				},
			},
			expected: "",
		},
		{
			name: "empty properties",
			page: NotionPage{
				Properties: map[string]interface{}{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRoadyIDFromPage(tt.page)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetPageTitle(t *testing.T) {
	tests := []struct {
		name     string
		page     NotionPage
		expected string
	}{
		{
			name: "with Name property",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Name": map[string]interface{}{
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "My Task Title",
							},
						},
					},
				},
			},
			expected: "My Task Title",
		},
		{
			name: "with Title property",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Title": map[string]interface{}{
						"title": []interface{}{
							map[string]interface{}{
								"plain_text": "Another Title",
							},
						},
					},
				},
			},
			expected: "Another Title",
		},
		{
			name: "no title property",
			page: NotionPage{
				Properties: map[string]interface{}{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPageTitle(tt.page)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMapNotionStatus(t *testing.T) {
	tests := []struct {
		name     string
		page     NotionPage
		expected planning.TaskStatus
	}{
		{
			name: "done status",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Status": map[string]interface{}{
						"status": map[string]interface{}{
							"name": "Done",
						},
					},
				},
			},
			expected: planning.StatusDone,
		},
		{
			name: "in progress status",
			page: NotionPage{
				Properties: map[string]interface{}{
					"Status": map[string]interface{}{
						"status": map[string]interface{}{
							"name": "In Progress",
						},
					},
				},
			},
			expected: planning.StatusInProgress,
		},
		{
			name: "no status",
			page: NotionPage{
				Properties: map[string]interface{}{},
			},
			expected: planning.StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapNotionStatus(tt.page)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMapRoadyToNotionStatus(t *testing.T) {
	tests := []struct {
		status   planning.TaskStatus
		expected string
	}{
		{planning.StatusDone, "Done"},
		{planning.StatusInProgress, "In Progress"},
		{planning.StatusBlocked, "Blocked"},
		{planning.StatusVerified, "Done"},
		{planning.StatusPending, "Not Started"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := mapRoadyToNotionStatus(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShortenID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"12345678-1234-1234-1234-123456789012", "12345678"},
		{"abc", "abc"},
		{"12345678901234567890", "12345678"},
	}

	for _, tt := range tests {
		result := shortenID(tt.id)
		if result != tt.expected {
			t.Errorf("shortenID(%q) = %q, want %q", tt.id, result, tt.expected)
		}
	}
}

func TestNotionSyncer_Push(t *testing.T) {
	var patchCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/databases/"):
			_ = json.NewEncoder(w).Encode(NotionQueryResult{
				Results: []NotionPage{
					{
						ID: "page-1",
						Properties: map[string]interface{}{
							"Roady ID": map[string]interface{}{
								"rich_text": []interface{}{
									map[string]interface{}{"plain_text": "t1"},
								},
							},
						},
					},
				},
				HasMore: false,
			})
		case r.Method == "PATCH":
			patchCalled = true
			_ = json.NewEncoder(w).Encode(NotionPage{ID: "page-1"})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := notionBaseURL
	notionBaseURL = server.URL
	defer func() { notionBaseURL = orig }()

	syncer := &NotionSyncer{
		token:      "test-token",
		databaseID: "test-db",
		client:     server.Client(),
	}

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !patchCalled {
		t.Error("expected PATCH to be called")
	}
}

func TestNotionSyncer_Push_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(NotionQueryResult{Results: []NotionPage{}, HasMore: false})
	}))
	defer server.Close()

	orig := notionBaseURL
	notionBaseURL = server.URL
	defer func() { notionBaseURL = orig }()

	syncer := &NotionSyncer{
		token:      "test-token",
		databaseID: "test-db",
		client:     server.Client(),
	}

	err := syncer.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for page not found")
	}
}

func TestMapNotionStatus_Blocked(t *testing.T) {
	page := NotionPage{
		Properties: map[string]interface{}{
			"Status": map[string]interface{}{
				"status": map[string]interface{}{"name": "Blocked"},
			},
		},
	}
	if got := mapNotionStatus(page); got != planning.StatusBlocked {
		t.Errorf("expected blocked, got %s", got)
	}
}

func TestMapNotionStatus_Verified(t *testing.T) {
	page := NotionPage{
		Properties: map[string]interface{}{
			"Status": map[string]interface{}{
				"status": map[string]interface{}{"name": "Verified"},
			},
		},
	}
	if got := mapNotionStatus(page); got != planning.StatusVerified {
		t.Errorf("expected verified, got %s", got)
	}
}
