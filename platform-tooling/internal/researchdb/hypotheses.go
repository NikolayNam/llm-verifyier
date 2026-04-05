package researchdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
)

type PersistHypothesisGenerationInput struct {
	WorkspaceRoot         string
	ProjectFolder         string
	ArtifactRoot          string
	SourceFilePath        string
	HistoricalTheoremPath string
	Config                lean4worker.HypothesisGenerationConfig
	Cases                 []lean4worker.HypothesisCase
	GenerationConfigPath  string
	DefaultPayloadPath    string
	CreatedAt             time.Time
}

type PersistHypothesisGenerationResult struct {
	ResearchID      int64
	GenerationJobID int64
	HypothesisSetID int64
	TheoremPackID   string
}

type HypothesisSetSelector struct {
	ProjectFolder   string
	HypothesisSetID int64
	TheoremPackID   string
}

type ResolvedHypothesisSet struct {
	ResearchID      int64
	HypothesisSetID int64
	GenerationJobID int64
	TheoremFileID   int64
	TheoremPackID   string
	GenerationMode  string
	LogicFragment   string
	CreatedAtUTC    string
}

type ExportSnapshotInput struct {
	WorkspaceRoot          string
	ResearchID             int64
	HypothesisSetID        int64
	TheoremPackID          string
	BenchmarkProjectFolder string
	ModeFilter             string
	InterestingOnly        bool
	MinimalOnly            bool
	LimitApplied           int
	OutputPath             string
	CreatedAt              time.Time
}

func PersistHypothesisGeneration(ctx context.Context, db *sql.DB, input PersistHypothesisGenerationInput) (PersistHypothesisGenerationResult, error) {
	workspaceRoot := strings.TrimSpace(input.WorkspaceRoot)
	projectFolder := strings.TrimSpace(input.ProjectFolder)
	artifactRoot := strings.TrimSpace(input.ArtifactRoot)
	if workspaceRoot == "" {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("workspace root is required")
	}
	if projectFolder == "" {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("project folder is required")
	}
	if artifactRoot == "" {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("artifact root is required")
	}
	if len(input.Cases) == 0 {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("at least one hypothesis case is required")
	}

	theoremPackID, generationMode, logicFragment, err := validateHypothesisCases(input.Cases)
	if err != nil {
		return PersistHypothesisGenerationResult{}, err
	}

	createdAt := input.CreatedAt.UTC()
	if input.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	createdAtUTC := createdAt.Format(time.RFC3339)

	researchID, err := ensureResearch(ctx, db, projectFolder, relativeSlash(workspaceRoot, artifactRoot))
	if err != nil {
		return PersistHypothesisGenerationResult{}, err
	}

	atomsJSON, err := marshalJSONString(input.Config.Atoms)
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("marshal atoms_json: %w", err)
	}
	filtersJSON, err := marshalJSONString(input.Config.Filters)
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("marshal filters_json: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("begin hypothesis persistence tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx, `
INSERT INTO research_generation_jobs(
    research_id, generation_mode, source_file_path, atoms_json, filters_json, max_formula_depth, max_assumptions,
    limit_requested, generated_count, artifact_root_relative_path, generation_config_relative_path,
    default_payload_relative_path, status, error_text, created_at_utc, completed_at_utc
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending_artifacts', '', ?, '')
`, researchID, generationMode, normalizePath(input.SourceFilePath), string(atomsJSON), string(filtersJSON),
		input.Config.MaxFormulaDepth, input.Config.MaxAssumptions, input.Config.Limit, len(input.Cases),
		relativeSlash(workspaceRoot, artifactRoot), relativePathOrEmpty(workspaceRoot, input.GenerationConfigPath),
		relativePathOrEmpty(workspaceRoot, input.DefaultPayloadPath), createdAtUTC)
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("insert research_generation_jobs: %w", err)
	}
	generationJobID, err := result.LastInsertId()
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("read generation job id: %w", err)
	}

	theoremFileID, err := upsertHypothesisTheoremFile(ctx, tx, workspaceRoot, researchID, artifactRoot, theoremPackID, strings.TrimSpace(input.HistoricalTheoremPath), createdAtUTC)
	if err != nil {
		return PersistHypothesisGenerationResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM research_theorems WHERE theorem_file_id = ?`, theoremFileID); err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("clear theorem rows for theorem_file_id %d: %w", theoremFileID, err)
	}
	for idx, hypothesis := range input.Cases {
		assumptionsJSON, err := marshalJSONString(hypothesis.Assumptions)
		if err != nil {
			return PersistHypothesisGenerationResult{}, fmt.Errorf("marshal assumptions for %s: %w", hypothesis.CaseID, err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO research_theorems(
    research_id, theorem_file_id, theorem_pack_id, theorem_id, row_order, case_id, generation_mode, category,
    derivability_status, interesting, minimal, difficulty, assumptions_json, goal, logic_fragment, lean_statement, comment
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, theoremFileID, theoremPackID, strings.TrimSpace(hypothesis.TheoremID), idx+1, strings.TrimSpace(hypothesis.CaseID),
			strings.TrimSpace(hypothesis.GenerationMode), strings.TrimSpace(hypothesis.Category), strings.TrimSpace(hypothesis.DerivabilityStatus),
			boolToInt(hypothesis.Interesting), boolToInt(hypothesis.Minimal), strings.TrimSpace(hypothesis.Difficulty), string(assumptionsJSON),
			strings.TrimSpace(hypothesis.Goal), strings.TrimSpace(hypothesis.LogicFragment), strings.TrimSpace(hypothesis.LeanStatement), strings.TrimSpace(hypothesis.Comment)); err != nil {
			return PersistHypothesisGenerationResult{}, fmt.Errorf("insert theorem-backed hypothesis row %s: %w", hypothesis.CaseID, err)
		}
	}

	result, err = tx.ExecContext(ctx, `
INSERT INTO research_hypothesis_sets(
    research_id, generation_job_id, theorem_file_id, theorem_pack_id, generation_mode, logic_fragment, hypothesis_count, created_at_utc
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, generationJobID, theoremFileID, theoremPackID, generationMode, logicFragment, len(input.Cases), createdAtUTC)
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("insert research_hypothesis_sets: %w", err)
	}
	hypothesisSetID, err := result.LastInsertId()
	if err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("read hypothesis set id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return PersistHypothesisGenerationResult{}, fmt.Errorf("commit hypothesis persistence tx: %w", err)
	}

	return PersistHypothesisGenerationResult{
		ResearchID:      researchID,
		GenerationJobID: generationJobID,
		HypothesisSetID: hypothesisSetID,
		TheoremPackID:   theoremPackID,
	}, nil
}

