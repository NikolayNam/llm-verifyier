package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/report"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

type BenchmarkRunOptions struct {
	Resume         bool
	SkipExisting   bool
	BestEffort     bool
	Jobs           int
	RequireSecrets bool
	ProgressWriter io.Writer
}

type BenchmarkRunResult struct {
	Plan          *planner.BenchmarkPlan
	Status        string
	CompletedJobs int
	SkippedJobs   int
	FailedJobs    int
	AbortedJobs   int
	ParallelJobs  int
	ReportPath    string
}

var (
	runBenchmarkJobFunc         = runBenchmarkJob
	ensureBenchmarkModelCatalog = report.EnsureModelCatalog
	generateBenchmarkReport     = report.GenerateBenchmarkReport
)

func RunBenchmark(ctx context.Context, loaded researchconfig.Loaded, store *state.Store, plan *planner.BenchmarkPlan, opts BenchmarkRunOptions) (BenchmarkRunResult, error) {
	if plan == nil {
		return BenchmarkRunResult{}, fmt.Errorf("benchmark plan is required")
	}
	if store == nil {
		return BenchmarkRunResult{}, fmt.Errorf("research state store is required")
	}
	if err := ensureBenchmarkPlanPaths(plan); err != nil {
		return BenchmarkRunResult{}, err
	}
	workerCount := resolveBenchmarkWorkerCount(opts.Jobs, len(plan.Jobs))
	progress := &benchmarkProgress{writer: opts.ProgressWriter}
	if err := writePlanManifest(plan); err != nil {
		return BenchmarkRunResult{}, err
	}
	progress.logf("benchmark run started phase=%s run_id=%s jobs=%d parallel_jobs=%d manifest=%s summary=%s report=%s", plan.Phase, plan.RunID, len(plan.Jobs), workerCount, filepath.ToSlash(plan.ManifestPath), filepath.ToSlash(plan.SummaryPath), filepath.ToSlash(plan.ReportPath))
	storeCtx := controlPlaneContext(ctx)
	if err := store.UpsertRun(storeCtx, state.Run{
		RunID:             plan.RunID,
		Command:           plan.Command,
		Phase:             plan.Phase,
		Status:            "running",
		ManifestPath:      plan.ManifestPath,
		ConfigFingerprint: plan.ConfigFingerprint,
		StartedAtUTC:      time.Now().UTC().Format(time.RFC3339),
		MetadataJSON: state.MetadataJSON(map[string]any{
			"project_folder":     plan.ProjectFolder,
			"input_kind":         plan.InputKind,
			"input_file":         plan.InputFile,
			"input_path":         plan.InputPath,
			"prompt_version":     plan.PromptVersion,
			"repeat_count":       plan.RepeatCount,
			"parallel_jobs":      workerCount,
			"summary_path":       plan.SummaryPath,
			"report_path":        plan.ReportPath,
			"model_catalog_path": plan.ModelCatalogPath,
			"experiments":        benchmarkPlanExperiments(plan.Jobs),
		}),
	}); err != nil {
		return BenchmarkRunResult{}, err
	}
	if err := store.SaveRunConfig(storeCtx, plan.RunID, loaded.ConfigPath, loaded.LocalConfigPath, loaded.ResolvedJSON, loaded.Fingerprint); err != nil {
		return BenchmarkRunResult{}, err
	}
	if err := store.RecordArtifact(storeCtx, state.Artifact{
		RunID:        plan.RunID,
		Role:         "benchmark-manifest",
		AbsolutePath: plan.ManifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{"phase": plan.Phase}),
	}); err != nil {
		return BenchmarkRunResult{}, err
	}
	if fileExists(plan.InputPath) {
		if err := store.RecordArtifact(storeCtx, state.Artifact{
			RunID:        plan.RunID,
			Role:         "benchmark-input",
			AbsolutePath: plan.InputPath,
			MetadataJSON: state.MetadataJSON(map[string]any{"input_file": plan.InputFile}),
		}); err != nil {
			return BenchmarkRunResult{}, err
		}
	}

	result := BenchmarkRunResult{Plan: plan, ParallelJobs: workerCount}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type benchmarkWorkItem struct {
		job     planner.BenchmarkJob
		ordinal int
	}
	workCh := make(chan benchmarkWorkItem)
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	var fatalMu sync.Mutex
	var fatalErr error
	setFatalErr := func(err error) {
		if err == nil {
			return
		}
		fatalMu.Lock()
		if fatalErr == nil {
			fatalErr = err
			cancel()
		}
		fatalMu.Unlock()
	}
	for workerIdx := 0; workerIdx < workerCount; workerIdx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				case item, ok := <-workCh:
					if !ok {
						return
					}
					jobResult := executeBenchmarkJob(runCtx, loaded, store, storeCtx, plan, item.job, item.ordinal, len(plan.Jobs), opts, progress)
					resultMu.Lock()
					result.CompletedJobs += jobResult.CompletedJobs
					result.SkippedJobs += jobResult.SkippedJobs
					result.FailedJobs += jobResult.FailedJobs
					result.AbortedJobs += jobResult.AbortedJobs
					resultMu.Unlock()
					if jobResult.FatalErr != nil {
						setFatalErr(jobResult.FatalErr)
						return
					}
				}
			}
		}()
	}
