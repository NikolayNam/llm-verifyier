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
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	leanlayer "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/lean4"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func runGenerateSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("generate requires: lean <bootstrap|job|cases> | benchmark cases | compositional cases | bridge-final hypotheses")
	}
	switch args[0] {
	case "lean":
		if len(args) < 2 {
			return fmt.Errorf("generate requires: lean <bootstrap|job|cases>")
		}
		switch args[1] {
		case "bootstrap":
			return runLeanBootstrapCommand(args[2:], stdout, stderr)
		case "job":
			return runLeanJobCommand(args[2:], stdout, stderr)
		case "cases":
			return runLeanCasesCommand(args[2:], stdout, stderr)
		default:
			return fmt.Errorf("unknown generate lean subcommand %q", args[1])
		}
	case "benchmark":
		if len(args) < 2 || args[1] != "cases" {
			return fmt.Errorf("generate benchmark requires: cases")
		}
		return runGenerateBenchmarkCasesCommand(args[2:], stdout, stderr)
	case "compositional":
		if len(args) < 2 || args[1] != "cases" {
			return fmt.Errorf("generate compositional requires: cases")
		}
		return runGenerateCompositionalCasesCommand(args[2:], stdout, stderr)
	case "bridge-final":
		if len(args) < 2 || args[1] != "hypotheses" {
			return fmt.Errorf("generate bridge-final requires: hypotheses")
		}
		return runGenerateBridgeFinalHypothesesCommand(args[2:], stdout, stderr)
	default:
		return fmt.Errorf("generate requires: lean <bootstrap|job|cases> | benchmark cases | compositional cases | bridge-final hypotheses")
	}
}

func runLeanBootstrapCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate-lean-bootstrap", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable Lean bootstrap identifier")

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
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "generate lean bootstrap", "lean-bootstrap", ""); err != nil {
		return err
	}

	result, err := leanlayer.Bootstrap(context.Background(), loaded, strings.TrimSpace(*runID))
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "generate lean bootstrap", "lean-bootstrap", "", err, map[string]any{"workspace_dir": result.WorkspaceDir})
		return err
	}
	if _, err := finishRun(context.Background(), store, *runID, "generate lean bootstrap", "lean-bootstrap", "", nil, map[string]any{"workspace_dir": result.WorkspaceDir}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "lean generation workspace bootstrapped\nrun_id: %s\nworkspace_dir: %s\n", *runID, result.WorkspaceDir)
	return nil
}

func runLeanJobCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate-lean-job", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable Lean job identifier")
	name := fs.String("name", "", "Lean4 job name")
	statement := fs.String("statement", "", "Lean theorem statement")
	payloadFile := fs.String("payload-file", "", "Optional JSON payload file")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("generate lean job requires --name")
	}
	if strings.TrimSpace(*statement) == "" {
		return fmt.Errorf("generate lean job requires --statement")
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "generate lean job", "lean-job", ""); err != nil {
		return err
	}

	var payload []byte
	if strings.TrimSpace(*payloadFile) != "" {
		payload, err = os.ReadFile(loaded.ResolvePath(*payloadFile))
		if err != nil {
			_, _ = finishRun(context.Background(), store, *runID, "generate lean job", "lean-job", "", err, nil)
			return fmt.Errorf("read payload file: %w", err)
		}
	}
	result, projectDir, err := leanlayer.RunJob(context.Background(), loaded, strings.TrimSpace(*runID), strings.TrimSpace(*name), strings.TrimSpace(*statement), payload)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "generate lean job", "lean-job", "", err, map[string]any{"project_dir": projectDir})
		return err
	}
	for _, artifact := range []state.Artifact{
		{
			RunID:        *runID,
			Role:         "lean-job-artifact-dir",
			AbsolutePath: result.ArtifactDir,
		},
		{
			RunID:        *runID,
			Role:         "lean-generated-module",
			AbsolutePath: result.ModulePath,
		},
		{
			RunID:        *runID,
			Role:         "lean-job-report",
			AbsolutePath: result.ReportPath,
		},
	} {
		if err := store.RecordArtifact(context.Background(), artifact); err != nil {
			return err
		}
	}
	if strings.TrimSpace(*payloadFile) != "" {
		if err := store.RecordArtifact(context.Background(), state.Artifact{
			RunID:        *runID,
			Role:         "lean-payload-source",
			AbsolutePath: loaded.ResolvePath(*payloadFile),
		}); err != nil {
			return err
		}
	}
	if _, err := finishRun(context.Background(), store, *runID, "generate lean job", "lean-job", "", nil, map[string]any{
		"project_dir":  projectDir,
		"artifact_dir": result.ArtifactDir,
		"module_path":  result.ModulePath,
		"report_path":  result.ReportPath,
		"ok":           result.OK,
		"exit_code":    result.ExitCode,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "lean research job completed\nrun_id: %s\nproject_dir: %s\nartifact_dir: %s\nmodule_path: %s\nreport_path: %s\nok: %t\nexit_code: %d\n", *runID, filepath.ToSlash(projectDir), filepath.ToSlash(result.ArtifactDir), filepath.ToSlash(result.ModulePath), filepath.ToSlash(result.ReportPath), result.OK, result.ExitCode)
	return nil
}

func runLeanCasesCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate-lean-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable Lean theorem-pack generation identifier")
	mode := fs.String("mode", "", "Hypothesis generation mode: enumerator|curated_backlog|model_proposed")
	atoms := fs.String("atoms", "", "Comma-separated atomic propositions")
	maxFormulaDepth := fs.Int("max-formula-depth", 0, "Maximum implicational formula depth")
	maxAssumptions := fs.Int("max-assumptions", 0, "Maximum assumptions per generated case")
	limit := fs.Int("limit", 0, "Maximum number of generated cases")
	filters := fs.String("filters", "", "Comma-separated generation filters")
	sourceFile := fs.String("source-file", "", "Optional seed theorem CSV for curated_backlog or model_proposed modes")

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
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "generate lean cases", "lean-cases", ""); err != nil {
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
	if *limit > 0 {
		genConfig.Limit = *limit
	}
	if strings.TrimSpace(*filters) != "" {
		genConfig.Filters = splitCSV(*filters)
	}
	if strings.TrimSpace(*sourceFile) != "" {
		genConfig.SourceFile = resolveLeanSourcePath(loaded, strings.TrimSpace(*sourceFile))
	}

	_, normalizedConfig, err := lean4worker.GenerateHypothesisCases(genConfig)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "generate lean cases", "lean-cases", "", err, nil)
		return err
	}
	theoremsPath, configOutPath, payloadPath, err := leanlayer.GenerateCases(loaded, genConfig)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "generate lean cases", "lean-cases", "", err, nil)
		return err
	}
	artifactRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder)))
	persisted, persistErr := persistLeanGenerationFromArtifacts(context.Background(), loaded.ResolvePath(loaded.Config.State.DBPath), loaded.WorkspaceRoot, loaded.Config.Generate.Lean4.ProjectFolder, artifactRoot, normalizedConfig, theoremsPath, configOutPath, payloadPath, time.Now().UTC())
	if persistErr != nil {
		_, _ = finishRun(context.Background(), store, *runID, "generate lean cases", "lean-cases", "", persistErr, nil)
		return persistErr
	}
	packID := persisted.TheoremPackID
	if err := store.RecordGeneratedPack(context.Background(), state.GeneratedPack{
		RunID:         *runID,
		PackKey:       packID,
		ProjectFolder: loaded.Config.Generate.Lean4.ProjectFolder,
		SourceKind:    normalizedConfig.Mode,
		SourcePath:    normalizedConfig.SourceFile,
		OutputPath:    theoremsPath,
		ManifestPath:  configOutPath,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"payload_template":  payloadPath,
			"filters":           normalizedConfig.Filters,
			"atoms":             normalizedConfig.Atoms,
			"generation_job_id": persisted.Result.GenerationJobID,
			"hypothesis_set_id": persisted.Result.HypothesisSetID,
			"historical_path":   persisted.HistoricalPath,
		}),
	}); err != nil {
		return err
	}
	for _, artifact := range []state.Artifact{
		{
			RunID:        *runID,
			Role:         "lean-theorem-pack",
			AbsolutePath: theoremsPath,
		},
		{
			RunID:        *runID,
			Role:         "lean-generation-config",
			AbsolutePath: configOutPath,
		},
		{
			RunID:        *runID,
			Role:         "lean-historical-theorem-pack",
			AbsolutePath: persisted.HistoricalPath,
		},
		{
			RunID:        *runID,
			Role:         "lean-payload-template",
			AbsolutePath: payloadPath,
		},
	} {
		if err := store.RecordArtifact(context.Background(), artifact); err != nil {
			return err
		}
	}
	if _, err := finishRun(context.Background(), store, *runID, "generate lean cases", "lean-cases", "", nil, map[string]any{
		"pack_key":          packID,
		"theorems_file":     theoremsPath,
		"generation_config": configOutPath,
		"payload_template":  payloadPath,
		"generation_mode":   genConfig.Mode,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "lean theorem pack generated\nrun_id: %s\npack_key: %s\ntheorems_file: %s\ngeneration_config: %s\ndefault_payload: %s\n", *runID, packID, filepath.ToSlash(theoremsPath), filepath.ToSlash(configOutPath), filepath.ToSlash(payloadPath))
	return nil
}

func runExportSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 || args[0] != "lean-to-benchmark" {
		return fmt.Errorf("export requires: lean-to-benchmark")
	}
	return runLeanExportCommand(args[1:], stdout, stderr)
}

func runLeanExportCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("export-lean-to-benchmark", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable Lean export identifier")
	sourceFile := fs.String("source-file", "", "Lean theorem CSV to export")
	outputFile := fs.String("output-file", "", "Benchmark-relative or absolute output CSV path")
	mode := fs.String("mode", "", "Optional generation_mode filter")
	interestingOnly := fs.Bool("interesting-only", false, "Export only interesting hypotheses")
	minimalOnly := fs.Bool("minimal-only", false, "Export only minimal hypotheses")
	limit := fs.Int("limit", 0, "Maximum number of exported benchmark cases")

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
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "export lean-to-benchmark", "lean-export", ""); err != nil {
		return err
	}

	resolvedSourceArg := strings.TrimSpace(*sourceFile)
	if resolvedSourceArg != "" {
		resolvedSourceArg = resolveLeanSourcePath(loaded, resolvedSourceArg)
	}
	resolvedOutputArg := strings.TrimSpace(*outputFile)
	if resolvedOutputArg != "" {
		resolvedOutputArg = resolveLeanExportOutputPath(loaded, resolvedOutputArg)
	}
	exported, err := leanlayer.ExportToBenchmark(loaded, resolvedSourceArg, resolvedOutputArg, lean4worker.BenchmarkCaseExportConfig{
		ModeFilter:      strings.TrimSpace(*mode),
		InterestingOnly: *interestingOnly,
		MinimalOnly:     *minimalOnly,
		Limit:           *limit,
	})
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "export lean-to-benchmark", "lean-export", "", err, nil)
		return err
	}

	resolvedSource := strings.TrimSpace(*sourceFile)
	if resolvedSource == "" {
		resolvedSource = filepath.ToSlash(filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder))), lean4worker.DefaultTheoremsDir, "theorems.csv"))
	} else {
		resolvedSource = filepath.ToSlash(resolveLeanSourcePath(loaded, resolvedSource))
	}
	resolvedOutput := strings.TrimSpace(*outputFile)
	if resolvedOutput == "" {
		resolvedOutput = filepath.ToSlash(filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.BenchmarkProjectFolder))), filepath.FromSlash(loaded.Config.Generate.Lean4.ExportOutputFile)))
	} else {
		resolvedOutput = filepath.ToSlash(resolveLeanExportOutputPath(loaded, resolvedOutput))
	}
	sourcePackID, packErr := loadFirstTheoremPackID(resolvedSource)
	if packErr != nil {
		_, _ = finishRun(context.Background(), store, *runID, "export lean-to-benchmark", "lean-export", "", packErr, nil)
		return packErr
	}
	if err := store.RecordExport(context.Background(), state.Export{
		RunID:                  *runID,
		ExportKey:              *runID,
		ProjectFolder:          loaded.Config.Generate.Lean4.ProjectFolder,
		BenchmarkProjectFolder: loaded.Config.Generate.Lean4.BenchmarkProjectFolder,
		SourcePackID:           sourcePackID,
		SourcePath:             resolvedSource,
		OutputPath:             resolvedOutput,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"mode_filter":      strings.TrimSpace(*mode),
			"interesting_only": *interestingOnly,
			"minimal_only":     *minimalOnly,
			"exported_cases":   exported,
		}),
	}); err != nil {
		return err
	}
	for _, artifact := range []state.Artifact{
		{
			RunID:        *runID,
			Role:         "lean-export-source",
			AbsolutePath: resolvedSource,
		},
		{
			RunID:        *runID,
			Role:         "lean-benchmark-export",
			AbsolutePath: resolvedOutput,
		},
	} {
		if err := store.RecordArtifact(context.Background(), artifact); err != nil {
			return err
		}
	}
	if _, err := finishRun(context.Background(), store, *runID, "export lean-to-benchmark", "lean-export", "", nil, map[string]any{
		"source_pack_id":    sourcePackID,
		"source_file":       resolvedSource,
		"output_file":       resolvedOutput,
		"exported_theorems": exported,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "lean theorem pack exported\nrun_id: %s\nsource_pack_id: %s\nsource_file: %s\noutput_file: %s\nexported_theorems: %d\n", *runID, sourcePackID, filepath.ToSlash(resolvedSource), filepath.ToSlash(resolvedOutput), exported)
	return nil
}

func runDBSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("db requires: sync|prune|overview|runs")
	}
	switch args[0] {
	case "sync":
		return runDBSyncCommand(args[1:], stdout, stderr)
	case "prune":
		return runDBPruneCommand(args[1:], stdout, stderr)
	case "overview":
		return runDBOverviewCommand(args[1:], stdout, stderr)
	case "runs":
		return runDBRunsCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("db requires: sync|prune|overview|runs")
	}
}

func runDBSyncCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("db-sync", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable DB sync identifier")
	projectFolder := fs.String("project-folder", "", "Research project folder under research/artifacts")
	dbPath := fs.String("db", "", "SQLite research DB path")

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
	if strings.TrimSpace(*projectFolder) == "" {
		*projectFolder = loaded.Config.DB.ProjectFolder
	}
	resolvedDBPath := strings.TrimSpace(*dbPath)
	if resolvedDBPath == "" {
		resolvedDBPath = loaded.ResolvePath(loaded.Config.State.DBPath)
	} else {
		resolvedDBPath = loaded.ResolvePath(resolvedDBPath)
	}
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, resolvedDBPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "db sync", "db-sync", ""); err != nil {
		return err
	}

	result, err := researchdb.SyncProject(context.Background(), resolvedDBPath, researchdb.SyncOptions{
		WorkspaceRoot: loaded.WorkspaceRoot,
		ProjectFolder: strings.TrimSpace(*projectFolder),
	})
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "db sync", "db-sync", "", err, nil)
		return err
	}
	if _, err := finishRun(context.Background(), store, *runID, "db sync", "db-sync", "", nil, map[string]any{
		"db_path":              result.DBPath,
		"research_slug":        result.ResearchSlug,
		"theorem_files":        result.TheoremFiles,
		"theorems_imported":    result.TheoremsImported,
		"result_files":         result.ResultFiles,
		"result_rows_imported": result.ResultRowsImported,
		"chain_summary_files":  result.ChainSummaryFiles,
		"chain_rows_imported":  result.ChainRowsImported,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research db sync completed\nrun_id: %s\ndb_path: %s\nresearch_slug: %s\ntheorem_files: %d\ntheorems_imported: %d\nresult_files: %d\nresult_rows_imported: %d\nchain_summary_files: %d\nchain_rows_imported: %d\n",
		*runID, filepath.ToSlash(result.DBPath), result.ResearchSlug, result.TheoremFiles, result.TheoremsImported, result.ResultFiles, result.ResultRowsImported, result.ChainSummaryFiles, result.ChainRowsImported)
	return nil
}

func runDBPruneCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("db-prune", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	runID := fs.String("run-id", timestampRunID(), "Stable DB prune identifier")
	projectFolder := fs.String("project-folder", "", "Research project folder under research/artifacts")
	dbPath := fs.String("db", "", "SQLite research DB path")
	apply := fs.Bool("apply", false, "Delete stale indexed rows instead of printing a dry-run summary")
	vacuum := fs.Bool("vacuum", false, "Run VACUUM after an applied prune")

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
	if strings.TrimSpace(*projectFolder) == "" {
		*projectFolder = loaded.Config.DB.ProjectFolder
	}
	resolvedDBPath := strings.TrimSpace(*dbPath)
	if resolvedDBPath == "" {
		resolvedDBPath = loaded.ResolvePath(loaded.Config.State.DBPath)
	} else {
		resolvedDBPath = loaded.ResolvePath(resolvedDBPath)
	}
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, resolvedDBPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := startRun(context.Background(), store, loaded, *runID, "db prune", "db-prune", ""); err != nil {
		return err
	}

	result, err := researchdb.PruneProject(context.Background(), resolvedDBPath, researchdb.PruneOptions{
		WorkspaceRoot: loaded.WorkspaceRoot,
		ProjectFolder: strings.TrimSpace(*projectFolder),
		Apply:         *apply,
		Vacuum:        *vacuum,
	})
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, "db prune", "db-prune", "", err, nil)
		return err
	}
	if _, err := finishRun(context.Background(), store, *runID, "db prune", "db-prune", "", nil, map[string]any{
		"db_path":                    result.DBPath,
		"research_slug":              result.ResearchSlug,
		"dry_run":                    result.DryRun,
		"missing_theorem_files":      result.MissingTheoremFiles,
		"missing_result_files":       result.MissingResultFiles,
		"deleted_theorem_files":      result.DeletedTheoremFiles,
		"deleted_result_files":       result.DeletedResultFiles,
		"deleted_theorem_rows":       result.DeletedTheoremRows,
		"deleted_summary_rows":       result.DeletedSummaryRows,
		"deleted_result_rows":        result.DeletedResultRows,
		"deleted_chain_summary_rows": result.DeletedChainSummaryRows,
		"dropped_legacy_union_table": result.DeletedLegacyUnionTable,
		"vacuumed":                   result.Vacuumed,
	}); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "research db prune completed\nrun_id: %s\ndb_path: %s\nresearch_slug: %s\ndry_run: %t\nmissing_theorem_files: %d\nmissing_result_files: %d\ndeleted_theorem_files: %d\ndeleted_result_files: %d\ndeleted_theorem_rows: %d\ndeleted_summary_rows: %d\ndeleted_result_rows: %d\ndeleted_chain_summary_rows: %d\ndropped_legacy_union_table: %t\nvacuumed: %t\n",
		*runID, filepath.ToSlash(result.DBPath), result.ResearchSlug, result.DryRun, result.MissingTheoremFiles, result.MissingResultFiles, result.DeletedTheoremFiles, result.DeletedResultFiles, result.DeletedTheoremRows, result.DeletedSummaryRows, result.DeletedResultRows, result.DeletedChainSummaryRows, result.DeletedLegacyUnionTable, result.Vacuumed)
	return nil
}

func loadFirstTheoremPackID(path string) (string, error) {
	cases, err := lean4worker.LoadHypothesisCasesCSV(path)
	if err != nil {
		return "", err
	}
	for _, hypothesis := range cases {
		if strings.TrimSpace(hypothesis.TheoremPackID) != "" {
			return strings.TrimSpace(hypothesis.TheoremPackID), nil
		}
	}
	return "", fmt.Errorf("no theorem_pack_id found in %s", path)
}

func resolveLeanSourcePath(loaded researchconfig.Loaded, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(filepath.FromSlash(raw)) {
		return filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	}
	if strings.HasPrefix(raw, "research/") || strings.HasPrefix(raw, ".tmp/") {
		return filepath.ToSlash(loaded.ResolvePath(raw))
	}
	return filepath.ToSlash(filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder))), filepath.FromSlash(raw)))
}

func resolveLeanExportOutputPath(loaded researchconfig.Loaded, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(filepath.FromSlash(raw)) {
		return filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	}
	if strings.HasPrefix(raw, "research/") || strings.HasPrefix(raw, ".tmp/") {
		return filepath.ToSlash(loaded.ResolvePath(raw))
	}
	return filepath.ToSlash(filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.BenchmarkProjectFolder))), filepath.FromSlash(raw)))
}
