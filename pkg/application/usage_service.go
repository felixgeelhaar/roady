package application

import (
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
)

// UsageService tracks command and AI token usage separately from audit logging.
type UsageService struct {
	repo domain.WorkspaceRepository
}

func NewUsageService(repo domain.WorkspaceRepository) *UsageService {
	return &UsageService{repo: repo}
}

// IncrementCommand records that a command was executed.
func (s *UsageService) IncrementCommand() error {
	stats, err := s.loadOrInitStats()
	if err != nil {
		return err
	}

	stats.TotalCommands++
	stats.LastCommandAt = time.Now()

	return s.repo.UpdateUsage(*stats)
}

// RecordTokenUsage records AI token usage for a specific model.
func (s *UsageService) RecordTokenUsage(model string, inputTokens, outputTokens int) error {
	stats, err := s.loadOrInitStats()
	if err != nil {
		return err
	}

	if inputTokens > 0 {
		stats.ProviderStats[model+":input"] += inputTokens
	}
	if outputTokens > 0 {
		stats.ProviderStats[model+":output"] += outputTokens
	}

	return s.repo.UpdateUsage(*stats)
}

// GetUsage returns the current usage statistics.
func (s *UsageService) GetUsage() (*domain.UsageStats, error) {
	stats, err := s.repo.LoadUsage()
	if err != nil || stats == nil {
		// Return empty stats if no usage file exists
		return &domain.UsageStats{ProviderStats: make(map[string]int)}, nil
	}
	return stats, nil
}

// GetTotalTokens returns the total token count across all providers.
func (s *UsageService) GetTotalTokens() (int, error) {
	stats, err := s.repo.LoadUsage()
	if err != nil || stats == nil {
		// Return 0 if no usage file exists or is invalid
		return 0, nil
	}

	total := 0
	for _, count := range stats.ProviderStats {
		total += count
	}
	return total, nil
}

func (s *UsageService) loadOrInitStats() (*domain.UsageStats, error) {
	stats, err := s.repo.LoadUsage()
	if err != nil || stats == nil {
		stats = &domain.UsageStats{
			ProviderStats: make(map[string]int),
		}
	}
	if stats.ProviderStats == nil {
		stats.ProviderStats = make(map[string]int)
	}
	return stats, nil
}
