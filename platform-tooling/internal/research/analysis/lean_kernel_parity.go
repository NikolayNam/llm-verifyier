package analysis

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
)

const (
	LeanKernelParityProjectFolderDefault = "hilbert-ai-verification-lean4-kernel-parity-v1"
	leanKernelParityModuleName           = "kernel_parity"
	leanKernelParityDefaultBatchSize     = 100
)

type LeanKernelParityOptions struct {
	WorkspaceDir  string
	ResultDir     string
	ProjectFolder string
	RunID         string
	CorpusRoot    string
	MaxCases      int
	LeanBatchSize int
	LakeBinary    string
	Timeout       time.Duration
}

type LeanKernelParityArtifacts struct {
	ProjectFolder        string
	RunID                string
	CorpusRoot           string
	WorkspaceDir         string
	ResultDir            string
	ModulePath           string
	CorpusManifestPath   string
	GoVerdictsPath       string
	LeanVerdictsPath     string
	ComparisonCSVPath    string
	MismatchManifestPath string
	MismatchCasesDir     string
	MismatchSummaryPath  string
	ReportMarkdownPath   string
	ReportJSONPath       string
	LeanStdoutPath       string
	LeanStderrPath       string
	LeanExitCode         int
	BatchCount           int
	Cases                []LeanKernelParityCase
	GoVerdicts           []ParityCaseVerdict
	LeanVerdicts         []ParityCaseVerdict
	ComparisonRows       []LeanKernelParityComparisonRow
}

type LeanKernelParityCase struct {
	CaseID       string
	Path         string
	LeanEligible bool
	RawJSON      []byte
	Raw          leanRawCertificate
}

type ParityVerdict struct {
	Accepted   bool
	Class      string
	ReasonCode string
	Detail     string
}

type ParityCaseVerdict struct {
	CaseID  string
	Path    string
	Verdict ParityVerdict
}

type LeanKernelParityComparisonRow struct {
	CaseID            string
	Path              string
	LeanEligible      bool
	GoAccepted        bool
	LeanAccepted      bool
	GoClass           string
	LeanClass         string
	GoReasonCode      string
	LeanReasonCode    string
	BehaviorMatch     bool
	DetailMatch       bool
	GoBehaviorBytes   string
	LeanBehaviorBytes string
	GoDetailBytes     string
	LeanDetailBytes   string
	GoDetail          string
	LeanDetail        string
	Note              string
}

type leanRawContext struct {
	Domain    string `json:"domain"`
	RulePack  string `json:"rule_pack"`
	Syntax    string `json:"syntax"`
	Generator string `json:"generator"`
}

type leanRawSourceRef struct {
	ID string `json:"id"`
}

type leanRawStep struct {
	Kind          string             `json:"kind"`
	AssumptionRef int                `json:"assumption_ref"`
	Axiom         string             `json:"axiom"`
	Premises      []int              `json:"premises"`
	Formula       string             `json:"formula"`
	SourceRefs    []leanRawSourceRef `json:"source_refs"`
}

type leanRawCertificate struct {
	CertificateVersion string         `json:"certificate_version"`
	ProofID            string         `json:"proof_id"`
	Goal               string         `json:"goal"`
	Assumptions        []string       `json:"assumptions"`
	Context            leanRawContext `json:"context"`
	Steps              []leanRawStep  `json:"steps"`
}

type leanExecutionReport struct {
	GeneratedAtUTC string `json:"generated_at_utc"`
	WorkspaceDir   string `json:"workspace_dir"`
	ResultDir      string `json:"result_dir"`
	ModulePath     string `json:"module_path"`
	LakeBinary     string `json:"lake_binary"`
	CorpusRoot     string `json:"corpus_root"`
	CorpusLimit    int    `json:"corpus_limit,omitempty"`
	LeanBatchSize  int    `json:"lean_batch_size,omitempty"`
	BatchCount     int    `json:"batch_count,omitempty"`
	RunID          string `json:"run_id"`
	ProjectFolder  string `json:"project_folder"`
	LeanExitCode   int    `json:"lean_exit_code"`
}

