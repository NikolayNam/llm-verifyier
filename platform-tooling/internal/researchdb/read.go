package researchdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type OverviewOptions struct {
	ProjectFolder string
}

type Overview struct {
	DBPath                string
	Scope                 string
	Researches            []ResearchOverview
	ArtifactGroups        []ArtifactGroupOverview
	TotalResearches       int
	TotalLLMModels        int
	TotalCertificates     int
	TotalGeneratedPacks   int
	TotalExports          int
	TotalHypothesisSets   int
	TotalChainSummaryRows int
	GlobalRuns            RunCounts
	GlobalJobs            JobCounts
	LatestRun             RunHeadline
}

type ResearchOverview struct {
	Slug                    string
	Title                   string
	ArtifactRoot            string
	TheoremFiles            int
	HistoricalTheoremFiles  int
	Theorems                int
	ResultFiles             int
	SummaryFiles            int
	ResultRowFiles          int
	ChainSummaryFiles       int
	SummaryRows             int
	ResultRows              int
	ChainSummaryRows        int
	HypothesisSets          int
	LatestArtifactImportUTC string
}

type ArtifactGroupOverview struct {
	ArtifactGroup     string
	ResultFiles       int
	ChainSummaryFiles int
	SummaryRows       int
	ResultRows        int
	ChainSummaryRows  int
}

type RunCounts struct {
	Total         int
	Running       int
	Completed     int
	PartialFailed int
	Failed        int
	Aborted       int
}

type JobCounts struct {
	Total     int
	Running   int
	Completed int
	Skipped   int
	Failed    int
	Aborted   int
}

type RunHeadline struct {
	RunID         string
	Command       string
	Phase         string
	Status        string
	StartedAtUTC  string
	FinishedAtUTC string
}

type ListRunsOptions struct {
	Phase  string
	Status string
	Limit  int
}

type RunListEntry struct {
	RunID         string
	Command       string
	Phase         string
	Status        string
	ManifestPath  string
	StartedAtUTC  string
	FinishedAtUTC string
	Error         string
	CompletedJobs int
	SkippedJobs   int
	FailedJobs    int
	AbortedJobs   int
	RunningJobs   int
	ArtifactCount int
}

func LoadOverview(ctx context.Context, dbPath string, opts OverviewOptions) (Overview, error) {
	db, err := Open(ctx, dbPath)
	if err != nil {
		return Overview{}, err
	}
	defer db.Close()

	projectFolder := strings.TrimSpace(opts.ProjectFolder)
	scope := "all"
	if projectFolder != "" {
		scope = projectFolder
	}

	researches, err := loadResearchOverviewRows(ctx, db, projectFolder)
	if err != nil {
		return Overview{}, err
	}
	artifactGroups, err := loadArtifactGroupOverviewRows(ctx, db, projectFolder)
	if err != nil {
		return Overview{}, err
	}
	totalResearches, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_researches`)
	if err != nil {
		return Overview{}, err
	}
	totalLLMModels, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_llm_models`)
	if err != nil {
		return Overview{}, err
	}
	totalCertificates, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_certificates`)
	if err != nil {
		return Overview{}, err
	}
	totalGeneratedPacks, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_generated_packs`)
	if err != nil {
		return Overview{}, err
	}
	totalExports, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_exports`)
	if err != nil {
		return Overview{}, err
	}
	totalHypothesisSets, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_hypothesis_sets`)
	if err != nil {
		return Overview{}, err
	}
	totalChainSummaryRows, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_chain_summary_rows`)
	if err != nil {
		return Overview{}, err
	}
	globalRuns, err := loadRunCounts(ctx, db)
	if err != nil {
		return Overview{}, err
	}
	globalJobs, err := loadJobCounts(ctx, db)
	if err != nil {
		return Overview{}, err
	}
	latestRun, err := loadLatestRun(ctx, db)
	if err != nil {
		return Overview{}, err
	}

	return Overview{
		DBPath:                dbPath,
		Scope:                 scope,
		Researches:            researches,
		ArtifactGroups:        artifactGroups,
		TotalResearches:       totalResearches,
		TotalLLMModels:        totalLLMModels,
		TotalCertificates:     totalCertificates,
		TotalGeneratedPacks:   totalGeneratedPacks,
		TotalExports:          totalExports,
		TotalHypothesisSets:   totalHypothesisSets,
		TotalChainSummaryRows: totalChainSummaryRows,
		GlobalRuns:            globalRuns,
		GlobalJobs:            globalJobs,
		LatestRun:             latestRun,
	}, nil
}

func ListRuns(ctx context.Context, dbPath string, opts ListRunsOptions) ([]RunListEntry, error) {
	db, err := Open(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	phase := strings.TrimSpace(opts.Phase)
	status := strings.TrimSpace(opts.Status)
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.QueryContext(ctx, `
SELECT
    run_id,
    command,
    phase,
    status,
    manifest_path,
    started_at_utc,
    finished_at_utc,
    error,
    (SELECT COUNT(*) FROM research_jobs WHERE run_id = research_runs.run_id AND status = 'completed') AS completed_jobs,
    (SELECT COUNT(*) FROM research_jobs WHERE run_id = research_runs.run_id AND status = 'skipped') AS skipped_jobs,
    (SELECT COUNT(*) FROM research_jobs WHERE run_id = research_runs.run_id AND status = 'failed') AS failed_jobs,
    (SELECT COUNT(*) FROM research_jobs WHERE run_id = research_runs.run_id AND status = 'aborted') AS aborted_jobs,
    (SELECT COUNT(*) FROM research_jobs WHERE run_id = research_runs.run_id AND status = 'running') AS running_jobs,
    (SELECT COUNT(*) FROM research_run_artifacts WHERE run_id = research_runs.run_id) AS artifact_count
