package mcp

import (
	"encoding/json"
	"testing"
)

func TestFlexBoolUnmarshal(t *testing.T) {
	var args StatusArgs

	if err := json.Unmarshal([]byte(`{"ready":true,"json":"yes"}`), &args); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !bool(args.Ready) {
		t.Error("expected Ready to be true")
	}
	if !bool(args.JSON) {
		t.Error("expected JSON to be true")
	}
}

func TestFlexBoolUnmarshal_Invalid(t *testing.T) {
	var args StatusArgs

	if err := json.Unmarshal([]byte(`{"ready": {}}`), &args); err == nil {
		t.Fatal("expected error for invalid flex bool")
	}
}

func TestFlexIntUnmarshal(t *testing.T) {
	var args StatusArgs

	if err := json.Unmarshal([]byte(`{"limit":"5"}`), &args); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if int(args.Limit) != 5 {
		t.Errorf("expected limit 5, got %d", int(args.Limit))
	}
}

func TestFlexIntUnmarshal_Invalid(t *testing.T) {
	var args StatusArgs

	if err := json.Unmarshal([]byte(`{"limit":"nope"}`), &args); err == nil {
		t.Fatal("expected error for invalid flex int")
	}
}
