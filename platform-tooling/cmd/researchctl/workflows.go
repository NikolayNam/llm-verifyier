package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	lean4worker "github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/artifactkey"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	leanlayer "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/lean4"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/report"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/runner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

func runPhase2Surface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("phase2 requires a subcommand: plan or run")
	}
	switch args[0] {
	case "plan":
		return runPairedBenchmarkPlanCommand("phase2", args[1:], stdout, stderr)
	case "run":
		return runPairedBenchmarkRunCommand("phase2", args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown phase2 subcommand %q", args[0])
	}
}

func runPhase3Surface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("phase3 requires a subcommand: plan, run, or compare")
	}
	switch args[0] {
	case "plan":
		return runPhase3PlanCommand(args[1:], stdout, stderr)
	case "run":
		return runPhase3RunCommand(args[1:], stdout, stderr)
	case "compare":
		return runPhase3CompareSurface(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown phase3 subcommand %q", args[0])
	}
}

func runPhase3CompareSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("phase3 compare requires a subcommand: plan or run")
	}
	switch args[0] {
	case "plan":
		return runPairedBenchmarkPlanCommand("phase3-compare", args[1:], stdout, stderr)
	case "run":
		return runPairedBenchmarkRunCommand("phase3-compare", args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown phase3 compare subcommand %q", args[0])
	}
}

func runPairedBenchmarkPlanCommand(phase string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(phase+"-plan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable workflow run identifier")
	requireSecrets := fs.Bool("require-secrets", false, "Fail plan creation when required API key env vars are missing")
	sourceFile := fs.String("source-file", "", "Optional benchmark input CSV override for phase3 compare")
	jobs := fs.Int("jobs", 1, "Maximum number of jobs to run concurrently inside each child benchmark lane")
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
	if *jobs < 1 {
		return fmt.Errorf("--jobs must be >= 1")
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
	plan, err := buildPairedPlan(phase, loaded, planner.WorkflowPlanOptions{
		RunID:                        strings.TrimSpace(*runID),
		Command:                      strings.ReplaceAll(phase, "-", " ") + " run",
		RequireSecrets:               *requireSecrets,
		RequestTimeoutAbortThreshold: optionalIntFlag(*requestTimeoutAbortThreshold),
		InterruptPolicy:              optionalStringFlag(*interruptPolicy),
	}, strings.TrimSpace(*sourceFile))
	if err != nil {
		return err
	}
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal %s manifest: %w", phase, err)
	}
	if err := writeManifestFile(plan.ManifestPath, raw); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research workflow plan ready\nphase: %s\nrun_id: %s\nmanifest_path: %s\ndirect_manifest_path: %s\nnd_manifest_path: %s\npair_report_path: %s\ndirect_summary_path: %s\nnd_summary_path: %s\ndirect_request_timeout_abort_threshold: %d\nnd_request_timeout_abort_threshold: %d\ndirect_interrupt_policy: %s\nnd_interrupt_policy: %s\nparallel_jobs: %d\ndirect_jobs: %d\nnd_jobs: %d\nconfig: %s\nlocal_config: %s\n",
		plan.Phase,
		plan.RunID,
		filepath.ToSlash(plan.ManifestPath),
		filepath.ToSlash(plan.Direct.ManifestPath),
		filepath.ToSlash(plan.ND.ManifestPath),
		filepath.ToSlash(plan.PairReportPath),
		filepath.ToSlash(plan.Direct.SummaryPath),
		filepath.ToSlash(plan.ND.SummaryPath),
		plan.Direct.RequestTimeoutAbortThreshold,
		plan.ND.RequestTimeoutAbortThreshold,
		plan.Direct.InterruptPolicy,
		plan.ND.InterruptPolicy,
		*jobs,
		len(plan.Direct.Jobs),
		len(plan.ND.Jobs),
		filepath.ToSlash(loaded.ConfigPath),
		filepath.ToSlash(loaded.LocalConfigPath),
	)
	return nil
}