func CompleteGenerationJob(ctx context.Context, db *sql.DB, generationJobID int64, completedAt time.Time) error {
	return updateGenerationJobStatus(ctx, db, generationJobID, "completed", "", completedAt)
}

func FailGenerationJob(ctx context.Context, db *sql.DB, generationJobID int64, errorText string, completedAt time.Time) error {
	return updateGenerationJobStatus(ctx, db, generationJobID, "artifact_write_failed", errorText, completedAt)
}

func ResolveHypothesisSet(ctx context.Context, db *sql.DB, selector HypothesisSetSelector) (ResolvedHypothesisSet, error) {
	if selector.HypothesisSetID > 0 {
		return queryHypothesisSet(ctx, db, `
SELECT hs.research_id, hs.id, hs.generation_job_id, COALESCE(hs.theorem_file_id, 0), hs.theorem_pack_id, hs.generation_mode, hs.logic_fragment, hs.created_at_utc
FROM research_hypothesis_sets hs
JOIN research_generation_jobs gj ON gj.id = hs.generation_job_id
WHERE hs.id = ? AND gj.status = 'completed'
LIMIT 1
`, selector.HypothesisSetID)
	}
	if theoremPackID := strings.TrimSpace(selector.TheoremPackID); theoremPackID != "" {
		if projectFolder := strings.TrimSpace(selector.ProjectFolder); projectFolder != "" {
			return queryHypothesisSet(ctx, db, `
SELECT hs.research_id, hs.id, hs.generation_job_id, COALESCE(hs.theorem_file_id, 0), hs.theorem_pack_id, hs.generation_mode, hs.logic_fragment, hs.created_at_utc
FROM research_hypothesis_sets hs
JOIN research_generation_jobs gj ON gj.id = hs.generation_job_id
JOIN research_researches rr ON rr.id = hs.research_id
WHERE hs.theorem_pack_id = ? AND rr.slug = ? AND gj.status = 'completed'
ORDER BY hs.created_at_utc DESC, hs.id DESC
LIMIT 1
`, theoremPackID, projectFolder)
		}
		return queryHypothesisSet(ctx, db, `
SELECT hs.research_id, hs.id, hs.generation_job_id, COALESCE(hs.theorem_file_id, 0), hs.theorem_pack_id, hs.generation_mode, hs.logic_fragment, hs.created_at_utc
FROM research_hypothesis_sets hs
JOIN research_generation_jobs gj ON gj.id = hs.generation_job_id
WHERE hs.theorem_pack_id = ? AND gj.status = 'completed'
ORDER BY hs.created_at_utc DESC, hs.id DESC
LIMIT 1
`, theoremPackID)
	}
	projectFolder := strings.TrimSpace(selector.ProjectFolder)
	if projectFolder == "" {
		return ResolvedHypothesisSet{}, fmt.Errorf("project folder is required when hypothesis_set_id and theorem_pack_id are empty")
	}
	return queryHypothesisSet(ctx, db, `
SELECT hs.research_id, hs.id, hs.generation_job_id, COALESCE(hs.theorem_file_id, 0), hs.theorem_pack_id, hs.generation_mode, hs.logic_fragment, hs.created_at_utc
FROM research_hypothesis_sets hs
JOIN research_generation_jobs gj ON gj.id = hs.generation_job_id
JOIN research_researches rr ON rr.id = hs.research_id
WHERE rr.slug = ? AND gj.status = 'completed'
ORDER BY hs.created_at_utc DESC, hs.id DESC
LIMIT 1
`, projectFolder)
}