enqueue:
	for idx, job := range plan.Jobs {
		item := benchmarkWorkItem{job: job, ordinal: idx + 1}
		select {
		case <-runCtx.Done():
			break enqueue
		case workCh <- item:
		}
	}
	close(workCh)
	wg.Wait()

	fatalMu.Lock()
	runErr := fatalErr
	fatalMu.Unlock()
	if runErr == nil && interruptedRun(ctx, nil) {
		runErr = context.Canceled
	}
	if runErr != nil {
		return finalizeBenchmarkRun(ctx, store, plan, result, runErr, progress)
	}

	if fileExists(plan.SummaryPath) {
		progress.logf("benchmark report generation started phase=%s run_id=%s summary=%s model_catalog=%s report=%s", plan.Phase, plan.RunID, filepath.ToSlash(plan.SummaryPath), filepath.ToSlash(plan.ModelCatalogPath), filepath.ToSlash(plan.ReportPath))
		if err := ensureBenchmarkModelCatalog(plan.ModelCatalogPath, plan.Jobs); err != nil {
			return finalizeBenchmarkRun(ctx, store, plan, result, err, progress)
		}
		if err := generateBenchmarkReport(ctx, loaded.WorkspaceRoot, plan.ProjectFolder, plan.InputFile, plan.SummaryPath, plan.ModelCatalogPath, plan.ReportPath); err != nil {
			return finalizeBenchmarkRun(ctx, store, plan, result, err, progress)
		}
		result.ReportPath = plan.ReportPath
		progress.logf("benchmark report generation completed phase=%s run_id=%s report=%s", plan.Phase, plan.RunID, filepath.ToSlash(plan.ReportPath))
		for _, artifact := range []state.Artifact{
			{
				RunID:        plan.RunID,
				Role:         "benchmark-summary",
				AbsolutePath: plan.SummaryPath,
			},
			{
				RunID:        plan.RunID,
				Role:         "benchmark-model-catalog",
				AbsolutePath: plan.ModelCatalogPath,
			},
			{
				RunID:        plan.RunID,
				Role:         "benchmark-report",
				AbsolutePath: plan.ReportPath,
			},
		} {
			if err := store.RecordArtifact(storeCtx, artifact); err != nil {
				return finalizeBenchmarkRun(ctx, store, plan, result, err, progress)
			}
		}
	} else if result.CompletedJobs > 0 {
		return finalizeBenchmarkRun(ctx, store, plan, result, fmt.Errorf("benchmark summary file %s was not produced", plan.SummaryPath), progress)
	}

	return finalizeBenchmarkRun(ctx, store, plan, result, runErr, progress)
}

