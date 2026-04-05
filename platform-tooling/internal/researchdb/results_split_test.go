package researchdb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMigratesLegacyResearchResultsTable(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "research.sqlite")

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer legacyDB.Close()

	if _, err := legacyDB.ExecContext(ctx, `
PRAGMA foreign_keys = ON;
CREATE TABLE research_types (
    id INTEGER PRIMARY KEY,
    type_code TEXT NOT NULL UNIQUE,
    domain TEXT NOT NULL,
    description TEXT NOT NULL
);
CREATE TABLE research_researches (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    artifact_root TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT ''
);
CREATE TABLE research_llm_models (
    id INTEGER PRIMARY KEY,
    llm_model TEXT NOT NULL UNIQUE,
    model_ref TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT ''
);
CREATE TABLE research_certificates (
    id INTEGER PRIMARY KEY,
    certificate_version TEXT NOT NULL UNIQUE
);
CREATE TABLE research_theorem_files (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL,
    type_id INTEGER,
    theorem_pack_id TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    absolute_path TEXT NOT NULL,
    is_historical INTEGER NOT NULL DEFAULT 0,
    imported_at_utc TEXT NOT NULL
);
CREATE TABLE research_result_files (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL,
    type_id INTEGER,
    artifact_group TEXT NOT NULL,
    file_kind TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    absolute_path TEXT NOT NULL,
    imported_at_utc TEXT NOT NULL
);
CREATE TABLE research_results (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL,
    result_file_id INTEGER NOT NULL,
    theorem_file_id INTEGER,
    type_id INTEGER,
    llm_model_id INTEGER,
    certificate_id INTEGER,
    source_kind TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    timestamp_utc TEXT NOT NULL DEFAULT '',
    project_folder TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    theorem_pack_id TEXT NOT NULL DEFAULT '',
    theorem_id TEXT NOT NULL DEFAULT '',
    case_id TEXT NOT NULL DEFAULT '',
    case_selector TEXT NOT NULL DEFAULT '',
    theorems_file TEXT NOT NULL DEFAULT '',
    theorems_path TEXT NOT NULL DEFAULT '',
    result_path TEXT NOT NULL DEFAULT '',
    raw_dir TEXT NOT NULL DEFAULT '',
    hypothesis TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    expected_label TEXT NOT NULL DEFAULT '',
    score_bucket TEXT NOT NULL DEFAULT '',
    raw_output_kind TEXT NOT NULL DEFAULT '',
    schema_status TEXT NOT NULL DEFAULT '',
    parse_status TEXT NOT NULL DEFAULT '',
    kernel_status TEXT NOT NULL DEFAULT '',
    cli_exit_code TEXT NOT NULL DEFAULT '',
    latency_ms REAL NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    cases_total INTEGER NOT NULL DEFAULT 0,
    pass_count INTEGER NOT NULL DEFAULT 0,
    false_refusal_count INTEGER NOT NULL DEFAULT 0,
    false_accept_count INTEGER NOT NULL DEFAULT 0,
    request_failure_count INTEGER NOT NULL DEFAULT 0,
    schema_failure_count INTEGER NOT NULL DEFAULT 0,
    parse_failure_count INTEGER NOT NULL DEFAULT 0,
    kernel_failure_count INTEGER NOT NULL DEFAULT 0,
    format_failure_count INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms REAL NOT NULL DEFAULT 0,
    max_latency_ms REAL NOT NULL DEFAULT 0,
    run_elapsed_seconds REAL NOT NULL DEFAULT 0,
    category_count INTEGER NOT NULL DEFAULT 0
);
INSERT INTO research_researches(id, slug, title, artifact_root) VALUES (1, 'demo', 'Demo', 'research/artifacts/demo');
INSERT INTO research_types(id, type_code, domain, description) VALUES (1, 'summary', 'result_row', 'summary'), (2, 'result_row', 'result_row', 'result row');
INSERT INTO research_llm_models(id, llm_model, model_ref, provider, display_name) VALUES (1, 'demo-model', 'compatible:demo-model', 'compatible', 'Demo Model');
INSERT INTO research_certificates(id, certificate_version) VALUES (1, '1.0.0');
INSERT INTO research_theorem_files(id, research_id, type_id, theorem_pack_id, relative_path, absolute_path, imported_at_utc) VALUES
    (1, 1, NULL, 'demo_pack', 'research/artifacts/demo/theorems/demo_pack.csv', '/tmp/demo_pack.csv', '2026-04-01T00:00:00Z');
INSERT INTO research_result_files(id, research_id, type_id, artifact_group, file_kind, relative_path, absolute_path, imported_at_utc) VALUES
    (1, 1, NULL, 'direct', 'summary', 'research/artifacts/result_research/direct/demo_summary.csv', '/tmp/demo_summary.csv', '2026-04-01T00:00:00Z'),
    (2, 1, NULL, 'family', 'result_rows', 'research/artifacts/demo/result/result_demo.csv', '/tmp/result_demo.csv', '2026-04-01T00:00:00Z');
INSERT INTO research_results(
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, certificate_id, source_kind, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, case_selector, theorems_file, theorems_path, result_path, raw_dir, hypothesis,
    cases_total, pass_count, false_refusal_count, false_accept_count, request_failure_count, schema_failure_count, parse_failure_count,
    kernel_failure_count, format_failure_count, avg_latency_ms, max_latency_ms, run_elapsed_seconds, category_count
) VALUES (
    10, 1, 1, 1, 1, 1, 1, 'summary', 'run-summary', '2026-04-01T00:00:00Z',
    'demo', 'prompt-v1', 'demo_pack', 'all', 'theorems/demo_pack.csv', 'theorems/demo_pack.csv', 'result/result_demo.csv', 'raw/run-summary', 'hypothesis',
    1, 1, 0, 0, 0, 0, 0, 0, 0, 12.5, 13.0, 8.0, 1
);
INSERT INTO research_results(
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, source_kind, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, theorem_id, case_id, category, expected_label, score_bucket,
    raw_output_kind, schema_status, parse_status, kernel_status, cli_exit_code, latency_ms, notes
) VALUES (
    11, 1, 2, 1, 2, 1, 'result_row', 'run-row', '2026-04-01T00:00:01Z',
    'demo', 'prompt-v1', 'demo_pack', 'THM01', 'THM01', 'assumption_import', 'entailed', 'pass',
    'certificate', 'ok', 'ok', 'ok', '', 14.0, 'verified'
);
`); err != nil {
		t.Fatalf("seed legacy schema error = %v", err)
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_results WHERE source_kind = 'summary'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_results WHERE source_kind = 'result_row'", 1)

	var legacyCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'research_results_legacy'`).Scan(&legacyCount); err != nil {
		t.Fatalf("lookup research_results_legacy error = %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("legacy table count = %d, want 0", legacyCount)
	}
}

func TestOpenAddsSamplingProvenanceColumnsToExistingSplitTables(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "research.sqlite")

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer legacyDB.Close()

	if _, err := legacyDB.ExecContext(ctx, `
PRAGMA foreign_keys = ON;
CREATE TABLE research_result_summaries (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL,
    result_file_id INTEGER NOT NULL,
    theorem_file_id INTEGER,
    type_id INTEGER,
    llm_model_id INTEGER,
    certificate_id INTEGER,
    run_id TEXT NOT NULL DEFAULT '',
    timestamp_utc TEXT NOT NULL DEFAULT '',
    project_folder TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    theorem_pack_id TEXT NOT NULL DEFAULT '',
    case_selector TEXT NOT NULL DEFAULT '',
    theorems_file TEXT NOT NULL DEFAULT '',
    theorems_path TEXT NOT NULL DEFAULT '',
    result_path TEXT NOT NULL DEFAULT '',
    raw_dir TEXT NOT NULL DEFAULT '',
    hypothesis TEXT NOT NULL DEFAULT '',
    cases_total INTEGER NOT NULL DEFAULT 0,
    pass_count INTEGER NOT NULL DEFAULT 0,
    false_refusal_count INTEGER NOT NULL DEFAULT 0,
    false_accept_count INTEGER NOT NULL DEFAULT 0,
    request_failure_count INTEGER NOT NULL DEFAULT 0,
    schema_failure_count INTEGER NOT NULL DEFAULT 0,
    parse_failure_count INTEGER NOT NULL DEFAULT 0,
    kernel_failure_count INTEGER NOT NULL DEFAULT 0,
    format_failure_count INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms REAL NOT NULL DEFAULT 0,
    max_latency_ms REAL NOT NULL DEFAULT 0,
    run_elapsed_seconds REAL NOT NULL DEFAULT 0,
    category_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE research_result_rows (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL,
    result_file_id INTEGER NOT NULL,
    theorem_file_id INTEGER,
    type_id INTEGER,
    llm_model_id INTEGER,
    run_id TEXT NOT NULL DEFAULT '',
    timestamp_utc TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    theorem_pack_id TEXT NOT NULL DEFAULT '',
    theorem_id TEXT NOT NULL DEFAULT '',
    case_id TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    expected_label TEXT NOT NULL DEFAULT '',
    score_bucket TEXT NOT NULL DEFAULT '',
    raw_output_kind TEXT NOT NULL DEFAULT '',
    schema_status TEXT NOT NULL DEFAULT '',
    parse_status TEXT NOT NULL DEFAULT '',
    kernel_status TEXT NOT NULL DEFAULT '',
    cli_exit_code TEXT NOT NULL DEFAULT '',
    latency_ms REAL NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT ''
);
CREATE VIEW research_results AS
SELECT
    id,
    research_id,
    result_file_id,
    theorem_file_id,
    type_id,
    llm_model_id,
    certificate_id,
    'summary' AS source_kind,
    run_id,
    timestamp_utc,
    project_folder,
    prompt_version,
    theorem_pack_id,
    '' AS theorem_id,
    '' AS case_id,
    case_selector,
    theorems_file,
    theorems_path,
    result_path,
    raw_dir,
    hypothesis,
    '' AS category,
    '' AS expected_label,
    '' AS score_bucket,
    '' AS raw_output_kind,
    '' AS schema_status,
    '' AS parse_status,
    '' AS kernel_status,
    '' AS cli_exit_code,
    0.0 AS latency_ms,
    '' AS notes,
    cases_total,
    pass_count,
    false_refusal_count,
    false_accept_count,
    request_failure_count,
    schema_failure_count,
    parse_failure_count,
    kernel_failure_count,
    format_failure_count,
    avg_latency_ms,
    max_latency_ms,
    run_elapsed_seconds,
    category_count
FROM research_result_summaries;
`); err != nil {
		t.Fatalf("seed split schema error = %v", err)
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	for _, tc := range []struct {
		table  string
		column string
	}{
		{table: "research_result_summaries", column: "experiment_id"},
		{table: "research_result_summaries", column: "requested_sampling_profile"},
		{table: "research_result_rows", column: "project_folder"},
		{table: "research_result_rows", column: "experiment_id"},
		{table: "research_result_rows", column: "unsupported_sampling_json"},
	} {
		assertColumnExists(t, db, tc.table, tc.column)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'view' AND name = 'research_results'`).Scan(&count); err != nil {
		t.Fatalf("lookup research_results view error = %v", err)
	}
	if count != 1 {
		t.Fatalf("research_results view count = %d, want 1", count)
	}
	var experimentID string
	if err := db.QueryRowContext(ctx, `SELECT experiment_id FROM research_results LIMIT 1`).Scan(&experimentID); err != nil && err != sql.ErrNoRows {
		t.Fatalf("select experiment_id from research_results error = %v", err)
	}
}

