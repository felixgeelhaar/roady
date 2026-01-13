package main

import (
	"log"
	"os"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"github.com/hashicorp/go-plugin"
)

type GitHubSyncer struct {
	token string
	repo  string
}

func (s *GitHubSyncer) Init(config map[string]string) error {
	if val, ok := config["token"]; ok {
		s.token = val
	}
	if val, ok := config["repo"]; ok {
		s.repo = val
	}
	// Fallback to env vars if not provided in config
	if s.token == "" {
		s.token = os.Getenv("GITHUB_TOKEN")
	}
	if s.repo == "" {
		s.repo = os.Getenv("GITHUB_REPO")
	}
	return nil
}

func (s *GitHubSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	log.Printf("GitHub Syncer: Syncing %d tasks for repo %s", len(plan.Tasks), s.repo)

	if s.token == "" {
		log.Println("GitHub Syncer: GITHUB_TOKEN not set, skipping API calls.")
		return nil, nil
	}

	// In Horizon 2 maturity, we would:
	// 1. Fetch all issues from GitHub for the repo.
	// 2. Map roady tasks to issues (by title or by a 'roady-id: <id>' tag in the description).
	// 3. Compare GitHub status (open/closed) with roady state.
	// 4. Return updates.

	log.Println("GitHub Syncer: (Stub) No updates fetched from GitHub API yet.")

	updates := make(map[string]planning.TaskStatus)
	return &domainPlugin.SyncResult{StatusUpdates: updates}, nil
}
func main() {
	s := &GitHubSyncer{
		token: os.Getenv("GITHUB_TOKEN"),
		repo:  os.Getenv("GITHUB_REPO"),
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: s},
		},
	})
}
