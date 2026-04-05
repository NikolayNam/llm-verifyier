package researchdb

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenInitializesControlPlaneTables(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "research.sqlite"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	tables := []string{
		"research_result_summaries",
		"research_result_rows",
		"research_chain_summary_rows",
		"research_runs",
		"research_run_configs",
		"research_jobs",
		"research_run_artifacts",
		"research_generated_packs",
		"research_exports",
	}
	for _, table := range tables {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatalf("QueryRowContext(%q) error = %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %q count = %d, want 1", table, count)
		}
	}
	var viewCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'view' AND name = 'research_results'`).Scan(&viewCount); err != nil {
		t.Fatalf("QueryRowContext(research_results view) error = %v", err)
	}
	if viewCount != 1 {
		t.Fatalf("view %q count = %d, want 1", "research_results", viewCount)
	}
	for _, objectName := range []string{
		"research_hypotheses",
		"idx_research_hypotheses_set_order",
		"research_results_legacy",
	} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE name = ?`, objectName).Scan(&count); err != nil {
			t.Fatalf("QueryRowContext(%q) error = %v", objectName, err)
		}
		if count != 0 {
			t.Fatalf("sqlite object %q count = %d, want 0", objectName, count)
		}
	}
}
