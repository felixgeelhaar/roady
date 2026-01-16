package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/domain/policy/rules"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

type PolicyService struct {
	repo domain.WorkspaceRepository
}

func NewPolicyService(repo domain.WorkspaceRepository) *PolicyService {
	return &PolicyService{repo: repo}
}

// CheckCompliance validates the current plan against active policies.
func (s *PolicyService) CheckCompliance() ([]policy.Violation, error) {
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	state, err := s.repo.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	var activeRules []policy.Rule
	if cfg != nil {
		activeRules = append(activeRules, &rules.MaxWIPRule{Limit: cfg.MaxWIP})
	}
	activeRules = append(activeRules, &rules.DependencyRule{})

	policySet := policy.PolicySet{
		Rules: activeRules,
	}

	return policySet.Validate(plan, state), nil
}

func (s *PolicyService) ValidateTransition(taskID string, event string) error {
	if event != "start" {
		return nil
	}

	// 1. Check WIP Limit
	cfg, err := s.repo.LoadPolicy()
	if err == nil && cfg != nil && cfg.MaxWIP > 0 {
		state, err := s.repo.LoadState()
		if err == nil {
			inProgressCount := 0
			for id, ts := range state.TaskStates {
				if id != taskID && ts.Status == "in_progress" {
					inProgressCount++
				}
			}
			if inProgressCount >= cfg.MaxWIP {
				return fmt.Errorf("WIP limit reached (current limit: %d). Please complete or stop an existing task before starting a new one.", cfg.MaxWIP)
			}
		}
	}

	// 2. Check Dependencies
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return err
	}
	if plan == nil {
		return nil // No plan, skip dependency validation
	}

	var targetTask *planning.Task
	for _, t := range plan.Tasks {
		if t.ID == taskID {
			targetTask = &t
			break
		}
	}

	if targetTask != nil && len(targetTask.DependsOn) > 0 {
		state, err := s.repo.LoadState()
		if err != nil {
			return err
		}
		for _, depID := range targetTask.DependsOn {
			// Handle Cross-repo Dependency (format: "project-name:task-id")
			if strings.Contains(depID, ":") {
				parts := strings.Split(depID, ":")
				extProject, extTask := parts[0], parts[1]

				// Discovery loop to find the project
				extRepoPath, found := s.findExternalProject(extProject)
				if !found {
					return fmt.Errorf("cannot start task '%s': depends on external project '%s' which cannot be found", taskID, extProject)
				}

				extRepo := storage.NewFilesystemRepository(extRepoPath)
				extState, err := extRepo.LoadState()
				if err != nil {
					return fmt.Errorf("cannot verify dependency '%s': failed to load external state", depID)
				}

				extStatus := planning.StatusPending
				if res, ok := extState.TaskStates[extTask]; ok {
					extStatus = res.Status
				}

				if extStatus != planning.StatusDone && extStatus != planning.StatusVerified {
					return fmt.Errorf("cannot start task '%s': it depends on '%s' in project '%s', which is currently '%s'", taskID, extTask, extProject, extStatus)
				}
				continue
			}

			// Local dependency
			if res, ok := state.TaskStates[depID]; !ok || (res.Status != planning.StatusDone && res.Status != planning.StatusVerified) {
				return fmt.Errorf("cannot start task '%s': it depends on '%s', which is not yet completed. Please finish all dependencies first.", taskID, depID)
			}
		}
	}

	return nil
}

func (s *PolicyService) findExternalProject(name string) (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}

	// Start from parent of current repo to find siblings
	root := filepath.Dir(cwd)

	foundPath := ""
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}
		if info.IsDir() && info.Name() == ".roady" {
			projectDir := filepath.Dir(path)
			repo := storage.NewFilesystemRepository(projectDir)
			spec, loadErr := repo.LoadSpec()
			if loadErr == nil && spec != nil && (spec.ID == name || spec.Title == name) {
				foundPath = projectDir
				return filepath.SkipDir
			}
		}
		return nil
	})

	if walkErr != nil {
		return "", false
	}

	return foundPath, foundPath != ""
}
