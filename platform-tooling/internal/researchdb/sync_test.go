package researchdb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncProjectImportsTheoremsSummariesAndResults(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-nd-v1"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "theorems", "historical"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "research", "artifacts", "result_research", "direct"), 0o755); err != nil {
		t.Fatal(err)
	}

	theoremCSV := `theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment
pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "theorems", "pilot_shared_20260327.csv"), []byte(theoremCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "theorems", "historical", "theorems_pilot_shared_20260327.csv"), []byte(theoremCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "theorems", "historical", "theorems_stale_mismatch.csv"), []byte(theoremCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	staleReservedCSV := `theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment
theorems,NDI99,NDI99,negative_refusal,not_entailed,easy,[],Q,implicational-prop-v1,Stale reserved-pack snapshot
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "theorems", "historical", "theorems_theorems.csv"), []byte(staleReservedCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,theorem_pack_id,theorems_file,case_selector,hypothesis,theorems_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
20260327T000001Z,2026-03-27T00:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-nd-v1,pilot_shared_20260327,theorems/pilot_shared_20260327.csv,all,test-hypothesis,theorems/pilot_shared_20260327.csv,result/result_20260327T000001Z.csv,raw/20260327T000001Z,1,1,0,0,0,0,0,0,0,12.5,13,7,1
`
	if err := os.WriteFile(filepath.Join(workspace, "research", "artifacts", "result_research", "direct", project+"_result_summary_v2_rerun_direct_20260327.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,theorem_pack_id,theorem_id,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
20260327T000001Z,2026-03-27T00:00:02Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,certificate,ok,ok,ok,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_20260327T000001Z.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	result, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if result.TheoremFiles != 2 {
		t.Fatalf("TheoremFiles = %d, want 2", result.TheoremFiles)
	}
	if result.ResultFiles != 2 {
		t.Fatalf("ResultFiles = %d, want 2", result.ResultFiles)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_theorems", 2)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_results WHERE source_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_results WHERE source_kind = 'result_row'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'direct'", 1)
}

func TestSyncProjectDedupesDuplicateSummaryAndResultRows(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "research", "artifacts", "result_research", "direct"), 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,category,label,difficulty,assumptions_json,goal,comment
V2E01,assumption_import,entailed,easy,"[""R -> S""]","R -> S",Import a single assumption directly
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryHeader := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count`
	summaryRow := `20260330_phase1_direct_gpt-oss-20b_r01,2026-03-30T08:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases.csv,all,test-hypothesis,cases.csv,result/result_20260330_phase1_direct_gpt-oss-20b_r01.csv,raw/20260330_phase1_direct_gpt-oss-20b_r01,1,1,0,0,0,0,0,0,0,10.0,10,1,1`
	summaryCSV := summaryHeader + "\n" + summaryRow + "\n" + summaryRow + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "research", "artifacts", "result_research", "direct", project+"_result_summary_phase1_direct_gpt-oss-20b_20260330.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultHeader := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes`
	resultRow := `20260330_phase1_direct_gpt-oss-20b_r01,2026-03-30T08:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,V2E01,assumption_import,entailed,certificate,pass,pass,accept,,pass,14,verified`
	resultCSV := resultHeader + "\n" + resultRow + "\n" + resultRow + "\n"
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_20260330_phase1_direct_gpt-oss-20b_r01.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	if _, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	}); err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 1)
}

func TestSyncProjectDetectsDirectGroupFromByDateSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	resultDir := filepath.Join(artifactRoot, "result", "by-date", "20260330", "compatible", "gpt-oss-20b", "direct")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "by-date", "20260330", "compatible", "gpt-oss-20b", "direct")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,category,label,difficulty,assumptions_json,goal,comment
V2E01,assumption_import,entailed,easy,"[""R -> S""]","R -> S",Import a single assumption directly
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
20260330_phase1_direct_compatible_gpt-oss-20b_r01,2026-03-30T08:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases.csv,all,test-hypothesis,cases.csv,result/by-date/20260330/compatible/gpt-oss-20b/direct/result_20260330_phase1_direct_compatible_gpt-oss-20b_r01.csv,raw/by-date/20260330/compatible/gpt-oss-20b/direct,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, project+"_result_summary_phase1_direct.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
20260330_phase1_direct_compatible_gpt-oss-20b_r01,2026-03-30T08:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,V2E01,assumption_import,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(resultDir, "result_20260330_phase1_direct_compatible_gpt-oss-20b_r01.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	if _, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	}); err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'direct'", 1)
}

func TestSyncProjectDetectsCompositionalGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,stage_id,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
CI01-S1,CI01,s1,compositional_assumption_import,entailed,easy,linear_chain,1,none,r0,"[]","[]","A -> B",prove,Local lemma 1
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
compositional-20260403,2026-04-03T08:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-assumption-import-phase1.csv,all,test-hypothesis,cases/compositional-assumption-import-phase1.csv,result/result_compositional-20260403.csv,raw/compositional-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-assumption-import_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
compositional-20260403,2026-04-03T08:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,CI01-S1,compositional_assumption_import,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_compositional-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
compositional-20260403,CI01,linear_chain,1,r0,1,1,1,0,1,0,1,1,0,1,0,s3m,schema_failure,model import degraded
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_compositional-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectDetectsCompositionalDepthLadderGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional-depth-ladder")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,chain_protocol,stage_id,stage_order,stage_role,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,trusted_reuse,import_stage_ids_json,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
CDL02-S1,CDL02,depth-ladder-v1,s1,1,seed_lemma,compositional_depth_ladder,entailed,hard,depth3_linear,3,none,r0,true,"[]","[]","[]","P -> Q",prove,Depth ladder seed
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
depth-ladder-20260403,2026-04-03T11:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-depth-ladder-phase1.csv,all,test-hypothesis,cases/compositional-depth-ladder-phase1.csv,result/result_depth-ladder-20260403.csv,raw/depth-ladder-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-depth-ladder_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
depth-ladder-20260403,2026-04-03T11:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,CDL02-S1,compositional_depth_ladder,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_depth-ladder-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
depth-ladder-20260403,CDL02,depth-ladder-v1,depth3_linear,3,r0,1,1,1,1,1,1,1,1,1,1,1,,,all good
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_depth-ladder-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_depth_ladder' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_depth_ladder' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectDetectsCompositionalBranchingGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional-branching")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,chain_protocol,stage_id,stage_order,stage_role,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,trusted_reuse,import_stage_ids_json,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
CBR01-S1,CBR01,branching-v1,s1,1,branch_left,compositional_branching,entailed,hard,fan_in_merge,2,none,r0,true,"[]","[]","[]","A -> B",prove,Branching seed
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
branching-20260403,2026-04-03T12:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-branching-phase1.csv,all,test-hypothesis,cases/compositional-branching-phase1.csv,result/result_branching-20260403.csv,raw/branching-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-branching_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
branching-20260403,2026-04-03T12:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,CBR01-S1,compositional_branching,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_branching-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
branching-20260403,CBR01,branching-v1,fan_in_merge,2,r0,1,1,1,1,1,1,1,1,1,1,1,,,all good
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_branching-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_branching' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_branching' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectDetectsCompositionalMixedFamilyReuseGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional-mixed-family-reuse")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,chain_protocol,stage_id,stage_order,stage_role,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,trusted_reuse,import_stage_ids_json,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
MFR01-S1,MFR01,mixed-family-v1,s1,1,source_family_left,compositional_mixed_family_reuse,entailed,medium,assumption_import_source,2,none,r0,true,"[]","[]","[]","A -> B",prove,Mixed-family seed
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
mixed-family-20260403,2026-04-03T13:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-mixed-family-reuse-phase1.csv,all,test-hypothesis,cases/compositional-mixed-family-reuse-phase1.csv,result/result_mixed-family-20260403.csv,raw/mixed-family-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-mixed-family-reuse_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
mixed-family-20260403,2026-04-03T13:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,MFR01-S1,compositional_mixed_family_reuse,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_mixed-family-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
mixed-family-20260403,MFR01,mixed-family-v1,mixed_family_reuse,2,r0,1,1,1,1,1,1,1,1,1,1,1,,,all good
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_mixed-family-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_reuse' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_reuse' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectDetectsCompositionalMixedFamilyGoldFirstGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional-mixed-family-gold-first")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,chain_protocol,stage_id,stage_order,stage_role,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,trusted_reuse,import_stage_ids_json,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
MFG01-FG,MFG01,mixed-family-gold-first-v1,fg,1,gold_import_stage,compositional_mixed_family_gold_first,entailed,medium,assumption_import_x_direct_axiom_instance_gold_first,1,gold,r0,false,"[]","[]","[""A -> B"",""B -> C""]","A -> C",prove,Gold-first final composition
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
mixed-family-gold-first-20260403,2026-04-03T15:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-mixed-family-gold-first-phase1.csv,all,test-hypothesis,cases/compositional-mixed-family-gold-first-phase1.csv,result/result_mixed-family-gold-first-20260403.csv,raw/mixed-family-gold-first-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-mixed-family-gold-first_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
mixed-family-gold-first-20260403,2026-04-03T15:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,MFG01-FG,compositional_mixed_family_gold_first,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_mixed-family-gold-first-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
mixed-family-gold-first-20260403,MFG01,mixed-family-gold-first-v1,assumption_import_x_direct_axiom_instance_gold_first,1,r0,false,false,true,false,false,true,false,true,false,false,false,,,gold-first final only
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_mixed-family-gold-first-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_gold_first' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_gold_first' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectDetectsCompositionalMixedFamilyGoldFirstHardGroupFromSharedSummaryPath(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	summaryDir := filepath.Join(workspace, "research", "artifacts", "result_research", "compositional-mixed-family-gold-first-hard")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,chain_id,chain_protocol,stage_id,stage_order,stage_role,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,source_family,target_family,transition_group,import_arity,reuse_shape,bridge_depth,symbol_overlap,lemma_surface_style,negative_twin_hardness,notes,trusted_reuse,import_stage_ids_json,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
GMFH01-FG,GMFH01,mixed-family-gold-first-hard-v1,fg,1,gold_import_stage,compositional_mixed_family_gold_first_hard,entailed,hard,assumption_import_to_direct_axiom_instance_gold_first_hard,1,gold,gfmfh-a-01,assumption_import,direct_axiom_instance,GF-MFH-A,1,one_import_long_bridge,2,medium,atomic_implication,hard,"hard pack",false,"[]","[""A"",""A -> B"",""B -> C""]","[""C -> D""]",D,prove,Hard gold-first final composition
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
mixed-family-gold-first-hard-20260403,2026-04-03T16:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases/compositional-mixed-family-gold-first-hard.csv,all,test-hypothesis,cases/compositional-mixed-family-gold-first-hard.csv,result/result_mixed-family-gold-first-hard-20260403.csv,raw/mixed-family-gold-first-hard-20260403,1,1,0,0,0,0,0,0,0,10.0,10,1,1
`
	if err := os.WriteFile(filepath.Join(summaryDir, "compositional-mixed-family-gold-first-hard_20260403.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
mixed-family-gold-first-hard-20260403,2026-04-03T16:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,direct,GMFH01-FG,compositional_mixed_family_gold_first_hard,entailed,certificate,pass,pass,accept,,pass,14,verified
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_mixed-family-gold-first-hard-20260403.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	chainSummaryCSV := `run_id,chain_id,chain_protocol,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
mixed-family-gold-first-hard-20260403,GMFH01,mixed-family-gold-first-hard-v1,assumption_import_to_direct_axiom_instance_gold_first_hard,1,gfmfh-a-01,false,false,true,false,false,true,false,true,false,false,false,,,hard final only
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_mixed-family-gold-first-hard-20260403.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	syncResult, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}
	if syncResult.ChainSummaryFiles != 1 || syncResult.ChainRowsImported != 1 {
		t.Fatalf("SyncProject() chain summary counts = %+v, want files=1 rows=1", syncResult)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_gold_first_hard' AND file_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files WHERE artifact_group = 'compositional_mixed_family_gold_first_hard' AND file_kind = 'chain_summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 1)
}

