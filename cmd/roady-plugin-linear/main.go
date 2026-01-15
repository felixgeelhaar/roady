package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"github.com/hashicorp/go-plugin"
)

type LinearSyncer struct {
	apiKey string
	teamID string
}

func (s *LinearSyncer) Init(config map[string]string) error {
	s.apiKey = config["api_key"]
	if s.apiKey == "" {
		s.apiKey = os.Getenv("LINEAR_API_KEY")
	}
	s.teamID = config["team_id"]
	if s.teamID == "" {
		s.teamID = os.Getenv("LINEAR_TEAM_ID")
	}

	if s.apiKey == "" {
		return fmt.Errorf("LINEAR_API_KEY is required")
	}
	if s.teamID == "" {
		return fmt.Errorf("LINEAR_TEAM_ID is required")
	}
	return nil
}

type linearIssue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       struct {
		Name string `json:"name"`
		Type string `json:"type"` // unstarted, started, completed, canceled, backlog, triage
	} `json:"state"`
	URL string `json:"url"`
}

func (s *LinearSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// 1. Fetch existing issues for the team
	existingIssues, err := s.fetchTeamIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch linear issues: %w", err)
	}

	// Create a map for lookup by roady-id (stored in description)
	issueByRoadyID := make(map[string]linearIssue)
	for _, issue := range existingIssues {
		if rid := extractRoadyID(issue.Description); rid != "" {
			issueByRoadyID[rid] = issue
		}
	}

	// 2. Iterate through Roady tasks
	for _, task := range plan.Tasks {
		var targetIssue *linearIssue

		// A. Check if we already have a link in state
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["linear"]; ok {
				// Try to find by ID
				for _, issue := range existingIssues {
					if issue.ID == ref.ID {
						targetIssue = &issue
						break
					}
				}
			}
		}

		// B. Fallback to matching by description marker
		if targetIssue == nil {
			if issue, ok := issueByRoadyID[task.ID]; ok {
				targetIssue = &issue
			}
		}

		// C. Create issue if missing
		if targetIssue == nil {
			issue, err := s.createIssue(task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create issue for %s: %v", task.ID, err))
				continue
			}
			targetIssue = issue
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           issue.ID,
				Identifier:   issue.Identifier,
				URL:          issue.URL,
				LastSyncedAt: time.Now(),
			}
		}

		// 3. Map status from Linear to Roady
		newStatus := mapLinearStatus(targetIssue.State.Type, targetIssue.State.Name)
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

func (s *LinearSyncer) query(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	req, _ := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("linear api returned status %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp struct {
		Data   map[string]interface{} `json:"data"`
		Errors []interface{}          `json:"errors"`
	}
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("linear graphql errors: %v", gqlResp.Errors)
	}

	return gqlResp.Data, nil
}

func (s *LinearSyncer) fetchTeamIssues() ([]linearIssue, error) {
	q := `query($teamId: String!) {
		team(id: $teamId) {
			issues {
				nodes {
					id
					identifier
					title
					description
					state {
						name
						type
					}
					url
				}
			}
		}
	}`
	data, err := s.query(q, map[string]interface{}{"teamId": s.teamID})
	if err != nil {
		return nil, err
	}

	team, ok := data["team"].(map[string]interface{})
	if !ok || team == nil {
		return nil, fmt.Errorf("team not found")
	}
	issuesData := team["issues"].(map[string]interface{})["nodes"]

	var issues []linearIssue
	marshaled, _ := json.Marshal(issuesData)
	json.Unmarshal(marshaled, &issues)

	return issues, nil
}

func (s *LinearSyncer) createIssue(task planning.Task) (*linearIssue, error) {
	q := `mutation($teamId: String!, $title: String!, $description: String!) {
		issueCreate(input: { teamId: $teamId, title: $title, description: $description }) {
			success
			issue {
				id
				identifier
				title
				description
				state {
					name
					type
				}
				url
			}
		}
	}`

	desc := fmt.Sprintf("%s\n\n---\nroady-id: %s", task.Description, task.ID)
	data, err := s.query(q, map[string]interface{}{
		"teamId":      s.teamID,
		"title":       task.Title,
		"description": desc,
	})
	if err != nil {
		return nil, err
	}

	createData := data["issueCreate"].(map[string]interface{})
	if !createData["success"].(bool) {
		return nil, fmt.Errorf("failed to create issue")
	}

	var issue linearIssue
	marshaled, _ := json.Marshal(createData["issue"])
	json.Unmarshal(marshaled, &issue)

	return &issue, nil
}

func extractRoadyID(desc string) string {
	if strings.Contains(desc, "roady-id: ") {
		parts := strings.Split(desc, "roady-id: ")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func mapLinearStatus(linearType string, linearName string) planning.TaskStatus {
	switch linearType {
	case "completed":
		return planning.StatusDone
	case "started":
		return planning.StatusInProgress
	case "canceled":
		return planning.StatusBlocked // Or some other appropriate mapping
	case "backlog", "unstarted", "triage":
		return planning.StatusPending
	default:
		return planning.StatusPending
	}
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &LinearSyncer{}},
		},
	})
}
