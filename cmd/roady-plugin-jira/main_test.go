package main

import (
	"testing"

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
