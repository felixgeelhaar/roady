package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type fakeTaskActions struct {
	startCalled    *startArgs
	completeCalled *completeArgs
	blockCalled    *blockArgs
	unblockCalled  *unblockArgs

	startErr    error
	completeErr error
	blockErr    error
	unblockErr  error
}

type startArgs struct{ id, owner, rateID string }
type completeArgs struct{ id, evidence string }
type blockArgs struct{ id, reason string }
type unblockArgs struct{ id string }

func (f *fakeTaskActions) StartTask(ctx context.Context, id, owner, rateID string) error {
	f.startCalled = &startArgs{id, owner, rateID}
	return f.startErr
}
func (f *fakeTaskActions) CompleteTask(ctx context.Context, id, evidence string) ([]string, error) {
	f.completeCalled = &completeArgs{id, evidence}
	return nil, f.completeErr
}
func (f *fakeTaskActions) BlockTask(ctx context.Context, id, reason string) error {
	f.blockCalled = &blockArgs{id, reason}
	return f.blockErr
}
func (f *fakeTaskActions) UnblockTask(ctx context.Context, id string) error {
	f.unblockCalled = &unblockArgs{id}
	return f.unblockErr
}

func newActionsServer(t *testing.T, actions TaskActions) *Server {
	t.Helper()
	srv, err := NewServer(":0", &kanbanStubProvider{plan: &planning.Plan{}, state: &planning.ExecutionState{}})
	if err != nil {
		t.Fatal(err)
	}
	if actions != nil {
		srv.EnableTaskActions(actions)
	}
	return srv
}

func postForm(t *testing.T, h http.HandlerFunc, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "/kanban")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestHandleTaskStart_HappyPath(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)

	rec := postForm(t, srv.handleTaskStart, "/actions/task/start", url.Values{"id": {"t-1"}, "owner": {"alice"}})

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", rec.Code)
	}
	if a.startCalled == nil || a.startCalled.id != "t-1" || a.startCalled.owner != "alice" {
		t.Errorf("StartTask call = %+v, want {t-1 alice}", a.startCalled)
	}
	if got := rec.Header().Get("Location"); got != "/kanban" {
		t.Errorf("redirect to %q, want /kanban", got)
	}
}

func TestHandleTaskStart_DefaultOwner(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	postForm(t, srv.handleTaskStart, "/actions/task/start", url.Values{"id": {"t-1"}})
	if a.startCalled.owner != "dashboard" {
		t.Errorf("default owner = %q, want dashboard", a.startCalled.owner)
	}
}

func TestHandleTaskStart_MissingID(t *testing.T) {
	srv := newActionsServer(t, &fakeTaskActions{})
	rec := postForm(t, srv.handleTaskStart, "/actions/task/start", url.Values{})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleTaskComplete_HappyPath(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskComplete, "/actions/task/complete", url.Values{"id": {"t-2"}, "evidence": {"PR #99"}})
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d", rec.Code)
	}
	if a.completeCalled == nil || a.completeCalled.id != "t-2" || a.completeCalled.evidence != "PR #99" {
		t.Errorf("CompleteTask call = %+v", a.completeCalled)
	}
}

func TestHandleTaskBlock_HappyPath(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskBlock, "/actions/task/block", url.Values{"id": {"t-3"}, "reason": {"waiting on legal"}})
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d", rec.Code)
	}
	if a.blockCalled == nil || a.blockCalled.id != "t-3" || a.blockCalled.reason != "waiting on legal" {
		t.Errorf("BlockTask call = %+v", a.blockCalled)
	}
}

func TestHandleTaskUnblock_HappyPath(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskUnblock, "/actions/task/unblock", url.Values{"id": {"t-4"}})
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d", rec.Code)
	}
	if a.unblockCalled == nil || a.unblockCalled.id != "t-4" {
		t.Errorf("UnblockTask call = %+v", a.unblockCalled)
	}
}

func TestHandlers_TaskActionsDisabled(t *testing.T) {
	srv := newActionsServer(t, nil) // no actions wired
	for _, tc := range []struct {
		name string
		h    http.HandlerFunc
		path string
	}{
		{"start", srv.handleTaskStart, "/actions/task/start"},
		{"complete", srv.handleTaskComplete, "/actions/task/complete"},
		{"block", srv.handleTaskBlock, "/actions/task/block"},
		{"unblock", srv.handleTaskUnblock, "/actions/task/unblock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := postForm(t, tc.h, tc.path, url.Values{"id": {"x"}})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want 503 when actions disabled", rec.Code)
			}
		})
	}
}

func TestHandlers_ServiceError(t *testing.T) {
	a := &fakeTaskActions{startErr: errors.New("nope")}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskStart, "/actions/task/start", url.Values{"id": {"t-1"}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 on service error", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nope") {
		t.Errorf("body missing service error: %q", rec.Body.String())
	}
}

func TestHandlers_RedirectFallback(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	req := httptest.NewRequest(http.MethodPost, "/actions/task/start", strings.NewReader(url.Values{"id": {"t-1"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// no Referer header
	rec := httptest.NewRecorder()
	srv.handleTaskStart(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/kanban" {
		t.Errorf("default redirect = %q, want /kanban", got)
	}
}

// Compile-time guard: the Server has the expected method.
var _ = func() {
	var s Server
	s.EnableTaskActions(nil)
}

// pretty-print helper for debug.
var _ = fmt.Stringer(nil)
