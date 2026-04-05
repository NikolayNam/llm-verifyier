package naturaldeduction

import "testing"

func TestDecodeJSON_AllowsTopLevelImpElimWithoutScope(t *testing.T) {
	proof, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "NDI02",
		"goal": "Q",
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"assumptions": ["P", "P -> Q"],
		"steps": [
			{"id": 1, "kind": "premise", "premise_ref": 1, "formula": "P"},
			{"id": 2, "kind": "premise", "premise_ref": 2, "formula": "P -> Q"},
			{"id": 3, "kind": "imp_elim", "from": [1, 2], "formula": "Q"}
		]
	}`))
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if proof == nil {
		t.Fatal("DecodeJSON() proof = nil")
	}
}

func TestDecodeJSON_RejectsNegationInFormula(t *testing.T) {
	_, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "bad-parse",
		"goal": "!P",
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"steps": [
			{"id": 1, "kind": "assume", "formula": "P", "scope": 1}
		]
	}`))
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want parse error")
	}
	if got := ClassOf(err); got != ErrorClassParse {
		t.Fatalf("ClassOf(err) = %q, want %q", got, ErrorClassParse)
	}
}

func TestDecodeJSON_RejectsClosedScopeReference(t *testing.T) {
	_, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "bad-scope",
		"goal": "P",
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"steps": [
			{"id": 1, "kind": "assume", "formula": "P", "scope": 1},
			{"id": 2, "kind": "reiterate", "from": [1], "formula": "P", "scope": 1},
			{"id": 3, "kind": "imp_intro", "from": [1, 2], "discharge_scope": 1, "formula": "P -> P"},
			{"id": 4, "kind": "reiterate", "from": [1], "formula": "P"}
		]
	}`))
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want validation error")
	}
	if got := ClassOf(err); got != ErrorClassValidation {
		t.Fatalf("ClassOf(err) = %q, want %q", got, ErrorClassValidation)
	}
}

func TestDecodeJSON_RejectsUnknownField(t *testing.T) {
	_, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "bad-shape",
		"goal": "P -> P",
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"steps": [
			{"id": 1, "kind": "assume", "formula": "P", "scope": 1, "unexpected": true}
		]
	}`))
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want schema error")
	}
	if got := ClassOf(err); got != ErrorClassSchema {
		t.Fatalf("ClassOf(err) = %q, want %q", got, ErrorClassSchema)
	}
}
