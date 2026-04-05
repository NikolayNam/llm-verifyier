package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunNaturalDeductionBenchmarkWritesRowsAndLoweredOutputs(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment",
		`NDI02,single_mp,entailed,easy,"[""P"",""P -> Q""]",Q,implicational-prop-v1,Single implication elimination`,
		`NDI07,negative_refusal,not_entailed,easy,"[""P -> Q""]",P,implicational-prop-v1,Consequent does not imply antecedent`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	summary, err := runNaturalDeductionBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "nd-run-001",
		LLMModel:      "test-model",
		PromptVersion: ndBenchmarkPromptVersionV1,
		ModelLabel:    "compatible:test-model",
	}, ndBenchmarkPromptProfiles[ndBenchmarkPromptVersionV1], &fakeBenchmarkModel{
		outputs: map[string]string{
			"NDI02": `{"proof_version":"1.0.0","proof_id":"NDI02","goal":"Q","assumptions":["P","P -> Q"],"context":{"domain":"hilbert-benchmark-nd-v1","logic_fragment":"implicational-prop-v1","front_end":"natural-deduction-v1","lowering_target":"certificate-format-v1","generator":"llm-benchmark-nd-v1"},"steps":[{"id":1,"kind":"premise","premise_ref":1,"formula":"P"},{"id":2,"kind":"premise","premise_ref":2,"formula":"P -> Q"},{"id":3,"kind":"imp_elim","from":[1,2],"formula":"Q"}]}`,
			"NDI07": "NOT_DERIVABLE",
		},
	}, &fakeBenchmarkVerifier{
		results: map[string]*benchmarkVerifyResult{
			"NDI02": {ExitCode: 0},
		},
	}, func() time.Time {
		return time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runNaturalDeductionBenchmark() error = %v", err)
	}

	if summary.CaseCount != 2 {
		t.Fatalf("summary.CaseCount = %d, want 2", summary.CaseCount)
	}
	if summary.Buckets["pass"] != 2 {
		t.Fatalf("summary.Buckets[pass] = %d, want 2", summary.Buckets["pass"])
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	resultsText := string(resultsRaw)
	if !strings.Contains(resultsText, "NDI02") || !strings.Contains(resultsText, "NDI07") {
		t.Fatalf("results.csv missing expected case rows:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, "nd_proof_status") || !strings.Contains(resultsText, "lowering_status") || !strings.Contains(resultsText, "pipeline_stage") || !strings.Contains(resultsText, "pipeline_detail") {
		t.Fatalf("results.csv missing ND trace columns:\n%s", resultsText)
	}
	if !strings.Contains(resultsText, "hilbert_artifact") || !strings.Contains(resultsText, "not_derivable") {
		t.Fatalf("results.csv missing expected ND trace values:\n%s", resultsText)
	}

	if _, err := os.Stat(filepath.Join(rawDir, "nd-run-001", "NDI02.nd.json")); err != nil {
		t.Fatalf("missing raw ND proof json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rawDir, "nd-run-001", "NDI02.json")); err != nil {
		t.Fatalf("missing lowered certificate json: %v", err)
	}
}

func TestRunNaturalDeductionBenchmarkRecordsLoweringTrace(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment",
		`NDI07,negative_refusal,not_entailed,easy,"[""P -> Q""]",P,implicational-prop-v1,Consequent does not imply antecedent`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	_, err := runNaturalDeductionBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:     casesPath,
		ResultsPath:   resultsPath,
		RawBaseDir:    rawDir,
		RunID:         "nd-run-lowering-001",
		LLMModel:      "test-model",
		PromptVersion: ndBenchmarkPromptVersionV11,
		ModelLabel:    "compatible:test-model",
	}, ndBenchmarkPromptProfiles[ndBenchmarkPromptVersionV11], &fakeBenchmarkModel{
		outputs: map[string]string{
			"NDI07": `{"proof_version":"1.0.0","proof_id":"NDI07","goal":"P","assumptions":["P -> Q"],"context":{"domain":"hilbert-benchmark-nd-v1","logic_fragment":"implicational-prop-v1","front_end":"natural-deduction-v1","lowering_target":"certificate-format-v1","generator":"llm-benchmark-nd-v1"},"steps":[{"id":1,"kind":"premise","premise_ref":1,"formula":"P -> Q"}]}`,
		},
	}, &fakeBenchmarkVerifier{}, func() time.Time {
		return time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runNaturalDeductionBenchmark() error = %v", err)
	}

	resultsRaw, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile(results.csv) error = %v", err)
	}
	resultsText := string(resultsRaw)
	if !strings.Contains(resultsText, "lowering") || !strings.Contains(resultsText, "validation_error") {
		t.Fatalf("results.csv missing lowering trace:\n%s", resultsText)
	}
}