func LoadHypothesisCasesBySet(ctx context.Context, db *sql.DB, hypothesisSetID int64) ([]lean4worker.HypothesisCase, error) {
	var theoremFileID int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(theorem_file_id, 0) FROM research_hypothesis_sets WHERE id = ?`, hypothesisSetID).Scan(&theoremFileID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("matching hypothesis set not found")
		}
		return nil, fmt.Errorf("lookup theorem_file_id for set %d: %w", hypothesisSetID, err)
	}
	if theoremFileID > 0 {
		cases, err := loadHypothesisCasesByTheoremFile(ctx, db, theoremFileID)
		if err != nil {
			return nil, fmt.Errorf("load canonical hypothesis cases for set %d: %w", hypothesisSetID, err)
		}
		return cases, nil
	}
	return nil, fmt.Errorf("hypothesis set %d is not linked to canonical theorem-backed storage", hypothesisSetID)
}

func StartExportSnapshot(ctx context.Context, db *sql.DB, input ExportSnapshotInput) (int64, error) {
	if input.ResearchID <= 0 {
		return 0, fmt.Errorf("research id is required")
	}
	if input.HypothesisSetID <= 0 {
		return 0, fmt.Errorf("hypothesis set id is required")
	}
	if strings.TrimSpace(input.TheoremPackID) == "" {
		return 0, fmt.Errorf("theorem pack id is required")
	}
	if strings.TrimSpace(input.BenchmarkProjectFolder) == "" {
		return 0, fmt.Errorf("benchmark project folder is required")
	}
	if strings.TrimSpace(input.OutputPath) == "" {
		return 0, fmt.Errorf("output path is required")
	}
	createdAt := input.CreatedAt.UTC()
	if input.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	modeFilter := strings.TrimSpace(input.ModeFilter)
	if modeFilter == "" {
		modeFilter = "*"
	}
	result, err := db.ExecContext(ctx, `
INSERT INTO research_hypothesis_export_snapshots(
    research_id, hypothesis_set_id, theorem_pack_id, benchmark_project_folder, mode_filter, interesting_only,
    minimal_only, limit_applied, exported_count, output_relative_path, historical_relative_path, status,
    error_text, created_at_utc, completed_at_utc
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, '', 'writing', '', ?, '')
`, input.ResearchID, input.HypothesisSetID, strings.TrimSpace(input.TheoremPackID), strings.TrimSpace(input.BenchmarkProjectFolder),
		modeFilter, boolToInt(input.InterestingOnly), boolToInt(input.MinimalOnly), input.LimitApplied,
		relativePathOrEmpty(strings.TrimSpace(input.WorkspaceRoot), input.OutputPath), createdAt.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("insert research_hypothesis_export_snapshots: %w", err)
	}
	snapshotID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read export snapshot id: %w", err)
	}
	return snapshotID, nil
}

func CompleteExportSnapshot(ctx context.Context, db *sql.DB, snapshotID int64, exportedCount int, workspaceRoot, historicalPath string, completedAt time.Time) error {
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
UPDATE research_hypothesis_export_snapshots
SET exported_count = ?,
    historical_relative_path = ?,
    status = 'completed',
    error_text = '',
    completed_at_utc = ?
WHERE id = ?
`, exportedCount, relativePathOrEmpty(workspaceRoot, historicalPath), completedAt.UTC().Format(time.RFC3339), snapshotID)
	if err != nil {
		return fmt.Errorf("complete export snapshot %d: %w", snapshotID, err)
	}
	return nil
}

