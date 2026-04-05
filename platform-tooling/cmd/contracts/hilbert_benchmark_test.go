package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

type fakeBenchmarkModel struct {
	outputs map[string]string
	errors  map[string]error
	calls   []string
	prompts map[string]string
}

func (f *fakeBenchmarkModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, time.Duration, error) {
	_ = ctx
	_ = systemPrompt
	caseID := extractCaseID(userPrompt)
	f.calls = append(f.calls, caseID)
	if f.prompts == nil {
		f.prompts = make(map[string]string)
	}
	f.prompts[caseID] = userPrompt
	if err := f.errors[caseID]; err != nil {
		return "", 15 * time.Millisecond, err
	}
	return f.outputs[caseID], 15 * time.Millisecond, nil
}

type fakeBenchmarkVerifier struct {
	results map[string]*benchmarkVerifyResult
}

func (f *fakeBenchmarkVerifier) Verify(ctx context.Context, certificatePath string) (*benchmarkVerifyResult, error) {
	_ = ctx
	caseID := strings.TrimSuffix(filepath.Base(certificatePath), filepath.Ext(certificatePath))
	if result, ok := f.results[caseID]; ok {
		return result, nil
	}
	return &benchmarkVerifyResult{ExitCode: 0}, nil
}

func writeBenchmarkSummaryCSV(t *testing.T, path string, rows ...benchmarkRunSummaryRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}
	projectFolder := ""
	if len(rows) > 0 {
		projectFolder = rows[0].ProjectFolder
	}
	header, _, err := benchmarkRunSummaryHeaderForPath(path, projectFolder)
	if err != nil {
		t.Fatalf("benchmarkRunSummaryHeaderForPath() error = %v", err)
	}
	lines := []string{strings.Join(header, ",")}
	for _, row := range rows {
		lines = append(lines, strings.Join(serializeBenchmarkRunSummaryRow(row, header), ","))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(summary csv) error = %v", err)
	}
}

func writeBenchmarkResultCSV(t *testing.T, path string, rows ...benchmarkResultRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(result dir) error = %v", err)
	}
	header := benchmarkCategoryResultsHeader()
	lines := []string{strings.Join(header, ",")}
	for _, row := range rows {
		lines = append(lines, strings.Join(serializeBenchmarkResultRow(row, header), ","))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result csv) error = %v", err)
	}
}