func finalizeBenchmarkRun(ctx context.Context, store *state.Store, plan *planner.BenchmarkPlan, result BenchmarkRunResult, runErr error, progress *benchmarkProgress) (BenchmarkRunResult, error) {
	status := "completed"
	if interruptedRun(ctx, runErr) {
		status = "aborted"
	} else if runErr != nil {
		if result.CompletedJobs > 0 || result.SkippedJobs > 0 {
			status = "partial_failed"
		} else {
			status = "failed"
		}
	} else if result.FailedJobs > 0 {
		status = "partial_failed"
	}
	result.Status = status
	if err := store.UpsertRun(controlPlaneContext(ctx), state.Run{
		RunID:             plan.RunID,
		Command:           plan.Command,
		Phase:             plan.Phase,
		Status:            status,
		ManifestPath:      plan.ManifestPath,
		ConfigFingerprint: plan.ConfigFingerprint,
		FinishedAtUTC:     time.Now().UTC().Format(time.RFC3339),
		Error:             errorText(runErr),
		MetadataJSON: state.MetadataJSON(map[string]any{
			"completed_jobs": result.CompletedJobs,
			"skipped_jobs":   result.SkippedJobs,
			"failed_jobs":    result.FailedJobs,
			"aborted_jobs":   result.AbortedJobs,
			"parallel_jobs":  result.ParallelJobs,
			"report_path":    result.ReportPath,
		}),
	}); err != nil {
		if runErr != nil {
			return result, fmt.Errorf("%w; finalize run: %v", runErr, err)
		}
		return result, err
	}
	progress.logf("benchmark run finished phase=%s run_id=%s status=%s completed_jobs=%d skipped_jobs=%d failed_jobs=%d aborted_jobs=%d report=%s error=%q", plan.Phase, plan.RunID, status, result.CompletedJobs, result.SkippedJobs, result.FailedJobs, result.AbortedJobs, filepath.ToSlash(result.ReportPath), trimProgressMessage(errorText(runErr)))
	return result, runErr
}

type benchmarkJobExecutionResult struct {
	CompletedJobs int
	SkippedJobs   int
	FailedJobs    int
	AbortedJobs   int
	FatalErr      error
}

