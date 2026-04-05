package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

func TestRun_ValidCertificate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{toolingpath.PlatformToolingPath("internal", "certificates", "testdata", "valid", "a_implies_a.json")}, &stdout, &stderr)
	if exitCode != exitOK {
		t.Fatalf("exitCode = %d, want %d, stderr=%s", exitCode, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "certificate accepted: example-a-implies-a") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRun_ExplainMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--explain", toolingpath.PlatformToolingPath("internal", "certificates", "testdata", "valid", "assumption_goal.json")}, &stdout, &stderr)
	if exitCode != exitOK {
		t.Fatalf("exitCode = %d, want %d, stderr=%s", exitCode, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "1. P [assumption 1]") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRun_InvalidCertificate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{toolingpath.PlatformToolingPath("internal", "certificates", "testdata", "invalid", "bad_assumption_ref.json")}, &stdout, &stderr)
	if exitCode != exitKernelError {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitKernelError)
	}
	if !strings.Contains(stderr.String(), "kernel_validation_error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "assumption_ref") {
		t.Fatalf("stderr = %q, want lower-layer detail", stderr.String())
	}
}

func TestRun_ParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{toolingpath.PlatformToolingPath("internal", "certificates", "testdata", "invalid", "bad_parse_formula.json")}, &stdout, &stderr)
	if exitCode != exitParseError {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitParseError)
	}
	if !strings.Contains(stderr.String(), "parse_error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRun_SchemaError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{toolingpath.PlatformToolingPath("internal", "certificates", "testdata", "invalid", "bad_schema_missing_proof_id.json")}, &stdout, &stderr)
	if exitCode != exitSchemaError {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitSchemaError)
	}
	if !strings.Contains(stderr.String(), "schema_error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRun_InternalError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"does-not-exist.json"}, &stdout, &stderr)
	if exitCode != exitInternalError {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitInternalError)
	}
	if !strings.Contains(stderr.String(), "internal_error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRun_UsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(nil, &stdout, &stderr)
	if exitCode != exitUsageError {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitUsageError)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
