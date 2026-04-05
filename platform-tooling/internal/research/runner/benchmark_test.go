package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func TestBenchmarkContractsSubcommand(t *testing.T) {
	tests := []struct {
		phase string
		want  string
	}{
		{phase: "phase1", want: "run-hilbert-benchmark"},
		{phase: "phase2-direct", want: "run-hilbert-benchmark"},
		{phase: "nd", want: "run-nd-hilbert-benchmark"},
		{phase: "phase2-nd", want: "run-nd-hilbert-benchmark"},
		{phase: "phase3-nd", want: "run-nd-hilbert-benchmark"},
	}

	for _, tc := range tests {
		if got := benchmarkContractsSubcommand(tc.phase); got != tc.want {
			t.Fatalf("benchmarkContractsSubcommand(%q) = %q, want %q", tc.phase, got, tc.want)
		}
	}
}

func TestFinalizeBenchmarkRunMarksInterruptedRunAborted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	workspace := t.TempDir()
	dbPath := filepath.Join(workspace, "research.sqlite")
	store, err := state.Open(context.Background(), workspace, dbPath)
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	defer store.Close()

	plan := &planner.BenchmarkPlan{
		Command:           "phase1 run",
		Phase:             "phase1",
		RunID:             "phase1-interrupted",
		ManifestPath:      filepath.Join(workspace, "manifest.json"),
		ConfigFingerprint: "fingerprint",
	}
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:             plan.RunID,
		Command:           plan.Command,
		Phase:             plan.Phase,
		Status:            "running",
		ManifestPath:      plan.ManifestPath,
		ConfigFingerprint: plan.ConfigFingerprint,
	}); err != nil {
		t.Fatalf("UpsertRun() error = %v", err)
	}

	result, err := finalizeBenchmarkRun(ctx, store, plan, BenchmarkRunResult{Plan: plan}, context.Canceled, &benchmarkProgress{})
	if err == nil {
		t.Fatalf("finalizeBenchmarkRun() error = nil, want context cancellation")
	}
	if result.Status != "aborted" {
		t.Fatalf("result.Status = %q, want aborted", result.Status)
	}

	db, err := researchdb.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("researchdb.Open() error = %v", err)
	}
	defer db.Close()

	var status string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM research_runs WHERE run_id = ?`, plan.RunID).Scan(&status); err != nil {
		t.Fatalf("QueryRow(status) error = %v", err)
	}
	if status != "aborted" {
		t.Fatalf("stored status = %q, want aborted", status)
	}
}

func TestRunBenchmarkRunsJobsConcurrentlyWhenJobsGreaterThanOne(t *testing.T) {
	workspace := t.TempDir()
	store := openBenchmarkTestStore(t, workspace)
	plan := newBenchmarkTestPlan(workspace, "parallel", 3)
	loaded := researchconfig.Loaded{WorkspaceRoot: workspace, ConfigPath: filepath.Join(workspace, "research.yaml"), LocalConfigPath: filepath.Join(workspace, "local.yaml"), Fingerprint: "fingerprint"}

	restore := stubBenchmarkExecution(func(ctx context.Context, _ string, job planner.BenchmarkJob) (string, error) {
		return simulateBenchmarkJob(ctx, job, 75*time.Millisecond)
	})
	defer restore()

	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	restoreConcurrencyProbe := wrapBenchmarkExecution(func(ctx context.Context, workspaceRoot string, job planner.BenchmarkJob, next func(context.Context, string, planner.BenchmarkJob) (string, error)) (string, error) {
		current := inFlight.Add(1)
		for {
			observed := maxInFlight.Load()
			if current <= observed || maxInFlight.CompareAndSwap(observed, current) {
				break
			}
		}
		defer inFlight.Add(-1)
		return next(ctx, workspaceRoot, job)
	})
	defer restoreConcurrencyProbe()

	result, err := RunBenchmark(context.Background(), loaded, store, plan, BenchmarkRunOptions{Jobs: 2})
	if err != nil {
		t.Fatalf("RunBenchmark() error = %v", err)
	}
	if result.CompletedJobs != 3 {
		t.Fatalf("CompletedJobs = %d, want 3", result.CompletedJobs)
	}
	if result.ParallelJobs != 2 {
		t.Fatalf("ParallelJobs = %d, want 2", result.ParallelJobs)
	}
	if maxInFlight.Load() < 2 {
		t.Fatalf("max concurrent jobs = %d, want at least 2", maxInFlight.Load())
	}
}

func TestRunBenchmarkBestEffortContinuesOtherJobsUnderConcurrency(t *testing.T) {
	workspace := t.TempDir()
	store := openBenchmarkTestStore(t, workspace)
	plan := newBenchmarkTestPlan(workspace, "best-effort", 3)
	loaded := researchconfig.Loaded{WorkspaceRoot: workspace, ConfigPath: filepath.Join(workspace, "research.yaml"), LocalConfigPath: filepath.Join(workspace, "local.yaml"), Fingerprint: "fingerprint"}

	var startedMu sync.Mutex
	started := make([]string, 0, len(plan.Jobs))
	restore := stubBenchmarkExecution(func(ctx context.Context, _ string, job planner.BenchmarkJob) (string, error) {
		startedMu.Lock()
		started = append(started, job.Key)
		startedMu.Unlock()
		if job.Key == "job-02" {
			return "synthetic failure", fmt.Errorf("synthetic failure")
		}
		return simulateBenchmarkJob(ctx, job, 50*time.Millisecond)
	})
	defer restore()

	result, err := RunBenchmark(context.Background(), loaded, store, plan, BenchmarkRunOptions{Jobs: 2, BestEffort: true})
	if err != nil {
		t.Fatalf("RunBenchmark() error = %v, want nil in best-effort mode", err)
	}
	if result.Status != "partial_failed" {
		t.Fatalf("Status = %q, want partial_failed", result.Status)
	}
	if result.CompletedJobs != 2 {
		t.Fatalf("CompletedJobs = %d, want 2", result.CompletedJobs)
	}
	if result.FailedJobs != 1 {
		t.Fatalf("FailedJobs = %d, want 1", result.FailedJobs)
	}
	startedMu.Lock()
	defer startedMu.Unlock()
	if len(started) != 3 {
		t.Fatalf("started jobs = %v, want all 3 jobs to start", started)
	}
}

func TestRunBenchmarkFailFastCancelsQueuedJobsUnderConcurrency(t *testing.T) {
	workspace := t.TempDir()
	store := openBenchmarkTestStore(t, workspace)
	plan := newBenchmarkTestPlan(workspace, "fail-fast", 3)
	loaded := researchconfig.Loaded{WorkspaceRoot: workspace, ConfigPath: filepath.Join(workspace, "research.yaml"), LocalConfigPath: filepath.Join(workspace, "local.yaml"), Fingerprint: "fingerprint"}

	var startedMu sync.Mutex
	started := make([]string, 0, len(plan.Jobs))
	restore := stubBenchmarkExecution(func(ctx context.Context, _ string, job planner.BenchmarkJob) (string, error) {
		startedMu.Lock()
		started = append(started, job.Key)
		startedMu.Unlock()
		if job.Key == "job-01" {
			return "synthetic fail-fast failure", fmt.Errorf("synthetic fail-fast failure")
		}
		select {
		case <-ctx.Done():
			return "canceled", ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return simulateBenchmarkJob(context.Background(), job, 0)
		}
	})
	defer restore()

	result, err := RunBenchmark(context.Background(), loaded, store, plan, BenchmarkRunOptions{Jobs: 2})
	if err == nil {
		t.Fatalf("RunBenchmark() error = nil, want fail-fast error")
	}
	if result.FailedJobs+result.AbortedJobs == 0 {
		t.Fatalf("expected failed or aborted jobs, got result=%+v", result)
	}
	startedMu.Lock()
	defer startedMu.Unlock()
	if slices.Contains(started, "job-03") {
		t.Fatalf("queued job-03 started despite fail-fast cancellation: %v", started)
	}
}

func openBenchmarkTestStore(t *testing.T, workspace string) *state.Store {
	t.Helper()
	store, err := state.Open(context.Background(), workspace, filepath.Join(workspace, "research.sqlite"))
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func newBenchmarkTestPlan(workspace, runID string, jobs int) *planner.BenchmarkPlan {
	plan := &planner.BenchmarkPlan{
		Command:           "phase1 run",
		Phase:             "phase1",
		RunID:             runID,
		ManifestPath:      filepath.Join(workspace, "manifests", runID+".json"),
		SummaryPath:       filepath.Join(workspace, "summaries", runID+".csv"),
		ReportPath:        filepath.Join(workspace, "reports", runID+".md"),
		ModelCatalogPath:  filepath.Join(workspace, "manifests", runID+"_catalog.csv"),
		ConfigFingerprint: "fingerprint",
		ProjectFolder:     "test-project",
		InputKind:         "benchmark_cases_csv",
		InputFile:         "cases.csv",
		InputPath:         filepath.Join(workspace, "cases.csv"),
	}
	if err := os.MkdirAll(filepath.Dir(plan.SummaryPath), 0o755); err == nil {
		_ = os.WriteFile(plan.SummaryPath, []byte("run_id\n"), 0o644)
	}
	for idx := 1; idx <= jobs; idx++ {
		key := fmt.Sprintf("job-%02d", idx)
		plan.Jobs = append(plan.Jobs, planner.BenchmarkJob{
			Key:         key,
			SpecHash:    key,
			Command:     "phase1 run",
			Phase:       "phase1",
			RunID:       fmt.Sprintf("%s_%s", runID, key),
			Family:      "test-family",
			Model:       key,
			Selector:    "cases.csv",
			Provider:    "compatible",
			BaseURL:     "http://localhost:11434",
			Timeout:     "30s",
			ResultsPath: filepath.Join(workspace, "results", key+".csv"),
			SummaryPath: plan.SummaryPath,
			RawDir:      filepath.Join(workspace, "raw", key),
		})
	}
	return plan
}

func stubBenchmarkExecution(stub func(context.Context, string, planner.BenchmarkJob) (string, error)) func() {
	previousRun := runBenchmarkJobFunc
	previousCatalog := ensureBenchmarkModelCatalog
	previousReport := generateBenchmarkReport
	runBenchmarkJobFunc = stub
	ensureBenchmarkModelCatalog = func(path string, _ []planner.BenchmarkJob) error {
		return os.WriteFile(path, []byte("model,provider\n"), 0o644)
	}
	generateBenchmarkReport = func(_ context.Context, _ string, _, _, _, _, outPath string) error {
		return os.WriteFile(outPath, []byte("# report\n"), 0o644)
	}
	return func() {
		runBenchmarkJobFunc = previousRun
		ensureBenchmarkModelCatalog = previousCatalog
		generateBenchmarkReport = previousReport
	}
}

func wrapBenchmarkExecution(wrapper func(context.Context, string, planner.BenchmarkJob, func(context.Context, string, planner.BenchmarkJob) (string, error)) (string, error)) func() {
	previous := runBenchmarkJobFunc
	runBenchmarkJobFunc = func(ctx context.Context, workspaceRoot string, job planner.BenchmarkJob) (string, error) {
		return wrapper(ctx, workspaceRoot, job, previous)
	}
	return func() {
		runBenchmarkJobFunc = previous
	}
}

func simulateBenchmarkJob(ctx context.Context, job planner.BenchmarkJob, delay time.Duration) (string, error) {
	if delay > 0 {
		select {
		case <-ctx.Done():
			return "canceled", ctx.Err()
		case <-time.After(delay):
		}
	}
	if err := os.MkdirAll(filepath.Dir(job.ResultsPath), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(job.SummaryPath), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(job.RawDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(job.ResultsPath, []byte("case_id,status\n"), 0o644); err != nil {
		return "", err
	}
	return "ok", nil
}
