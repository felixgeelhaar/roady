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

const (
	notionAPIVersion = "2022-06-28"
	notionBaseURL    = "https://api.notion.com/v1"
)

// NotionSyncer syncs tasks with a Notion database.
type NotionSyncer struct {
	token      string
	databaseID string
	client     *http.Client
}

func (s *NotionSyncer) Init(config map[string]string) error {
	s.token = config["token"]
	s.databaseID = config["database_id"]

	// Fallback to env vars
	if s.token == "" {
		s.token = os.Getenv("NOTION_TOKEN")
	}
	if s.databaseID == "" {
		s.databaseID = os.Getenv("NOTION_DATABASE_ID")
	}

	if s.token == "" {
		return fmt.Errorf("notion token is required (config 'token' or env NOTION_TOKEN)")
	}
	if s.databaseID == "" {
		return fmt.Errorf("notion database_id is required (config 'database_id' or env NOTION_DATABASE_ID)")
	}

	s.client = &http.Client{Timeout: 30 * time.Second}

	return nil
}

// NotionPage represents a Notion database page.
type NotionPage struct {
	ID         string                 `json:"id"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"properties"`
}

// NotionQueryResult is the response from querying a database.
type NotionQueryResult struct {
	Results    []NotionPage `json:"results"`
	HasMore    bool         `json:"has_more"`
	NextCursor string       `json:"next_cursor"`
}

func (s *NotionSyncer) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, notionBaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("notion API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *NotionSyncer) queryDatabase(ctx context.Context) ([]NotionPage, error) {
	var allPages []NotionPage
	var cursor string

	for {
		body := map[string]interface{}{
			"page_size": 100,
		}
		if cursor != "" {
			body["start_cursor"] = cursor
		}

		respBody, err := s.doRequest(ctx, "POST", "/databases/"+s.databaseID+"/query", body)
		if err != nil {
			return nil, err
		}

		var result NotionQueryResult
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		allPages = append(allPages, result.Results...)

		if !result.HasMore {
			break
		}
		cursor = result.NextCursor
	}

	return allPages, nil
}

func (s *NotionSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	ctx := context.Background()
	log.Printf("Notion Syncer: Syncing %d tasks with database %s", len(plan.Tasks), s.databaseID)

	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// Query all pages from the database
	pages, err := s.queryDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("query database: %w", err)
	}

	// Index pages by roady-id and title
	pageByRoadyID := make(map[string]NotionPage)
	pageByTitle := make(map[string]NotionPage)
	for _, page := range pages {
		// Check for roady-id in properties
		if rid := extractRoadyIDFromPage(page); rid != "" {
			pageByRoadyID[rid] = page
		}
		// Index by title
		if title := getPageTitle(page); title != "" {
			pageByTitle[title] = page
		}
	}

	for _, task := range plan.Tasks {
		var targetPage *NotionPage

		// 1. Check state refs first
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["notion"]; ok {
				for _, page := range pages {
					if page.ID == ref.ID {
						targetPage = &page
						break
					}
				}
			}
		}

		// 2. Match by roady-id
		if targetPage == nil {
			if page, ok := pageByRoadyID[task.ID]; ok {
				targetPage = &page
			}
		}

		// 3. Fall back to title matching
		if targetPage == nil {
			if page, ok := pageByTitle[task.Title]; ok {
				targetPage = &page
			}
		}

		// 4. Create page if not found
		if targetPage == nil {
			page, err := s.createPage(ctx, task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create page for %s: %v", task.ID, err))
				continue
			}
			targetPage = page
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           page.ID,
				Identifier:   shortenID(page.ID),
				URL:          page.URL,
				LastSyncedAt: time.Now(),
			}
		} else {
			// Update link for existing page
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           targetPage.ID,
				Identifier:   shortenID(targetPage.ID),
				URL:          targetPage.URL,
				LastSyncedAt: time.Now(),
			}
		}

		// Map Notion status to Roady status
		newStatus := mapNotionStatus(*targetPage)
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

