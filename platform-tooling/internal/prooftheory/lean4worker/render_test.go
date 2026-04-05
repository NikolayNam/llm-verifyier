package lean4worker

import (
	"strings"
	"testing"
)

func TestRenderModule_AutoVariablesAndPlaceholderProof(t *testing.T) {
	module, err := RenderModule(Job{
		Name:      "identity-case",
		Statement: "P -> P",
	})
	if err != nil {
		t.Fatalf("RenderModule() error = %v", err)
	}
	if !strings.Contains(module, "import Mathlib.Tactic") {
		t.Fatalf("module missing default Mathlib.Tactic import:\n%s", module)
	}
	if !strings.Contains(module, "variable {P : Prop}") {
		t.Fatalf("module missing inferred proposition variable:\n%s", module)
	}
	if !strings.Contains(module, "theorem identity_case : P -> P := by") {
		t.Fatalf("module missing theorem declaration:\n%s", module)
	}
	if !strings.Contains(module, "exact ?_") {
		t.Fatalf("module missing deterministic placeholder proof:\n%s", module)
	}
}

func TestRenderModule_UsesPayloadImportsHelpersAndProof(t *testing.T) {
	module, err := RenderModule(Job{
		Name:      "comp",
		Statement: "P -> P",
		Payload:   []byte(`{"imports":["Mathlib.Tactic"],"helpers":["def keepP : Prop := P"],"proof":"intro h\nexact h"}`),
	})
	if err != nil {
		t.Fatalf("RenderModule() error = %v", err)
	}
	if !strings.Contains(module, "import Mathlib.Tactic") {
		t.Fatalf("module missing payload import:\n%s", module)
	}
	if !strings.Contains(module, "def keepP : Prop := P") {
		t.Fatalf("module missing helper definition:\n%s", module)
	}
	if !strings.Contains(module, "  intro h") || !strings.Contains(module, "  exact h") {
		t.Fatalf("module missing proof lines:\n%s", module)
	}
}
