package application_test

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/plugin"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

type MockPluginConfigRepo struct {
	configs *plugin.PluginConfigs
	err     error
}

func (m *MockPluginConfigRepo) LoadPluginConfigs() (*plugin.PluginConfigs, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.configs, nil
}

func (m *MockPluginConfigRepo) GetPluginConfig(name string) (*plugin.PluginConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	cfg := m.configs.Get(name)
	if cfg == nil {
		return nil, &pluginNotFoundError{name: name}
	}
	return cfg, nil
}

func (m *MockPluginConfigRepo) SetPluginConfig(name string, cfg plugin.PluginConfig) error {
	if m.err != nil {
		return m.err
	}
	m.configs.Set(name, cfg)
	return nil
}

type pluginNotFoundError struct{ name string }

func (e *pluginNotFoundError) Error() string { return "not found: " + e.name }

func newTestSyncService(pluginRepo application.PluginConfigRepository) (*application.SyncService, *MockRepo) {
	repo := &MockRepo{
		Spec: &spec.ProductSpec{ID: "s1"},
		Plan: &planning.Plan{
			ID:             "p1",
			ApprovalStatus: planning.ApprovalApproved,
			Tasks:          []planning.Task{{ID: "t1", Title: "Task 1"}},
		},
		State:  planning.NewExecutionState("p1"),
		Policy: &domain.PolicyConfig{MaxWIP: 5, AllowAI: true},
	}
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	taskSvc := application.NewTaskService(repo, audit, policy)
	svc := application.NewSyncServiceWithPlugins(repo, pluginRepo, taskSvc)
	return svc, repo
}

func TestSyncService_NoPluginRepo(t *testing.T) {
	repo := &MockRepo{
		Spec:   &spec.ProductSpec{ID: "s1"},
		Policy: &domain.PolicyConfig{MaxWIP: 5},
	}
	audit := application.NewAuditService(repo)
	policy := application.NewPolicyService(repo)
	taskSvc := application.NewTaskService(repo, audit, policy)
	svc := application.NewSyncService(repo, taskSvc)

	_, err := svc.SyncWithNamedPlugin("github")
	if err == nil {
		t.Error("expected error when pluginRepo is nil")
	}

	_, err = svc.ListPluginConfigs()
	if err == nil {
		t.Error("expected error when pluginRepo is nil")
	}

	_, err = svc.GetPluginConfig("github")
	if err == nil {
		t.Error("expected error when pluginRepo is nil")
	}

	err = svc.SetPluginConfig("github", plugin.PluginConfig{})
	if err == nil {
		t.Error("expected error when pluginRepo is nil")
	}
}

func TestSyncService_SyncWithNamedPlugin_NotFound(t *testing.T) {
	pluginRepo := &MockPluginConfigRepo{configs: plugin.NewPluginConfigs()}
	svc, _ := newTestSyncService(pluginRepo)

	_, err := svc.SyncWithNamedPlugin("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestSyncService_SyncWithPlugin_InvalidBinary(t *testing.T) {
	pluginRepo := &MockPluginConfigRepo{configs: plugin.NewPluginConfigs()}
	svc, _ := newTestSyncService(pluginRepo)

	_, err := svc.SyncWithPlugin("/nonexistent/binary")
	if err == nil {
		t.Error("expected error for invalid plugin binary")
	}
}

func TestSyncService_ListPluginConfigs(t *testing.T) {
	configs := plugin.NewPluginConfigs()
	configs.Set("github", plugin.PluginConfig{Binary: "/bin/gh"})
	configs.Set("jira", plugin.PluginConfig{Binary: "/bin/jira"})

	pluginRepo := &MockPluginConfigRepo{configs: configs}
	svc, _ := newTestSyncService(pluginRepo)

	names, err := svc.ListPluginConfigs()
	if err != nil {
		t.Fatalf("ListPluginConfigs: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestSyncService_GetAndSetPluginConfig(t *testing.T) {
	configs := plugin.NewPluginConfigs()
	pluginRepo := &MockPluginConfigRepo{configs: configs}
	svc, _ := newTestSyncService(pluginRepo)

	cfg := plugin.PluginConfig{Binary: "/bin/test", Config: map[string]string{"key": "val"}}
	if err := svc.SetPluginConfig("test", cfg); err != nil {
		t.Fatalf("SetPluginConfig: %v", err)
	}

	got, err := svc.GetPluginConfig("test")
	if err != nil {
		t.Fatalf("GetPluginConfig: %v", err)
	}
	if got.Binary != "/bin/test" {
		t.Errorf("binary = %s, want /bin/test", got.Binary)
	}
}
