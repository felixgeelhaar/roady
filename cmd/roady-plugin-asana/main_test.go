package main

import (
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