func AnalyzeLeanKernelParity(ctx context.Context, opts LeanKernelParityOptions) (*LeanKernelParityArtifacts, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(opts.WorkspaceDir) == "" {
		return nil, fmt.Errorf("workspace dir is required")
	}
	if strings.TrimSpace(opts.ResultDir) == "" {
		return nil, fmt.Errorf("result dir is required")
	}
	if strings.TrimSpace(opts.CorpusRoot) == "" {
		return nil, fmt.Errorf("corpus root is required")
	}
	if strings.TrimSpace(opts.RunID) == "" {
		return nil, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(opts.LakeBinary) == "" {
		opts.LakeBinary = lean4worker.DefaultLakeBinary
	}
	if opts.Timeout <= 0 {
		opts.Timeout = lean4worker.DefaultRunTimeout
	}
	if opts.LeanBatchSize <= 0 {
		opts.LeanBatchSize = leanKernelParityDefaultBatchSize
	}
	if err := os.MkdirAll(opts.ResultDir, 0o755); err != nil {
		return nil, fmt.Errorf("create result dir: %w", err)
	}

	cases, err := discoverLeanKernelParityCorpus(opts.CorpusRoot, opts.MaxCases)
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("no json certificate files found in %s", filepath.ToSlash(opts.CorpusRoot))
	}

	goVerdicts := make([]ParityCaseVerdict, 0, len(cases))
	leanCases := make([]LeanKernelParityCase, 0, len(cases))
	for _, corpusCase := range cases {
		goVerdicts = append(goVerdicts, ParityCaseVerdict{
			CaseID:  corpusCase.CaseID,
			Path:    corpusCase.Path,
			Verdict: evaluateGoParityVerdict(corpusCase.RawJSON),
		})
		if corpusCase.LeanEligible {
			leanCases = append(leanCases, corpusCase)
		}
	}

	if err := lean4worker.EnsureProjectScaffold(opts.WorkspaceDir); err != nil {
		return nil, err
	}
	if err := bootstrapLeanKernelParityWorkspace(ctx, opts.WorkspaceDir, opts.LakeBinary); err != nil {
		return nil, err
	}
	batchSize := opts.LeanBatchSize
	if len(leanCases) > 0 && batchSize > len(leanCases) {
		batchSize = len(leanCases)
	}
	if batchSize <= 0 {
		batchSize = 1
	}
	batchCount := 0
	var leanStdoutBuilder strings.Builder
	var leanStderrBuilder strings.Builder
	leanVerdicts := make([]ParityCaseVerdict, 0, len(leanCases))
	leanExitCode := 0
	runErr := error(nil)
	generatedArtifactPath := filepath.Join(opts.ResultDir, "Generated.lean")
	if len(leanCases) > batchSize {
		generatedArtifactPath = filepath.Join(opts.ResultDir, "batches")
	}
	for start := 0; start < len(leanCases); start += batchSize {
		end := start + batchSize
		if end > len(leanCases) {
			end = len(leanCases)
		}
		batchCount++
		batchLabel := fmt.Sprintf("batch-%03d", batchCount)
		batchCases := leanCases[start:end]

		moduleDir := filepath.Join(opts.WorkspaceDir, lean4worker.DefaultGeneratedDir, opts.RunID, leanKernelParityModuleName, batchLabel)
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			return nil, fmt.Errorf("create lean parity module dir: %w", err)
		}
		modulePath := filepath.Join(moduleDir, "Generated.lean")
		moduleText := renderLeanKernelParityModule(batchCases)
		if err := os.WriteFile(modulePath, []byte(moduleText), 0o644); err != nil {
			return nil, fmt.Errorf("write lean parity module: %w", err)
		}

		batchArtifactDir := opts.ResultDir
		if len(leanCases) > batchSize {
			batchArtifactDir = filepath.Join(opts.ResultDir, "batches", batchLabel)
		}
		if err := os.MkdirAll(batchArtifactDir, 0o755); err != nil {
			return nil, fmt.Errorf("create lean parity batch artifact dir: %w", err)
		}
		batchGeneratedArtifactPath := filepath.Join(batchArtifactDir, "Generated.lean")
		if err := os.WriteFile(batchGeneratedArtifactPath, []byte(moduleText), 0o644); err != nil {
			return nil, fmt.Errorf("write generated lean artifact copy: %w", err)
		}

		batchStdout, batchStderr, batchExitCode, batchRunErr := runLeanKernelParityModule(ctx, opts.WorkspaceDir, opts.LakeBinary, opts.Timeout, modulePath)
		if leanStdoutBuilder.Len() > 0 {
			leanStdoutBuilder.WriteString("\n")
		}
		if len(leanCases) > batchSize {
			leanStdoutBuilder.WriteString("## ")
			leanStdoutBuilder.WriteString(batchLabel)
			leanStdoutBuilder.WriteString("\n")
		}
		leanStdoutBuilder.WriteString(batchStdout)
		if batchStderr != "" {
			if leanStderrBuilder.Len() > 0 {
				leanStderrBuilder.WriteString("\n")
			}
			if len(leanCases) > batchSize {
				leanStderrBuilder.WriteString("## ")
				leanStderrBuilder.WriteString(batchLabel)
				leanStderrBuilder.WriteString("\n")
			}
			leanStderrBuilder.WriteString(batchStderr)
		}
		if err := os.WriteFile(filepath.Join(batchArtifactDir, "lean_stdout.txt"), []byte(batchStdout), 0o644); err != nil {
			return nil, fmt.Errorf("write lean batch stdout: %w", err)
		}
		if err := os.WriteFile(filepath.Join(batchArtifactDir, "lean_stderr.txt"), []byte(batchStderr), 0o644); err != nil {
			return nil, fmt.Errorf("write lean batch stderr: %w", err)
		}
		if batchRunErr != nil {
			leanExitCode = batchExitCode
			runErr = fmt.Errorf("batch %s: %w", batchLabel, batchRunErr)
			break
		}
		batchVerdicts, err := parseLeanKernelParityOutput(batchStdout, batchCases)
		if err != nil {
			leanExitCode = batchExitCode
			runErr = fmt.Errorf("batch %s: %w", batchLabel, err)
			break
		}
		leanVerdicts = append(leanVerdicts, batchVerdicts...)
	}
	leanStdout := leanStdoutBuilder.String()
	leanStderr := leanStderrBuilder.String()
	leanStdoutPath := filepath.Join(opts.ResultDir, "lean_stdout.txt")
	leanStderrPath := filepath.Join(opts.ResultDir, "lean_stderr.txt")
	if err := os.WriteFile(leanStdoutPath, []byte(leanStdout), 0o644); err != nil {
		return nil, fmt.Errorf("write lean stdout: %w", err)
	}
	if err := os.WriteFile(leanStderrPath, []byte(leanStderr), 0o644); err != nil {
		return nil, fmt.Errorf("write lean stderr: %w", err)
	}
	reportPath := filepath.Join(opts.ResultDir, "report.json")
	reportBytes, err := json.MarshalIndent(leanExecutionReport{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		WorkspaceDir:   filepath.ToSlash(opts.WorkspaceDir),
		ResultDir:      filepath.ToSlash(opts.ResultDir),
		ModulePath:     filepath.ToSlash(generatedArtifactPath),
		LakeBinary:     opts.LakeBinary,
		CorpusRoot:     filepath.ToSlash(opts.CorpusRoot),
		CorpusLimit:    opts.MaxCases,
		LeanBatchSize:  batchSize,
		BatchCount:     batchCount,
		RunID:          opts.RunID,
		ProjectFolder:  strings.TrimSpace(opts.ProjectFolder),
		LeanExitCode:   leanExitCode,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal lean parity report: %w", err)
	}
	if err := os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write lean parity report: %w", err)
	}
	if runErr != nil {
		return nil, fmt.Errorf("run lean kernel parity module: %w", runErr)
	}
	comparisonRows := compareLeanKernelParityVerdicts(cases, goVerdicts, leanVerdicts)

	artifacts := &LeanKernelParityArtifacts{
		ProjectFolder:        strings.TrimSpace(opts.ProjectFolder),
		RunID:                opts.RunID,
		CorpusRoot:           filepath.ToSlash(opts.CorpusRoot),
		WorkspaceDir:         filepath.ToSlash(opts.WorkspaceDir),
		ResultDir:            filepath.ToSlash(opts.ResultDir),
		ModulePath:           filepath.ToSlash(generatedArtifactPath),
		CorpusManifestPath:   filepath.ToSlash(filepath.Join(opts.ResultDir, "corpus_manifest.csv")),
		GoVerdictsPath:       filepath.ToSlash(filepath.Join(opts.ResultDir, "go_verdicts.csv")),
		LeanVerdictsPath:     filepath.ToSlash(filepath.Join(opts.ResultDir, "lean_verdicts.csv")),
		ComparisonCSVPath:    filepath.ToSlash(filepath.Join(opts.ResultDir, "comparison.csv")),
		MismatchManifestPath: filepath.ToSlash(filepath.Join(opts.ResultDir, "mismatch_manifest.csv")),
		MismatchCasesDir:     filepath.ToSlash(filepath.Join(opts.ResultDir, "mismatch_cases")),
		MismatchSummaryPath:  filepath.ToSlash(filepath.Join(opts.ResultDir, "mismatch_summary.md")),
		ReportMarkdownPath:   filepath.ToSlash(filepath.Join(opts.ResultDir, "report.md")),
		ReportJSONPath:       filepath.ToSlash(reportPath),
		LeanStdoutPath:       filepath.ToSlash(leanStdoutPath),
		LeanStderrPath:       filepath.ToSlash(leanStderrPath),
		LeanExitCode:         leanExitCode,
		BatchCount:           batchCount,
		Cases:                cases,
		GoVerdicts:           goVerdicts,
		LeanVerdicts:         leanVerdicts,
		ComparisonRows:       comparisonRows,
	}
	if err := writeLeanKernelParityCorpusManifest(artifacts.CorpusManifestPath, cases); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityVerdictsCSV(artifacts.GoVerdictsPath, goVerdicts); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityVerdictsCSV(artifacts.LeanVerdictsPath, leanVerdicts); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityComparisonCSV(artifacts.ComparisonCSVPath, comparisonRows); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityMismatchArtifacts(artifacts.MismatchManifestPath, artifacts.MismatchCasesDir, cases, comparisonRows); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityMismatchSummary(artifacts.MismatchSummaryPath, comparisonRows); err != nil {
		return nil, err
	}
	if err := writeLeanKernelParityMarkdown(artifacts.ReportMarkdownPath, artifacts); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func discoverLeanKernelParityCorpus(root string, maxCases int) ([]LeanKernelParityCase, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	entries := make([]LeanKernelParityCase, 0, 32)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".json" {
			return nil
		}
		rawBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read corpus file %s: %w", filepath.ToSlash(path), err)
		}
		rawCert, _ := decodeLeanRawCertificate(rawBytes)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", filepath.ToSlash(path), err)
		}
		entries = append(entries, LeanKernelParityCase{
			CaseID:       filepath.ToSlash(rel),
			Path:         filepath.ToSlash(path),
			LeanEligible: true,
			RawJSON:      append([]byte(nil), rawBytes...),
			Raw:          rawCert,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk corpus root %s: %w", filepath.ToSlash(root), err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CaseID < entries[j].CaseID })
	if maxCases > 0 && len(entries) > maxCases {
		entries = entries[:maxCases]
	}
	return entries, nil
}

func decodeLeanRawCertificate(raw []byte) (leanRawCertificate, bool) {
	var cert leanRawCertificate
	if err := json.Unmarshal(raw, &cert); err != nil {
		return leanRawCertificate{}, false
	}
	return cert, true
}

func evaluateGoParityVerdict(rawJSON []byte) ParityVerdict {
	cert, err := certificates.DecodeJSON(rawJSON)
	if err != nil {
		detail := strings.TrimSpace(err.Error())
		return ParityVerdict{
			Accepted:   false,
			Class:      string(certificates.ClassOf(err)),
			ReasonCode: classifyGoParityFailure(detail, certificates.ClassOf(err)),
			Detail:     detail,
		}
	}
	_, err = hilbert.VerifyCertificate(cert)
	if err != nil {
		detail := strings.TrimSpace(err.Error())
		return ParityVerdict{
			Accepted:   false,
			Class:      string(certificates.ClassOf(err)),
			ReasonCode: classifyGoParityFailure(detail, certificates.ClassOf(err)),
			Detail:     detail,
		}
	}
	return ParityVerdict{Accepted: true, Class: "accepted", ReasonCode: "accepted"}
}

func classifyGoParityFailure(detail string, class certificates.ErrorClass) string {
	detail = strings.TrimSpace(detail)
	switch class {
	case certificates.ErrorClassSchema:
		switch {
		case strings.Contains(detail, "decode certificate json:"):
			return "json_decode_error"
		case strings.Contains(detail, "certificate_version must be"):
			return "invalid_certificate_version"
		case strings.Contains(detail, "proof_id is required"):
			return "missing_proof_id"
		case strings.Contains(detail, "goal is required"):
			return "missing_goal"
		case strings.Contains(detail, "context.domain is required"):
			return "missing_context_domain"
		case strings.Contains(detail, "context.rule_pack is required"):
			return "missing_context_rule_pack"
		case strings.Contains(detail, "context.syntax is required"):
			return "missing_context_syntax"
		case strings.Contains(detail, "steps must not be empty"):
			return "missing_steps"
		case strings.Contains(detail, "assumptions[") && strings.Contains(detail, "must not be blank"):
			return "blank_assumption"
		case strings.Contains(detail, "duplicates an earlier assumption"):
			return "duplicate_assumption"
		case strings.Contains(detail, "formula is required"):
			return "missing_step_formula"
		case strings.Contains(detail, "assumption_ref must be positive"):
			return "non_positive_assumption_ref"
		case strings.Contains(detail, "assumption step must not define axiom"):
			return "assumption_declares_axiom"
		case strings.Contains(detail, "assumption step must not define premises"):
			return "assumption_declares_premises"
		case strings.Contains(detail, "axiom step requires axiom"):
			return "missing_axiom_name"
		case strings.Contains(detail, "axiom step must not define assumption_ref"):
			return "axiom_declares_assumption_ref"
		case strings.Contains(detail, "axiom step must not define premises"):
			return "axiom_declares_premises"
		case strings.Contains(detail, "modus_ponens requires exactly 2 premises"):
			return "mp_wrong_premise_arity"
		case strings.Contains(detail, "modus_ponens must not define assumption_ref"):
			return "mp_declares_assumption_ref"
		case strings.Contains(detail, "modus_ponens must not define axiom"):
			return "mp_declares_axiom"
		case strings.Contains(detail, "source_refs[") && strings.Contains(detail, ".id is required"):
			return "missing_source_ref_id"
		case strings.Contains(detail, "unsupported step kind"):
			return "unsupported_step_kind"
		default:
			return "unknown_schema_failure"
		}
	case certificates.ErrorClassParse:
		switch {
		case strings.Contains(detail, "unsupported syntax"):
			return "unsupported_syntax"
		case strings.Contains(detail, "parse goal:"):
			return "goal_parse_error"
		case strings.Contains(detail, "parse assumptions["):
			return "assumption_parse_error"
		case strings.Contains(detail, "parse formula:"):
			return "step_formula_parse_error"
		default:
			return "unknown_parse_failure"
		}
	case certificates.ErrorClassKernel:
		switch {
		case strings.Contains(detail, "unsupported rule pack"):
			return "unsupported_rule_pack"
		case strings.Contains(detail, "assumption_ref") && strings.Contains(detail, "exceeds assumptions length"):
			return "assumption_ref_out_of_range"
		case strings.Contains(detail, "assumption mismatch"):
			return "assumption_mismatch"
		case strings.Contains(detail, "unknown axiom"):
			return "unknown_axiom"
		case strings.Contains(detail, "does not match axiom"):
			return "axiom_instance_mismatch"
		case strings.Contains(detail, "MP premises must be positive"):
			return "mp_non_positive_premise"
		case strings.Contains(detail, "MP premises must reference previous lines only"):
			return "mp_future_reference"
		case strings.Contains(detail, "is not an implication, cannot use MP"):
			return "mp_non_implication_premise"
		case strings.Contains(detail, "implication antecedent is"):
			return "mp_antecedent_mismatch"
		case strings.Contains(detail, "expected consequent"):
			return "mp_consequent_mismatch"
		case strings.Contains(detail, "unsupported step kind"):
			return "unsupported_step_kind"
		case strings.Contains(detail, "final goal mismatch"):
			return "final_goal_mismatch"
		default:
			return "unknown_kernel_failure"
		}
	default:
		return "unknown_internal_failure"
	}
}

func renderLeanKernelParityModule(cases []LeanKernelParityCase) string {
	var b strings.Builder
	b.WriteString(lean4worker.KernelParityModuleSource())
	b.WriteString("\n")
	b.WriteString("open CollabSphereLean\n\n")
	b.WriteString(renderLeanKernelParityDecoderSupport())
	b.WriteString("\n")
	b.WriteString("def corpus : List RawParityCase := [\n")
	for idx, corpusCase := range cases {
		if idx > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  { caseId := ")
		b.WriteString(leanStringLiteral(corpusCase.CaseID))
		b.WriteString(", rawJson := ")
		b.WriteString(leanStringLiteral(string(corpusCase.RawJSON)))
		b.WriteString(", cert := ")
		b.WriteString(renderLeanRawCertificate(corpusCase.Raw))
		b.WriteString(" }")
	}
	b.WriteString("\n]\n\n")
	b.WriteString("def main : IO Unit := do\n")
	b.WriteString("  IO.println (renderRawCorpusResults corpus)\n")
	return b.String()
}

func renderLeanKernelParityDecoderSupport() string {
	return `
structure RawParityCase where
  caseId : String
  rawJson : String
  cert : RawCertificate
deriving Repr

def jsonDecodeError (detail : String) : Failure :=
  schemaFailure "json_decode_error" ("decode certificate json: " ++ detail)

def firstSegmentBefore (needle value : String) : String :=
  match value.splitOn needle with
  | [] => value
  | head :: _ => head

def containsSubstring (value needle : String) : Bool :=
  match value.splitOn needle with
  | [] => false
  | [_] => false
  | _ => true

def containsUnknownConclusionAdmissionPattern (rawJson : String) : Bool :=
  containsSubstring rawJson "\"conclusion\""

def containsObjectAssumptionsAdmissionPattern (rawJson : String) : Bool :=
  containsSubstring (firstSegmentBefore "\"steps\"" rawJson) "\"assumptions\"" &&
    containsSubstring (firstSegmentBefore "\"steps\"" rawJson) "\"formula\""

def precheckCertificateJSON (rawJson : String) : Except Failure Unit := do
  match Lean.Json.parse rawJson with
  | .ok _ => pure ()
  | .error detail => throw <| jsonDecodeError detail
  if containsUnknownConclusionAdmissionPattern rawJson then
    throw <| jsonDecodeError "json: unknown field \"conclusion\""
  if containsObjectAssumptionsAdmissionPattern rawJson then
    throw <| jsonDecodeError "json: cannot unmarshal object into Go struct field Certificate.assumptions of type string"

def verifyRawCertificateJSON (rawJson : String) (fallback : RawCertificate) : Verdict :=
  match precheckCertificateJSON rawJson with
  | .error failure => failureVerdict failure
  | .ok _ => verifyCertificate fallback

def renderRawCorpusResults (cases : List RawParityCase) : String :=
  String.intercalate "\n" <| cases.map fun entry =>
    renderVerdictRow entry.caseId (verifyRawCertificateJSON entry.rawJson entry.cert)
`
}

func renderLeanRawCertificate(cert leanRawCertificate) string {
	var b strings.Builder
	b.WriteString("{ certificateVersion := ")
	b.WriteString(leanStringLiteral(cert.CertificateVersion))
	b.WriteString(", proofId := ")
	b.WriteString(leanStringLiteral(cert.ProofID))
	b.WriteString(", goal := ")
	b.WriteString(leanStringLiteral(cert.Goal))
	b.WriteString(", assumptions := [")
	for idx, assumption := range cert.Assumptions {
		if idx > 0 {
			b.WriteString(", ")
		}
		b.WriteString(leanStringLiteral(assumption))
	}
	b.WriteString("], context := { domain := ")
	b.WriteString(leanStringLiteral(cert.Context.Domain))
	b.WriteString(", rulePack := ")
	b.WriteString(leanStringLiteral(cert.Context.RulePack))
	b.WriteString(", syntaxProfile := ")
	b.WriteString(leanStringLiteral(cert.Context.Syntax))
	b.WriteString(", generator := ")
	b.WriteString(leanStringLiteral(cert.Context.Generator))
	b.WriteString(" }, steps := [")
	for idx, step := range cert.Steps {
		if idx > 0 {
			b.WriteString(", ")
		}
		b.WriteString("{ kind := ")
		b.WriteString(leanStringLiteral(step.Kind))
		b.WriteString(", assumptionRef := ")
		b.WriteString(fmt.Sprintf("%d", step.AssumptionRef))
		b.WriteString(", axiomName := ")
		b.WriteString(leanStringLiteral(step.Axiom))
		b.WriteString(", premises := [")
		for premiseIdx, premise := range step.Premises {
			if premiseIdx > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%d", premise))
		}
		b.WriteString("], formula := ")
		b.WriteString(leanStringLiteral(step.Formula))
		b.WriteString(", sourceRefs := [")
		for sourceIdx, source := range step.SourceRefs {
			if sourceIdx > 0 {
				b.WriteString(", ")
			}
			b.WriteString("{ id := ")
			b.WriteString(leanStringLiteral(source.ID))
			b.WriteString(" }")
		}
		b.WriteString("] }")
	}
	b.WriteString("] }")
	return b.String()
}

func leanStringLiteral(value string) string {
	quoted, err := json.Marshal(value)
	if err != nil {
		return "\"\""
	}
	return string(quoted)
}

func bootstrapLeanKernelParityWorkspace(ctx context.Context, projectDir, lakeBinary string) error {
	commands := make([][]string, 0, 3)
	if !dirExists(filepath.Join(projectDir, ".lake", "packages", "mathlib")) {
		commands = append(commands, []string{"update"}, []string{"exe", "cache", "get"})
	}
	commands = append(commands, []string{"build"})
	for _, args := range commands {
		cmd := exec.CommandContext(ctx, lakeBinary, args...)
		cmd.Dir = projectDir
		cmd.Env = leanKernelParityCommandEnv(projectDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %s %s: %w\n%s", lakeBinary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runLeanKernelParityModule(ctx context.Context, projectDir, lakeBinary string, timeout time.Duration, modulePath string) (string, string, int, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, lakeBinary, "env", "lean", "--run", modulePath)
	cmd.Dir = projectDir
	cmd.Env = leanKernelParityCommandEnv(projectDir)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err == nil {
		return stdoutBuf.String(), stderrBuf.String(), 0, nil
	}
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

func leanKernelParityCommandEnv(projectDir string) []string {
	base := make([]string, 0, len(os.Environ())+8)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GIT_CONFIG_COUNT=") ||
			strings.HasPrefix(entry, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(entry, "GIT_CONFIG_VALUE_") {
			continue
		}
		base = append(base, entry)
	}
	dirs := leanKernelParitySafeDirectories(projectDir)
	if len(dirs) == 0 {
		return base
	}
	base = append(base, fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(dirs)))
	for idx, dir := range dirs {
		base = append(base,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=safe.directory", idx),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", idx, dir),
		)
	}
	return base
}

func leanKernelParitySafeDirectories(projectDir string) []string {
	seen := map[string]struct{}{}
	dirs := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
	}
	add(projectDir)
	packagesRoot := filepath.Join(projectDir, ".lake", "packages")
	add(packagesRoot)
	entries, err := os.ReadDir(packagesRoot)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				add(filepath.Join(packagesRoot, entry.Name()))
			}
		}
	}
	sort.Strings(dirs)
	return dirs
}

