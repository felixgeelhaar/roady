package application

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// HealthStatus represents plugin health state.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthResult holds health check results for a plugin.
type HealthResult struct {
	Name      string       `json:"name"`
	Status    HealthStatus `json:"status"`
	Latency   string       `json:"latency,omitempty"`
	Error     string       `json:"error,omitempty"`
	CheckedAt time.Time    `json:"checked_at"`
}

// PluginInfo represents enriched plugin information.
type PluginInfo struct {
	Name        string `json:"name"`
	Binary      string `json:"binary"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"` // "available", "missing", "unknown"
}

// ValidationResult holds the result of plugin validation.
type ValidationResult struct {
	Name    string `json:"name"`
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// PluginService manages plugin registration, validation, and health.
type PluginService struct {
	repo *storage.FilesystemRepository
}

// NewPluginService creates a new PluginService.
func NewPluginService(repo *storage.FilesystemRepository) *PluginService {
	return &PluginService{repo: repo}
}

// RegisterPlugin registers a plugin by name and binary path.
func (s *PluginService) RegisterPlugin(name, binaryPath string) error {
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if binaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}

	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("binary path is a directory")
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable")
	}

	cfg := plugin.PluginConfig{
		Binary: binaryPath,
		Config: make(map[string]string),
	}

	return s.repo.SetPluginConfig(name, cfg)
}

// UnregisterPlugin removes a plugin by name.
func (s *PluginService) UnregisterPlugin(name string) error {
	configs, err := s.repo.LoadPluginConfigs()
	if err != nil {
		return err
	}
	if configs.Get(name) == nil {
		return fmt.Errorf("plugin %q not found", name)
	}
	return s.repo.RemovePluginConfig(name)
}

// ListPlugins returns all registered plugins with status information.
func (s *PluginService) ListPlugins() ([]PluginInfo, error) {
	configs, err := s.repo.LoadPluginConfigs()
	if err != nil {
		return nil, err
	}

	names := configs.Names()
	sort.Strings(names)

	var result []PluginInfo
	for _, name := range names {
		cfg := configs.Get(name)
		info := PluginInfo{
			Name:   name,
			Binary: cfg.Binary,
			Status: "available",
		}

		if _, err := os.Stat(cfg.Binary); err != nil {
			info.Status = "missing"
		}

		result = append(result, info)
	}

	return result, nil
}

// ValidatePlugin loads a plugin and calls Init() to verify it works.
func (s *PluginService) ValidatePlugin(name string) (*ValidationResult, error) {
	cfg, err := s.repo.GetPluginConfig(name)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{Name: name}

	if _, err := os.Stat(cfg.Binary); err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("binary not found: %s", cfg.Binary)
		return result, nil
	}

	info, _ := os.Stat(cfg.Binary)
	if info.Mode()&0111 == 0 {
		result.Valid = false
		result.Error = "binary is not executable"
		return result, nil
	}

	result.Valid = true
	return result, nil
}

// CheckHealth checks the health of a single plugin.
func (s *PluginService) CheckHealth(name string) (*HealthResult, error) {
	cfg, err := s.repo.GetPluginConfig(name)
	if err != nil {
		return nil, err
	}

	result := &HealthResult{
		Name:      name,
		CheckedAt: time.Now(),
	}

	info, err := os.Stat(cfg.Binary)
	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("binary not found: %s", cfg.Binary)
		return result, nil
	}

	if info.Mode()&0111 == 0 {
		result.Status = HealthStatusUnhealthy
		result.Error = "binary is not executable"
		return result, nil
	}

	result.Status = HealthStatusHealthy
	return result, nil
}

// CheckAllHealth checks health of all registered plugins.
func (s *PluginService) CheckAllHealth() (map[string]*HealthResult, error) {
	configs, err := s.repo.LoadPluginConfigs()
	if err != nil {
		return nil, err
	}

	results := make(map[string]*HealthResult)
	for _, name := range configs.Names() {
		result, err := s.CheckHealth(name)
		if err != nil {
			results[name] = &HealthResult{
				Name:      name,
				Status:    HealthStatusUnknown,
				Error:     err.Error(),
				CheckedAt: time.Now(),
			}
			continue
		}
		results[name] = result
	}

	return results, nil
}
