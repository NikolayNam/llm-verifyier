package researchdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type PruneOptions struct {
	WorkspaceRoot string
	ProjectFolder string
	Apply         bool
	Vacuum        bool
}

type PruneResult struct {
	DBPath                  string
	ResearchSlug            string
	DryRun                  bool
	MissingTheoremFiles     int
	MissingResultFiles      int
	DeletedTheoremFiles     int
	DeletedResultFiles      int
	DeletedTheoremRows      int
	DeletedSummaryRows      int
	DeletedResultRows       int
	DeletedChainSummaryRows int
	DeletedLegacyUnionTable bool
	Vacuumed                bool
}

func PruneProject(ctx context.Context, dbPath string, opts PruneOptions) (PruneResult, error) {
	projectFolder := strings.TrimSpace(opts.ProjectFolder)
	if projectFolder == "" {
		return PruneResult{}, fmt.Errorf("project folder is required")
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		return PruneResult{}, err
	}
	defer db.Close()

	researchID, err := lookupResearchID(ctx, db, projectFolder)
	if err != nil {
		return PruneResult{}, err
	}
	if !researchID.Valid {
		return PruneResult{
			DBPath:       dbPath,
			ResearchSlug: projectFolder,
			DryRun:       !opts.Apply,
		}, nil
	}

	theoremIDs, theoremRows, err := collectMissingArtifactIDs(ctx, db, `
SELECT id, absolute_path,
       (SELECT COUNT(*) FROM research_theorems WHERE theorem_file_id = research_theorem_files.id) AS dependent_rows
FROM research_theorem_files
WHERE research_id = ?
`, researchID.Int64)
	if err != nil {
		return PruneResult{}, err
	}
	resultIDs, dependentRows, err := collectMissingResultFileIDs(ctx, db, researchID.Int64)
	if err != nil {
		return PruneResult{}, err
	}

	legacyTableExists, objectType, err := lookupDBObject(ctx, db, "research_results_legacy")
	if err != nil {
		return PruneResult{}, err
	}
	canDropLegacyUnionTable := legacyTableExists && objectType == "table"

	result := PruneResult{
		DBPath:                  dbPath,
		ResearchSlug:            projectFolder,
		DryRun:                  !opts.Apply,
		MissingTheoremFiles:     len(theoremIDs),
		MissingResultFiles:      len(resultIDs),
		DeletedTheoremRows:      theoremRows,
		DeletedSummaryRows:      dependentRows.Summaries,
		DeletedResultRows:       dependentRows.Rows,
		DeletedChainSummaryRows: dependentRows.ChainRows,
		DeletedLegacyUnionTable: canDropLegacyUnionTable && opts.Apply,
	}
	if !opts.Apply {
		return result, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return PruneResult{}, fmt.Errorf("begin prune transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if len(resultIDs) > 0 {
		affected, err := deleteIDs(ctx, tx, "research_result_files", resultIDs)
		if err != nil {
			return PruneResult{}, err
		}
		result.DeletedResultFiles = affected
	}
	if len(theoremIDs) > 0 {
		affected, err := deleteIDs(ctx, tx, "research_theorem_files", theoremIDs)
		if err != nil {
			return PruneResult{}, err
		}
		result.DeletedTheoremFiles = affected
	}
	if canDropLegacyUnionTable {
		if _, err := tx.ExecContext(ctx, `DROP TABLE research_results_legacy`); err != nil {
			return PruneResult{}, fmt.Errorf("drop stale research_results_legacy table: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return PruneResult{}, fmt.Errorf("commit prune transaction: %w", err)
	}

	if opts.Vacuum {
		if _, err := db.ExecContext(ctx, `VACUUM`); err != nil {
			return PruneResult{}, fmt.Errorf("vacuum research db: %w", err)
		}
		result.Vacuumed = true
	}
	return result, nil
}

type rowDependencyCounts struct {
	Summaries int
	Rows      int
	ChainRows int
}

func collectMissingArtifactIDs(ctx context.Context, db *sql.DB, query string, researchID int64) ([]int64, int, error) {
	rows, err := db.QueryContext(ctx, query, researchID)
	if err != nil {
		return nil, 0, fmt.Errorf("query missing artifact candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	totalRows := 0
	for rows.Next() {
		var id int64
		var absolutePath string
		var dependentRows int
		if err := rows.Scan(&id, &absolutePath, &dependentRows); err != nil {
			return nil, 0, fmt.Errorf("scan missing artifact candidate: %w", err)
		}
		if fileStillExists(absolutePath) {
			continue
		}
		ids = append(ids, id)
		totalRows += dependentRows
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate missing artifact candidates: %w", err)
	}
	return ids, totalRows, nil
}

func collectMissingResultFileIDs(ctx context.Context, db *sql.DB, researchID int64) ([]int64, rowDependencyCounts, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, absolute_path,
       (SELECT COUNT(*) FROM research_result_summaries WHERE result_file_id = research_result_files.id) AS summary_rows,
       (SELECT COUNT(*) FROM research_result_rows WHERE result_file_id = research_result_files.id) AS result_rows,
       (SELECT COUNT(*) FROM research_chain_summary_rows WHERE result_file_id = research_result_files.id) AS chain_summary_rows
FROM research_result_files
WHERE research_id = ?
`, researchID)
	if err != nil {
		return nil, rowDependencyCounts{}, fmt.Errorf("query missing result file candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	counts := rowDependencyCounts{}
	for rows.Next() {
		var id int64
		var absolutePath string
		var summaries int
		var resultRows int
		var chainRows int
		if err := rows.Scan(&id, &absolutePath, &summaries, &resultRows, &chainRows); err != nil {
			return nil, rowDependencyCounts{}, fmt.Errorf("scan missing result file candidate: %w", err)
		}
		if fileStillExists(absolutePath) {
			continue
		}
		ids = append(ids, id)
		counts.Summaries += summaries
		counts.Rows += resultRows
		counts.ChainRows += chainRows
	}
	if err := rows.Err(); err != nil {
		return nil, rowDependencyCounts{}, fmt.Errorf("iterate missing result file candidates: %w", err)
	}
	return ids, counts, nil
}

func deleteIDs(ctx context.Context, tx *sql.Tx, table string, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", table, strings.Join(placeholders, ","))
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete %s rows: %w", table, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected for %s delete: %w", table, err)
	}
	return int(affected), nil
}

func lookupResearchID(ctx context.Context, db *sql.DB, slug string) (sql.NullInt64, error) {
	var id int64
	err := db.QueryRowContext(ctx, `SELECT id FROM research_researches WHERE slug = ?`, slug).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.NullInt64{}, nil
		}
		return sql.NullInt64{}, fmt.Errorf("lookup research slug %q: %w", slug, err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func fileStillExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