func parseLeanKernelParityOutput(stdout string, cases []LeanKernelParityCase) ([]ParityCaseVerdict, error) {
	caseByID := make(map[string]LeanKernelParityCase, len(cases))
	for _, corpusCase := range cases {
		caseByID[corpusCase.CaseID] = corpusCase
	}
	lines := strings.Split(strings.ReplaceAll(stdout, "\r\n", "\n"), "\n")
	records := make([]ParityCaseVerdict, 0, len(lines))
	for _, line := range lines {
		rawLine := strings.TrimRight(line, "\r")
		if strings.TrimSpace(rawLine) == "" {
			continue
		}
		parts := strings.SplitN(rawLine, "\t", 5)
		if len(parts) != 5 {
			if strings.Contains(rawLine, ": warning:") {
				continue
			}
			return nil, fmt.Errorf("parse lean parity output line %q: expected 5 tab-separated columns", rawLine)
		}
		caseID := unescapeLeanKernelParityField(parts[0])
		corpusCase, ok := caseByID[caseID]
		if !ok {
			return nil, fmt.Errorf("lean parity output referenced unknown case_id %q", caseID)
		}
		records = append(records, ParityCaseVerdict{
			CaseID: caseID,
			Path:   corpusCase.Path,
			Verdict: ParityVerdict{
				Accepted:   strings.TrimSpace(parts[1]) == "true",
				Class:      unescapeLeanKernelParityField(parts[2]),
				ReasonCode: unescapeLeanKernelParityField(parts[3]),
				Detail:     unescapeLeanKernelParityField(parts[4]),
			},
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CaseID < records[j].CaseID })
	return records, nil
}

func unescapeLeanKernelParityField(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch != '\\' || i+1 >= len(value) {
			b.WriteByte(ch)
			continue
		}
		i++
		switch value[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte(value[i])
		}
	}
	return b.String()
}

