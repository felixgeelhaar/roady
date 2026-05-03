package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/storage"
)

func TestNotifyOps_AddListRemove(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init: %v", err)
	}

	out := new(bytes.Buffer)
	if err := notifyAdd(out, "ci", "webhook", "https://example.test"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out.String(), `Added webhook adapter "ci"`) {
		t.Errorf("unexpected add output: %q", out.String())
	}

	out.Reset()
	if err := notifyList(out); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), `"ci"`) {
		t.Errorf("expected adapter in list output: %q", out.String())
	}

	out.Reset()
	if err := notifyRemove(out, "ci"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !strings.Contains(out.String(), `Removed adapter "ci"`) {
		t.Errorf("unexpected remove output: %q", out.String())
	}

	out.Reset()
	if err := notifyList(out); err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if !strings.Contains(out.String(), "No notification adapters configured.") {
		t.Errorf("expected empty-state output, got %q", out.String())
	}
}

func TestNotifyOps_DuplicateAndMissing(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	_ = os.Chdir(dir)

	repo := storage.NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init: %v", err)
	}

	out := new(bytes.Buffer)
	if err := notifyAdd(out, "ci", "webhook", "https://example.test"); err != nil {
		t.Fatal(err)
	}
	if err := notifyAdd(out, "ci", "webhook", "https://other.test"); err == nil {
		t.Error("expected duplicate error on second add")
	}

	if err := notifyRemove(out, "does-not-exist"); err == nil {
		t.Error("expected error removing missing adapter")
	}

	if err := notifyTest(out, "does-not-exist"); err == nil {
		t.Error("expected error testing missing adapter")
	}
}

func TestMessagingDeprecation_DelegatesToNotify(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	_ = os.Chdir(dir)

	repo := storage.NewFilesystemRepository(dir)
	if err := repo.Initialize(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Use the legacy command path; it must persist config to the same place
	// the new notify path reads from.
	if err := messagingAddCmd.RunE(messagingAddCmd, []string{"ci", "webhook", "https://example.test"}); err != nil {
		t.Fatalf("messaging add: %v", err)
	}

	out := new(bytes.Buffer)
	if err := notifyList(out); err != nil {
		t.Fatalf("notify list: %v", err)
	}
	if !strings.Contains(out.String(), `"ci"`) {
		t.Errorf("expected legacy-added adapter to be visible via notify list, got %q", out.String())
	}
}