func runPairedBenchmarkRunCommand(phase string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(phase+"-run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable workflow run identifier")
	resume := fs.Bool("resume", false, "Skip jobs already marked completed in the control-plane DB for the same child run_id")
	skipExisting := fs.Bool("skip-existing", false, "Skip jobs when the expected result file already exists")
	jobs := fs.Int("jobs", 1, "Maximum number of jobs to run concurrently inside each child benchmark lane")
	bestEffort := fs.Bool("best-effort", false, "Continue the paired workflow after one child benchmark lane fails")
	failFast := fs.Bool("fail-fast", false, "Explicit alias for the default fail-fast mode")
	sourceFile := fs.String("source-file", "", "Optional benchmark input CSV override for phase3 compare")
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
	if *jobs < 1 {
		return fmt.Errorf("--jobs must be >= 1")
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
	ctx := commandContext()
	store, err := state.Open(controlContext(ctx), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()

	commandName := strings.ReplaceAll(phase, "-", " ") + " run"
	plan, err := buildPairedPlan(phase, loaded, planner.WorkflowPlanOptions{
		RunID:                        strings.TrimSpace(*runID),
		Command:                      commandName,
		RequireSecrets:               true,
		RequestTimeoutAbortThreshold: optionalIntFlag(*requestTimeoutAbortThreshold),
		InterruptPolicy:              optionalStringFlag(*interruptPolicy),
	}, strings.TrimSpace(*sourceFile))
	if err != nil {
		return err
	}
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal %s manifest: %w", phase, err)
	}
	if err := writeManifestFile(plan.ManifestPath, raw); err != nil {
		return err
	}
	if err := startRun(ctx, store, loaded, plan.RunID, commandName, plan.Phase, plan.ManifestPath); err != nil {
		return err
	}
	if err := store.RecordArtifact(controlContext(ctx), state.Artifact{
		RunID:        plan.RunID,
		Role:         "workflow-manifest",
		AbsolutePath: plan.ManifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{"phase": plan.Phase}),
	}); err != nil {
		return err
	}

	directResult, directErr := runner.RunBenchmark(ctx, loaded, store, plan.Direct, runner.BenchmarkRunOptions{
		Resume:         *resume,
		SkipExisting:   *skipExisting,
		Jobs:           *jobs,
		BestEffort:     *bestEffort,
		RequireSecrets: true,
		ProgressWriter: stdout,
	})
	if isInterruptedCommand(ctx, directErr) {
		if _, cleanupErr := maybeDropInterruptedBenchmarkRun(loaded, store, plan.Direct, directResult, stdout); cleanupErr != nil {
			return cleanupErr
		}
		_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, runner.BenchmarkRunResult{}, context.Canceled, "", "", false)
		if finishErr != nil {
			return finishErr
		}
		if _, cleanupErr := maybeDropInterruptedWorkflowRun(loaded, store, plan, directResult, runner.BenchmarkRunResult{}, "", plan.Direct.InterruptPolicy, stdout); cleanupErr != nil {
			return cleanupErr
		}
		return directErr
	}
	if directErr != nil && !*bestEffort {
		_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, runner.BenchmarkRunResult{}, directErr, "", "", false)
		if finishErr != nil {
			return finishErr
		}
		return directErr
	}

	ndResult, ndErr := runner.RunBenchmark(ctx, loaded, store, plan.ND, runner.BenchmarkRunOptions{
		Resume:         *resume,
		SkipExisting:   *skipExisting,
		Jobs:           *jobs,
		BestEffort:     *bestEffort,
		RequireSecrets: true,
		ProgressWriter: stdout,
	})
	if isInterruptedCommand(ctx, ndErr) {
		if _, cleanupErr := maybeDropInterruptedBenchmarkRun(loaded, store, plan.ND, ndResult, stdout); cleanupErr != nil {
			return cleanupErr
		}
		_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, ndResult, context.Canceled, "", "", benchmarkRunHasMeaningfulOutputs(plan.Direct, directResult))
		if finishErr != nil {
			return finishErr
		}
		if _, cleanupErr := maybeDropInterruptedWorkflowRun(loaded, store, plan, directResult, ndResult, "", plan.ND.InterruptPolicy, stdout); cleanupErr != nil {
			return cleanupErr
		}
		return ndErr
	}
	if ndErr != nil && !*bestEffort {
		errToReturn := ndErr
		if directErr != nil {
			errToReturn = fmt.Errorf("%v; %w", directErr, ndErr)
		}
		_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, ndResult, errToReturn, "", "", directErr == nil || ndResult.CompletedJobs > 0 || ndResult.SkippedJobs > 0)
		if finishErr != nil {
			return finishErr
		}
		return errToReturn
	}

	pairReportPath := ""
	pairCatalogPath := ""
	if fileExists(plan.Direct.SummaryPath) && fileExists(plan.ND.SummaryPath) {
		pairCatalogPath = plan.PairModelCatalog
		if err := report.EnsureModelCatalog(pairCatalogPath, plan.AllJobs()); err != nil {
			workflowErr := combineWorkflowErrors(directErr, ndErr, err)
			_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, ndResult, workflowErr, pairReportPath, pairCatalogPath, directErr != nil || ndErr != nil || directResult.CompletedJobs > 0 || ndResult.CompletedJobs > 0 || directResult.SkippedJobs > 0 || ndResult.SkippedJobs > 0)
			if finishErr != nil {
				return finishErr
			}
			if isInterruptedCommand(ctx, workflowErr) {
				if _, cleanupErr := maybeDropInterruptedWorkflowRun(loaded, store, plan, directResult, ndResult, pairReportPath, plan.ND.InterruptPolicy, stdout); cleanupErr != nil {
					return cleanupErr
				}
			}
			return workflowErr
		}
		if err := report.GenerateResearchReport(ctx, loaded.WorkspaceRoot, plan.ProjectFolder, plan.InputFile, []string{plan.Direct.SummaryPath, plan.ND.SummaryPath}, pairCatalogPath, plan.PairReportPath); err != nil {
			workflowErr := combineWorkflowErrors(directErr, ndErr, err)
			_, finishErr := finishWorkflowRun(ctx, store, plan, loaded, directResult, ndResult, workflowErr, pairReportPath, pairCatalogPath, true)
			if finishErr != nil {
				return finishErr
			}
			if isInterruptedCommand(ctx, workflowErr) {
				if _, cleanupErr := maybeDropInterruptedWorkflowRun(loaded, store, plan, directResult, ndResult, pairReportPath, plan.ND.InterruptPolicy, stdout); cleanupErr != nil {
					return cleanupErr
				}
			}
			return workflowErr
		}
		pairReportPath = plan.PairReportPath
		for _, artifact := range []state.Artifact{
			{
				RunID:        plan.RunID,
				Role:         "workflow-pair-model-catalog",
				AbsolutePath: pairCatalogPath,
			},
			{
				RunID:        plan.RunID,
				Role:         "workflow-pair-report",
				AbsolutePath: pairReportPath,
			},
		} {
			if err := store.RecordArtifact(controlContext(ctx), artifact); err != nil {
				return err
			}
		}
	}

	workflowErr := combineWorkflowErrors(directErr, ndErr)
	partial := workflowErr != nil && (directResult.CompletedJobs > 0 || directResult.SkippedJobs > 0 || ndResult.CompletedJobs > 0 || ndResult.SkippedJobs > 0)
	status, err := finishWorkflowRun(ctx, store, plan, loaded, directResult, ndResult, workflowErr, pairReportPath, pairCatalogPath, partial)
	if err != nil {
		return err
	}
	if isInterruptedCommand(ctx, workflowErr) {
		if _, cleanupErr := maybeDropInterruptedWorkflowRun(loaded, store, plan, directResult, ndResult, pairReportPath, plan.ND.InterruptPolicy, stdout); cleanupErr != nil {
			return cleanupErr
		}
	}
	_, _ = fmt.Fprintf(stdout, "research workflow run completed\nphase: %s\nstatus: %s\nrun_id: %s\nmanifest_path: %s\ndirect_run_id: %s\nnd_run_id: %s\npair_report_path: %s\ndirect_report_path: %s\nnd_report_path: %s\ndirect_interrupt_policy: %s\nnd_interrupt_policy: %s\nparallel_jobs: %d\ndirect_completed_jobs: %d\nnd_completed_jobs: %d\ndirect_failed_jobs: %d\nnd_failed_jobs: %d\ndirect_aborted_jobs: %d\nnd_aborted_jobs: %d\n",
		plan.Phase,
		status,
		plan.RunID,
		filepath.ToSlash(plan.ManifestPath),
		plan.Direct.RunID,
		plan.ND.RunID,
		filepath.ToSlash(pairReportPath),
		filepath.ToSlash(directResult.ReportPath),
		filepath.ToSlash(ndResult.ReportPath),
		plan.Direct.InterruptPolicy,
		plan.ND.InterruptPolicy,
		*jobs,
		directResult.CompletedJobs,
		ndResult.CompletedJobs,
		directResult.FailedJobs,
		ndResult.FailedJobs,
		directResult.AbortedJobs,
		ndResult.AbortedJobs,
	)
	return workflowErr
}

