package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestValidateProjectName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty is root project", "", false},
		{"simple", "auth", false},
		{"with dash", "feature-auth", false},
		{"with underscore", "feature_auth", false},
		{"with dot", "v1.2", false},
		{"alphanum", "abc123", false},

		{"dot reserved", ".", true},
		{"dotdot reserved", "..", true},
		{"projects literal reserved", "projects", true},
		{"path separator", "foo/bar", true},
		{"backslash separator", "foo\\bar", true},
		{"uppercase rejected", "Auth", true},
		{"leading dash rejected", "-auth", true},
		{"leading dot rejected", ".auth", true},
		{"too long", strings.Repeat("a", 65), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProjectName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateProjectName(%q) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestNewFilesystemRepositoryForProject_Invalid(t *testing.T) {
	_, err := NewFilesystemRepositoryForProject("/tmp", "Bad/Name")
	if err == nil {
		t.Fatal("expected error for invalid project name")
	}
}

func TestFilesystemRepository_SubProjectPaths(t *testing.T) {
	root := t.TempDir()

	rootRepo := NewFilesystemRepository(root)
	if rootRepo.IsSubProject() {
		t.Error("root-project repo reports IsSubProject true")
	}
	if got := rootRepo.SubProject(); got != "" {
		t.Errorf("root SubProject = %q, want empty", got)
	}
	if got := rootRepo.ProjectBase(); got != filepath.Join(root, RoadyDir) {
		t.Errorf("root ProjectBase = %q, want %q", got, filepath.Join(root, RoadyDir))
	}

	subRepo, err := NewFilesystemRepositoryForProject(root, "feature-auth")
	if err != nil {
		t.Fatalf("NewFilesystemRepositoryForProject: %v", err)
	}
	if !subRepo.IsSubProject() {
		t.Error("sub-project repo reports IsSubProject false")
	}
	if got := subRepo.SubProject(); got != "feature-auth" {
		t.Errorf("sub SubProject = %q, want feature-auth", got)
	}
	wantBase := filepath.Join(root, RoadyDir, ProjectsDir, "feature-auth")
	if got := subRepo.ProjectBase(); got != wantBase {
		t.Errorf("sub ProjectBase = %q, want %q", got, wantBase)
	}

	resolved, err := subRepo.ResolvePath(SpecFile)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want := filepath.Join(wantBase, SpecFile)
	if resolved != want {
		t.Errorf("ResolvePath = %q, want %q", resolved, want)
	}

	// Traversal must still be rejected on sub-project repos.
	if _, err := subRepo.ResolvePath("../" + SpecFile); err == nil {
		t.Error("expected error for traversal out of sub-project")
	}
	if _, err := subRepo.ResolvePath("sub/nested.yaml"); err == nil {
		t.Error("expected error for nested filename under sub-project")
	}
}

func TestFilesystemRepository_SubProject_InitializeAndIsInitialized(t *testing.T) {
	root := t.TempDir()
	subRepo, err := NewFilesystemRepositoryForProject(root, "billing")
	if err != nil {
		t.Fatal(err)
	}

	if subRepo.IsInitialized() {
		t.Fatal("expected IsInitialized=false before Initialize")
	}
	if err := subRepo.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !subRepo.IsInitialized() {
		t.Fatal("expected IsInitialized=true after Initialize")
	}

	wantPath := filepath.Join(root, RoadyDir, ProjectsDir, "billing")
	if fi, err := os.Stat(wantPath); err != nil || !fi.IsDir() {
		t.Fatalf("expected sub-project dir at %q, err=%v", wantPath, err)
	}
}

func TestFilesystemRepository_SubProject_Isolation(t *testing.T) {
	root := t.TempDir()

	// Set up: root project + two sub-projects, all with distinct specs.
	rootRepo := NewFilesystemRepository(root)
	if err := rootRepo.Initialize(); err != nil {
		t.Fatal(err)
	}
	authRepo, err := NewFilesystemRepositoryForProject(root, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if err := authRepo.Initialize(); err != nil {
		t.Fatal(err)
	}
	billingRepo, err := NewFilesystemRepositoryForProject(root, "billing")
	if err != nil {
		t.Fatal(err)
	}
	if err := billingRepo.Initialize(); err != nil {
		t.Fatal(err)
	}

	rootSpec := &spec.ProductSpec{ID: "root-spec"}
	authSpec := &spec.ProductSpec{ID: "auth-spec"}
	billingSpec := &spec.ProductSpec{ID: "billing-spec"}

	if err := rootRepo.SaveSpec(rootSpec); err != nil {
		t.Fatal(err)
	}
	if err := authRepo.SaveSpec(authSpec); err != nil {
		t.Fatal(err)
	}
	if err := billingRepo.SaveSpec(billingSpec); err != nil {
		t.Fatal(err)
	}

	got1, err := rootRepo.LoadSpec()
	if err != nil {
		t.Fatal(err)
	}
	if got1.ID != "root-spec" {
		t.Errorf("rootRepo LoadSpec ID = %q, want root-spec", got1.ID)
	}
	got2, err := authRepo.LoadSpec()
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != "auth-spec" {
		t.Errorf("authRepo LoadSpec ID = %q, want auth-spec", got2.ID)
	}
	got3, err := billingRepo.LoadSpec()
	if err != nil {
		t.Fatal(err)
	}
	if got3.ID != "billing-spec" {
		t.Errorf("billingRepo LoadSpec ID = %q, want billing-spec", got3.ID)
	}

	// Verify the files actually live where expected on disk.
	for _, want := range []string{
		filepath.Join(root, RoadyDir, SpecFile),
		filepath.Join(root, RoadyDir, ProjectsDir, "auth", SpecFile),
		filepath.Join(root, RoadyDir, ProjectsDir, "billing", SpecFile),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("expected spec file at %q: %v", want, err)
		}
	}
}
