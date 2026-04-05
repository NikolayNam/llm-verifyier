package certificates

import "testing"

func TestDecodeJSON_DefaultsSyntax(t *testing.T) {
	cert, err := DecodeJSON([]byte(`{
		"certificate_version": "1.0.0",
		"proof_id": "proof-1",
		"goal": "P",
		"context": {
			"domain": "mevp-epcp",
			"rule_pack": "classical-hilbert-v1"
		},
		"steps": [
			{
				"kind": "assumption",
				"assumption_ref": 1,
				"formula": "P"
			}
		],
		"assumptions": ["P"]
	}`))
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if cert.Context.Syntax != DefaultSyntaxV1 {
		t.Fatalf("Context.Syntax = %q, want %q", cert.Context.Syntax, DefaultSyntaxV1)
	}
}

func TestDecodeJSON_RejectsDuplicateAssumptions(t *testing.T) {
	_, err := DecodeJSON([]byte(`{
		"certificate_version": "1.0.0",
		"proof_id": "proof-1",
		"goal": "P",
		"context": {
			"domain": "mevp-epcp",
			"rule_pack": "classical-hilbert-v1"
		},
		"steps": [
			{
				"kind": "assumption",
				"assumption_ref": 1,
				"formula": "P"
			}
		],
		"assumptions": ["P", "P"]
	}`))
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want duplicate assumption error")
	}
}

func TestDecodeJSON_RejectsUnknownField(t *testing.T) {
	_, err := DecodeJSON([]byte(`{
		"certificate_version": "1.0.0",
		"proof_id": "proof-1",
		"goal": "P",
		"context": {
			"domain": "mevp-epcp",
			"rule_pack": "classical-hilbert-v1"
		},
		"steps": [
			{
				"kind": "assumption",
				"assumption_ref": 1,
				"formula": "P",
				"unexpected": true
			}
		],
		"assumptions": ["P"]
	}`))
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want unknown field error")
	}
}
