package mcp

import (
	"context"

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
func (s *Server) handleOrgDetectDrift(ctx context.Context, args struct{}) (any, error) {
	report, err := s.orgSvc.DetectCrossDrift()
	if err != nil {
		return nil, mcpErr("Failed to detect cross-project drift.")
	}
	return report, nil
}

// Plugin handlers

func (s *Server) handlePluginList(ctx context.Context, args struct{}) (any, error) {
	plugins, err := s.pluginSvc.ListPlugins()
	if err != nil {
		return nil, mcpErr("Failed to list plugins.")
	}
	return plugins, nil
}

type PluginValidateArgs struct {
	Name string `json:"name" jsonschema:"description=Name of the plugin to validate"`
}

func (s *Server) handlePluginValidate(ctx context.Context, args PluginValidateArgs) (any, error) {
	result, err := s.pluginSvc.ValidatePlugin(args.Name)
	if err != nil {
		return nil, mcpErr("Failed to validate plugin.")
	}
	return result, nil
}

type PluginStatusArgs struct {
	Name string `json:"name,omitempty" jsonschema:"description=Name of the plugin to check (omit for all)"`
}

func (s *Server) handlePluginStatus(ctx context.Context, args PluginStatusArgs) (any, error) {
	if args.Name != "" {
		result, err := s.pluginSvc.CheckHealth(args.Name)
		if err != nil {
			return nil, mcpErr("Failed to check plugin health.")
		}
		return result, nil
	}
	results, err := s.pluginSvc.CheckAllHealth()
	if err != nil {
		return nil, mcpErr("Failed to check plugin health.")
	}
	return results, nil
}

// Messaging handler

func (s *Server) handleMessagingList(ctx context.Context, args struct{}) (any, error) {
	repo := storage.NewFilesystemRepository(s.root)
	cfg, err := repo.LoadMessagingConfig()
	if err != nil {
		return nil, mcpErr("Failed to load messaging config.")
	}
	return cfg, nil
}
