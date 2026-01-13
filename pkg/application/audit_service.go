package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/google/uuid"
)

type AuditService struct {
	repo domain.WorkspaceRepository
}

func NewAuditService(repo domain.WorkspaceRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(action string, actor string, metadata map[string]interface{}) error {
	// 1. Get the latest event to continue the hash chain
	events, _ := s.repo.LoadEvents()
	prevHash := ""
	if len(events) > 0 {
		prevHash = events[len(events)-1].Hash
	}

	event := domain.Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Action:    action,
		Actor:     actor,
		Metadata:  metadata,
		PrevHash:  prevHash,
	}
	event.Hash = event.CalculateHash()

	if err := s.repo.RecordEvent(event); err != nil {
		return err
	}

	// Update Usage Stats
	stats, err := s.repo.LoadUsage()
	if err != nil || stats == nil {
		stats = &domain.UsageStats{
			ProviderStats: make(map[string]int),
		}
	}
	if stats.ProviderStats == nil {
		stats.ProviderStats = make(map[string]int)
	}

	stats.TotalCommands++
	stats.LastCommandAt = time.Now()

	// Track AI tokens if present in metadata
	if actor == "ai" && metadata != nil {
		if model, ok := metadata["model"].(string); ok {
			if in, ok := metadata["input_tokens"].(int); ok {
				stats.ProviderStats[model+":input"] += in
			}
			if out, ok := metadata["output_tokens"].(int); ok {
				stats.ProviderStats[model+":output"] += out
			}
		}
	}

	return s.repo.UpdateUsage(*stats)
}

func (s *AuditService) GetTimeline() ([]domain.Event, error) {
	return s.repo.LoadEvents()
}

func (s *AuditService) VerifyIntegrity() ([]string, error) {
	events, err := s.repo.LoadEvents()
	if err != nil {
		return nil, err
	}

	var violations []string
	lastHash := ""

	for i, e := range events {
		// 1. Verify links
		if e.PrevHash != lastHash {
			violations = append(violations, fmt.Sprintf("Event %d (%s): PrevHash mismatch. Audit trail broken.", i, e.ID))
		}

		// 2. Verify self-hash (requires a shallow copy to check without the hash field)
		expected := e.CalculateHash()
		if e.Hash != expected {
			violations = append(violations, fmt.Sprintf("Event %d (%s): Content hash mismatch. Possible tampering.", i, e.ID))
		}

		lastHash = e.Hash
	}

	return violations, nil
}

// GetVelocity returns the average verified tasks per day over the last 7 days.
func (s *AuditService) GetVelocity() (float64, error) {
	events, err := s.repo.LoadEvents()
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	var firstVerify time.Time
	verifiedCount := 0
	
	for _, e := range events {
		if e.Action == "task.transition" && e.Metadata["status"] == "verified" {
			if firstVerify.IsZero() {
				firstVerify = e.Timestamp
			}
			verifiedCount++
		}
	}

	if verifiedCount == 0 {
		return 0, nil
	}

	days := time.Since(firstVerify).Hours() / 24.0
	if days < 1 {
		days = 1 // Floor at 1 day to avoid infinity/large spikes
	}

	return float64(verifiedCount) / days, nil
}
