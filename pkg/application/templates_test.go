package application

import "testing"

func TestBuiltinTemplates(t *testing.T) {
	templates := BuiltinTemplates()
	if len(templates) < 3 {
		t.Fatalf("expected at least 3 templates, got %d", len(templates))
	}

	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("template name is empty")
		}
		spec := tmpl.Spec("test-project")
		if spec.ID != "test-project" {
			t.Errorf("template %s: expected ID test-project, got %s", tmpl.Name, spec.ID)
		}
		if len(spec.Features) == 0 {
			t.Errorf("template %s: expected at least 1 feature", tmpl.Name)
		}
	}
}

func TestFindTemplate(t *testing.T) {
	tmpl := FindTemplate("web-api")
	if tmpl == nil {
		t.Fatal("expected to find web-api template")
	}
	if tmpl.Name != "web-api" {
		t.Errorf("expected web-api, got %s", tmpl.Name)
	}

	if FindTemplate("nonexistent") != nil {
		t.Error("expected nil for unknown template")
	}
}
