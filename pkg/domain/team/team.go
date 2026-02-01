package team

import "fmt"

// Role defines the access level of a team member.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// ValidRoles returns all valid role values.
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleMember, RoleViewer}
}

// IsValid checks if the role is a recognized value.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember, RoleViewer:
		return true
	}
	return false
}

// CanTransitionTasks returns true if the role allows task state changes.
func (r Role) CanTransitionTasks() bool {
	return r == RoleAdmin || r == RoleMember
}

// CanEditPlan returns true if the role allows plan modifications.
func (r Role) CanEditPlan() bool {
	return r == RoleAdmin || r == RoleMember
}

// CanManageTeam returns true if the role allows team membership changes.
func (r Role) CanManageTeam() bool {
	return r == RoleAdmin
}

// Member represents a team member with a role.
type Member struct {
	Name string `yaml:"name" json:"name"`
	Role Role   `yaml:"role" json:"role"`
}

// TeamConfig holds the team configuration stored in .roady/team.yaml.
type TeamConfig struct {
	Members []Member `yaml:"members" json:"members"`
}

// FindMember returns the member with the given name, or nil if not found.
func (t *TeamConfig) FindMember(name string) *Member {
	for i := range t.Members {
		if t.Members[i].Name == name {
			return &t.Members[i]
		}
	}
	return nil
}

// AddMember adds a member or updates their role if they already exist.
func (t *TeamConfig) AddMember(name string, role Role) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid role: %s", role)
	}
	if name == "" {
		return fmt.Errorf("member name cannot be empty")
	}
	for i := range t.Members {
		if t.Members[i].Name == name {
			t.Members[i].Role = role
			return nil
		}
	}
	t.Members = append(t.Members, Member{Name: name, Role: role})
	return nil
}

// RemoveMember removes a member by name. Returns error if not found.
func (t *TeamConfig) RemoveMember(name string) error {
	for i := range t.Members {
		if t.Members[i].Name == name {
			t.Members = append(t.Members[:i], t.Members[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("member not found: %s", name)
}
