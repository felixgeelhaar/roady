package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

// GitHubHandler handles GitHub webhooks.
type GitHubHandler struct{}

// NewGitHubHandler creates a new GitHub webhook handler.
func NewGitHubHandler() *GitHubHandler {
	return &GitHubHandler{}
}

// Provider returns the provider name.
func (h *GitHubHandler) Provider() string {
	return "github"
}

// ValidateSignature validates the GitHub webhook signature.
func (h *GitHubHandler) ValidateSignature(r *http.Request, secret string) bool {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	// Reset body for later use
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// GitHubIssuePayload represents a GitHub issue webhook payload.
type GitHubIssuePayload struct {
	Action string `json:"action"`
	Issue  struct {
		Number   int    `json:"number"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		State    string `json:"state"`
		HTMLURL  string `json:"html_url"`
		Assignee *struct {
			Login string `json:"login"`
		} `json:"assignee"`
	} `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// ParseEvent parses a GitHub webhook request into an Event.
func (h *GitHubHandler) ParseEvent(r *http.Request) (*Event, error) {
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	event := &Event{
		Provider:  "github",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	switch eventType {
	case "issues":
		var payload GitHubIssuePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("parse issues payload: %w", err)
		}

		event.EventType = fmt.Sprintf("issue.%s", payload.Action)
		event.ExternalID = fmt.Sprintf("%d", payload.Issue.Number)
		event.TaskID = extractRoadyID(payload.Issue.Body)
		event.Status = mapGitHubIssueStatus(payload.Issue.State, payload.Issue.Assignee != nil)

		event.Metadata["title"] = payload.Issue.Title
		event.Metadata["url"] = payload.Issue.HTMLURL
		event.Metadata["repo"] = payload.Repository.FullName
		event.Metadata["action"] = payload.Action

	case "ping":
		event.EventType = "ping"

	default:
		event.EventType = eventType
	}

	return event, nil
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

func mapGitHubIssueStatus(state string, hasAssignee bool) planning.TaskStatus {
	if state == "closed" {
		return planning.StatusDone
	}
	if hasAssignee {
		return planning.StatusInProgress
	}
	return planning.StatusPending
}
