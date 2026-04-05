package researchdb

import (
	"context"
	"database/sql"
	"fmt"
)

func normalizeResearchResultsSchema(ctx context.Context, db *sql.DB) error {
	for _, migration := range []struct {
		table  string
		column string
		ddl    string
	}{
		{"research_result_summaries", "experiment_name", `ALTER TABLE research_result_summaries ADD COLUMN experiment_name TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "experiment_id", `ALTER TABLE research_result_summaries ADD COLUMN experiment_id TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "sampling_surface", `ALTER TABLE research_result_summaries ADD COLUMN sampling_surface TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "requested_sampling_profile", `ALTER TABLE research_result_summaries ADD COLUMN requested_sampling_profile TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "effective_sampling_profile", `ALTER TABLE research_result_summaries ADD COLUMN effective_sampling_profile TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "requested_sampling_json", `ALTER TABLE research_result_summaries ADD COLUMN requested_sampling_json TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "effective_sampling_json", `ALTER TABLE research_result_summaries ADD COLUMN effective_sampling_json TEXT NOT NULL DEFAULT ''`},
		{"research_result_summaries", "unsupported_sampling_json", `ALTER TABLE research_result_summaries ADD COLUMN unsupported_sampling_json TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "project_folder", `ALTER TABLE research_result_rows ADD COLUMN project_folder TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "experiment_name", `ALTER TABLE research_result_rows ADD COLUMN experiment_name TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "experiment_id", `ALTER TABLE research_result_rows ADD COLUMN experiment_id TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "sampling_surface", `ALTER TABLE research_result_rows ADD COLUMN sampling_surface TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "requested_sampling_profile", `ALTER TABLE research_result_rows ADD COLUMN requested_sampling_profile TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "effective_sampling_profile", `ALTER TABLE research_result_rows ADD COLUMN effective_sampling_profile TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "requested_sampling_json", `ALTER TABLE research_result_rows ADD COLUMN requested_sampling_json TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "effective_sampling_json", `ALTER TABLE research_result_rows ADD COLUMN effective_sampling_json TEXT NOT NULL DEFAULT ''`},
		{"research_result_rows", "unsupported_sampling_json", `ALTER TABLE research_result_rows ADD COLUMN unsupported_sampling_json TEXT NOT NULL DEFAULT ''`},
	} {
		if err := ensureColumnExists(ctx, db, migration.table, migration.column, migration.ddl); err != nil {
			return err
		}
	}
	if err := refreshResearchResultsView(ctx, db); err != nil {
		return err
	}
	return nil
}

func stageLegacyResearchResultsMigration(ctx context.Context, db *sql.DB) error {
	exists, objectType, err := lookupDBObject(ctx, db, "research_results")
	if err != nil {
		return err
	}
	if !exists || objectType != "table" {
		return nil
	}
	legacyExists, _, err := lookupDBObject(ctx, db, "research_results_legacy")
	if err != nil {
		return err
	}
	if legacyExists {
		return nil
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE research_results RENAME TO research_results_legacy`); err != nil {
		return fmt.Errorf("rename legacy research_results table: %w", err)
	}
	return nil
}

func finalizeLegacyResearchResultsMigration(ctx context.Context, db *sql.DB) error {
	exists, objectType, err := lookupDBObject(ctx, db, "research_results_legacy")
	if err != nil {
		return err
	}
	if !exists || objectType != "table" {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy research_results migration: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO research_result_summaries(
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, certificate_id, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, case_selector, theorems_file, theorems_path, result_path, raw_dir, hypothesis,
    cases_total, pass_count, false_refusal_count, false_accept_count, request_failure_count, schema_failure_count,
    parse_failure_count, kernel_failure_count, format_failure_count, avg_latency_ms, max_latency_ms, run_elapsed_seconds, category_count
)
SELECT
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, certificate_id, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, case_selector, theorems_file, theorems_path, result_path, raw_dir, hypothesis,
    cases_total, pass_count, false_refusal_count, false_accept_count, request_failure_count, schema_failure_count,
    parse_failure_count, kernel_failure_count, format_failure_count, avg_latency_ms, max_latency_ms, run_elapsed_seconds, category_count
FROM research_results_legacy
WHERE source_kind = 'summary'
`); err != nil {
		return fmt.Errorf("backfill research_result_summaries: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO research_result_rows(
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, theorem_id, case_id, category, expected_label, score_bucket,
    raw_output_kind, schema_status, parse_status, kernel_status, cli_exit_code, latency_ms, notes
)
SELECT
    id, research_id, result_file_id, theorem_file_id, type_id, llm_model_id, run_id, timestamp_utc,
    project_folder, prompt_version, theorem_pack_id, theorem_id, case_id, category, expected_label, score_bucket,
    raw_output_kind, schema_status, parse_status, kernel_status, cli_exit_code, latency_ms, notes
FROM research_results_legacy
WHERE source_kind = 'result_row'
`); err != nil {
		return fmt.Errorf("backfill research_result_rows: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DROP TABLE research_results_legacy`); err != nil {
		return fmt.Errorf("drop legacy research_results table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy research_results migration: %w", err)
	}
	return nil
}

func lookupDBObject(ctx context.Context, db *sql.DB, name string) (bool, string, error) {
	var objectType string
	err := db.QueryRowContext(ctx, `SELECT type FROM sqlite_master WHERE name = ? LIMIT 1`, name).Scan(&objectType)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("lookup sqlite object %q: %w", name, err)
	}
	return true, objectType, nil
}

func refreshResearchResultsView(ctx context.Context, db *sql.DB) error {
	exists, objectType, err := lookupDBObject(ctx, db, "research_results")
	if err != nil {
		return err
	}
	if !exists || objectType != "view" {
		return nil
	}
	if _, err := db.ExecContext(ctx, `DROP VIEW research_results`); err != nil {
		return fmt.Errorf("drop stale research_results view: %w", err)
	}
	return nil
}