func compareLeanKernelParityVerdicts(cases []LeanKernelParityCase, goVerdicts, leanVerdicts []ParityCaseVerdict) []LeanKernelParityComparisonRow {
	goByID := make(map[string]ParityCaseVerdict, len(goVerdicts))
	leanByID := make(map[string]ParityCaseVerdict, len(leanVerdicts))
	for _, verdict := range goVerdicts {
		goByID[verdict.CaseID] = verdict
	}
	for _, verdict := range leanVerdicts {
		leanByID[verdict.CaseID] = verdict
	}
	rows := make([]LeanKernelParityComparisonRow, 0, len(cases))
	for _, corpusCase := range cases {
		goVerdict, goOK := goByID[corpusCase.CaseID]
		leanVerdict, leanOK := leanByID[corpusCase.CaseID]
		note := ""
		if !corpusCase.LeanEligible {
			note = "lean_ineligible_raw_json"
		} else if !leanOK {
			note = "missing_lean_verdict"
		} else if !goOK {
			note = "missing_go_verdict"
		}

		goBehaviorBytes := ""
		leanBehaviorBytes := ""
		goDetailBytes := ""
		leanDetailBytes := ""
		if goOK {
			goBehaviorBytes = canonicalLeanKernelParityBytes(goVerdict.Verdict, false)
			goDetailBytes = canonicalLeanKernelParityBytes(goVerdict.Verdict, true)
		}
		if leanOK {
			leanBehaviorBytes = canonicalLeanKernelParityBytes(leanVerdict.Verdict, false)
			leanDetailBytes = canonicalLeanKernelParityBytes(leanVerdict.Verdict, true)
		}

		rows = append(rows, LeanKernelParityComparisonRow{
			CaseID:            corpusCase.CaseID,
			Path:              corpusCase.Path,
			LeanEligible:      corpusCase.LeanEligible,
			GoAccepted:        goVerdict.Verdict.Accepted,
			LeanAccepted:      leanVerdict.Verdict.Accepted,
			GoClass:           goVerdict.Verdict.Class,
			LeanClass:         leanVerdict.Verdict.Class,
			GoReasonCode:      goVerdict.Verdict.ReasonCode,
			LeanReasonCode:    leanVerdict.Verdict.ReasonCode,
			BehaviorMatch:     goOK && leanOK && goBehaviorBytes == leanBehaviorBytes,
			DetailMatch:       goOK && leanOK && goDetailBytes == leanDetailBytes,
			GoBehaviorBytes:   goBehaviorBytes,
			LeanBehaviorBytes: leanBehaviorBytes,
			GoDetailBytes:     goDetailBytes,
			LeanDetailBytes:   leanDetailBytes,
			GoDetail:          goVerdict.Verdict.Detail,
			LeanDetail:        leanVerdict.Verdict.Detail,
			Note:              note,
		})
	}
	return rows
}

