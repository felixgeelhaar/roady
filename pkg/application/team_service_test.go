package application_test

import (
	"os"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/team"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// newTeamTestHarness creates a temp directory, initializes a repo, and returns
// the service, repo, audit mock, and a cleanup function.
// It reuses the mockAuditLogger defined in billing_service_test.go.
func newTeamTestHarness(t *testing.T) (*application.TeamService, *storage.FilesystemRepository, *mockAuditLogger, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "roady-team-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	repo := storage.NewFilesystemRepository(tmpDir)
	if err := repo.Initialize(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("initialize repo: %v", err)
	}
	audit := newTestAudit()
	svc := application.NewTeamService(repo, audit)
	cleanup := func() { os.RemoveAll(tmpDir) }
	return svc, repo, audit, cleanup
}

func TestTeamService_NewTeamService(t *testing.T) {
	svc, _, _, cleanup := newTeamTestHarness(t)
	defer cleanup()

	if svc == nil {
		t.Fatal("expected NewTeamService to return a non-nil service")
	}
}

func TestTeamService_ListMembers(t *testing.T) {
	t.Run("empty list on fresh repo", func(t *testing.T) {
		svc, _, _, cleanup := newTeamTestHarness(t)
		defer cleanup()

		cfg, err := svc.ListMembers()
		if err != nil {
			t.Fatalf("ListMembers failed: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil TeamConfig")
		}
		if len(cfg.Members) != 0 {
			t.Fatalf("expected 0 members on fresh repo, got %d", len(cfg.Members))
		}
	})

	t.Run("list with members", func(t *testing.T) {
		svc, _, _, cleanup := newTeamTestHarness(t)
		defer cleanup()

		if err := svc.AddMember("alice", team.RoleAdmin); err != nil {
			t.Fatalf("AddMember alice: %v", err)
		}
		if err := svc.AddMember("bob", team.RoleMember); err != nil {
			t.Fatalf("AddMember bob: %v", err)
		}

		cfg, err := svc.ListMembers()
		if err != nil {
			t.Fatalf("ListMembers failed: %v", err)
		}
		if len(cfg.Members) != 2 {
			t.Fatalf("expected 2 members, got %d", len(cfg.Members))
		}
	})
}

func TestTeamService_AddMember(t *testing.T) {
	tests := []struct {
		name      string
		member    string
		role      team.Role
		setup     func(svc *application.TeamService)
		wantErr   bool
		wantCount int
		wantRole  team.Role
	}{
		{
			name:      "happy path adds new member",
			member:    "alice",
			role:      team.RoleAdmin,
			setup:     func(svc *application.TeamService) {},
			wantErr:   false,
			wantCount: 1,
			wantRole:  team.RoleAdmin,
		},
		{
			name:   "update existing member role",
			member: "alice",
			role:   team.RoleViewer,
			setup: func(svc *application.TeamService) {
				_ = svc.AddMember("alice", team.RoleAdmin)
			},
			wantErr:   false,
			wantCount: 1,
			wantRole:  team.RoleViewer,
		},
		{
			name:      "invalid role returns error",
			member:    "charlie",
			role:      team.Role("superadmin"),
			setup:     func(svc *application.TeamService) {},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, audit, cleanup := newTeamTestHarness(t)
			defer cleanup()

			tt.setup(svc)
			audit.Events = nil // reset audit events from setup

			err := svc.AddMember(tt.member, tt.role)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddMember() error = %v, wantErr %v", err, tt.wantErr)
			}

			cfg, loadErr := svc.ListMembers()
			if loadErr != nil {
				t.Fatalf("ListMembers after add: %v", loadErr)
			}

			if len(cfg.Members) != tt.wantCount {
				t.Fatalf("expected %d members, got %d", tt.wantCount, len(cfg.Members))
			}

			if !tt.wantErr {
				m := cfg.FindMember(tt.member)
				if m == nil {
					t.Fatalf("expected to find member %s", tt.member)
				}
				if m.Role != tt.wantRole {
					t.Fatalf("expected role %s, got %s", tt.wantRole, m.Role)
				}

				// Verify audit was called
				if len(audit.Events) != 1 {
					t.Fatalf("expected 1 audit event, got %d", len(audit.Events))
				}
				if audit.Events[0].Action != "team.add_member" {
					t.Fatalf("expected audit action team.add_member, got %s", audit.Events[0].Action)
				}
				if audit.Events[0].Actor != tt.member {
					t.Fatalf("expected audit actor %s, got %s", tt.member, audit.Events[0].Actor)
				}
			}
		})
	}
}

func TestTeamService_RemoveMember(t *testing.T) {
	tests := []struct {
		name    string
		member  string
		setup   func(svc *application.TeamService)
		wantErr bool
	}{
		{
			name:   "happy path removes existing member",
			member: "alice",
			setup: func(svc *application.TeamService) {
				_ = svc.AddMember("alice", team.RoleAdmin)
			},
			wantErr: false,
		},
		{
			name:    "not found returns error",
			member:  "ghost",
			setup:   func(svc *application.TeamService) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, audit, cleanup := newTeamTestHarness(t)
			defer cleanup()

			tt.setup(svc)
			audit.Events = nil // reset

			err := svc.RemoveMember(tt.member)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RemoveMember() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				cfg, loadErr := svc.ListMembers()
				if loadErr != nil {
					t.Fatalf("ListMembers after remove: %v", loadErr)
				}
				if cfg.FindMember(tt.member) != nil {
					t.Fatalf("expected member %s to be removed", tt.member)
				}

				// Verify audit was called
				if len(audit.Events) != 1 {
					t.Fatalf("expected 1 audit event, got %d", len(audit.Events))
				}
				if audit.Events[0].Action != "team.remove_member" {
					t.Fatalf("expected audit action team.remove_member, got %s", audit.Events[0].Action)
				}
			}
		})
	}
}

func TestTeamService_GetMemberRole(t *testing.T) {
	tests := []struct {
		name     string
		member   string
		setup    func(svc *application.TeamService)
		wantRole team.Role
		wantErr  bool
	}{
		{
			name:   "found returns role",
			member: "alice",
			setup: func(svc *application.TeamService) {
				_ = svc.AddMember("alice", team.RoleAdmin)
			},
			wantRole: team.RoleAdmin,
			wantErr:  false,
		},
		{
			name:     "not found returns empty role",
			member:   "ghost",
			setup:    func(svc *application.TeamService) {},
			wantRole: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, cleanup := newTeamTestHarness(t)
			defer cleanup()

			tt.setup(svc)

			role, err := svc.GetMemberRole(tt.member)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetMemberRole() error = %v, wantErr %v", err, tt.wantErr)
			}
			if role != tt.wantRole {
				t.Fatalf("GetMemberRole() = %s, want %s", role, tt.wantRole)
			}
		})
	}
}