FROM research_runs
WHERE (? = '' OR phase = ?)
  AND (? = '' OR status = ?)
ORDER BY COALESCE(NULLIF(started_at_utc, ''), NULLIF(finished_at_utc, ''), run_id) DESC, run_id DESC
LIMIT ?
`, phase, phase, status, status, limit)
	if err != nil {
		return nil, fmt.Errorf("query research runs: %w", err)
	}
	defer rows.Close()

	result := make([]RunListEntry, 0, limit)
	for rows.Next() {
		var entry RunListEntry
		if err := rows.Scan(
			&entry.RunID,
			&entry.Command,
			&entry.Phase,
			&entry.Status,
			&entry.ManifestPath,
			&entry.StartedAtUTC,
			&entry.FinishedAtUTC,
			&entry.Error,
			&entry.CompletedJobs,
			&entry.SkippedJobs,
			&entry.FailedJobs,
			&entry.AbortedJobs,
			&entry.RunningJobs,
			&entry.ArtifactCount,
		); err != nil {
			return nil, fmt.Errorf("scan research run row: %w", err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research run rows: %w", err)
	}
	return result, nil
}

func loadResearchOverviewRows(ctx context.Context, db *sql.DB, projectFolder string) ([]ResearchOverview, error) {
	query := `
SELECT
    slug,
    title,
    artifact_root,
    (SELECT COUNT(*) FROM research_theorem_files WHERE research_id = research_researches.id) AS theorem_files,
    (SELECT COUNT(*) FROM research_theorem_files WHERE research_id = research_researches.id AND is_historical = 1) AS historical_theorem_files,
    (SELECT COUNT(*) FROM research_theorems WHERE research_id = research_researches.id) AS theorems,
    (SELECT COUNT(*) FROM research_result_files WHERE research_id = research_researches.id) AS result_files,
    (SELECT COUNT(*) FROM research_result_files WHERE research_id = research_researches.id AND file_kind = 'summary') AS summary_files,
    (SELECT COUNT(*) FROM research_result_files WHERE research_id = research_researches.id AND file_kind = 'result_rows') AS result_row_files,
    (SELECT COUNT(*) FROM research_result_files WHERE research_id = research_researches.id AND file_kind = 'chain_summary') AS chain_summary_files,
    (SELECT COUNT(*) FROM research_result_summaries WHERE research_id = research_researches.id) AS summary_rows,
    (SELECT COUNT(*) FROM research_result_rows WHERE research_id = research_researches.id) AS result_rows,
    (SELECT COUNT(*) FROM research_chain_summary_rows WHERE research_id = research_researches.id) AS chain_summary_rows,
    (SELECT COUNT(*) FROM research_hypothesis_sets WHERE research_id = research_researches.id) AS hypothesis_sets,
    COALESCE((
        SELECT MAX(imported_at_utc)
        FROM (
            SELECT imported_at_utc FROM research_theorem_files WHERE research_id = research_researches.id
            UNION ALL
            SELECT imported_at_utc FROM research_result_files WHERE research_id = research_researches.id
        )
    ), '') AS latest_artifact_import_utc
FROM research_researches
`
	args := make([]any, 0, 1)
	if projectFolder != "" {
		query += ` WHERE slug = ?`
		args = append(args, projectFolder)
	}
	query += ` ORDER BY slug`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query research overview rows: %w", err)
	}
	defer rows.Close()

	result := make([]ResearchOverview, 0, 8)
	for rows.Next() {
		var row ResearchOverview
		if err := rows.Scan(
			&row.Slug,
			&row.Title,
			&row.ArtifactRoot,
			&row.TheoremFiles,
			&row.HistoricalTheoremFiles,
			&row.Theorems,
			&row.ResultFiles,
			&row.SummaryFiles,
			&row.ResultRowFiles,
			&row.ChainSummaryFiles,
			&row.SummaryRows,
			&row.ResultRows,
			&row.ChainSummaryRows,
			&row.HypothesisSets,
			&row.LatestArtifactImportUTC,
		); err != nil {
			return nil, fmt.Errorf("scan research overview row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research overview rows: %w", err)
	}
	return result, nil
}

func loadArtifactGroupOverviewRows(ctx context.Context, db *sql.DB, projectFolder string) ([]ArtifactGroupOverview, error) {
	query := `