func TestSyncProjectKeepsDistinctExperimentRows(t *testing.T) {
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-v2-held-out"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "research", "artifacts", "result_research", "waves"), 0o755); err != nil {
		t.Fatal(err)
	}

	casesCSV := `case_id,category,label,difficulty,assumptions_json,goal,comment
V2E01,assumption_import,entailed,easy,"[""R -> S""]","R -> S",Import a single assumption directly
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "cases.csv"), []byte(casesCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	summaryCSV := strings.Join([]string{
		"run_id,timestamp_utc,model,llm_model,experiment_name,experiment_id,sampling_surface,requested_sampling_profile,effective_sampling_profile,requested_sampling_json,effective_sampling_json,unsupported_sampling_json,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count",
		`phase1-sampling-r01,2026-04-02T07:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,formal-mode,formal-mode-temp0-seed42-topp1-r01-20260402,compatible_openai_chat_completions,temp0_seed42_topp1,temp0_seed42_topp1,"{""temperature"":0,""seed"":42,""top_p"":1}","{""temperature"":0,""seed"":42,""top_p"":1}","",hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases.csv,all,test-hypothesis,cases.csv,result/result_phase1-sampling-r01.csv,raw/phase1-sampling-r01,1,1,0,0,0,0,0,0,0,10.0,10,1,1`,
		`phase1-sampling-r01,2026-04-02T07:00:00Z,compatible:gpt-oss:20b,gpt-oss:20b,balanced-mode,balanced-mode-temp0-seed42-topp09-r01-20260402,compatible_openai_chat_completions,temp0_seed42_topp0.9,temp0_seed42_topp0.9,"{""temperature"":0,""seed"":42,""top_p"":0.9}","{""temperature"":0,""seed"":42,""top_p"":0.9}","",hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-v2-held-out,direct,cases.csv,all,test-hypothesis,cases.csv,result/result_phase1-sampling-r01.csv,raw/phase1-sampling-r01,1,1,0,0,0,0,0,0,0,10.0,10,1,1`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(workspace, "research", "artifacts", "result_research", "waves", project+"_phase1_sampling_compare.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	resultCSV := strings.Join([]string{
		"run_id,timestamp_utc,model,llm_model,experiment_name,experiment_id,sampling_surface,requested_sampling_profile,effective_sampling_profile,requested_sampling_json,effective_sampling_json,unsupported_sampling_json,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes",
		`phase1-sampling-r01,2026-04-02T07:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,formal-mode,formal-mode-temp0-seed42-topp1-r01-20260402,compatible_openai_chat_completions,temp0_seed42_topp1,temp0_seed42_topp1,"{""temperature"":0,""seed"":42,""top_p"":1}","{""temperature"":0,""seed"":42,""top_p"":1}","",hilbert-ai-verification-benchmark-v1.3,direct,V2E01,assumption_import,entailed,certificate,pass,pass,accept,,pass,14,verified`,
		`phase1-sampling-r01,2026-04-02T07:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,balanced-mode,balanced-mode-temp0-seed42-topp09-r01-20260402,compatible_openai_chat_completions,temp0_seed42_topp0.9,temp0_seed42_topp0.9,"{""temperature"":0,""seed"":42,""top_p"":0.9}","{""temperature"":0,""seed"":42,""top_p"":0.9}","",hilbert-ai-verification-benchmark-v1.3,direct,V2E01,assumption_import,entailed,certificate,pass,pass,accept,,pass,14,verified`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_phase1-sampling-r01.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	if _, err := SyncProject(context.Background(), dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	}); err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 2)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 2)

	var got string
	if err := db.QueryRow(`SELECT GROUP_CONCAT(experiment_id, ',') FROM (SELECT experiment_id FROM research_result_summaries ORDER BY experiment_id)`).Scan(&got); err != nil {
		t.Fatalf("summary experiment ids query error = %v", err)
	}
	if got != "balanced-mode-temp0-seed42-topp09-r01-20260402,formal-mode-temp0-seed42-topp1-r01-20260402" {
		t.Fatalf("summary experiment ids = %q", got)
	}
	if err := db.QueryRow(`SELECT GROUP_CONCAT(requested_sampling_profile, ',') FROM (SELECT requested_sampling_profile FROM research_result_rows ORDER BY requested_sampling_profile)`).Scan(&got); err != nil {
		t.Fatalf("result requested_sampling_profile query error = %v", err)
	}
	if got != "temp0_seed42_topp0.9,temp0_seed42_topp1" {
		t.Fatalf("result requested_sampling_profile = %q", got)
	}
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != want {
		t.Fatalf("%s => %d, want %d", query, got, want)
	}
}
