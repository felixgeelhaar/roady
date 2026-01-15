package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"github.com/google/go-github/v69/github"
	"github.com/hashicorp/go-plugin"
	"golang.org/x/oauth2"
)

type GitHubSyncer struct {
	client *github.Client
	token  string
	repo   string
	owner  string
	name   string
}

func (s *GitHubSyncer) Init(config map[string]string) error {
	s.token = config["token"]
	s.repo = config["repo"]

	// Fallback to env vars
	if s.token == "" {
		s.token = os.Getenv("GITHUB_TOKEN")
	}
	if s.repo == "" {
		s.repo = os.Getenv("GITHUB_REPO")
	}

	if s.token == "" {
		return fmt.Errorf("github token is required (config 'token' or env GITHUB_TOKEN)")
	}
	if s.repo == "" {
		return fmt.Errorf("github repo is required (config 'repo' or env GITHUB_REPO)")
	}

	parts := strings.Split(s.repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repo must be in format 'owner/name', got: %s", s.repo)
	}
	s.owner = parts[0]
	s.name = parts[1]

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: s.token},
	)
	tc := oauth2.NewClient(ctx, ts)
	s.client = github.NewClient(tc)

	return nil
}

// fetchAllIssues retrieves all issues from the repository with pagination.
func (s *GitHubSyncer) fetchAllIssues(ctx context.Context) ([]*github.Issue, error) {
	var allIssues []*github.Issue
	opts := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		issues, resp, err := s.client.Issues.ListByRepo(ctx, s.owner, s.name, opts)
		if err != nil {
			return nil, fmt.Errorf("list issues: %w", err)
		}
		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

func (s *GitHubSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	ctx := context.Background()
	log.Printf("GitHub Syncer: Syncing %d tasks for repo %s/%s", len(plan.Tasks), s.owner, s.name)

	// Fetch all issues with pagination
	issues, err := s.fetchAllIssues(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Index issues by title for simple matching
	// In a real world, we'd use a robust ID correlation (e.g. metadata in issue body).
	issueMap := make(map[string]*github.Issue)
	for _, issue := range issues {
		// Skip pull requests
		if issue.IsPullRequest() {
			continue
		}
		issueMap[issue.GetTitle()] = issue
	}

	updates := make(map[string]planning.TaskStatus)
	linkUpdates := make(map[string]planning.ExternalRef)
	var errors []string

	for _, task := range plan.Tasks {
		issue, exists := issueMap[task.Title]

		if !exists {
			// Option: Create issue if it doesn't exist?
			// For this MVP, we only read state to avoid spamming repos unexpectedly.
			continue
		}

		// Link update
		linkUpdates[task.ID] = planning.ExternalRef{
			ID:           fmt.Sprintf("%d", issue.GetNumber()),
			Identifier:   fmt.Sprintf("#%d", issue.GetNumber()),
			URL:          issue.GetHTMLURL(),
			LastSyncedAt: time.Now(),
		}

		// Status update
		ghState := issue.GetState()
		var roadyStatus planning.TaskStatus

		if ghState == "closed" {
			roadyStatus = planning.StatusDone
		} else {
			// Map open to InProgress if assigned, else Todo?
			// For now, let's just say if it's open, verify it's not Done in Roady.
			// Ideally we map: Open -> Todo/InProgress, Closed -> Done
			if issue.GetAssignee() != nil {
				roadyStatus = planning.StatusInProgress
			} else {
				roadyStatus = planning.StatusPending
			}
		}

		// Only propose update if different
		// (The caller will handle the diff, we just report what the external world says)
		updates[task.ID] = roadyStatus
	}

	return &domainPlugin.SyncResult{
		StatusUpdates: updates,
		LinkUpdates:   linkUpdates,
		Errors:        errors,
	}, nil
}

func main() {
	// Plugin serving
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &GitHubSyncer{}},
		},
	})
}
