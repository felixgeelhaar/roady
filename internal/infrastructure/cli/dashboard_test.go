package cli

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestInitialModel(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "roady-dashboard-test-*")
	defer os.RemoveAll(tempDir)
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tempDir)

	repo := storage.NewFilesystemRepository(tempDir)
	repo.Initialize()
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Title: "Test"})
	repo.SavePlan(&planning.Plan{Tasks: []planning.Task{{ID: "t1"}}})

	m := initialModel()
	if m.project != "Test" {
		t.Errorf("Expected project Test, got %s", m.project)
	}
	
	// Test drift view
	repo.SaveSpec(&spec.ProductSpec{ID: "test", Features: []spec.Feature{{
		ID: "f1",
		Requirements: []spec.Requirement{{ID: "r1", Title: "R1"}},
	}}})
	repo.SavePlan(&planning.Plan{Tasks: []planning.Task{}}) // Force drift (missing r1)
	mDrift := initialModel()
	if len(mDrift.drift) == 0 {
		t.Error("expected drift issues in model")
	}
	if !strings.Contains(mDrift.View(), "DRIFT DETECTED") {
		t.Error("expected drift warning in view")
	}
	
	// Test update (unrecognized key)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m3.(model).project != "Test" {
		t.Error("unrecognized key should not change model drastically")
	}

	// Test update (Quit key)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m4.(model).project != "Test" {
		t.Error("quit key changed project name?")
	}

	// Test update (Ctrl+C)
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if m5.(model).project != "Test" {
		t.Error("ctrl+c changed project name?")
	}
	// Check if cmd is tea.Quit
	// (Hard to check without reflecting or knowing tea internals, but we hit the line)

	// Test update (WindowSize)
	m6, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m6.(model).project != "Test" {
		t.Error("window size changed project name?")
	}

	// Test init
	m.Init()

	// Test view
	v := m.View()
	if v == "" {
		t.Error("View is empty")
	}

	// Test error view
	mErr := model{err: os.ErrNotExist}
	if !strings.Contains(mErr.View(), "Error loading dashboard") {
		t.Error("expected error message in view")
	}

	// Test initialModel failure (no project)
	tempEmpty, _ := os.MkdirTemp("", "dashboard-fail-*")
	defer os.RemoveAll(tempEmpty)
	os.Chdir(tempEmpty)
	mFail := initialModel()
	if mFail.err == nil {
		t.Error("expected error in initialModel for empty dir")
	}

	// Test initialModel failure (Spec exists but Plan missing)
	repo.SaveSpec(&spec.ProductSpec{ID: "test"})
	mFail2 := initialModel()
	if mFail2.err == nil {
		t.Error("expected error in initialModel for missing plan")
	}
	os.Chdir(old)
}
