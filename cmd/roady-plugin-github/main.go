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
	goplugin "github.com/hashicorp/go-plugin"
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

	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// Fetch all issues with pagination
	issues, err := s.fetchAllIssues(ctx)
	if err != nil {
		return nil, err
	}

	// Index issues by roady-id marker in body (preferred) and title (fallback)
	issueByRoadyID := make(map[string]*github.Issue)
	issueByTitle := make(map[string]*github.Issue)
	for _, issue := range issues {
		// Skip pull requests
		if issue.IsPullRequest() {
			continue
		}
		// Check for roady-id marker in body
		if rid := extractRoadyID(issue.GetBody()); rid != "" {
			issueByRoadyID[rid] = issue
		}
		issueByTitle[issue.GetTitle()] = issue
	}

	for _, task := range plan.Tasks {
		var targetIssue *github.Issue

		// 1. Check state refs first (already linked)
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["github"]; ok {
				for _, issue := range issues {
					if issue.IsPullRequest() {
						continue
					}
					if fmt.Sprintf("%d", issue.GetNumber()) == ref.ID {
						targetIssue = issue
						break
					}
				}
			}
		}

		// 2. Match by roady-id marker in issue body
		if targetIssue == nil {
			if issue, ok := issueByRoadyID[task.ID]; ok {
				targetIssue = issue
			}
		}

		// 3. Fall back to title matching
		if targetIssue == nil {
			if issue, ok := issueByTitle[task.Title]; ok {
				targetIssue = issue
			}
		}

		// 4. Create issue if not found
		if targetIssue == nil {
			issue, err := s.createIssue(ctx, task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create issue for %s: %v", task.ID, err))
				continue
			}
			targetIssue = issue
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           fmt.Sprintf("%d", issue.GetNumber()),
				Identifier:   fmt.Sprintf("#%d", issue.GetNumber()),
				URL:          issue.GetHTMLURL(),
				LastSyncedAt: time.Now(),
			}
		} else {
			// Update link for existing issue
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           fmt.Sprintf("%d", targetIssue.GetNumber()),
				Identifier:   fmt.Sprintf("#%d", targetIssue.GetNumber()),
				URL:          targetIssue.GetHTMLURL(),
				LastSyncedAt: time.Now(),
			}
		}

		// Map GitHub state to Roady status
		newStatus := mapGitHubStatus(targetIssue)
		currentStatus := planning.StatusPending
		if res, ok := state.TaskStates[task.ID]; ok {
			currentStatus = res.Status
		}

		if newStatus != currentStatus {
			result.StatusUpdates[task.ID] = newStatus
		}
	}

	return result, nil
}

func (s *GitHubSyncer) createIssue(ctx context.Context, task planning.Task) (*github.Issue, error) {
	body := task.Description
	if body == "" {
		body = task.Title
	}
	body = fmt.Sprintf("%s\n\nroady-id: %s", body, task.ID)

	issue, _, err := s.client.Issues.Create(ctx, s.owner, s.name, &github.IssueRequest{
		Title: github.String(task.Title),
		Body:  github.String(body),
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func extractRoadyID(body string) string {
	if strings.Contains(body, "roady-id: ") {
		idx := strings.Index(body, "roady-id: ")
		remaining := body[idx+10:]
		if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
			return strings.TrimSpace(remaining[:nlIdx])
		}
		return strings.TrimSpace(remaining)
	}
	return ""
}

func mapGitHubStatus(issue *github.Issue) planning.TaskStatus {
	if issue.GetState() == "closed" {
		return planning.StatusDone
	}
	// Open issue: check if assigned
	if issue.GetAssignee() != nil {
		return planning.StatusInProgress
	}
	return planning.StatusPending
}

func (s *GitHubSyncer) Push(taskID string, status planning.TaskStatus) error {
	ctx := context.Background()
	log.Printf("GitHub Syncer: Pushing status %s for task %s", status, taskID)

	// Find the issue by roady-id marker
	issues, err := s.fetchAllIssues(ctx)
	if err != nil {
		return fmt.Errorf("fetch issues: %w", err)
	}

	var targetIssue *github.Issue
	for _, issue := range issues {
		if issue.IsPullRequest() {
			continue
		}
		if extractRoadyID(issue.GetBody()) == taskID {
			targetIssue = issue
			break
		}
	}

	if targetIssue == nil {
		return fmt.Errorf("issue not found for task %s", taskID)
	}

	// Map Roady status to GitHub state
	var newState string
	switch status {
	case planning.StatusDone:
		newState = "closed"
	case planning.StatusPending, planning.StatusInProgress, planning.StatusBlocked:
		newState = "open"
	default:
		return fmt.Errorf("unsupported status: %s", status)
	}

	// Only update if state differs
	if targetIssue.GetState() == newState {
		log.Printf("GitHub Syncer: Issue #%d already in state %s", targetIssue.GetNumber(), newState)
		return nil
	}

	_, _, err = s.client.Issues.Edit(ctx, s.owner, s.name, targetIssue.GetNumber(), &github.IssueRequest{
		State: github.String(newState),
	})
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}

	log.Printf("GitHub Syncer: Updated issue #%d to state %s", targetIssue.GetNumber(), newState)
	return nil
}

func main() {
	// Plugin serving
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]goplugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &GitHubSyncer{}},
		},
	})
}
