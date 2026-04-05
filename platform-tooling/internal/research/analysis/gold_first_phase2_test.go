package analysis

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeGoldFirstPhase2(t *testing.T) {
	root := t.TempDir()
	summaryDir := filepath.Join(root, "summaries")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatalf("mkdir summaries: %v", err)
	}

	validResult := filepath.Join(root, "result_valid.csv")
	validChain := filepath.Join(root, "chain_valid.csv")
	invalidResult := filepath.Join(root, "result_invalid.csv")
	invalidChain := filepath.Join(root, "chain_invalid.csv")

	writeTestFile(t, validResult, strings.Join([]string{
		"run_id,llm_model,stage_id,expected_label,score_bucket,transition_group,import_arity,reuse_shape,bridge_depth,symbol_overlap,negative_twin_hardness",
		"job-valid,gpt-oss:120b-cloud,fg,entailed,pass,GF-MF-A,1,one_import_linear,0,high,easy",
		"job-valid,gpt-oss:120b-cloud,fg,entailed,schema_failure,GF-MF-B,2,two_import_merge,1,low,easy",
		"job-valid,gpt-oss:120b-cloud,neg,not_entailed,pass,GF-MF-B,2,two_import_merge,1,low,hard",
		"job-valid,gpt-oss:120b-cloud,neg,not_entailed,pass,GF-MF-A,1,one_import_linear,0,high,hard",
	}, "\n"))
	writeTestFile(t, validChain, strings.Join([]string{
		"run_id,final_composition_pass_gold,negative_twin_pass,failure_type",
		"job-valid,true,true,",
		"job-valid,true,true,",
		"job-valid,false,true,schema_failure",
	}, "\n"))
	writeTestFile(t, invalidResult, strings.Join([]string{
		"run_id,llm_model,stage_id,expected_label,score_bucket,transition_group,import_arity,reuse_shape,bridge_depth,symbol_overlap,negative_twin_hardness",
		"job-invalid,gpt-oss:120b-cloud,fg,entailed,request_failure,GF-MF-E,2,two_import_merge,2,low,hard",
		"job-invalid,gpt-oss:120b-cloud,neg,not_entailed,request_failure,GF-MF-E,2,two_import_merge,2,low,hard",
	}, "\n"))
	writeTestFile(t, invalidChain, strings.Join([]string{
		"run_id,final_composition_pass_gold,negative_twin_pass,failure_type",
		"job-invalid,false,false,request_failure",
		"job-invalid,false,false,request_failure",
	}, "\n"))

	writeTestFile(t, filepath.Join(summaryDir, GoldFirstPhase2Phase+"_20260403T100000+0300.csv"), strings.Join([]string{
		"run_id,llm_model,cases_total,request_failure_count,schema_failure_count,kernel_failure_count,parse_failure_count,format_failure_count,false_accept_count,false_refusal_count,result_file,chain_result_file",
		"job-invalid,gpt-oss:120b-cloud,100,30,2,0,0,0,0,0," + filepath.ToSlash(invalidResult) + "," + filepath.ToSlash(invalidChain),
	}, "\n"))
	writeTestFile(t, filepath.Join(summaryDir, GoldFirstPhase2Phase+"_20260403T110000+0300.csv"), strings.Join([]string{
		"run_id,llm_model,cases_total,request_failure_count,schema_failure_count,kernel_failure_count,parse_failure_count,format_failure_count,false_accept_count,false_refusal_count,result_file,chain_result_file",
		"job-valid,gpt-oss:120b-cloud,100,0,4,1,0,0,0,0," + filepath.ToSlash(validResult) + "," + filepath.ToSlash(validChain),
	}, "\n"))

	artifacts, err := AnalyzeGoldFirstPhase2(context.Background(), GoldFirstPhase2Options{
		SummaryDir:           summaryDir,
		ValidityCSVPath:      filepath.Join(root, "analysis", "validity.csv"),
		ValidityMarkdownPath: filepath.Join(root, "analysis", "validity.md"),
		SubgroupCSVPath:      filepath.Join(root, "analysis", "subgroups.csv"),
		SubgroupMarkdownPath: filepath.Join(root, "analysis", "subgroups.md"),
	})
	if err != nil {
		t.Fatalf("AnalyzeGoldFirstPhase2 returned error: %v", err)
	}

	if len(artifacts.ValidityRows) != 2 {
		t.Fatalf("expected 2 validity rows, got %d", len(artifacts.ValidityRows))
	}
	if artifacts.ValidityRows[0].Classification != "execution-invalidated" {
		t.Fatalf("expected first run to be execution-invalidated, got %q", artifacts.ValidityRows[0].Classification)
	}
	if artifacts.ValidityRows[1].Classification != "reasoning-valid" {
		t.Fatalf("expected second run to be reasoning-valid, got %q", artifacts.ValidityRows[1].Classification)
	}
	if len(artifacts.SubgroupRows) == 0 {
		t.Fatalf("expected subgroup rows to be generated")
	}

	subgroupCSV, err := os.ReadFile(artifacts.SubgroupCSVPath)
	if err != nil {
		t.Fatalf("read subgroup csv: %v", err)
	}
	if !strings.Contains(string(subgroupCSV), "transition_group") {
		t.Fatalf("expected subgroup csv to contain transition_group rows")
	}

	validityMD, err := os.ReadFile(artifacts.ValidityMarkdownPath)
	if err != nil {
		t.Fatalf("read validity md: %v", err)
	}
	if !strings.Contains(string(validityMD), "execution-invalidated") {
		t.Fatalf("expected validity markdown to mention execution-invalidated")
	}

	subgroupMD, err := os.ReadFile(artifacts.SubgroupMarkdownPath)
	if err != nil {
		t.Fatalf("read subgroup md: %v", err)
	}
	if !strings.Contains(string(subgroupMD), "Gold Final pass rate by transition group") {
		t.Fatalf("expected subgroup markdown to contain transition-group section")
	}
}