func createPhase1ObservedFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkV2HeldOutProjectFolder)
	casesPath := filepath.Join(benchmarkRoot, "cases.csv")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")
	resultPath := filepath.Join(benchmarkRoot, "result", "result_run-phase1.csv")
	rawDir := filepath.Join(benchmarkRoot, "raw", "run-phase1")

	casesCSV := strings.Join([]string{
		"case_id,category,label,difficulty,assumptions_json,goal,comment",
		`V2E01,assumption_import,entailed,easy,"[""R -> S""]","R -> S",Import a single assumption directly`,
		`V2E06,direct_axiom_instance,entailed,medium,[],"(R -> (S -> T)) -> ((R -> S) -> (R -> T))",Direct A2 instance on held-out atoms`,
		`V2E07,direct_axiom_instance,entailed,medium,[],"(!S -> !R) -> (R -> S)",Direct A3 instance on held-out atoms`,
		`V2E08,mixed_proof,entailed,hard,"[""!S -> !R"",""R""]",S,Mixed A3 plus assumption derivation`,
		`V2E12,theorem_synthesis,entailed,hard,[],"R -> R",Theorem synthesis without top-level assumptions`,
		`V2N07,negative_refusal,not_entailed,medium,"[""R -> (S -> T)"",""R""]",T,Intermediate antecedent is missing`,
		`V2N09,negative_refusal,not_entailed,hard,"[""R -> (S -> T)"",""R -> S""]",T,Potential chain still lacks R`,
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(casesPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(cases dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte(casesCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(raw dir) error = %v", err)
	}

	summaryRow := benchmarkRunSummaryRow{
		RunID:               "20260330_phase1_direct_gpt-oss-20b_r01",
		TimestampUTC:        "2026-03-30T08:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkV2HeldOutProjectFolder,
		Surface:             "direct",
		CasesFile:           "cases.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           casesPath,
		ResultsPath:         resultPath,
		RawDir:              rawDir,
		CasesTotal:          "7",
		PassCount:           "3",
		FalseRefusalCount:   "1",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "1",
		ParseFailureCount:   "0",
		KernelFailureCount:  "1",
		FormatFailureCount:  "1",
		AvgLatencyMS:        "21.00",
		MaxLatencyMS:        "33",
		RunElapsedSeconds:   "72",
		CategoryCount:       "5",
	}
	writeBenchmarkSummaryCSV(t, summaryPath, summaryRow, summaryRow)

	baseRows := []benchmarkResultRow{
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:01Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2E01", TheoremID: "V2E01", Category: "assumption_import", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "10", Notes: "verified"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:02Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2E06", TheoremID: "V2E06", Category: "direct_axiom_instance", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "fail", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "schema_failure", LatencyMS: "12", Notes: "schema mismatch"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:03Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2E07", TheoremID: "V2E07", Category: "direct_axiom_instance", ExpectedLabel: "entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "false_refusal", LatencyMS: "18", Notes: "refused derivable case"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:04Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2E08", TheoremID: "V2E08", Category: "mixed_proof", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "reject", ScoreBucket: "kernel_failure", LatencyMS: "20", Notes: "kernel rejected"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:05Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2E12", TheoremID: "V2E12", Category: "theorem_synthesis", ExpectedLabel: "entailed", RawOutputKind: "other", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "format_failure", LatencyMS: "33", Notes: "wrapper text"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:06Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2N07", TheoremID: "V2N07", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "pass", LatencyMS: "25", Notes: "correct refusal"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T08:00:07Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2N09", TheoremID: "V2N09", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "pass", LatencyMS: "29", Notes: "correct refusal"},
	}
	rows := append([]benchmarkResultRow{}, baseRows...)
	rows = append(rows, baseRows[0])
	writeBenchmarkResultCSV(t, resultPath, rows...)

	return casesPath, summaryPath
}

func createPhase1ExpandedObservedFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkV2HeldOutProjectFolder)
	casesPath := filepath.Join(benchmarkRoot, "cases", "phase1-expanded-100.csv")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")
	resultPath := filepath.Join(benchmarkRoot, "result", "result_run-phase1-expanded.csv")
	rawDir := filepath.Join(benchmarkRoot, "raw", "run-phase1-expanded")

	casesCSV := strings.Join([]string{
		"case_id,source_case_id,category,label,difficulty,assumptions_json,goal,comment",
		`V2XE001,V2E01,assumption_import,entailed,easy,"[""P001 -> Q001""]","P001 -> Q001",Expanded from V2E01 with canonical atom renaming #001. Import a single assumption directly`,
		`V2XE006,V2E06,direct_axiom_instance,entailed,medium,[],"(P006 -> (Q006 -> R006)) -> ((P006 -> Q006) -> (P006 -> R006))",Expanded from V2E06 with canonical atom renaming #006. Direct A2 instance on held-out atoms`,
		`V2XE007,V2E07,direct_axiom_instance,entailed,medium,[],"(!Q007 -> !P007) -> (P007 -> Q007)",Expanded from V2E07 with canonical atom renaming #007. Direct A3 instance on held-out atoms`,
		`V2XE008,V2E08,mixed_proof,entailed,hard,"[""!Q008 -> !P008"",""P008""]",Q008,Expanded from V2E08 with canonical atom renaming #008. Mixed A3 plus assumption derivation`,
		`V2XE012,V2E12,theorem_synthesis,entailed,hard,[],"P012 -> P012",Expanded from V2E12 with canonical atom renaming #012. Theorem synthesis without top-level assumptions`,
		`V2XN007,V2N07,negative_refusal,not_entailed,medium,"[""P007 -> (Q007 -> R007)"",""P007""]",R007,Expanded from V2N07 with canonical atom renaming #007. Intermediate antecedent is missing`,
		`V2XN009,V2N09,negative_refusal,not_entailed,hard,"[""P009 -> (Q009 -> R009)"",""P009 -> Q009""]",R009,Expanded from V2N09 with canonical atom renaming #009. Potential chain still lacks P009`,
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(casesPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(cases dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte(casesCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(expanded cases csv) error = %v", err)
	}
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(raw dir) error = %v", err)
	}

	summaryRow := benchmarkRunSummaryRow{
		RunID:               "20260404_phase1_direct_expanded_qwen_r01",
		TimestampUTC:        "2026-04-04T08:00:00Z",
		Model:               "compatible:qwen3.5:397b-cloud",
		LLMModel:            "qwen3.5:397b-cloud",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkV2HeldOutProjectFolder,
		Surface:             "direct",
		CasesFile:           "cases/phase1-expanded-100.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           casesPath,
		ResultsPath:         resultPath,
		RawDir:              rawDir,
		CasesTotal:          "7",
		PassCount:           "4",
		FalseRefusalCount:   "1",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "1",
		ParseFailureCount:   "0",
		KernelFailureCount:  "1",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "17.00",
		MaxLatencyMS:        "24",
		RunElapsedSeconds:   "49",
		CategoryCount:       "5",
	}
	writeBenchmarkSummaryCSV(t, summaryPath, summaryRow, summaryRow)

	baseRows := []benchmarkResultRow{
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:01Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XE001", TheoremID: "V2XE001", Category: "assumption_import", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "11", Notes: "verified"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:02Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XE006", TheoremID: "V2XE006", Category: "direct_axiom_instance", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "fail", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "schema_failure", LatencyMS: "14", Notes: "schema mismatch"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:03Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XE007", TheoremID: "V2XE007", Category: "direct_axiom_instance", ExpectedLabel: "entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "false_refusal", LatencyMS: "16", Notes: "refused derivable case"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:04Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XE008", TheoremID: "V2XE008", Category: "mixed_proof", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "reject", ScoreBucket: "kernel_failure", LatencyMS: "19", Notes: "kernel rejected"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:05Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XE012", TheoremID: "V2XE012", Category: "theorem_synthesis", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "20", Notes: "verified"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:06Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XN007", TheoremID: "V2XN007", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "pass", LatencyMS: "24", Notes: "correct refusal"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-04-04T08:00:07Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "direct", CaseID: "V2XN009", TheoremID: "V2XN009", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "pass", LatencyMS: "15", Notes: "correct refusal"},
	}
	rows := append([]benchmarkResultRow{}, baseRows...)
	rows = append(rows, baseRows[0])
	writeBenchmarkResultCSV(t, resultPath, rows...)

	return casesPath, summaryPath
}

func createNDObservedFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkNDProjectFolder)
	theoremsPath := filepath.Join(benchmarkRoot, "theorems", "pilot_shared_20260327.csv")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary_v2.csv")
	resultPath := filepath.Join(benchmarkRoot, "result", "result_run-phase2-direct.csv")
	rawDir := filepath.Join(benchmarkRoot, "raw", "run-phase2-direct")

	theoremsCSV := strings.Join([]string{
		"theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment",
		`pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI03,NDI03,mp_chain,entailed,medium,"[""P"",""P -> Q"",""Q -> R""]",R,implicational-prop-v1,Two-step implication chain`,
		`pilot_shared_20260327,NDI05,NDI05,theorem_synthesis,entailed,medium,[],"P -> P",implicational-prop-v1,Identity theorem without premises`,
		`pilot_shared_20260327,NDI07,NDI07,negative_refusal,not_entailed,easy,"[""P -> Q""]",P,implicational-prop-v1,Consequent does not imply antecedent`,
		`pilot_shared_20260327,NDI08,NDI08,negative_refusal,not_entailed,easy,"[""P""]",Q,implicational-prop-v1,Irrelevant atomic goal should be refused`,
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(theoremsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(theorems dir) error = %v", err)
	}
	if err := os.WriteFile(theoremsPath, []byte(theoremsCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(theorems.csv) error = %v", err)
	}
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(raw dir) error = %v", err)
	}

	summaryRow := benchmarkRunSummaryRow{
		RunID:               "20260330_phase2_nd_gpt-oss-20b_r01",
		TimestampUTC:        "2026-03-30T09:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkNDProjectFolder,
		Surface:             "nd",
		TheoremPackID:       "pilot_shared_20260327",
		TheoremsFile:        benchmarkNDTheoremPack,
		CaseSelector:        "all",
		Hypothesis:          ndBenchmarkHypothesis,
		TheoremsPath:        theoremsPath,
		ResultsPath:         resultPath,
		RawDir:              rawDir,
		CasesTotal:          "5",
		PassCount:           "2",
		FalseRefusalCount:   "1",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "1",
		ParseFailureCount:   "0",
		KernelFailureCount:  "1",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "18.00",
		MaxLatencyMS:        "31",
		RunElapsedSeconds:   "61",
		CategoryCount:       "4",
	}
	writeBenchmarkSummaryCSV(t, summaryPath, summaryRow, summaryRow)

	baseRows := []benchmarkResultRow{
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T09:00:01Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "pass", PipelineStage: "hilbert_artifact", PipelineDetail: "pass", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI01", CaseID: "NDI01", Category: "assumption_import", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "11", Notes: "verified"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T09:00:02Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "nd", NDProofStatus: "fail", LoweringStatus: "not_run", PipelineStage: "nd_proof_object", PipelineDetail: "schema_error", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI03", CaseID: "NDI03", Category: "mp_chain", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "fail", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "schema_failure", LatencyMS: "14", Notes: "schema mismatch"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T09:00:03Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "nd", NDProofStatus: "not_run", LoweringStatus: "not_run", PipelineStage: "llm_output", PipelineDetail: "not_derivable", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI05", CaseID: "NDI05", Category: "theorem_synthesis", ExpectedLabel: "entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "false_refusal", LatencyMS: "19", Notes: "refused derivable theorem"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T09:00:04Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "fail", PipelineStage: "lowering", PipelineDetail: "validation_error", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI07", CaseID: "NDI07", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "not_run", ScoreBucket: "kernel_failure", LatencyMS: "15", Notes: "lowering failed verification"},
		{RunID: summaryRow.RunID, TimestampUTC: "2026-03-30T09:00:05Z", Model: summaryRow.Model, LLMModel: summaryRow.LLMModel, PromptVersion: summaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "pass", PipelineStage: "hilbert_artifact", PipelineDetail: "kernel_failure", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI08", CaseID: "NDI08", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "reject", ScoreBucket: "kernel_failure", LatencyMS: "31", Notes: "kernel rejected"},
	}
	rows := append([]benchmarkResultRow{}, baseRows...)
	rows = append(rows, baseRows[0])
	writeBenchmarkResultCSV(t, resultPath, rows...)

	return theoremsPath, summaryPath
}

func TestRunHilbertBenchmarkWritesRowsAndRawOutputs(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,label,difficulty,assumptions_json,goal,comment",
		`H01,entailed,easy,"[""P""]",P,Import a single assumption directly`,
		`H11,not_entailed,easy,"[""P""]",Q,Unrelated target`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "run-001",
		LLMModel:      "test-model",
		PromptVersion: benchmarkPromptVersionV1,
		ModelLabel:    "compatible:test-model",
	}, benchmarkPromptProfiles[benchmarkPromptVersionV1], &fakeBenchmarkModel{
		outputs: map[string]string{
			"H01": `{"certificate_version":"1.0.0","proof_id":"H01","goal":"P","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":["P"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"P"}]}`,
			"H11": "NOT_DERIVABLE",
		},
	}, &fakeBenchmarkVerifier{
		results: map[string]*benchmarkVerifyResult{
			"H01": {ExitCode: 0},
		},
	}, func() time.Time {
		return time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runHilbertBenchmark() error = %v", err)
	}

	if summary.CaseCount != 2 {
		t.Fatalf("summary.CaseCount = %d, want 2", summary.CaseCount)
	}
	if summary.PromptVersion != benchmarkPromptVersionV1 {
		t.Fatalf("summary.PromptVersion = %q, want %q", summary.PromptVersion, benchmarkPromptVersionV1)
	}
	if summary.Buckets["pass"] != 2 {
		t.Fatalf("summary.Buckets[pass] = %d, want 2", summary.Buckets["pass"])
	}
	if summary.AvgLatencyMS != 15 {
		t.Fatalf("summary.AvgLatencyMS = %.2f, want 15.00", summary.AvgLatencyMS)
	}
	if summary.MaxLatencyMS != 15 {
		t.Fatalf("summary.MaxLatencyMS = %d, want 15", summary.MaxLatencyMS)
	}
	if summary.RunElapsedS != 0 {
		t.Fatalf("summary.RunElapsedS = %d, want 0", summary.RunElapsedS)
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	resultsText := string(resultsRaw)
	if !strings.Contains(resultsText, "H01") || !strings.Contains(resultsText, "H11") {
		t.Fatalf("results.csv missing expected case rows:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, "compatible:test-model") {
		t.Fatalf("results.csv missing model label:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, ",test-model,") {
		t.Fatalf("results.csv missing llm_model:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, benchmarkPromptVersionV1) {
		t.Fatalf("results.csv missing prompt version:\n%s", resultsText)
	}

	rawH01 := filepath.Join(rawDir, "run-001", "H01.txt")
	if _, err := os.Stat(rawH01); err != nil {
		t.Fatalf("raw output for H01 missing: %v", err)
	}
	jsonH01 := filepath.Join(rawDir, "run-001", "H01.json")
	if _, err := os.Stat(jsonH01); err != nil {
		t.Fatalf("candidate json for H01 missing: %v", err)
	}
}

func TestRunHilbertBenchmarkCompositionalWritesTrustedImportArtifacts(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,chain_id,stage_id,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,assumptions_json,imported_lemmas_json,import_stage_ids_json,goal,expected_behavior,comment",
		`CI01-S1,CI01,s1,compositional_assumption_import,entailed,easy,linear_chain,1,none,r0,"[]","[]","[]","A -> B",prove,seed 1`,
		`CI01-S2,CI01,s2,compositional_assumption_import,entailed,easy,linear_chain,1,none,r0,"[]","[]","[]","B -> C",prove,seed 2`,
		`CI01-S3M,CI01,s3m,compositional_assumption_import,entailed,easy,linear_chain,1,model,r0,"[]","[""A -> B"",""B -> C""]","[""s1"",""s2""]","A -> C",prove,model import`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	model := &fakeBenchmarkModel{
		outputs: map[string]string{
			"CI01-S1":  `{"certificate_version":"1.0.0","proof_id":"CI01-S1","goal":"A -> B","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":[],"steps":[{"kind":"axiom","axiom":"A1","formula":"A -> (B -> A)"}]}`,
			"CI01-S2":  `{"certificate_version":"1.0.0","proof_id":"CI01-S2","goal":"B -> C","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":[],"steps":[{"kind":"axiom","axiom":"A1","formula":"B -> (C -> B)"}]}`,
			"CI01-S3M": `{"certificate_version":"1.0.0","proof_id":"CI01-S3M","goal":"A -> C","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":[],"steps":[{"kind":"axiom","axiom":"A2","formula":"(A -> (B -> C)) -> ((A -> B) -> (A -> C))"}]}`,
		},
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		ProjectFolder: "hilbert-ai-verification-benchmark-v2-held-out",
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "compositional-001",
		LLMModel:      "test-model",
		PromptVersion: benchmarkPromptVersionV13,
		ModelLabel:    "compatible:test-model",
		Surface:       "direct",
	}, benchmarkPromptProfiles[benchmarkPromptVersionV13], model, &fakeBenchmarkVerifier{
		results: map[string]*benchmarkVerifyResult{
			"CI01-S1":  {ExitCode: 0},
			"CI01-S2":  {ExitCode: 0},
			"CI01-S3M": {ExitCode: 0},
		},
	}, func() time.Time {
		return time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runHilbertBenchmark() error = %v", err)
	}

	if !strings.Contains(summary.ImportCatalogPath, "imported_certificate_catalog_compositional-001.csv") {
		t.Fatalf("summary.ImportCatalogPath = %q, want imported certificate catalog path", summary.ImportCatalogPath)
	}
	if !strings.Contains(model.calls[len(model.calls)-1], "CI01-S3M") {
		t.Fatalf("last model call = %#v, want CI01-S3M", model.calls)
	}
	prompt := model.prompts["CI01-S3M"]
	if !strings.Contains(prompt, "Imported lemmas:\n1. A -> B\n2. B -> C\n") {
		t.Fatalf("CI01-S3M prompt missing imported lemmas from trusted artifacts:\n%s", prompt)
	}

	resultsRows, err := loadBenchmarkResultRows(resultsPath)
	if err != nil {
		t.Fatalf("loadBenchmarkResultRows() error = %v", err)
	}
	var modelRow benchmarkResultRow
	for _, row := range resultsRows {
		if row.CaseID == "CI01-S3M" {
			modelRow = row
			break
		}
	}
	if !strings.Contains(modelRow.ResolvedImportRefsJSON, "CI01-S1") || !strings.Contains(modelRow.ResolvedImportRefsJSON, "CI01-S2") {
		t.Fatalf("CI01-S3M ResolvedImportRefsJSON = %q, want refs to CI01-S1 and CI01-S2", modelRow.ResolvedImportRefsJSON)
	}

	objectPath := filepath.Join(root, "imported_certificates", "compositional-001", "CI01-S1.json")
	if _, err := os.Stat(objectPath); err != nil {
		t.Fatalf("trusted imported certificate object for CI01-S1 missing: %v", err)
	}
	catalogPath := filepath.Join(root, "imported_certificate_catalog_compositional-001.csv")
	catalogRaw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("ReadFile(imported certificate catalog) error = %v", err)
	}
	catalogText := string(catalogRaw)
	if !strings.Contains(catalogText, "CI01-S1") || !strings.Contains(catalogText, "true") {
		t.Fatalf("imported certificate catalog missing trusted reuse entries:\n%s", catalogText)
	}
}

func TestRunHilbertBenchmarkRequestFailureUsesDedicatedBucket(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,label,difficulty,assumptions_json,goal,comment",
		`H01,entailed,easy,"[""P""]",P,Import a single assumption directly`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "run-req-001",
		LLMModel:      "test-model",
		PromptVersion: benchmarkPromptVersionV11,
		ModelLabel:    "compatible:test-model",
	}, benchmarkPromptProfiles[benchmarkPromptVersionV11], &fakeBenchmarkModel{
		errors: map[string]error{
			"H01": context.DeadlineExceeded,
		},
	}, &fakeBenchmarkVerifier{}, func() time.Time {
		return time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runHilbertBenchmark() error = %v", err)
	}

	if summary.Buckets["request_failure"] != 1 {
		t.Fatalf("summary.Buckets[request_failure] = %d, want 1", summary.Buckets["request_failure"])
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	if !strings.Contains(string(resultsRaw), "request_failure") {
		t.Fatalf("results.csv missing request_failure bucket:\n%s", string(resultsRaw))
	}
}

func TestRunHilbertBenchmarkShortCircuitsRemainingCasesAfterConfiguredTimeoutThreshold(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,label,difficulty,assumptions_json,goal,comment",
		`H01,entailed,easy,"[""P""]",P,Import a single assumption directly`,
		`H02,entailed,easy,"[""P""]",P,Import a single assumption directly`,
		`H03,entailed,easy,"[""P""]",P,Import a single assumption directly`,
		`H04,entailed,easy,"[""P""]",P,Import a single assumption directly`,
		`H05,entailed,easy,"[""P""]",P,Import a single assumption directly`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	model := &fakeBenchmarkModel{
		errors: map[string]error{
			"H01": context.DeadlineExceeded,
			"H02": context.DeadlineExceeded,
			"H03": context.DeadlineExceeded,
		},
		outputs: map[string]string{
			"H04": "NOT_DERIVABLE",
			"H05": "NOT_DERIVABLE",
		},
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:                    casesPath,
		ResultsPath:                  resultsPath,
		RawBaseDir:                   rawDir,
		RunID:                        "run-timeout-abort-001",
		LLMModel:                     "test-model",
		PromptVersion:                benchmarkPromptVersionV11,
		ModelLabel:                   "compatible:test-model",
		RequestTimeoutAbortThreshold: 3,
	}, benchmarkPromptProfiles[benchmarkPromptVersionV11], model, &fakeBenchmarkVerifier{}, func() time.Time {
		return time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runHilbertBenchmark() error = %v", err)
	}

	if got := len(model.calls); got != 3 {
		t.Fatalf("Generate() calls = %d, want 3", got)
	}
	if summary.CaseCount != 5 {
		t.Fatalf("summary.CaseCount = %d, want 5", summary.CaseCount)
	}
	if summary.Buckets["request_failure"] != 5 {
		t.Fatalf("summary.Buckets[request_failure] = %d, want 5", summary.Buckets["request_failure"])
	}
	if summary.AvgLatencyMS != 15 {
		t.Fatalf("summary.AvgLatencyMS = %.2f, want 15.00", summary.AvgLatencyMS)
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	resultsText := string(resultsRaw)
	for _, caseID := range []string{"H01", "H02", "H03", "H04", "H05"} {
		if !strings.Contains(resultsText, caseID) {
			t.Fatalf("results.csv missing case %s:\n%s", caseID, resultsText)
		}
	}
	if !strings.Contains(resultsText, "short-circuited remaining cases after 3 consecutive request timeouts") {
		t.Fatalf("results.csv missing timeout short-circuit note:\n%s", resultsText)
	}

	for _, caseID := range []string{"H04", "H05"} {
		rawPath := filepath.Join(rawDir, "run-timeout-abort-001", caseID+".txt")
		rawText, err := os.ReadFile(rawPath)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", rawPath, err)
		}
		if !strings.Contains(string(rawText), "short-circuited remaining cases after 3 consecutive request timeouts") {
			t.Fatalf("synthetic raw output for %s missing timeout short-circuit note:\n%s", caseID, string(rawText))
		}
	}
}

func TestRunHilbertBenchmarkCategorySummaryAndResultsSchema(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,category,label,difficulty,assumptions_json,goal,comment",
		`V2E01,assumption_import,entailed,easy,"[""R -> S""]","R -> S",Import a single assumption directly`,
		`V2N01,negative_refusal,not_entailed,easy,"[""R""]",S,Unrelated target`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "run-cat-001",
		LLMModel:      "test-model",
		PromptVersion: benchmarkPromptVersionV12,
		ModelLabel:    "compatible:test-model",
	}, benchmarkPromptProfiles[benchmarkPromptVersionV12], &fakeBenchmarkModel{
		outputs: map[string]string{
			"V2E01": `{"certificate_version":"1.0.0","proof_id":"V2E01","goal":"R -> S","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":["R -> S"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"R -> S"}]}`,
			"V2N01": "NOT_DERIVABLE",
		},
	}, &fakeBenchmarkVerifier{
		results: map[string]*benchmarkVerifyResult{
			"V2E01": {ExitCode: 0},
		},
	}, func() time.Time {
		return time.Date(2026, 3, 26, 12, 30, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runHilbertBenchmark() error = %v", err)
	}

	if summary.CategoryStats["assumption_import"]["pass"] != 1 {
		t.Fatalf("assumption_import pass = %d, want 1", summary.CategoryStats["assumption_import"]["pass"])
	}
	if summary.CategoryStats["negative_refusal"]["pass"] != 1 {
		t.Fatalf("negative_refusal pass = %d, want 1", summary.CategoryStats["negative_refusal"]["pass"])
	}
	if summary.CategoryStats["assumption_import"]["cases_total"] != 1 {
		t.Fatalf("assumption_import cases_total = %d, want 1", summary.CategoryStats["assumption_import"]["cases_total"])
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	resultsText := string(resultsRaw)
	if !strings.Contains(resultsText, "category") {
		t.Fatalf("results.csv missing category header:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, "assumption_import") || !strings.Contains(resultsText, "negative_refusal") {
		t.Fatalf("results.csv missing category values:\n%s", resultsText)
	}
}

func TestClassifyBenchmarkRawOutput(t *testing.T) {
	tests := map[string]string{
		"NOT_DERIVABLE":           "not_derivable",
		"  NOT_DERIVABLE  ":       "not_derivable",
		`{"goal":"P"}`:            "certificate",
		`{"goal":"P"`:             "invalid_json",
		"certificate accepted: P": "other",
	}

	for input, want := range tests {
		if got := classifyBenchmarkRawOutput(input); got != want {
			t.Fatalf("classifyBenchmarkRawOutput(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildBenchmarkUserPromptHandlesEmptyAssumptions(t *testing.T) {
	prompt := buildBenchmarkUserPrompt(benchmarkCase{
		CaseID: "H08",
		Goal:   "P -> (Q -> P)",
	})
	if !strings.Contains(prompt, "Assumptions:\n(none)\n") {
		t.Fatalf("prompt does not show empty assumptions:\n%s", prompt)
	}
}

func TestResolveBenchmarkPromptProfile(t *testing.T) {
	if _, err := resolveBenchmarkPromptProfile("unknown"); err == nil {
		t.Fatal("resolveBenchmarkPromptProfile(unknown) expected error")
	}

	for _, version := range []string{
		benchmarkPromptVersionV1,
		benchmarkPromptVersionV11,
		benchmarkPromptVersionV12,
		benchmarkPromptVersionV13,
		benchmarkPromptVersionBridgeOnlyV1,
		benchmarkPromptVersionBridgeOnlySkeletonV1,
		benchmarkPromptVersionBridgeOnlyCanonicalVarsV1,
		benchmarkPromptVersionGoldFinalOnlyV1,
		benchmarkPromptVersionGoldFinalOnlyExplicitImportRefsV1,
	} {
		profile, err := resolveBenchmarkPromptProfile(version)
		if err != nil {
			t.Fatalf("resolveBenchmarkPromptProfile(%q) error = %v", version, err)
		}
		if profile.Version != version {
			t.Fatalf("profile.Version = %q, want %q", profile.Version, version)
		}
		if strings.TrimSpace(profile.SystemPrompt) == "" {
			t.Fatalf("profile.SystemPrompt for %q is empty", version)
		}
	}
}

func TestLoadBenchmarkCasesSupportsCategoryColumn(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	casesCSV := strings.Join([]string{
		"case_id,category,label,difficulty,assumptions_json,goal,comment",
		`V2E01,theorem_synthesis,entailed,hard,[],R -> R,Synthesize theorem without top-level assumptions`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("len(cases) = %d, want 1", len(cases))
	}
	if cases[0].Category != "theorem_synthesis" {
		t.Fatalf("cases[0].Category = %q, want theorem_synthesis", cases[0].Category)
	}
}

func TestBenchmarkPromptV11TightensContract(t *testing.T) {
	profile := benchmarkPromptProfiles[benchmarkPromptVersionV11]
	requiredFragments := []string{
		"certificate_version",
		"proof_id",
		"goal",
		"context",
		"steps",
		`Do not output the fields "line" or "goal_line".`,
		"assumption steps require assumption_ref",
		`"assumptions":["P"]`,
		`"kind":"assumption","assumption_ref":1,"formula":"P"`,
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(profile.SystemPrompt, fragment) {
			t.Fatalf("v1.1 prompt missing fragment %q", fragment)
		}
	}
}

func TestBenchmarkPromptV12AddsAxiomContract(t *testing.T) {
	profile := benchmarkPromptProfiles[benchmarkPromptVersionV12]
	requiredFragments := []string{
		`every step with "kind": "axiom" MUST include the field "axiom"`,
		`"A1", "A2", "A3"`,
		`"proof_id":"H08"`,
		`"kind":"axiom","axiom":"A1","formula":"P -> (Q -> P)"`,
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(profile.SystemPrompt, fragment) {
			t.Fatalf("v1.2 prompt missing fragment %q", fragment)
		}
	}
}

func TestBenchmarkPromptV13RemovesExamplesButKeepsContractTightening(t *testing.T) {
	profile := benchmarkPromptProfiles[benchmarkPromptVersionV13]
	requiredFragments := []string{
		`The context object must contain only:`,
		`do not use "name" instead of "axiom"`,
		`do not invent alternative schema names such as "theorem", "theorems", "axioms", "rules", or "goal_line"`,
		`This prompt intentionally contains no inline benchmark examples.`,
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(profile.SystemPrompt, fragment) {
			t.Fatalf("v1.3 prompt missing fragment %q", fragment)
		}
	}
	forbiddenFragments := []string{
		`"proof_id":"H01"`,
		`"proof_id":"H08"`,
		`Minimal valid shape example`,
	}
	for _, fragment := range forbiddenFragments {
		if strings.Contains(profile.SystemPrompt, fragment) {
			t.Fatalf("v1.3 prompt unexpectedly contains example fragment %q", fragment)
		}
	}
}

func TestBenchmarkRunSummaryHeaderIncludesRunID(t *testing.T) {
	header := benchmarkRunSummaryHeader()
	if len(header) == 0 || header[0] != "run_id" {
		got := ""
		if len(header) > 0 {
			got = header[0]
		}
		t.Fatalf("benchmarkRunSummaryHeader()[0] = %q, want run_id", got)
	}
	required := []string{"prompt_version", "certificate_version", "project_folder", "surface", "cases_file", "case_selector", "result_file", "chain_result_file", "import_catalog_file", "pass_count", "google_thinking_mode"}
	required = append(required, "llm_model")
	required = append(required, "avg_latency_ms", "max_latency_ms", "run_elapsed_seconds")
	for _, column := range required {
		if !slices.Contains(header, column) {
			t.Fatalf("benchmarkRunSummaryHeader() missing %q", column)
		}
	}
}

func TestBenchmarkRunSummaryHeaderV2UsesTheoremFields(t *testing.T) {
	header := benchmarkRunSummaryHeaderV2()
	if len(header) == 0 || header[0] != "run_id" {
		got := ""
		if len(header) > 0 {
			got = header[0]
		}
		t.Fatalf("benchmarkRunSummaryHeaderV2()[0] = %q, want run_id", got)
	}
	required := []string{"prompt_version", "certificate_version", "project_folder", "surface", "theorem_pack_id", "theorems_file", "case_selector", "result_file", "chain_result_file", "import_catalog_file", "pass_count", "theorems_path", "google_thinking_mode"}
	for _, column := range required {
		if !slices.Contains(header, column) {
			t.Fatalf("benchmarkRunSummaryHeaderV2() missing %q", column)
		}
	}
	if slices.Contains(header, "cases_file") || slices.Contains(header, "cases_path") {
		t.Fatalf("benchmarkRunSummaryHeaderV2() must not contain legacy cases_* columns: %v", header)
	}
}

func TestDefaultBenchmarkHistoricalPaths(t *testing.T) {
	root := filepath.Join("research", "artifacts", "hilbert-ai-verification-benchmark-v1")
	runID := "20260326T151500Z"

	resultsPath := filepath.ToSlash(defaultBenchmarkResultsPath(root, runID))
	if resultsPath != "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_20260326T151500Z.csv" {
		t.Fatalf("defaultBenchmarkResultsPath() = %q", resultsPath)
	}

	summaryPath := filepath.ToSlash(defaultBenchmarkSummaryPath(root))
	if summaryPath != "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_summary.csv" {
		t.Fatalf("defaultBenchmarkSummaryPath() = %q", summaryPath)
	}

	ndRoot := filepath.Join("research", "artifacts", benchmarkNDProjectFolder)
	ndSummaryPath := filepath.ToSlash(defaultBenchmarkSummaryPath(ndRoot))
	if ndSummaryPath != "research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/result_summary_v2.csv" {
		t.Fatalf("defaultBenchmarkSummaryPath(ND) = %q", ndSummaryPath)
	}
	ndSummaryPaths := defaultBenchmarkSummaryPaths(ndRoot)
	if len(ndSummaryPaths) != 2 {
		t.Fatalf("len(defaultBenchmarkSummaryPaths(ND)) = %d, want 2", len(ndSummaryPaths))
	}
	if filepath.ToSlash(ndSummaryPaths[0]) != "research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/result_summary_v2.csv" {
		t.Fatalf("defaultBenchmarkSummaryPaths(ND)[0] = %q", filepath.ToSlash(ndSummaryPaths[0]))
	}
	if filepath.ToSlash(ndSummaryPaths[1]) != "research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/result_summary.csv" {
		t.Fatalf("defaultBenchmarkSummaryPaths(ND)[1] = %q", filepath.ToSlash(ndSummaryPaths[1]))
	}
}

func TestAppendBenchmarkRunSummaryWritesRunID(t *testing.T) {
	root := t.TempDir()
	summaryPath := filepath.Join(root, "result", "result_summary.csv")

	err := appendBenchmarkRunSummary(summaryPath, benchmarkRunSummaryRow{
		RunID:               "20260326T151500Z",
		TimestampUTC:        "2026-03-26T15:15:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		GoogleThinkingMode:  "default",
		PromptVersion:       benchmarkPromptVersionV12,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       "hilbert-ai-verification-benchmark-v1",
		Surface:             "direct",
		CasesFile:           "cases.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv",
		ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_20260326T151500Z.csv",
		RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v1/raw/20260326T151500Z",
		CasesTotal:          "16",
		PassCount:           "16",
		FalseRefusalCount:   "0",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "0",
		ParseFailureCount:   "0",
		KernelFailureCount:  "0",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "1234.50",
		MaxLatencyMS:        "4321",
		RunElapsedSeconds:   "17",
		CategoryCount:       "0",
	})
	if err != nil {
		t.Fatalf("appendBenchmarkRunSummary() error = %v", err)
	}

	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile(result_summary.csv) error = %v", err)
	}
	text := string(raw)
	if !strings.HasPrefix(text, "run_id,") {
		t.Fatalf("result_summary.csv missing run_id header:\n%s", text)
	}
	if !strings.Contains(text, "20260326T151500Z") {
		t.Fatalf("result_summary.csv missing run_id row value:\n%s", text)
	}
	if !strings.Contains(text, benchmarkCertificateVersion) {
		t.Fatalf("result_summary.csv missing certificate_version:\n%s", text)
	}
	if !strings.Contains(text, "default") {
		t.Fatalf("result_summary.csv missing google_thinking_mode:\n%s", text)
	}
	if !strings.Contains(text, "gpt-oss:20b") {
		t.Fatalf("result_summary.csv missing llm_model:\n%s", text)
	}
	if !strings.Contains(text, "1234.50,4321,17") {
		t.Fatalf("result_summary.csv missing timing fields:\n%s", text)
	}
	if !strings.Contains(text, "hilbert-ai-verification-benchmark-v1,direct,cases.csv,all") {
		t.Fatalf("result_summary.csv missing project/surface/cases metadata:\n%s", text)
	}
}

func TestAppendBenchmarkRunSummaryDedupesDuplicateRunID(t *testing.T) {
	root := t.TempDir()
	summaryPath := filepath.Join(root, "result", "result_summary.csv")
	row := benchmarkRunSummaryRow{
		RunID:               "20260330_phase1_direct_gpt-oss-20b_r01",
		TimestampUTC:        "2026-03-30T08:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		GoogleThinkingMode:  "default",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkV2HeldOutProjectFolder,
		Surface:             "direct",
		CasesFile:           "cases.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           filepath.Join(root, "cases.csv"),
		ResultsPath:         filepath.Join(root, "result", "result_20260330_phase1_direct_gpt-oss-20b_r01.csv"),
		RawDir:              filepath.Join(root, "raw", "20260330_phase1_direct_gpt-oss-20b_r01"),
		CasesTotal:          "24",
		PassCount:           "11",
		FalseRefusalCount:   "0",
		FalseAcceptCount:    "0",
		RequestFailureCount: "2",
		SchemaFailureCount:  "5",
		ParseFailureCount:   "0",
		KernelFailureCount:  "2",
		FormatFailureCount:  "4",
		AvgLatencyMS:        "18863.04",
		MaxLatencyMS:        "60056",
		RunElapsedSeconds:   "452",
		CategoryCount:       "5",
	}
	if err := appendBenchmarkRunSummary(summaryPath, row); err != nil {
		t.Fatalf("appendBenchmarkRunSummary() first write error = %v", err)
	}
	row.PassCount = "13"
	row.TimestampUTC = "2026-03-30T08:03:00Z"
	row.AvgLatencyMS = "17777.77"
	if err := appendBenchmarkRunSummary(summaryPath, row); err != nil {
		t.Fatalf("appendBenchmarkRunSummary() second write error = %v", err)
	}

	rows, err := loadBenchmarkRunSummaryRows(summaryPath)
	if err != nil {
		t.Fatalf("loadBenchmarkRunSummaryRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].PassCount != "13" || rows[0].AvgLatencyMS != "17777.77" {
		t.Fatalf("deduped row = %+v, want updated latest values", rows[0])
	}
	if rows[0].GoogleThinkingMode != "default" {
		t.Fatalf("deduped row google_thinking_mode = %q, want default", rows[0].GoogleThinkingMode)
	}

	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile(summaryPath) error = %v", err)
	}
	if got := strings.Count(strings.TrimSpace(string(raw)), "\n"); got != 1 {
		t.Fatalf("summary file line count = %d separators, want 1 header/data separator", got)
	}
}

func TestAppendBenchmarkRunSummaryWritesTheoremSummaryV2(t *testing.T) {
	root := t.TempDir()
	summaryPath := filepath.Join(root, "result", "result_summary_v2.csv")

	err := appendBenchmarkRunSummary(summaryPath, benchmarkRunSummaryRow{
		RunID:               "20260327T091500Z",
		TimestampUTC:        "2026-03-27T09:15:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		GoogleThinkingMode:  "low",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkNDProjectFolder,
		Surface:             "nd",
		CasesFile:           benchmarkNDTheoremPack,
		CaseSelector:        "all",
		Hypothesis:          ndBenchmarkHypothesis,
		CasesPath:           filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
		ResultsPath:         filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_20260327T091500Z.csv")),
		RawDir:              filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "20260327T091500Z")),
		CasesTotal:          "8",
		PassCount:           "6",
		FalseRefusalCount:   "1",
		FalseAcceptCount:    "0",
		RequestFailureCount: "1",
		SchemaFailureCount:  "0",
		ParseFailureCount:   "0",
		KernelFailureCount:  "0",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "456.50",
		MaxLatencyMS:        "1200",
		RunElapsedSeconds:   "9",
		CategoryCount:       "4",
	})
	if err != nil {
		t.Fatalf("appendBenchmarkRunSummary(v2) error = %v", err)
	}

	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile(result_summary_v2.csv) error = %v", err)
	}
	text := string(raw)
	if !strings.HasPrefix(text, "run_id,") {
		t.Fatalf("result_summary_v2.csv missing run_id header:\n%s", text)
	}
	if !strings.Contains(text, "theorems_file") || !strings.Contains(text, "theorems_path") || !strings.Contains(text, "theorem_pack_id") {
		t.Fatalf("result_summary_v2.csv missing theorem columns:\n%s", text)
	}
	if strings.Contains(text, "cases_file") || strings.Contains(text, "cases_path") {
		t.Fatalf("result_summary_v2.csv unexpectedly contains legacy columns:\n%s", text)
	}
	if !strings.Contains(text, benchmarkNDTheoremPack) {
		t.Fatalf("result_summary_v2.csv missing theorem pack value:\n%s", text)
	}
	if !strings.Contains(text, ",nd,") {
		t.Fatalf("result_summary_v2.csv missing surface value:\n%s", text)
	}
	if !strings.Contains(text, "pilot_shared_20260327") {
		t.Fatalf("result_summary_v2.csv missing theorem_pack_id value:\n%s", text)
	}
	if !strings.Contains(text, ",low,") {
		t.Fatalf("result_summary_v2.csv missing google_thinking_mode value:\n%s", text)
	}
}

func TestAppendBenchmarkResultsOverwritesDuplicateRunRows(t *testing.T) {
	root := t.TempDir()
	resultsPath := filepath.Join(root, "result", "result_20260330_phase1_direct_gpt-oss-20b_r01.csv")
	row := benchmarkResultRow{
		RunID:              "20260330_phase1_direct_gpt-oss-20b_r01",
		TimestampUTC:       "2026-03-30T08:00:01Z",
		Model:              "compatible:gpt-oss:20b",
		LLMModel:           "gpt-oss:20b",
		GoogleThinkingMode: "high",
		PromptVersion:      benchmarkPromptVersionV13,
		Surface:            "direct",
		CaseID:             "V2E01",
		TheoremID:          "V2E01",
		Category:           "assumption_import",
		ExpectedLabel:      "entailed",
		RawOutputKind:      "certificate",
		SchemaStatus:       "pass",
		ParseStatus:        "pass",
		KernelStatus:       "accept",
		ScoreBucket:        "schema_failure",
		LatencyMS:          "44",
		Notes:              "stale",
	}
	if err := appendBenchmarkResults(resultsPath, []benchmarkResultRow{row}); err != nil {
		t.Fatalf("appendBenchmarkResults() first write error = %v", err)
	}
	row.ScoreBucket = "pass"
	row.LatencyMS = "12"
	row.Notes = "updated"
	if err := appendBenchmarkResults(resultsPath, []benchmarkResultRow{row}); err != nil {
		t.Fatalf("appendBenchmarkResults() second write error = %v", err)
	}

	rows, err := loadBenchmarkResultRows(resultsPath)
	if err != nil {
		t.Fatalf("loadBenchmarkResultRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ScoreBucket != "pass" || rows[0].Notes != "updated" {
		t.Fatalf("result row = %+v, want overwritten latest values", rows[0])
	}
	if rows[0].GoogleThinkingMode != "high" {
		t.Fatalf("result row google_thinking_mode = %q, want high", rows[0].GoogleThinkingMode)
	}
}

func TestRunHilbertBenchmarkSummaryCommandAggregatesByPromptAndCertificateVersion(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v1")
	casesPath := filepath.Join(benchmarkRoot, "cases.csv")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte("case_id,label,difficulty,assumptions_json,goal,comment\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-a",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v1",
			CasesFile:           "cases.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_run-a.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v1/raw/run-a",
			CasesTotal:          "16",
			PassCount:           "16",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "0",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "1000.00",
			MaxLatencyMS:        "1200",
			RunElapsedSeconds:   "20",
			CategoryCount:       "0",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-b",
			TimestampUTC:        "2026-03-26T15:30:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v1",
			CasesFile:           "cases.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_run-b.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v1/raw/run-b",
			CasesTotal:          "8",
			PassCount:           "6",
			FalseRefusalCount:   "1",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "2000.00",
			MaxLatencyMS:        "3100",
			RunElapsedSeconds:   "30",
			CategoryCount:       "0",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-c",
			TimestampUTC:        "2026-03-26T16:00:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV11,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v1",
			CasesFile:           "cases.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_run-c.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v1/raw/run-c",
			CasesTotal:          "16",
			PassCount:           "12",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "2",
			SchemaFailureCount:  "2",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "4000.00",
			MaxLatencyMS:        "5000",
			RunElapsedSeconds:   "45",
			CategoryCount:       "0",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkSummaryCommand([]string{
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkSummaryCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "total_runs: 3") {
		t.Fatalf("summary output missing total_runs:\n%s", output)
	}
	if !strings.Contains(output, "group[model=compatible:gpt-oss:20b llm_model=gpt-oss:20b prompt_version=hilbert-ai-verification-benchmark-v1.2 certificate_version=1.0.0]: runs=2 cases_total=24 pass=22 pass_rate=91.67%") {
		t.Fatalf("summary output missing aggregated v1.2 group:\n%s", output)
	}
	if !strings.Contains(output, "avg_latency_ms=1333.33 max_latency_ms=3100 run_elapsed_seconds_total=50 avg_run_elapsed_seconds=25.00 max_run_elapsed_seconds=30") {
		t.Fatalf("summary output missing aggregated timing fields for v1.2 group:\n%s", output)
	}
	if !strings.Contains(output, "latest_run_id=run-b") {
		t.Fatalf("summary output missing latest run id for v1.2 group:\n%s", output)
	}
	if !strings.Contains(output, "group[model=compatible:gpt-oss:20b llm_model=gpt-oss:20b prompt_version=hilbert-ai-verification-benchmark-v1.1 certificate_version=1.0.0]: runs=1 cases_total=16 pass=12 pass_rate=75.00%") {
		t.Fatalf("summary output missing v1.1 group:\n%s", output)
	}
}

func TestRunHilbertBenchmarkSummaryCommandFiltersByCasesFile(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out")
	casesPath := filepath.Join(benchmarkRoot, "cases", "theorem-synthesis-stress.csv")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(casesPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(cases dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte("case_id,category,label,difficulty,assumptions_json,goal,comment\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-stress",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/theorem-synthesis-stress.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/theorem-synthesis-stress.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-stress.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-stress",
			CasesTotal:          "5",
			PassCount:           "4",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "1500.00",
			MaxLatencyMS:        "1600",
			RunElapsedSeconds:   "21",
			CategoryCount:       "3",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-cross",
			TimestampUTC:        "2026-03-26T15:20:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-cross.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-cross",
			CasesTotal:          "8",
			PassCount:           "7",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "2500.00",
			MaxLatencyMS:        "2600",
			RunElapsedSeconds:   "31",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkSummaryCommand([]string{
		"-project-folder", "hilbert-ai-verification-benchmark-v2-held-out",
		"-cases-file", "cases/theorem-synthesis-stress.csv",
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkSummaryCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "cases_file: cases/theorem-synthesis-stress.csv") {
		t.Fatalf("summary output missing filtered cases_file:\n%s", output)
	}
	if !strings.Contains(output, "total_runs: 1") {
		t.Fatalf("summary output should show a single filtered run:\n%s", output)
	}
	if !strings.Contains(output, "cases_total=5 pass=4 pass_rate=80.00%") {
		t.Fatalf("summary output missing filtered aggregate:\n%s", output)
	}
	if strings.Contains(output, "run-cross") || strings.Contains(output, "cases_total=13") {
		t.Fatalf("summary output unexpectedly includes other cases_file rows:\n%s", output)
	}
}

func TestRunHilbertBenchmarkSummaryCommandRendersPhase1ObservedSlices(t *testing.T) {
	casesPath, summaryPath := createPhase1ObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkSummaryCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases.csv",
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkSummaryCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"total_runs: 1",
		"slice[aggregate]: groups=1",
		"slice_group[aggregate][model=compatible:gpt-oss:20b llm_model=gpt-oss:20b prompt_version=hilbert-ai-verification-benchmark-v1.3 surface=direct]: runs=1 cases_total=7 pass=3 pass_rate=42.86%",
		"slice[entailed-only]: groups=1",
		"cases_total=5 pass=1 pass_rate=20.00%",
		"slice[not-entailed-only]: groups=1",
		"cases_total=2 pass=2 pass_rate=100.00%",
		"slice[category=assumption_import]: groups=1",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("summary output missing %q:\n%s", snippet, output)
		}
	}
}

func TestRunHilbertBenchmarkSummaryCommandRendersNDObservedSlices(t *testing.T) {
	theoremsPath, summaryPath := createNDObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkSummaryCommand([]string{
		"-project-folder", benchmarkNDProjectFolder,
		"-cases-file", benchmarkNDTheoremPack,
		"-cases", theoremsPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkSummaryCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"total_runs: 1",
		"slice[aggregate]: groups=1",
		"slice_group[aggregate][model=compatible:gpt-oss:20b llm_model=gpt-oss:20b prompt_version=hilbert-ai-verification-benchmark-v1.3 surface=nd]: runs=1 cases_total=5 pass=1 pass_rate=20.00%",
		"slice[entailed-only]: groups=1",
		"cases_total=3 pass=1 pass_rate=33.33%",
		"slice[not-entailed-only]: groups=1",
		"cases_total=2 pass=0 pass_rate=0.00%",
		"slice[theorem_synthesis-focus]: groups=1",
		"slice[audit=assumption_import]: groups=1",
		"nd_pipeline_trace: groups=1",
		"slice[category=negative_refusal]: groups=1",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("summary output missing %q:\n%s", snippet, output)
		}
	}
}

func TestBuildBenchmarkNDPipelineTraceAggregatesInfersLegacyStages(t *testing.T) {
	rows := []benchmarkResultRow{
		{RunID: "run-1", TimestampUTC: "2026-03-30T09:00:01Z", Model: "compatible:gpt-oss:20b", LLMModel: "gpt-oss:20b", PromptVersion: ndBenchmarkPromptVersionV11, Surface: "nd", TheoremID: "NDI01", CaseID: "NDI01", RawOutputKind: "proof_object", SchemaStatus: "fail", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "schema_failure"},
		{RunID: "run-1", TimestampUTC: "2026-03-30T09:00:02Z", Model: "compatible:gpt-oss:20b", LLMModel: "gpt-oss:20b", PromptVersion: ndBenchmarkPromptVersionV11, Surface: "nd", TheoremID: "NDI02", CaseID: "NDI02", RawOutputKind: "proof_object", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "not_run", ScoreBucket: "kernel_failure"},
		{RunID: "run-1", TimestampUTC: "2026-03-30T09:00:03Z", Model: "compatible:gpt-oss:20b", LLMModel: "gpt-oss:20b", PromptVersion: ndBenchmarkPromptVersionV11, Surface: "nd", TheoremID: "NDI03", CaseID: "NDI03", RawOutputKind: "proof_object", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "reject", ScoreBucket: "kernel_failure"},
		{RunID: "run-1", TimestampUTC: "2026-03-30T09:00:04Z", Model: "compatible:gpt-oss:20b", LLMModel: "gpt-oss:20b", PromptVersion: ndBenchmarkPromptVersionV11, Surface: "nd", TheoremID: "NDI04", CaseID: "NDI04", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "pass"},
		{RunID: "run-1", TimestampUTC: "2026-03-30T09:00:05Z", Model: "compatible:gpt-oss:20b", LLMModel: "gpt-oss:20b", PromptVersion: ndBenchmarkPromptVersionV11, Surface: "nd", TheoremID: "NDI05", CaseID: "NDI05", RawOutputKind: "other", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "request_failure"},
	}

	aggregates := buildBenchmarkNDPipelineTraceAggregates(rows)
	if len(aggregates) != 1 {
		t.Fatalf("expected 1 aggregate, got %d", len(aggregates))
	}
	aggregate := aggregates[0]
	if aggregate.NDProofObjectFailures != 1 {
		t.Fatalf("expected nd proof object failures = 1, got %d", aggregate.NDProofObjectFailures)
	}
	if aggregate.LoweringFailures != 1 {
		t.Fatalf("expected lowering failures = 1, got %d", aggregate.LoweringFailures)
	}
	if aggregate.HilbertArtifactFailures != 1 {
		t.Fatalf("expected hilbert artifact failures = 1, got %d", aggregate.HilbertArtifactFailures)
	}
	if aggregate.OutputRefusals != 1 {
		t.Fatalf("expected output refusals = 1, got %d", aggregate.OutputRefusals)
	}
	if aggregate.RequestFailures != 1 {
		t.Fatalf("expected request failures = 1, got %d", aggregate.RequestFailures)
	}
	if aggregate.UnknownStageRows != 0 {
		t.Fatalf("expected unknown stage rows = 0, got %d", aggregate.UnknownStageRows)
	}
}

func TestRunHilbertBenchmarkSummaryCommandSupportsWildcardCasesFile(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-a",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/theorem-synthesis-stress.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/theorem-synthesis-stress.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-a.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-a",
			CasesTotal:          "5",
			PassCount:           "4",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "1500.00",
			MaxLatencyMS:        "1600",
			RunElapsedSeconds:   "21",
			CategoryCount:       "3",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-b",
			TimestampUTC:        "2026-03-26T15:20:00Z",
			Model:               "compatible:gemma3:12b",
			LLMModel:            "gemma3:12b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-b.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-b",
			CasesTotal:          "8",
			PassCount:           "7",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "2500.00",
			MaxLatencyMS:        "2600",
			RunElapsedSeconds:   "31",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkSummaryCommand([]string{
		"-project-folder", "hilbert-ai-verification-benchmark-v2-held-out",
		"-cases-file", "*",
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkSummaryCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "cases_file: *") {
		t.Fatalf("summary output missing wildcard marker:\n%s", output)
	}
	if !strings.Contains(output, "total_runs: 2") {
		t.Fatalf("summary output should include both project runs:\n%s", output)
	}
	if !strings.Contains(output, "llm_model=gpt-oss:20b") || !strings.Contains(output, "llm_model=gemma3:12b") {
		t.Fatalf("summary output should keep separate model groups:\n%s", output)
	}
}

func TestLoadBenchmarkRunSummaryRowsFromPathsReadsTheoremSummaryV2AndLegacyCompat(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkNDProjectFolder)
	legacySummaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")
	summaryPathV2 := filepath.Join(benchmarkRoot, "result", "result_summary_v2.csv")

	if err := os.MkdirAll(filepath.Dir(legacySummaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}

	legacyLines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "legacy-run",
			TimestampUTC:        "2026-03-27T09:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       benchmarkNDProjectFolder,
			CasesFile:           "cases.csv",
			CaseSelector:        "all",
			Hypothesis:          ndBenchmarkHypothesis,
			CasesPath:           filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "cases.csv")),
			ResultsPath:         filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_legacy-run.csv")),
			RawDir:              filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "legacy-run")),
			CasesTotal:          "8",
			PassCount:           "5",
			FalseRefusalCount:   "1",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "1",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "700.00",
			MaxLatencyMS:        "900",
			RunElapsedSeconds:   "12",
			CategoryCount:       "4",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(legacySummaryPath, []byte(strings.Join(legacyLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	v2Lines := []string{
		strings.Join(benchmarkRunSummaryHeaderV2(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "v2-run",
			TimestampUTC:        "2026-03-27T10:00:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       ndBenchmarkPromptVersionV1,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       benchmarkNDProjectFolder,
			TheoremsFile:        benchmarkNDTheoremPack,
			CaseSelector:        "all",
			Hypothesis:          ndBenchmarkHypothesis,
			TheoremsPath:        filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
			ResultsPath:         filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_v2-run.csv")),
			RawDir:              filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "v2-run")),
			CasesTotal:          "8",
			PassCount:           "6",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "1",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "500.00",
			MaxLatencyMS:        "800",
			RunElapsedSeconds:   "9",
			CategoryCount:       "4",
		}, benchmarkRunSummaryHeaderV2()), ","),
	}
	if err := os.WriteFile(summaryPathV2, []byte(strings.Join(v2Lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary_v2.csv) error = %v", err)
	}

	rows, err := loadBenchmarkRunSummaryRowsFromPaths(defaultBenchmarkSummaryPaths(benchmarkRoot))
	if err != nil {
		t.Fatalf("loadBenchmarkRunSummaryRowsFromPaths() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	filtered := filterBenchmarkRunSummaryRows(rows, root, benchmarkNDProjectFolder, benchmarkNDTheoremPack)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	aggregates, err := aggregateBenchmarkRunSummary(filtered)
	if err != nil {
		t.Fatalf("aggregateBenchmarkRunSummary() error = %v", err)
	}
	if len(aggregates) != 2 {
		t.Fatalf("len(aggregates) = %d, want 2", len(aggregates))
	}
	foundLegacyPrompt := false
	foundV2Prompt := false
	for _, row := range filtered {
		if row.PromptVersion == benchmarkPromptVersionV13 {
			foundLegacyPrompt = true
		}
		if row.PromptVersion == ndBenchmarkPromptVersionV1 {
			foundV2Prompt = true
		}
	}
	if !foundLegacyPrompt || !foundV2Prompt {
		t.Fatalf("filtered rows missing expected prompt versions: %+v", filtered)
	}
}

func TestRunHilbertBenchmarkReportCommandRendersMarkdownByCasePack(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-stress-a",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/theorem-synthesis-stress.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/theorem-synthesis-stress.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-stress-a.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-stress-a",
			CasesTotal:          "5",
			PassCount:           "4",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "1500.00",
			MaxLatencyMS:        "1600",
			RunElapsedSeconds:   "21",
			CategoryCount:       "3",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-cross-a",
			TimestampUTC:        "2026-03-26T15:20:00Z",
			Model:               "compatible:gemma3:12b",
			LLMModel:            "gemma3:12b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-cross-a.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-cross-a",
			CasesTotal:          "8",
			PassCount:           "7",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "2500.00",
			MaxLatencyMS:        "2600",
			RunElapsedSeconds:   "31",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", "hilbert-ai-verification-benchmark-v2-held-out",
		"-cases-file", "*",
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "# Hilbert Benchmark Model Comparison Report") {
		t.Fatalf("report output missing markdown title:\n%s", output)
	}
	if !strings.Contains(output, "## Case Pack: `cases/cross-model-sanity.csv`") || !strings.Contains(output, "## Case Pack: `cases/theorem-synthesis-stress.csv`") {
		t.Fatalf("report output missing case pack sections:\n%s", output)
	}
	if !strings.Contains(output, "| `compatible:gemma3:12b` | 1 | 8 | 7 | 87.50% |") {
		t.Fatalf("report output missing model comparison row:\n%s", output)
	}
	if !strings.Contains(output, "Avg Latency (ms)") || !strings.Contains(output, "Avg Run Elapsed (s)") {
		t.Fatalf("report output missing timing columns:\n%s", output)
	}
}

func TestRunHilbertBenchmarkReportCommandRendersPhase1ObservedSectionsAndHardCases(t *testing.T) {
	casesPath, summaryPath := createPhase1ObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases.csv",
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Aggregate",
		"- Prompt version: `hilbert-ai-verification-benchmark-v1.3`",
		"- Certificate version: `1.0.0`",
		"- Surface: `direct`",
		"### Entailed-only",
		"### Not-entailed-only",
		"### Per-category",
		"### Per-case hardest failures",
		"### Hard-case audit targets",
		"#### `V2E07`",
		"#### `V2E08`",
		"#### `V2N09`",
		"| `V2E06` | `direct_axiom_instance` | `entailed` | `medium` |",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"### Chain Summary",
		"chain summary file missing for run",
		"Size Bucket",
		"Size Label",
		"| Model | Surface | Runs | Cases | Pass | Pass Rate |",
		"| Model | Surface | Runs | Cases | ND Proof Object Failure |",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func TestRunHilbertBenchmarkReportCommandUsesSourceCaseIDForExpandedHardCaseAudit(t *testing.T) {
	casesPath, summaryPath := createPhase1ExpandedObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases/phase1-expanded-100.csv",
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Hard-case audit targets",
		"#### `V2E07`",
		"#### `V2E08`",
		"#### `V2N09`",
		"- Observed descendant case ids: `V2XE007`",
		"- Observed descendant case ids: `V2XN009`",
		"- Representative descendant case: `V2XE007`",
		"- Representative descendant case: `V2XN009`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expanded report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"hard-case audit target `V2E01` is missing from cases.csv",
		"hard-case audit target `V2E01` is missing from the selected case pack",
		"hard-case audit target `V2N09` is missing from the selected case pack",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("expanded report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func TestRunHilbertBenchmarkReportCommandRendersNDObservedSectionsAndHardestFailures(t *testing.T) {
	theoremsPath, summaryPath := createNDObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", benchmarkNDProjectFolder,
		"-cases-file", benchmarkNDTheoremPack,
		"-cases", theoremsPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Aggregate",
		"- Surface: `nd`",
		"### Entailed-only",
		"### Not-entailed-only",
		"### Theorem-synthesis focus",
		"### Foundational-category audit",
		"### Audit: `assumption_import`",
		"### Per-category",
		"### ND pipeline trace",
		"### Per-case hardest failures",
		"- Prompt version: `hilbert-ai-verification-benchmark-v1.3`",
		"| `NDI05` | `theorem_synthesis` | `entailed` | `medium` |",
		"| `compatible:gpt-oss:20b` | 1 | 5 | 1 | 1 | 1 | 1 | 0 | 0 | 0 |",
		"### Hard-case audit targets",
		"No hard-case audit targets were available.",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"### Chain Summary",
		"chain summary file missing for run",
		"| Model | Surface | Runs | Cases | Pass | Pass Rate |",
		"| Model | Surface | Runs | Cases | ND Proof Object Failure |",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func TestRunHilbertBenchmarkMetaReportCommandOmitsSurfaceSectionForSingleSurface(t *testing.T) {
	_, summaryPath := createPhase1ObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkMetaReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases.csv",
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkMetaReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"# Hilbert Benchmark Meta Report",
		"## Pack overview",
		"### Case Pack: `cases.csv`",
		"- Surface: `direct`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("meta report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"## Surface summary",
		"| Pack | Summary Rows | Surfaces | Best Slice |",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("meta report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func assertMarkdownTableColumnConsistency(t *testing.T, markdown string) {
	t.Helper()
	lines := strings.Split(markdown, "\n")
	for i := 0; i < len(lines)-1; i++ {
		header := strings.TrimSpace(lines[i])
		separator := strings.TrimSpace(lines[i+1])
		if !strings.HasPrefix(header, "| ") || !strings.HasSuffix(header, " |") {
			continue
		}
		if !strings.HasPrefix(separator, "| ---") {
			continue
		}
		headerColumns := strings.Count(header, "|") - 1
		separatorColumns := strings.Count(separator, "|") - 1
		if headerColumns != separatorColumns {
			t.Fatalf("markdown table header/separator column mismatch at line %d: header=%d separator=%d\nheader=%s\nseparator=%s", i+1, headerColumns, separatorColumns, header, separator)
		}
	}
}

func TestRunHilbertBenchmarkReportCommandWritesToFile(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v1")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")
	reportPath := filepath.Join(root, "report.md")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-a",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v1",
			CasesFile:           "cases.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_run-a.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v1/raw/run-a",
			CasesTotal:          "16",
			PassCount:           "15",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "0",
			SchemaFailureCount:  "1",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "8000.00",
			MaxLatencyMS:        "22000",
			RunElapsedSeconds:   "110",
			CategoryCount:       "0",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", "hilbert-ai-verification-benchmark-v1",
		"-cases-file", "cases.csv",
		"-summary", summaryPath,
		"-out", reportPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "hilbert benchmark markdown report written") {
		t.Fatalf("expected file-write confirmation, got:\n%s", stdout.String())
	}
	reportRaw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("ReadFile(report.md) error = %v", err)
	}
	if !strings.Contains(string(reportRaw), "## Case Pack: `cases.csv`") {
		t.Fatalf("report file missing case pack section:\n%s", string(reportRaw))
	}
}

func TestRunHilbertBenchmarkResearchReportCommandCombinesMultipleSummaries(t *testing.T) {
	root := t.TempDir()
	directSummaryPath := filepath.Join(root, "result_summary_v2_direct.csv")
	ndSummaryPath := filepath.Join(root, "result_summary_v2_nd.csv")

	directLines := []string{
		strings.Join(benchmarkRunSummaryHeaderV2(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-direct",
			TimestampUTC:        "2026-03-27T08:21:59Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       benchmarkNDProjectFolder,
			CasesFile:           benchmarkNDTheoremPack,
			CaseSelector:        "all",
			Hypothesis:          benchmarkHypothesis,
			CasesPath:           filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
			ResultsPath:         filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_direct.csv")),
			RawDir:              filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "run-direct")),
			CasesTotal:          "8",
			PassCount:           "2",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "5",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "25990.00",
			MaxLatencyMS:        "60005",
			RunElapsedSeconds:   "208",
			CategoryCount:       "6",
		}, benchmarkRunSummaryHeaderV2()), ","),
	}
	if err := os.WriteFile(directSummaryPath, []byte(strings.Join(directLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(direct summary) error = %v", err)
	}

	ndLines := []string{
		strings.Join(benchmarkRunSummaryHeaderV2(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-nd",
			TimestampUTC:        "2026-03-27T08:28:22Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       ndBenchmarkPromptVersionV11,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       benchmarkNDProjectFolder,
			CasesFile:           benchmarkNDTheoremPack,
			CaseSelector:        "all",
			Hypothesis:          ndBenchmarkHypothesis,
			CasesPath:           filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
			ResultsPath:         filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_nd.csv")),
			RawDir:              filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "run-nd")),
			CasesTotal:          "8",
			PassCount:           "2",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "5",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "21195.38",
			MaxLatencyMS:        "60005",
			RunElapsedSeconds:   "169",
			CategoryCount:       "6",
		}, benchmarkRunSummaryHeaderV2()), ","),
	}
	if err := os.WriteFile(ndSummaryPath, []byte(strings.Join(ndLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(nd summary) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkResearchReportCommand([]string{
		"-project-folder", benchmarkNDProjectFolder,
		"-cases-file", benchmarkNDTheoremPack,
		"-summary", directSummaryPath + "," + ndSummaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkResearchReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"# Hilbert Benchmark Research Report",
		"## Research Summary: `theorems/pilot_shared_20260327.csv`",
		"Summary sources: `2`",
		"`ND -> Hilbert` matched direct Hilbert on verified pass rate at `2/8` (`25.00%`)",
		"`false_accept = 0` in both runs",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("research report output missing %q:\n%s", snippet, output)
		}
	}
}

func TestRunHilbertBenchmarkMetaReportCommandCombinesMultipleSummaries(t *testing.T) {
	root := t.TempDir()
	directSummaryPath := filepath.Join(root, "result_summary_v2_direct.csv")
	ndSummaryPath := filepath.Join(root, "result_summary_v2_nd.csv")

	directLines := []string{
		strings.Join(benchmarkRunSummaryHeaderV2(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:                "run-direct",
			TimestampUTC:         "2026-03-27T08:21:59Z",
			Model:                "compatible:gpt-oss:20b",
			LLMModel:             "gpt-oss:20b",
			PromptVersion:        benchmarkPromptVersionV13,
			CertificateVersion:   benchmarkCertificateVersion,
			ProjectFolder:        benchmarkNDProjectFolder,
			TheoremPackID:        "pilot_shared_20260327",
			CasesFile:            benchmarkNDTheoremPack,
			CaseSelector:         "all",
			Hypothesis:           benchmarkHypothesis,
			CasesPath:            filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
			ResultsPath:          filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_direct.csv")),
			RawDir:               filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "run-direct")),
			CasesTotal:           "8",
			PassCount:            "2",
			FalseRefusalCount:    "0",
			FalseAcceptCount:     "0",
			RequestFailureCount:  "1",
			SchemaFailureCount:   "5",
			ParseFailureCount:    "0",
			KernelFailureCount:   "0",
			ContractFailureCount: "0",
			FormatFailureCount:   "0",
			AvgLatencyMS:         "25990.00",
			MaxLatencyMS:         "60005",
			RunElapsedSeconds:    "208",
			CategoryCount:        "6",
		}, benchmarkRunSummaryHeaderV2()), ","),
	}
	if err := os.WriteFile(directSummaryPath, []byte(strings.Join(directLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(direct summary) error = %v", err)
	}

	ndLines := []string{
		strings.Join(benchmarkRunSummaryHeaderV2(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:                "run-nd",
			TimestampUTC:         "2026-03-27T08:28:22Z",
			Model:                "compatible:gpt-oss:20b",
			LLMModel:             "gpt-oss:20b",
			PromptVersion:        ndBenchmarkPromptVersionV11,
			CertificateVersion:   benchmarkCertificateVersion,
			ProjectFolder:        benchmarkNDProjectFolder,
			TheoremPackID:        "pilot_shared_20260327",
			CasesFile:            benchmarkNDTheoremPack,
			CaseSelector:         "all",
			Hypothesis:           ndBenchmarkHypothesis,
			CasesPath:            filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, benchmarkNDTheoremPack)),
			ResultsPath:          filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "result", "result_nd.csv")),
			RawDir:               filepath.ToSlash(filepath.Join("research", "artifacts", benchmarkNDProjectFolder, "raw", "run-nd")),
			CasesTotal:           "8",
			PassCount:            "2",
			FalseRefusalCount:    "0",
			FalseAcceptCount:     "0",
			RequestFailureCount:  "1",
			SchemaFailureCount:   "5",
			ParseFailureCount:    "0",
			KernelFailureCount:   "0",
			ContractFailureCount: "0",
			FormatFailureCount:   "0",
			AvgLatencyMS:         "21195.38",
			MaxLatencyMS:         "60005",
			RunElapsedSeconds:    "169",
			CategoryCount:        "6",
		}, benchmarkRunSummaryHeaderV2()), ","),
	}
	if err := os.WriteFile(ndSummaryPath, []byte(strings.Join(ndLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(nd summary) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkMetaReportCommand([]string{
		"-project-folder", benchmarkNDProjectFolder,
		"-cases-file", benchmarkNDTheoremPack,
		"-summary", directSummaryPath + "," + ndSummaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkMetaReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"# Hilbert Benchmark Meta Report",
		"This report stays on the summary layer only.",
		"## Overview",
		"## Top recorded slices",
		"- Prompt versions: `hilbert-ai-verification-benchmark-nd-v1.1`, `hilbert-ai-verification-benchmark-v1.3`",
		"- Certificate version: `1.0.0`",
		"## Pack overview",
		"## Direct vs ND pair snapshot",
		"### Theorem Pack: `theorems/pilot_shared_20260327.csv`",
		"`ND -> Hilbert` matched direct Hilbert on verified pass rate at `2/8` (`25.00%`)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("meta report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"### Entailed-only",
		"### Hard-case audit targets",
		"### Chain Summary",
		"Size Bucket",
		"Size Label",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("meta report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func TestRunHilbertBenchmarkCaseReportCommandRendersPerCaseTables(t *testing.T) {
	casesPath, summaryPath := createPhase1ObservedFixture(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkCaseReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases.csv",
		"-cases", casesPath,
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkCaseReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"# Hilbert Benchmark Case Report",
		"## Overview",
		"## Hardest observed cases",
		"## Per-pack case tables",
		"### Pack: `cases.csv`",
		"Latest Run (UTC)",
		"`V2E06`",
		"`schema_failure (1)`",
		"`V2N07`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("case report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"### Chain Summary",
		"### Entailed-only",
		"### Hard-case audit targets",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("case report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
	assertMarkdownTableColumnConsistency(t, output)
}

func TestRunHilbertBenchmarkResearchReportCommandIncludesObservedSectionsForNDPack(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkNDProjectFolder)
	theoremsPath := filepath.Join(benchmarkRoot, "theorems", "pilot_shared_20260327.csv")
	directSummaryPath := filepath.Join(root, "result_summary_v2_direct.csv")
	ndSummaryPath := filepath.Join(root, "result_summary_v2_nd.csv")
	directResultPath := filepath.Join(benchmarkRoot, "result", "result_direct.csv")
	ndResultPath := filepath.Join(benchmarkRoot, "result", "result_nd.csv")
	directRawDir := filepath.Join(benchmarkRoot, "raw", "run-direct")
	ndRawDir := filepath.Join(benchmarkRoot, "raw", "run-nd")

	theoremsCSV := strings.Join([]string{
		"theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment",
		`pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI02,NDI02,single_mp,entailed,easy,"[""P"",""P -> Q""]",Q,implicational-prop-v1,Single modus ponens`,
		`pilot_shared_20260327,NDI05,NDI05,theorem_synthesis,entailed,medium,[],"P -> P",implicational-prop-v1,Identity theorem without premises`,
		`pilot_shared_20260327,NDI07,NDI07,negative_refusal,not_entailed,easy,"[""P -> Q""]",P,implicational-prop-v1,Consequent does not imply antecedent`,
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(theoremsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(theorems dir) error = %v", err)
	}
	if err := os.WriteFile(theoremsPath, []byte(theoremsCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(theorems.csv) error = %v", err)
	}
	if err := os.MkdirAll(directRawDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(direct raw dir) error = %v", err)
	}
	if err := os.MkdirAll(ndRawDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(nd raw dir) error = %v", err)
	}

	directSummaryRow := benchmarkRunSummaryRow{
		RunID:               "20260330_phase2_direct_gpt-oss-20b_r01",
		TimestampUTC:        "2026-03-30T09:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkNDProjectFolder,
		Surface:             "direct",
		TheoremPackID:       "pilot_shared_20260327",
		TheoremsFile:        benchmarkNDTheoremPack,
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		TheoremsPath:        theoremsPath,
		ResultsPath:         directResultPath,
		RawDir:              directRawDir,
		CasesTotal:          "4",
		PassCount:           "3",
		FalseRefusalCount:   "0",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "1",
		ParseFailureCount:   "0",
		KernelFailureCount:  "0",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "12.50",
		MaxLatencyMS:        "18",
		RunElapsedSeconds:   "48",
		CategoryCount:       "4",
	}
	ndSummaryRow := benchmarkRunSummaryRow{
		RunID:               "20260330_phase2_nd_gpt-oss-20b_r01",
		TimestampUTC:        "2026-03-30T09:01:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       ndBenchmarkPromptVersionV11,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkNDProjectFolder,
		Surface:             "nd",
		TheoremPackID:       "pilot_shared_20260327",
		TheoremsFile:        benchmarkNDTheoremPack,
		CaseSelector:        "all",
		Hypothesis:          ndBenchmarkHypothesis,
		TheoremsPath:        theoremsPath,
		ResultsPath:         ndResultPath,
		RawDir:              ndRawDir,
		CasesTotal:          "4",
		PassCount:           "2",
		FalseRefusalCount:   "1",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "0",
		ParseFailureCount:   "0",
		KernelFailureCount:  "1",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "15.50",
		MaxLatencyMS:        "22",
		RunElapsedSeconds:   "55",
		CategoryCount:       "4",
	}
	writeBenchmarkSummaryCSV(t, directSummaryPath, directSummaryRow)
	writeBenchmarkSummaryCSV(t, ndSummaryPath, ndSummaryRow)

	writeBenchmarkResultCSV(t, directResultPath,
		benchmarkResultRow{RunID: directSummaryRow.RunID, TimestampUTC: "2026-03-30T09:00:01Z", Model: directSummaryRow.Model, LLMModel: directSummaryRow.LLMModel, PromptVersion: directSummaryRow.PromptVersion, Surface: "direct", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI01", CaseID: "NDI01", Category: "assumption_import", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "10", Notes: "verified"},
		benchmarkResultRow{RunID: directSummaryRow.RunID, TimestampUTC: "2026-03-30T09:00:02Z", Model: directSummaryRow.Model, LLMModel: directSummaryRow.LLMModel, PromptVersion: directSummaryRow.PromptVersion, Surface: "direct", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI02", CaseID: "NDI02", Category: "single_mp", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "11", Notes: "verified"},
		benchmarkResultRow{RunID: directSummaryRow.RunID, TimestampUTC: "2026-03-30T09:00:03Z", Model: directSummaryRow.Model, LLMModel: directSummaryRow.LLMModel, PromptVersion: directSummaryRow.PromptVersion, Surface: "direct", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI05", CaseID: "NDI05", Category: "theorem_synthesis", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "fail", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "schema_failure", LatencyMS: "18", Notes: "schema mismatch"},
		benchmarkResultRow{RunID: directSummaryRow.RunID, TimestampUTC: "2026-03-30T09:00:04Z", Model: directSummaryRow.Model, LLMModel: directSummaryRow.LLMModel, PromptVersion: directSummaryRow.PromptVersion, Surface: "direct", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI07", CaseID: "NDI07", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "11", Notes: "correct refusal"},
	)
	writeBenchmarkResultCSV(t, ndResultPath,
		benchmarkResultRow{RunID: ndSummaryRow.RunID, TimestampUTC: "2026-03-30T09:01:01Z", Model: ndSummaryRow.Model, LLMModel: ndSummaryRow.LLMModel, PromptVersion: ndSummaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "pass", PipelineStage: "hilbert_artifact", PipelineDetail: "pass", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI01", CaseID: "NDI01", Category: "assumption_import", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "13", Notes: "verified"},
		benchmarkResultRow{RunID: ndSummaryRow.RunID, TimestampUTC: "2026-03-30T09:01:02Z", Model: ndSummaryRow.Model, LLMModel: ndSummaryRow.LLMModel, PromptVersion: ndSummaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "pass", PipelineStage: "hilbert_artifact", PipelineDetail: "pass", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI02", CaseID: "NDI02", Category: "single_mp", ExpectedLabel: "entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "accept", ScoreBucket: "pass", LatencyMS: "14", Notes: "verified"},
		benchmarkResultRow{RunID: ndSummaryRow.RunID, TimestampUTC: "2026-03-30T09:01:03Z", Model: ndSummaryRow.Model, LLMModel: ndSummaryRow.LLMModel, PromptVersion: ndSummaryRow.PromptVersion, Surface: "nd", NDProofStatus: "not_run", LoweringStatus: "not_run", PipelineStage: "llm_output", PipelineDetail: "not_derivable", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI05", CaseID: "NDI05", Category: "theorem_synthesis", ExpectedLabel: "entailed", RawOutputKind: "not_derivable", SchemaStatus: "not_run", ParseStatus: "not_run", KernelStatus: "not_run", ScoreBucket: "false_refusal", LatencyMS: "22", Notes: "refused derivable theorem"},
		benchmarkResultRow{RunID: ndSummaryRow.RunID, TimestampUTC: "2026-03-30T09:01:04Z", Model: ndSummaryRow.Model, LLMModel: ndSummaryRow.LLMModel, PromptVersion: ndSummaryRow.PromptVersion, Surface: "nd", NDProofStatus: "pass", LoweringStatus: "fail", PipelineStage: "lowering", PipelineDetail: "validation_error", TheoremPackID: "pilot_shared_20260327", TheoremID: "NDI07", CaseID: "NDI07", Category: "negative_refusal", ExpectedLabel: "not_entailed", RawOutputKind: "certificate", SchemaStatus: "pass", ParseStatus: "pass", KernelStatus: "not_run", ScoreBucket: "kernel_failure", LatencyMS: "13", Notes: "lowering failed verification"},
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkResearchReportCommand([]string{
		"-project-folder", benchmarkNDProjectFolder,
		"-cases-file", benchmarkNDTheoremPack,
		"-summary", directSummaryPath + "," + ndSummaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkResearchReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Aggregate",
		"### Theorem-synthesis focus",
		"### Foundational-category audit",
		"### Audit: `assumption_import`",
		"### Audit: `single_mp`",
		"### ND pipeline trace",
		"### Per-case hardest failures",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("research report output missing %q:\n%s", snippet, output)
		}
	}
}

func TestRunHilbertBenchmarkReportCommandRendersCompositionalChainSummarySection(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkV2HeldOutProjectFolder)
	summaryPath := filepath.Join(root, "compositional-summary.csv")
	resultPath := filepath.Join(benchmarkRoot, "result", "result_compositional.csv")
	chainSummaryPath := filepath.Join(benchmarkRoot, "result", "chain_result_compositional.csv")
	casesPath := filepath.Join(benchmarkRoot, "cases", "compositional-assumption-import-phase1.csv")

	if err := os.MkdirAll(filepath.Dir(resultPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(result dir) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(casesPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(cases dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte("case_id,category,label,difficulty,assumptions_json,goal,comment\nCI01-S1,compositional_assumption_import,entailed,easy,[],A -> B,seed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cases) error = %v", err)
	}

	writeBenchmarkSummaryCSV(t, summaryPath, benchmarkRunSummaryRow{
		RunID:               "compositional-20260403",
		TimestampUTC:        "2026-04-03T09:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkV2HeldOutProjectFolder,
		Surface:             "direct",
		CasesFile:           "cases/compositional-assumption-import-phase1.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           casesPath,
		ResultsPath:         resultPath,
		ChainSummaryPath:    chainSummaryPath,
		RawDir:              filepath.Join(benchmarkRoot, "raw", "compositional-20260403"),
		CasesTotal:          "5",
		PassCount:           "4",
		FalseRefusalCount:   "0",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "1",
		ParseFailureCount:   "0",
		KernelFailureCount:  "0",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "12.50",
		MaxLatencyMS:        "18",
		RunElapsedSeconds:   "48",
		CategoryCount:       "1",
	})
	writeBenchmarkResultCSV(t, resultPath,
		benchmarkResultRow{
			RunID:         "compositional-20260403",
			TimestampUTC:  "2026-04-03T09:00:01Z",
			Model:         "compatible:gpt-oss:20b",
			LLMModel:      "gpt-oss:20b",
			PromptVersion: benchmarkPromptVersionV13,
			Surface:       "direct",
			TheoremID:     "CI01-S1",
			CaseID:        "CI01-S1",
			Category:      "compositional_assumption_import",
			ExpectedLabel: "entailed",
			RawOutputKind: "certificate",
			SchemaStatus:  "pass",
			ParseStatus:   "pass",
			KernelStatus:  "accept",
			ScoreBucket:   "pass",
			LatencyMS:     "12",
			Notes:         "verified",
		},
	)
	chainLines := []string{
		"run_id,chain_id,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes",
		"compositional-20260403,CI01,linear_chain,1,r0,true,true,true,false,true,false,true,true,false,true,false,s3m,schema_failure,model import degraded",
		"compositional-20260403,CI02,linear_chain,1,r1,true,true,true,true,true,true,true,true,true,true,true,,,all good",
	}
	if err := os.WriteFile(chainSummaryPath, []byte(strings.Join(chainLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(chain summary) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases/compositional-assumption-import-phase1.csv",
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Chain Summary",
		"- Surface: `direct`",
		"#### Chain failure breakdown",
		"`GPT OSS 20B`",
		"1/2 (50.00%)",
		"`semantic_transport_failure`, `import_binding_failure`, `certificate_assembly_failure`, and `final_proof_closure_failure`",
		"`s3m`",
		"`schema_failure`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("compositional report output missing %q:\n%s", snippet, output)
		}
	}
	for _, forbidden := range []string{
		"| Model | Surface | Runs | Chains |",
		"| Model | Surface | Failure Stage | Stage Role | Failure Class | Failure Type | Chains | Latest Run (UTC) |",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("compositional report output unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
}

func TestRunHilbertBenchmarkReportCommandRendersGenericChainStageProtocolSection(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", benchmarkV2HeldOutProjectFolder)
	summaryPath := filepath.Join(root, "depth-ladder-summary.csv")
	resultPath := filepath.Join(benchmarkRoot, "result", "result_depth-ladder.csv")
	chainSummaryPath := filepath.Join(benchmarkRoot, "result", "chain_result_depth-ladder.csv")
	casesPath := filepath.Join(benchmarkRoot, "cases", "compositional-depth-ladder-phase1.csv")

	if err := os.MkdirAll(filepath.Dir(resultPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(result dir) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(casesPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(cases dir) error = %v", err)
	}
	if err := os.WriteFile(casesPath, []byte("case_id,category,label,difficulty,assumptions_json,goal,comment\nCDL02-S1,compositional_depth_ladder,entailed,hard,[],P -> Q,seed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(cases) error = %v", err)
	}

	writeBenchmarkSummaryCSV(t, summaryPath, benchmarkRunSummaryRow{
		RunID:               "depth-ladder-20260403",
		TimestampUTC:        "2026-04-03T10:00:00Z",
		Model:               "compatible:gpt-oss:20b",
		LLMModel:            "gpt-oss:20b",
		PromptVersion:       benchmarkPromptVersionV13,
		CertificateVersion:  benchmarkCertificateVersion,
		ProjectFolder:       benchmarkV2HeldOutProjectFolder,
		Surface:             "direct",
		CasesFile:           "cases/compositional-depth-ladder-phase1.csv",
		CaseSelector:        "all",
		Hypothesis:          benchmarkHypothesis,
		CasesPath:           casesPath,
		ResultsPath:         resultPath,
		ChainSummaryPath:    chainSummaryPath,
		RawDir:              filepath.Join(benchmarkRoot, "raw", "depth-ladder-20260403"),
		CasesTotal:          "8",
		PassCount:           "8",
		FalseRefusalCount:   "0",
		FalseAcceptCount:    "0",
		RequestFailureCount: "0",
		SchemaFailureCount:  "0",
		ParseFailureCount:   "0",
		KernelFailureCount:  "0",
		FormatFailureCount:  "0",
		AvgLatencyMS:        "11.00",
		MaxLatencyMS:        "18",
		RunElapsedSeconds:   "60",
		CategoryCount:       "1",
	})
	writeBenchmarkResultCSV(t, resultPath,
		benchmarkResultRow{
			RunID:         "depth-ladder-20260403",
			TimestampUTC:  "2026-04-03T10:00:01Z",
			Model:         "compatible:gpt-oss:20b",
			LLMModel:      "gpt-oss:20b",
			PromptVersion: benchmarkPromptVersionV13,
			Surface:       "direct",
			TheoremID:     "CDL02-S1",
			CaseID:        "CDL02-S1",
			Category:      "compositional_depth_ladder",
			ExpectedLabel: "entailed",
			RawOutputKind: "certificate",
			SchemaStatus:  "pass",
			ParseStatus:   "pass",
			KernelStatus:  "accept",
			ScoreBucket:   "pass",
			LatencyMS:     "12",
			Notes:         "verified",
		},
	)
	chainLines := []string{
		"run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,stage_sequence_json,stage_roles_json,stage_passes_json,stage_failures_json,trusted_reuse_stages_json,gold_import_stages_json,model_import_stages_json,failure_stage,failure_type,notes",
		`depth-ladder-20260403,CDL02,depth-ladder-v1,depth3_linear,3,r0,true,true,true,true,true,true,true,true,true,true,true,"[""s1"",""s2"",""s3"",""d2m"",""d3m"",""neg""]","{""s1"":""seed_lemma"",""s2"":""bridge_lemma"",""s3"":""bridge_lemma"",""d2m"":""intermediate_model"",""d3m"":""final_model"",""neg"":""negative_control""}","{""s1"":true,""s2"":true,""s3"":true,""d2m"":true,""d3m"":true,""neg"":true}","{""s1"":""pass"",""s2"":""pass"",""s3"":""pass"",""d2m"":""pass"",""d3m"":""pass"",""neg"":""pass""}","[""s1"",""s2"",""s3"",""d2m""]","[]","[""d2m"",""d3m""]",,,all good`,
	}
	if err := os.WriteFile(chainSummaryPath, []byte(strings.Join(chainLines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(chain summary) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", benchmarkV2HeldOutProjectFolder,
		"-cases-file", "cases/compositional-depth-ladder-phase1.csv",
		"-summary", summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Chain Summary",
		"- Surface: `direct`",
		"#### Stage protocol breakdown",
		"`d3m`",
		"`final_model`",
		"1/1 (100.00%)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("depth ladder report output missing %q:\n%s", snippet, output)
		}
	}
}

func TestBenchmarkClassifyChainStageFailureUsesBridgeImportFinalTaxonomy(t *testing.T) {
	tests := []struct {
		name            string
		tc              benchmarkCase
		row             benchmarkResultRow
		stagePresent    bool
		wantClass       string
		wantDetailMatch string
	}{
		{
			name: "starter failure is semantic transport",
			tc: benchmarkCase{
				StageID:        "b2",
				StageRole:      "bridge_right",
				ProvenanceMode: "none",
			},
			row:             benchmarkResultRow{ScoreBucket: "schema_failure"},
			stagePresent:    true,
			wantClass:       "semantic_transport_failure",
			wantDetailMatch: "prerequisite transport stage",
		},
		{
			name: "missing model import binding is classified separately",
			tc: benchmarkCase{
				StageID:        "fm",
				StageRole:      "final_model",
				ProvenanceMode: "model",
				ImportStageIDs: []string{"b1", "b2"},
			},
			row: benchmarkResultRow{
				ScoreBucket:            "schema_failure",
				ResolvedImportRefsJSON: `[{"stage_id":"b1","goal":"A -> B","trusted_for_reuse":true}]`,
				EffectiveImportsJSON:   `["A -> B"]`,
				RequestedImportsJSON:   `["A -> B","B -> C"]`,
			},
			stagePresent:    true,
			wantClass:       "import_binding_failure",
			wantDetailMatch: "missing requested prerequisite stage(s)",
		},
		{
			name: "request failure at final gold stage is certificate assembly",
			tc: benchmarkCase{
				StageID:        "fg",
				StageRole:      "final_gold",
				ProvenanceMode: "gold",
			},
			row:             benchmarkResultRow{ScoreBucket: "request_failure"},
			stagePresent:    true,
			wantClass:       "certificate_assembly_failure",
			wantDetailMatch: "failed before a verifier-acceptable certificate could be assembled",
		},
		{
			name: "kernel failure at final gold stage is proof closure",
			tc: benchmarkCase{
				StageID:        "fg",
				StageRole:      "final_gold",
				ProvenanceMode: "gold",
			},
			row:             benchmarkResultRow{ScoreBucket: "kernel_failure"},
			stagePresent:    true,
			wantClass:       "final_proof_closure_failure",
			wantDetailMatch: "proof-closure failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotClass, gotDetail := benchmarkClassifyChainStageFailure(tc.tc, tc.row, tc.stagePresent)
			if gotClass != tc.wantClass {
				t.Fatalf("benchmarkClassifyChainStageFailure() class = %q, want %q", gotClass, tc.wantClass)
			}
			if !strings.Contains(gotDetail, tc.wantDetailMatch) {
				t.Fatalf("benchmarkClassifyChainStageFailure() detail = %q, want substring %q", gotDetail, tc.wantDetailMatch)
			}
		})
	}
}

func TestLoadBenchmarkModelCatalogAssignsBuckets(t *testing.T) {
	root := t.TempDir()
	catalogPath := filepath.Join(root, "model-catalog.csv")
	catalogCSV := strings.Join([]string{
		"llm_model,display_name,size_bucket,size_label,tier,comparison_status,notes",
		"gpt-oss:20b,GPT OSS 20B,<=20b,20b,free-current,primary,primary small-model slice",
		"glm-5:cloud,GLM-5 Cloud,unknown,unknown,free-current,appendix,unknown size",
		"qwen3:12b,Qwen3 12B,<=20b,12b,free-current,insufficient-evidence,missing clean baseline run",
	}, "\n")
	if err := os.WriteFile(catalogPath, []byte(catalogCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(model-catalog.csv) error = %v", err)
	}

	rows, err := loadBenchmarkModelCatalog(catalogPath)
	if err != nil {
		t.Fatalf("loadBenchmarkModelCatalog() error = %v", err)
	}
	index := indexBenchmarkModelCatalog(rows)

	if got := index["gpt-oss:20b"].SizeBucket; got != "<=20b" {
		t.Fatalf("gpt-oss:20b size_bucket = %q, want <=20b", got)
	}
	if got := index["glm-5:cloud"].SizeBucket; got != "unknown" {
		t.Fatalf("glm-5:cloud size_bucket = %q, want unknown", got)
	}
	if got := index["qwen3:12b"].ComparisonStatus; got != "insufficient-evidence" {
		t.Fatalf("qwen3:12b comparison_status = %q, want insufficient-evidence", got)
	}
}

func TestBuildBenchmarkComparativeSlicesExcludesPureRequestFailureFromCleanBaseline(t *testing.T) {
	catalog := indexBenchmarkModelCatalog([]benchmarkModelCatalogRow{
		{
			LLMModel:         "gpt-oss:20b",
			DisplayName:      "GPT OSS 20B",
			SizeBucket:       "<=20b",
			SizeLabel:        "20b",
			Tier:             "free-current",
			ComparisonStatus: "primary",
		},
	})

	slicesReport, err := buildBenchmarkComparativeSlices([]benchmarkRunSummaryRow{
		{
			RunID:               "valid-run",
			TimestampUTC:        "2026-03-26T17:19:30Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			CasesTotal:          "24",
			PassCount:           "16",
			FalseRefusalCount:   "1",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "6",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "8311.08",
			MaxLatencyMS:        "60005",
			RunElapsedSeconds:   "199",
		},
		{
			RunID:               "invalid-run",
			TimestampUTC:        "2026-03-26T17:25:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			CasesTotal:          "24",
			PassCount:           "0",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "24",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "0.50",
			MaxLatencyMS:        "2",
			RunElapsedSeconds:   "0",
		},
	}, catalog)
	if err != nil {
		t.Fatalf("buildBenchmarkComparativeSlices() error = %v", err)
	}

	if len(slicesReport.CleanBaseline) != 1 {
		t.Fatalf("len(CleanBaseline) = %d, want 1", len(slicesReport.CleanBaseline))
	}
	if len(slicesReport.LE20B) != 1 {
		t.Fatalf("len(LE20B) = %d, want 1", len(slicesReport.LE20B))
	}
	if len(slicesReport.Appendix) != 1 {
		t.Fatalf("len(Appendix) = %d, want 1", len(slicesReport.Appendix))
	}
	if slicesReport.Appendix[0].Status != "invalid-run" {
		t.Fatalf("Appendix[0].Status = %q, want invalid-run", slicesReport.Appendix[0].Status)
	}
	if !strings.Contains(slicesReport.Appendix[0].Reason, "request layer") {
		t.Fatalf("Appendix[0].Reason = %q, want request-layer invalidation", slicesReport.Appendix[0].Reason)
	}
}

func TestRunHilbertBenchmarkReportCommandRendersCohortSectionsAndAppendix(t *testing.T) {
	root := t.TempDir()
	benchmarkRoot := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out")
	summaryPath := filepath.Join(benchmarkRoot, "result", "result_summary.csv")
	modelCatalogPath := filepath.Join(benchmarkRoot, "model-catalog.csv")
	invalidResultPath := filepath.Join(benchmarkRoot, "result", "result_run-invalid.csv")

	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(summary dir) error = %v", err)
	}
	catalogCSV := strings.Join([]string{
		"llm_model,display_name,size_bucket,size_label,tier,comparison_status,notes",
		"deepseek-r1:14b,DeepSeek R1 14B,<=20b,14b,free-current,primary,primary small-model slice",
		"gpt-oss:20b,GPT OSS 20B,<=20b,20b,free-current,primary,primary small-model slice",
		"qwen3:12b,Qwen3 12B,<=20b,12b,free-current,insufficient-evidence,wait for valid v1.3 run",
		"gpt-oss:120b-cloud,GPT OSS 120B Cloud,>20b,120b,free-current,primary,primary large-model slice",
		"glm-5:cloud,GLM-5 Cloud,unknown,unknown,free-current,appendix,keep out of size-based conclusion",
	}, "\n")
	if err := os.WriteFile(modelCatalogPath, []byte(catalogCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(model-catalog.csv) error = %v", err)
	}
	invalidResultsCSV := strings.Join([]string{
		"run_id,timestamp_utc,model,llm_model,prompt_version,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes",
		"run-invalid,2026-03-26T17:25:00Z,compatible:deepseek-r1:14b,deepseek-r1:14b,hilbert-ai-verification-benchmark-v1.3,V2E01,assumption_import,entailed,other,not_run,not_run,not_run,,request_failure,2,model 'deepseek-r1:14b' not found",
	}, "\n")
	if err := os.WriteFile(invalidResultPath, []byte(invalidResultsCSV+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_run-invalid.csv) error = %v", err)
	}

	lines := []string{
		strings.Join(benchmarkRunSummaryHeader(), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-v12-gpt20",
			TimestampUTC:        "2026-03-26T15:15:00Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-v12-gpt20.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-v12-gpt20",
			CasesTotal:          "8",
			PassCount:           "7",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "0",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "1",
			AvgLatencyMS:        "9000.00",
			MaxLatencyMS:        "38000",
			RunElapsedSeconds:   "70",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-v13-gpt20",
			TimestampUTC:        "2026-03-26T17:19:30Z",
			Model:               "compatible:gpt-oss:20b",
			LLMModel:            "gpt-oss:20b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-v13-gpt20.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-v13-gpt20",
			CasesTotal:          "8",
			PassCount:           "6",
			FalseRefusalCount:   "1",
			FalseAcceptCount:    "0",
			RequestFailureCount: "1",
			SchemaFailureCount:  "1",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "12000.00",
			MaxLatencyMS:        "42000",
			RunElapsedSeconds:   "90",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-v13-gpt120",
			TimestampUTC:        "2026-03-26T17:26:02Z",
			Model:               "compatible:gpt-oss:120b-cloud",
			LLMModel:            "gpt-oss:120b-cloud",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-v13-gpt120.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-v13-gpt120",
			CasesTotal:          "8",
			PassCount:           "8",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "0",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "6000.00",
			MaxLatencyMS:        "19000",
			RunElapsedSeconds:   "49",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-v12-glm",
			TimestampUTC:        "2026-03-26T16:56:42Z",
			Model:               "compatible:glm-5:cloud",
			LLMModel:            "glm-5:cloud",
			PromptVersion:       benchmarkPromptVersionV12,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_run-v12-glm.csv",
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-v12-glm",
			CasesTotal:          "8",
			PassCount:           "8",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "0",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "11632.62",
			MaxLatencyMS:        "20380",
			RunElapsedSeconds:   "93",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
		strings.Join(serializeBenchmarkRunSummaryRow(benchmarkRunSummaryRow{
			RunID:               "run-invalid",
			TimestampUTC:        "2026-03-26T17:25:00Z",
			Model:               "compatible:deepseek-r1:14b",
			LLMModel:            "deepseek-r1:14b",
			PromptVersion:       benchmarkPromptVersionV13,
			CertificateVersion:  benchmarkCertificateVersion,
			ProjectFolder:       "hilbert-ai-verification-benchmark-v2-held-out",
			CasesFile:           "cases/cross-model-sanity.csv",
			CaseSelector:        "all",
			Hypothesis:          "hyp",
			CasesPath:           "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv",
			ResultsPath:         invalidResultPath,
			RawDir:              "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/run-invalid",
			CasesTotal:          "8",
			PassCount:           "0",
			FalseRefusalCount:   "0",
			FalseAcceptCount:    "0",
			RequestFailureCount: "8",
			SchemaFailureCount:  "0",
			ParseFailureCount:   "0",
			KernelFailureCount:  "0",
			FormatFailureCount:  "0",
			AvgLatencyMS:        "0.50",
			MaxLatencyMS:        "2",
			RunElapsedSeconds:   "0",
			CategoryCount:       "5",
		}, benchmarkRunSummaryHeader()), ","),
	}
	if err := os.WriteFile(summaryPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result_summary.csv) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runHilbertBenchmarkReportCommand([]string{
		"-project-folder", "hilbert-ai-verification-benchmark-v2-held-out",
		"-cases-file", "cases/cross-model-sanity.csv",
		"-summary", summaryPath,
		"-model-catalog", modelCatalogPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHilbertBenchmarkReportCommand() error = %v, stderr=%s", err, stderr.String())
	}

	output := stdout.String()
	requiredSnippets := []string{
		"### Engineering-best-performance (v1.2)",
		"### Clean baseline (v1.3)",
		"### <=20b cohort",
		"### >20b cohort",
		"### Unknown-size appendix",
		"### Insufficient evidence / invalid runs",
		"`GPT OSS 20B`",
		"`GPT OSS 120B Cloud`",
		"`Qwen3 12B`",
		"model not found on the target endpoint",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("report output missing %q:\n%s", snippet, output)
		}
	}
}

func TestInferBenchmarkProjectFolderAndCasesFile(t *testing.T) {
	workspaceRoot := filepath.Join("mnt", "c", "Users", "nokclock", "Documents", "GitHub", "collabsphere")
	casesPath := filepath.Join(workspaceRoot, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out", "cases", "generalization-strict.csv")

	projectFolder := inferBenchmarkProjectFolder(casesPath, workspaceRoot, "")
	if projectFolder != "hilbert-ai-verification-benchmark-v2-held-out" {
		t.Fatalf("inferBenchmarkProjectFolder() = %q", projectFolder)
	}

	casesFile := inferBenchmarkCasesFile(casesPath, workspaceRoot, projectFolder, "")
	if normalizeBenchmarkPath(casesFile) != "cases/generalization-strict.csv" {
		t.Fatalf("inferBenchmarkCasesFile() = %q", casesFile)
	}

	if selector := deriveBenchmarkCaseSelector("V2E01", 1); selector != "case_id=V2E01;limit=1" {
		t.Fatalf("deriveBenchmarkCaseSelector() = %q", selector)
	}
}

func TestCanonicalBenchmarkInputFile_NDProjectUsesTheoremPackAlias(t *testing.T) {
	if got := defaultBenchmarkInputFile(benchmarkNDProjectFolder); got != benchmarkNDTheoremPack {
		t.Fatalf("defaultBenchmarkInputFile(ND) = %q, want %q", got, benchmarkNDTheoremPack)
	}
	if got := canonicalBenchmarkInputFile(benchmarkNDProjectFolder, "cases.csv"); got != benchmarkNDTheoremPack {
		t.Fatalf("canonicalBenchmarkInputFile(ND, cases.csv) = %q, want %q", got, benchmarkNDTheoremPack)
	}
	if got := canonicalBenchmarkInputFile("hilbert-ai-verification-benchmark-v1", "cases.csv"); got != "cases.csv" {
		t.Fatalf("canonicalBenchmarkInputFile(v1, cases.csv) = %q, want cases.csv", got)
	}
}

func TestResolveBenchmarkArgPathHandlesRelativeAndAbsolutePaths(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "collabsphere-root")
	relative := filepath.ToSlash(filepath.Join("research", "artifacts", "family", "result", "result_summary_v2.csv"))
	resolved := filepath.ToSlash(resolveBenchmarkArgPath(workspaceRoot, relative))
	want := filepath.ToSlash(filepath.Join(workspaceRoot, "research", "artifacts", "family", "result", "result_summary_v2.csv"))
	if resolved != want {
		t.Fatalf("resolveBenchmarkArgPath(relative) = %q", resolved)
	}

	absolute := filepath.Join(workspaceRoot, "docs", "x.csv")
	if got := resolveBenchmarkArgPath(workspaceRoot, absolute); got != absolute {
		t.Fatalf("resolveBenchmarkArgPath(absolute) = %q, want %q", got, absolute)
	}
}

func TestLoadBenchmarkCases_NextCycleFocusedPacksParseSuccessfully(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	packs := []struct {
		path      string
		wantCount int
	}{
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv", wantCount: 8},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/assumption-import-canary.csv", wantCount: 6},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-assumption-import-phase1.csv", wantCount: 20},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-import-final-research-phase1.csv", wantCount: 20},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-phase1.csv", wantCount: 8},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-canonical-phase1.csv", wantCount: 8},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-phase2.csv", wantCount: 150},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/gold-final-composition-only-phase1.csv", wantCount: 8},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/gold-final-composition-only-phase2.csv", wantCount: 150},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-depth-ladder-phase1.csv", wantCount: 21},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-branching-phase1.csv", wantCount: 18},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-reuse-phase1.csv", wantCount: 15},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase1.csv", wantCount: 30},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase2.csv", wantCount: 100},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard.csv", wantCount: 40},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-phase2.csv", wantCount: 20},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-micro.csv", wantCount: 20},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-semi-gold-stable.csv", wantCount: 30},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-semi-gold-frontier.csv", wantCount: 30},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/entailed-gap-audit.csv", wantCount: 10},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/negative-refusal-sanity.csv", wantCount: 8},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/direct-axiom-to-mixed-proof-ladder.csv", wantCount: 10},
		{path: "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/theorem-synthesis-deep-stress.csv", wantCount: 10},
	}

	for _, pack := range packs {
		casesPath := filepath.Join(repoRoot, filepath.FromSlash(pack.path))
		cases, err := loadBenchmarkCases(casesPath)
		if err != nil {
			t.Fatalf("loadBenchmarkCases(%q) error = %v", pack.path, err)
		}
		if len(cases) != pack.wantCount {
			t.Fatalf("loadBenchmarkCases(%q) count = %d, want %d", pack.path, len(cases), pack.wantCount)
		}
		for _, item := range cases {
			if strings.TrimSpace(item.CaseID) == "" {
				t.Fatalf("loadBenchmarkCases(%q) returned blank case_id", pack.path)
			}
			if strings.TrimSpace(item.Goal) == "" {
				t.Fatalf("loadBenchmarkCases(%q) returned blank goal for case %q", pack.path, item.CaseID)
			}
			if _, err := hilbert.ParseFormula(item.Goal); err != nil {
				t.Fatalf("hilbert.ParseFormula(goal) for %q/%q error = %v", pack.path, item.CaseID, err)
			}
			for _, assumption := range item.Assumptions {
				if _, err := hilbert.ParseFormula(assumption); err != nil {
					t.Fatalf("hilbert.ParseFormula(assumption) for %q/%q error = %v", pack.path, item.CaseID, err)
				}
			}
			for _, imported := range item.ImportedLemmas {
				if _, err := hilbert.ParseFormula(imported); err != nil {
					t.Fatalf("hilbert.ParseFormula(imported lemma) for %q/%q error = %v", pack.path, item.CaseID, err)
				}
			}
		}
	}
}

func TestLoadBenchmarkCases_BridgeImportFinalResearchParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-import-final-research-phase1.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 20 {
		t.Fatalf("len(cases) = %d, want 20", len(cases))
	}
	entailed := 0
	notEntailed := 0
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
	}
	if entailed != 16 {
		t.Fatalf("entailed count = %d, want 16", entailed)
	}
	if notEntailed != 4 {
		t.Fatalf("not_entailed count = %d, want 4", notEntailed)
	}
	first := cases[0]
	if first.ChainProtocol != "bridge-import-final-v1" {
		t.Fatalf("first.ChainProtocol = %q, want bridge-import-final-v1", first.ChainProtocol)
	}
	if first.StageRole != "bridge_left" {
		t.Fatalf("first.StageRole = %q, want bridge_left", first.StageRole)
	}
	if !benchmarkCaseTrustedReuseEligible(first) {
		t.Fatalf("first trusted reuse should be eligible")
	}
	finalModel := cases[3]
	if finalModel.StageID != "fm" {
		t.Fatalf("finalModel.StageID = %q, want fm", finalModel.StageID)
	}
	if finalModel.StageRole != "final_model" {
		t.Fatalf("finalModel.StageRole = %q, want final_model", finalModel.StageRole)
	}
	if finalModel.ProvenanceMode != "model" {
		t.Fatalf("finalModel.ProvenanceMode = %q, want model", finalModel.ProvenanceMode)
	}
	if len(finalModel.ImportStageIDs) != 2 || finalModel.ImportStageIDs[0] != "b1" || finalModel.ImportStageIDs[1] != "b2" {
		t.Fatalf("finalModel.ImportStageIDs = %#v, want [b1 b2]", finalModel.ImportStageIDs)
	}
}

func TestLoadBenchmarkCases_BridgeOnlyAuthoringPhase2Balanced(t *testing.T) {
	// This keeps the expanded pack honest: it should stay a balanced bridge-only
	// probe instead of drifting back into a mixed final-composition lane.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-phase2.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 150 {
		t.Fatalf("len(cases) = %d, want 150", len(cases))
	}
	stageCounts := map[string]int{}
	difficultyCounts := map[string]int{}
	for _, item := range cases {
		if item.ChainProtocol != "bridge-only-authoring-v2" {
			t.Fatalf("item.ChainProtocol = %q, want bridge-only-authoring-v2", item.ChainProtocol)
		}
		if item.Category != "bridge_only_authoring" {
			t.Fatalf("item.Category = %q, want bridge_only_authoring", item.Category)
		}
		if item.Label != "entailed" {
			t.Fatalf("item.Label = %q, want entailed", item.Label)
		}
		if len(item.ImportedLemmas) != 0 {
			t.Fatalf("case %q imported lemmas = %#v, want none", item.CaseID, item.ImportedLemmas)
		}
		stageCounts[item.StageID]++
		difficultyCounts[item.Difficulty]++
	}
	if stageCounts["b1"] != 75 || stageCounts["b2"] != 75 {
		t.Fatalf("stageCounts = %#v, want b1=75 and b2=75", stageCounts)
	}
	if difficultyCounts["easy"] != 50 || difficultyCounts["medium"] != 50 || difficultyCounts["hard"] != 50 {
		t.Fatalf("difficultyCounts = %#v, want easy=50 medium=50 hard=50", difficultyCounts)
	}
}

func TestLoadBenchmarkCases_GoldFinalCompositionOnlyPhase2Balanced(t *testing.T) {
	// This guards the intended split: paired fg/neg rows with balanced
	// difficulty and a real imported-lemma payload on every case.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/gold-final-composition-only-phase2.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 150 {
		t.Fatalf("len(cases) = %d, want 150", len(cases))
	}
	stageCounts := map[string]int{}
	difficultyCounts := map[string]int{}
	labelCounts := map[string]int{}
	for _, item := range cases {
		if item.ChainProtocol != "gold-final-composition-only-v2" {
			t.Fatalf("item.ChainProtocol = %q, want gold-final-composition-only-v2", item.ChainProtocol)
		}
		if item.Category != "gold_final_composition_only" {
			t.Fatalf("item.Category = %q, want gold_final_composition_only", item.Category)
		}
		if len(item.ImportedLemmas) < 2 {
			t.Fatalf("case %q imported lemmas = %#v, want at least two", item.CaseID, item.ImportedLemmas)
		}
		stageCounts[item.StageID]++
		difficultyCounts[item.Difficulty]++
		labelCounts[item.Label]++
	}
	if stageCounts["fg"] != 75 || stageCounts["neg"] != 75 {
		t.Fatalf("stageCounts = %#v, want fg=75 and neg=75", stageCounts)
	}
	if difficultyCounts["easy"] != 50 || difficultyCounts["medium"] != 50 || difficultyCounts["hard"] != 50 {
		t.Fatalf("difficultyCounts = %#v, want easy=50 medium=50 hard=50", difficultyCounts)
	}
	if labelCounts["entailed"] != 75 || labelCounts["not_entailed"] != 75 {
		t.Fatalf("labelCounts = %#v, want entailed=75 and not_entailed=75", labelCounts)
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilyGoldFirstPhase2ParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase2.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 100 {
		t.Fatalf("len(cases) = %d, want 100", len(cases))
	}
	entailed := 0
	notEntailed := 0
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
	}
	if entailed != 50 {
		t.Fatalf("entailed count = %d, want 50", entailed)
	}
	if notEntailed != 50 {
		t.Fatalf("not_entailed count = %d, want 50", notEntailed)
	}
	first := cases[0]
	if first.SourceFamily != "assumption_import" {
		t.Fatalf("first.SourceFamily = %q, want assumption_import", first.SourceFamily)
	}
	if first.TargetFamily != "direct_axiom_instance" {
		t.Fatalf("first.TargetFamily = %q, want direct_axiom_instance", first.TargetFamily)
	}
	if first.TransitionGroup != "GF-MF-A" {
		t.Fatalf("first.TransitionGroup = %q, want GF-MF-A", first.TransitionGroup)
	}
	if first.ImportArity != "1" {
		t.Fatalf("first.ImportArity = %q, want 1", first.ImportArity)
	}
	if first.ReuseShape != "one_import_linear" {
		t.Fatalf("first.ReuseShape = %q, want one_import_linear", first.ReuseShape)
	}
	if first.BridgeDepth != "0" {
		t.Fatalf("first.BridgeDepth = %q, want 0", first.BridgeDepth)
	}
	if first.SymbolOverlap != "high" {
		t.Fatalf("first.SymbolOverlap = %q, want high", first.SymbolOverlap)
	}
	if first.LemmaSurfaceStyle != "atomic_implication" {
		t.Fatalf("first.LemmaSurfaceStyle = %q, want atomic_implication", first.LemmaSurfaceStyle)
	}
	if first.NegativeTwinHardness != "easy" {
		t.Fatalf("first.NegativeTwinHardness = %q, want easy", first.NegativeTwinHardness)
	}
	if strings.TrimSpace(first.Notes) == "" {
		t.Fatalf("first.Notes is blank")
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilyGoldFirstHardParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 40 {
		t.Fatalf("len(cases) = %d, want 40", len(cases))
	}
	entailed := 0
	notEntailed := 0
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
	}
	if entailed != 20 {
		t.Fatalf("entailed count = %d, want 20", entailed)
	}
	if notEntailed != 20 {
		t.Fatalf("not_entailed count = %d, want 20", notEntailed)
	}
	first := cases[0]
	if first.SourceFamily != "assumption_import" {
		t.Fatalf("first.SourceFamily = %q, want assumption_import", first.SourceFamily)
	}
	if first.TargetFamily != "direct_axiom_instance" {
		t.Fatalf("first.TargetFamily = %q, want direct_axiom_instance", first.TargetFamily)
	}
	if first.TransitionGroup != "GF-MFH-A" {
		t.Fatalf("first.TransitionGroup = %q, want GF-MFH-A", first.TransitionGroup)
	}
	if first.ImportArity != "1" {
		t.Fatalf("first.ImportArity = %q, want 1", first.ImportArity)
	}
	if first.ReuseShape != "one_import_long_bridge" {
		t.Fatalf("first.ReuseShape = %q, want one_import_long_bridge", first.ReuseShape)
	}
	if first.BridgeDepth != "2" {
		t.Fatalf("first.BridgeDepth = %q, want 2", first.BridgeDepth)
	}
	if first.SymbolOverlap != "medium" {
		t.Fatalf("first.SymbolOverlap = %q, want medium", first.SymbolOverlap)
	}
	if first.LemmaSurfaceStyle != "atomic_implication" {
		t.Fatalf("first.LemmaSurfaceStyle = %q, want atomic_implication", first.LemmaSurfaceStyle)
	}
	if first.NegativeTwinHardness != "hard" {
		t.Fatalf("first.NegativeTwinHardness = %q, want hard", first.NegativeTwinHardness)
	}
	if strings.TrimSpace(first.Notes) == "" {
		t.Fatalf("first.Notes is blank")
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilyGoldFirstHardMicroParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-micro.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 20 {
		t.Fatalf("len(cases) = %d, want 20", len(cases))
	}
	entailed := 0
	notEntailed := 0
	groups := map[string]struct{}{}
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
		groups[item.TransitionGroup] = struct{}{}
	}
	if entailed != 10 {
		t.Fatalf("entailed count = %d, want 10", entailed)
	}
	if notEntailed != 10 {
		t.Fatalf("not_entailed count = %d, want 10", notEntailed)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if _, ok := groups["GF-MFH-C"]; !ok {
		t.Fatalf("groups missing GF-MFH-C: %#v", groups)
	}
	if _, ok := groups["GF-MFH-D"]; !ok {
		t.Fatalf("groups missing GF-MFH-D: %#v", groups)
	}
	first := cases[0]
	if first.ImportArity != "2" {
		t.Fatalf("first.ImportArity = %q, want 2", first.ImportArity)
	}
	if first.ReuseShape != "two_import_merge" {
		t.Fatalf("first.ReuseShape = %q, want two_import_merge", first.ReuseShape)
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilyGoldFirstHardPhase2ParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-phase2.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 20 {
		t.Fatalf("len(cases) = %d, want 20", len(cases))
	}
	entailed := 0
	notEntailed := 0
	groups := map[string]struct{}{}
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
		groups[item.TransitionGroup] = struct{}{}
	}
	if entailed != 10 {
		t.Fatalf("entailed count = %d, want 10", entailed)
	}
	if notEntailed != 10 {
		t.Fatalf("not_entailed count = %d, want 10", notEntailed)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if _, ok := groups["GF-MF-D"]; !ok {
		t.Fatalf("groups missing GF-MF-D: %#v", groups)
	}
	if _, ok := groups["GF-MF-E"]; !ok {
		t.Fatalf("groups missing GF-MF-E: %#v", groups)
	}
	first := cases[0]
	if first.TransitionGroup != "GF-MF-D" {
		t.Fatalf("first.TransitionGroup = %q, want GF-MF-D", first.TransitionGroup)
	}
	if first.BridgeDepth != "2" {
		t.Fatalf("first.BridgeDepth = %q, want 2", first.BridgeDepth)
	}
	if first.NegativeTwinHardness != "hard" {
		t.Fatalf("first.NegativeTwinHardness = %q, want hard", first.NegativeTwinHardness)
	}
	if !strings.Contains(first.Notes, "phase2 subgroup analysis") {
		t.Fatalf("first.Notes = %q, want phase2 subgroup analysis note", first.Notes)
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilySemiGoldStableParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-semi-gold-stable.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 30 {
		t.Fatalf("len(cases) = %d, want 30", len(cases))
	}
	entailed := 0
	notEntailed := 0
	stageCounts := map[string]int{}
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
		stageCounts[item.StageID]++
	}
	if entailed != 20 {
		t.Fatalf("entailed count = %d, want 20", entailed)
	}
	if notEntailed != 10 {
		t.Fatalf("not_entailed count = %d, want 10", notEntailed)
	}
	if stageCounts["m1"] != 10 || stageCounts["fm"] != 10 || stageCounts["neg"] != 10 {
		t.Fatalf("stage counts = %#v, want 10 each for m1/fm/neg", stageCounts)
	}
	first := cases[0]
	if first.TransitionGroup != "SG-STABLE-LINEAR" {
		t.Fatalf("first.TransitionGroup = %q, want SG-STABLE-LINEAR", first.TransitionGroup)
	}
	if first.ProvenanceMode != "none" {
		t.Fatalf("first.ProvenanceMode = %q, want none", first.ProvenanceMode)
	}
	if first.ImportArity != "0" {
		t.Fatalf("first.ImportArity = %q, want 0", first.ImportArity)
	}
	if first.ReuseShape != "local_bridge_seed" {
		t.Fatalf("first.ReuseShape = %q, want local_bridge_seed", first.ReuseShape)
	}
	if first.BridgeDepth != "0" {
		t.Fatalf("first.BridgeDepth = %q, want 0", first.BridgeDepth)
	}
	if first.SymbolOverlap != "high" {
		t.Fatalf("first.SymbolOverlap = %q, want high", first.SymbolOverlap)
	}
	if !benchmarkCaseTrustedReuseEligible(first) {
		t.Fatalf("first trusted_reuse = false, want true")
	}
}

func TestLoadBenchmarkCases_CompositionalMixedFamilySemiGoldFrontierParsesAnnotations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	casesPath := filepath.Join(repoRoot, filepath.FromSlash("research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-semi-gold-frontier.csv"))
	cases, err := loadBenchmarkCases(casesPath)
	if err != nil {
		t.Fatalf("loadBenchmarkCases() error = %v", err)
	}
	if len(cases) != 30 {
		t.Fatalf("len(cases) = %d, want 30", len(cases))
	}
	entailed := 0
	notEntailed := 0
	stageCounts := map[string]int{}
	groups := map[string]struct{}{}
	for _, item := range cases {
		switch item.Label {
		case "entailed":
			entailed++
		case "not_entailed":
			notEntailed++
		}
		stageCounts[item.StageID]++
		groups[item.TransitionGroup] = struct{}{}
	}
	if entailed != 20 {
		t.Fatalf("entailed count = %d, want 20", entailed)
	}
	if notEntailed != 10 {
		t.Fatalf("not_entailed count = %d, want 10", notEntailed)
	}
	if stageCounts["m1"] != 10 || stageCounts["fm"] != 10 || stageCounts["neg"] != 10 {
		t.Fatalf("stage counts = %#v, want 10 each for m1/fm/neg", stageCounts)
	}
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if _, ok := groups["SG-FRONTIER-MERGE"]; !ok {
		t.Fatalf("groups missing SG-FRONTIER-MERGE: %#v", groups)
	}
	if _, ok := groups["SG-FRONTIER-ADVERSARIAL"]; !ok {
		t.Fatalf("groups missing SG-FRONTIER-ADVERSARIAL: %#v", groups)
	}
	first := cases[0]
	if first.TransitionGroup != "SG-FRONTIER-MERGE" {
		t.Fatalf("first.TransitionGroup = %q, want SG-FRONTIER-MERGE", first.TransitionGroup)
	}
	if first.ImportArity != "0" {
		t.Fatalf("first.ImportArity = %q, want 0", first.ImportArity)
	}
	if first.ReuseShape != "local_bridge_seed" {
		t.Fatalf("first.ReuseShape = %q, want local_bridge_seed", first.ReuseShape)
	}
	if first.BridgeDepth != "1" {
		t.Fatalf("first.BridgeDepth = %q, want 1", first.BridgeDepth)
	}
	if first.SymbolOverlap != "low" {
		t.Fatalf("first.SymbolOverlap = %q, want low", first.SymbolOverlap)
	}
	if !strings.Contains(first.Notes, "frontier-axis semi-gold") {
		t.Fatalf("first.Notes = %q, want frontier-axis note", first.Notes)
	}
}

func TestBuildBenchmarkChainSummaryRowsCollapsesHeterogeneousCaseFamilies(t *testing.T) {
	cases := []benchmarkCase{
		{CaseID: "MFR01-S1", ChainID: "MFR01", ChainProtocol: "mixed-family-v1", StageID: "s1", StageOrder: 1, StageRole: "source_family_left", CaseFamily: "assumption_import_source", ChainDepth: 2, TrustedReuse: boolPtr(true)},
		{CaseID: "MFR01-S2", ChainID: "MFR01", ChainProtocol: "mixed-family-v1", StageID: "s2", StageOrder: 2, StageRole: "source_family_right", CaseFamily: "direct_axiom_instance_source", ChainDepth: 2, TrustedReuse: boolPtr(true)},
		{CaseID: "MFR01-FM", ChainID: "MFR01", ChainProtocol: "mixed-family-v1", StageID: "fm", StageOrder: 3, StageRole: "final_model", CaseFamily: "mixed_family_merge", ChainDepth: 2, ProvenanceMode: "model"},
	}
	rows := []benchmarkResultRow{
		{CaseID: "MFR01-S1", ScoreBucket: "pass"},
		{CaseID: "MFR01-S2", ScoreBucket: "pass"},
		{CaseID: "MFR01-FM", ScoreBucket: "pass"},
	}

	summaries := buildBenchmarkChainSummaryRows("mixed-family-20260403", cases, rows)
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].CaseFamily != "mixed_family_reuse" {
		t.Fatalf("summaries[0].CaseFamily = %q, want mixed_family_reuse", summaries[0].CaseFamily)
	}
}

func TestRenderBenchmarkChainSummarySections_HidesLegacyModelColumnsForGoldFirstLane(t *testing.T) {
	data := &benchmarkChainSummaryData{
		Aggregates: []benchmarkChainSummaryAggregate{
			{
				Model:                         "compatible:gpt-oss:20b",
				LLMModel:                      "gpt-oss:20b",
				DisplayName:                   "gpt-oss:20b [exp: gold-first]",
				PromptVersion:                 "hilbert-ai-verification-benchmark-v1.3",
				Surface:                       "direct",
				Runs:                          2,
				ChainsTotal:                   3,
				FinalCompositionPassGoldCount: 2,
				NegativeTwinPassCount:         3,
				AllStagesPassCount:            2,
				ObservedStageIDs:              []string{"fg", "neg"},
				LatestRunID:                   "gold-first-20260403",
			},
		},
		StageAggregates: []benchmarkChainStageAggregate{
			{StageID: "fg", StageRole: "gold_import_stage", Chains: 3, PassCount: 2},
			{StageID: "neg", StageRole: "negative_control_stage", Chains: 3, PassCount: 3},
		},
	}

	var b strings.Builder
	renderBenchmarkChainSummarySections(&b, data, "cases/compositional-mixed-family-gold-first-phase1.csv")
	output := b.String()
	if !strings.Contains(output, "Gold Final") {
		t.Fatalf("chain summary output missing Gold Final column:\n%s", output)
	}
	if strings.Contains(output, "Model Final") {
		t.Fatalf("chain summary output should hide Model Final for gold-first lane:\n%s", output)
	}
	if strings.Contains(output, "Conditional Gold") {
		t.Fatalf("chain summary output should hide Conditional Gold when no import/resource stage is present:\n%s", output)
	}
	if !strings.Contains(output, "This lane records gold-final composition and any available negative-control discipline") {
		t.Fatalf("chain summary output missing gold-first explanatory text:\n%s", output)
	}
	if !strings.Contains(output, "#### Starter-stage breakdown") {
		t.Fatalf("chain summary output missing starter-stage grouped section:\n%s", output)
	}
	if !strings.Contains(output, "No starter-stage rows are present in this pack.") {
		t.Fatalf("chain summary output should state that starter-stage rows are absent in gold-first lane:\n%s", output)
	}
	if !strings.Contains(output, "#### Import-stage breakdown") {
		t.Fatalf("chain summary output missing import-stage grouped section:\n%s", output)
	}
	if !strings.Contains(output, "#### Negative-control breakdown") {
		t.Fatalf("chain summary output missing negative-control grouped section:\n%s", output)
	}
}

func TestRenderBenchmarkChainSummarySections_RendersPhase2GroupedSlices(t *testing.T) {
	data := &benchmarkChainSummaryData{
		Aggregates: []benchmarkChainSummaryAggregate{
			{
				Model:                         "compatible:glm-5:cloud",
				LLMModel:                      "glm-5:cloud",
				DisplayName:                   "glm-5:cloud [exp: gold-first-phase2]",
				PromptVersion:                 "hilbert-ai-verification-benchmark-v1.3",
				Surface:                       "direct",
				Runs:                          1,
				ChainsTotal:                   50,
				FinalCompositionPassGoldCount: 38,
				NegativeTwinPassCount:         47,
				AllStagesPassCount:            38,
				ObservedStageIDs:              []string{"fg", "neg"},
				LatestRunID:                   "gold-first-phase2-20260403",
			},
		},
		StageAggregates: []benchmarkChainStageAggregate{
			{StageID: "fg", StageRole: "gold_import_stage", Chains: 50, PassCount: 38},
			{StageID: "neg", StageRole: "negative_control_stage", Chains: 50, PassCount: 47},
		},
		TransitionGroups: []benchmarkChainMetadataAggregate{
			{GroupValue: "GF-MF-A", ChainsTotal: 10, GoldFinalPassCount: 9, NegativeTwinPassCount: 10},
		},
		ImportArities: []benchmarkChainMetadataAggregate{
			{GroupValue: "2", ChainsTotal: 30, GoldFinalPassCount: 21, NegativeTwinPassCount: 27},
		},
		ReuseShapes: []benchmarkChainMetadataAggregate{
			{GroupValue: "adversarial_gold_reuse", ChainsTotal: 10, GoldFinalPassCount: 6, NegativeTwinPassCount: 8},
		},
		BridgeDepths: []benchmarkChainMetadataAggregate{
			{GroupValue: "2", ChainsTotal: 20, GoldFinalPassCount: 14, NegativeTwinPassCount: 18},
		},
		SymbolOverlaps: []benchmarkChainMetadataAggregate{
			{GroupValue: "low", ChainsTotal: 20, GoldFinalPassCount: 13, NegativeTwinPassCount: 17},
		},
		NegativeTwinHardnesses: []benchmarkChainMetadataAggregate{
			{GroupValue: "hard", ChainsTotal: 20, GoldFinalPassCount: 14, NegativeTwinPassCount: 18},
		},
		FGFailureMix: []benchmarkFGFailureAggregate{
			{FailureType: "schema_failure", Cases: 4},
			{FailureType: "kernel_failure", Cases: 2},
		},
		HardestFGFailures: []benchmarkFGCaseFailureAggregate{
			{CaseID: "GMF2E04-FG", ChainID: "GMF2E04", TransitionGroup: "GF-MF-E", FailureCount: 2, SchemaFailureCount: 2},
		},
	}

	var b strings.Builder
	renderBenchmarkChainSummarySections(&b, data, "cases/compositional-mixed-family-gold-first-phase2.csv")
	output := b.String()
	for _, needle := range []string{
		"#### Per-transition-group",
		"#### Per-import-arity",
		"#### Per-reuse-shape",
		"#### Per-bridge-depth",
		"#### Per-symbol-overlap",
		"#### Per-negative-twin-hardness",
		"#### FG failure-type mix",
		"#### Hardest FG failures",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("chain summary output missing %q:\n%s", needle, output)
		}
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func TestResolveBenchmarkPromptCase_ModelImportUsesTrustedArtifactObjects(t *testing.T) {
	priorArtifacts := []benchmarkTrustedCertificateObject{
		{RunID: "run-001", CaseID: "CI01-S1", ChainID: "CI01", StageID: "s1", Goal: "A -> B", TrustedForReuse: true, CertificateObjectPath: "result/imported_certificates/run-001/CI01-S1.json"},
		{RunID: "run-001", CaseID: "CI01-S2", ChainID: "CI01", StageID: "s2", Goal: "B -> C", TrustedForReuse: false, CertificateObjectPath: "result/imported_certificates/run-001/CI01-S2.json"},
	}
	source := benchmarkCase{
		CaseID:         "CI01-S3M",
		ChainID:        "CI01",
		StageID:        "s3m",
		ProvenanceMode: "model",
		ImportStageIDs: []string{"s1"},
		ImportedLemmas: []string{"A -> B", "B -> C"},
		Assumptions:    []string{"D -> E"},
		Goal:           "A -> C",
	}

	resolved, err := resolveBenchmarkPromptCase(source, priorArtifacts)
	if err != nil {
		t.Fatalf("resolveBenchmarkPromptCase() error = %v", err)
	}
	if !slices.Equal(resolved.ResolvedCase.ImportedLemmas, []string{"A -> B"}) {
		t.Fatalf("resolveBenchmarkPromptCase() imported lemmas = %#v, want only trusted reusable lemma", resolved.ResolvedCase.ImportedLemmas)
	}
	if len(resolved.ResolvedArtifacts) != 1 || resolved.ResolvedArtifacts[0].CaseID != "CI01-S1" {
		t.Fatalf("resolveBenchmarkPromptCase() resolved artifacts = %#v, want only CI01-S1 trusted artifact", resolved.ResolvedArtifacts)
	}

	prompt := buildBenchmarkUserPrompt(resolved.ResolvedCase)
	if !strings.Contains(prompt, "Imported lemmas:\n1. A -> B\n") {
		t.Fatalf("buildBenchmarkUserPrompt() missing imported lemma section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Available assumptions for certificate construction:\n1. D -> E\n2. A -> B\n") {
		t.Fatalf("buildBenchmarkUserPrompt() missing available assumptions section:\n%s", prompt)
	}
}

func TestResolveBenchmarkPromptCase_SemiGoldMergesStaticAndTrustedModelImports(t *testing.T) {
	priorArtifacts := []benchmarkTrustedCertificateObject{
		{RunID: "run-002", CaseID: "SG01-M1", ChainID: "SG01", StageID: "m1", Goal: "B", TrustedForReuse: true, CertificateObjectPath: "result/imported_certificates/run-002/SG01-M1.json"},
		{RunID: "run-002", CaseID: "SG01-M2", ChainID: "SG01", StageID: "m2", Goal: "C", TrustedForReuse: false, CertificateObjectPath: "result/imported_certificates/run-002/SG01-M2.json"},
	}
	source := benchmarkCase{
		CaseID:         "SG01-FM",
		ChainID:        "SG01",
		StageID:        "fm",
		ProvenanceMode: "semi_gold",
		ImportStageIDs: []string{"m1"},
		ImportedLemmas: []string{"B -> C"},
		Goal:           "C",
	}

	resolved, err := resolveBenchmarkPromptCase(source, priorArtifacts)
	if err != nil {
		t.Fatalf("resolveBenchmarkPromptCase() error = %v", err)
	}
	if !slices.Equal(resolved.ResolvedCase.ImportedLemmas, []string{"B -> C", "B"}) {
		t.Fatalf("resolveBenchmarkPromptCase() imported lemmas = %#v, want static gold plus trusted model bridge", resolved.ResolvedCase.ImportedLemmas)
	}
	if len(resolved.ResolvedArtifacts) != 1 || resolved.ResolvedArtifacts[0].CaseID != "SG01-M1" {
		t.Fatalf("resolveBenchmarkPromptCase() resolved artifacts = %#v, want only SG01-M1 trusted artifact", resolved.ResolvedArtifacts)
	}

	prompt := buildBenchmarkUserPrompt(resolved.ResolvedCase)
	if !strings.Contains(prompt, "Imported lemmas:\n1. B -> C\n2. B\n") {
		t.Fatalf("buildBenchmarkUserPrompt() missing semi-gold imported lemmas section:\n%s", prompt)
	}
}

func TestResolveBenchmarkPromptCaseFailsWhenRequestedImportDidNotMaterialize(t *testing.T) {
	source := benchmarkCase{
		CaseID:         "CI01-S3M",
		ChainID:        "CI01",
		StageID:        "s3m",
		ProvenanceMode: "model",
		ImportStageIDs: []string{"s1"},
		Goal:           "A -> C",
	}

	_, err := resolveBenchmarkPromptCase(source, nil)
	if err == nil || !strings.Contains(err.Error(), "unmaterialized_import_artifact") {
		t.Fatalf("resolveBenchmarkPromptCase() error = %v, want unmaterialized_import_artifact", err)
	}
}

func TestPopulateBenchmarkContractStatusRejectsWrongDomain(t *testing.T) {
	row := benchmarkResultRow{
		ScoreBucket:    "pass",
		ContractStatus: "not_run",
	}
	rawOutput := `{"certificate_version":"1.0.0","proof_id":"H01","goal":"P","context":{"domain":"wrong-domain","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":["P"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"P"}]}`

	if err := populateBenchmarkContractStatus(&row, rawOutput); err != nil {
		t.Fatalf("populateBenchmarkContractStatus() error = %v", err)
	}
	if row.ContractStatus != "fail" {
		t.Fatalf("ContractStatus = %q, want fail", row.ContractStatus)
	}
	if row.ScoreBucket != "contract_failure" {
		t.Fatalf("ScoreBucket = %q, want contract_failure", row.ScoreBucket)
	}
	if !strings.Contains(row.Notes, "context.domain") {
		t.Fatalf("Notes = %q, want benchmark contract detail", row.Notes)
	}
}

func TestPopulateBenchmarkContractStatusAcceptsCanonicalBenchmarkMetadata(t *testing.T) {
	row := benchmarkResultRow{
		ScoreBucket:    "pass",
		ContractStatus: "not_run",
	}
	rawOutput := `{"certificate_version":"` + certificates.FormatVersionV1 + `","proof_id":"H01","goal":"P","context":{"domain":"hilbert-benchmark-v1","rule_pack":"` + hilbert.RulePackID + `","syntax":"` + hilbert.SyntaxID + `","generator":"llm-benchmark-v1"},"assumptions":["P"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"P"}]}`

	if err := populateBenchmarkContractStatus(&row, rawOutput); err != nil {
		t.Fatalf("populateBenchmarkContractStatus() error = %v", err)
	}
	if row.ContractStatus != "pass" {
		t.Fatalf("ContractStatus = %q, want pass", row.ContractStatus)
	}
	if row.ScoreBucket != "pass" {
		t.Fatalf("ScoreBucket = %q, want pass", row.ScoreBucket)
	}
}

func TestActiveHilbertBenchmarkPromptsDoNotPromiseUnverifiedProvenanceFields(t *testing.T) {
	for _, version := range []string{
		benchmarkPromptVersionV11,
		benchmarkPromptVersionV12,
		benchmarkPromptVersionV13,
		benchmarkPromptVersionBridgeOnlyV1,
		benchmarkPromptVersionBridgeOnlySkeletonV1,
		benchmarkPromptVersionBridgeOnlyCanonicalVarsV1,
		benchmarkPromptVersionGoldFinalOnlyV1,
		benchmarkPromptVersionGoldFinalOnlyExplicitImportRefsV1,
	} {
		profile, err := resolveBenchmarkPromptProfile(version)
		if err != nil {
			t.Fatalf("resolveBenchmarkPromptProfile(%q) error = %v", version, err)
		}
		if strings.Contains(profile.SystemPrompt, "- sources") || strings.Contains(profile.SystemPrompt, "- hashes") {
			t.Fatalf("%s prompt still mentions sources/hashes as allowed top-level fields:\n%s", version, profile.SystemPrompt)
		}
	}
}

func extractCaseID(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		if strings.HasPrefix(line, "Theorem ID: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Theorem ID: "))
		}
		if strings.HasPrefix(line, "Case ID: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Case ID: "))
		}
	}
	return ""
}
