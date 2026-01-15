package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// JiraHandler handles Jira webhooks.
type JiraHandler struct{}

// NewJiraHandler creates a new Jira webhook handler.
func NewJiraHandler() *JiraHandler {
	return &JiraHandler{}
}

// Provider returns the provider name.
func (h *JiraHandler) Provider() string {
	return "jira"
}

// ValidateSignature validates the Jira webhook.
// Jira Cloud uses a shared secret that must be validated via the webhook URL itself.
// For Jira Server/Data Center, you might use IP whitelisting or other methods.
// This implementation trusts the configured secret as a simple token check.
func (h *JiraHandler) ValidateSignature(r *http.Request, secret string) bool {
	// Jira doesn't use HMAC signatures by default.
	// Common approaches:
	// 1. Include secret in webhook URL as query param
	// 2. Use IP whitelisting
	// 3. Check for specific header

	// Check for secret in query param
	if r.URL.Query().Get("secret") == secret {
		return true
	}

	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "Bearer "+secret {
		return true
	}

	// If no secret configured, allow all
	return secret == ""
}

// JiraIssuePayload represents a Jira issue webhook payload.
type JiraIssuePayload struct {
	WebhookEvent string `json:"webhookEvent"`
	Issue        struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Self   string `json:"self"`
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Status      struct {
				Name       string `json:"name"`
				StatusCategory struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
		} `json:"fields"`
	} `json:"issue"`
	Changelog *struct {
		Items []struct {
			Field      string `json:"field"`
			FromString string `json:"fromString"`
			ToString   string `json:"toString"`
		} `json:"items"`
	} `json:"changelog"`
}

// ParseEvent parses a Jira webhook request into an Event.
func (h *JiraHandler) ParseEvent(r *http.Request) (*Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var payload JiraIssuePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse jira payload: %w", err)
	}

	event := &Event{
		Provider:   "jira",
		EventType:  payload.WebhookEvent,
		ExternalID: payload.Issue.Key,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Extract roady-id from description
	event.TaskID = extractRoadyIDJira(payload.Issue.Fields.Description)

	// Map status
	event.Status = mapJiraStatusToRoady(
		payload.Issue.Fields.Status.Name,
		payload.Issue.Fields.Status.StatusCategory.Key,
	)

	event.Metadata["key"] = payload.Issue.Key
	event.Metadata["summary"] = payload.Issue.Fields.Summary
	event.Metadata["status"] = payload.Issue.Fields.Status.Name

	// Track status changes
	if payload.Changelog != nil {
		for _, item := range payload.Changelog.Items {
			if item.Field == "status" {
				event.Metadata["status_from"] = item.FromString
				event.Metadata["status_to"] = item.ToString
			}
		}
	}

	return event, nil
}

func extractRoadyIDJira(desc string) string {
	if strings.Contains(desc, "roady-id: ") {
		idx := strings.Index(desc, "roady-id: ")
		remaining := desc[idx+10:]
		if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
			return strings.TrimSpace(remaining[:nlIdx])
		}
		return strings.TrimSpace(remaining)
	}
	return ""
}

func mapJiraStatusToRoady(statusName, categoryKey string) planning.TaskStatus {
	// Jira status categories: new, indeterminate, done
	switch categoryKey {
	case "done":
		return planning.StatusDone
	case "indeterminate":
		return planning.StatusInProgress
	case "new":
		return planning.StatusPending
	}

	// Fallback to name-based matching
	name := strings.ToLower(statusName)
	switch {
	case strings.Contains(name, "done") || strings.Contains(name, "closed") || strings.Contains(name, "resolved"):
		return planning.StatusDone
	case strings.Contains(name, "progress") || strings.Contains(name, "started"):
		return planning.StatusInProgress
	case strings.Contains(name, "blocked") || strings.Contains(name, "hold"):
		return planning.StatusBlocked
	default:
		return planning.StatusPending
	}
}
