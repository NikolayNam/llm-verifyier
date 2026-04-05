package researchdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
)

func TestPersistHypothesisGenerationRoundTrip(t *testing.T) {
	workspace := t.TempDir()
	project := "lean4-db-first-test"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	cases, normalized, err := lean4worker.GenerateHypothesisCases(lean4worker.DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisCases() error = %v", err)
	}
	createdAt := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	theoremPackID, finalized := lean4worker.FinalizeHypothesisCases(cases, normalized.Mode, createdAt)
	supportArtifacts := lean4worker.HypothesisSupportArtifactPaths(artifactRoot)

	persisted, err := PersistHypothesisGeneration(context.Background(), db, PersistHypothesisGenerationInput{
		WorkspaceRoot:        workspace,
		ProjectFolder:        project,
		ArtifactRoot:         artifactRoot,
		Config:               normalized,
		Cases:                finalized,
		GenerationConfigPath: supportArtifacts.GenerationConfigPath,
		DefaultPayloadPath:   supportArtifacts.DefaultPayloadPath,
		CreatedAt:            createdAt,
	})
	if err != nil {
		t.Fatalf("PersistHypothesisGeneration() error = %v", err)
	}
	if persisted.TheoremPackID != theoremPackID {
		t.Fatalf("PersistHypothesisGeneration().TheoremPackID = %q, want %q", persisted.TheoremPackID, theoremPackID)
	}
	if err := CompleteGenerationJob(context.Background(), db, persisted.GenerationJobID, createdAt.Add(time.Minute)); err != nil {
		t.Fatalf("CompleteGenerationJob() error = %v", err)
	}

	resolved, err := ResolveHypothesisSet(context.Background(), db, HypothesisSetSelector{
		ProjectFolder: project,
	})
	if err != nil {
		t.Fatalf("ResolveHypothesisSet() error = %v", err)
	}
	if resolved.HypothesisSetID != persisted.HypothesisSetID {
		t.Fatalf("ResolveHypothesisSet().HypothesisSetID = %d, want %d", resolved.HypothesisSetID, persisted.HypothesisSetID)
	}

	loaded, err := LoadHypothesisCasesBySet(context.Background(), db, resolved.HypothesisSetID)
	if err != nil {
		t.Fatalf("LoadHypothesisCasesBySet() error = %v", err)
	}
	if len(loaded) != len(finalized) {
		t.Fatalf("len(loaded) = %d, want %d", len(loaded), len(finalized))
	}
	if loaded[0].TheoremPackID != theoremPackID {
		t.Fatalf("loaded[0].TheoremPackID = %q, want %q", loaded[0].TheoremPackID, theoremPackID)
	}
	if loaded[0].CaseID != finalized[0].CaseID {
		t.Fatalf("loaded[0].CaseID = %q, want %q", loaded[0].CaseID, finalized[0].CaseID)
	}

	assertCount(t, db, "SELECT COUNT(*) FROM research_generation_jobs", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_hypothesis_sets", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_theorem_files", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM research_theorems", len(finalized))
	assertDBObjectAbsent(t, db, "research_hypotheses")
}

func TestGenerationAndExportFailureStatuses(t *testing.T) {
	workspace := t.TempDir()
	project := "lean4-db-failure-test"
	artifactRoot := filepath.Join(workspace, "research", "artifacts", project)
	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	cases, normalized, err := lean4worker.GenerateHypothesisCases(lean4worker.DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisCases() error = %v", err)
	}
	createdAt := time.Date(2026, 3, 30, 13, 0, 0, 0, time.UTC)
	_, finalized := lean4worker.FinalizeHypothesisCases(cases, normalized.Mode, createdAt)
	supportArtifacts := lean4worker.HypothesisSupportArtifactPaths(artifactRoot)

	persisted, err := PersistHypothesisGeneration(context.Background(), db, PersistHypothesisGenerationInput{
		WorkspaceRoot:        workspace,
		ProjectFolder:        project,
		ArtifactRoot:         artifactRoot,
		Config:               normalized,
		Cases:                finalized,
		GenerationConfigPath: supportArtifacts.GenerationConfigPath,
		DefaultPayloadPath:   supportArtifacts.DefaultPayloadPath,
		CreatedAt:            createdAt,
	})
	if err != nil {
		t.Fatalf("PersistHypothesisGeneration() error = %v", err)
	}
	if err := FailGenerationJob(context.Background(), db, persisted.GenerationJobID, "support artifact write failed", createdAt.Add(time.Minute)); err != nil {
		t.Fatalf("FailGenerationJob() error = %v", err)
	}

	var generationStatus string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM research_generation_jobs WHERE id = ?`, persisted.GenerationJobID).Scan(&generationStatus); err != nil {
		t.Fatalf("query generation status: %v", err)
	}
	if generationStatus != "artifact_write_failed" {
		t.Fatalf("generation status = %q, want artifact_write_failed", generationStatus)
	}

	if err := CompleteGenerationJob(context.Background(), db, persisted.GenerationJobID, createdAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("CompleteGenerationJob() error = %v", err)
	}
	snapshotID, err := StartExportSnapshot(context.Background(), db, ExportSnapshotInput{
		WorkspaceRoot:          workspace,
		ResearchID:             persisted.ResearchID,
		HypothesisSetID:        persisted.HypothesisSetID,
		TheoremPackID:          persisted.TheoremPackID,
		BenchmarkProjectFolder: "benchmark-test",
		ModeFilter:             "*",
		OutputPath:             filepath.Join(workspace, "research", "artifacts", "benchmark-test", "theorems", "lean4-generated.csv"),
		CreatedAt:              createdAt.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("StartExportSnapshot() error = %v", err)
	}
	if err := FailExportSnapshot(context.Background(), db, snapshotID, "benchmark csv write failed", createdAt.Add(4*time.Minute)); err != nil {
		t.Fatalf("FailExportSnapshot() error = %v", err)
	}

	var exportStatus string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM research_hypothesis_export_snapshots WHERE id = ?`, snapshotID).Scan(&exportStatus); err != nil {
		t.Fatalf("query export status: %v", err)
	}
	if exportStatus != "write_failed" {
		t.Fatalf("export status = %q, want write_failed", exportStatus)
	}
}

