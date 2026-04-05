package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	certapp "github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/application"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/naturaldeduction"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

func main() {
	args, restoreWorkspaceRootOverride, err := applyWorkspaceRootOverride(os.Args[1:])
	if err != nil {
		fail(err)
	}
	defer restoreWorkspaceRootOverride()
	if len(args) == 0 {
		printUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "run-hilbert-benchmark":
		if err := runHilbertBenchmarkCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "run-nd-hilbert-benchmark":
		if err := runNaturalDeductionBenchmarkCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "run-lean4-research-job":
		if err := runLean4ResearchJobCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "generate-lean4-hypothesis-cases":
		if err := runGenerateLean4HypothesisCasesCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "export-lean4-hypothesis-cases":
		if err := runExportLean4HypothesisCasesCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "hilbert-benchmark-summary":
		if err := runHilbertBenchmarkSummaryCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "hilbert-benchmark-report":
		if err := runHilbertBenchmarkReportCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "hilbert-benchmark-research-report":
		if err := runHilbertBenchmarkResearchReportCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "hilbert-benchmark-meta-report":
		if err := runHilbertBenchmarkMetaReportCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "hilbert-benchmark-case-report":
		if err := runHilbertBenchmarkCaseReportCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "research-db-sync":
		if err := runResearchDBSyncCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "research-db-init":
		if err := runResearchDBInitCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "research-db-reset":
		if err := runResearchDBResetCommand(args[1:], os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
	case "check-certificates":
		if err := checkCertificates(); err != nil {
			fail(err)
		}
	case "check-natural-deduction":
		if err := checkNaturalDeduction(); err != nil {
			fail(err)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/contracts [--workspace-root PATH] <check-certificates|check-natural-deduction|run-hilbert-benchmark|run-nd-hilbert-benchmark|run-lean4-research-job|generate-lean4-hypothesis-cases|export-lean4-hypothesis-cases|hilbert-benchmark-summary|hilbert-benchmark-report|hilbert-benchmark-research-report|hilbert-benchmark-meta-report|hilbert-benchmark-case-report|research-db-init|research-db-reset|research-db-sync>")
	fmt.Fprintln(os.Stderr, "global options:")
	fmt.Fprintln(os.Stderr, "  --workspace-root <path>   override detected workspace root (env: COLLABSPHERE_WORKSPACE_ROOT, RESEARCH_WORKSPACE_ROOT)")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func checkCertificates() error {
	service := certapp.NewService()

	validFixtures := []string{
		mustResolveCertificatesPath("testdata", "valid", "a_implies_a.json"),
		mustResolveCertificatesPath("testdata", "valid", "assumption_goal.json"),
	}
	for _, path := range validFixtures {
		result, err := service.VerifyFile(context.Background(), path)
		if err != nil {
			return fmt.Errorf("verify valid certificate %q: %w", path, err)
		}
		if result == nil || result.Certificate == nil || result.Report == nil {
			return fmt.Errorf("verify valid certificate %q: incomplete verification result", path)
		}
	}

	invalidFixtures := []struct {
		path     string
		wantCode string
	}{
		{
			path:     mustResolveCertificatesPath("testdata", "invalid", "bad_schema_missing_proof_id.json"),
			wantCode: certapp.CodeSchemaInvalid,
		},
		{
			path:     mustResolveCertificatesPath("testdata", "invalid", "bad_parse_formula.json"),
			wantCode: certapp.CodeParseFailed,
		},
		{
			path:     mustResolveCertificatesPath("testdata", "invalid", "bad_assumption_ref.json"),
			wantCode: certapp.CodeKernelRejected,
		},
		{
			path:     filepath.Join(mustResolveCertificatesPath("testdata", "invalid"), "does-not-exist.json"),
			wantCode: certapp.CodeIOFailed,
		},
	}

	for _, tc := range invalidFixtures {
		result, err := service.VerifyFile(context.Background(), tc.path)
		if err == nil {
			return fmt.Errorf("verify invalid certificate %q: unexpected success with result %#v", tc.path, result)
		}
		if got := certapp.CodeOf(err); got != tc.wantCode {
			return fmt.Errorf("verify invalid certificate %q: code %q, want %q", tc.path, got, tc.wantCode)
		}
	}

	return nil
}

func checkNaturalDeduction() error {
	validFixtures := []string{
		mustResolveToolingPath("internal", "prooftheory", "naturaldeduction", "testdata", "valid", "identity.json"),
		mustResolveToolingPath("internal", "prooftheory", "naturaldeduction", "testdata", "valid", "premise_mp.json"),
	}
	for _, path := range validFixtures {
		proof, err := naturaldeduction.LoadJSONFile(path)
		if err != nil {
			return fmt.Errorf("load valid natural deduction proof %q: %w", path, err)
		}
		if proof == nil {
			return fmt.Errorf("load valid natural deduction proof %q: nil proof", path)
		}
		cert, err := naturaldeduction.LowerToCertificate(proof, naturaldeduction.LoweringOptions{})
		if err != nil {
			return fmt.Errorf("lower valid natural deduction proof %q: %w", path, err)
		}
		if _, err := hilbert.VerifyCertificate(cert); err != nil {
			return fmt.Errorf("verify lowered natural deduction proof %q: %w", path, err)
		}
	}

	invalidFixtures := []struct {
		path      string
		wantClass naturaldeduction.ErrorClass
	}{
		{
			path:      mustResolveToolingPath("internal", "prooftheory", "naturaldeduction", "testdata", "invalid", "bad_schema_missing_proof_id.json"),
			wantClass: naturaldeduction.ErrorClassSchema,
		},
		{
			path:      mustResolveToolingPath("internal", "prooftheory", "naturaldeduction", "testdata", "invalid", "bad_parse_negation.json"),
			wantClass: naturaldeduction.ErrorClassParse,
		},
		{
			path:      mustResolveToolingPath("internal", "prooftheory", "naturaldeduction", "testdata", "invalid", "bad_validation_closed_scope_reference.json"),
			wantClass: naturaldeduction.ErrorClassValidation,
		},
	}

	for _, tc := range invalidFixtures {
		proof, err := naturaldeduction.LoadJSONFile(tc.path)
		if err == nil {
			return fmt.Errorf("load invalid natural deduction proof %q: unexpected success with proof %#v", tc.path, proof)
		}
		if got := naturaldeduction.ClassOf(err); got != tc.wantClass {
			return fmt.Errorf("load invalid natural deduction proof %q: class %q, want %q", tc.path, got, tc.wantClass)
		}
	}

	return nil
}

func mustResolveCertificatesPath(parts ...string) string {
	return toolingpath.FirstExisting(
		toolingpath.PlatformToolingPath(append([]string{"internal", "certificates"}, parts...)...),
		toolingpath.RepoPath(append([]string{"platform-tooling", "internal", "certificates"}, parts...)...),
		filepath.Join(parts...),
	)
}

func mustResolveToolingPath(parts ...string) string {
	return toolingpath.FirstExisting(
		toolingpath.PlatformToolingPath(parts...),
		toolingpath.RepoPath(append([]string{"platform-tooling"}, parts...)...),
		filepath.Join(parts...),
		filepath.Join(append([]string{"..", ".."}, parts...)...),
	)
}