func runPhase3PlanCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("phase3-plan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable workflow run identifier")
	jobName := fs.String("job-name", "", "Lean smoke-check job name override")
	jobStatement := fs.String("job-statement", "", "Lean smoke-check theorem statement override")
	jobPayloadFile := fs.String("job-payload-file", "", "Lean smoke-check payload JSON override")

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
	plan := planner.BuildPhase3Plan(loaded, planner.Phase3PlanOptions{
		RunID:          strings.TrimSpace(*runID),
		Command:        "phase3 run",
		JobName:        strings.TrimSpace(*jobName),
		JobStatement:   strings.TrimSpace(*jobStatement),
		JobPayloadFile: strings.TrimSpace(*jobPayloadFile),
	})
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal phase3 manifest: %w", err)
	}
	if err := writeManifestFile(plan.ManifestPath, raw); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research workflow plan ready\nphase: %s\nrun_id: %s\nmanifest_path: %s\nworkspace_dir: %s\njob_name: %s\njob_statement: %s\njob_payload_path: %s\ntheorem_pack_path: %s\nexport_output_path: %s\nconfig: %s\nlocal_config: %s\n",
		plan.Phase,
		plan.RunID,
		filepath.ToSlash(plan.ManifestPath),
		filepath.ToSlash(plan.WorkspaceDir),
		plan.JobName,
		plan.JobStatement,
		filepath.ToSlash(plan.JobPayloadPath),
		filepath.ToSlash(plan.TheoremPackPath),
		filepath.ToSlash(plan.ExportOutputPath),
		filepath.ToSlash(loaded.ConfigPath),
		filepath.ToSlash(loaded.LocalConfigPath),
	)
	return nil
}

