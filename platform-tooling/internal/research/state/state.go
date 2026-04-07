package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

type Store struct {
	db            *sql.DB
	workspaceRoot string
}

type Run struct {
	RunID             string
	Command           string
	Phase             string
	Status            string
	ManifestPath      string
	ConfigFingerprint string
	StartedAtUTC      string
	FinishedAtUTC     string
	Error             string
	MetadataJSON      string
}

type Job struct {
	RunID         string
	JobKey        string
	SpecHash      string
	Command       string
	Phase         string
	Family        string
	Model         string
	Selector      string
	Status        string
	ResultPath    string
	SummaryPath   string
	RawDir        string
	StartedAtUTC  string
	FinishedAtUTC string
	Error         string
	MetadataJSON  string
}

type Artifact struct {
	RunID        string
	JobKey       string
	Role         string
	AbsolutePath string
	RelativePath string
	ChecksumSHA  string
	MetadataJSON string
	CreatedAtUTC string
}

type GeneratedPack struct {
	RunID         string
	PackKey       string
	ProjectFolder string
	SourceKind    string
	SourcePath    string
	OutputPath    string
	ManifestPath  string
	MetadataJSON  string
	CreatedAtUTC  string
}

type Export struct {
	RunID                  string
	ExportKey              string
	ProjectFolder          string
	BenchmarkProjectFolder string
	SourcePackID           string
	SourcePath             string
	OutputPath             string
	MetadataJSON           string
	CreatedAtUTC           string
}

func Open(ctx context.Context, workspaceRoot, dbPath string) (*Store, error) {
	db, err := researchdb.Open(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, workspaceRoot: strings.TrimSpace(workspaceRoot)}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) UpsertRun(ctx context.Context, run Run) error {
	run.MetadataJSON = normalizeJSONText(run.MetadataJSON)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_runs(run_id, command, phase, status, manifest_path, config_fingerprint, started_at_utc, finished_at_utc, error, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id) DO UPDATE SET
    command = excluded.command,
    phase = excluded.phase,
    status = excluded.status,
    manifest_path = COALESCE(NULLIF(excluded.manifest_path, ''), research_runs.manifest_path),
    config_fingerprint = COALESCE(NULLIF(excluded.config_fingerprint, ''), research_runs.config_fingerprint),
    started_at_utc = COALESCE(NULLIF(excluded.started_at_utc, ''), research_runs.started_at_utc),
    finished_at_utc = COALESCE(NULLIF(excluded.finished_at_utc, ''), research_runs.finished_at_utc),
    error = excluded.error,
    metadata_json = excluded.metadata_json
`, run.RunID, run.Command, run.Phase, run.Status, filepath.ToSlash(run.ManifestPath), run.ConfigFingerprint, run.StartedAtUTC, run.FinishedAtUTC, run.Error, run.MetadataJSON)
	if err != nil {
		return fmt.Errorf("upsert research run %q: %w", run.RunID, err)
	}
	return nil
}

func (s *Store) SaveRunConfig(ctx context.Context, runID, configPath, localConfigPath string, resolvedConfigJSON []byte, sourceFingerprint string) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_run_configs(run_id, config_path, local_config_path, resolved_config_json, source_fingerprint, created_at_utc)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id) DO UPDATE SET
    config_path = excluded.config_path,
    local_config_path = excluded.local_config_path,
    resolved_config_json = excluded.resolved_config_json,
    source_fingerprint = excluded.source_fingerprint,
    created_at_utc = excluded.created_at_utc
`, runID, filepath.ToSlash(configPath), filepath.ToSlash(localConfigPath), string(resolvedConfigJSON), sourceFingerprint, createdAt)
	if err != nil {
		return fmt.Errorf("save research run config %q: %w", runID, err)
	}
	return nil
}