func executeBenchmarkJob(ctx context.Context, loaded researchconfig.Loaded, store *state.Store, storeCtx context.Context, plan *planner.BenchmarkPlan, job planner.BenchmarkJob, jobOrdinal, totalJobs int, opts BenchmarkRunOptions, progress *benchmarkProgress) benchmarkJobExecutionResult {
	if interruptedRun(ctx, nil) {
		return benchmarkJobExecutionResult{FatalErr: context.Canceled}
	}
	if opts.Resume {
		done, err := store.HasCompletedJob(storeCtx, plan.RunID, job.SpecHash)
		if err != nil {
			return benchmarkJobExecutionResult{FatalErr: fmt.Errorf("resume lookup for %s: %w", job.Key, err)}
		}
		if done {
			progress.logf("job skipped phase=%s run_id=%s job=%s index=%d/%d reason=resume-already-completed", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs)
			return benchmarkJobExecutionResult{SkippedJobs: 1}
		}
	}
	if opts.SkipExisting && fileExists(job.ResultsPath) {
		if err := store.UpsertJob(storeCtx, state.Job{
			RunID:       plan.RunID,
			JobKey:      job.Key,
			SpecHash:    job.SpecHash,
			Command:     job.Command,
			Phase:       job.Phase,
			Family:      job.Family,
			Model:       job.Model,
			Selector:    job.Selector,
			Status:      "skipped",
			ResultPath:  job.ResultsPath,
			SummaryPath: job.SummaryPath,
			RawDir:      job.RawDir,
			MetadataJSON: state.MetadataJSON(map[string]any{
				"worker_run_id": job.RunID,
				"skip_reason":   "existing_result",
				"experiment_id": job.ExperimentID,
			}),
		}); err != nil {
			return benchmarkJobExecutionResult{FatalErr: err}
		}
		_ = store.RecordArtifact(storeCtx, state.Artifact{
			RunID:        plan.RunID,
			JobKey:       job.Key,
			Role:         "benchmark-result",
			AbsolutePath: job.ResultsPath,
		})
		progress.logf("job skipped phase=%s run_id=%s job=%s index=%d/%d reason=existing-result results=%s", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs, filepath.ToSlash(job.ResultsPath))
		return benchmarkJobExecutionResult{SkippedJobs: 1}
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)
	jobStartedAt := time.Now().UTC()
	progress.logf("job started phase=%s run_id=%s job=%s index=%d/%d model=%s selector=%s provider=%s results=%s summary=%s raw_dir=%s", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs, job.Model, job.Selector, job.Provider, filepath.ToSlash(job.ResultsPath), filepath.ToSlash(job.SummaryPath), filepath.ToSlash(job.RawDir))
	if err := store.UpsertJob(storeCtx, state.Job{
		RunID:        plan.RunID,
		JobKey:       job.Key,
		SpecHash:     job.SpecHash,
		Command:      job.Command,
		Phase:        job.Phase,
		Family:       job.Family,
		Model:        job.Model,
		Selector:     job.Selector,
		Status:       "running",
		ResultPath:   job.ResultsPath,
		SummaryPath:  job.SummaryPath,
		RawDir:       job.RawDir,
		StartedAtUTC: startedAt,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"worker_run_id":              job.RunID,
			"provider":                   job.Provider,
			"base_url":                   job.BaseURL,
			"timeout":                    job.Timeout,
			"input_path":                 job.InputPath,
			"experiment_id":              job.ExperimentID,
			"experiment_name":            job.ExperimentName,
			"sampling_surface":           job.SamplingSurface,
			"requested_sampling_profile": job.RequestedSamplingProfile,
			"effective_sampling_profile": job.EffectiveSamplingProfile,
			"requested_sampling_json":    benchmarkSamplingJSON(job.RequestedSampling),
			"effective_sampling_json":    benchmarkSamplingJSON(job.EffectiveSampling),
			"unsupported_sampling_json":  benchmarkSamplingJSON(job.UnsupportedSampling),
		}),
	}); err != nil {
		return benchmarkJobExecutionResult{FatalErr: err}
	}

	output, err := runBenchmarkJobFunc(ctx, loaded.WorkspaceRoot, job)
	finishedAt := time.Now().UTC().Format(time.RFC3339)
	duration := time.Since(jobStartedAt).Round(time.Second)
	if err != nil {
		jobStatus := "failed"
		result := benchmarkJobExecutionResult{FailedJobs: 1}
		if interruptedRun(ctx, err) {
			jobStatus = "aborted"
			result = benchmarkJobExecutionResult{AbortedJobs: 1, FatalErr: context.Canceled}
		} else if !opts.BestEffort {
			result.FatalErr = fmt.Errorf("benchmark job %s failed: %w", job.Key, err)
		}
		jobErr := store.UpsertJob(storeCtx, state.Job{
			RunID:         plan.RunID,
			JobKey:        job.Key,
			SpecHash:      job.SpecHash,
			Command:       job.Command,
			Phase:         job.Phase,
			Family:        job.Family,
			Model:         job.Model,
			Selector:      job.Selector,
			Status:        jobStatus,
			ResultPath:    job.ResultsPath,
			SummaryPath:   job.SummaryPath,
			RawDir:        job.RawDir,
			StartedAtUTC:  startedAt,
			FinishedAtUTC: finishedAt,
			Error:         trimLog(firstNonEmpty(err.Error(), output)),
			MetadataJSON: state.MetadataJSON(map[string]any{
				"worker_run_id":              job.RunID,
				"provider":                   job.Provider,
				"base_url":                   job.BaseURL,
				"timeout":                    job.Timeout,
				"output":                     trimLog(output),
				"experiment_id":              job.ExperimentID,
				"experiment_name":            job.ExperimentName,
				"sampling_surface":           job.SamplingSurface,
				"requested_sampling_profile": job.RequestedSamplingProfile,
				"effective_sampling_profile": job.EffectiveSamplingProfile,
				"requested_sampling_json":    benchmarkSamplingJSON(job.RequestedSampling),
				"effective_sampling_json":    benchmarkSamplingJSON(job.EffectiveSampling),
				"unsupported_sampling_json":  benchmarkSamplingJSON(job.UnsupportedSampling),
			}),
		})
		if jobErr != nil {
			return benchmarkJobExecutionResult{FatalErr: jobErr}
		}
		if jobStatus == "aborted" {
			progress.logf("job aborted phase=%s run_id=%s job=%s index=%d/%d duration=%s error=%q", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs, duration, trimProgressMessage(firstNonEmpty(trimLog(output), err.Error())))
			return result
		}
		progress.logf("job failed phase=%s run_id=%s job=%s index=%d/%d duration=%s error=%q", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs, duration, trimProgressMessage(firstNonEmpty(trimLog(output), err.Error())))
		return result
	}

	if err := store.UpsertJob(storeCtx, state.Job{
		RunID:         plan.RunID,
		JobKey:        job.Key,
		SpecHash:      job.SpecHash,
		Command:       job.Command,
		Phase:         job.Phase,
		Family:        job.Family,
		Model:         job.Model,
		Selector:      job.Selector,
		Status:        "completed",
		ResultPath:    job.ResultsPath,
		SummaryPath:   job.SummaryPath,
		RawDir:        job.RawDir,
		StartedAtUTC:  startedAt,
		FinishedAtUTC: finishedAt,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"worker_run_id":              job.RunID,
			"provider":                   job.Provider,
			"base_url":                   job.BaseURL,
			"timeout":                    job.Timeout,
			"output":                     trimLog(output),
			"experiment_id":              job.ExperimentID,
			"experiment_name":            job.ExperimentName,
			"sampling_surface":           job.SamplingSurface,
			"requested_sampling_profile": job.RequestedSamplingProfile,
			"effective_sampling_profile": job.EffectiveSamplingProfile,
			"requested_sampling_json":    benchmarkSamplingJSON(job.RequestedSampling),
			"effective_sampling_json":    benchmarkSamplingJSON(job.EffectiveSampling),
			"unsupported_sampling_json":  benchmarkSamplingJSON(job.UnsupportedSampling),
		}),
	}); err != nil {
		return benchmarkJobExecutionResult{FatalErr: err}
	}
	progress.logf("job completed phase=%s run_id=%s job=%s index=%d/%d duration=%s results=%s summary=%s", job.Phase, plan.RunID, job.Key, jobOrdinal, totalJobs, duration, filepath.ToSlash(job.ResultsPath), filepath.ToSlash(job.SummaryPath))
	for _, artifact := range []state.Artifact{
		{
			RunID:        plan.RunID,
			JobKey:       job.Key,
			Role:         "benchmark-result",
			AbsolutePath: job.ResultsPath,
		},
		{
			RunID:        plan.RunID,
			JobKey:       job.Key,
			Role:         "benchmark-summary",
			AbsolutePath: job.SummaryPath,
		},
		{
			RunID:        plan.RunID,
			JobKey:       job.Key,
			Role:         "benchmark-raw-dir",
			AbsolutePath: job.RawDir,
		},
	} {
		if err := store.RecordArtifact(storeCtx, artifact); err != nil {
			return benchmarkJobExecutionResult{FatalErr: err}
		}
	}
	return benchmarkJobExecutionResult{CompletedJobs: 1}
}

