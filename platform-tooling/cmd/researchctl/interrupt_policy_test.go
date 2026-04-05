package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/runner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func TestMaybeDropInterruptedBenchmarkRunDropsNoResultRun(t *testing.T) {
	workspace := t.TempDir()
	researchctlProcessContext = context.Background()

	manifestPath := filepath.Join(workspace, "research", "artifacts", "result_research", "manifests", "phase1_test.json")
	summaryPath := filepath.Join(workspace, "research", "artifacts", "result_research", "waves", "phase1_test.csv")
	reportPath := filepath.Join(workspace, "research", "result_research_report", "direct", "phase1_test.md")
	modelCatalogPath := filepath.Join(workspace, "research", "artifacts", "result_research", "manifests", "phase1_test_model_catalog.csv")
	resultPath := filepath.Join(workspace, "research", "artifacts", "bench", "result", "result_run.csv")
	rawRunDir := filepath.Join(workspace, "research", "artifacts", "bench", "raw", "run-1")
	for _, dir := range []string{filepath.Dir(manifestPath), filepath.Dir(summaryPath), filepath.Dir(reportPath), filepath.Dir(modelCatalogPath), filepath.Dir(resultPath), rawRunDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	for _, file := range []string{manifestPath, resultPath} {
		if err := os.WriteFile(file, []byte("header\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", file, err)
		}
	}
	if err := os.WriteFile(filepath.Join(rawRunDir, "V2E01.txt"), []byte("timeout"), 0o644); err != nil {
		t.Fatalf("WriteFile(raw) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "research.sqlite")
	store, err := state.Open(context.Background(), workspace, dbPath)
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	defer store.Close()
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:        "run-1",
		Command:      "phase1 run",
		Phase:        "phase1",
		Status:       "aborted",
		ManifestPath: manifestPath,
	}); err != nil {
		t.Fatalf("UpsertRun() error = %v", err)
	}

	loaded := researchconfig.Loaded{WorkspaceRoot: workspace}
	plan := &planner.BenchmarkPlan{
		RunID:            "run-1",
		ManifestPath:     manifestPath,
		SummaryPath:      summaryPath,
		ReportPath:       reportPath,
		ModelCatalogPath: modelCatalogPath,
		InterruptPolicy:  researchconfig.InterruptPolicyDropIfNoResults,
		Jobs: []planner.BenchmarkJob{
			{
				RunID:       "run-1",
				ResultsPath: resultPath,
				RawDir:      filepath.Dir(rawRunDir),
			},
		},
	}

	dropped, err := maybeDropInterruptedBenchmarkRun(loaded, store, plan, runner.BenchmarkRunResult{Plan: plan}, nil)
	if err != nil {
		t.Fatalf("maybeDropInterruptedBenchmarkRun() error = %v", err)
	}
	if !dropped {
		t.Fatalf("dropped = false, want true")
	}

	db, err := researchdb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("researchdb.Open() error = %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM research_runs WHERE run_id = 'run-1'`).Scan(&count); err != nil {
		t.Fatalf("count run error = %v", err)
	}
	if count != 0 {
		t.Fatalf("run count = %d, want 0", count)
	}
	for _, target := range []string{manifestPath, resultPath, rawRunDir} {
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("target %q still exists after cleanup", target)
		}
	}
}

func TestMaybeDropInterruptedBenchmarkRunKeepsMeaningfulOutputs(t *testing.T) {
	workspace := t.TempDir()
	researchctlProcessContext = context.Background()

	summaryPath := filepath.Join(workspace, "research", "artifacts", "result_research", "waves", "phase1_test.csv")
	resultPath := filepath.Join(workspace, "research", "artifacts", "bench", "result", "result_run.csv")
	for _, dir := range []string{filepath.Dir(summaryPath), filepath.Dir(resultPath)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	if err := os.WriteFile(summaryPath, []byte("header\nvalue\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(summary) error = %v", err)
	}
	if err := os.WriteFile(resultPath, []byte("header\nvalue\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(result) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "research.sqlite")
	store, err := state.Open(context.Background(), workspace, dbPath)
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	defer store.Close()
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:   "run-2",
		Command: "phase1 run",
		Phase:   "phase1",
		Status:  "aborted",
	}); err != nil {
		t.Fatalf("UpsertRun() error = %v", err)
	}

	loaded := researchconfig.Loaded{WorkspaceRoot: workspace}
	plan := &planner.BenchmarkPlan{
		RunID:           "run-2",
		SummaryPath:     summaryPath,
		InterruptPolicy: researchconfig.InterruptPolicyDropIfNoResults,
		Jobs: []planner.BenchmarkJob{
			{ResultsPath: resultPath},
		},
	}
	dropped, err := maybeDropInterruptedBenchmarkRun(loaded, store, plan, runner.BenchmarkRunResult{Plan: plan, CompletedJobs: 1}, nil)
	if err != nil {
		t.Fatalf("maybeDropInterruptedBenchmarkRun() error = %v", err)
	}
	if dropped {
		t.Fatalf("dropped = true, want false")
	}
}
