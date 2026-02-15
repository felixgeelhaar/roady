package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/fortify/retry"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"gopkg.in/yaml.v3"
)

const RoadyDir = ".roady"
const SpecFile = "spec.yaml"
const SpecLockFile = "spec.lock.json"
const PlanFile = "plan.json"
const PolicyFile = "policy.yaml"
const StateFile = "state.json"
const EventsFile = "events.jsonl"
const UsageFile = "usage.json"
const WebhookFile = "webhooks.yaml"
const DeadLetterFile = "deadletters.jsonl"
const RatesFile = "rates.yaml"
const TimeEntriesFile = "time_entries.yaml"

type FilesystemRepository struct {
	root        string
	retryConfig retry.Config
}

func NewFilesystemRepository(root string) *FilesystemRepository {
	return &FilesystemRepository{
		root: root,
		retryConfig: retry.Config{
			MaxAttempts:   3,
			InitialDelay:  10 * time.Millisecond,
			BackoffPolicy: retry.BackoffExponential,
		},
	}
}

// Root returns the workspace root directory.
func (r *FilesystemRepository) Root() string {
	return r.root
}

// ResolvePath ensures the path is within the .roady directory and prevents traversal.
func (r *FilesystemRepository) ResolvePath(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	// Base directory is strictly root/.roady
	baseDir := filepath.Join(r.root, RoadyDir)
	fullPath := filepath.Join(baseDir, filename)
	cleanPath := filepath.Clean(fullPath)

	// Check for traversal and ensure it's a direct child (no nested subdirs in .roady for now)
	if !strings.HasPrefix(cleanPath, baseDir) || filepath.Dir(cleanPath) != baseDir {
		return "", fmt.Errorf("invalid file path: %s", filename)
	}

	return cleanPath, nil
}

func (r *FilesystemRepository) Initialize() error {
	path := filepath.Join(r.root, RoadyDir)
	// G301: Use 0700 for directories
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("failed to create .roady directory: %w", err)
	}
	return nil
}

func (r *FilesystemRepository) IsInitialized() bool {
	_, err := os.Stat(filepath.Join(r.root, RoadyDir))
	return err == nil
}

func (r *FilesystemRepository) SaveSpec(s *spec.ProductSpec) error {
	path, err := r.ResolvePath(SpecFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	// G306: Use 0600 for files
	return os.WriteFile(path, data, 0600)
}

func (r *FilesystemRepository) LoadSpec() (*spec.ProductSpec, error) {
	retryer := retry.New[*spec.ProductSpec](r.retryConfig)

	return retryer.Do(context.Background(), func(ctx context.Context) (*spec.ProductSpec, error) {
		path, err := r.ResolvePath(SpecFile)
		if err != nil {
			return nil, err
		}

		// #nosec G304 -- Path is resolved and validated via resolvePath
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file: %w", err)
		}

		var s spec.ProductSpec
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
		}

		return &s, nil
	})
}

func (r *FilesystemRepository) SaveSpecLock(s *spec.ProductSpec) error {
	path, err := r.ResolvePath(SpecLockFile)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal spec lock: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func (r *FilesystemRepository) LoadSpecLock() (*spec.ProductSpec, error) {
	path, err := r.ResolvePath(SpecLockFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec lock file: %w", err)
	}

	var s spec.ProductSpec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec lock: %w", err)
	}

	return &s, nil
}

func (r *FilesystemRepository) UpdateUsage(stats domain.UsageStats) error {
	path, err := r.ResolvePath(UsageFile)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal usage stats: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func (r *FilesystemRepository) LoadUsage() (*domain.UsageStats, error) {
	path, err := r.ResolvePath(UsageFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)

	var stats domain.UsageStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage stats: %w", err)
	}

	return &stats, nil
}

// SaveWebhookConfig saves the webhook configuration to .roady/webhooks.yaml.
func (r *FilesystemRepository) SaveWebhookConfig(config *events.WebhookConfig) error {
	path, err := r.ResolvePath(WebhookFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadWebhookConfig loads the webhook configuration from .roady/webhooks.yaml.
func (r *FilesystemRepository) LoadWebhookConfig() (*events.WebhookConfig, error) {
	path, err := r.ResolvePath(WebhookFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook config: %w", err)
	}

	var config events.WebhookConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook config: %w", err)
	}

	return &config, nil
}

// SaveRates saves the rate configuration to .roady/rates.yaml.
func (r *FilesystemRepository) SaveRates(config *billing.RateConfig) error {
	path, err := r.ResolvePath(RatesFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal rates: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadRates loads the rate configuration from .roady/rates.yaml.
func (r *FilesystemRepository) LoadRates() (*billing.RateConfig, error) {
	path, err := r.ResolvePath(RatesFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &billing.RateConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read rates file: %w", err)
	}

	var config billing.RateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rates: %w", err)
	}

	return &config, nil
}

// SaveTimeEntries saves time entries to .roady/time_entries.yaml.
func (r *FilesystemRepository) SaveTimeEntries(entries []billing.TimeEntry) error {
	path, err := r.ResolvePath(TimeEntriesFile)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal time entries: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadTimeEntries loads time entries from .roady/time_entries.yaml.
func (r *FilesystemRepository) LoadTimeEntries() ([]billing.TimeEntry, error) {
	path, err := r.ResolvePath(TimeEntriesFile)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is resolved and validated via ResolvePath
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []billing.TimeEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read time entries file: %w", err)
	}

	var entries []billing.TimeEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal time entries: %w", err)
	}

	return entries, nil
}