func FailExportSnapshot(ctx context.Context, db *sql.DB, snapshotID int64, errorText string, completedAt time.Time) error {
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
UPDATE research_hypothesis_export_snapshots
SET status = 'write_failed',
    error_text = ?,
    completed_at_utc = ?
WHERE id = ?
`, strings.TrimSpace(errorText), completedAt.UTC().Format(time.RFC3339), snapshotID)
	if err != nil {
		return fmt.Errorf("fail export snapshot %d: %w", snapshotID, err)
	}
	return nil
}

func updateGenerationJobStatus(ctx context.Context, db *sql.DB, generationJobID int64, status, errorText string, completedAt time.Time) error {
	if generationJobID <= 0 {
		return fmt.Errorf("generation job id is required")
	}
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
UPDATE research_generation_jobs
SET status = ?,
    error_text = ?,
    completed_at_utc = ?
WHERE id = ?
`, status, strings.TrimSpace(errorText), completedAt.UTC().Format(time.RFC3339), generationJobID)
	if err != nil {
		return fmt.Errorf("update generation job %d status: %w", generationJobID, err)
	}
	return nil
}

func queryHypothesisSet(ctx context.Context, db *sql.DB, query string, args ...any) (ResolvedHypothesisSet, error) {
	var resolved ResolvedHypothesisSet
	err := db.QueryRowContext(ctx, query, args...).Scan(&resolved.ResearchID, &resolved.HypothesisSetID, &resolved.GenerationJobID, &resolved.TheoremFileID,
		&resolved.TheoremPackID, &resolved.GenerationMode, &resolved.LogicFragment, &resolved.CreatedAtUTC)
	if err != nil {
		if err == sql.ErrNoRows {
			return ResolvedHypothesisSet{}, fmt.Errorf("matching hypothesis set not found")
		}
		return ResolvedHypothesisSet{}, fmt.Errorf("query hypothesis set: %w", err)
	}
	return resolved, nil
}

func upsertHypothesisTheoremFile(ctx context.Context, tx *sql.Tx, workspaceRoot string, researchID int64, artifactRoot, theoremPackID, explicitHistoricalPath, importedAtUTC string) (int64, error) {
	theoremsDir := filepath.Join(artifactRoot, lean4worker.DefaultTheoremsDir)
	historicalPath := strings.TrimSpace(explicitHistoricalPath)
	if historicalPath == "" {
		historicalPath = lean4worker.HistoricalTheoremSnapshotPath(theoremsDir, theoremPackID)
	}
	var typeID sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT id FROM research_types WHERE type_code = ?`, "historical_pack").Scan(&typeID)
	if err != nil {
		return 0, fmt.Errorf("lookup research type %q: %w", "historical_pack", err)
	}
	if !typeID.Valid {
		return 0, fmt.Errorf("research type %q not found", "historical_pack")
	}
	relativePath := relativeSlash(workspaceRoot, historicalPath)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO research_theorem_files(research_id, type_id, theorem_pack_id, relative_path, absolute_path, is_historical, imported_at_utc)
VALUES (?, ?, ?, ?, ?, 1, ?)
ON CONFLICT(relative_path) DO UPDATE SET
    research_id = excluded.research_id,
    type_id = excluded.type_id,
    theorem_pack_id = excluded.theorem_pack_id,
    absolute_path = excluded.absolute_path,
    is_historical = excluded.is_historical,
    imported_at_utc = excluded.imported_at_utc
`, researchID, typeID.Int64, theoremPackID, relativePath, filepath.ToSlash(historicalPath), importedAtUTC); err != nil {
		return 0, fmt.Errorf("upsert hypothesis theorem file %s: %w", relativePath, err)
	}
	var theoremFileID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM research_theorem_files WHERE relative_path = ?`, relativePath).Scan(&theoremFileID); err != nil {
		return 0, fmt.Errorf("lookup hypothesis theorem file %s: %w", relativePath, err)
	}
	return theoremFileID, nil
}

func loadHypothesisCasesByTheoremFile(ctx context.Context, db *sql.DB, theoremFileID int64) ([]lean4worker.HypothesisCase, error) {
	rows, err := db.QueryContext(ctx, `
