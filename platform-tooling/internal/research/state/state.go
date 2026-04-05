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