func (s *Store) UpsertJob(ctx context.Context, job Job) error {
	job.MetadataJSON = normalizeJSONText(job.MetadataJSON)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_jobs(run_id, job_key, spec_hash, command, phase, family, model, selector, status, result_path, summary_path, raw_dir, started_at_utc, finished_at_utc, error, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id, job_key) DO UPDATE SET
    spec_hash = excluded.spec_hash,
    command = excluded.command,
    phase = excluded.phase,
    family = excluded.family,
    model = excluded.model,
    selector = excluded.selector,
    status = excluded.status,
    result_path = excluded.result_path,
    summary_path = excluded.summary_path,
    raw_dir = excluded.raw_dir,
    started_at_utc = excluded.started_at_utc,
    finished_at_utc = excluded.finished_at_utc,
    error = excluded.error,
    metadata_json = excluded.metadata_json
`, job.RunID, job.JobKey, job.SpecHash, job.Command, job.Phase, job.Family, job.Model, job.Selector, job.Status, filepath.ToSlash(job.ResultPath), filepath.ToSlash(job.SummaryPath), filepath.ToSlash(job.RawDir), job.StartedAtUTC, job.FinishedAtUTC, job.Error, job.MetadataJSON)
	if err != nil {
		return fmt.Errorf("upsert research job %q/%q: %w", job.RunID, job.JobKey, err)
	}
	return nil
}

func (s *Store) HasCompletedJob(ctx context.Context, runID, specHash string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM research_jobs WHERE run_id = ? AND spec_hash = ? AND status = 'completed'`, runID, specHash).Scan(&count); err != nil {
		return false, fmt.Errorf("lookup completed research job %q/%q: %w", runID, specHash, err)
	}
	return count > 0, nil
}

func (s *Store) DeleteRun(ctx context.Context, runID string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run id is required")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM research_runs WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("delete research run %q: %w", runID, err)
	}
	return nil
}

func (s *Store) RecordArtifact(ctx context.Context, artifact Artifact) error {
	if strings.TrimSpace(artifact.AbsolutePath) == "" {
		return nil
	}
	if artifact.RelativePath == "" {
		artifact.RelativePath = s.relativePath(artifact.AbsolutePath)
	}
	if artifact.CreatedAtUTC == "" {
		artifact.CreatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	if artifact.ChecksumSHA == "" {
		checksum, err := checksumFile(artifact.AbsolutePath)
		if err == nil {
			artifact.ChecksumSHA = checksum
		}
	}
	artifact.MetadataJSON = normalizeJSONText(artifact.MetadataJSON)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_run_artifacts(run_id, job_key, role, relative_path, absolute_path, checksum_sha256, metadata_json, created_at_utc)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id, role, absolute_path) DO UPDATE SET
    job_key = excluded.job_key,
    relative_path = excluded.relative_path,
    checksum_sha256 = excluded.checksum_sha256,
    metadata_json = excluded.metadata_json,
    created_at_utc = excluded.created_at_utc
`, artifact.RunID, artifact.JobKey, artifact.Role, filepath.ToSlash(artifact.RelativePath), filepath.ToSlash(artifact.AbsolutePath), artifact.ChecksumSHA, artifact.MetadataJSON, artifact.CreatedAtUTC)
	if err != nil {
		return fmt.Errorf("record research artifact %q: %w", artifact.AbsolutePath, err)
	}
	return nil
}

func (s *Store) RecordGeneratedPack(ctx context.Context, pack GeneratedPack) error {
	pack.MetadataJSON = normalizeJSONText(pack.MetadataJSON)
	if pack.CreatedAtUTC == "" {
		pack.CreatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_generated_packs(run_id, pack_key, project_folder, source_kind, source_path, output_path, manifest_path, metadata_json, created_at_utc)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(pack_key) DO UPDATE SET
    run_id = excluded.run_id,
    project_folder = excluded.project_folder,
    source_kind = excluded.source_kind,
    source_path = excluded.source_path,
    output_path = excluded.output_path,
    manifest_path = excluded.manifest_path,
    metadata_json = excluded.metadata_json,
    created_at_utc = excluded.created_at_utc
`, pack.RunID, pack.PackKey, pack.ProjectFolder, pack.SourceKind, filepath.ToSlash(pack.SourcePath), filepath.ToSlash(pack.OutputPath), filepath.ToSlash(pack.ManifestPath), pack.MetadataJSON, pack.CreatedAtUTC)
	if err != nil {
		return fmt.Errorf("record research generated pack %q: %w", pack.PackKey, err)
	}
	return nil
}