func resolveBenchmarkWorkerCount(requested, totalJobs int) int {
	if totalJobs <= 0 {
		return 1
	}
	if requested <= 0 {
		return 1
	}
	if requested > totalJobs {
		return totalJobs
	}
	return requested
}

func ensureBenchmarkPlanPaths(plan *planner.BenchmarkPlan) error {
	dirs := []string{
		filepath.Dir(plan.ManifestPath),
		filepath.Dir(plan.SummaryPath),
		filepath.Dir(plan.ReportPath),
		filepath.Dir(plan.ModelCatalogPath),
	}
	for _, job := range plan.Jobs {
		dirs = append(dirs, filepath.Dir(job.ResultsPath), job.RawDir)
	}
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create research directory %s: %w", dir, err)
		}
	}
	return nil
}

func writePlanManifest(plan *planner.BenchmarkPlan) error {
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal benchmark manifest: %w", err)
	}
	if err := os.WriteFile(plan.ManifestPath, raw, 0o644); err != nil {
		return fmt.Errorf("write benchmark manifest %s: %w", plan.ManifestPath, err)
	}
	return nil
}

func runBenchmarkJob(ctx context.Context, workspaceRoot string, job planner.BenchmarkJob) (string, error) {
	subcommand := benchmarkContractsSubcommand(job.Phase)
	goBinary, err := resolveGoBinary()
	if err != nil {
		return "", err
	}
	toolingRoot := filepath.Join(workspaceRoot, "platform-tooling")
	args := []string{
		"-C", toolingRoot,
		"run", "./cmd/contracts", subcommand,
		"-provider", job.Provider,
		"-base-url", job.BaseURL,
		"-model", job.Model,
		"-timeout", job.Timeout,
		"-project-folder", job.ProjectFolder,
		"-cases-file", job.InputFile,
		"-cases", job.InputPath,
		"-results", job.ResultsPath,
		"-summary", job.SummaryPath,
		"-raw-dir", job.RawDir,
		"-run-id", job.RunID,
		"-prompt-version", job.PromptVersion,
		"-request-timeout-abort-threshold", strconv.Itoa(job.RequestTimeoutAbortThreshold),
	}
	if strings.TrimSpace(job.ExperimentName) != "" {
		args = append(args, "-experiment-name", job.ExperimentName)
	}
	if strings.TrimSpace(job.ExperimentID) != "" {
		args = append(args, "-experiment-id", job.ExperimentID)
	}
	if job.RequestedTemperature != nil {
		args = append(args, "-temperature", strconv.FormatFloat(*job.RequestedTemperature, 'f', -1, 64))
	}
	if job.RequestedSeed != nil {
		args = append(args, "-seed", strconv.FormatInt(*job.RequestedSeed, 10))
	}
	if job.RequestedTopP != nil {
		args = append(args, "-top-p", strconv.FormatFloat(*job.RequestedTopP, 'f', -1, 64))
	}
	if strings.TrimSpace(job.APIKeyEnv) != "" {
		apiKey := strings.TrimSpace(os.Getenv(job.APIKeyEnv))
		if apiKey == "" && job.Provider != "compatible" {
			return "", fmt.Errorf("required API key env %q is empty", job.APIKeyEnv)
		}
		if apiKey != "" {
			args = append(args, "-api-key", apiKey)
		}
	}
	cmd := exec.CommandContext(ctx, goBinary, args...)
	cmd.Dir = workspaceRoot
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", toolingpath.WorkspaceRootEnvKey, workspaceRoot))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("run contracts %s for job %s: %w", subcommand, job.Key, err)
	}
	return string(output), nil
}

