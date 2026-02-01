package application

import (
	"fmt"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/team"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// TeamService manages team membership and role-based access.
type TeamService struct {
	repo  *storage.FilesystemRepository
	audit domain.AuditLogger
}

func NewTeamService(repo *storage.FilesystemRepository, audit domain.AuditLogger) *TeamService {
	return &TeamService{repo: repo, audit: audit}
}

// ListMembers returns the current team configuration.
func (s *TeamService) ListMembers() (*team.TeamConfig, error) {
	return s.repo.LoadTeam()
}

// AddMember adds or updates a team member.
func (s *TeamService) AddMember(name string, role team.Role) error {
	cfg, err := s.repo.LoadTeam()
	if err != nil {
		return fmt.Errorf("load team: %w", err)
	}

	if err := cfg.AddMember(name, role); err != nil {
		return err
	}

	if err := s.repo.SaveTeam(cfg); err != nil {
		return fmt.Errorf("save team: %w", err)
	}

	return s.audit.Log("team.add_member", name, map[string]interface{}{
		"member": name,
		"role":   string(role),
	})
}

// RemoveMember removes a team member.
func (s *TeamService) RemoveMember(name string) error {
	cfg, err := s.repo.LoadTeam()
	if err != nil {
		return fmt.Errorf("load team: %w", err)
	}

	if err := cfg.RemoveMember(name); err != nil {
		return err
	}

	if err := s.repo.SaveTeam(cfg); err != nil {
		return fmt.Errorf("save team: %w", err)
	}

	return s.audit.Log("team.remove_member", name, map[string]interface{}{
		"member": name,
	})
}

// GetMemberRole returns the role for a given member name, or empty if not found.
func (s *TeamService) GetMemberRole(name string) (team.Role, error) {
	cfg, err := s.repo.LoadTeam()
	if err != nil {
		return "", fmt.Errorf("load team: %w", err)
	}

	m := cfg.FindMember(name)
	if m == nil {
		return "", nil
	}
	return m.Role, nil
}
