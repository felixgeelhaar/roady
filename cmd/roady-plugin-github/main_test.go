package main

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestGitHubSyncer_InitFallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "owner/repo")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.token != "token-123" {
		t.Fatalf("expected token from env, got %q", s.token)
	}
	if s.repo != "owner/repo" {
		t.Fatalf("expected repo from env, got %q", s.repo)
	}
}

func TestGitHubSyncer_Sync_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_REPO", "owner/repo")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
		},
	}
	state := planning.NewExecutionState("p1")

	res, err := s.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil sync result when token is missing, got %+v", res)
	}
}

func TestGitHubSyncer_Sync_WithToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_REPO", "owner/repo")

	s := &GitHubSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
		},
	}
	state := planning.NewExecutionState("p1")

	res, err := s.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected sync result with token present")
	}
	if len(res.StatusUpdates) != 0 {
		t.Fatalf("expected no status updates, got %v", res.StatusUpdates)
	}
}