func runPhase3RunCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("phase3-run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", "", "Stable workflow run identifier")
	jobName := fs.String("job-name", "", "Lean smoke-check job name override")
	jobStatement := fs.String("job-statement", "", "Lean smoke-check theorem statement override")
	jobPayloadFile := fs.String("job-payload-file", "", "Lean smoke-check payload JSON override")
	mode := fs.String("mode", "", "Hypothesis generation mode override")
	atoms := fs.String("atoms", "", "Comma-separated atomic propositions override")
	maxFormulaDepth := fs.Int("max-formula-depth", 0, "Maximum implicational formula depth override")
	maxAssumptions := fs.Int("max-assumptions", 0, "Maximum assumptions per generated case override")
	generationLimit := fs.Int("generation-limit", 0, "Maximum number of generated theorem cases")
	exportLimit := fs.Int("export-limit", 0, "Maximum number of exported benchmark rows")
	filters := fs.String("filters", "", "Comma-separated generation filters override")
	generationSourceFile := fs.String("generation-source-file", "", "Optional seed theorem CSV for curated_backlog or model_proposed generation modes")
	exportSourceFile := fs.String("export-source-file", "", "Optional theorem CSV override for benchmark export")
	outputFile := fs.String("output-file", "", "Benchmark-relative or absolute export output CSV override")
	exportMode := fs.String("export-mode", "", "Optional export generation_mode filter")
	interestingOnly := fs.Bool("interesting-only", false, "Export only interesting hypotheses")
	minimalOnly := fs.Bool("minimal-only", false, "Export only minimal hypotheses")

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
	plan := planner.BuildPhase3Plan(loaded, planner.Phase3PlanOptions{
		RunID:          strings.TrimSpace(*runID),
		Command:        "phase3 run",
		JobName:        strings.TrimSpace(*jobName),
		JobStatement:   strings.TrimSpace(*jobStatement),
		JobPayloadFile: strings.TrimSpace(*jobPayloadFile),
	})
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal phase3 manifest: %w", err)
	}
	if err := writeManifestFile(plan.ManifestPath, raw); err != nil {
		return err
	}

	ctx := commandContext()
	store, err := state.Open(controlContext(ctx), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(ctx, store, loaded, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath); err != nil {
		return err
	}
	if err := store.RecordArtifact(ctx, state.Artifact{
		RunID:        plan.RunID,
		Role:         "workflow-manifest",
		AbsolutePath: plan.ManifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{"phase": plan.Phase}),
	}); err != nil {
		return err
	}

	bootstrapResult, err := leanlayer.Bootstrap(ctx, loaded, plan.RunID)
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"workspace_dir": plan.WorkspaceDir})
		return err
	}

	payload, err := os.ReadFile(plan.JobPayloadPath)
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"workspace_dir": bootstrapResult.WorkspaceDir})
		return fmt.Errorf("read phase3 job payload: %w", err)
	}
	jobResult, projectDir, err := leanlayer.RunJob(ctx, loaded, plan.RunID, plan.JobName, plan.JobStatement, payload)
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"workspace_dir": bootstrapResult.WorkspaceDir, "project_dir": projectDir})
		return err
	}

	genConfig := lean4worker.DefaultHypothesisGenerationConfig()
	if strings.TrimSpace(*mode) != "" {
		genConfig.Mode = strings.TrimSpace(*mode)
	}
	if strings.TrimSpace(*atoms) != "" {
		genConfig.Atoms = splitCSV(*atoms)
	}
	if *maxFormulaDepth > 0 {
		genConfig.MaxFormulaDepth = *maxFormulaDepth
	}
	if *maxAssumptions > 0 {
		genConfig.MaxAssumptions = *maxAssumptions
	}
	if *generationLimit > 0 {
		genConfig.Limit = *generationLimit
	}
	if strings.TrimSpace(*filters) != "" {
		genConfig.Filters = splitCSV(*filters)
	}
	if strings.TrimSpace(*generationSourceFile) != "" {
		genConfig.SourceFile = resolveLeanSourcePath(loaded, strings.TrimSpace(*generationSourceFile))
	}

	_, normalizedGenConfig, err := lean4worker.GenerateHypothesisCases(genConfig)
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"workspace_dir": bootstrapResult.WorkspaceDir, "project_dir": projectDir})
		return err
	}
	theoremsPath, configOutPath, payloadTemplatePath, err := leanlayer.GenerateCases(loaded, genConfig)
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"workspace_dir": bootstrapResult.WorkspaceDir, "project_dir": projectDir})
		return err
	}
	artifactRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder)))
	persisted, err := persistLeanGenerationFromArtifacts(ctx, loaded.ResolvePath(loaded.Config.State.DBPath), loaded.WorkspaceRoot, loaded.Config.Generate.Lean4.ProjectFolder, artifactRoot, normalizedGenConfig, theoremsPath, configOutPath, payloadTemplatePath, time.Now().UTC())
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"theorems_file": theoremsPath})
		return err
	}
	packID := persisted.TheoremPackID
	if err := store.RecordGeneratedPack(ctx, state.GeneratedPack{
		RunID:         plan.RunID,
		PackKey:       packID,
		ProjectFolder: loaded.Config.Generate.Lean4.ProjectFolder,
		SourceKind:    normalizedGenConfig.Mode,
		SourcePath:    normalizedGenConfig.SourceFile,
		OutputPath:    theoremsPath,
		ManifestPath:  configOutPath,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"payload_template":  payloadTemplatePath,
			"filters":           normalizedGenConfig.Filters,
			"atoms":             normalizedGenConfig.Atoms,
			"generation_job_id": persisted.Result.GenerationJobID,
			"hypothesis_set_id": persisted.Result.HypothesisSetID,
			"historical_path":   persisted.HistoricalPath,
		}),
	}); err != nil {
		return err
	}

	exportSource := strings.TrimSpace(*exportSourceFile)
	if exportSource == "" {
		exportSource = theoremsPath
	} else {
		exportSource = resolveLeanSourcePath(loaded, exportSource)
	}
	resolvedOutputArg := strings.TrimSpace(*outputFile)
	if resolvedOutputArg != "" {
		resolvedOutputArg = resolveLeanExportOutputPath(loaded, resolvedOutputArg)
	} else {
		resolvedOutputArg = plan.ExportOutputPath
	}
	exportArtifactKey := artifactkey.Export(plan.RunID, plan.RunID)
	resolvedOutputArg = compactArtifactOutputPath(loaded, resolvedOutputArg, "export", exportArtifactKey)
	exported, err := leanlayer.ExportToBenchmark(loaded, exportSource, resolvedOutputArg, lean4worker.BenchmarkCaseExportConfig{
		ModeFilter:      strings.TrimSpace(*exportMode),
		InterestingOnly: *interestingOnly,
		MinimalOnly:     *minimalOnly,
		Limit:           *exportLimit,
	})
	if err != nil {
		_, _ = finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, err, map[string]any{"pack_key": packID})
		return err
	}
	resolvedSource := exportSource
	if strings.TrimSpace(*exportSourceFile) != "" {
		resolvedSource = exportSource
	}
	resolvedOutput := plan.ExportOutputPath
	if strings.TrimSpace(*outputFile) != "" {
		resolvedOutput = resolvedOutputArg
	}
	if err := store.RecordExport(ctx, state.Export{
		RunID:                  plan.RunID,
		ExportKey:              plan.RunID,
		ProjectFolder:          loaded.Config.Generate.Lean4.ProjectFolder,
		BenchmarkProjectFolder: loaded.Config.Generate.Lean4.BenchmarkProjectFolder,
		SourcePackID:           packID,
		SourcePath:             resolvedSource,
		OutputPath:             resolvedOutput,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"artifact_key":     exportArtifactKey,
			"mode_filter":      strings.TrimSpace(*exportMode),
			"interesting_only": *interestingOnly,
			"minimal_only":     *minimalOnly,
			"exported_cases":   exported,
		}),
	}); err != nil {
		return err
	}

	for _, artifact := range []state.Artifact{
		{
			RunID:        plan.RunID,
			Role:         "lean-workspace",
			AbsolutePath: bootstrapResult.WorkspaceDir,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-job-artifact-dir",
			AbsolutePath: jobResult.ArtifactDir,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-generated-module",
			AbsolutePath: jobResult.ModulePath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-job-report",
			AbsolutePath: jobResult.ReportPath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-theorem-pack",
			AbsolutePath: theoremsPath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-historical-theorem-pack",
			AbsolutePath: persisted.HistoricalPath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-generation-config",
			AbsolutePath: configOutPath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-payload-template",
			AbsolutePath: payloadTemplatePath,
		},
		{
			RunID:        plan.RunID,
			Role:         "lean-benchmark-export",
			AbsolutePath: resolvedOutput,
		},
	} {
		if err := store.RecordArtifact(ctx, artifact); err != nil {
			return err
		}
	}
	if err := store.RecordArtifact(ctx, state.Artifact{
		RunID:        plan.RunID,
		Role:         "lean-job-payload",
		AbsolutePath: plan.JobPayloadPath,
	}); err != nil {
		return err
	}

	if _, err := finishRun(ctx, store, plan.RunID, "phase3 run", plan.Phase, plan.ManifestPath, nil, map[string]any{
		"workspace_dir":          bootstrapResult.WorkspaceDir,
		"project_dir":            projectDir,
		"artifact_dir":           jobResult.ArtifactDir,
		"module_path":            jobResult.ModulePath,
		"report_path":            jobResult.ReportPath,
		"job_ok":                 jobResult.OK,
		"job_exit_code":          jobResult.ExitCode,
		"pack_key":               packID,
		"theorems_file":          theoremsPath,
		"generation_config":      configOutPath,
		"payload_template":       payloadTemplatePath,
		"export_output":          resolvedOutput,
		"benchmark_cases":        exported,
		"benchmark_project":      loaded.Config.Generate.Lean4.BenchmarkProjectFolder,
		"job_payload_path":       plan.JobPayloadPath,
		"job_statement":          plan.JobStatement,
		"generation_mode":        genConfig.Mode,
		"generation_source_file": genConfig.SourceFile,
		"export_source_file":     resolvedSource,
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "phase3 workflow completed\nrun_id: %s\nmanifest_path: %s\nworkspace_dir: %s\njob_report_path: %s\ntheorem_pack_path: %s\nexport_output_path: %s\npack_key: %s\nbenchmark_cases: %d\n",
		plan.RunID,
		filepath.ToSlash(plan.ManifestPath),
		filepath.ToSlash(bootstrapResult.WorkspaceDir),
		filepath.ToSlash(jobResult.ReportPath),
		filepath.ToSlash(theoremsPath),
		filepath.ToSlash(resolvedOutput),
		packID,
		exported,
	)
	return nil
}