func benchmarkContractsSubcommand(phase string) string {
	phase = strings.ToLower(strings.TrimSpace(phase))
	if phase == "nd" || strings.HasSuffix(phase, "-nd") {
		return "run-nd-hilbert-benchmark"
	}
	return "run-hilbert-benchmark"
}

func resolveGoBinary() (string, error) {
	if goBinary, err := exec.LookPath("go"); err == nil {
		return goBinary, nil
	}
	if _, err := os.Stat("/usr/local/go/bin/go"); err == nil {
		return "/usr/local/go/bin/go", nil
	}
	return "", fmt.Errorf("could not locate go binary in PATH or /usr/local/go/bin/go")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func trimLog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8000 {
		return value
	}
	return value[:8000]
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return trimLog(err.Error())
}

type benchmarkProgress struct {
	writer io.Writer
	mu     sync.Mutex
}

func (p *benchmarkProgress) logf(format string, args ...any) {
	if p == nil || p.writer == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = fmt.Fprintf(p.writer, "[%s] %s\n", time.Now().Local().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func trimProgressMessage(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) <= 300 {
		return value
	}
	return value[:300]
}

func benchmarkSamplingJSON(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func benchmarkPlanExperiments(jobs []planner.BenchmarkJob) []map[string]string {
	seen := make(map[string]struct{}, len(jobs))
	result := make([]map[string]string, 0, len(jobs))
	for _, job := range jobs {
		key := strings.TrimSpace(job.ExperimentID)
		if key == "" {
			key = strings.TrimSpace(job.ExperimentName)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, map[string]string{
			"experiment_id":              job.ExperimentID,
			"experiment_name":            job.ExperimentName,
			"sampling_surface":           job.SamplingSurface,
			"requested_sampling_profile": job.RequestedSamplingProfile,
			"effective_sampling_profile": job.EffectiveSamplingProfile,
		})
	}
	return result
}

func interruptedRun(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled))
}

func controlPlaneContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}
