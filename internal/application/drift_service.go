package application

import (
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/roady/internal/domain"
	"github.com/felixgeelhaar/roady/internal/domain/drift"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

type DriftService struct {
	repo domain.WorkspaceRepository
}

func NewDriftService(repo domain.WorkspaceRepository) *DriftService {
	return &DriftService{repo: repo}
}

func (s *DriftService) DetectDrift() (*drift.Report, error) {

	spec, err := s.repo.LoadSpec()

	if err != nil {

		return nil, err

	}



	plan, err := s.repo.LoadPlan()

	if err != nil {

		return nil, err

	}



	state, err := s.repo.LoadState()

	if err != nil {

		return nil, err

	}



		report := &drift.Report{



			ID:        fmt.Sprintf("drift-%d", time.Now().Unix()),



			CreatedAt: time.Now(),



			Issues:    make([]drift.Issue, 0),



		}



	



		// 0. Intent Drift (Spec vs Lock)



		lock, err := s.repo.LoadSpecLock()



		if err == nil && lock != nil {



			if spec.Hash() != lock.Hash() {



				report.Issues = append(report.Issues, drift.Issue{



					ID:       "intent-drift",



					Type:     drift.DriftTypeSpec,



					Category: drift.CategoryMismatch,



					Severity: drift.SeverityMedium,



					Message:  "The Specification has changed since the Plan was last updated. Your intent and plan may be out of sync.",



					Hint:     "Review the changes and run 'roady plan generate' to align your plan with the new Spec.",



				})



			}



		}



	



		// 1. Plan vs Spec



	
	taskIDMap := make(map[string]bool)
	specRequirementIDs := make(map[string]bool)

	if plan != nil {
		for _, t := range plan.Tasks {
			taskIDMap[t.ID] = true
		}
	}

	for _, f := range spec.Features {
		for _, r := range f.Requirements {
			taskID := fmt.Sprintf("task-%s", r.ID)
			specRequirementIDs[taskID] = true
			if !taskIDMap[taskID] {
				report.Issues = append(report.Issues, drift.Issue{
					ID:          fmt.Sprintf("missing-task-%s", r.ID),
					Type:        drift.DriftTypePlan,
					Category:    drift.CategoryMissing,
					Severity:    drift.SeverityHigh,
					ComponentID: r.ID,
					Message:     fmt.Sprintf("Requirement '%s' (Feature: %s) is missing from Plan.", r.Title, f.Title),
					Hint:        "Run 'roady plan generate' to update your plan.",
				})
			}
		}
	}

	// Detect Orphan Tasks (in Plan but not in Spec)
	if plan != nil {
		specFeatureIDs := make(map[string]bool)
		for _, f := range spec.Features {
			specFeatureIDs[f.ID] = true
		}

		for _, t := range plan.Tasks {
			// A task is an orphan only if it doesn't match a Requirement AND doesn't match a Feature
			if !specRequirementIDs[t.ID] && !specFeatureIDs[t.FeatureID] {
				report.Issues = append(report.Issues, drift.Issue{
					ID:          fmt.Sprintf("orphan-task-%s", t.ID),
					Type:        drift.DriftTypePlan,
					Category:    drift.CategoryOrphan,
					Severity:    drift.SeverityMedium,
					ComponentID: t.ID,
					Message:     fmt.Sprintf("Task '%s' (ID: %s) exists in Plan but corresponds to no active Feature or Requirement in Spec.", t.Title, t.ID),
					Hint:        "Run 'roady plan prune' to remove orphan tasks or update your Spec to include this intent.",
				})
			}
		}
	}

	// 2. Code vs State (Implementation Drift)
	if plan != nil && state != nil {
		for _, task := range plan.Tasks {
			result := state.TaskStates[task.ID]
			if result.Path == "" {
				continue
			}

			info, err := os.Stat(result.Path)
			exists := err == nil

			if result.Status == planning.StatusDone {
				if !exists {
					report.Issues = append(report.Issues, drift.Issue{
						ID:          fmt.Sprintf("missing-code-%s", task.ID),
						Type:        drift.DriftTypeCode,
						Category:    drift.CategoryImplementation,
						Severity:    drift.SeverityCritical,
						ComponentID: task.ID,
						Message:     fmt.Sprintf("Task '%s' is DONE but path '%s' is missing.", task.Title, result.Path),
						Hint:        "Restore the missing file or mark the task as incomplete using 'roady task reopen'.",
					})
				} else if info.Size() == 0 {
					report.Issues = append(report.Issues, drift.Issue{
						ID:          fmt.Sprintf("empty-code-%s", task.ID),
						Type:        drift.DriftTypeCode,
						Category:    drift.CategoryImplementation,
						Severity:    drift.SeverityHigh,
						ComponentID: task.ID,
						Message:     fmt.Sprintf("Task '%s' is DONE but file '%s' is empty.", task.Title, result.Path),
						Hint:        "Ensure the task implementation is committed to the file.",
					})
				}
			}
		}
	}

	// 3. Policy vs State (Policy Drift)
	policySvc := NewPolicyService(s.repo)
	violations, _ := policySvc.CheckCompliance()

	for _, v := range violations {
		severity := drift.SeverityHigh
		if v.Level == "warning" {
			severity = drift.SeverityMedium
		}

		report.Issues = append(report.Issues, drift.Issue{
			ID:       fmt.Sprintf("policy-%s", v.RuleID),
			Type:     drift.DriftTypePolicy,
			Category: drift.CategoryViolation,
			Severity: severity,
			Message:  v.Message,
			Hint:     "Adjust your execution state or update policy.yaml to resolve this violation.",
		})
	}



	



		return report, nil



	}



	
