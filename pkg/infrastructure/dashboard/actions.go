package dashboard

import (
	"context"
	"net/http"
)

// TaskActions exposes the subset of TaskService methods the Kanban board
// needs to mutate task state. The dashboard server takes this as an optional
// dependency; when nil, the action endpoints stay unregistered and the board
// is read-only.
type TaskActions interface {
	StartTask(ctx context.Context, taskID, owner, rateID string) error
	CompleteTask(ctx context.Context, taskID, evidence string) ([]string, error)
	BlockTask(ctx context.Context, taskID, reason string) error
	UnblockTask(ctx context.Context, taskID string) error
}

// EnableTaskActions wires POST handlers that mutate task state from the
// dashboard (Kanban card buttons). Pass nil to keep the board read-only.
func (s *Server) EnableTaskActions(svc TaskActions) {
	s.taskActions = svc
}

// redirectAfterAction sends the user back to the page they came from
// (Referer when set, /kanban as a sane default).
func redirectAfterAction(w http.ResponseWriter, r *http.Request) {
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/kanban"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (s *Server) handleTaskStart(w http.ResponseWriter, r *http.Request) {
	if s.taskActions == nil {
		http.Error(w, "task actions not enabled", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.PostForm.Get("id")
	if id == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	owner := strDefault(r.PostForm.Get("owner"), "dashboard")
	if err := s.taskActions.StartTask(r.Context(), id, owner, ""); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirectAfterAction(w, r)
}

func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	if s.taskActions == nil {
		http.Error(w, "task actions not enabled", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.PostForm.Get("id")
	if id == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	evidence := strDefault(r.PostForm.Get("evidence"), "completed via dashboard")
	if _, err := s.taskActions.CompleteTask(r.Context(), id, evidence); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirectAfterAction(w, r)
}

func (s *Server) handleTaskBlock(w http.ResponseWriter, r *http.Request) {
	if s.taskActions == nil {
		http.Error(w, "task actions not enabled", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.PostForm.Get("id")
	if id == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	reason := strDefault(r.PostForm.Get("reason"), "blocked via dashboard")
	if err := s.taskActions.BlockTask(r.Context(), id, reason); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirectAfterAction(w, r)
}

func (s *Server) handleTaskUnblock(w http.ResponseWriter, r *http.Request) {
	if s.taskActions == nil {
		http.Error(w, "task actions not enabled", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.PostForm.Get("id")
	if id == "" {
		http.Error(w, "missing task id", http.StatusBadRequest)
		return
	}
	if err := s.taskActions.UnblockTask(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirectAfterAction(w, r)
}

func strDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
