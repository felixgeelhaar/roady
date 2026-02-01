package mcp

import (
	"regexp"
	"testing"
)

func TestSchemaVersionIsSemver(t *testing.T) {
	re := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !re.MatchString(SchemaVersion) {
		t.Fatalf("SchemaVersion %q is not valid semver", SchemaVersion)
	}
}

func TestDeprecatedFieldsPopulated(t *testing.T) {
	for i, d := range deprecatedFields() {
		if d.Tool == "" {
			t.Errorf("deprecatedFields()[%d].Tool is empty", i)
		}
		if d.Field == "" {
			t.Errorf("deprecatedFields()[%d].Field is empty", i)
		}
		if d.Since == "" {
			t.Errorf("deprecatedFields()[%d].Since is empty", i)
		}
		if d.RemovedIn == "" {
			t.Errorf("deprecatedFields()[%d].RemovedIn is empty", i)
		}
		if d.Migration == "" {
			t.Errorf("deprecatedFields()[%d].Migration is empty", i)
		}
	}
}
