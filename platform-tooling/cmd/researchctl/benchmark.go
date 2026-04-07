package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/report"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/runner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

func runBenchmarkSurface(phase string, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires a subcommand: plan or run", phase)
	}
	switch args[0] {
	case "plan":
		return runBenchmarkPlanCommand(phase, args[1:], stdout, stderr)
	case "run":
		return runBenchmarkRunCommand(phase, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown %s subcommand %q", phase, args[0])
	}
}

func runBenchmarkPlanCommand(phase string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(phase+"-plan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultBenchmarkLocalConfigPath(phase), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable benchmark run identifier")
	requireSecrets := fs.Bool("require-secrets", false, "Fail plan creation when required API key env vars are missing")
	jobs := fs.Int("jobs", -1, "Maximum number of benchmark jobs to run concurrently within this researchctl invocation; defaults to benchmarks.<phase>.jobs when omitted")
	requestTimeoutAbortThreshold := fs.Int("request-timeout-abort-threshold", -1, "After N consecutive request timeouts, stop new LLM requests and mark remaining cases as request_failure; defaults to benchmark config when omitted")
	interruptPolicy := fs.String("interrupt-policy", "", "Interrupted-run handling: keep|drop_if_no_results|drop_always; defaults to benchmark config when omitted")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if *requestTimeoutAbortThreshold < -1 {
		return fmt.Errorf("--request-timeout-abort-threshold must be >= -1")
	}
	if *jobs == 0 || *jobs < -1 {
		return fmt.Errorf("--jobs must be >= 1 when provided")
	}
	if strings.TrimSpace(*interruptPolicy) != "" {
		if _, err := resolveInterruptPolicy("", optionalStringFlag(*interruptPolicy)); err != nil {
			return err
		}
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	benchmarkConfig, err := selectBenchmark(phase, loaded.Config)
	if err != nil {
		return err
	}
	parallelJobs, err := resolveBenchmarkParallelJobs(benchmarkConfig, *jobs)
	if err != nil {
		return err
	}
	plan, err := buildBenchmarkPlan(phase, loaded, planner.BenchmarkPlanOptions{
		RunID:                        strings.TrimSpace(*runID),
		Command:                      normalizePhaseName(phase) + " run",
		RequireSecrets:               *requireSecrets,
		RequestTimeoutAbortThreshold: optionalIntFlag(*requestTimeoutAbortThreshold),
		InterruptPolicy:              optionalStringFlag(*interruptPolicy),
	})
	if err != nil {
		return err
	}
	if err := writeBenchmarkManifest(plan); err != nil {
		return err
	}
	if err := printPlanSummary(stdout, loaded, plan); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "parallel_jobs: %d\n", parallelJobs)
	return nil
}

