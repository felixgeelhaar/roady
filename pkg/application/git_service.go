package application

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
)

type GitService struct {
	repo    domain.WorkspaceRepository
	taskSvc *TaskService
}

func NewGitService(repo domain.WorkspaceRepository, taskSvc *TaskService) *GitService {
	return &GitService{repo: repo, taskSvc: taskSvc}
}

// SyncMarkers scans the last n commits for [roady:task-id] markers and completes tasks.
func (s *GitService) SyncMarkers(n int) ([]string, error) {
	// Validate input bounds to prevent abuse
	if n < 1 {
		n = 1
	}
	if n > 1000 {
		n = 1000 // Cap at reasonable maximum
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// #nosec G204 -- n is bounds-checked integer, safe for command line
	cmd := exec.CommandContext(ctx, "git", "log", "-n", fmt.Sprintf("%d", n), "--pretty=format:%H|%s")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git log timed out after 30 seconds")
		}
		return nil, fmt.Errorf("failed to read git log: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	results := []string{}

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		hash, message := parts[0], parts[1]

		if strings.Contains(message, "[roady:") {
			start := strings.Index(message, "[roady:") + 7
			end := strings.Index(message[start:], "]")
			if end != -1 {
				taskID := message[start : start+end]

				err := s.taskSvc.TransitionTask(taskID, "complete", "git-automation", "Commit: "+hash)
				if err != nil {
					results = append(results, fmt.Sprintf("Task %s: skip (%v)", taskID, err))
				} else {
					results = append(results, fmt.Sprintf("Task %s: completed via %s", taskID, hash[:8]))
				}
			}
		}
	}

	return results, nil
}