SELECT
    rf.artifact_group,
    COUNT(*) AS result_files,
    COALESCE(SUM(CASE WHEN rf.file_kind = 'chain_summary' THEN 1 ELSE 0 END), 0) AS chain_summary_files,
    COALESCE(SUM((SELECT COUNT(*) FROM research_result_summaries WHERE result_file_id = rf.id)), 0) AS summary_rows,
    COALESCE(SUM((SELECT COUNT(*) FROM research_result_rows WHERE result_file_id = rf.id)), 0) AS result_rows,
    COALESCE(SUM((SELECT COUNT(*) FROM research_chain_summary_rows WHERE result_file_id = rf.id)), 0) AS chain_summary_rows
FROM research_result_files rf
JOIN research_researches rr ON rr.id = rf.research_id
`
	args := make([]any, 0, 1)
	if projectFolder != "" {
		query += ` WHERE rr.slug = ?`
		args = append(args, projectFolder)
	}
	query += ` GROUP BY rf.artifact_group ORDER BY rf.artifact_group`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query artifact group overview rows: %w", err)
	}
	defer rows.Close()

	result := make([]ArtifactGroupOverview, 0, 8)
	for rows.Next() {
		var row ArtifactGroupOverview
		if err := rows.Scan(&row.ArtifactGroup, &row.ResultFiles, &row.ChainSummaryFiles, &row.SummaryRows, &row.ResultRows, &row.ChainSummaryRows); err != nil {
			return nil, fmt.Errorf("scan artifact group overview row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifact group overview rows: %w", err)
	}
	return result, nil
}

func loadRunCounts(ctx context.Context, db *sql.DB) (RunCounts, error) {
	total, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs`)
	if err != nil {
		return RunCounts{}, err
	}
	running, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs WHERE status = 'running'`)
	if err != nil {
		return RunCounts{}, err
	}
	completed, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs WHERE status = 'completed'`)
	if err != nil {
		return RunCounts{}, err
	}
	partialFailed, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs WHERE status = 'partial_failed'`)
	if err != nil {
		return RunCounts{}, err
	}
	failed, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs WHERE status = 'failed'`)
	if err != nil {
		return RunCounts{}, err
	}
	aborted, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_runs WHERE status = 'aborted'`)
	if err != nil {
		return RunCounts{}, err
	}
	return RunCounts{
		Total:         total,
		Running:       running,
		Completed:     completed,
		PartialFailed: partialFailed,
		Failed:        failed,
		Aborted:       aborted,
	}, nil
}

func loadJobCounts(ctx context.Context, db *sql.DB) (JobCounts, error) {
	total, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs`)
	if err != nil {
		return JobCounts{}, err
	}
	running, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs WHERE status = 'running'`)
	if err != nil {
		return JobCounts{}, err
	}
	completed, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs WHERE status = 'completed'`)
	if err != nil {
		return JobCounts{}, err
	}
	skipped, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs WHERE status = 'skipped'`)
	if err != nil {
		return JobCounts{}, err
	}
	failed, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs WHERE status = 'failed'`)
	if err != nil {
		return JobCounts{}, err
	}
	aborted, err := countQuery(ctx, db, `SELECT COUNT(*) FROM research_jobs WHERE status = 'aborted'`)
	if err != nil {
		return JobCounts{}, err
	}
	return JobCounts{
		Total:     total,
		Running:   running,
		Completed: completed,
		Skipped:   skipped,
		Failed:    failed,
		Aborted:   aborted,
	}, nil
}

func loadLatestRun(ctx context.Context, db *sql.DB) (RunHeadline, error) {
	var row RunHeadline
	err := db.QueryRowContext(ctx, `
SELECT run_id, command, phase, status, started_at_utc, finished_at_utc
FROM research_runs
ORDER BY COALESCE(NULLIF(started_at_utc, ''), NULLIF(finished_at_utc, ''), run_id) DESC, run_id DESC
LIMIT 1
`).Scan(&row.RunID, &row.Command, &row.Phase, &row.Status, &row.StartedAtUTC, &row.FinishedAtUTC)
	if err != nil {
		if err == sql.ErrNoRows {
			return RunHeadline{}, nil
		}
		return RunHeadline{}, fmt.Errorf("query latest research run: %w", err)
	}
	return row, nil
}

func countQuery(ctx context.Context, db *sql.DB, query string, args ...any) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}
	return count, nil
}
