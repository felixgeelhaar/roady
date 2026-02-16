package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
	"github.com/hashicorp/go-plugin"
)

var trelloBaseURL = "https://api.trello.com/1"

// TrelloSyncer syncs tasks with a Trello board.
type TrelloSyncer struct {
	apiKey    string
	token     string
	boardID   string
	todoListID string
	doneListID string
	client    *http.Client
	lists     map[string]TrelloList // cached lists
}

// TrelloList represents a Trello list.
type TrelloList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TrelloCard represents a Trello card.
type TrelloCard struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Desc    string `json:"desc"`
	IDList  string `json:"idList"`
	URL     string `json:"url"`
	ShortID int    `json:"idShort"`
	Members []struct {
		ID string `json:"id"`
	} `json:"idMembers"`
}

func (s *TrelloSyncer) Init(config map[string]string) error {
	s.apiKey = config["api_key"]
	s.token = config["token"]
	s.boardID = config["board_id"]
	s.todoListID = config["todo_list_id"]
	s.doneListID = config["done_list_id"]

	// Fallback to env vars
	if s.apiKey == "" {
		s.apiKey = os.Getenv("TRELLO_API_KEY")
	}
	if s.token == "" {
		s.token = os.Getenv("TRELLO_TOKEN")
	}
	if s.boardID == "" {
		s.boardID = os.Getenv("TRELLO_BOARD_ID")
	}
	if s.todoListID == "" {
		s.todoListID = os.Getenv("TRELLO_TODO_LIST_ID")
	}
	if s.doneListID == "" {
		s.doneListID = os.Getenv("TRELLO_DONE_LIST_ID")
	}

	if s.apiKey == "" {
		return fmt.Errorf("trello api_key is required (config 'api_key' or env TRELLO_API_KEY)")
	}
	if s.token == "" {
		return fmt.Errorf("trello token is required (config 'token' or env TRELLO_TOKEN)")
	}
	if s.boardID == "" {
		return fmt.Errorf("trello board_id is required (config 'board_id' or env TRELLO_BOARD_ID)")
	}

	s.client = &http.Client{Timeout: 30 * time.Second}
	s.lists = make(map[string]TrelloList)

	// Load and cache lists
	if err := s.loadLists(context.Background()); err != nil {
		return fmt.Errorf("load lists: %w", err)
	}

	return nil
}

func (s *TrelloSyncer) buildURL(endpoint string, params map[string]string) string {
	u := trelloBaseURL + endpoint
	v := url.Values{}
	v.Set("key", s.apiKey)
	v.Set("token", s.token)
	for k, val := range params {
		v.Set(k, val)
	}
	return u + "?" + v.Encode()
}

