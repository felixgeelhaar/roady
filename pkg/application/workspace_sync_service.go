package application

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// WorkspaceSyncService handles git-based synchronization of the .roady/ directory.
type WorkspaceSyncService struct {
	root  string
	audit domain.AuditLogger
}

// SyncResult holds the outcome of a push or pull operation.
type SyncResult struct {
	Action   string   `json:"action"`
	Files    []string `json:"files,omitempty"`
	Conflict bool     `json:"conflict"`
	Message  string   `json:"message"`
}

func NewWorkspaceSyncService(root string, audit domain.AuditLogger) *WorkspaceSyncService {
	return &WorkspaceSyncService{root: root, audit: audit}
}

// Push stages and commits .roady/ changes, then pushes to the remote.
func (s *WorkspaceSyncService) Push(ctx context.Context) (*SyncResult, error) {
	// Check for changes in .roady/
	changed, err := s.changedFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("check changes: %w", err)
	}
	if len(changed) == 0 {
		return &SyncResult{Action: "push", Message: "No changes to push"}, nil
	}

	// Stage .roady/ files
	if err := s.git(ctx, "add", storage.RoadyDir+"/"); err != nil {
		return nil, fmt.Errorf("git add: %w", err)
	}

	// Commit
	msg := fmt.Sprintf("roady: sync workspace state [%s]", time.Now().Format("2006-01-02T15:04:05"))
	if err := s.git(ctx, "commit", "-m", msg); err != nil {
		return nil, fmt.Errorf("git commit: %w", err)
	}

	// Push
	if err := s.git(ctx, "push"); err != nil {
		return nil, fmt.Errorf("git push: %w", err)
	}

	_ = s.audit.Log("workspace.push", "cli", map[string]interface{}{
		"files": changed,
	})

	return &SyncResult{
		Action:  "push",
		Files:   changed,
		Message: fmt.Sprintf("Pushed %d file(s)", len(changed)),
	}, nil
}

// Pull fetches remote changes and merges .roady/ files.
// Returns conflict info if merge fails.
func (s *WorkspaceSyncService) Pull(ctx context.Context) (*SyncResult, error) {
	// Stash any local .roady/ changes
	hasChanges := len(s.mustChangedFiles(ctx)) > 0
	if hasChanges {
		_ = s.git(ctx, "stash", "push", "-m", "roady-sync-stash", "--", storage.RoadyDir+"/")
	}

	// Pull
	pullErr := s.git(ctx, "pull", "--rebase")

	// Pop stash if we stashed
	if hasChanges {
		popErr := s.git(ctx, "stash", "pop")
		if popErr != nil {
			// Merge conflict
			conflictFiles := s.mustChangedFiles(ctx)
			_ = s.audit.Log("workspace.pull_conflict", "cli", map[string]interface{}{
				"files": conflictFiles,
			})
			return &SyncResult{
				Action:   "pull",
				Files:    conflictFiles,
				Conflict: true,
				Message:  "Merge conflict in .roady/ files. Resolve manually, then run 'roady workspace push'.",
			}, nil
		}
	}

	if pullErr != nil {
		return nil, fmt.Errorf("git pull: %w", pullErr)
	}

	_ = s.audit.Log("workspace.pull", "cli", nil)

	return &SyncResult{
		Action:  "pull",
		Message: "Pulled latest workspace state",
	}, nil
}

func (s *WorkspaceSyncService) git(ctx context.Context, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = s.root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *WorkspaceSyncService) changedFiles(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", storage.RoadyDir+"/")
	cmd.Dir = s.root
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				files = append(files, parts[len(parts)-1])
			}
		}
	}
	return files, nil
}

func (s *WorkspaceSyncService) mustChangedFiles(ctx context.Context) []string {
	files, _ := s.changedFiles(ctx)
	return files
}
