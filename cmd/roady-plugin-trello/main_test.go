package main

import (
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
