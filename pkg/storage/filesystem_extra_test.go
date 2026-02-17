package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/billing"
	"github.com/felixgeelhaar/roady/pkg/domain/events"
	"github.com/felixgeelhaar/roady/pkg/domain/messaging"
	"github.com/felixgeelhaar/roady/pkg/domain/team"
)

func setupRepo(t *testing.T) *FilesystemRepository {
	t.Helper()
	dir := t.TempDir()
	repo := NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return repo
}

// --- Root() ---

func TestRoot(t *testing.T) {
	dir := t.TempDir()
	repo := NewFilesystemRepository(dir)
	if repo.Root() != dir {
		t.Errorf("Root() = %q, want %q", repo.Root(), dir)
	}
}

// --- LoadUsage / UpdateUsage ---

func TestUsageRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	stats := domain.UsageStats{
		TotalCommands: 42,
		LastCommandAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		ProviderStats: map[string]int{"openai-tokens": 1500, "anthropic-tokens": 800},
	}

	if err := repo.UpdateUsage(stats); err != nil {
		t.Fatalf("UpdateUsage: %v", err)
	}

	loaded, err := repo.LoadUsage()
	if err != nil {
		t.Fatalf("LoadUsage: %v", err)
	}
	if loaded.TotalCommands != 42 {
		t.Errorf("TotalCommands = %d, want 42", loaded.TotalCommands)
	}
	if loaded.ProviderStats["openai-tokens"] != 1500 {
		t.Errorf("openai-tokens = %d, want 1500", loaded.ProviderStats["openai-tokens"])
	}
}

// --- SaveWebhookConfig / LoadWebhookConfig ---

func TestWebhookConfigRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	config := &events.WebhookConfig{
		Webhooks: []events.WebhookEndpoint{
			{
				Name:         "github-hook",
				URL:          "https://example.com/webhook",
				Secret:       "s3cret",
				EventFilters: []string{"task.completed"},
				MaxRetries:   3,
			},
		},
	}

	if err := repo.SaveWebhookConfig(config); err != nil {
		t.Fatalf("SaveWebhookConfig: %v", err)
	}

	loaded, err := repo.LoadWebhookConfig()
	if err != nil {
		t.Fatalf("LoadWebhookConfig: %v", err)
	}
	if len(loaded.Webhooks) != 1 {
		t.Fatalf("got %d webhooks, want 1", len(loaded.Webhooks))
	}
	if loaded.Webhooks[0].Name != "github-hook" {
		t.Errorf("name = %q, want %q", loaded.Webhooks[0].Name, "github-hook")
	}
	if loaded.Webhooks[0].URL != "https://example.com/webhook" {
		t.Errorf("url = %q, want %q", loaded.Webhooks[0].URL, "https://example.com/webhook")
	}
}

func TestLoadWebhookConfig_NotFound(t *testing.T) {
	repo := setupRepo(t)
	_, err := repo.LoadWebhookConfig()
	if err == nil {
		t.Error("expected error loading missing webhook config")
	}
}

// --- SaveRates / LoadRates ---

func TestRatesRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	config := &billing.RateConfig{
		Currency: "USD",
		Rates: []billing.Rate{
			{ID: "senior", Name: "Senior Dev", HourlyRate: 150.0, IsDefault: true},
			{ID: "junior", Name: "Junior Dev", HourlyRate: 80.0},
		},
	}

	if err := repo.SaveRates(config); err != nil {
		t.Fatalf("SaveRates: %v", err)
	}

	loaded, err := repo.LoadRates()
	if err != nil {
		t.Fatalf("LoadRates: %v", err)
	}
	if loaded.Currency != "USD" {
		t.Errorf("currency = %q, want %q", loaded.Currency, "USD")
	}
	if len(loaded.Rates) != 2 {
		t.Fatalf("got %d rates, want 2", len(loaded.Rates))
	}
}

func TestLoadRates_NotFound(t *testing.T) {
	repo := setupRepo(t)
	loaded, err := repo.LoadRates()
	if err != nil {
		t.Fatalf("LoadRates should return empty config, got error: %v", err)
	}
	if len(loaded.Rates) != 0 {
		t.Errorf("expected empty rates, got %d", len(loaded.Rates))
	}
}

// --- SaveTimeEntries / LoadTimeEntries ---

func TestTimeEntriesRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	entries := []billing.TimeEntry{
		{
			ID:          "te-1",
			TaskID:      "task-1",
			RateID:      "senior",
			Minutes:     120,
			Description: "Backend work",
			CreatedAt:   time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC),
		},
		{
			ID:          "te-2",
			TaskID:      "task-2",
			RateID:      "junior",
			Minutes:     60,
			Description: "Frontend work",
			CreatedAt:   time.Date(2024, 3, 15, 14, 0, 0, 0, time.UTC),
		},
	}

	if err := repo.SaveTimeEntries(entries); err != nil {
		t.Fatalf("SaveTimeEntries: %v", err)
	}

	loaded, err := repo.LoadTimeEntries()
	if err != nil {
		t.Fatalf("LoadTimeEntries: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("got %d entries, want 2", len(loaded))
	}
	if loaded[0].ID != "te-1" {
		t.Errorf("first entry ID = %q, want %q", loaded[0].ID, "te-1")
	}
	if loaded[0].Minutes != 120 {
		t.Errorf("first entry minutes = %d, want 120", loaded[0].Minutes)
	}
}

func TestLoadTimeEntries_NotFound(t *testing.T) {
	repo := setupRepo(t)
	loaded, err := repo.LoadTimeEntries()
	if err != nil {
		t.Fatalf("LoadTimeEntries should return empty slice, got error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty entries, got %d", len(loaded))
	}
}

