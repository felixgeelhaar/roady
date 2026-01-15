package main

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/google/go-github/v69/github"
)

func TestGitHubSyncer_InitFallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "owner/name")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.token != "token-123" {
		t.Fatalf("expected token from env, got %q", s.token)
	}
	if s.owner != "owner" {
		t.Fatalf("expected owner from env, got %q", s.owner)
	}
	if s.name != "name" {
		t.Fatalf("expected name from env, got %q", s.name)
	}
}

func TestGitHubSyncer_Init_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_REPO", "owner/repo")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestGitHubSyncer_Init_WithToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "owner/repo")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestGitHubSyncer_Init_NoRepo(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err == nil {
		t.Fatal("expected error when repo is missing")
	}
}

func TestGitHubSyncer_Init_InvalidRepoFormat(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "invalid-format")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err == nil {
		t.Fatal("expected error for invalid repo format")
	}
}

func TestGitHubSyncer_Init_ConfigOverride(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	t.Setenv("GITHUB_REPO", "env-owner/env-repo")

	s := &GitHubSyncer{}
	cfg := map[string]string{
		"token": "config-token",
		"repo":  "config-owner/config-repo",
	}
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.token != "config-token" {
		t.Fatalf("expected config token to override env, got %q", s.token)
	}
	if s.owner != "config-owner" {
		t.Fatalf("expected config owner, got %q", s.owner)
	}
}

func TestExtractRoadyID(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "with roady-id at end",
			body:     "Task description\n\nroady-id: task-123",
			expected: "task-123",
		},
		{
			name:     "with roady-id in middle",
			body:     "Description\nroady-id: task-456\nmore content",
			expected: "task-456",
		},
		{
			name:     "no roady-id marker",
			body:     "Just a regular issue body",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "roady-id with spaces",
			body:     "Content\nroady-id:   task-789  \nEnd",
			expected: "task-789",
		},
		{
			name:     "roady-id with complex id",
			body:     "roady-id: task-req-core-api",
			expected: "task-req-core-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRoadyID(tt.body)
			if result != tt.expected {
				t.Errorf("extractRoadyID(%q) = %q, want %q", tt.body, result, tt.expected)
			}
		})
	}
}

func TestMapGitHubStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		assignee bool
		expected planning.TaskStatus
	}{
		{
			name:     "closed issue",
			state:    "closed",
			assignee: false,
			expected: planning.StatusDone,
		},
		{
			name:     "closed with assignee",
			state:    "closed",
			assignee: true,
			expected: planning.StatusDone,
		},
		{
			name:     "open with assignee",
			state:    "open",
			assignee: true,
			expected: planning.StatusInProgress,
		},
		{
			name:     "open without assignee",
			state:    "open",
			assignee: false,
			expected: planning.StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &github.Issue{
				State: github.String(tt.state),
			}
			if tt.assignee {
				issue.Assignee = &github.User{Login: github.String("user")}
			}

			result := mapGitHubStatus(issue)
			if result != tt.expected {
				t.Errorf("mapGitHubStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}
