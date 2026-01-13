package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/internal/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/internal/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/internal/infrastructure/plugin"
	"github.com/hashicorp/go-plugin"
)

type JiraSyncer struct {
	domain     string
	projectKey string
	email      string
	apiToken   string
}

func (s *JiraSyncer) Init(config map[string]string) error {
	s.domain = config["domain"]
	if s.domain == "" {
		s.domain = os.Getenv("JIRA_DOMAIN")
	}
	s.projectKey = config["project_key"]
	if s.projectKey == "" {
		s.projectKey = os.Getenv("JIRA_PROJECT_KEY")
	}
	s.email = config["email"]
	if s.email == "" {
		s.email = os.Getenv("JIRA_EMAIL")
	}
	s.apiToken = config["api_token"]
	if s.apiToken == "" {
		s.apiToken = os.Getenv("JIRA_API_TOKEN")
	}

	if s.domain == "" || s.projectKey == "" || s.email == "" || s.apiToken == "" {
		return fmt.Errorf("Jira configuration missing (domain, project_key, email, api_token required)")
	}

	if !strings.HasPrefix(s.domain, "http") {
		s.domain = "https://" + s.domain
	}
	return nil
}

type jiraIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"status"`
	} `json:"fields"`
	Self string `json:"self"`
}

func (s *JiraSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// 1. Fetch issues for the project
	existingIssues, err := s.fetchProjectIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jira issues: %w", err)
	}

	issueByRoadyID := make(map[string]jiraIssue)
	for _, issue := range existingIssues {
		if rid := extractRoadyID(issue.Fields.Description); rid != "" {
			issueByRoadyID[rid] = issue
		}
	}

	// 2. Iterate through Roady tasks
	for _, task := range plan.Tasks {
		var targetIssue *jiraIssue

		// A. Check state links
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["jira"]; ok {
				for _, issue := range existingIssues {
					if issue.ID == ref.ID || issue.Key == ref.Identifier {
						targetIssue = &issue
						break
					}
				}
			}
		}

		// B. Match by marker
		if targetIssue == nil {
			if issue, ok := issueByRoadyID[task.ID]; ok {
				targetIssue = &issue
			}
		}

		// C. Create if missing
		if targetIssue == nil {
			issue, err := s.createIssue(task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create jira issue for %s: %v", task.ID, err))
				continue
			}
			targetIssue = issue
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           issue.ID,
				Identifier:   issue.Key,
				URL:          fmt.Sprintf("%s/browse/%s", s.domain, issue.Key),
				LastSyncedAt: time.Now(),
			}
		}

		// 3. Map Status
		newStatus := mapJiraStatus(targetIssue.Fields.Status.Name)
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

func (s *JiraSyncer) request(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	}

	url := fmt.Sprintf("%s/rest/api/2/%s", s.domain, path)
	req, _ := http.NewRequest(method, url, bodyReader)
	
	auth := base64.StdEncoding.EncodeToString([]byte(s.email + ":" + s.apiToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jira api error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *JiraSyncer) fetchProjectIssues() ([]jiraIssue, error) {
	jql := fmt.Sprintf("project = '%s'", s.projectKey)
	path := fmt.Sprintf("search?jql=%s&fields=summary,description,status", jql)
	
	data, err := s.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var searchResp struct {
		Issues []jiraIssue `json:"issues"`
	}
	if err := json.Unmarshal(data, &searchResp); err != nil {
		return nil, err
	}

	return searchResp.Issues, nil
}

func (s *JiraSyncer) createIssue(task planning.Task) (*jiraIssue, error) {
	description := fmt.Sprintf("%s\n\nroady-id: %s", task.Description, task.ID)
	
	input := map[string]interface{}{
		"fields": map[string]interface{}{
			"project":     map[string]string{"key": s.projectKey},
			"summary":     task.Title,
			"description": description,
			"issuetype":   map[string]string{"name": "Task"},
		},
	}

	data, err := s.request("POST", "issue", input)
	if err != nil {
		return nil, err
	}

	var created struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	json.Unmarshal(data, &created)

	// Fetch full details to get status etc
	return s.getIssue(created.ID)
}

func (s *JiraSyncer) getIssue(id string) (*jiraIssue, error) {
	data, err := s.request("GET", "issue/"+id, nil)
	if err != nil {
		return nil, err
	}
	var issue jiraIssue
	json.Unmarshal(data, &issue)
	return &issue, nil
}

func extractRoadyID(desc string) string {
	if strings.Contains(desc, "roady-id: ") {
		idx := strings.Index(desc, "roady-id: ")
		return strings.TrimSpace(desc[idx+10:])
	}
	return ""
}

func mapJiraStatus(jiraName string) planning.TaskStatus {
	name := strings.ToLower(jiraName)
	switch name {
	case "done", "resolved", "closed", "verified":
		return planning.StatusDone
	case "in progress", "started", "active", "doing":
		return planning.StatusInProgress
	case "blocked", "on hold":
		return planning.StatusBlocked
	default:
		return planning.StatusPending
	}
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &JiraSyncer{}},
		},
	})
}