func canonicalLeanKernelParityBytes(verdict ParityVerdict, includeDetail bool) string {
	payload := map[string]any{
		"accepted":    verdict.Accepted,
		"class":       verdict.Class,
		"reason_code": verdict.ReasonCode,
	}
	if includeDetail {
		payload["detail"] = verdict.Detail
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func writeLeanKernelParityCorpusManifest(path string, cases []LeanKernelParityCase) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create corpus manifest %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"case_id", "path", "lean_eligible"}); err != nil {
		return fmt.Errorf("write corpus manifest header: %w", err)
	}
	for _, corpusCase := range cases {
		if err := writer.Write([]string{
			corpusCase.CaseID,
			filepath.ToSlash(corpusCase.Path),
			fmt.Sprintf("%t", corpusCase.LeanEligible),
		}); err != nil {
			return fmt.Errorf("write corpus manifest row %q: %w", corpusCase.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush corpus manifest %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeLeanKernelParityVerdictsCSV(path string, verdicts []ParityCaseVerdict) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create verdict csv %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"case_id", "path", "accepted", "class", "reason_code", "detail"}); err != nil {
		return fmt.Errorf("write verdict header: %w", err)
	}
	for _, verdict := range verdicts {
		if err := writer.Write([]string{
			verdict.CaseID,
			filepath.ToSlash(verdict.Path),
			fmt.Sprintf("%t", verdict.Verdict.Accepted),
			verdict.Verdict.Class,
			verdict.Verdict.ReasonCode,
			verdict.Verdict.Detail,
		}); err != nil {
			return fmt.Errorf("write verdict row %q: %w", verdict.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush verdict csv %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeLeanKernelParityComparisonCSV(path string, rows []LeanKernelParityComparisonRow) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create comparison csv %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"case_id", "path", "lean_eligible",
		"go_accepted", "lean_accepted", "go_class", "lean_class", "go_reason_code", "lean_reason_code",
		"behavior_match", "detail_match", "go_behavior_bytes", "lean_behavior_bytes",
		"go_detail_bytes", "lean_detail_bytes", "go_detail", "lean_detail", "note",
	}); err != nil {
		return fmt.Errorf("write comparison header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.CaseID,
			filepath.ToSlash(row.Path),
			fmt.Sprintf("%t", row.LeanEligible),
			fmt.Sprintf("%t", row.GoAccepted),
			fmt.Sprintf("%t", row.LeanAccepted),
			row.GoClass,
			row.LeanClass,
			row.GoReasonCode,
			row.LeanReasonCode,
			fmt.Sprintf("%t", row.BehaviorMatch),
			fmt.Sprintf("%t", row.DetailMatch),
			row.GoBehaviorBytes,
			row.LeanBehaviorBytes,
			row.GoDetailBytes,
			row.LeanDetailBytes,
			row.GoDetail,
			row.LeanDetail,
			row.Note,
		}); err != nil {
			return fmt.Errorf("write comparison row %q: %w", row.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush comparison csv %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeLeanKernelParityMismatchArtifacts(manifestPath, casesDir string, cases []LeanKernelParityCase, rows []LeanKernelParityComparisonRow) error {
	if err := ensureParentDir(manifestPath); err != nil {
		return err
	}
	if err := os.RemoveAll(casesDir); err != nil {
		return fmt.Errorf("reset mismatch cases dir %s: %w", filepath.ToSlash(casesDir), err)
	}
	if err := os.MkdirAll(casesDir, 0o755); err != nil {
		return fmt.Errorf("create mismatch cases dir %s: %w", filepath.ToSlash(casesDir), err)
	}

	caseByID := make(map[string]LeanKernelParityCase, len(cases))
	for _, corpusCase := range cases {
		caseByID[corpusCase.CaseID] = corpusCase
	}

	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("create mismatch manifest %s: %w", filepath.ToSlash(manifestPath), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"case_id", "source_path", "mismatch_copy_path",
		"behavior_match", "detail_match",
		"go_class", "go_reason_code", "lean_class", "lean_reason_code",
		"go_detail", "lean_detail", "note",
	}); err != nil {
		return fmt.Errorf("write mismatch manifest header: %w", err)
	}

	for _, row := range rows {
		if row.BehaviorMatch && row.DetailMatch {
			continue
		}
		corpusCase, ok := caseByID[row.CaseID]
		if !ok {
			return fmt.Errorf("mismatch row references unknown case %q", row.CaseID)
		}
		copyPath := filepath.Join(casesDir, filepath.FromSlash(row.CaseID))
		if err := os.MkdirAll(filepath.Dir(copyPath), 0o755); err != nil {
			return fmt.Errorf("create mismatch case parent dir for %s: %w", filepath.ToSlash(copyPath), err)
		}
		if err := os.WriteFile(copyPath, corpusCase.RawJSON, 0o644); err != nil {
			return fmt.Errorf("write mismatch case copy %s: %w", filepath.ToSlash(copyPath), err)
		}
		if err := writer.Write([]string{
			row.CaseID,
			filepath.ToSlash(corpusCase.Path),
			filepath.ToSlash(copyPath),
			fmt.Sprintf("%t", row.BehaviorMatch),
			fmt.Sprintf("%t", row.DetailMatch),
			row.GoClass,
			row.GoReasonCode,
			row.LeanClass,
			row.LeanReasonCode,
			row.GoDetail,
			row.LeanDetail,
			row.Note,
		}); err != nil {
			return fmt.Errorf("write mismatch manifest row %q: %w", row.CaseID, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush mismatch manifest %s: %w", filepath.ToSlash(manifestPath), err)
	}
	return nil
}

func writeLeanKernelParityMismatchSummary(path string, rows []LeanKernelParityComparisonRow) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}

	decoderRows := make([]LeanKernelParityComparisonRow, 0)
	kernelRows := make([]LeanKernelParityComparisonRow, 0)
	for _, row := range rows {
		if row.BehaviorMatch && row.DetailMatch {
			continue
		}
		if leanKernelParityMismatchCategory(row) == "decoder_parity" {
			decoderRows = append(decoderRows, row)
			continue
		}
		kernelRows = append(kernelRows, row)
	}

	var b strings.Builder
	b.WriteString("# Lean4 Kernel Parity Mismatch Summary\n\n")
	b.WriteString("This artifact contains only non-matching cases from the current parity run.\n\n")
	b.WriteString("Classification rule:\n\n")
	b.WriteString("- `decoder_parity` means the mismatch happens before Hilbert proof checking starts: raw JSON admission, schema validation, or formula parse admission.\n")
	b.WriteString("- `kernel_parity` means the mismatch remains after certificate admission into proof checking and therefore points at rule-pack or proof-closure behavior.\n\n")
	b.WriteString("## Totals\n\n")
	b.WriteString(fmt.Sprintf("- total_mismatches: `%d`\n", len(decoderRows)+len(kernelRows)))
	b.WriteString(fmt.Sprintf("- decoder_parity_mismatches: `%d`\n", len(decoderRows)))
	b.WriteString(fmt.Sprintf("- kernel_parity_mismatches: `%d`\n", len(kernelRows)))
	b.WriteString("\n")

	writeLeanKernelParityMismatchSection(&b, "Decoder Parity", decoderRows)
	writeLeanKernelParityMismatchSection(&b, "Kernel Parity", kernelRows)

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeLeanKernelParityMismatchSection(b *strings.Builder, title string, rows []LeanKernelParityComparisonRow) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(rows) == 0 {
		b.WriteString("None.\n\n")
		return
	}
	b.WriteString("| Case | Go class | Lean class | Go reason | Lean reason | Note |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
			row.CaseID,
			row.GoClass,
			row.LeanClass,
			row.GoReasonCode,
			row.LeanReasonCode,
			firstNonEmptyString(row.Note, "mismatch"),
		))
	}
	b.WriteString("\n")
}

func leanKernelParityMismatchCategory(row LeanKernelParityComparisonRow) string {
	if strings.TrimSpace(row.Note) == "lean_ineligible_raw_json" {
		return "decoder_parity"
	}
	if leanKernelParityIsPreKernelClass(row.GoClass) || leanKernelParityIsPreKernelClass(row.LeanClass) {
		return "decoder_parity"
	}
	return "kernel_parity"
}

func leanKernelParityIsPreKernelClass(class string) bool {
	switch strings.TrimSpace(class) {
	case "schema_error", "parse_error":
		return true
	default:
		return false
	}
}

func writeLeanKernelParityMarkdown(path string, artifacts *LeanKernelParityArtifacts) error {
	if artifacts == nil {
		return fmt.Errorf("lean kernel parity artifacts are required")
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}

	total := len(artifacts.ComparisonRows)
	leanEligible := 0
	behaviorMatches := 0
	detailMatches := 0
	mismatches := make([]LeanKernelParityComparisonRow, 0, total)
	for _, row := range artifacts.ComparisonRows {
		if row.LeanEligible {
			leanEligible++
		}
		if row.BehaviorMatch {
			behaviorMatches++
		} else {
			mismatches = append(mismatches, row)
		}
		if row.DetailMatch {
			detailMatches++
		}
	}

	var b strings.Builder
	b.WriteString("# Lean4 Kernel Parity Report\n\n")
	b.WriteString("Status: generated artifact  \n")
	b.WriteString("Scope: compare the repo-tracked certificate corpus across the current Go Hilbert checker and a replicated Lean4 minimal checker\n\n")
	b.WriteString(fmt.Sprintf("- project_folder: `%s`\n", artifacts.ProjectFolder))
	b.WriteString(fmt.Sprintf("- run_id: `%s`\n", artifacts.RunID))
	b.WriteString(fmt.Sprintf("- corpus_root: `%s`\n", artifacts.CorpusRoot))
	if total > 0 {
		b.WriteString(fmt.Sprintf("- corpus_selection: first `%d` sorted JSON files under corpus root\n", total))
	}
	b.WriteString(fmt.Sprintf("- total_cases: `%d`\n", total))
	b.WriteString(fmt.Sprintf("- lean_eligible_cases: `%d`\n", leanEligible))
	b.WriteString(fmt.Sprintf("- behavior_matches: `%d`\n", behaviorMatches))
	b.WriteString(fmt.Sprintf("- detail_matches: `%d`\n", detailMatches))
	b.WriteString(fmt.Sprintf("- lean_exit_code: `%d`\n", artifacts.LeanExitCode))
	b.WriteString("\n")

	b.WriteString("## Artifact Paths\n\n")
	b.WriteString(fmt.Sprintf("- corpus_manifest: `%s`\n", artifacts.CorpusManifestPath))
	b.WriteString(fmt.Sprintf("- go_verdicts: `%s`\n", artifacts.GoVerdictsPath))
	b.WriteString(fmt.Sprintf("- lean_verdicts: `%s`\n", artifacts.LeanVerdictsPath))
	b.WriteString(fmt.Sprintf("- comparison_csv: `%s`\n", artifacts.ComparisonCSVPath))
	b.WriteString(fmt.Sprintf("- mismatch_manifest: `%s`\n", artifacts.MismatchManifestPath))
	b.WriteString(fmt.Sprintf("- mismatch_cases_dir: `%s`\n", artifacts.MismatchCasesDir))
	b.WriteString(fmt.Sprintf("- mismatch_summary: `%s`\n", artifacts.MismatchSummaryPath))
	if artifacts.BatchCount > 1 {
		b.WriteString(fmt.Sprintf("- generated_modules_dir: `%s`\n", artifacts.ModulePath))
		b.WriteString(fmt.Sprintf("- batch_count: `%d`\n", artifacts.BatchCount))
	} else {
		b.WriteString(fmt.Sprintf("- generated_module: `%s`\n", artifacts.ModulePath))
	}
	b.WriteString(fmt.Sprintf("- lean_stdout: `%s`\n", artifacts.LeanStdoutPath))
	b.WriteString(fmt.Sprintf("- lean_stderr: `%s`\n", artifacts.LeanStderrPath))
	b.WriteString("\n")

	if len(mismatches) == 0 {
		b.WriteString("## Verdict\n\n")
		b.WriteString("No behavior mismatches were detected on the current fixed corpus.\n\n")
	} else {
		b.WriteString("## Behavior Mismatches\n\n")
		b.WriteString("| Case | Go | Lean | Go reason | Lean reason | Note |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, row := range mismatches {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
				row.CaseID,
				row.GoClass,
				row.LeanClass,
				row.GoReasonCode,
				row.LeanReasonCode,
				firstNonEmptyString(row.Note, "behavior_mismatch"),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Notes\n\n")
	b.WriteString("- Primary parity criterion is canonicalized `accepted + class + reason_code`, not human-readable wording.\n")
	b.WriteString("- `detail_match` is tracked separately because exact message text is weaker evidence than verdict behavior.\n")
	b.WriteString("- This v1 surface targets minimal checker parity on the fixed corpus; malformed-JSON / unknown-field CLI parity remains a separate extension step.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