SELECT theorem_pack_id, theorem_id, case_id, generation_mode, category, derivability_status, interesting,
       minimal, difficulty, assumptions_json, goal, logic_fragment, lean_statement, comment
FROM research_theorems
WHERE theorem_file_id = ?
ORDER BY CASE WHEN row_order > 0 THEN row_order ELSE 2147483647 END ASC, theorem_id ASC, id ASC
`, theoremFileID)
	if err != nil {
		return nil, fmt.Errorf("query research_theorems for theorem_file_id %d: %w", theoremFileID, err)
	}
	defer rows.Close()
	return scanHypothesisCases(rows, fmt.Sprintf("theorem_file_id %d", theoremFileID))
}

func scanHypothesisCases(rows *sql.Rows, source string) ([]lean4worker.HypothesisCase, error) {
	cases := make([]lean4worker.HypothesisCase, 0, 64)
	for rows.Next() {
		var (
			hypothesis           lean4worker.HypothesisCase
			interesting, minimal int
			assumptionsJSON      string
		)
		if err := rows.Scan(&hypothesis.TheoremPackID, &hypothesis.TheoremID, &hypothesis.CaseID,
			&hypothesis.GenerationMode, &hypothesis.Category, &hypothesis.DerivabilityStatus, &interesting,
			&minimal, &hypothesis.Difficulty, &assumptionsJSON, &hypothesis.Goal, &hypothesis.LogicFragment,
			&hypothesis.LeanStatement, &hypothesis.Comment); err != nil {
			return nil, fmt.Errorf("scan hypothesis row for %s: %w", source, err)
		}
		if err := json.Unmarshal([]byte(assumptionsJSON), &hypothesis.Assumptions); err != nil {
			return nil, fmt.Errorf("decode assumptions_json for %s theorem %s: %w", source, hypothesis.TheoremID, err)
		}
		hypothesis.Interesting = interesting != 0
		hypothesis.Minimal = minimal != 0
		cases = append(cases, hypothesis)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hypothesis rows for %s: %w", source, err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("no hypotheses found for %s", source)
	}
	return cases, nil
}

func validateHypothesisCases(cases []lean4worker.HypothesisCase) (string, string, string, error) {
	theoremPackID := strings.TrimSpace(cases[0].TheoremPackID)
	generationMode := strings.TrimSpace(cases[0].GenerationMode)
	logicFragment := strings.TrimSpace(cases[0].LogicFragment)
	if theoremPackID == "" {
		return "", "", "", fmt.Errorf("hypothesis theorem_pack_id is required")
	}
	if generationMode == "" {
		return "", "", "", fmt.Errorf("hypothesis generation_mode is required")
	}
	for _, hypothesis := range cases {
		if strings.TrimSpace(hypothesis.TheoremPackID) != theoremPackID {
			return "", "", "", fmt.Errorf("hypothesis theorem_pack_id must be consistent across one set")
		}
		if strings.TrimSpace(hypothesis.GenerationMode) != generationMode {
			return "", "", "", fmt.Errorf("hypothesis generation_mode must be consistent across one set")
		}
		if strings.TrimSpace(hypothesis.TheoremID) == "" {
			return "", "", "", fmt.Errorf("hypothesis theorem_id is required")
		}
		if strings.TrimSpace(hypothesis.CaseID) == "" {
			return "", "", "", fmt.Errorf("hypothesis case_id is required")
		}
		if logicFragment == "" && strings.TrimSpace(hypothesis.LogicFragment) != "" {
			logicFragment = strings.TrimSpace(hypothesis.LogicFragment)
		}
	}
	return theoremPackID, generationMode, logicFragment, nil
}

func marshalJSONString(value any) ([]byte, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return []byte("[]"), nil
	}
	return bytes, nil
}

func normalizePath(path string) string {
	return filepath.ToSlash(strings.TrimSpace(path))
}

func relativePathOrEmpty(workspaceRoot, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return normalizePath(path)
	}
	return relativeSlash(workspaceRoot, path)
}