func runBenchmarkRunCommand(phase string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(phase+"-run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultBenchmarkLocalConfigPath(phase), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable benchmark run identifier")
	resume := fs.Bool("resume", false, "Skip jobs already marked completed in the control-plane DB for the same run_id")
	skipExisting := fs.Bool("skip-existing", false, "Skip jobs when the expected result file already exists")
	jobs := fs.Int("jobs", -1, "Maximum number of benchmark jobs to run concurrently within this researchctl invocation; defaults to benchmarks.<phase>.jobs when omitted")
	bestEffort := fs.Bool("best-effort", false, "Continue other jobs after one benchmark job fails")
	failFast := fs.Bool("fail-fast", false, "Explicit alias for the default fail-fast mode")
	requestTimeoutAbortThreshold := fs.Int("request-timeout-abort-threshold", -1, "After N consecutive request timeouts, stop new LLM requests and mark remaining cases as request_failure; defaults to benchmark config when omitted")
	interruptPolicy := fs.String("interrupt-policy", "", "Interrupted-run handling: keep|drop_if_no_results|drop_always; defaults to benchmark config when omitted")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if *bestEffort && *failFast {
		return fmt.Errorf("--best-effort and --fail-fast are mutually exclusive")
	}
	if *requestTimeoutAbortThreshold < -1 {
		return fmt.Errorf("--request-timeout-abort-threshold must be >= -1")
	}
	if *jobs == 0 || *jobs < -1 {
		return fmt.Errorf("--jobs must be >= 1 when provided")
	}
	if strings.TrimSpace(*interruptPolicy) != "" {
		if _, err := resolveInterruptPolicy("", optionalStringFlag(*interruptPolicy)); err != nil {
			return err
		}
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	benchmarkConfig, err := selectBenchmark(phase, loaded.Config)
	if err != nil {
		return err
	}
	parallelJobs, err := resolveBenchmarkParallelJobs(benchmarkConfig, *jobs)
	if err != nil {
		return err
	}
	ctx := commandContext()
	store, err := state.Open(controlContext(ctx), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()

	plan, err := buildBenchmarkPlan(phase, loaded, planner.BenchmarkPlanOptions{
		RunID:                        strings.TrimSpace(*runID),
		Command:                      normalizePhaseName(phase) + " run",
		RequireSecrets:               true,
		RequestTimeoutAbortThreshold: optionalIntFlag(*requestTimeoutAbortThreshold),
		InterruptPolicy:              optionalStringFlag(*interruptPolicy),
	})
	if err != nil {
		return err
	}
	result, err := runner.RunBenchmark(ctx, loaded, store, plan, runner.BenchmarkRunOptions{
		Resume:         *resume,
		SkipExisting:   *skipExisting,
		Jobs:           parallelJobs,
		BestEffort:     *bestEffort,
		RequireSecrets: true,
		ProgressWriter: stdout,
	})
	if isInterruptedCommand(ctx, err) {
		if _, cleanupErr := maybeDropInterruptedBenchmarkRun(loaded, store, plan, result, stdout); cleanupErr != nil {
			return cleanupErr
		}
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research benchmark run completed\nphase: %s\nstatus: %s\nrun_id: %s\nmanifest_path: %s\nsummary_path: %s\nreport_path: %s\ninterrupt_policy: %s\nparallel_jobs: %d\ncompleted_jobs: %d\nskipped_jobs: %d\nfailed_jobs: %d\naborted_jobs: %d\n",
		normalizePhaseName(phase), firstNonEmptyText(result.Status, "completed"), result.Plan.RunID, filepath.ToSlash(result.Plan.ManifestPath), filepath.ToSlash(result.Plan.SummaryPath), filepath.ToSlash(result.ReportPath), plan.InterruptPolicy, result.ParallelJobs, result.CompletedJobs, result.SkippedJobs, result.FailedJobs, result.AbortedJobs)
	return nil
}

func runReportSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("report requires: benchmark, research, meta, or cases")
	}
	switch args[0] {
	case "benchmark":
		return runBenchmarkReportCommand(args[1:], stdout, stderr)
	case "research":
		return runResearchReportCommand(args[1:], stdout, stderr)
	case "meta":
		return runMetaReportCommand(args[1:], stdout, stderr)
	case "cases":
		return runCaseReportCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown report subcommand %q", args[0])
	}
}

func runBenchmarkReportCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("report-benchmark", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	phase := fs.String("phase", "phase1", "Benchmark phase: phase1|nd|compositional-assumption-import|bridge-import-final-research|bridge-only-authoring|gold-final-composition-only|compositional-depth-ladder|compositional-branching|compositional-mixed-family-reuse|compositional-mixed-family-gold-first|compositional-mixed-family-gold-first-phase2|compositional-mixed-family-gold-first-hard|compositional-mixed-family-semi-gold-stable|compositional-mixed-family-semi-gold-frontier")
	runID := fs.String("run-id", "", "Benchmark run identifier used to derive default summary/report paths")
	summaryPath := fs.String("summary", "", "Path to benchmark summary CSV")
	reportOut := fs.String("report-out", "", "Path to markdown report output")
	modelCatalogPath := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	phaseName := normalizePhaseName(*phase)
	benchmarkConfig, err := selectBenchmark(phaseName, loaded.Config)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*summaryPath) == "" {
		if strings.TrimSpace(*runID) == "" {
			return fmt.Errorf("report benchmark requires --summary or --run-id")
		}
		*summaryPath = filepath.ToSlash(filepath.Join(loaded.ResolvePath(benchmarkConfig.SummaryDir), fmt.Sprintf("%s_%s.csv", phaseName, strings.TrimSpace(*runID))))
	} else {
		*summaryPath = filepath.ToSlash(loaded.ResolvePath(*summaryPath))
	}
	if strings.TrimSpace(*reportOut) == "" {
		if strings.TrimSpace(*runID) == "" {
			return fmt.Errorf("report benchmark requires --report-out or --run-id")
		}
		reportDir := loaded.ResolvePath(benchmarkConfig.ReportDir)
		*reportOut = filepath.ToSlash(filepath.Join(reportDir, planner.BenchmarkReportFilename(loaded, phaseName, reportDir, strings.TrimSpace(*runID))))
	} else {
		*reportOut = filepath.ToSlash(loaded.ResolvePath(*reportOut))
	}
	if strings.TrimSpace(*modelCatalogPath) == "" {
		switch {
		case strings.TrimSpace(*runID) != "":
			*modelCatalogPath = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("%s_%s_model_catalog.csv", phaseName, strings.TrimSpace(*runID))))
		case strings.TrimSpace(benchmarkConfig.ModelCatalog) != "":
			*modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(benchmarkConfig.ModelCatalog))
		}
	} else {
		*modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(*modelCatalogPath))
	}
	if err := report.GenerateBenchmarkReport(commandContext(), loaded.WorkspaceRoot, benchmarkConfig.ProjectFolder, benchmarkConfig.InputFile, *summaryPath, *modelCatalogPath, *reportOut); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "benchmark report written\nphase: %s\nsummary_path: %s\nreport_path: %s\n", phaseName, filepath.ToSlash(*summaryPath), filepath.ToSlash(*reportOut))
	return nil
}

func runResearchReportCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("report-research", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	phase := fs.String("phase", "nd", "Benchmark context used for project/input defaults")
	summaryPaths := fs.String("summary", "", "Comma-separated list of summary CSV files")
	reportOut := fs.String("report-out", "", "Path to markdown research report output")
	modelCatalogPath := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if strings.TrimSpace(*summaryPaths) == "" {
		return fmt.Errorf("report research requires --summary")
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	phaseName := normalizePhaseName(*phase)
	benchmarkConfig, err := selectBenchmark(phaseName, loaded.Config)
	if err != nil {
		return err
	}

	resolvedSummaries := make([]string, 0, 4)
	for _, path := range splitCSV(*summaryPaths) {
		resolvedSummaries = append(resolvedSummaries, filepath.ToSlash(loaded.ResolvePath(path)))
	}
	if strings.TrimSpace(*reportOut) == "" {
		name := defaultReportFilename("research", phaseName, "")
		*reportOut = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Reports.ResearchDir), name))
	} else {
		*reportOut = filepath.ToSlash(loaded.ResolvePath(*reportOut))
	}
	if strings.TrimSpace(*modelCatalogPath) == "" {
		*modelCatalogPath = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), "research_report_model_catalog.csv"))
	} else {
		*modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(*modelCatalogPath))
	}
	if err := report.GenerateResearchReport(commandContext(), loaded.WorkspaceRoot, benchmarkConfig.ProjectFolder, benchmarkConfig.InputFile, resolvedSummaries, *modelCatalogPath, *reportOut); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research report written\nphase: %s\nsummary_paths: %s\nreport_path: %s\n", phaseName, strings.Join(resolvedSummaries, ","), filepath.ToSlash(*reportOut))
	return nil
}

func runMetaReportCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("report-meta", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	phase := fs.String("phase", "phase1", "Benchmark context used for project/input defaults")
	summaryPaths := fs.String("summary", "", "Comma-separated list of summary CSV files")
	summaryGlobs := fs.String("summary-glob", "", "Comma-separated list of glob patterns for summary CSV files")
	latest := fs.Int("latest", 0, "Automatically use the latest N summary CSV files from the phase summary_dir")
	reportOut := fs.String("report-out", "", "Path to markdown meta report output")
	modelCatalogPath := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if *latest < 0 {
		return fmt.Errorf("--latest must be >= 0")
	}
	if err := validateReportSummarySelection(*summaryPaths, *summaryGlobs, *latest); err != nil {
		return err
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	phaseName := normalizePhaseName(*phase)
	benchmarkConfig, err := selectBenchmark(phaseName, loaded.Config)
	if err != nil {
		return err
	}

	resolvedSummaries, err := resolveReportSummaryPaths(loaded, benchmarkConfig, phaseName, strings.TrimSpace(*summaryPaths), strings.TrimSpace(*summaryGlobs), *latest)
	if err != nil {
		return err
	}
	if len(resolvedSummaries) == 0 {
		return fmt.Errorf("report meta did not resolve any summary files")
	}
	if strings.TrimSpace(*reportOut) == "" {
		name := defaultReportFilename("meta", phaseName, "")
		*reportOut = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Reports.MetaDir), name))
	} else {
		*reportOut = filepath.ToSlash(loaded.ResolvePath(*reportOut))
	}
	if strings.TrimSpace(*modelCatalogPath) == "" {
		if strings.TrimSpace(benchmarkConfig.ModelCatalog) != "" {
			*modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(benchmarkConfig.ModelCatalog))
		}
	} else {
		*modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(*modelCatalogPath))
	}
	if err := report.GenerateMetaReport(commandContext(), loaded.WorkspaceRoot, benchmarkConfig.ProjectFolder, benchmarkConfig.InputFile, resolvedSummaries, *modelCatalogPath, *reportOut); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "meta report written\nphase: %s\nsummary_paths: %s\nreport_path: %s\n", phaseName, strings.Join(resolvedSummaries, ","), filepath.ToSlash(*reportOut))
	return nil
}

func runCaseReportCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("report-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	phase := fs.String("phase", "phase1", "Benchmark context used for project/input defaults")
	summaryPaths := fs.String("summary", "", "Comma-separated list of summary CSV files")
	summaryGlobs := fs.String("summary-glob", "", "Comma-separated list of glob patterns for summary CSV files")
	latest := fs.Int("latest", 0, "Automatically use the latest N summary CSV files from the phase summary_dir")
	reportOut := fs.String("report-out", "", "Path to markdown case report output")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if *latest < 0 {
		return fmt.Errorf("--latest must be >= 0")
	}
	if err := validateReportSummarySelection(*summaryPaths, *summaryGlobs, *latest); err != nil {
		return err
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	phaseName := normalizePhaseName(*phase)
	benchmarkConfig, err := selectBenchmark(phaseName, loaded.Config)
	if err != nil {
		return err
	}

	resolvedSummaries, err := resolveReportSummaryPaths(loaded, benchmarkConfig, phaseName, strings.TrimSpace(*summaryPaths), strings.TrimSpace(*summaryGlobs), *latest)
	if err != nil {
		return err
	}
	if len(resolvedSummaries) == 0 {
		return fmt.Errorf("report cases did not resolve any summary files")
	}
	if strings.TrimSpace(*reportOut) == "" {
		name := defaultReportFilename("cases", phaseName, "")
		*reportOut = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Reports.CasesDir), name))
	} else {
		*reportOut = filepath.ToSlash(loaded.ResolvePath(*reportOut))
	}
	if err := report.GenerateCaseReport(commandContext(), loaded.WorkspaceRoot, benchmarkConfig.ProjectFolder, benchmarkConfig.InputFile, resolvedSummaries, *reportOut); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "case report written\nphase: %s\nsummary_paths: %s\nreport_path: %s\n", phaseName, strings.Join(resolvedSummaries, ","), filepath.ToSlash(*reportOut))
	return nil
}

