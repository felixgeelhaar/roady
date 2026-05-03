package application

import (
	"testing"

	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

func TestPopulateTaskSources_BackfillsAITasks(t *testing.T) {
	productSpec := &spec.ProductSpec{
		ID: "demo",
		Features: []spec.Feature{
			{
				ID:     "auth",
				Title:  "Auth",
				Source: spec.Source{Doc: "docs/auth.md", Line: 10},
				Requirements: []spec.Requirement{
					{ID: "auth-signup", Source: spec.Source{Doc: "docs/auth.md", Line: 22}},
					{ID: "auth-login"}, // no requirement-level source
				},
			},
			{
				ID:     "tasks",
				Title:  "Tasks",
				Source: spec.Source{Doc: "docs/tasks.md", Line: 4},
			},
		},
	}

	tasks := []planning.Task{
		// AI-generated task aligned with a requirement that has its own source.
		{ID: "task-auth-signup", FeatureID: "auth", Origin: planning.OriginAI},
		// AI-generated task aligned with a requirement that has no source — falls back to feature.
		{ID: "task-auth-login", FeatureID: "auth", Origin: planning.OriginAI},
		// AI-generated task that names a feature directly.
		{ID: "task-build-tasks-list", FeatureID: "tasks", Origin: planning.OriginAI},
		// Already-populated source must not be overwritten.
		{ID: "task-custom", FeatureID: "auth", Source: planning.TaskSource{Doc: "manual.md", Line: 1}, Origin: planning.OriginHuman},
		// Unknown feature — left empty.
		{ID: "task-orphan", FeatureID: "nope", Origin: planning.OriginAI},
	}

	populateTaskSources(tasks, productSpec)

	want := map[string]planning.TaskSource{
		"task-auth-signup":      {Doc: "docs/auth.md", Line: 22},
		"task-auth-login":       {Doc: "docs/auth.md", Line: 10},
		"task-build-tasks-list": {Doc: "docs/tasks.md", Line: 4},
		"task-custom":           {Doc: "manual.md", Line: 1},
		"task-orphan":           {},
	}
	for _, task := range tasks {
		if got := task.Source; got != want[task.ID] {
			t.Errorf("task %q source = %+v, want %+v", task.ID, got, want[task.ID])
		}
	}
}

func TestPopulateTaskSources_NilSpecIsNoOp(t *testing.T) {
	tasks := []planning.Task{{ID: "task-x"}}
	populateTaskSources(tasks, nil)
	if !tasks[0].Source.IsZero() {
		t.Error("expected unchanged source when spec is nil")
	}
}
