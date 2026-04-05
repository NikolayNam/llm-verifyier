package naturaldeduction

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNaturalDeductionSchemaContract(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "contracts", "schemas", "natural-deduction.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}

	if got := schema["type"]; got != "object" {
		t.Fatalf("schema type = %#v, want object", got)
	}
	if got := schema["additionalProperties"]; got != false {
		t.Fatalf("schema additionalProperties = %#v, want false", got)
	}

	required := toStringSet(t, schema["required"])
	for _, field := range []string{"proof_version", "proof_id", "goal", "context", "steps"} {
		if _, ok := required[field]; !ok {
			t.Fatalf("required fields missing %q", field)
		}
	}

	properties := schema["properties"].(map[string]any)
	proofVersion := properties["proof_version"].(map[string]any)
	if got := proofVersion["const"]; got != ProofVersionV1 {
		t.Fatalf("proof_version const = %#v, want %q", got, ProofVersionV1)
	}

	context := properties["context"].(map[string]any)
	contextProps := context["properties"].(map[string]any)
	if got := contextProps["logic_fragment"].(map[string]any)["const"]; got != LogicFragmentV1 {
		t.Fatalf("context.logic_fragment const = %#v, want %q", got, LogicFragmentV1)
	}

	defs := schema["$defs"].(map[string]any)
	reiterateStep := defs["reiterateStep"].(map[string]any)
	reiterateRequired := toStringSet(t, reiterateStep["required"])
	if _, ok := reiterateRequired["scope"]; ok {
		t.Fatal("reiterateStep.required unexpectedly contains scope")
	}

	impElimStep := defs["impElimStep"].(map[string]any)
	impElimRequired := toStringSet(t, impElimStep["required"])
	if _, ok := impElimRequired["scope"]; ok {
		t.Fatal("impElimStep.required unexpectedly contains scope")
	}
	if got := impElimStep["properties"].(map[string]any)["kind"].(map[string]any)["const"]; got != string(StepKindImpElim) {
		t.Fatalf("impElim kind const = %#v", got)
	}
}

func toStringSet(t *testing.T, raw any) map[string]struct{} {
	t.Helper()

	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", raw)
	}
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("expected string item, got %T", item)
		}
		out[s] = struct{}{}
	}
	return out
}
