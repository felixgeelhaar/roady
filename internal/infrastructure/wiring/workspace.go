package wiring

import (
	"path/filepath"

	webhook "github.com/felixgeelhaar/roady/internal/infrastructure/webhook"
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// Workspace bundles core infrastructure dependencies.
type Workspace struct {
	Repo     *storage.FilesystemRepository
	Audit    *application.AuditService
	Usage    *application.UsageService
	Notifier *webhook.Notifier
}

func NewWorkspace(root string) *Workspace {
	repo := storage.NewFilesystemRepository(root)

	// Load webhook config and create notifier if configured
	var notifier *webhook.Notifier
	if config, err := repo.LoadWebhookConfig(); err == nil && len(config.Webhooks) > 0 {
		dlPath := filepath.Join(root, storage.RoadyDir, storage.DeadLetterFile)
		dlStore := webhook.NewDeadLetterStore(dlPath)
		notifier = webhook.NewNotifier(config.Webhooks, dlStore)
	}

	return &Workspace{
		Repo:     repo,
		Audit:    application.NewAuditService(repo),
		Usage:    application.NewUsageService(repo),
		Notifier: notifier,
	}
}
