package wiring

import (
	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// Workspace bundles core infrastructure dependencies.
type Workspace struct {
	Repo  *storage.FilesystemRepository
	Audit *application.AuditService
}

func NewWorkspace(root string) *Workspace {
	repo := storage.NewFilesystemRepository(root)
	return &Workspace{
		Repo:  repo,
		Audit: application.NewAuditService(repo),
	}
}
