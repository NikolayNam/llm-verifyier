package hilbert

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
)

func loadFixture(t *testing.T, relative string) *certificates.Certificate {
	t.Helper()

	path := filepath.Join("..", "testdata", relative)
	cert, err := certificates.LoadJSONFile(path)
	if err != nil {
		t.Fatalf("LoadJSONFile(%q) error = %v", path, err)
	}
	return cert
}

func TestVerifyCertificate_AImpliesA(t *testing.T) {
	cert := loadFixture(t, filepath.Join("valid", "a_implies_a.json"))
	report, err := VerifyCertificate(cert)
	if err != nil {
		t.Fatalf("VerifyCertificate() error = %v", err)
	}
	if len(report.Steps) != 5 {
		t.Fatalf("len(report.Steps) = %d, want 5", len(report.Steps))
	}
	if report.Steps[len(report.Steps)-1].Formula.String() != "A -> A" {
		t.Fatalf("final formula = %q, want %q", report.Steps[len(report.Steps)-1].Formula.String(), "A -> A")
	}
}

func TestVerifyCertificate_AssumptionImport(t *testing.T) {
	cert := loadFixture(t, filepath.Join("valid", "assumption_goal.json"))
	report, err := VerifyCertificate(cert)
	if err != nil {
		t.Fatalf("VerifyCertificate() error = %v", err)
	}
	if got := report.Steps[0].Reason; got != "assumption 1" {
		t.Fatalf("reason = %q, want %q", got, "assumption 1")
	}
}

func TestVerifyCertificate_SingleModusPonens(t *testing.T) {
	cert := loadFixture(t, filepath.Join("valid", "single_modus_ponens.json"))
	report, err := VerifyCertificate(cert)
	if err != nil {
		t.Fatalf("VerifyCertificate() error = %v", err)
	}
	if len(report.Steps) != 3 {
		t.Fatalf("len(report.Steps) = %d, want 3", len(report.Steps))
	}
	if got := report.Steps[2].Reason; got != "MP 1,2" {
		t.Fatalf("reason = %q, want %q", got, "MP 1,2")
	}
	if got := report.Steps[2].Formula.String(); got != "Q" {
		t.Fatalf("final formula = %q, want %q", got, "Q")
	}
}

func TestVerifyCertificate_InvalidAssumptionRef(t *testing.T) {
	cert := loadFixture(t, filepath.Join("invalid", "bad_assumption_ref.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), "assumption_ref 2 exceeds assumptions length 1") {
		t.Fatalf("VerifyCertificate() error = %v, want assumption_ref failure", err)
	}
}

func TestVerifyCertificate_InvalidAxiomInstance(t *testing.T) {
	cert := loadFixture(t, filepath.Join("invalid", "bad_axiom_instance.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), "does not match axiom A1") {
		t.Fatalf("VerifyCertificate() error = %v, want axiom failure", err)
	}
}

func TestVerifyCertificate_InvalidModusPonensOrder(t *testing.T) {
	cert := loadFixture(t, filepath.Join("invalid", "bad_modus_ponens_order.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), "MP mismatch") {
		t.Fatalf("VerifyCertificate() error = %v, want MP ordering failure", err)
	}
}

func TestVerifyCertificate_UnsupportedSyntaxMessage(t *testing.T) {
	cert := loadFixture(t, filepath.Join("conformance", "invalid_unsupported_syntax.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), `unsupported syntax "hilbert-prop-utf8-v1" for rule pack "classical-hilbert-v1"`) {
		t.Fatalf("VerifyCertificate() error = %v, want unsupported syntax failure", err)
	}
}

func TestVerifyCertificate_FinalGoalMismatchMessage(t *testing.T) {
	cert := loadFixture(t, filepath.Join("conformance", "invalid_final_goal_mismatch.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), `final goal mismatch, expected "Q" but got "P"`) {
		t.Fatalf("VerifyCertificate() error = %v, want final goal mismatch failure", err)
	}
}

func TestVerifyCertificate_ModusPonensFutureReferenceMessage(t *testing.T) {
	cert := loadFixture(t, filepath.Join("conformance", "invalid_mp_future_reference.json"))
	_, err := VerifyCertificate(cert)
	if err == nil || !strings.Contains(err.Error(), "MP premises must reference previous lines only") {
		t.Fatalf("VerifyCertificate() error = %v, want future-reference failure", err)
	}
}

func TestVerifyCertificate_DeterministicReplay(t *testing.T) {
	cert := loadFixture(t, filepath.Join("valid", "a_implies_a.json"))

	report1, err := VerifyCertificate(cert)
	if err != nil {
		t.Fatalf("VerifyCertificate() error = %v", err)
	}
	report2, err := VerifyCertificate(cert)
	if err != nil {
		t.Fatalf("VerifyCertificate() second run error = %v", err)
	}
	if report1.String() != report2.String() {
		t.Fatalf("report replay mismatch:\n1=%s\n2=%s", report1.String(), report2.String())
	}
}
