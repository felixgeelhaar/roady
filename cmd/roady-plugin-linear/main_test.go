package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

func TestLinearSyncer_InitMissing(t *testing.T) {
	s := &LinearSyncer{}
	if err := s.Init(map[string]string{}); err == nil {
		t.Fatal("expected error for missing Linear config")
	}
}

func TestLinearSyncer_InitConfig(t *testing.T) {
	s := &LinearSyncer{}
	cfg := map[string]string{
		"api_key": "api-123",
		"team_id": "team-123",
	}
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if s.apiKey != "api-123" || s.teamID != "team-123" {
		t.Fatalf("unexpected config: apiKey=%q teamID=%q", s.apiKey, s.teamID)
	}
}

func TestLinearHelpers(t *testing.T) {
	id := extractRoadyID("hello\nroady-id: task-456")
	if id != "task-456" {
		t.Fatalf("expected roady id, got %q", id)
	}

	if got := mapLinearStatus("completed", "Done"); got != "done" {
		t.Fatalf("expected done, got %q", got)
	}
	if got := mapLinearStatus("started", "In Progress"); got != "in_progress" {
		t.Fatalf("expected in_progress, got %q", got)
	}
	if got := mapLinearStatus("canceled", "Canceled"); got != "blocked" {
		t.Fatalf("expected blocked, got %q", got)
	}
	if got := mapLinearStatus("backlog", "Backlog"); got != "pending" {
		t.Fatalf("expected pending, got %q", got)
	}
	if got := mapLinearStatus("unknown", "Unknown"); got != "pending" {
		t.Fatalf("expected pending, got %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLinearSyncer_SyncCreatesIssues(t *testing.T) {
	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		payload := string(body)

		var respBody string
		switch {
		case bytes.Contains([]byte(payload), []byte("issueCreate")):
			respBody = `{"data":{"issueCreate":{"success":true,"issue":{"id":"L2","identifier":"LIN-2","title":"Task 2","description":"roady-id: t2","state":{"name":"Done","type":"completed"},"url":"https://linear.app/issue/LIN-2"}}}}`
		case bytes.Contains([]byte(payload), []byte("team")):
			respBody = `{"data":{"team":{"issues":{"nodes":[{"id":"L1","identifier":"LIN-1","title":"Task 1","description":"roady-id: t1","state":{"name":"In Progress","type":"started"},"url":"https://linear.app/issue/LIN-1"}]}}}}`
		default:
			respBody = `{"data":{}}`
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	s := &LinearSyncer{
		apiKey: "token",
		teamID: "team",
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

func TestLinearSyncer_Push(t *testing.T) {
	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	var updateCalled bool
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		payload := string(body)

		var respBody string
		switch {
		case bytes.Contains([]byte(payload), []byte("issueUpdate")):
			updateCalled = true
			respBody = `{"data":{"issueUpdate":{"success":true}}}`
		case bytes.Contains([]byte(payload), []byte("states")):
			respBody = `{"data":{"team":{"states":{"nodes":[{"id":"state-1","name":"Done","type":"completed"},{"id":"state-2","name":"In Progress","type":"started"},{"id":"state-3","name":"Todo","type":"unstarted"}]}}}}`
		case bytes.Contains([]byte(payload), []byte("team")):
			respBody = `{"data":{"team":{"issues":{"nodes":[{"id":"L1","identifier":"LIN-1","title":"Task 1","description":"roady-id: t1","state":{"name":"In Progress","type":"started"},"url":"https://linear.app/LIN-1"}]}}}}`
		default:
			respBody = `{"data":{}}`
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	s := &LinearSyncer{apiKey: "token", teamID: "team"}

	err := s.Push("t1", planning.StatusDone)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !updateCalled {
		t.Error("expected issueUpdate to be called")
	}
}

func TestLinearSyncer_Push_NotFound(t *testing.T) {
	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		respBody := `{"data":{"team":{"issues":{"nodes":[]}}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	s := &LinearSyncer{apiKey: "token", teamID: "team"}
	err := s.Push("nonexistent", planning.StatusDone)
	if err == nil {
		t.Error("expected error for issue not found")
	}
}

func TestLinearSyncer_Sync_WithExistingRef(t *testing.T) {
	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		respBody := `{"data":{"team":{"issues":{"nodes":[{"id":"L1","identifier":"LIN-1","title":"Task 1","description":"roady-id: t1","state":{"name":"Done","type":"completed"},"url":"https://linear.app/LIN-1"}]}}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}, nil
	})

	s := &LinearSyncer{apiKey: "token", teamID: "team"}

	plan := &planning.Plan{
		ID:    "p1",
		Tasks: []planning.Task{{ID: "t1", Title: "Task 1"}},
	}
	state := planning.NewExecutionState("p1")
	state.TaskStates["t1"] = planning.TaskResult{
		Status: planning.StatusPending,
		ExternalRefs: map[string]planning.ExternalRef{
			"linear": {ID: "L1"},
		},
	}

	res, err := s.Sync(plan, state)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if res.StatusUpdates["t1"] != planning.StatusDone {
		t.Errorf("expected t1 status done, got %q", res.StatusUpdates["t1"])
	}
}

func TestLinearSyncer_InitEnvFallback(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "env-key")
	t.Setenv("LINEAR_TEAM_ID", "env-team")

	s := &LinearSyncer{}
	if err := s.Init(map[string]string{}); err != nil {
		t.Fatalf("Init with env vars failed: %v", err)
	}
	if s.apiKey != "env-key" {
		t.Errorf("expected env api key, got %q", s.apiKey)
	}
	if s.teamID != "env-team" {
		t.Errorf("expected env team id, got %q", s.teamID)
	}
}

func TestLinearSyncer_InitMissingTeamID(t *testing.T) {
	s := &LinearSyncer{}
	err := s.Init(map[string]string{"api_key": "key"})
	if err == nil {
		t.Error("expected error for missing team_id")
	}
}
