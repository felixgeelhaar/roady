package main

import (
	"testing"
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
