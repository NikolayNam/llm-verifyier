package certificates

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

func TestCertificateSchemaContract(t *testing.T) {
	path := toolingpath.RepoPath("docs", "contracts", "schemas", "certificate.schema.json")
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
	for _, field := range []string{"certificate_version", "proof_id", "goal", "context", "steps"} {
		if _, ok := required[field]; !ok {
			t.Fatalf("required fields missing %q", field)
		}
	}

	properties := schema["properties"].(map[string]any)
	certVersion := properties["certificate_version"].(map[string]any)
	if got := certVersion["const"]; got != FormatVersionV1 {
		t.Fatalf("certificate_version const = %#v, want %q", got, FormatVersionV1)
	}

	context := properties["context"].(map[string]any)
	contextProps := context["properties"].(map[string]any)
	if _, ok := contextProps["rule_pack"]; !ok {
		t.Fatal("context.properties.rule_pack missing")
	}

	defs := schema["$defs"].(map[string]any)
	assumptionStep := defs["assumptionStep"].(map[string]any)
	if got := assumptionStep["properties"].(map[string]any)["kind"].(map[string]any)["const"]; got != string(StepKindAssumption) {
		t.Fatalf("assumption kind const = %#v", got)
	}
	modusPonensStep := defs["modusPonensStep"].(map[string]any)
	if got := modusPonensStep["properties"].(map[string]any)["kind"].(map[string]any)["const"]; got != string(StepKindModusPonens) {
		t.Fatalf("modus ponens kind const = %#v", got)
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
