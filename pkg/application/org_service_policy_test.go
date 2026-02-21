package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/org"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"gopkg.in/yaml.v3"
)

func TestLoadMergedPolicy_OrgOnly(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project1")
	roadyDir := filepath.Join(root, ".roady")

	if err := os.MkdirAll(roadyDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".roady"), 0700); err != nil {
		t.Fatal(err)
	}

	orgConfig := org.OrgConfig{
		Name: "test-org",
		SharedPolicy: &org.SharedPolicy{
			MaxWIP:     3,
			AllowAI:    true,
			TokenLimit: 5000,
		},
	}
	data, _ := yaml.Marshal(orgConfig)
	if err := os.WriteFile(filepath.Join(roadyDir, "org.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}

	svc := application.NewOrgService(root)
	merged, err := svc.LoadMergedPolicy(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.MaxWIP != 3 {
		t.Errorf("expected MaxWIP=3, got %d", merged.MaxWIP)
	}
	if !merged.AllowAI {
		t.Error("expected AllowAI=true")
	}
	if merged.TokenLimit != 5000 {
		t.Errorf("expected TokenLimit=5000, got %d", merged.TokenLimit)
	}
}

func TestLoadMergedPolicy_ProjectOverrides(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project1")
	roadyDir := filepath.Join(root, ".roady")
	projectRoadyDir := filepath.Join(projectDir, ".roady")

	_ = os.MkdirAll(roadyDir, 0700)
	_ = os.MkdirAll(projectRoadyDir, 0700)

	orgConfig := org.OrgConfig{
		Name: "test-org",
		SharedPolicy: &org.SharedPolicy{
			MaxWIP:     3,
			AllowAI:    false,
			TokenLimit: 5000,
		},
	}
	data, _ := yaml.Marshal(orgConfig)
	_ = os.WriteFile(filepath.Join(roadyDir, "org.yaml"), data, 0600)

	projectPolicy := policy.PolicyConfig{
		MaxWIP:     5,
		TokenLimit: 10000,
	}
	policyData, _ := yaml.Marshal(projectPolicy)
	_ = os.WriteFile(filepath.Join(projectRoadyDir, "policy.yaml"), policyData, 0600)

	svc := application.NewOrgService(root)
	merged, err := svc.LoadMergedPolicy(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.MaxWIP != 5 {
		t.Errorf("expected MaxWIP=5 (project override), got %d", merged.MaxWIP)
	}
	if merged.TokenLimit != 10000 {
		t.Errorf("expected TokenLimit=10000 (project override), got %d", merged.TokenLimit)
	}
}

func TestLoadMergedPolicy_NoOrg(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project1")
	projectRoadyDir := filepath.Join(projectDir, ".roady")
	_ = os.MkdirAll(projectRoadyDir, 0700)

	projectPolicy := policy.PolicyConfig{MaxWIP: 2}
	policyData, _ := yaml.Marshal(projectPolicy)
	_ = os.WriteFile(filepath.Join(projectRoadyDir, "policy.yaml"), policyData, 0600)

	svc := application.NewOrgService(root)
	merged, err := svc.LoadMergedPolicy(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.MaxWIP != 2 {
		t.Errorf("expected MaxWIP=2, got %d", merged.MaxWIP)
	}
}
