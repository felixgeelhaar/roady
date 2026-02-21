package application

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/felixgeelhaar/roady/pkg/plugin"
)

// PluginConfigRepository provides access to plugin configurations
type PluginConfigRepository interface {
	LoadPluginConfigs() (*domainPlugin.PluginConfigs, error)
	GetPluginConfig(name string) (*domainPlugin.PluginConfig, error)
	SetPluginConfig(name string, cfg domainPlugin.PluginConfig) error
}

type SyncService struct {
	repo       domain.WorkspaceRepository
	pluginRepo PluginConfigRepository
	taskSvc    *TaskService
}

func NewSyncService(repo domain.WorkspaceRepository, taskSvc *TaskService) *SyncService {
	return &SyncService{repo: repo, taskSvc: taskSvc}
}

// NewSyncServiceWithPlugins creates a SyncService with plugin config support
func NewSyncServiceWithPlugins(repo domain.WorkspaceRepository, pluginRepo PluginConfigRepository, taskSvc *TaskService) *SyncService {
	return &SyncService{repo: repo, pluginRepo: pluginRepo, taskSvc: taskSvc}
}

// SyncWithNamedPlugin syncs using a named plugin configuration from plugins.yaml
func (s *SyncService) SyncWithNamedPlugin(name string) ([]string, error) {
	if s.pluginRepo == nil {
		return nil, fmt.Errorf("plugin configuration not available")
	}

	cfg, err := s.pluginRepo.GetPluginConfig(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin config '%s': %w", name, err)
	}

	return s.SyncWithPluginConfig(cfg.Binary, cfg.Config)
}

// SyncWithPlugin syncs using a plugin binary path (uses empty config, relies on env vars)
func (s *SyncService) SyncWithPlugin(pluginPath string) ([]string, error) {
	return s.SyncWithPluginConfig(pluginPath, map[string]string{})
}

// SyncWithPluginConfig syncs using a plugin binary path with explicit configuration
func (s *SyncService) SyncWithPluginConfig(pluginPath string, config map[string]string) ([]string, error) {
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

	if err := syncer.Init(config); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %w", err)
	}

	result, err := syncer.Sync(plan, state)
	if err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}

	results := []string{}

	// 1. Handle Link Updates
	provider := "external"
	switch {
	case strings.Contains(pluginPath, "linear"):
		provider = "linear"
	case strings.Contains(pluginPath, "jira"):
		provider = "jira"
	case strings.Contains(pluginPath, "github"):
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

// ListPluginConfigs returns all configured plugin names
func (s *SyncService) ListPluginConfigs() ([]string, error) {
	if s.pluginRepo == nil {
		return nil, fmt.Errorf("plugin configuration not available")
	}

	configs, err := s.pluginRepo.LoadPluginConfigs()
	if err != nil {
		return nil, err
	}

	return configs.Names(), nil
}

// GetPluginConfig returns the configuration for a named plugin
func (s *SyncService) GetPluginConfig(name string) (*domainPlugin.PluginConfig, error) {
	if s.pluginRepo == nil {
		return nil, fmt.Errorf("plugin configuration not available")
	}

	return s.pluginRepo.GetPluginConfig(name)
}

// SetPluginConfig saves a plugin configuration
func (s *SyncService) SetPluginConfig(name string, cfg domainPlugin.PluginConfig) error {
	if s.pluginRepo == nil {
		return fmt.Errorf("plugin configuration not available")
	}

	return s.pluginRepo.SetPluginConfig(name, cfg)
}
