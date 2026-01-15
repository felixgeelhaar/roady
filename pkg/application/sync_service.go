package application

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/plugin"
)

type SyncService struct {
	repo    domain.WorkspaceRepository
	taskSvc *TaskService
}

func NewSyncService(repo domain.WorkspaceRepository, taskSvc *TaskService) *SyncService {
	return &SyncService{repo: repo, taskSvc: taskSvc}
}

func (s *SyncService) SyncWithPlugin(pluginPath string) ([]string, error) {
	loader := plugin.NewLoader()
	defer loader.Cleanup()

	syncer, err := loader.Load(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	plan, err := s.repo.LoadPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	state, err := s.repo.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	if err := syncer.Init(map[string]string{}); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %w", err)
	}

	result, err := syncer.Sync(plan, state)
	if err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}

	results := []string{}

	// 1. Handle Link Updates
	provider := "external"
	if strings.Contains(pluginPath, "linear") {
		provider = "linear"
	} else if strings.Contains(pluginPath, "jira") {
		provider = "jira"
	} else if strings.Contains(pluginPath, "github") {
		provider = "github"
	}

	for id, ref := range result.LinkUpdates {
		if err := s.taskSvc.LinkTask(id, provider, ref); err != nil {
			results = append(results, fmt.Sprintf("Link Task %s: error (%v)", id, err))
		} else {
			results = append(results, fmt.Sprintf("Link Task %s: linked to %s (%s)", id, provider, ref.Identifier))
		}
	}

	// 2. Handle Status Updates
	for id, status := range result.StatusUpdates {
		var event string
		switch status {
		case "done":
			event = "complete"
		case "in_progress":
			event = "start"
		}

		if event != "" {
			if err := s.taskSvc.TransitionTask(id, event, "sync-plugin", ""); err != nil {
				results = append(results, fmt.Sprintf("Status Task %s: skip (%v)", id, err))
			} else {
				results = append(results, fmt.Sprintf("Status Task %s: %s", id, status))
			}
		}
	}

	for _, e := range result.Errors {
		results = append(results, fmt.Sprintf("Plugin Error: %s", e))
	}

	return results, nil
}