func validateReportSummarySelection(summaryPaths, summaryGlobs string, latest int) error {
	selected := 0
	if strings.TrimSpace(summaryPaths) != "" {
		selected++
	}
	if strings.TrimSpace(summaryGlobs) != "" {
		selected++
	}
	if latest > 0 {
		selected++
	}
	if selected == 0 {
		return fmt.Errorf("report requires one of --summary, --summary-glob, or --latest")
	}
	if selected > 1 {
		return fmt.Errorf("--summary, --summary-glob, and --latest are mutually exclusive")
	}
	return nil
}

func resolveReportSummaryPaths(loaded researchconfig.Loaded, benchmarkConfig researchconfig.Benchmark, phaseName, summaryPathsRaw, summaryGlobsRaw string, latest int) ([]string, error) {
	switch {
	case strings.TrimSpace(summaryPathsRaw) != "":
		resolved := make([]string, 0, 4)
		for _, path := range splitCSV(summaryPathsRaw) {
			resolved = append(resolved, filepath.ToSlash(loaded.ResolvePath(path)))
		}
		return resolved, nil
	case strings.TrimSpace(summaryGlobsRaw) != "":
		return resolveReportSummaryGlobs(loaded, splitCSV(summaryGlobsRaw))
	case latest > 0:
		return resolveReportLatestSummaries(loaded, benchmarkConfig, phaseName, latest)
	default:
		return nil, nil
	}
}

func resolveReportSummaryGlobs(loaded researchconfig.Loaded, patterns []string) ([]string, error) {
	matches := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	for _, pattern := range patterns {
		resolvedPattern := loaded.ResolvePath(pattern)
		globMatches, err := filepath.Glob(resolvedPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid summary glob %q: %w", pattern, err)
		}
		for _, match := range globMatches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("stat summary match %s: %w", match, err)
			}
			if info.IsDir() {
				continue
			}
			clean := filepath.Clean(match)
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			matches = append(matches, filepath.ToSlash(clean))
		}
	}
	slices.Sort(matches)
	return matches, nil
}

func resolveReportLatestSummaries(loaded researchconfig.Loaded, benchmarkConfig researchconfig.Benchmark, phaseName string, latest int) ([]string, error) {
	summaryDir := loaded.ResolvePath(benchmarkConfig.SummaryDir)
	pattern := filepath.Join(summaryDir, phaseName+"_*.csv")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob latest summaries for phase %q: %w", phaseName, err)
	}
	type candidate struct {
		path    string
		modTime time.Time
	}
	candidates := make([]candidate, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("stat latest summary candidate %s: %w", match, err)
		}
		if info.IsDir() {
			continue
		}
		candidates = append(candidates, candidate{
			path:    filepath.ToSlash(filepath.Clean(match)),
			modTime: info.ModTime().UTC(),
		})
	}
	slices.SortFunc(candidates, func(a, b candidate) int {
		switch {
		case a.modTime.After(b.modTime):
			return -1
		case a.modTime.Before(b.modTime):
			return 1
		default:
			return strings.Compare(a.path, b.path)
		}
	})
	if latest > len(candidates) {
		latest = len(candidates)
	}
	resolved := make([]string, 0, latest)
	for _, item := range candidates[:latest] {
		resolved = append(resolved, item.path)
	}
	return resolved, nil
}

