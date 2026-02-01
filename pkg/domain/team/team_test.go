package team

import "testing"

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role  Role
		valid bool
	}{
		{RoleAdmin, true},
		{RoleMember, true},
		{RoleViewer, true},
		{Role("superadmin"), false},
		{Role(""), false},
	}
	for _, tt := range tests {
		if got := tt.role.IsValid(); got != tt.valid {
			t.Errorf("Role(%q).IsValid() = %v, want %v", tt.role, got, tt.valid)
		}
	}
}

func TestRole_Permissions(t *testing.T) {
	if !RoleAdmin.CanManageTeam() {
		t.Error("admin should be able to manage team")
	}
	if RoleMember.CanManageTeam() {
		t.Error("member should not manage team")
	}
	if !RoleMember.CanTransitionTasks() {
		t.Error("member should transition tasks")
	}
	if RoleViewer.CanTransitionTasks() {
		t.Error("viewer should not transition tasks")
	}
	if RoleViewer.CanEditPlan() {
		t.Error("viewer should not edit plan")
	}
}

func TestTeamConfig_AddMember(t *testing.T) {
	cfg := &TeamConfig{}

	if err := cfg.AddMember("alice", RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(cfg.Members))
	}

	// Update existing
	if err := cfg.AddMember("alice", RoleMember); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Members) != 1 {
		t.Fatalf("expected 1 member after update, got %d", len(cfg.Members))
	}
	if cfg.Members[0].Role != RoleMember {
		t.Errorf("expected role member, got %s", cfg.Members[0].Role)
	}

	// Invalid role
	if err := cfg.AddMember("bob", Role("invalid")); err == nil {
		t.Error("expected error for invalid role")
	}

	// Empty name
	if err := cfg.AddMember("", RoleMember); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestTeamConfig_RemoveMember(t *testing.T) {
	cfg := &TeamConfig{Members: []Member{{Name: "alice", Role: RoleAdmin}}}

	if err := cfg.RemoveMember("alice"); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(cfg.Members))
	}

	if err := cfg.RemoveMember("nobody"); err == nil {
		t.Error("expected error removing non-existent member")
	}
}

func TestTeamConfig_FindMember(t *testing.T) {
	cfg := &TeamConfig{Members: []Member{{Name: "alice", Role: RoleAdmin}}}

	m := cfg.FindMember("alice")
	if m == nil {
		t.Fatal("expected to find alice")
	}
	if m.Role != RoleAdmin {
		t.Errorf("expected admin, got %s", m.Role)
	}

	if cfg.FindMember("nobody") != nil {
		t.Error("expected nil for unknown member")
	}
}
