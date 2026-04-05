package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func TestGenerateLean4HypothesisCasesCommand(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	dbPath := filepath.Join(root, "research_db.sqlite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runGenerateLean4HypothesisCasesCommand([]string{
		"-project-folder", "ignored-for-test",
		"-artifact-root", root,
		"-project-dir", projectDir,
		"-db-path", dbPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runGenerateLean4HypothesisCasesCommand() error = %v; stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "theorems", "theorems.csv")); err != nil {
		t.Fatalf("Stat(theorems.csv) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "theorems", "historical")); err != nil {
		t.Fatalf("Stat(theorem historical dir) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "theorems", "generation-config.json")); err != nil {
		t.Fatalf("Stat(theorem generation-config.json) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "cases", "generation-config.json")); err != nil {
		t.Fatalf("Stat(generation-config.json) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "lakefile.lean")); err != nil {
		t.Fatalf("Stat(lakefile.lean) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "lean4 theorem backlog generated") {
		t.Fatalf("stdout missing success marker:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "generation_mode: enumerator") {
		t.Fatalf("stdout missing enumerator mode:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "generation_job_id:") {
		t.Fatalf("stdout missing generation_job_id:\n%s", stdout.String())
	}
	db, err := researchdb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("researchdb.Open() error = %v", err)
	}
	defer db.Close()
	var jobCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM research_generation_jobs`).Scan(&jobCount); err != nil {
		t.Fatalf("query research_generation_jobs count: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("research_generation_jobs count = %d, want 1", jobCount)
	}
	var setCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM research_hypothesis_sets`).Scan(&setCount); err != nil {
		t.Fatalf("query research_hypothesis_sets count: %v", err)
	}
	if setCount != 1 {
		t.Fatalf("research_hypothesis_sets count = %d, want 1", setCount)
	}
	var theoremCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM research_theorems`).Scan(&theoremCount); err != nil {
		t.Fatalf("query research_theorems count: %v", err)
	}
	if theoremCount == 0 {
		t.Fatalf("research_theorems count = 0, want > 0")
	}
}

func TestExportLean4HypothesisCasesCommand_DBFirst(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	dbPath := filepath.Join(root, "research_db.sqlite")
	outputPath := filepath.Join(root, "benchmark", "theorems", "lean4-generated.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runGenerateLean4HypothesisCasesCommand([]string{
		"-project-folder", "ignored-for-test",
		"-artifact-root", root,
		"-project-dir", projectDir,
		"-db-path", dbPath,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("runGenerateLean4HypothesisCasesCommand() error = %v; stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := runExportLean4HypothesisCasesCommand([]string{
		"-project-folder", "ignored-for-test",
		"-db-path", dbPath,
		"-benchmark-project-folder", "benchmark-test",
		"-output-file", outputPath,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("runExportLean4HypothesisCasesCommand() error = %v; stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Stat(outputPath) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "benchmark", "theorems", "historical")); err != nil {
		t.Fatalf("Stat(historical export dir) error = %v", err)
	}
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Open(outputPath) error = %v", err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll(outputPath) error = %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("len(records) = %d, want at least 2", len(records))
	}
	if got := records[0][4]; got != "label" {
		t.Fatalf("header[4] = %q, want label", got)
	}
	if !strings.Contains(stdout.String(), "hypothesis_set_id:") {
		t.Fatalf("stdout missing hypothesis_set_id:\n%s", stdout.String())
	}

	db, err := researchdb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("researchdb.Open() error = %v", err)
	}
	defer db.Close()
	var snapshotCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM research_hypothesis_export_snapshots WHERE status = 'completed'`).Scan(&snapshotCount); err != nil {
		t.Fatalf("query export snapshots count: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("completed export snapshots = %d, want 1", snapshotCount)
	}
}

func TestExportLean4HypothesisCasesCommand_LegacySourceFile(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "research_db.sqlite")
	sourcePath, _, _, err := lean4worker.GenerateHypothesisArtifacts(filepath.Join(root, "legacy"), lean4worker.DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisArtifacts() error = %v", err)
	}
	outputPath := filepath.Join(root, "benchmark", "theorems", "legacy-generated.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runExportLean4HypothesisCasesCommand([]string{
		"-project-folder", "ignored-for-test",
		"-db-path", dbPath,
		"-source-file", sourcePath,
		"-benchmark-project-folder", "benchmark-test",
		"-output-file", outputPath,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("runExportLean4HypothesisCasesCommand() error = %v; stderr=%s", err, stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Stat(outputPath) error = %v", err)
	}
	if !strings.Contains(stdout.String(), filepath.ToSlash(sourcePath)) {
		t.Fatalf("stdout missing explicit source file:\n%s", stdout.String())
	}
}

func TestRunLean4ResearchJobCommand_MissingLakeBinary(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	resultRoot := filepath.Join(root, "result")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runLean4ResearchJobCommand([]string{
		"-project-folder", "ignored-for-test",
		"-project-dir", projectDir,
		"-result-root", resultRoot,
		"-lake-binary", "definitely-not-a-real-lake-binary",
		"-name", "identity",
		"-statement", "P -> P",
		"-run-id", "run-test",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runLean4ResearchJobCommand() error = %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok: false") {
		t.Fatalf("stdout missing failure state:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(resultRoot, "run-test", "identity", "report.json")); err != nil {
		t.Fatalf("Stat(report.json) error = %v", err)
	}
}
