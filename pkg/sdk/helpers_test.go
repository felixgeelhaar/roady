package sdk

import (
	"testing"
)

func TestStatusRequest_ArgsBuilding(t *testing.T) {
	req := StatusRequest{
		Status:   "pending,done",
		Priority: "high",
		Ready:    true,
		Limit:    5,
	}
	if req.Status != "pending,done" {
		t.Errorf("unexpected status: %s", req.Status)
	}
	if req.Limit != 5 {
		t.Errorf("unexpected limit: %d", req.Limit)
	}
}

func TestTransitionRequest_Fields(t *testing.T) {
	req := TransitionRequest{
		TaskID:   "task-1",
		Event:    "start",
		Evidence: "commit abc123",
		Actor:    "dev",
	}
	if req.TaskID != "task-1" {
		t.Errorf("unexpected task id: %s", req.TaskID)
	}
	if req.Actor != "dev" {
		t.Errorf("unexpected actor: %s", req.Actor)
	}
}
