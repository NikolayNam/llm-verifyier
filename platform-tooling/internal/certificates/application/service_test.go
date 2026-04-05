package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceVerifyFileAndJSONMatch(t *testing.T) {
	service := NewService()
	path := filepath.Join("..", "testdata", "valid", "a_implies_a.json")

	fromFile, err := service.VerifyFile(context.Background(), path)
	if err != nil {
		t.Fatalf("VerifyFile() error = %v", err)
	}

	raw := mustReadFile(t, path)
	fromJSON, err := service.VerifyJSON(context.Background(), raw)
	if err != nil {
		t.Fatalf("VerifyJSON() error = %v", err)
	}

	if fromFile.Certificate.ProofID != fromJSON.Certificate.ProofID {
		t.Fatalf("proof id mismatch: %q vs %q", fromFile.Certificate.ProofID, fromJSON.Certificate.ProofID)
	}
	if fromFile.Report.String() != fromJSON.Report.String() {
		t.Fatalf("report mismatch:\nfile=%s\njson=%s", fromFile.Report.String(), fromJSON.Report.String())
	}
}

func TestServiceVerifyJSONMapsVerificationCodes(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		path     string
		wantCode string
	}{
		{
			name:     "schema",
			path:     filepath.Join("..", "testdata", "invalid", "bad_schema_missing_proof_id.json"),
			wantCode: CodeSchemaInvalid,
		},
		{
			name:     "parse",
			path:     filepath.Join("..", "testdata", "invalid", "bad_parse_formula.json"),
			wantCode: CodeParseFailed,
		},
		{
			name:     "kernel",
			path:     filepath.Join("..", "testdata", "invalid", "bad_assumption_ref.json"),
			wantCode: CodeKernelRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.VerifyJSON(context.Background(), mustReadFile(t, tt.path))
			if err == nil {
				t.Fatal("VerifyJSON() error = nil, want non-nil")
			}
			appErr, ok := AsVerificationError(err)
			if !ok || appErr == nil {
				t.Fatalf("AsVerificationError(%v) failed", err)
			}
			if appErr.Code != tt.wantCode {
				t.Fatalf("verification code = %q, want %q", appErr.Code, tt.wantCode)
			}
			if appErr.Cause == nil {
				t.Fatal("verification cause is nil")
			}
			if tt.wantCode == CodeKernelRejected && appErr.Message != "" && !strings.Contains(appErr.Message, "assumption_ref") {
				t.Fatalf("kernel error message = %q, want lower-layer detail", appErr.Message)
			}
		})
	}
}

func TestServiceVerifyFileMapsIOFailure(t *testing.T) {
	service := NewService()

	_, err := service.VerifyFile(context.Background(), "does-not-exist.json")
	if err == nil {
		t.Fatal("VerifyFile() error = nil, want non-nil")
	}
	appErr, ok := AsVerificationError(err)
	if !ok || appErr == nil {
		t.Fatalf("AsVerificationError(%v) failed", err)
	}
	if appErr.Code != CodeIOFailed {
		t.Fatalf("verification code = %q, want %q", appErr.Code, CodeIOFailed)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}