func (s *TrelloSyncer) doGet(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.buildURL(endpoint, params), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("trello API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (s *TrelloSyncer) doPost(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.buildURL(endpoint, params), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("trello API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (s *TrelloSyncer) doPut(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", s.buildURL(endpoint, params), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("trello API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (s *TrelloSyncer) loadLists(ctx context.Context) error {
	body, err := s.doGet(ctx, "/boards/"+s.boardID+"/lists", nil)
	if err != nil {
		return err
	}

	var lists []TrelloList
	if err := json.Unmarshal(body, &lists); err != nil {
		return err
	}

	for _, list := range lists {
		s.lists[list.ID] = list
		// Auto-detect todo/done lists by name if not configured
		lowerName := strings.ToLower(list.Name)
		if s.todoListID == "" && (lowerName == "to do" || lowerName == "todo" || lowerName == "backlog") {
			s.todoListID = list.ID
		}
		if s.doneListID == "" && (lowerName == "done" || lowerName == "complete" || lowerName == "completed") {
			s.doneListID = list.ID
		}
	}

	return nil
}

func (s *TrelloSyncer) getCards(ctx context.Context) ([]TrelloCard, error) {
	body, err := s.doGet(ctx, "/boards/"+s.boardID+"/cards", map[string]string{
		"fields": "id,name,desc,idList,url,idShort,idMembers",
	})
	if err != nil {
		return nil, err
	}

	var cards []TrelloCard
	if err := json.Unmarshal(body, &cards); err != nil {
		return nil, err
	}

	return cards, nil
}

func (s *TrelloSyncer) Sync(plan *planning.Plan, state *planning.ExecutionState) (*domainPlugin.SyncResult, error) {
	ctx := context.Background()
	log.Printf("Trello Syncer: Syncing %d tasks with board %s", len(plan.Tasks), s.boardID)

	result := &domainPlugin.SyncResult{
		StatusUpdates: make(map[string]planning.TaskStatus),
		LinkUpdates:   make(map[string]planning.ExternalRef),
	}

	// Get all cards from the board
	cards, err := s.getCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cards: %w", err)
	}

	// Index cards by roady-id and name
	cardByRoadyID := make(map[string]TrelloCard)
	cardByName := make(map[string]TrelloCard)
	for _, card := range cards {
		if rid := extractRoadyIDFromDesc(card.Desc); rid != "" {
			cardByRoadyID[rid] = card
		}
		cardByName[card.Name] = card
	}

	for _, task := range plan.Tasks {
		var targetCard *TrelloCard

		// 1. Check state refs first
		if res, ok := state.TaskStates[task.ID]; ok {
			if ref, ok := res.ExternalRefs["trello"]; ok {
				for _, c := range cards {
					if c.ID == ref.ID {
						targetCard = &c
						break
					}
				}
			}
		}

		// 2. Match by roady-id in description
		if targetCard == nil {
			if c, ok := cardByRoadyID[task.ID]; ok {
				targetCard = &c
			}
		}

		// 3. Fall back to name matching
		if targetCard == nil {
			if c, ok := cardByName[task.Title]; ok {
				targetCard = &c
			}
		}

		// 4. Create card if not found
		if targetCard == nil {
			c, err := s.createCard(ctx, task)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create card for %s: %v", task.ID, err))
				continue
			}
			targetCard = c
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           c.ID,
				Identifier:   fmt.Sprintf("#%d", c.ShortID),
				URL:          c.URL,
				LastSyncedAt: time.Now(),
			}
		} else {
			// Update link for existing card
			result.LinkUpdates[task.ID] = planning.ExternalRef{
				ID:           targetCard.ID,
				Identifier:   fmt.Sprintf("#%d", targetCard.ShortID),
				URL:          targetCard.URL,
				LastSyncedAt: time.Now(),
			}
		}

		// Map Trello list to Roady status
		newStatus := s.mapTrelloStatus(*targetCard)
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

func (s *TrelloSyncer) createCard(ctx context.Context, task planning.Task) (*TrelloCard, error) {
	desc := task.Description
	if desc == "" {
		desc = task.Title
	}
	desc = desc + "\n\nroady-id: " + task.ID

	listID := s.todoListID
	if listID == "" {
		// Use first list if todo list not configured
		for id := range s.lists {
			listID = id
			break
		}
	}

	body, err := s.doPost(ctx, "/cards", map[string]string{
		"name":   task.Title,
		"desc":   desc,
		"idList": listID,
	})
	if err != nil {
		return nil, err
	}

	var card TrelloCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, err
	}

	return &card, nil
}

func (s *TrelloSyncer) Push(taskID string, status planning.TaskStatus) error {
	ctx := context.Background()
	log.Printf("Trello Syncer: Pushing status %s for task %s", status, taskID)

	// Find the card by roady-id
	cards, err := s.getCards(ctx)
	if err != nil {
		return fmt.Errorf("get cards: %w", err)
	}

	var targetCard *TrelloCard
	for _, c := range cards {
		if extractRoadyIDFromDesc(c.Desc) == taskID {
			targetCard = &c
			break
		}
	}

	if targetCard == nil {
		return fmt.Errorf("card not found for task %s", taskID)
	}

	// Map Roady status to Trello list
	targetListID := s.mapRoadyToTrelloList(status)
	if targetListID == "" {
		log.Printf("Trello Syncer: No target list configured for status %s", status)
		return nil
	}

	if targetCard.IDList == targetListID {
		log.Printf("Trello Syncer: Card #%d already in correct list", targetCard.ShortID)
		return nil
	}

	_, err = s.doPut(ctx, "/cards/"+targetCard.ID, map[string]string{
		"idList": targetListID,
	})
	if err != nil {
		return fmt.Errorf("update card: %w", err)
	}

	listName := "unknown"
	if list, ok := s.lists[targetListID]; ok {
		listName = list.Name
	}
	log.Printf("Trello Syncer: Moved card #%d to list %s", targetCard.ShortID, listName)
	return nil
}

func extractRoadyIDFromDesc(desc string) string {
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

func (s *TrelloSyncer) mapTrelloStatus(card TrelloCard) planning.TaskStatus {
	if card.IDList == s.doneListID {
		return planning.StatusDone
	}
	// Has members means in progress
	if len(card.Members) > 0 {
		return planning.StatusInProgress
	}
	// Check list name for common patterns
	if list, ok := s.lists[card.IDList]; ok {
		lowerName := strings.ToLower(list.Name)
		switch {
		case strings.Contains(lowerName, "done") || strings.Contains(lowerName, "complete"):
			return planning.StatusDone
		case strings.Contains(lowerName, "progress") || strings.Contains(lowerName, "doing"):
			return planning.StatusInProgress
		case strings.Contains(lowerName, "block"):
			return planning.StatusBlocked
		}
	}
	return planning.StatusPending
}

func (s *TrelloSyncer) mapRoadyToTrelloList(status planning.TaskStatus) string {
	switch status {
	case planning.StatusDone, planning.StatusVerified:
		return s.doneListID
	default:
		return s.todoListID
	}
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: infraPlugin.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"syncer": &domainPlugin.SyncerPlugin{Impl: &TrelloSyncer{}},
		},
	})
}