// --- SaveMessagingConfig / LoadMessagingConfig ---

func TestMessagingConfigRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	config := &messaging.MessagingConfig{
		Adapters: []messaging.AdapterConfig{
			{
				Type: "slack",
				Name: "team-slack",
			},
		},
	}

	if err := repo.SaveMessagingConfig(config); err != nil {
		t.Fatalf("SaveMessagingConfig: %v", err)
	}

	loaded, err := repo.LoadMessagingConfig()
	if err != nil {
		t.Fatalf("LoadMessagingConfig: %v", err)
	}
	if len(loaded.Adapters) != 1 {
		t.Fatalf("got %d adapters, want 1", len(loaded.Adapters))
	}
	if loaded.Adapters[0].Type != "slack" {
		t.Errorf("adapter type = %q, want %q", loaded.Adapters[0].Type, "slack")
	}
}

func TestLoadMessagingConfig_NotFound(t *testing.T) {
	repo := setupRepo(t)
	loaded, err := repo.LoadMessagingConfig()
	if err != nil {
		t.Fatalf("LoadMessagingConfig should return empty config, got error: %v", err)
	}
	if len(loaded.Adapters) != 0 {
		t.Errorf("expected empty adapters, got %d", len(loaded.Adapters))
	}
}

// --- SaveTeam / LoadTeam ---

func TestTeamRoundtrip(t *testing.T) {
	repo := setupRepo(t)

	cfg := &team.TeamConfig{
		Members: []team.Member{
			{Name: "alice", Role: team.RoleAdmin},
			{Name: "bob", Role: team.RoleMember},
		},
	}

	if err := repo.SaveTeam(cfg); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}

	loaded, err := repo.LoadTeam()
	if err != nil {
		t.Fatalf("LoadTeam: %v", err)
	}
	if len(loaded.Members) != 2 {
		t.Fatalf("got %d members, want 2", len(loaded.Members))
	}
	if loaded.Members[0].Name != "alice" {
		t.Errorf("first member name = %q, want %q", loaded.Members[0].Name, "alice")
	}
}

func TestLoadTeam_NotFound(t *testing.T) {
	repo := setupRepo(t)
	loaded, err := repo.LoadTeam()
	if err != nil {
		t.Fatalf("LoadTeam should return empty config, got error: %v", err)
	}
	if len(loaded.Members) != 0 {
		t.Errorf("expected empty members, got %d", len(loaded.Members))
	}
}

// --- LoadSpecLock not found ---

func TestLoadSpecLock_NotFound(t *testing.T) {
	repo := setupRepo(t)
	_, err := repo.LoadSpecLock()
	if err == nil {
		t.Error("expected error loading missing spec lock")
	}
}

// --- GetDependency not found ---

func TestGetDependency_NotFound(t *testing.T) {
	repo := setupRepo(t)
	dep, err := repo.GetDependency("nonexistent")
	if err != nil {
		t.Fatalf("GetDependency: %v", err)
	}
	if dep != nil {
		t.Errorf("expected nil for nonexistent dependency, got %+v", dep)
	}
}

// --- LoadRange (event store) ---

func TestFileEventStore_LoadRange(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileEventStore(dir)
	if err != nil {
		t.Fatalf("NewFileEventStore: %v", err)
	}

	// Append events at different times
	now := time.Now()
	for i := 0; i < 5; i++ {
		evt := &events.BaseEvent{
			Type:      "test.event",
			Timestamp: now.Add(time.Duration(i) * time.Hour),
		}
		if err := store.Append(evt); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Load range: events 1-3 (hours 1-3)
	from := now.Add(1 * time.Hour)
	to := now.Add(3 * time.Hour)
	result, err := store.LoadRange(from, to)
	if err != nil {
		t.Fatalf("LoadRange: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("got %d events, want 3 (hours 1,2,3)", len(result))
	}
}

func TestFileEventStore_LoadRange_Empty(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileEventStore(dir)
	if err != nil {
		t.Fatalf("NewFileEventStore: %v", err)
	}

	from := time.Now()
	to := from.Add(time.Hour)
	result, err := store.LoadRange(from, to)
	if err != nil {
		t.Fatalf("LoadRange: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d events, want 0", len(result))
	}
}

// --- ResolvePath edge cases ---

func TestResolvePath_ValidFiles(t *testing.T) {
	repo := setupRepo(t)
	tests := []string{
		SpecFile,
		PlanFile,
		StateFile,
		PolicyFile,
		EventsFile,
		UsageFile,
		WebhookFile,
		RatesFile,
		TimeEntriesFile,
		MessagingFile,
		TeamFile,
	}
	for _, f := range tests {
		t.Run(f, func(t *testing.T) {
			path, err := repo.ResolvePath(f)
			if err != nil {
				t.Errorf("ResolvePath(%q) failed: %v", f, err)
			}
			expected := filepath.Join(repo.root, RoadyDir, f)
			if path != expected {
				t.Errorf("got %q, want %q", path, expected)
			}
		})
	}
}

// --- File permission checks ---

func TestSaveCreatesFileWithRestrictedPermissions(t *testing.T) {
	repo := setupRepo(t)

	config := &billing.RateConfig{Currency: "EUR"}
	if err := repo.SaveRates(config); err != nil {
		t.Fatalf("SaveRates: %v", err)
	}

	path, _ := repo.ResolvePath(RatesFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// Check file permissions are restrictive (0600)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}
