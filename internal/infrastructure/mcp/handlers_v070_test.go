package mcp

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestServer_HandleOrgPolicy(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleOrgPolicy(context.Background(), OrgPolicyArgs{})
	if err != nil {
		t.Fatalf("handleOrgPolicy failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestServer_HandleOrgPolicyWithPath(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 3, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleOrgPolicy(context.Background(), OrgPolicyArgs{ProjectPath: root})
	if err != nil {
		t.Fatalf("handleOrgPolicy with path failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestServer_HandleOrgDetectDrift(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleOrgDetectDrift(context.Background(), GetSpecArgs{})
	if err != nil {
		t.Fatalf("handleOrgDetectDrift failed: %v", err)
	}

	_ = result
}

func TestServer_HandlePluginList(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handlePluginList(context.Background(), GetSpecArgs{})
	if err != nil {
		t.Fatalf("handlePluginList failed: %v", err)
	}

	_ = result
}

func TestServer_HandlePluginValidate(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	_, _ = server.handlePluginValidate(context.Background(), PluginValidateArgs{Name: "nonexistent"})
}

func TestServer_HandlePluginStatus(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	t.Run("AllPlugins", func(t *testing.T) {
		result, err := server.handlePluginStatus(context.Background(), PluginStatusArgs{})
		if err != nil {
			t.Fatalf("handlePluginStatus failed: %v", err)
		}
		_ = result
	})

	t.Run("SpecificPlugin", func(t *testing.T) {
		_, _ = server.handlePluginStatus(context.Background(), PluginStatusArgs{Name: "nonexistent"})
	})
}

func TestServer_HandleMessagingList(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleMessagingList(context.Background(), GetSpecArgs{})
	if err != nil {
		t.Fatalf("handleMessagingList failed: %v", err)
	}

	_ = result
}

func TestServer_HandleTeamList(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleTeamList(context.Background(), GetSpecArgs{})
	if err != nil {
		t.Fatalf("handleTeamList failed: %v", err)
	}

	_ = result
}

func TestServer_HandleTeamAdd(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleTeamAdd(context.Background(), TeamAddArgs{Name: "Alice", Role: "admin"})
	if err != nil {
		t.Fatalf("handleTeamAdd failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleTeamAdd_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	t.Run("EmptyName", func(t *testing.T) {
		_, err := server.handleTeamAdd(context.Background(), TeamAddArgs{Name: "", Role: "admin"})
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("EmptyRole", func(t *testing.T) {
		_, err := server.handleTeamAdd(context.Background(), TeamAddArgs{Name: "Bob", Role: ""})
		if err == nil {
			t.Error("expected error for empty role")
		}
	})
}

func TestServer_HandleTeamRemove(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	// First add a member
	_, _ = server.handleTeamAdd(context.Background(), TeamAddArgs{Name: "Charlie", Role: "member"})

	// Then remove them
	result, err := server.handleTeamRemove(context.Background(), TeamRemoveArgs{Name: "Charlie"})
	if err != nil {
		t.Fatalf("handleTeamRemove failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleTeamRemove_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	_, err = server.handleTeamRemove(context.Background(), TeamRemoveArgs{Name: ""})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestServer_HandleRateList(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleRateList(context.Background(), RateListArgs{})
	if err != nil {
		t.Fatalf("handleRateList failed: %v", err)
	}

	_ = result
}

func TestServer_HandleRateAdd(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleRateAdd(context.Background(), RateAddArgs{
		ID:         "senior",
		Name:       "Senior Developer",
		HourlyRate: 150.0,
		IsDefault:  true,
	})
	if err != nil {
		t.Fatalf("handleRateAdd failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleRateAdd_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	t.Run("EmptyID", func(t *testing.T) {
		_, err := server.handleRateAdd(context.Background(), RateAddArgs{
			ID:   "",
			Name: "Test",
		})
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		_, err := server.handleRateAdd(context.Background(), RateAddArgs{
			ID:   "test",
			Name: "",
		})
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestServer_HandleRateRemove(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	// First add a rate
	_, _ = server.handleRateAdd(context.Background(), RateAddArgs{
		ID:         "junior",
		Name:       "Junior Developer",
		HourlyRate: 75.0,
	})

	// Then remove it
	result, err := server.handleRateRemove(context.Background(), RateRemoveArgs{ID: "junior"})
	if err != nil {
		t.Fatalf("handleRateRemove failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleRateRemove_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	_, err = server.handleRateRemove(context.Background(), RateRemoveArgs{ID: ""})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestServer_HandleRateSetDefault(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	// First add a rate
	_, _ = server.handleRateAdd(context.Background(), RateAddArgs{
		ID:         "mid",
		Name:       "Mid Developer",
		HourlyRate: 100.0,
	})

	// Set it as default
	result, err := server.handleRateSetDefault(context.Background(), RateSetDefaultArgs{ID: "mid"})
	if err != nil {
		t.Fatalf("handleRateSetDefault failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleRateSetDefault_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	_, err = server.handleRateSetDefault(context.Background(), RateSetDefaultArgs{ID: ""})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestServer_HandleRateTax(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleRateTax(context.Background(), RateTaxArgs{
		Name:     "VAT",
		Percent:  20.0,
		Included: false,
	})
	if err != nil {
		t.Fatalf("handleRateTax failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleRateTax_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	t.Run("EmptyName", func(t *testing.T) {
		_, err := server.handleRateTax(context.Background(), RateTaxArgs{
			Name:    "",
			Percent: 20.0,
		})
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("NegativePercent", func(t *testing.T) {
		_, err := server.handleRateTax(context.Background(), RateTaxArgs{
			Name:    "VAT",
			Percent: -10.0,
		})
		if err == nil {
			t.Error("expected error for negative percent")
		}
	})

	t.Run("PercentOver100", func(t *testing.T) {
		_, err := server.handleRateTax(context.Background(), RateTaxArgs{
			Name:    "VAT",
			Percent: 150.0,
		})
		if err == nil {
			t.Error("expected error for percent over 100")
		}
	})
}

func TestServer_HandleTaskLogTime(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	// Add a rate first
	_, _ = server.handleRateAdd(context.Background(), RateAddArgs{
		ID:         "default",
		Name:       "Default Rate",
		HourlyRate: 100.0,
		IsDefault:  true,
	})

	result, err := server.handleTaskLogTime(context.Background(), TaskLogTimeArgs{
		TaskID:      "task-1",
		Minutes:     60,
		RateID:      "",
		Description: "Worked on feature",
	})
	if err != nil {
		t.Fatalf("handleTaskLogTime failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleTaskLogTime_Validation(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	t.Run("EmptyTaskID", func(t *testing.T) {
		_, err := server.handleTaskLogTime(context.Background(), TaskLogTimeArgs{
			TaskID:  "",
			Minutes: 60,
		})
		if err == nil {
			t.Error("expected error for empty task ID")
		}
	})

	t.Run("ZeroMinutes", func(t *testing.T) {
		_, err := server.handleTaskLogTime(context.Background(), TaskLogTimeArgs{
			TaskID:  "task-1",
			Minutes: 0,
		})
		if err == nil {
			t.Error("expected error for zero minutes")
		}
	})

	t.Run("NegativeMinutes", func(t *testing.T) {
		_, err := server.handleTaskLogTime(context.Background(), TaskLogTimeArgs{
			TaskID:  "task-1",
			Minutes: -30,
		})
		if err == nil {
			t.Error("expected error for negative minutes")
		}
	})
}

func TestServer_HandleCostReport(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleCostReport(context.Background(), CostReportArgs{
		TaskID: "",
		Period: "",
		Format: "text",
	})
	if err != nil {
		t.Fatalf("handleCostReport failed: %v", err)
	}

	_ = result
}

func TestServer_HandleCostBudget(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleCostBudget(context.Background(), CostBudgetArgs{})
	if err != nil {
		t.Fatalf("handleCostBudget failed: %v", err)
	}

	_ = result
}

func TestServer_HandleAssignTask(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	spec := &spec.ProductSpec{
		ID:    "assign-spec",
		Title: "Assign Project",
		Features: []spec.Feature{
			{ID: "feat-1", Title: "Feature 1"},
		},
	}
	if err := repo.SaveSpec(spec); err != nil {
		t.Fatalf("save spec: %v", err)
	}

	plan := &planning.Plan{
		ID:             "plan-assign",
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalApproved,
		Tasks: []planning.Task{
			{ID: "task-assign", FeatureID: "feat-1", Title: "Assign Task"},
		},
	}
	if err := repo.SavePlan(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	state := planning.NewExecutionState(plan.ID)
	if err := repo.SaveState(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	result, err := server.handleAssignTask(context.Background(), AssignTaskArgs{TaskID: "task-assign", Assignee: "alice"})
	if err != nil {
		t.Fatalf("handleAssignTask failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestServer_HandleSync_Error(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	_, err = server.handleSync(context.Background(), SyncArgs{PluginPath: "/nonexistent/binary"})
	if err == nil {
		t.Error("expected sync error for missing plugin")
	}
}

func TestServer_HandleWorkspacePushPull_Error(t *testing.T) {
	root := t.TempDir()
	repo := storage.NewFilesystemRepository(root)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("initialize repo: %v", err)
	}

	if err := repo.SavePolicy(&domain.PolicyConfig{MaxWIP: 2, AllowAI: true}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	server, err := NewServer(root)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	if _, err := server.handleWorkspacePush(context.Background(), WorkspacePushArgs{}); err == nil {
		t.Error("expected error for workspace push without git")
	}
	if _, err := server.handleWorkspacePull(context.Background(), WorkspacePullArgs{}); err == nil {
		t.Error("expected error for workspace pull without git")
	}
}

func TestServer_HandleSmartDecompose_NoService(t *testing.T) {
	server := &Server{root: t.TempDir()}
	_, err := server.handleSmartDecompose(context.Background(), SmartDecomposeArgs{})
	if err == nil {
		t.Error("expected error when AI service is nil")
	}
}
