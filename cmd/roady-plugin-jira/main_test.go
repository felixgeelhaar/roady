package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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
	if s.domain != "https://example.atlassian.net" {
		t.Fatalf("expected https prefix, got %q", s.domain)
	}
}

func TestJiraHelpers(t *testing.T) {
	id := extractRoadyID("hello\nroady-id: task-123")
	if id != "task-123" {
		t.Fatalf("expected roady id, got %q", id)
	}

	if got := mapJiraStatus("Done"); got != "done" {
		t.Fatalf("expected done, got %q", got)
	}
	if got := mapJiraStatus("in progress"); got != "in_progress" {
		t.Fatalf("expected in_progress, got %q", got)
	}
	if got := mapJiraStatus("blocked"); got != "blocked" {
		t.Fatalf("expected blocked, got %q", got)
	}
	if got := mapJiraStatus("unknown"); got != "pending" {
		t.Fatalf("expected pending, got %q", got)
	}
}

func TestJiraSyncer_SyncCreatesIssues(t *testing.T) {
	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var respBody string
		switch {
		case req.URL.Path == "/rest/api/2/search":
			respBody = `{"issues":[{"id":"2001","key":"RD-1","fields":{"summary":"Task 1","description":"roady-id: t1","status":{"name":"In Progress","id":"3"}},"self":"self"}]}`
		case req.URL.Path == "/rest/api/2/issue" && req.Method == http.MethodPost:
			respBody = `{"id":"3001","key":"RD-2","self":"self"}`
		case req.URL.Path == "/rest/api/2/issue/3001":
			respBody = `{"id":"3001","key":"RD-2","fields":{"summary":"Task 2","description":"roady-id: t2","status":{"name":"Done","id":"4"}},"self":"self"}`
		default:
			respBody = `{}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	s := &JiraSyncer{
		domain:     "http://jira.test",
		projectKey: "RD",
		email:      "test@example.com",
		apiToken:   "token",
	}

	plan := &planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "t1", Title: "Task 1"},
			{ID: "t2", Title: "Task 2"},
		},
	}
	state := planning.NewExecutionState("p1")

	res, err := s.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if res.StatusUpdates["t1"] != planning.StatusInProgress {
		t.Fatalf("expected t1 status update, got %q", res.StatusUpdates["t1"])
	}
	if res.StatusUpdates["t2"] != planning.StatusDone {
		t.Fatalf("expected t2 status update, got %q", res.StatusUpdates["t2"])
	}
	if _, ok := res.LinkUpdates["t2"]; !ok {
		t.Fatal("expected link update for t2")
	}
}
