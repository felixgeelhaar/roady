package mcp

import (
	"context"
)

// Org policy handler
type OrgPolicyArgs struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"description=Path to the project (default: current directory)"`
}

func (s *Server) handleOrgPolicy(ctx context.Context, args OrgPolicyArgs) (any, error) {
	projectPath := s.root
	if args.ProjectPath != "" {
		projectPath = args.ProjectPath
	}

	merged, err := s.orgSvc.LoadMergedPolicy(projectPath)
	if err != nil {
		return nil, mcpErr("Failed to load merged policy.")
	}
	return merged, nil
}

// Org cross-drift handler
func (s *Server) handleOrgDetectDrift(ctx context.Context, args struct{}) (any, error) {
	report, err := s.orgSvc.DetectCrossDrift()
	if err != nil {
		return nil, mcpErr("Failed to detect cross-project drift.")
	}
	return report, nil
}