func defaultReportFilename(kind, phaseName, runID string) string {
	phaseName = strings.TrimSpace(phaseName)
	runID = strings.TrimSpace(runID)
	switch kind {
	case "benchmark":
		if runID == "" {
			return fmt.Sprintf("benchmark-%s.md", timestampRunID())
		}
		return fmt.Sprintf("benchmark-%s.md", runID)
	case "research":
		return fmt.Sprintf("research-%s-%s.md", phaseName, timestampRunID())
	case "meta":
		return fmt.Sprintf("meta-%s-%s.md", phaseName, timestampRunID())
	case "cases":
		return fmt.Sprintf("cases-%s-%s.md", phaseName, timestampRunID())
	default:
		return fmt.Sprintf("%s-%s.md", kind, timestampRunID())
	}
}

func buildBenchmarkPlan(phase string, loaded researchconfig.Loaded, opts planner.BenchmarkPlanOptions) (*planner.BenchmarkPlan, error) {
	switch normalizePhaseName(phase) {
	case "phase1":
		return planner.BuildPhase1Plan(loaded, opts)
	case "nd":
		return planner.BuildNDPlan(loaded, opts)
	case "compositional-assumption-import":
		return planner.BuildCompositionalAssumptionImportPlan(loaded, opts)
	case "bridge-import-final-research":
		return planner.BuildBridgeImportFinalResearchPlan(loaded, opts)
	case "bridge-only-authoring":
		return planner.BuildBridgeOnlyAuthoringPlan(loaded, opts)
	case "gold-final-composition-only":
		return planner.BuildGoldFinalCompositionOnlyPlan(loaded, opts)
	case "compositional-depth-ladder":
		return planner.BuildCompositionalDepthLadderPlan(loaded, opts)
	case "compositional-branching":
		return planner.BuildCompositionalBranchingPlan(loaded, opts)
	case "compositional-mixed-family-reuse":
		return planner.BuildCompositionalMixedFamilyReusePlan(loaded, opts)
	case "compositional-mixed-family-gold-first":
		return planner.BuildCompositionalMixedFamilyGoldFirstPlan(loaded, opts)
	case "compositional-mixed-family-gold-first-phase2":
		return planner.BuildCompositionalMixedFamilyGoldFirstPhase2Plan(loaded, opts)
	case "compositional-mixed-family-gold-first-hard":
		return planner.BuildCompositionalMixedFamilyGoldFirstHardPlan(loaded, opts)
	case "compositional-mixed-family-semi-gold-stable":
		return planner.BuildCompositionalMixedFamilySemiGoldStablePlan(loaded, opts)
	case "compositional-mixed-family-semi-gold-frontier":
		return planner.BuildCompositionalMixedFamilySemiGoldFrontierPlan(loaded, opts)
	default:
		return nil, fmt.Errorf("unsupported benchmark phase %q", phase)
	}
}

func selectBenchmark(phase string, cfg researchconfig.Config) (researchconfig.Benchmark, error) {
	switch normalizePhaseName(phase) {
	case "phase1":
		return cfg.Benchmarks.Phase1, nil
	case "nd":
		return cfg.Benchmarks.ND, nil
	case "compositional-assumption-import":
		return cfg.Benchmarks.CompositionalAssumptionImport, nil
	case "bridge-import-final-research":
		return cfg.Benchmarks.BridgeImportFinalResearch, nil
	case "bridge-only-authoring":
		return cfg.Benchmarks.BridgeOnlyAuthoring, nil
	case "gold-final-composition-only":
		return cfg.Benchmarks.GoldFinalCompositionOnly, nil
	case "compositional-depth-ladder":
		return cfg.Benchmarks.CompositionalDepthLadder, nil
	case "compositional-branching":
		return cfg.Benchmarks.CompositionalBranching, nil
	case "compositional-mixed-family-reuse":
		return cfg.Benchmarks.CompositionalMixedFamilyReuse, nil
	case "compositional-mixed-family-gold-first":
		return cfg.Benchmarks.CompositionalMixedFamilyGoldFirst, nil
	case "compositional-mixed-family-gold-first-phase2":
		return cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2, nil
	case "compositional-mixed-family-gold-first-hard":
		return cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard, nil
	case "compositional-mixed-family-semi-gold-stable":
		return cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable, nil
	case "compositional-mixed-family-semi-gold-frontier":
		return cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier, nil
	default:
		return researchconfig.Benchmark{}, fmt.Errorf("unsupported benchmark phase %q", phase)
	}
}