func TestLoadHypothesisCasesBySetRequiresCanonicalStorage(t *testing.T) {
	workspace := t.TempDir()
	project := "lean4-db-canonical-required"
	dbPath := filepath.Join(workspace, "research", "artifacts", "result_research", "research_db", "research_db.sqlite")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	researchID, err := ensureResearch(context.Background(), db, project, filepath.ToSlash(filepath.Join("research", "artifacts", project)))
	if err != nil {
		t.Fatalf("ensureResearch() error = %v", err)
	}
	createdAt := "2026-04-01T12:00:00Z"
	jobResult, err := db.ExecContext(context.Background(), `
INSERT INTO research_generation_jobs(
    research_id, generation_mode, generated_count, artifact_root_relative_path, status, created_at_utc, completed_at_utc
) VALUES (?, 'enumerator', 2, ?, 'completed', ?, ?)
`, researchID, filepath.ToSlash(filepath.Join("research", "artifacts", project)), createdAt, createdAt)
	if err != nil {
		t.Fatalf("insert research_generation_jobs error = %v", err)
	}
	generationJobID, err := jobResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}
	setResult, err := db.ExecContext(context.Background(), `
INSERT INTO research_hypothesis_sets(
    research_id, generation_job_id, theorem_pack_id, generation_mode, logic_fragment, hypothesis_count, created_at_utc
) VALUES (?, ?, 'legacy_pack_1', 'enumerator', 'implicational-prop-v1', 2, ?)
`, researchID, generationJobID, createdAt)
	if err != nil {
		t.Fatalf("insert research_hypothesis_sets error = %v", err)
	}
	hypothesisSetID, err := setResult.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}
	_, err = LoadHypothesisCasesBySet(context.Background(), db, hypothesisSetID)
	if err == nil {
		t.Fatalf("LoadHypothesisCasesBySet() error = nil, want canonical-storage failure")
	}
	if !strings.Contains(err.Error(), "not linked to canonical theorem-backed storage") {
		t.Fatalf("LoadHypothesisCasesBySet() error = %v, want canonical-storage failure", err)
	}
}

func assertDBObjectAbsent(t *testing.T, db *sql.DB, objectName string) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM sqlite_master WHERE name = ?`, objectName).Scan(&count); err != nil {
		t.Fatalf("QueryRowContext(%q) error = %v", objectName, err)
	}
	if count != 0 {
		t.Fatalf("sqlite object %q count = %d, want 0", objectName, count)
	}
}
