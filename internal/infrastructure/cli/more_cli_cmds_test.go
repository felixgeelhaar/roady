package cli

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain"
	domainmsg "github.com/felixgeelhaar/roady/pkg/domain/messaging"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// ---------------------------------------------------------------------------
// Messaging command tests
// ---------------------------------------------------------------------------

func TestMessagingListCmd_NoConfig(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	output := captureStdout(t, func() {
		if err := messagingListCmd.RunE(messagingListCmd, []string{}); err != nil {
			t.Fatalf("messaging list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No messaging adapters configured") {
		t.Fatalf("expected no-config message, got:\n%s", output)
	}
}

func TestMessagingListCmd_WithAdapters(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	cfg := &domainmsg.MessagingConfig{
		Adapters: []domainmsg.AdapterConfig{
			{
				Name:    "my-hook",
				Type:    "webhook",
				URL:     "http://example.com/hook",
				Enabled: true,
			},
		},
	}
	if err := repo.SaveMessagingConfig(cfg); err != nil {
		t.Fatalf("save messaging config: %v", err)
	}

	output := captureStdout(t, func() {
		if err := messagingListCmd.RunE(messagingListCmd, []string{}); err != nil {
			t.Fatalf("messaging list failed: %v", err)
		}
	})

	if !strings.Contains(output, "my-hook") {
		t.Fatalf("expected adapter name in output, got:\n%s", output)
	}
}

func TestMessagingAddCmd(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	output := captureStdout(t, func() {
		if err := messagingAddCmd.RunE(messagingAddCmd, []string{"my-hook", "webhook", "http://example.com"}); err != nil {
			t.Fatalf("messaging add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Added webhook adapter") {
		t.Fatalf("expected add confirmation, got:\n%s", output)
	}

	// Verify the adapter was persisted
	loaded, err := repo.LoadMessagingConfig()
	if err != nil {
		t.Fatalf("load messaging config: %v", err)
	}
	if len(loaded.Adapters) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(loaded.Adapters))
	}
	if loaded.Adapters[0].Name != "my-hook" {
		t.Fatalf("expected adapter name 'my-hook', got %q", loaded.Adapters[0].Name)
	}
}

func TestMessagingAddCmd_Duplicate(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	cfg := &domainmsg.MessagingConfig{
		Adapters: []domainmsg.AdapterConfig{
			{
				Name:    "my-hook",
				Type:    "webhook",
				URL:     "http://example.com/existing",
				Enabled: true,
			},
		},
	}
	if err := repo.SaveMessagingConfig(cfg); err != nil {
		t.Fatalf("save messaging config: %v", err)
	}

	err := messagingAddCmd.RunE(messagingAddCmd, []string{"my-hook", "webhook", "http://example.com"})
	if err == nil {
		t.Fatal("expected error for duplicate adapter")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Doctor command tests
// ---------------------------------------------------------------------------

func TestDoctorCmd_Uninitialized(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	// Do NOT initialize the repo -- doctor should detect the missing .roady dir.
	err := doctorCmd.RunE(doctorCmd, []string{})
	if err == nil {
		t.Fatal("expected doctor to report issues for uninitialized directory")
	}
	if !strings.Contains(err.Error(), "doctor found issues") {
		t.Fatalf("expected 'doctor found issues' error, got: %v", err)
	}
}

func TestDoctorCmd_AllPass(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	_ = repo.SaveSpec(&spec.ProductSpec{ID: "spec-1", Title: "Project"})
	_ = repo.SavePlan(&planning.Plan{ID: "p1"})
	state := planning.NewExecutionState("p1")
	state.ProjectID = "p1"
	_ = repo.SaveState(state)
	_ = repo.SavePolicy(&domain.PolicyConfig{})

	audit := application.NewAuditService(repo)
	if err := audit.Log("spec.update", "tester", nil); err != nil {
		t.Fatalf("log event: %v", err)
	}

	output := captureStdout(t, func() {
		if err := doctorCmd.RunE(doctorCmd, []string{}); err != nil {
			t.Fatalf("doctor failed: %v", err)
		}
	})

	if !strings.Contains(output, "Everything looks good") {
		t.Fatalf("expected all-pass output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Org command tests
// ---------------------------------------------------------------------------

func TestOrgPolicyCmd_NoOrg(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	repo := storage.NewFilesystemRepository(".")
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	// orgPolicyCmd may return an error or output defaults -- we only verify
	// it does not panic.
	_ = orgPolicyCmd.RunE(orgPolicyCmd, []string{"."})
}

func TestOrgDriftCmd_NoProjects(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	output := captureStdout(t, func() {
		if err := orgDriftCmd.RunE(orgDriftCmd, []string{"."}); err != nil {
			t.Fatalf("org drift failed: %v", err)
		}
	})

	if !strings.Contains(output, "No Roady projects found") {
		t.Fatalf("expected no-projects message, got:\n%s", output)
	}
}

func TestOrgDriftCmd_WithProject(t *testing.T) {
	root, cleanup := withTempDir(t)
	defer cleanup()

	project := root + "/project-a"
	repo := storage.NewFilesystemRepository(project)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	_ = repo.SaveSpec(&spec.ProductSpec{
		ID:    "s1",
		Title: "Alpha",
		Features: []spec.Feature{
			{ID: "f1", Title: "F1"},
		},
	})
	_ = repo.SavePlan(&planning.Plan{
		ID: "p1",
		Tasks: []planning.Task{
			{ID: "task-f1", FeatureID: "f1", Title: "T1"},
		},
	})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	output := captureStdout(t, func() {
		if err := orgDriftCmd.RunE(orgDriftCmd, []string{root}); err != nil {
			t.Fatalf("org drift failed: %v", err)
		}
	})

	if !strings.Contains(output, "Cross-Project Drift Report") {
		t.Fatalf("expected drift report header, got:\n%s", output)
	}
}

func TestOrgStatusCmd_NoProjects(t *testing.T) {
	_, cleanup := withTempDir(t)
	defer cleanup()

	output := captureStdout(t, func() {
		_ = orgStatusCmd.RunE(orgStatusCmd, []string{"."})
	})

	if !strings.Contains(output, "No Roady projects found") {
		t.Fatalf("expected no-projects message, got:\n%s", output)
	}
}

func TestOrgStatusCmd_JSON(t *testing.T) {
	root, cleanup := withTempDir(t)
	defer cleanup()

	project := root + "/project-a"
	repo := storage.NewFilesystemRepository(project)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	_ = repo.SaveSpec(&spec.ProductSpec{ID: "s1", Title: "Alpha"})
	_ = repo.SavePlan(&planning.Plan{ID: "p1"})
	_ = repo.SaveState(planning.NewExecutionState("p1"))

	orgJSON = true
	defer func() { orgJSON = false }()

	output := captureStdout(t, func() {
		if err := orgStatusCmd.RunE(orgStatusCmd, []string{root}); err != nil {
			t.Fatalf("org status json failed: %v", err)
		}
	})

	if !strings.Contains(output, "total_projects") {
		t.Fatalf("expected JSON field 'total_projects', got:\n%s", output)
	}
}