func TestPruneProjectRemovesMissingIndexedArtifacts(t *testing.T) {
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

	theoremPath := filepath.Join(artifactRoot, "theorems", "pilot_shared_20260327.csv")
	summaryPath := filepath.Join(workspace, "research", "artifacts", "result_research", "direct", project+"_result_summary_v2_rerun_direct_20260327.csv")
	resultPath := filepath.Join(artifactRoot, "result", "result_20260327T000001Z.csv")
	chainSummaryPath := filepath.Join(artifactRoot, "result", "chain_result_20260327T000001Z.csv")

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
	if err := os.WriteFile(theoremPath, []byte(theoremCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryPath, []byte(summaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, []byte(resultCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chainSummaryPath, []byte(chainSummaryCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")
	if _, err := SyncProject(ctx, dbPath, SyncOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
	}); err != nil {
		t.Fatalf("SyncProject() error = %v", err)
	}

	if err := os.Remove(theoremPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(summaryPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(resultPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(chainSummaryPath); err != nil {
		t.Fatal(err)
	}

	dryRun, err := PruneProject(ctx, dbPath, PruneOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
		Apply:         false,
	})
	if err != nil {
		t.Fatalf("PruneProject(dry-run) error = %v", err)
	}
	if !dryRun.DryRun {
		t.Fatalf("DryRun = false, want true")
	}
	if dryRun.MissingTheoremFiles != 1 || dryRun.MissingResultFiles != 3 {
		t.Fatalf("dry-run missing files = theorem:%d result:%d, want theorem:1 result:3", dryRun.MissingTheoremFiles, dryRun.MissingResultFiles)
	}
	if dryRun.DeletedTheoremRows != 1 || dryRun.DeletedSummaryRows != 1 || dryRun.DeletedResultRows != 1 || dryRun.DeletedChainSummaryRows != 1 {
		t.Fatalf("dry-run dependent rows = theorem:%d summary:%d result:%d chain:%d, want 1/1/1/1", dryRun.DeletedTheoremRows, dryRun.DeletedSummaryRows, dryRun.DeletedResultRows, dryRun.DeletedChainSummaryRows)
	}

	applied, err := PruneProject(ctx, dbPath, PruneOptions{
		WorkspaceRoot: workspace,
		ProjectFolder: project,
		Apply:         true,
	})
	if err != nil {
		t.Fatalf("PruneProject(apply) error = %v", err)
	}
	if applied.DryRun {
		t.Fatalf("DryRun = true after apply, want false")
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	assertCount(t, db, "SELECT COUNT(*) FROM research_theorem_files", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM research_theorems", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_files", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_summaries", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM research_result_rows", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM research_chain_summary_rows", 0)
}

func assertColumnExists(t *testing.T, db *sql.DB, tableName, columnName string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) error = %v", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			pk           int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info(%s) error = %v", tableName, err)
		}
		if name == columnName {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s) error = %v", tableName, err)
	}
	t.Fatalf("column %s.%s not found", tableName, columnName)
}
