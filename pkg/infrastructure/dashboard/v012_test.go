package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// --- Reopen handler ----------------------------------------------------------

func TestHandleTaskReopen_HappyPath(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskReopen, "/actions/task/reopen", url.Values{"id": {"r-1"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if a.reopenCalled == nil || a.reopenCalled.id != "r-1" {
		t.Errorf("ReopenTask call = %+v, want {r-1}", a.reopenCalled)
	}
}

func TestHandleTaskReopen_Disabled(t *testing.T) {
	srv := newActionsServer(t, nil)
	rec := postForm(t, srv.handleTaskReopen, "/actions/task/reopen", url.Values{"id": {"x"}})
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

// --- Org actions routing ----------------------------------------------------

type stubOrgActions struct {
	called  atomic.Int32
	gotPath string
	gotProj string
	out     TaskActions
	err     error
}

func (s *stubOrgActions) ResolveTaskActions(path, proj string) (TaskActions, error) {
	s.called.Add(1)
	s.gotPath = path
	s.gotProj = proj
	return s.out, s.err
}

func TestPickActions_FallsBackToDefault(t *testing.T) {
	defaultActions := &fakeTaskActions{}
	srv := newActionsServer(t, defaultActions)
	srv.EnableOrgTaskActions(&stubOrgActions{out: &fakeTaskActions{}})

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("id=t"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	got, err := srv.pickActions(req)
	if err != nil {
		t.Fatal(err)
	}
	if got != TaskActions(defaultActions) {
		t.Errorf("got != default; got=%T", got)
	}
}

func TestPickActions_RoutesToOrgWhenProjectSet(t *testing.T) {
	defaultActions := &fakeTaskActions{}
	orgActions := &fakeTaskActions{}
	stub := &stubOrgActions{out: orgActions}

	srv := newActionsServer(t, defaultActions)
	srv.EnableOrgTaskActions(stub)

	req := httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader("id=t&project_path=/repo&project=auth"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	got, err := srv.pickActions(req)
	if err != nil {
		t.Fatal(err)
	}
	if got != TaskActions(orgActions) {
		t.Errorf("got != orgActions; got=%T", got)
	}
	if stub.called.Load() != 1 {
		t.Errorf("resolver called %d times, want 1", stub.called.Load())
	}
	if stub.gotPath != "/repo" || stub.gotProj != "auth" {
		t.Errorf("resolver got (%q, %q), want (/repo, auth)", stub.gotPath, stub.gotProj)
	}
}

func TestPickActions_ResolverErrorBubbles(t *testing.T) {
	srv := newActionsServer(t, nil)
	srv.EnableOrgTaskActions(&stubOrgActions{err: errors.New("boom")})

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("project=auth"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	if _, err := srv.pickActions(req); err == nil {
		t.Error("expected error from resolver")
	}
}

func TestHandleTaskStart_RoutedViaOrgResolver(t *testing.T) {
	orgActions := &fakeTaskActions{}
	stub := &stubOrgActions{out: orgActions}
	srv := newActionsServer(t, &fakeTaskActions{})
	srv.EnableOrgTaskActions(stub)

	rec := postForm(t, srv.handleTaskStart, "/actions/task/start",
		url.Values{"id": {"t-9"}, "project_path": {"/repo"}, "project": {"auth"}})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rec.Code)
	}
	if orgActions.startCalled == nil || orgActions.startCalled.id != "t-9" {
		t.Errorf("expected start on org-routed actions; got %+v", orgActions.startCalled)
	}
	if stub.called.Load() == 0 {
		t.Error("resolver never called")
	}
}

// --- Auth middleware --------------------------------------------------------

func newAuthedServer(t *testing.T, token string) *Server {
	t.Helper()
	srv, err := NewServer(":0", &kanbanStubProvider{plan: &planning.Plan{}, state: &planning.ExecutionState{}})
	if err != nil {
		t.Fatal(err)
	}
	srv.EnableAuthToken(token)
	return srv
}

func TestAuthMiddleware_NoTokenIsPublic(t *testing.T) {
	srv := newAuthedServer(t, "")
	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	if srv.authToken != "" {
		handler = authMiddleware(srv.authToken, handler)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != 200 {
		t.Errorf("public path: got %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_RejectsNoAuth(t *testing.T) {
	h := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestAuthMiddleware_AcceptsBearer(t *testing.T) {
	h := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("got %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_AcceptsCookie(t *testing.T) {
	h := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "secret"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("got %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_QueryHandshakeSetsCookieAndRedirects(t *testing.T) {
	h := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kanban?token=secret&other=keep", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("got %d, want 303", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc == "" || strings.Contains(loc, "token=") {
		t.Errorf("redirect should strip token; got %q", loc)
	}
	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == authCookieName && c.Value == "secret" {
			found = true
		}
	}
	if !found {
		t.Error("expected cookie to be set")
	}
}

func TestAuthMiddleware_BadTokenRejected(t *testing.T) {
	h := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rec.Code)
	}
}

// --- SSE hub ----------------------------------------------------------------

func TestSSEHub_BroadcastDeliversToSubscribers(t *testing.T) {
	h := newSSEHub()
	a := h.subscribe()
	b := h.subscribe()

	go h.broadcast("task-changed")

	for _, ch := range []chan string{a, b} {
		select {
		case msg := <-ch:
			if msg != "task-changed" {
				t.Errorf("got %q, want task-changed", msg)
			}
		case <-time.After(time.Second):
			t.Fatal("subscriber timed out")
		}
	}
}

func TestSSEHub_UnsubscribeRemovesClient(t *testing.T) {
	h := newSSEHub()
	a := h.subscribe()
	h.unsubscribe(a)
	// channel should be closed
	select {
	case _, ok := <-a:
		if ok {
			t.Error("expected channel closed on unsubscribe")
		}
	case <-time.After(time.Second):
		t.Error("unsubscribe did not close channel")
	}
}

func TestBroadcastChange_IsNilSafe(t *testing.T) {
	srv, err := NewServer(":0", &kanbanStubProvider{plan: &planning.Plan{}, state: &planning.ExecutionState{}})
	if err != nil {
		t.Fatal(err)
	}
	// sse is nil before Start(); should not panic.
	srv.broadcastChange()
}

// --- Reopen on coordinator round-trip --------------------------------------

func TestReopenTask_PropagatesThroughActions(t *testing.T) {
	a := &fakeTaskActions{}
	srv := newActionsServer(t, a)
	rec := postForm(t, srv.handleTaskReopen, "/actions/task/reopen", url.Values{"id": {"t-1"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("got %d, want 303", rec.Code)
	}
	if a.reopenCalled == nil {
		t.Fatal("ReopenTask not invoked")
	}
}

// Ensure dashboard.TaskActions interface stays compatible with the real
// application.TaskService implementation we depend on.
var _ TaskActions = (interface {
	StartTask(ctx context.Context, taskID, owner, rateID string) error
	CompleteTask(ctx context.Context, taskID, evidence string) ([]string, error)
	BlockTask(ctx context.Context, taskID, reason string) error
	UnblockTask(ctx context.Context, taskID string) error
	ReopenTask(ctx context.Context, taskID string) error
})(nil)
