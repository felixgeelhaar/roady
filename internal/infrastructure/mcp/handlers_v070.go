package mcp

import (
	"context"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/storage"
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
func (s *Server) handleOrgDetectDrift(ctx context.Context, args GetSpecArgs) (any, error) {
	report, err := s.orgSvc.DetectCrossDrift()
	if err != nil {
		return nil, mcpErr("Failed to detect cross-project drift.")
	}
	return report, nil
}

// Plugin handlers

func (s *Server) pluginSvcForPath(projectPath string) *application.PluginService {
	if projectPath == "" || projectPath == s.root {
		return s.pluginSvc
	}
	svc, err := s.servicesForPath(projectPath)
	if err != nil {
		return s.pluginSvc
	}
	return svc.Plugin
}

func (s *Server) handlePluginList(ctx context.Context, args GetSpecArgs) (any, error) {
	plugins, err := s.pluginSvcForPath(args.ProjectPath).ListPlugins()
	if err != nil {
		return nil, mcpErr("Failed to list plugins.")
	}
	return plugins, nil
}

type PluginValidateArgs struct {
	Name        string `json:"name" jsonschema:"description=Name of the plugin to validate"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"description=Path to the roady project directory (default: server root)"`
}

func (s *Server) handlePluginValidate(ctx context.Context, args PluginValidateArgs) (any, error) {
	result, err := s.pluginSvcForPath(args.ProjectPath).ValidatePlugin(args.Name)
	if err != nil {
		return nil, mcpErr("Failed to validate plugin.")
	}
	return result, nil
}

type PluginStatusArgs struct {
	Name        string `json:"name,omitempty" jsonschema:"description=Name of the plugin to check (omit for all)"`
	ProjectPath string `json:"project_path,omitempty" jsonschema:"description=Path to the roady project directory (default: server root)"`
}

func (s *Server) handlePluginStatus(ctx context.Context, args PluginStatusArgs) (any, error) {
	psvc := s.pluginSvcForPath(args.ProjectPath)
	if args.Name != "" {
		result, err := psvc.CheckHealth(args.Name)
		if err != nil {
			return nil, mcpErr("Failed to check plugin health.")
		}
		return result, nil
	}
	results, err := psvc.CheckAllHealth()
	if err != nil {
		return nil, mcpErr("Failed to check plugin health.")
	}
	return results, nil
}

// Messaging handler

func (s *Server) handleMessagingList(ctx context.Context, args GetSpecArgs) (any, error) {
	root := s.root
	if args.ProjectPath != "" {
		root = args.ProjectPath
	}
	repo := storage.NewFilesystemRepository(root)
	cfg, err := repo.LoadMessagingConfig()
	if err != nil {
		return nil, mcpErr("Failed to load messaging config.")
	}
	return cfg, nil
}
