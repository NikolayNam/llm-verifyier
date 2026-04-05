package researchdb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOverviewReportsResearchAndGlobalCounts(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	project := "hilbert-ai-verification-benchmark-nd-v1"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	if err := os.MkdirAll(filepath.Join(artifactRoot, "theorems"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(artifactRoot, "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "research", "artifacts", "result_research", "direct"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "research", "artifacts", "result_research", "compositional"), 0o755); err != nil {
		t.Fatal(err)
	}

	theoremCSV := `theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment
pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import
`
	summaryCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,theorem_pack_id,theorems_file,case_selector,hypothesis,theorems_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
20260327T000001Z,2026-03-27T00:00:01Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,1.0.0,hilbert-ai-verification-benchmark-nd-v1,pilot_shared_20260327,theorems/pilot_shared_20260327.csv,all,test-hypothesis,theorems/pilot_shared_20260327.csv,result/result_20260327T000001Z.csv,raw/20260327T000001Z,1,1,0,0,0,0,0,0,0,12.5,13,7,1
`
	resultCSV := `run_id,timestamp_utc,model,llm_model,prompt_version,theorem_pack_id,theorem_id,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
20260327T000001Z,2026-03-27T00:00:02Z,compatible:gpt-oss:20b,gpt-oss:20b,hilbert-ai-verification-benchmark-v1.3,pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,certificate,ok,ok,ok,,pass,14,verified
`
	chainSummaryCSV := `run_id,chain_id,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
20260327T000001Z,CI01,linear_chain,1,r0,1,1,1,1,1,1,1,1,1,1,1,,,
`
	if err := os.WriteFile(filepath.Join(artifactRoot, "theorems", "pilot_shared_20260327.csv"), []byte(theoremCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "research", "artifacts", "result_research", "direct", project+"_result_summary_v2_rerun_direct_20260327.csv"), []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "result_20260327T000001Z.csv"), []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "result", "chain_result_20260327T000001Z.csv"), []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	if _, err := SyncProject(ctx, dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	}); err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
INSERT INTO research_runs(run_id, command, phase, status, started_at_utc, finished_at_utc)
VALUES ('phase1-demo', 'phase1 run', 'phase1', 'completed', '2026-04-01T12:00:00Z', '2026-04-01T12:10:00Z');
INSERT INTO research_jobs(run_id, job_key, spec_hash, command, phase, family, model, selector, status, result_path)
VALUES
  ('phase1-demo', 'job-1', 'spec-1', 'phase1 run', 'phase1', 'local-compatible', 'gpt-oss:20b', 'all', 'completed', ?),
  ('phase1-demo', 'job-2', 'spec-2', 'phase1 run', 'phase1', 'local-compatible', 'gpt-oss:20b', 'subset', 'failed', ''),
  ('phase1-demo', 'job-3', 'spec-3', 'phase1 run', 'phase1', 'local-compatible', 'gpt-oss:20b', 'tail', 'aborted', '');
INSERT INTO research_generated_packs(run_id, pack_key, project_folder, source_kind, output_path, created_at_utc)
VALUES ('phase1-demo', 'generated-pack-1', 'hilbert-ai-verification-lean4-worker-v1', 'enumerator', ?, '2026-04-01T12:11:00Z');
INSERT INTO research_exports(run_id, export_key, project_folder, benchmark_project_folder, source_pack_id, output_path, created_at_utc)
VALUES ('phase1-demo', 'export-1', 'hilbert-ai-verification-lean4-worker-v1', ?, 'generated-pack-1', ?, '2026-04-01T12:12:00Z');
`, filepath.Join(artifactRoot, "result", "result_20260327T000001Z.csv"), filepath.Join(workspace, "research", "artifacts", "hilbert-ai-verification-lean4-worker-v1", "theorems", "theorems.csv"), project, filepath.Join(artifactRoot, "theorems", "lean4-generated.csv")); err != nil {
		t.Fatalf("seed run/job/export rows error = %v", err)
	}

	overview, err := LoadOverview(ctx, dbPath, OverviewOptions{ProjectFolder: project})
	if err != nil {
		t.Fatalf("LoadOverview() error = %v", err)
	}
	if overview.Scope != project {
		t.Fatalf("overview.Scope = %q, want %q", overview.Scope, project)
	}
	if overview.TotalResearches != 1 {
		t.Fatalf("overview.TotalResearches = %d, want 1", overview.TotalResearches)
	}
	if overview.TotalGeneratedPacks != 1 || overview.TotalExports != 1 {
		t.Fatalf("overview generated/export counts = %d/%d, want 1/1", overview.TotalGeneratedPacks, overview.TotalExports)
	}
	if overview.GlobalRuns.Total != 1 || overview.GlobalJobs.Total != 3 || overview.GlobalJobs.Completed != 1 || overview.GlobalJobs.Failed != 1 || overview.GlobalJobs.Aborted != 1 {
		t.Fatalf("overview global run/job counts = %+v / %+v", overview.GlobalRuns, overview.GlobalJobs)
	}
	if len(overview.Researches) != 1 {
		t.Fatalf("len(overview.Researches) = %d, want 1", len(overview.Researches))
	}
	row := overview.Researches[0]
	if row.TheoremFiles != 1 || row.Theorems != 1 || row.ResultFiles != 3 || row.ChainSummaryFiles != 1 || row.SummaryRows != 1 || row.ResultRows != 1 || row.ChainSummaryRows != 1 {
		t.Fatalf("research row counts = %+v", row)
	}
	if overview.TotalChainSummaryRows != 1 {
		t.Fatalf("overview.TotalChainSummaryRows = %d, want 1", overview.TotalChainSummaryRows)
	}
	if len(overview.ArtifactGroups) < 2 {
		t.Fatalf("artifact groups = %+v, want direct and compositional entries", overview.ArtifactGroups)
	}
	foundDirect := false
	foundCompositional := false
	for _, group := range overview.ArtifactGroups {
		switch group.ArtifactGroup {
		case "direct":
			foundDirect = true
			if group.ResultFiles != 1 || group.SummaryRows != 1 || group.ResultRows != 0 || group.ChainSummaryRows != 0 {
				t.Fatalf("direct artifact group counts = %+v", group)
			}
		case "family":
			if group.ResultFiles != 1 || group.ResultRows != 1 {
				t.Fatalf("family artifact group counts = %+v", group)
			}
		case "compositional":
			foundCompositional = true
			if group.ResultFiles != 1 || group.ChainSummaryFiles != 1 || group.ChainSummaryRows != 1 {
				t.Fatalf("compositional artifact group counts = %+v", group)
			}
		}
	}
	if !foundDirect || !foundCompositional {
		t.Fatalf("artifact groups = %+v, want direct and compositional entries", overview.ArtifactGroups)
	}
}

func TestListRunsFiltersAndOrders(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "research.sqlite")
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
INSERT INTO research_runs(run_id, command, phase, status, started_at_utc, finished_at_utc)
VALUES
  ('run-older', 'phase1 run', 'phase1', 'completed', '2026-04-01T10:00:00Z', '2026-04-01T10:05:00Z'),
  ('run-newer', 'phase3 run', 'phase3', 'partial_failed', '2026-04-01T11:00:00Z', '2026-04-01T11:15:00Z');
INSERT INTO research_jobs(run_id, job_key, spec_hash, status)
VALUES
  ('run-newer', 'a', 'a', 'completed'),
  ('run-newer', 'b', 'b', 'failed'),
  ('run-newer', 'd', 'd', 'aborted'),
  ('run-older', 'c', 'c', 'completed');
`); err != nil {
		t.Fatalf("seed runs/jobs error = %v", err)
	}

	allRuns, err := ListRuns(ctx, dbPath, ListRunsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListRuns(all) error = %v", err)
	}
	if len(allRuns) != 2 {
		t.Fatalf("len(ListRuns(all)) = %d, want 2", len(allRuns))
	}
	if allRuns[0].RunID != "run-newer" {
		t.Fatalf("latest run_id = %q, want run-newer", allRuns[0].RunID)
	}
	if allRuns[0].CompletedJobs != 1 || allRuns[0].FailedJobs != 1 || allRuns[0].AbortedJobs != 1 {
		t.Fatalf("run-newer job counts = %+v", allRuns[0])
	}

	filtered, err := ListRuns(ctx, dbPath, ListRunsOptions{Phase: "phase1", Limit: 10})
	if err != nil {
		t.Fatalf("ListRuns(filtered) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].RunID != "run-older" {
		t.Fatalf("filtered runs = %+v, want only run-older", filtered)
	}
}