func (s *Store) RecordExport(ctx context.Context, export Export) error {
	export.MetadataJSON = normalizeJSONText(export.MetadataJSON)
	if export.CreatedAtUTC == "" {
		export.CreatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO research_exports(run_id, export_key, project_folder, benchmark_project_folder, source_pack_id, source_path, output_path, metadata_json, created_at_utc)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(export_key) DO UPDATE SET
    run_id = excluded.run_id,
    project_folder = excluded.project_folder,
    benchmark_project_folder = excluded.benchmark_project_folder,
    source_pack_id = excluded.source_pack_id,
    source_path = excluded.source_path,
    output_path = excluded.output_path,
    metadata_json = excluded.metadata_json,
    created_at_utc = excluded.created_at_utc
`, export.RunID, export.ExportKey, export.ProjectFolder, export.BenchmarkProjectFolder, export.SourcePackID, filepath.ToSlash(export.SourcePath), filepath.ToSlash(export.OutputPath), export.MetadataJSON, export.CreatedAtUTC)
	if err != nil {
		return fmt.Errorf("record research export %q: %w", export.ExportKey, err)
	}
	return nil
}

func (s *Store) DeleteArtifact(ctx context.Context, runID, role, absolutePath string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM research_run_artifacts WHERE run_id = ? AND role = ? AND absolute_path = ?`, runID, role, filepath.ToSlash(strings.TrimSpace(absolutePath)))
	if err != nil {
		return fmt.Errorf("delete research artifact %q/%q/%q: %w", runID, role, absolutePath, err)
	}
	return nil
}

func (s *Store) ListRuns(ctx context.Context) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, command, phase, status, manifest_path, config_fingerprint, started_at_utc, finished_at_utc, error, metadata_json
FROM research_runs
ORDER BY run_id`)
	if err != nil {
		return nil, fmt.Errorf("list research runs: %w", err)
	}
	defer rows.Close()
	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(
			&run.RunID, &run.Command, &run.Phase, &run.Status, &run.ManifestPath, &run.ConfigFingerprint,
			&run.StartedAtUTC, &run.FinishedAtUTC, &run.Error, &run.MetadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan research run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research runs: %w", err)
	}
	return runs, nil
}

func (s *Store) ListJobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, job_key, spec_hash, command, phase, family, model, selector, status, result_path, summary_path, raw_dir, started_at_utc, finished_at_utc, error, metadata_json
FROM research_jobs
ORDER BY run_id, job_key`)
	if err != nil {
		return nil, fmt.Errorf("list research jobs: %w", err)
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var job Job
		if err := rows.Scan(
			&job.RunID, &job.JobKey, &job.SpecHash, &job.Command, &job.Phase, &job.Family, &job.Model, &job.Selector,
			&job.Status, &job.ResultPath, &job.SummaryPath, &job.RawDir, &job.StartedAtUTC, &job.FinishedAtUTC,
			&job.Error, &job.MetadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan research job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research jobs: %w", err)
	}
	return jobs, nil
}

func (s *Store) ListArtifacts(ctx context.Context) ([]Artifact, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, job_key, role, relative_path, absolute_path, checksum_sha256, metadata_json, created_at_utc
FROM research_run_artifacts
ORDER BY run_id, role, absolute_path`)
	if err != nil {
		return nil, fmt.Errorf("list research artifacts: %w", err)
	}
	defer rows.Close()
	var artifacts []Artifact
	for rows.Next() {
		var artifact Artifact
		if err := rows.Scan(
			&artifact.RunID, &artifact.JobKey, &artifact.Role, &artifact.RelativePath, &artifact.AbsolutePath,
			&artifact.ChecksumSHA, &artifact.MetadataJSON, &artifact.CreatedAtUTC,
		); err != nil {
			return nil, fmt.Errorf("scan research artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research artifacts: %w", err)
	}
	return artifacts, nil
}

func (s *Store) ListGeneratedPacks(ctx context.Context) ([]GeneratedPack, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, pack_key, project_folder, source_kind, source_path, output_path, manifest_path, metadata_json, created_at_utc
FROM research_generated_packs
ORDER BY pack_key`)
	if err != nil {
		return nil, fmt.Errorf("list research generated packs: %w", err)
	}
	defer rows.Close()
	var packs []GeneratedPack
	for rows.Next() {
		var pack GeneratedPack
		if err := rows.Scan(
			&pack.RunID, &pack.PackKey, &pack.ProjectFolder, &pack.SourceKind, &pack.SourcePath,
			&pack.OutputPath, &pack.ManifestPath, &pack.MetadataJSON, &pack.CreatedAtUTC,
		); err != nil {
			return nil, fmt.Errorf("scan research generated pack: %w", err)
		}
		packs = append(packs, pack)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research generated packs: %w", err)
	}
	return packs, nil
}

