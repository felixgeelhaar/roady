package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestTrelloSyncer_Init_MissingConfig(t *testing.T) {
	syncer := &TrelloSyncer{}

	// Missing api_key
	err := syncer.Init(map[string]string{
		"token":    "test-token",
		"board_id": "test-board",
	})
	if err == nil {
		t.Error("Expected error for missing api_key")
	}

	// Missing token
	err = syncer.Init(map[string]string{
		"api_key":  "test-key",
		"board_id": "test-board",
	})
	if err == nil {
		t.Error("Expected error for missing token")
	}

	// Missing board_id
	err = syncer.Init(map[string]string{
		"api_key": "test-key",
		"token":   "test-token",
	})
	if err == nil {
		t.Error("Expected error for missing board_id")
	}
}

func TestExtractRoadyIDFromDesc(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		expected string
	}{
		{
			name:     "with roady-id",
			desc:     "Card description\n\nroady-id: task-123",
			expected: "task-123",
		},
		{
			name:     "roady-id with newline after",
			desc:     "Description\n\nroady-id: task-456\nMore text",
			expected: "task-456",
		},
		{
			name:     "no roady-id",
			desc:     "Just a regular card description",
			expected: "",
		},
		{
			name:     "empty description",
			desc:     "",
			expected: "",
		},
		{
			name:     "roady-id with spaces",
			desc:     "roady-id:   task-789  \n",
			expected: "task-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRoadyIDFromDesc(tt.desc)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTrelloSyncer_MapTrelloStatus(t *testing.T) {
	syncer := &TrelloSyncer{
		doneListID: "done-list-id",
		lists: map[string]TrelloList{
			"todo-list-id":       {ID: "todo-list-id", Name: "To Do"},
			"progress-list-id":   {ID: "progress-list-id", Name: "In Progress"},
			"done-list-id":       {ID: "done-list-id", Name: "Done"},
			"blocked-list-id":    {ID: "blocked-list-id", Name: "Blocked"},
		},
	}

	tests := []struct {
		name     string
		card     TrelloCard
		expected planning.TaskStatus
	}{
		{
			name: "card in done list",
			card: TrelloCard{
				IDList: "done-list-id",
			},
			expected: planning.StatusDone,
		},
		{
			name: "card with members (in progress)",
			card: TrelloCard{
				IDList: "todo-list-id",
				Members: []struct {
					ID string `json:"id"`
				}{{ID: "member-1"}},
			},
			expected: planning.StatusInProgress,
		},
		{
			name: "card in progress list",
			card: TrelloCard{
				IDList: "progress-list-id",
			},
			expected: planning.StatusInProgress,
		},
		{
			name: "card in blocked list",
			card: TrelloCard{
				IDList: "blocked-list-id",
			},
			expected: planning.StatusBlocked,
		},
		{
			name: "card in todo list without members",
			card: TrelloCard{
				IDList: "todo-list-id",
			},
			expected: planning.StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncer.mapTrelloStatus(tt.card)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTrelloSyncer_MapRoadyToTrelloList(t *testing.T) {
	syncer := &TrelloSyncer{
		todoListID: "todo-list-id",
		doneListID: "done-list-id",
	}

	tests := []struct {
		status   planning.TaskStatus
		expected string
	}{
		{planning.StatusDone, "done-list-id"},
		{planning.StatusVerified, "done-list-id"},
		{planning.StatusPending, "todo-list-id"},
		{planning.StatusInProgress, "todo-list-id"},
		{planning.StatusBlocked, "todo-list-id"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := syncer.mapRoadyToTrelloList(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTrelloSyncer_BuildURL(t *testing.T) {
	syncer := &TrelloSyncer{
		apiKey: "test-api-key",
		token:  "test-token",
	}

	url := syncer.buildURL("/boards/123/cards", map[string]string{
		"fields": "id,name",
	})

	// Verify URL contains expected parts
	if url == "" {
		t.Error("Expected non-empty URL")
	}
	if !contains(url, "key=test-api-key") {
		t.Error("Expected URL to contain API key")
	}
	if !contains(url, "token=test-token") {
		t.Error("Expected URL to contain token")
	}
	if !contains(url, "fields=id") {
		t.Error("Expected URL to contain fields parameter")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTrelloList_Struct(t *testing.T) {
	list := TrelloList{
		ID:   "list-123",
		Name: "Test List",
	}

	if list.ID != "list-123" {
		t.Errorf("Expected ID list-123, got %s", list.ID)
	}
	if list.Name != "Test List" {
		t.Errorf("Expected Name 'Test List', got %s", list.Name)
	}
}

func TestTrelloCard_Struct(t *testing.T) {
	card := TrelloCard{
		ID:      "card-123",
		Name:    "Test Card",
		Desc:    "Test description\n\nroady-id: task-1",
		IDList:  "list-456",
		URL:     "https://trello.com/c/abc123",
		ShortID: 42,
	}

	if card.ID != "card-123" {
		t.Errorf("Expected ID card-123, got %s", card.ID)
	}
	if card.ShortID != 42 {
		t.Errorf("Expected ShortID 42, got %d", card.ShortID)
	}
}

func TestTrelloSyncer_Sync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/boards/board-1/cards"):
			_ = json.NewEncoder(w).Encode([]TrelloCard{
				{ID: "c1", Name: "Task 1", Desc: "roady-id: t1", IDList: "todo-list", URL: "https://trello.com/c1", ShortID: 1},
			})
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/cards"):
			_ = json.NewEncoder(w).Encode(TrelloCard{
				ID: "c2", Name: "Task 2", Desc: "roady-id: t2", IDList: "todo-list", URL: "https://trello.com/c2", ShortID: 2,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := trelloBaseURL
	trelloBaseURL = server.URL
	defer func() { trelloBaseURL = orig }()

	syncer := &TrelloSyncer{
		apiKey:     "key",
		token:      "token",
		boardID:    "board-1",
		todoListID: "todo-list",
		doneListID: "done-list",
		client:     server.Client(),
		lists: map[string]TrelloList{
			"todo-list": {ID: "todo-list", Name: "To Do"},
			"done-list": {ID: "done-list", Name: "Done"},
		},
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
	if _, ok := result.LinkUpdates["t1"]; !ok {
		t.Error("expected link update for t1")
	}
	if _, ok := result.LinkUpdates["t2"]; !ok {
		t.Error("expected link update for created t2")
	}
}

func TestTrelloSyncer_Push(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/cards"):
			_ = json.NewEncoder(w).Encode([]TrelloCard{
				{ID: "c1", Desc: "roady-id: t1", IDList: "todo-list", ShortID: 1},
			})
		case r.Method == "PUT":
			receivedPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(TrelloCard{ID: "c1", IDList: "done-list"})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	orig := trelloBaseURL
	trelloBaseURL = server.URL
	defer func() { trelloBaseURL = orig }()

	syncer := &TrelloSyncer{
		apiKey:     "key",
		token:      "token",
		boardID:    "board-1",
		todoListID: "todo-list",
		doneListID: "done-list",
		client:     server.Client(),
		lists: map[string]TrelloList{
			"todo-list": {ID: "todo-list", Name: "To Do"},
			"done-list": {ID: "done-list", Name: "Done"},
		},
	}

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !strings.Contains(receivedPath, "/cards/c1") {
		t.Errorf("expected PUT to /cards/c1, got %s", receivedPath)
	}
}

func TestTrelloSyncer_Push_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]TrelloCard{})
	}))
	defer server.Close()

	orig := trelloBaseURL
	trelloBaseURL = server.URL
	defer func() { trelloBaseURL = orig }()

	syncer := &TrelloSyncer{
		apiKey:  "key",
		token:   "token",
		boardID: "board-1",
		client:  server.Client(),
		lists:   map[string]TrelloList{},
	}

	err := syncer.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for card not found")
	}
}

func TestTrelloSyncer_Init_WithHTTPTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]TrelloList{
			{ID: "list-1", Name: "To Do"},
			{ID: "list-2", Name: "Done"},
		})
	}))
	defer server.Close()

	orig := trelloBaseURL
	trelloBaseURL = server.URL
	defer func() { trelloBaseURL = orig }()

	syncer := &TrelloSyncer{}
	err := syncer.Init(map[string]string{
		"api_key":  "key",
		"token":    "token",
		"board_id": "board-1",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if syncer.todoListID != "list-1" {
		t.Errorf("expected todo list auto-detected, got %q", syncer.todoListID)
	}
	if syncer.doneListID != "list-2" {
		t.Errorf("expected done list auto-detected, got %q", syncer.doneListID)
	}
}
