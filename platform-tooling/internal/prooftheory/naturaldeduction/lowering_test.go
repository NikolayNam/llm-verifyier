package naturaldeduction

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

func TestLowerToCertificate_IdentityFixture(t *testing.T) {
	proof := loadNDFixture(t, filepath.Join("valid", "identity.json"))

	cert, err := LowerToCertificate(proof, LoweringOptions{})
	if err != nil {
		t.Fatalf("LowerToCertificate() error = %v", err)
	}
	if cert.ProofID != proof.ProofID {
		t.Fatalf("cert.ProofID = %q, want %q", cert.ProofID, proof.ProofID)
	}
	if _, err := hilbert.VerifyCertificate(cert); err != nil {
		t.Fatalf("VerifyCertificate(lowered) error = %v", err)
	}
}

func TestLowerToCertificate_TopLevelPremiseMP(t *testing.T) {
	proof := loadNDFixture(t, filepath.Join("valid", "premise_mp.json"))

	cert, err := LowerToCertificate(proof, LoweringOptions{})
	if err != nil {
		t.Fatalf("LowerToCertificate() error = %v", err)
	}
	if len(cert.Assumptions) != 2 {
		t.Fatalf("len(cert.Assumptions) = %d, want 2", len(cert.Assumptions))
	}
	if _, err := hilbert.VerifyCertificate(cert); err != nil {
		t.Fatalf("VerifyCertificate(lowered) error = %v", err)
	}
}

func TestLowerToCertificate_NestedImpIntroWithOuterPremise(t *testing.T) {
	proof, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "NDI04",
		"goal": "Q -> P",
		"assumptions": ["P"],
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"steps": [
			{"id": 1, "kind": "premise", "premise_ref": 1, "formula": "P"},
			{"id": 2, "kind": "assume", "formula": "Q", "scope": 1},
			{"id": 3, "kind": "reiterate", "from": [1], "formula": "P", "scope": 1},
			{"id": 4, "kind": "imp_intro", "from": [2, 3], "discharge_scope": 1, "formula": "Q -> P"}
		]
	}`))
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}

	cert, err := LowerToCertificate(proof, LoweringOptions{})
	if err != nil {
		t.Fatalf("LowerToCertificate() error = %v", err)
	}
	if _, err := hilbert.VerifyCertificate(cert); err != nil {
		t.Fatalf("VerifyCertificate(lowered) error = %v", err)
	}
}

func TestLowerToCertificate_CompositionTheorem(t *testing.T) {
	proof, err := DecodeJSON([]byte(`{
		"proof_version": "1.0.0",
		"proof_id": "NDI06",
		"goal": "(P -> (Q -> R)) -> ((P -> Q) -> (P -> R))",
		"context": {
			"domain": "hilbert-benchmark-nd-v1",
			"logic_fragment": "implicational-prop-v1",
			"front_end": "natural-deduction-v1",
			"lowering_target": "certificate-format-v1",
			"generator": "llm-benchmark-nd-v1"
		},
		"steps": [
			{"id": 1, "kind": "assume", "formula": "P -> (Q -> R)", "scope": 1},
			{"id": 2, "kind": "assume", "formula": "P -> Q", "scope": 2},
			{"id": 3, "kind": "assume", "formula": "P", "scope": 3},
			{"id": 4, "kind": "reiterate", "from": [1], "formula": "P -> (Q -> R)", "scope": 3},
			{"id": 5, "kind": "reiterate", "from": [3], "formula": "P", "scope": 3},
			{"id": 6, "kind": "imp_elim", "from": [5, 4], "formula": "Q -> R", "scope": 3},
			{"id": 7, "kind": "reiterate", "from": [2], "formula": "P -> Q", "scope": 3},
			{"id": 8, "kind": "imp_elim", "from": [5, 7], "formula": "Q", "scope": 3},
			{"id": 9, "kind": "imp_elim", "from": [8, 6], "formula": "R", "scope": 3},
			{"id": 10, "kind": "imp_intro", "from": [3, 9], "discharge_scope": 3, "formula": "P -> R"},
			{"id": 11, "kind": "imp_intro", "from": [2, 10], "discharge_scope": 2, "formula": "(P -> Q) -> (P -> R)"},
			{"id": 12, "kind": "imp_intro", "from": [1, 11], "discharge_scope": 1, "formula": "(P -> (Q -> R)) -> ((P -> Q) -> (P -> R))"}
		]
	}`))
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}

	cert, err := LowerToCertificate(proof, LoweringOptions{})
	if err != nil {
		t.Fatalf("LowerToCertificate() error = %v", err)
	}
	if _, err := hilbert.VerifyCertificate(cert); err != nil {
		t.Fatalf("VerifyCertificate(lowered) error = %v", err)
	}
}

func loadNDFixture(t *testing.T, parts ...string) *Proof {
	t.Helper()

	path := filepath.Join(append([]string{"testdata"}, parts...)...)
	proof, err := LoadJSONFile(path)
	if err != nil {
		t.Fatalf("LoadJSONFile(%q) error = %v", path, err)
	}
	return proof
}

func researchArtifactPath(parts ...string) string {
	base := []string{"..", "..", "..", "..", "research", "artifacts"}
	return filepath.Join(append(base, parts...)...)
}

func ndPilotTheoremPackPath() string {
	return researchArtifactPath("hilbert-ai-verification-benchmark-nd-v1", "theorems", "pilot_shared_20260327.csv")
}

func TestNDTheoremPackExistsForLoweringPilot(t *testing.T) {
	path := ndPilotTheoremPackPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}
