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

// LinearHandler handles Linear webhooks.
type LinearHandler struct{}

// NewLinearHandler creates a new Linear webhook handler.
func NewLinearHandler() *LinearHandler {
	return &LinearHandler{}
}

// Provider returns the provider name.
func (h *LinearHandler) Provider() string {
	return "linear"
}

// ValidateSignature validates the Linear webhook signature.
// Linear uses HMAC-SHA256 with the webhook secret.
func (h *LinearHandler) ValidateSignature(r *http.Request, secret string) bool {
	signature := r.Header.Get("Linear-Signature")
	if signature == "" {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// LinearPayload represents a Linear webhook payload.
type LinearPayload struct {
	Action        string `json:"action"`
	Type          string `json:"type"`
	CreatedAt     string `json:"createdAt"`
	OrganizationID string `json:"organizationId"`
	Data          struct {
		ID          string `json:"id"`
		Identifier  string `json:"identifier"`
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		State       struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"` // triage, backlog, unstarted, started, completed, canceled
		} `json:"state"`
		Assignee *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"assignee"`
		Team struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"team"`
	} `json:"data"`
	UpdatedFrom *struct {
		StateID string `json:"stateId"`
	} `json:"updatedFrom"`
}

// ParseEvent parses a Linear webhook request into an Event.
func (h *LinearHandler) ParseEvent(r *http.Request) (*Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var payload LinearPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse linear payload: %w", err)
	}

	event := &Event{
		Provider:   "linear",
		EventType:  fmt.Sprintf("%s.%s", payload.Type, payload.Action),
		ExternalID: payload.Data.ID,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Extract roady-id from description
	event.TaskID = extractRoadyIDLinear(payload.Data.Description)

	// Map status
	event.Status = mapLinearStateToRoady(payload.Data.State.Type)

	event.Metadata["identifier"] = payload.Data.Identifier
	event.Metadata["title"] = payload.Data.Title
	event.Metadata["url"] = payload.Data.URL
	event.Metadata["state_name"] = payload.Data.State.Name
	event.Metadata["state_type"] = payload.Data.State.Type
	event.Metadata["team_key"] = payload.Data.Team.Key

	return event, nil
}

func extractRoadyIDLinear(desc string) string {
	if strings.Contains(desc, "roady-id: ") {
		idx := strings.Index(desc, "roady-id: ")
		remaining := desc[idx+10:]
		// Take until newline or end
		if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
			return strings.TrimSpace(remaining[:nlIdx])
		}
		return strings.TrimSpace(remaining)
	}
	return ""
}

func mapLinearStateToRoady(stateType string) planning.TaskStatus {
	switch stateType {
	case "completed":
		return planning.StatusDone
	case "started":
		return planning.StatusInProgress
	case "canceled":
		return planning.StatusBlocked
	case "backlog", "unstarted", "triage":
		return planning.StatusPending
	default:
		return planning.StatusPending
	}
}