func (s *Store) ListExports(ctx context.Context) ([]Export, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, export_key, project_folder, benchmark_project_folder, source_pack_id, source_path, output_path, metadata_json, created_at_utc
FROM research_exports
ORDER BY export_key`)
	if err != nil {
		return nil, fmt.Errorf("list research exports: %w", err)
	}
	defer rows.Close()
	var exports []Export
	for rows.Next() {
		var export Export
		if err := rows.Scan(
			&export.RunID, &export.ExportKey, &export.ProjectFolder, &export.BenchmarkProjectFolder, &export.SourcePackID,
			&export.SourcePath, &export.OutputPath, &export.MetadataJSON, &export.CreatedAtUTC,
		); err != nil {
			return nil, fmt.Errorf("scan research export: %w", err)
		}
		exports = append(exports, export)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research exports: %w", err)
	}
	return exports, nil
}

func (s *Store) RewritePaths(ctx context.Context, mapping map[string]string) error {
	if len(mapping) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research path rewrite tx: %w", err)
	}
	defer tx.Rollback()

	for oldPath, newPath := range normalizePathMapping(mapping) {
		newRel := s.relativePath(newPath)
		if _, err := tx.ExecContext(ctx, `UPDATE research_runs SET manifest_path = ? WHERE manifest_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_runs.manifest_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_jobs SET result_path = ? WHERE result_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_jobs.result_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_jobs SET summary_path = ? WHERE summary_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_jobs.summary_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_jobs SET raw_dir = ? WHERE raw_dir = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_jobs.raw_dir %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_run_artifacts SET absolute_path = ?, relative_path = ? WHERE absolute_path = ?`, newPath, newRel, oldPath); err != nil {
			return fmt.Errorf("rewrite research_run_artifacts.absolute_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_generated_packs SET source_path = ? WHERE source_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_generated_packs.source_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_generated_packs SET output_path = ? WHERE output_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_generated_packs.output_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_generated_packs SET manifest_path = ? WHERE manifest_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_generated_packs.manifest_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_exports SET source_path = ? WHERE source_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_exports.source_path %q: %w", oldPath, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE research_exports SET output_path = ? WHERE output_path = ?`, newPath, oldPath); err != nil {
			return fmt.Errorf("rewrite research_exports.output_path %q: %w", oldPath, err)
		}
	}

	for _, table := range []string{
		"research_runs",
		"research_jobs",
		"research_run_artifacts",
		"research_generated_packs",
		"research_exports",
	} {
		if err := rewriteMetadataJSONPaths(ctx, tx, table, mapping); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research path rewrite tx: %w", err)
	}
	return nil
}

func normalizeJSONText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	return raw
}

func (s *Store) relativePath(path string) string {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(s.workspaceRoot) == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(s.workspaceRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func checksumFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func MetadataJSON(value any) string {
	if value == nil {
		return "{}"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func normalizePathMapping(mapping map[string]string) map[string]string {
	normalized := make(map[string]string, len(mapping))
	for oldPath, newPath := range mapping {
		oldPath = filepath.ToSlash(strings.TrimSpace(oldPath))
		newPath = filepath.ToSlash(strings.TrimSpace(newPath))
		if oldPath == "" || newPath == "" || oldPath == newPath {
			continue
		}
		normalized[oldPath] = newPath
	}
	return normalized
}

func rewriteMetadataJSONPaths(ctx context.Context, tx *sql.Tx, table string, mapping map[string]string) error {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT rowid, metadata_json FROM %s`, table))
	if err != nil {
		return fmt.Errorf("query %s metadata json: %w", table, err)
	}
	defer rows.Close()
	var updates []struct {
		rowID int64
		value string
	}
	for rows.Next() {
		var rowID int64
		var value string
		if err := rows.Scan(&rowID, &value); err != nil {
			return fmt.Errorf("scan %s metadata json: %w", table, err)
		}
		rewritten := rewriteMappedText(value, mapping)
		if rewritten != value {
			updates = append(updates, struct {
				rowID int64
				value string
			}{rowID: rowID, value: rewritten})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s metadata json: %w", table, err)
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET metadata_json = ? WHERE rowid = ?`, table), update.value, update.rowID); err != nil {
			return fmt.Errorf("update %s metadata json row %d: %w", table, update.rowID, err)
		}
	}
	return nil
}

func rewriteMappedText(value string, mapping map[string]string) string {
	type pair struct {
		old string
		new string
	}
	pairs := make([]pair, 0, len(mapping))
	for oldPath, newPath := range normalizePathMapping(mapping) {
		pairs = append(pairs, pair{old: oldPath, new: newPath})
	}
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if len(pairs[j].old) > len(pairs[i].old) {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	rewritten := value
	for _, pair := range pairs {
		rewritten = strings.ReplaceAll(rewritten, pair.old, pair.new)
		rewritten = strings.ReplaceAll(rewritten, strings.ReplaceAll(pair.old, "/", `\`), strings.ReplaceAll(pair.new, "/", `\`))
	}
	return rewritten
}