func (s *NotionSyncer) createPage(ctx context.Context, task planning.Task) (*NotionPage, error) {
	body := map[string]interface{}{
		"parent": map[string]string{
			"database_id": s.databaseID,
		},
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]string{
							"content": task.Title,
						},
					},
				},
			},
			"Roady ID": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"text": map[string]string{
							"content": task.ID,
						},
					},
				},
			},
		},
		"children": []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]string{
								"content": task.Description,
							},
						},
					},
				},
			},
		},
	}

	respBody, err := s.doRequest(ctx, "POST", "/pages", body)
	if err != nil {
		return nil, err
	}

	var page NotionPage
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

func (s *NotionSyncer) Push(taskID string, status planning.TaskStatus) error {
	ctx := context.Background()
	log.Printf("Notion Syncer: Pushing status %s for task %s", status, taskID)

	// Find the page by roady-id
	pages, err := s.queryDatabase(ctx)
	if err != nil {
		return fmt.Errorf("query database: %w", err)
	}

	var targetPage *NotionPage
	for _, page := range pages {
		if extractRoadyIDFromPage(page) == taskID {
			targetPage = &page
			break
		}
	}

	if targetPage == nil {
		return fmt.Errorf("page not found for task %s", taskID)
	}

	// Map Roady status to Notion status property
	notionStatus := mapRoadyToNotionStatus(status)

	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"Status": map[string]interface{}{
				"status": map[string]string{
					"name": notionStatus,
				},
			},
		},
	}

	_, err = s.doRequest(ctx, "PATCH", "/pages/"+targetPage.ID, body)
	if err != nil {
		return fmt.Errorf("update page: %w", err)
	}

	log.Printf("Notion Syncer: Updated page %s to status %s", shortenID(targetPage.ID), notionStatus)
	return nil
}

func extractRoadyIDFromPage(page NotionPage) string {
	// Try to get roady-id from "Roady ID" property
	if prop, ok := page.Properties["Roady ID"]; ok {
		if richText, ok := prop.(map[string]interface{})["rich_text"]; ok {
			if items, ok := richText.([]interface{}); ok && len(items) > 0 {
				if item, ok := items[0].(map[string]interface{}); ok {
					if text, ok := item["plain_text"].(string); ok {
						return strings.TrimSpace(text)
					}
				}
			}
		}
	}
	return ""
}

func getPageTitle(page NotionPage) string {
	// Try common title property names
	for _, propName := range []string{"Name", "Title", "name", "title"} {
		if prop, ok := page.Properties[propName]; ok {
			if titleProp, ok := prop.(map[string]interface{})["title"]; ok {
				if items, ok := titleProp.([]interface{}); ok && len(items) > 0 {
					if item, ok := items[0].(map[string]interface{}); ok {
						if text, ok := item["plain_text"].(string); ok {
							return text
						}
					}
				}
			}
		}
	}
	return ""
}

func mapNotionStatus(page NotionPage) planning.TaskStatus {
	// Try to get status from "Status" property
	if prop, ok := page.Properties["Status"]; ok {
		if statusProp, ok := prop.(map[string]interface{})["status"]; ok {
			if statusObj, ok := statusProp.(map[string]interface{}); ok {
				if name, ok := statusObj["name"].(string); ok {
					switch strings.ToLower(name) {
					case "done", "complete", "completed", "finished":
						return planning.StatusDone
					case "in progress", "in_progress", "doing", "started":
						return planning.StatusInProgress
					case "blocked":
						return planning.StatusBlocked
					case "verified":
						return planning.StatusVerified
					}
				}
			}
		}
	}
	return planning.StatusPending
}

func mapRoadyToNotionStatus(status planning.TaskStatus) string {
	switch status {
	case planning.StatusDone:
		return "Done"
	case planning.StatusInProgress:
		return "In Progress"
	case planning.StatusBlocked:
		return "Blocked"
	case planning.StatusVerified:
		return "Done" // Notion doesn't have "Verified", map to Done
	default:
		return "Not Started"
	}
}

func shortenID(id string) string {
	id = strings.ReplaceAll(id, "-", "")
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &NotionSyncer{}},
		},
	})
}
