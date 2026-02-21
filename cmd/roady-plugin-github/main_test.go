package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
				State: github.Ptr(tt.state),
			}
			if tt.assignee {
				issue.Assignee = &github.User{Login: github.Ptr("user")}
			}

			result := mapGitHubStatus(issue)
			if result != tt.expected {
				t.Errorf("mapGitHubStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitHubSyncer_Sync(t *testing.T) {
	mux := http.NewServeMux()
	// List issues endpoint
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			issues := []*github.Issue{
				{
					Number:   github.Ptr(1),
					Title:    github.Ptr("Task 1"),
					Body:     github.Ptr("Description\n\nroady-id: t1"),
					State:    github.Ptr("open"),
					HTMLURL:  github.Ptr("https://github.com/owner/repo/issues/1"),
					Assignee: &github.User{Login: github.Ptr("dev1")},
				},
			}
			_ = json.NewEncoder(w).Encode(issues)
			return
		}
		if r.Method == "POST" {
			// Create issue
			issue := &github.Issue{
				Number:  github.Ptr(2),
				Title:   github.Ptr("Task 2"),
				Body:    github.Ptr("roady-id: t2"),
				State:   github.Ptr("open"),
				HTMLURL: github.Ptr("https://github.com/owner/repo/issues/2"),
			}
			_ = json.NewEncoder(w).Encode(issue)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		repo:  "owner/repo",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

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

	// t1 should match by roady-id, open+assigned = in_progress
	if result.StatusUpdates["t1"] != planning.StatusInProgress {
		t.Errorf("expected t1 in_progress, got %q", result.StatusUpdates["t1"])
	}
	// t2 should be created
	if _, ok := result.LinkUpdates["t2"]; !ok {
		t.Error("expected link update for created t2")
	}
}

func TestGitHubSyncer_Push(t *testing.T) {
	var editCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		issues := []*github.Issue{
			{
				Number:  github.Ptr(1),
				Body:    github.Ptr("roady-id: t1"),
				State:   github.Ptr("open"),
				HTMLURL: github.Ptr("https://github.com/owner/repo/issues/1"),
			},
		}
		_ = json.NewEncoder(w).Encode(issues)
	})
	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			editCalled = true
			_ = json.NewEncoder(w).Encode(&github.Issue{
				Number: github.Ptr(1),
				State:  github.Ptr("closed"),
			})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !editCalled {
		t.Error("expected edit to be called to close the issue")
	}
}

func TestGitHubSyncer_Push_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Issue{})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	err := syncer.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for issue not found")
	}
}

func TestGitHubSyncer_Push_UnsupportedStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Issue{
			{
				Number: github.Ptr(1),
				Body:   github.Ptr("roady-id: t1"),
				State:  github.Ptr("open"),
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	err := syncer.Push("t1", planning.StatusVerified)
	if err == nil {
		t.Error("expected error for unsupported status")
	}
}

func TestGitHubSyncer_Push_AlreadyCorrectState(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Issue{
			{
				Number: github.Ptr(1),
				Body:   github.Ptr("roady-id: t1"),
				State:  github.Ptr("closed"),
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	// Push done to already-closed issue should succeed without edit
	err := syncer.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestGitHubSyncer_Sync_WithExistingRef(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		issues := []*github.Issue{
			{
				Number:  github.Ptr(42),
				Title:   github.Ptr("Task 1"),
				Body:    github.Ptr("roady-id: t1"),
				State:   github.Ptr("closed"),
				HTMLURL: github.Ptr("https://github.com/owner/repo/issues/42"),
			},
		}
		_ = json.NewEncoder(w).Encode(issues)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	state := planning.NewExecutionState("p1")
	state.TaskStates["t1"] = planning.TaskResult{
		Status: planning.StatusPending,
		ExternalRefs: map[string]planning.ExternalRef{
			"github": {ID: "42"},
		},
	}

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if result.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("expected t1 done, got %q", result.StatusUpdates["t1"])
	}
}

func TestGitHubSyncer_Push_ReopenIssue(t *testing.T) {
	var editCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Issue{
			{
				Number: github.Ptr(1),
				Body:   github.Ptr("roady-id: t1"),
				State:  github.Ptr("closed"),
			},
		})
	})
	mux.HandleFunc("/repos/owner/repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			editCalled = true
			_ = json.NewEncoder(w).Encode(&github.Issue{
				Number: github.Ptr(1),
				State:  github.Ptr("open"),
			})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	// Push pending to a closed issue should reopen it
	err := syncer.Push("t1", planning.StatusPending)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !editCalled {
		t.Error("expected edit to be called to reopen the issue")
	}
}

func TestGitHubSyncer_Sync_TitleMatchFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Issue without roady-id marker but matching title
			issues := []*github.Issue{
				{
					Number:  github.Ptr(5),
					Title:   github.Ptr("Implement auth"),
					Body:    github.Ptr("No roady marker here"),
					State:   github.Ptr("open"),
					HTMLURL: github.Ptr("https://github.com/owner/repo/issues/5"),
				},
			}
			_ = json.NewEncoder(w).Encode(issues)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL + "/")

	syncer := &GitHubSyncer{
		token: "test-token",
		owner: "owner",
		name:  "repo",
	}
	syncer.client = github.NewClient(nil)
	syncer.client.BaseURL = serverURL

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Implement auth"},
		},
	}
	state := planning.NewExecutionState("p1")

	result, err := syncer.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Should match by title and link
	if ref, ok := result.LinkUpdates["t1"]; !ok {
		t.Error("expected link update for title-matched t1")
	} else if ref.ID != fmt.Sprintf("%d", 5) {
		t.Errorf("expected ref ID '5', got %q", ref.ID)
	}

	// Open without assignee = pending, same as default, so no status update
	if _, ok := result.StatusUpdates["t1"]; ok {
		t.Errorf("expected no status update for pending task matching open unassigned issue")
	}
}
