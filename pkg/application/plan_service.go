package application

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
)

type PlanService struct {
	repo       domain.WorkspaceRepository
	audit      domain.AuditLogger
	reconciler *planning.PlanReconciler
}

func NewPlanService(repo domain.WorkspaceRepository, audit domain.AuditLogger) *PlanService {
	return &PlanService{
		repo:       repo,
		audit:      audit,
		reconciler: planning.NewPlanReconciler(),
	}
}

// GeneratePlan updates the Plan based on the current Spec using a default heuristic.
func (s *PlanService) GeneratePlan(ctx context.Context) (*planning.Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := s.audit.Log("plan.generate", "cli", nil); err != nil {
		return nil, fmt.Errorf("write audit log: %w", err)
	}
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
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
	plan, err := s.ReconcilePlan(tasks)
	if err != nil {
		return nil, err
	}

	if err := s.audit.Log("plan.update_smart", "ai", map[string]interface{}{
		"plan_id":    plan.ID,
		"spec_id":    plan.SpecID,
		"task_count": len(tasks),
	}); err != nil {
		return nil, fmt.Errorf("write audit log: %w", err)
	}

	return plan, nil
}

// ReconcilePlan merges new tasks with the existing plan state.
func (s *PlanService) ReconcilePlan(proposedTasks []planning.Task) (*planning.Plan, error) {
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	existingPlan, _ := s.repo.LoadPlan()

	newPlan, err := s.reconciler.Reconcile(existingPlan, proposedTasks, planning.ReconcileOptions{
		SpecID: spec.ID,
	})
	if err != nil {
		return nil, err
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
	if err := s.repo.SavePlan(plan); err != nil {
		return err
	}
	if err := s.audit.Log("plan.approve", "cli", map[string]interface{}{
		"plan_id": plan.ID,
		"spec_id": plan.SpecID,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
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

	plan.Tasks = s.reconciler.FilterValidTasks(plan.Tasks, validTaskIDs, validFeatureIDs)
	plan.UpdatedAt = time.Now()
	if err := s.audit.Log("plan.prune", "cli", map[string]interface{}{
		"plan_id": plan.ID,
		"spec_id": plan.SpecID,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
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
	if err := s.audit.Log("plan.reject", "cli", map[string]interface{}{
		"plan_id": plan.ID,
		"spec_id": plan.SpecID,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return s.repo.SavePlan(plan)
}
