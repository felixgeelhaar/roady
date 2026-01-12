package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/internal/domain"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

type PlanService struct {
	repo  domain.WorkspaceRepository
	audit *AuditService
}

func NewPlanService(repo domain.WorkspaceRepository, audit *AuditService) *PlanService {
	return &PlanService{repo: repo, audit: audit}
}

// GeneratePlan updates the Plan based on the current Spec using a default heuristic.
func (s *PlanService) GeneratePlan() (*planning.Plan, error) {
	if err := s.audit.Log("plan.generate", "cli", nil); err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}

	// Default Heuristic: 1 Requirement = 1 Task
	heuristicTasks := make([]planning.Task, 0)
	for _, feat := range spec.Features {
		if len(feat.Requirements) == 0 {
			// Fallback if no requirements
			heuristicTasks = append(heuristicTasks, planning.Task{
				ID:          fmt.Sprintf("task-%s", feat.ID),
				Title:       fmt.Sprintf("Implement %s", feat.Title),
				Description: fmt.Sprintf("Implement the feature: %s. %s", feat.Title, feat.Description),
				FeatureID:   feat.ID,
				DependsOn:   []string{},
			})
			continue
		}

		for _, req := range feat.Requirements {
			deps := req.DependsOn
			if deps == nil {
				deps = []string{}
			}
			// Map requirement IDs to task IDs (prefix with task-)
			taskDeps := make([]string, len(deps))
			for i, d := range deps {
				taskDeps[i] = fmt.Sprintf("task-%s", d)
			}

			heuristicTasks = append(heuristicTasks, planning.Task{
				ID:          fmt.Sprintf("task-%s", req.ID),
				Title:       fmt.Sprintf("%s (%s)", req.Title, feat.Title),
				Description: req.Description,
				Priority:    planning.TaskPriority(req.Priority),
				Estimate:    req.Estimate,
				FeatureID:   feat.ID,
				DependsOn:   taskDeps,
			})
		}
	}

	return s.ReconcilePlan(heuristicTasks)
}

// UpdatePlan allows external agents (AI) to provide a specific list of tasks.
func (s *PlanService) UpdatePlan(tasks []planning.Task) (*planning.Plan, error) {
	if err := s.audit.Log("plan.update_smart", "ai", map[string]interface{}{
		"task_count": len(tasks),
	}); err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}
	return s.ReconcilePlan(tasks)
}

// ReconcilePlan merges new tasks with the existing plan state.
func (s *PlanService) ReconcilePlan(proposedTasks []planning.Task) (*planning.Plan, error) {
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}

	existingPlan, err := s.repo.LoadPlan()
	
	planID := fmt.Sprintf("plan-%s-%d", spec.ID, time.Now().Unix())
	createdAt := time.Now()
	
	currentTaskState := make(map[string]planning.Task) // ID -> Task

	if err == nil && existingPlan != nil {
		planID = existingPlan.ID
		createdAt = existingPlan.CreatedAt
		for _, t := range existingPlan.Tasks {
			currentTaskState[t.ID] = t
		}
	}

	newPlan := &planning.Plan{
		ID:             planID,
		SpecID:         spec.ID,
		ApprovalStatus: planning.ApprovalPending,
		CreatedAt:      createdAt,
		UpdatedAt:      time.Now(),
		Tasks:          make([]planning.Task, 0),
	}

	for _, proposed := range proposedTasks {
		if proposed.ID == "" || proposed.Title == "" {
			continue // Skip malformed proposed tasks
		}
		if _, ok := currentTaskState[proposed.ID]; ok {
			// In the structural plan, we just accept the proposed structure.
			// Execution state (Status/Path) is persisted separately in state.json.
			delete(currentTaskState, proposed.ID)
		}
		newPlan.Tasks = append(newPlan.Tasks, proposed)
	}

	// Keep Orphans (tasks that were manual or already exist but weren't in proposed)
	for _, orphan := range currentTaskState {
		if orphan.ID == "" || orphan.Title == "" {
			continue // Auto-clean hallucinations from history
		}
		newPlan.Tasks = append(newPlan.Tasks, orphan)
	}

	if err := newPlan.ValidateDAG(); err != nil {
		return nil, fmt.Errorf("invalid plan dependency graph: %w", err)
	}

	if err := s.repo.SavePlan(newPlan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	return newPlan, nil
}

func (s *PlanService) GetPlan() (*planning.Plan, error) {

	return s.repo.LoadPlan()

}



func (s *PlanService) GetState() (*planning.ExecutionState, error) {

	return s.repo.LoadState()

}

func (s *PlanService) GetUsage() (*domain.UsageStats, error) {
	return s.repo.LoadUsage()
}

func (s *PlanService) ApprovePlan() error {
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("no plan found to approve")
	}

	plan.ApprovalStatus = planning.ApprovalApproved
	plan.UpdatedAt = time.Now()
	return s.repo.SavePlan(plan)
}

func (s *PlanService) PrunePlan() error {
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return err
	}
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return err
	}

	validTaskIDs := make(map[string]bool)
	validFeatureIDs := make(map[string]bool)
	for _, f := range spec.Features {
		validFeatureIDs[f.ID] = true
		for _, r := range f.Requirements {
			validTaskIDs[fmt.Sprintf("task-%s", r.ID)] = true
		}
	}

	newTasks := make([]planning.Task, 0)
	for _, t := range plan.Tasks {
		// Task is valid if it matches a requirement ID OR its feature ID exists in the spec
		if validTaskIDs[t.ID] || validFeatureIDs[t.FeatureID] {
			newTasks = append(newTasks, t)
		}
	}

	plan.Tasks = newTasks
	plan.UpdatedAt = time.Now()
	return s.repo.SavePlan(plan)
}

func (s *PlanService) RejectPlan() error {
	plan, err := s.repo.LoadPlan()
	if err != nil {
		return err
	}
	if plan == nil {
		return fmt.Errorf("no plan found to reject")
	}

	plan.ApprovalStatus = planning.ApprovalRejected
	plan.UpdatedAt = time.Now()
	return s.repo.SavePlan(plan)
}
