package application

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/roady/internal/domain"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
)

type TaskService struct {
	repo  domain.WorkspaceRepository
	audit *AuditService
}

func NewTaskService(repo domain.WorkspaceRepository, audit *AuditService) *TaskService {
	return &TaskService{repo: repo, audit: audit}
}

func (s *TaskService) TransitionTask(taskID string, event string, actor string, evidence string) error {

	plan, err := s.repo.LoadPlan()

	if err != nil {

		return err

	}

	if plan == nil {

		return fmt.Errorf("no plan found")

	}



	found := false

	for _, t := range plan.Tasks {

		if t.ID == taskID {

			found = true

			break

		}

	}

	if !found {

		return fmt.Errorf("task not found in plan: %s", taskID)

	}



	state, err := s.repo.LoadState()

	if err != nil {

		return err

	}



	// 1. Get Current Status (default to Pending if not in state)

	currentStatus := planning.StatusPending

	if result, ok := state.TaskStates[taskID]; ok {

		currentStatus = result.Status

	}



	// 2. Setup Guard with Policy Service

	policySvc := NewPolicyService(s.repo)

	guard := func(tid string, ev string) bool {

		// Block starts if plan not approved

		if ev == "start" {

			plan, err := s.repo.LoadPlan()

			if err != nil || plan == nil || plan.ApprovalStatus != planning.ApprovalApproved {

				return false

			}

		}

		err := policySvc.ValidateTransition(tid, ev)

		return err == nil

	}



	// 3. FSM Transition

	fsm, err := planning.NewTaskStateMachine(string(currentStatus), taskID, guard)

	if err != nil {

		return err

	}



	if err := fsm.Transition(event); err != nil {

		// Try to get a more specific error from policy service or plan approval

		if event == "start" {

			plan, _ := s.repo.LoadPlan()

			if plan != nil && plan.ApprovalStatus != planning.ApprovalApproved {

				return fmt.Errorf("cannot start task: the plan is currently '%s'. Please approve the plan using 'roady plan approve' before starting work.", plan.ApprovalStatus)

			}

			if pErr := policySvc.ValidateTransition(taskID, event); pErr != nil {

				return pErr

			}

		}

		return err

	}



	// 4. Update State

	newState := fsm.Current()

	result := state.TaskStates[taskID]

	result.Status = planning.TaskStatus(newState)



	// Ownership Inference (Horizon 3)

	if event == "start" {

		result.Owner = actor

	}



	// Status Confidence / Evidence (Horizon 3)

	if evidence != "" {

		result.Evidence = append(result.Evidence, evidence)

	}



		state.TaskStates[taskID] = result



		state.UpdatedAt = time.Now()



	



		if err := s.repo.SaveState(state); err != nil {



			return err



		}



	



		return s.audit.Log("task.transition", actor, map[string]interface{}{



			"task_id":  taskID,



			"event":    event,



			"status":   newState,



			"evidence": evidence,



		})



	}



	