func TestAnalyzeGoldFirstPhase2HardPack(t *testing.T) {
	root := t.TempDir()
	summaryDir := filepath.Join(root, "summaries")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatalf("mkdir summaries: %v", err)
	}

	matchingResult := filepath.Join(root, "result_hard_phase2.csv")
	matchingChain := filepath.Join(root, "chain_hard_phase2.csv")
	otherResult := filepath.Join(root, "result_other.csv")
	otherChain := filepath.Join(root, "chain_other.csv")

	writeTestFile(t, matchingResult, strings.Join([]string{
		"run_id,llm_model,case_id,stage_id,expected_label,score_bucket,transition_group,import_arity,reuse_shape,bridge_depth,symbol_overlap,negative_twin_hardness",
		"job-hard,gpt-oss:120b-cloud,GMF2D01-FG,fg,entailed,pass,GF-MF-D,1,bridge_depth_ge_1,2,medium,hard",
		"job-hard,gpt-oss:120b-cloud,GMF2E01-FG,fg,entailed,schema_failure,GF-MF-E,2,adversarial_gold_reuse,2,low,hard",
		"job-hard,gpt-oss:120b-cloud,GMF2D01-NEG,neg,not_entailed,pass,GF-MF-D,1,bridge_depth_ge_1,2,medium,hard",
		"job-hard,gpt-oss:120b-cloud,GMF2E01-NEG,neg,not_entailed,pass,GF-MF-E,2,adversarial_gold_reuse,2,low,hard",
	}, "\n"))
	writeTestFile(t, matchingChain, strings.Join([]string{
		"run_id,final_composition_pass_gold,negative_twin_pass,failure_type",
		"job-hard,true,true,",
		"job-hard,false,true,schema_failure",
	}, "\n"))
	writeTestFile(t, otherResult, strings.Join([]string{
		"run_id,llm_model,case_id,stage_id,expected_label,score_bucket,transition_group,import_arity,reuse_shape,bridge_depth,symbol_overlap,negative_twin_hardness",
		"job-other,gpt-oss:120b-cloud,GMFH01-FG,fg,entailed,pass,GF-MFH-A,1,one_import_long_bridge,2,medium,hard",
	}, "\n"))
	writeTestFile(t, otherChain, strings.Join([]string{
		"run_id,final_composition_pass_gold,negative_twin_pass,failure_type",
		"job-other,true,true,",
	}, "\n"))

	writeTestFile(t, filepath.Join(summaryDir, GoldFirstPhase2HardPackPhase+"_20260404T090000+0300.csv"), strings.Join([]string{
		"run_id,llm_model,cases_file,cases_total,request_failure_count,schema_failure_count,kernel_failure_count,parse_failure_count,format_failure_count,false_accept_count,false_refusal_count,result_file,chain_result_file",
		"job-other,gpt-oss:120b-cloud,cases/compositional-mixed-family-gold-first-hard.csv,40,0,0,0,0,0,0,0," + filepath.ToSlash(otherResult) + "," + filepath.ToSlash(otherChain),
	}, "\n"))
	writeTestFile(t, filepath.Join(summaryDir, GoldFirstPhase2HardPackPhase+"_20260404T100000+0300.csv"), strings.Join([]string{
		"run_id,llm_model,cases_file,cases_total,request_failure_count,schema_failure_count,kernel_failure_count,parse_failure_count,format_failure_count,false_accept_count,false_refusal_count,result_file,chain_result_file",
		"job-hard,gpt-oss:120b-cloud,cases/compositional-mixed-family-gold-first-hard-phase2.csv,20,0,1,0,0,0,0,0," + filepath.ToSlash(matchingResult) + "," + filepath.ToSlash(matchingChain),
	}, "\n"))

	artifacts, err := AnalyzeGoldFirstPhase2HardPack(context.Background(), GoldFirstPhase2HardPackOptions{
		SummaryDir:         summaryDir,
		CasesFile:          GoldFirstPhase2HardPackCasesFile,
		ReportMarkdownPath: filepath.Join(root, "analysis", "gold_first_phase2_hard_pack_report.md"),
	})
	if err != nil {
		t.Fatalf("AnalyzeGoldFirstPhase2HardPack returned error: %v", err)
	}

	if len(artifacts.RunRows) != 1 {
		t.Fatalf("expected 1 hard-pack run row, got %d", len(artifacts.RunRows))
	}
	if len(artifacts.ModelRows) != 1 {
		t.Fatalf("expected 1 hard-pack model row, got %d", len(artifacts.ModelRows))
	}
	if artifacts.EntailedChains != 2 {
		t.Fatalf("artifacts.EntailedChains = %d, want 2", artifacts.EntailedChains)
	}
	if artifacts.NegativeTwinChains != 2 {
		t.Fatalf("artifacts.NegativeTwinChains = %d, want 2", artifacts.NegativeTwinChains)
	}
	reportMD, err := os.ReadFile(artifacts.ReportMarkdownPath)
	if err != nil {
		t.Fatalf("read hard-pack markdown: %v", err)
	}
	reportText := string(reportMD)
	if !strings.Contains(reportText, "Gold-First Phase2 Hard Pack Report") {
		t.Fatalf("expected hard-pack markdown title, got:\n%s", reportText)
	}
	if !strings.Contains(reportText, "GF-MF-D") || !strings.Contains(reportText, "GF-MF-E") {
		t.Fatalf("expected hard-pack markdown to mention GF-MF-D/E, got:\n%s", reportText)
	}
	if strings.Contains(reportText, "GMFH01") {
		t.Fatalf("hard-pack markdown unexpectedly included other cases_file rows:\n%s", reportText)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write test file %s: %v", path, err)
	}
}