func TestRunNaturalDeductionBenchmarkShortCircuitsRemainingCasesAfterConfiguredTimeoutThreshold(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.csv")
	resultsPath := filepath.Join(root, "results.csv")
	rawDir := filepath.Join(root, "raw")

	casesCSV := strings.Join([]string{
		"theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment",
		`pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI02,NDI02,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI03,NDI03,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI04,NDI04,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
		`pilot_shared_20260327,NDI05,NDI05,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import`,
	}, "\n")
	if err := os.WriteFile(casesPath, []byte(casesCSV), 0o644); err != nil {
		t.Fatalf("WriteFile(cases.csv) error = %v", err)
	}

	model := &fakeBenchmarkModel{
		errors: map[string]error{
			"NDI01": context.DeadlineExceeded,
			"NDI02": context.DeadlineExceeded,
			"NDI03": context.DeadlineExceeded,
		},
		outputs: map[string]string{
			"NDI04": "NOT_DERIVABLE",
			"NDI05": "NOT_DERIVABLE",
		},
	}

	summary, err := runNaturalDeductionBenchmark(context.Background(), benchmarkRunOptions{
		CasesPath:                    casesPath,
		ResultsPath:                  resultsPath,
		RawBaseDir:                   rawDir,
		RunID:                        "nd-timeout-abort-001",
		LLMModel:                     "test-model",
		PromptVersion:                ndBenchmarkPromptVersionV11,
		ModelLabel:                   "compatible:test-model",
		RequestTimeoutAbortThreshold: 3,
	}, ndBenchmarkPromptProfiles[ndBenchmarkPromptVersionV11], model, &fakeBenchmarkVerifier{}, func() time.Time {
		return time.Date(2026, 4, 2, 7, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("runNaturalDeductionBenchmark() error = %v", err)
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
	for _, caseID := range []string{"NDI01", "NDI02", "NDI03", "NDI04", "NDI05"} {
		if !strings.Contains(resultsText, caseID) {
			t.Fatalf("results.csv missing case %s:\n%s", caseID, resultsText)
		}
	}
	if !strings.Contains(resultsText, "short-circuited remaining cases after 3 consecutive request timeouts") {
		t.Fatalf("results.csv missing timeout short-circuit note:\n%s", resultsText)
	}

	for _, caseID := range []string{"NDI04", "NDI05"} {
		rawPath := filepath.Join(rawDir, "nd-timeout-abort-001", caseID+".txt")
		rawText, err := os.ReadFile(rawPath)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", rawPath, err)
		}
		if !strings.Contains(string(rawText), "short-circuited remaining cases after 3 consecutive request timeouts") {
			t.Fatalf("synthetic raw output for %s missing timeout short-circuit note:\n%s", caseID, string(rawText))
		}
	}
}

func TestResolveNDBenchmarkPromptProfile(t *testing.T) {
	if _, err := resolveNDBenchmarkPromptProfile("unknown"); err == nil {
		t.Fatal("resolveNDBenchmarkPromptProfile(unknown) expected error")
	}

	for _, version := range []string{ndBenchmarkPromptVersionV1, ndBenchmarkPromptVersionV11} {
		profile, err := resolveNDBenchmarkPromptProfile(version)
		if err != nil {
			t.Fatalf("resolveNDBenchmarkPromptProfile(%q) error = %v", version, err)
		}
		if profile.Version != version {
			t.Fatalf("profile.Version = %q, want %q", profile.Version, version)
		}
		if strings.TrimSpace(profile.SystemPrompt) == "" {
			t.Fatalf("profile.SystemPrompt for %q is empty", version)
		}
	}
}

func TestNDBenchmarkPromptV11TightensRequiredStepFields(t *testing.T) {
	profile := ndBenchmarkPromptProfiles[ndBenchmarkPromptVersionV11]
	requiredFragments := []string{
		`every step object MUST include: id, kind, formula`,
		`premise MUST omit from, scope, and discharge_scope`,
		`imp_intro requires discharge_scope`,
		`do not omit id`,
		`do not omit formula`,
		`do not rename discharge_scope`,
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(profile.SystemPrompt, fragment) {
			t.Fatalf("v1.1 prompt missing fragment %q", fragment)
		}
	}
	if !strings.Contains(profile.SystemPrompt, `This prompt intentionally contains no inline examples.`) {
		t.Fatalf("v1.1 prompt unexpectedly lost the no-example constraint")
	}
}
