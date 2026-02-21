package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"github.com/hashicorp/go-plugin"
)

var asanaBaseURL = "https://app.asana.com/api/1.0"

// AsanaSyncer syncs tasks with an Asana project.
type AsanaSyncer struct {
	token     string
	projectID string
	client    *http.Client
}

func (s *AsanaSyncer) Init(config map[string]string) error {
	s.token = config["token"]
	s.projectID = config["project_id"]

	// Fallback to env vars
	if s.token == "" {
		s.token = os.Getenv("ASANA_TOKEN")
	}
	if s.projectID == "" {
		s.projectID = os.Getenv("ASANA_PROJECT_ID")
	}

	if s.token == "" {
		return fmt.Errorf("asana token is required (config 'token' or env ASANA_TOKEN)")
	}
	if s.projectID == "" {
		return fmt.Errorf("asana project_id is required (config 'project_id' or env ASANA_PROJECT_ID)")
	}

	s.client = &http.Client{Timeout: 30 * time.Second}

	return nil
}

// AsanaTask represents an Asana task.
type AsanaTask struct {
	GID         string  `json:"gid"`
	Name        string  `json:"name"`
	Notes       string  `json:"notes"`
	Completed   bool    `json:"completed"`
	PermalinkURL string `json:"permalink_url"`
	Assignee    *struct {
		GID string `json:"gid"`
	} `json:"assignee,omitempty"`
	CustomFields []AsanaCustomField `json:"custom_fields,omitempty"`
}

// AsanaCustomField represents a custom field on an Asana task.
type AsanaCustomField struct {
	GID         string `json:"gid"`
	Name        string `json:"name"`
	TextValue   string `json:"text_value,omitempty"`
	DisplayValue string `json:"display_value,omitempty"`
}

// AsanaTasksResponse is the response from listing tasks.
type AsanaTasksResponse struct {
	Data     []AsanaTask `json:"data"`
	NextPage *struct {
		Offset string `json:"offset"`
		URI    string `json:"uri"`
	} `json:"next_page,omitempty"`
}

// AsanaTaskResponse is the response from getting/creating a task.
type AsanaTaskResponse struct {
	Data AsanaTask `json:"data"`
}

func (s *AsanaSyncer) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, asanaBaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close on read body

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("asana API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *AsanaSyncer) listTasks(ctx context.Context) ([]AsanaTask, error) {
	var allTasks []AsanaTask
	endpoint := "/projects/" + s.projectID + "/tasks?opt_fields=gid,name,notes,completed,permalink_url,assignee,custom_fields"

	for {
		respBody, err := s.doRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		var result AsanaTasksResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		allTasks = append(allTasks, result.Data...)

		if result.NextPage == nil {
			break
		}
		endpoint = result.NextPage.URI[len(asanaBaseURL):]
	}

	return allTasks, nil
}

func (s *AsanaSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	ctx := context.Background()
	log.Printf("Asana Syncer: Syncing %d tasks with project %s", len(plan.Tasks), s.projectID)

	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// List all tasks from the project
	tasks, err := s.listTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// Index tasks by roady-id and name
	taskByRoadyID := make(map[string]AsanaTask)
	taskByName := make(map[string]AsanaTask)
	for _, task := range tasks {
		if rid := extractRoadyIDFromNotes(task.Notes); rid != "" {
			taskByRoadyID[rid] = task
		}
		taskByName[task.Name] = task
	}

	for _, task := range plan.Tasks {
		var targetTask *AsanaTask

		// 1. Check state refs first
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["asana"]; ok {
				for _, t := range tasks {
					if t.GID == ref.ID {
						targetTask = &t
						break
					}
				}
			}
		}

		// 2. Match by roady-id in notes
		if targetTask == nil {
			if t, ok := taskByRoadyID[task.ID]; ok {
				targetTask = &t
			}
		}

		// 3. Fall back to name matching
		if targetTask == nil {
			if t, ok := taskByName[task.Title]; ok {
				targetTask = &t
			}
		}

		// 4. Create task if not found
		if targetTask == nil {
			t, err := s.createTask(ctx, task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create task for %s: %v", task.ID, err))
				continue
			}
			targetTask = t
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           t.GID,
				Identifier:   shortenGID(t.GID),
				URL:          t.PermalinkURL,
				LastSyncedAt: time.Now(),
			}
		} else {
			// Update link for existing task
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           targetTask.GID,
				Identifier:   shortenGID(targetTask.GID),
				URL:          targetTask.PermalinkURL,
				LastSyncedAt: time.Now(),
			}
		}

		// Map Asana task state to Roady status
		newStatus := mapAsanaStatus(*targetTask)
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

func (s *AsanaSyncer) createTask(ctx context.Context, task planning.Task) (*AsanaTask, error) {
	notes := task.Description
	if notes == "" {
		notes = task.Title
	}
	notes = notes + "\n\nroady-id: " + task.ID

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"name":     task.Title,
			"notes":    notes,
			"projects": []string{s.projectID},
		},
	}

	respBody, err := s.doRequest(ctx, "POST", "/tasks", body)
	if err != nil {
		return nil, err
	}

	var result AsanaTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (s *AsanaSyncer) Push(taskID string, status planning.TaskStatus) error {
	ctx := context.Background()
	log.Printf("Asana Syncer: Pushing status %s for task %s", status, taskID)

	// Find the task by roady-id
	tasks, err := s.listTasks(ctx)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	var targetTask *AsanaTask
	for _, t := range tasks {
		if extractRoadyIDFromNotes(t.Notes) == taskID {
			targetTask = &t
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task not found for %s", taskID)
	}

	// Map Roady status to Asana completed flag
	completed := status == planning.StatusDone || status == planning.StatusVerified

	if targetTask.Completed == completed {
		log.Printf("Asana Syncer: Task %s already in correct state", shortenGID(targetTask.GID))
		return nil
	}

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"completed": completed,
		},
	}

	_, err = s.doRequest(ctx, "PUT", "/tasks/"+targetTask.GID, body)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	log.Printf("Asana Syncer: Updated task %s completed=%v", shortenGID(targetTask.GID), completed)
	return nil
}

func extractRoadyIDFromNotes(notes string) string {
	if strings.Contains(notes, "roady-id: ") {
		idx := strings.Index(notes, "roady-id: ")
		remaining := notes[idx+10:]
		if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
			return strings.TrimSpace(remaining[:nlIdx])
		}
		return strings.TrimSpace(remaining)
	}
	return ""
}

func mapAsanaStatus(task AsanaTask) planning.TaskStatus {
	if task.Completed {
		return planning.StatusDone
	}
	// Has assignee means in progress
	if task.Assignee != nil {
		return planning.StatusInProgress
	}
	return planning.StatusPending
}

func shortenGID(gid string) string {
	if len(gid) > 8 {
		return gid[:8]
	}
	return gid
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &AsanaSyncer{}},
		},
	})
}
