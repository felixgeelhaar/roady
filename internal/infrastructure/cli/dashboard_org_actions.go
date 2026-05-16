package cli

import (
	"fmt"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
	"github.com/felixgeelhaar/roady/pkg/infrastructure/dashboard"
)

// orgTaskActionsResolver implements dashboard.OrgTaskActions by building
// AppServices for the requested (projectPath, project) pair on demand. The
// underlying wiring.BuildAppServicesForProject is cached internally.
type orgTaskActionsResolver struct {
	defaultPath string
}

func newOrgTaskActionsResolver(defaultPath string) *orgTaskActionsResolver {
	return &orgTaskActionsResolver{defaultPath: defaultPath}
}

func (r *orgTaskActionsResolver) ResolveTaskActions(projectPath, project string) (dashboard.TaskActions, error) {
	root := projectPath
	if root == "" {
		root = r.defaultPath
	}
	svc, err := wiring.BuildAppServicesForProject(root, project)
	if err != nil && svc == nil {
		return nil, fmt.Errorf("build services for %s / %s: %w", root, project, err)
	}
	return svc.Task, nil
}