func buildPairedPlan(phase string, loaded researchconfig.Loaded, opts planner.WorkflowPlanOptions, sourceFile string) (*planner.PairedBenchmarkPlan, error) {
	switch phase {
	case "phase2":
		return planner.BuildPhase2Plan(loaded, opts)
	case "phase3-compare":
		return planner.BuildPhase3ComparePlan(loaded, opts, sourceFile)
	default:
		return nil, fmt.Errorf("unsupported paired research workflow %q", phase)
	}
}

func finishWorkflowRun(ctx context.Context, store *state.Store, plan *planner.PairedBenchmarkPlan, loaded researchconfig.Loaded, directResult, ndResult runner.BenchmarkRunResult, runErr error, pairReportPath, pairCatalogPath string, partial bool) (string, error) {
	status := "completed"
	if isInterruptedCommand(ctx, runErr) {
		status = "aborted"
	} else if runErr != nil {
		if partial {
			status = "partial_failed"
		} else {
			status = "failed"
		}
	}
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	if err := store.UpsertRun(controlContext(ctx), state.Run{
		RunID:             plan.RunID,
		Command:           plan.Command,
		Phase:             plan.Phase,
		Status:            status,
		ManifestPath:      plan.ManifestPath,
		ConfigFingerprint: loaded.Fingerprint,
		FinishedAtUTC:     time.Now().UTC().Format(time.RFC3339),
		Error:             errText,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"direct_run_id":         plan.Direct.RunID,
			"nd_run_id":             plan.ND.RunID,
			"direct_summary_path":   plan.Direct.SummaryPath,
			"nd_summary_path":       plan.ND.SummaryPath,
			"pair_report_path":      pairReportPath,
			"pair_model_catalog":    pairCatalogPath,
			"direct_completed_jobs": directResult.CompletedJobs,
			"direct_skipped_jobs":   directResult.SkippedJobs,
			"direct_failed_jobs":    directResult.FailedJobs,
			"direct_aborted_jobs":   directResult.AbortedJobs,
			"nd_completed_jobs":     ndResult.CompletedJobs,
			"nd_skipped_jobs":       ndResult.SkippedJobs,
			"nd_failed_jobs":        ndResult.FailedJobs,
			"nd_aborted_jobs":       ndResult.AbortedJobs,
		}),
	}); err != nil {
		if runErr != nil {
			return status, fmt.Errorf("%w; finalize workflow run: %v", runErr, err)
		}
		return status, err
	}
	return status, runErr
}

func combineWorkflowErrors(errs ...error) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		parts = append(parts, err.Error())
	}
	if len(parts) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}