func printPlanSummary(stdout io.Writer, loaded researchconfig.Loaded, plan *planner.BenchmarkPlan) error {
	_, err := fmt.Fprintf(stdout, "research benchmark plan ready\nphase: %s\nrun_id: %s\nmanifest_path: %s\nsummary_path: %s\nreport_path: %s\nmodel_catalog_path: %s\nrequest_timeout_abort_threshold: %d\ninterrupt_policy: %s\njobs: %d\nconfig: %s\nlocal_config: %s\n",
		plan.Phase,
		plan.RunID,
		filepath.ToSlash(plan.ManifestPath),
		filepath.ToSlash(plan.SummaryPath),
		filepath.ToSlash(plan.ReportPath),
		filepath.ToSlash(plan.ModelCatalogPath),
		plan.RequestTimeoutAbortThreshold,
		plan.InterruptPolicy,
		len(plan.Jobs),
		filepath.ToSlash(loaded.ConfigPath),
		filepath.ToSlash(loaded.LocalConfigPath),
	)
	return err
}

func optionalIntFlag(value int) *int {
	if value < 0 {
		return nil
	}
	resolved := value
	return &resolved
}

func resolveBenchmarkParallelJobs(benchmarkConfig researchconfig.Benchmark, jobsFlag int) (int, error) {
	switch {
	case jobsFlag > 0:
		return jobsFlag, nil
	case jobsFlag == -1 && benchmarkConfig.ParallelJobs > 0:
		return benchmarkConfig.ParallelJobs, nil
	case jobsFlag == -1:
		return 0, fmt.Errorf("benchmark config is missing jobs > 0")
	default:
		return 0, fmt.Errorf("--jobs must be >= 1 when provided")
	}
}

func defaultBenchmarkLocalConfigPath(phase string) string {
	switch normalizePhaseName(phase) {
	case "compositional-assumption-import":
		return compositionalConfigPath("assumption-import.yaml")
	case "bridge-import-final-research":
		return bridgeConfigPath("import-final-research.yaml")
	case "bridge-only-authoring":
		// The default stays on the small baseline pack; larger phase2 packs are
		// explicit opt-ins via --local-config to avoid silently changing run size.
		return bridgeConfigPath("only-authoring.yaml")
	case "gold-final-composition-only":
		// Keep the baseline profile as the default surface for the same reason as
		// bridge-only-authoring: phase2 should never be selected implicitly.
		return goldConfigPath("final-composition-only.yaml")
	case "compositional-depth-ladder":
		return compositionalConfigPath("depth-ladder.yaml")
	case "compositional-branching":
		return compositionalConfigPath("branching.yaml")
	case "compositional-mixed-family-reuse":
		return compositionalConfigPath("mixed-family-reuse.yaml")
	case "compositional-mixed-family-gold-first":
		return compositionalConfigPath("mixed-family-gold-first.yaml")
	case "compositional-mixed-family-gold-first-phase2":
		return compositionalConfigPath("mixed-family-gold-first-phase2.yaml")
	case "compositional-mixed-family-gold-first-hard":
		return compositionalConfigPath("mixed-family-gold-first-hard.yaml")
	case "compositional-mixed-family-semi-gold-stable":
		return compositionalConfigPath("mixed-family-semi-gold-stable.yaml")
	case "compositional-mixed-family-semi-gold-frontier":
		return compositionalConfigPath("mixed-family-semi-gold-frontier.yaml")
	default:
		return defaultWorkflowLocalConfigPath()
	}
}
