package researchdb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

func Open(ctx context.Context, path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("research db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create research db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite research db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite research db: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite busy_timeout: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite journal_mode: %w", err)
	}
	if err := stageLegacyResearchResultsMigration(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := normalizeHypothesisStorageSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := normalizeResearchResultsSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize sqlite research db schema: %w", err)
	}
	if err := finalizeLegacyResearchResultsMigration(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := seedTypes(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func seedTypes(ctx context.Context, db *sql.DB) error {
	type seedRow struct {
		Code        string
		Domain      string
		Description string
	}
	rows := []seedRow{
		{Code: "canonical_pack", Domain: "theorem_file", Description: "Canonical theorem-pack CSV"},
		{Code: "historical_pack", Domain: "theorem_file", Description: "Immutable historical theorem-pack snapshot"},
		{Code: "direct", Domain: "result_file", Description: "Direct Hilbert benchmark artifact"},
		{Code: "nd", Domain: "result_file", Description: "Natural Deduction to Hilbert benchmark artifact"},
		{Code: "pair", Domain: "result_file", Description: "Paired comparison artifact"},
		{Code: "compositional", Domain: "result_file", Description: "Compositional assumption-import artifact"},
		{Code: "compositional_depth_ladder", Domain: "result_file", Description: "Compositional depth-ladder artifact"},
		{Code: "compositional_branching", Domain: "result_file", Description: "Compositional branching artifact"},
		{Code: "compositional_mixed_family_reuse", Domain: "result_file", Description: "Compositional mixed-family reuse artifact"},
		{Code: "compositional_mixed_family_gold_first", Domain: "result_file", Description: "Compositional mixed-family gold-first artifact"},
		{Code: "compositional_mixed_family_gold_first_hard", Domain: "result_file", Description: "Compositional mixed-family gold-first hard artifact"},
		{Code: "compositional_mixed_family_semi_gold_stable", Domain: "result_file", Description: "Compositional mixed-family semi-gold stable artifact"},
		{Code: "compositional_mixed_family_semi_gold_frontier", Domain: "result_file", Description: "Compositional mixed-family semi-gold frontier artifact"},
		{Code: "wave", Domain: "result_file", Description: "Cross-run or wave artifact"},
		{Code: "family", Domain: "result_file", Description: "Family-local artifact"},
		{Code: "summary", Domain: "result_row", Description: "Summary-level imported row"},
		{Code: "result_row", Domain: "result_row", Description: "Per-theorem imported result row"},
		{Code: "chain_summary_row", Domain: "result_row", Description: "Per-chain compositional summary row"},
		{Code: "unknown", Domain: "result_file", Description: "Unclassified artifact"},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_types(type_code, domain, description)
VALUES (?, ?, ?)
ON CONFLICT(type_code) DO UPDATE SET
    domain = excluded.domain,
    description = excluded.description
`, row.Code, row.Domain, row.Description); err != nil {
			return fmt.Errorf("seed research type %q: %w", row.Code, err)
		}
	}
	return nil
}
