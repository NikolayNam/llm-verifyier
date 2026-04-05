package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	researchanalysis "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/analysis"
	leanlayer "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/lean4"
)

func runAnalyzeSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("analyze requires: gold-first-phase2|gold-first-phase2-hard-pack|lean-kernel-parity")
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "gold-first-phase2", "gold_first_phase2":
		return runAnalyzeGoldFirstPhase2Command(args[1:], stdout, stderr)
	case "gold-first-phase2-hard-pack", "gold_first_phase2_hard_pack":
		return runAnalyzeGoldFirstPhase2HardPackCommand(args[1:], stdout, stderr)
	case "lean-kernel-parity", "lean_kernel_parity":
		return runAnalyzeLeanKernelParityCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown analyze subcommand %q", args[0])
	}
}

func runAnalyzeLeanKernelParityCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("analyze-lean-kernel-parity", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	projectFolder := fs.String("project-folder", researchanalysis.LeanKernelParityProjectFolderDefault, "Lean parity project folder under research/artifacts")
	corpusRoot := fs.String("corpus-root", filepath.ToSlash(filepath.Join("platform-tooling", "internal", "certificates", "testdata")), "Root directory with fixed certificate corpus")
	corpusLimit := fs.Int("corpus-limit", 0, "Optional deterministic cap: select the first N sorted JSON files under corpus-root")
	leanBatchSize := fs.Int("lean-batch-size", 100, "Max number of cases per generated Lean module; larger corpora are run as multiple batches")
	workspaceDir := fs.String("workspace-dir", "", "Optional existing Lean project directory with lakefile.lean to reuse instead of the default .tmp workspace")
	runID := fs.String("run-id", "", "Stable parity run identifier")
	timeout := fs.Duration("timeout", 0, "Optional Lean execution timeout override")

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
	if strings.TrimSpace(*runID) == "" {
		*runID = timestampRunID()
	}
	resolvedTimeout := *timeout
	if resolvedTimeout <= 0 {
		resolvedTimeout, err = time.ParseDuration(loaded.Config.Generate.Lean4.Timeout)
		if err != nil {
			return fmt.Errorf("parse lean timeout from config: %w", err)
		}
	}

	parityLoaded := loaded
	parityLoaded.Config.Generate.Lean4.ProjectFolder = strings.TrimSpace(*projectFolder)
	resolvedWorkspaceDir := strings.TrimSpace(*workspaceDir)
	if resolvedWorkspaceDir == "" {
		resolvedWorkspaceDir = leanlayer.WorkspaceDir(parityLoaded, strings.TrimSpace(*runID))
	} else {
		resolvedWorkspaceDir = loaded.ResolvePath(resolvedWorkspaceDir)
	}
	resultDir := loaded.ResolvePath(filepath.ToSlash(filepath.Join("research", "artifacts", strings.TrimSpace(*projectFolder), "result", strings.TrimSpace(*runID), "kernel_parity")))

	artifacts, err := researchanalysis.AnalyzeLeanKernelParity(commandContext(), researchanalysis.LeanKernelParityOptions{
		WorkspaceDir:  resolvedWorkspaceDir,
		ResultDir:     resultDir,
		ProjectFolder: strings.TrimSpace(*projectFolder),
		RunID:         strings.TrimSpace(*runID),
		CorpusRoot:    loaded.ResolvePath(*corpusRoot),
		MaxCases:      *corpusLimit,
		LeanBatchSize: *leanBatchSize,
		LakeBinary:    loaded.Config.Generate.Lean4.LakeBinary,
		Timeout:       resolvedTimeout,
	})
	if err != nil {
		return err
	}

	behaviorMismatches := 0
	detailMismatches := 0
	for _, row := range artifacts.ComparisonRows {
		if !row.BehaviorMatch {
			behaviorMismatches++
		}
		if !row.DetailMatch {
			detailMismatches++
		}
	}

	_, _ = fmt.Fprintf(stdout, "research analysis completed\nanalysis: lean-kernel-parity\nrun_id: %s\nproject_folder: %s\ncorpus_root: %s\ntotal_cases: %d\nbehavior_mismatches: %d\ndetail_mismatches: %d\ncomparison_csv: %s\nmismatch_manifest: %s\nmismatch_cases_dir: %s\nmismatch_summary: %s\nreport_md: %s\nresult_dir: %s\n",
		artifacts.RunID,
		artifacts.ProjectFolder,
		artifacts.CorpusRoot,
		len(artifacts.ComparisonRows),
		behaviorMismatches,
		detailMismatches,
		artifacts.ComparisonCSVPath,
		artifacts.MismatchManifestPath,
		artifacts.MismatchCasesDir,
		artifacts.MismatchSummaryPath,
		artifacts.ReportMarkdownPath,
		artifacts.ResultDir,
	)
	return nil
}

func runAnalyzeGoldFirstPhase2Command(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("analyze-gold-first-phase2", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", compositionalConfigPath("mixed-family-gold-first-phase2.yaml"), "Optional local YAML override")
	summaryDir := fs.String("summary-dir", "", "Optional override for the phase2 summary directory")
	validityCSV := fs.String("validity-csv", filepath.ToSlash(filepath.Join("research", "artifacts", "analysis", "gold_first_phase2_run_validity.csv")), "Output CSV for P0.2 run validity classification")
	validityMD := fs.String("validity-md", filepath.ToSlash(filepath.Join("research", "artifacts", "analysis", "gold_first_phase2_run_validity.md")), "Output markdown for P0.2 run validity classification")
	subgroupCSV := fs.String("subgroup-csv", filepath.ToSlash(filepath.Join("research", "artifacts", "analysis", "gold_first_phase2_subgroup_analysis.csv")), "Output CSV for P0.3 subgroup analysis")
	subgroupMD := fs.String("subgroup-md", filepath.ToSlash(filepath.Join("research", "artifacts", "analysis", "gold_first_phase2_subgroup_analysis.md")), "Output markdown for P0.3 subgroup analysis")

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
	benchmarkConfig, err := selectBenchmark(researchanalysis.GoldFirstPhase2Phase, loaded.Config)
	if err != nil {
		return err
	}
	resolvedSummaryDir := strings.TrimSpace(*summaryDir)
	if resolvedSummaryDir == "" {
		resolvedSummaryDir = loaded.ResolvePath(benchmarkConfig.SummaryDir)
	} else {
		resolvedSummaryDir = loaded.ResolvePath(resolvedSummaryDir)
	}

	artifacts, err := researchanalysis.AnalyzeGoldFirstPhase2(commandContext(), researchanalysis.GoldFirstPhase2Options{
		Phase:                researchanalysis.GoldFirstPhase2Phase,
		SummaryDir:           resolvedSummaryDir,
		ValidityCSVPath:      loaded.ResolvePath(*validityCSV),
		ValidityMarkdownPath: loaded.ResolvePath(*validityMD),
		SubgroupCSVPath:      loaded.ResolvePath(*subgroupCSV),
		SubgroupMarkdownPath: loaded.ResolvePath(*subgroupMD),
	})
	if err != nil {
		return err
	}

	reasoningValid := 0
	executionInvalidated := 0
	ambiguous := 0
	for _, row := range artifacts.ValidityRows {
		switch row.Classification {
		case "reasoning-valid":
			reasoningValid++
		case "execution-invalidated":
			executionInvalidated++
		default:
			ambiguous++
		}
	}

	_, _ = fmt.Fprintf(stdout, "research analysis completed\nanalysis: gold-first-phase2\nsummary_dir: %s\nsummary_files: %d\nvalidity_rows: %d\nreasoning_valid_runs: %d\nexecution_invalidated_runs: %d\nambiguous_runs: %d\nvalidity_csv: %s\nvalidity_md: %s\nsubgroup_csv: %s\nsubgroup_md: %s\n",
		filepath.ToSlash(artifacts.SummaryDir),
		len(artifacts.SummaryFiles),
		len(artifacts.ValidityRows),
		reasoningValid,
		executionInvalidated,
		ambiguous,
		filepath.ToSlash(artifacts.ValidityCSVPath),
		filepath.ToSlash(artifacts.ValidityMarkdownPath),
		filepath.ToSlash(artifacts.SubgroupCSVPath),
		filepath.ToSlash(artifacts.SubgroupMarkdownPath),
	)
	return nil
}

func runAnalyzeGoldFirstPhase2HardPackCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("analyze-gold-first-phase2-hard-pack", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", compositionalConfigPath("mixed-family-gold-first-hard-phase2.yaml"), "Optional local YAML override")
	summaryDir := fs.String("summary-dir", "", "Optional override for the hard-pack summary directory")
	casesFile := fs.String("cases-file", researchanalysis.GoldFirstPhase2HardPackCasesFile, "Benchmark cases_file selector for the hard pack")
	reportMD := fs.String("report-md", filepath.ToSlash(filepath.Join("research", "artifacts", "analysis", "gold_first_phase2_hard_pack_report.md")), "Output markdown report for the hard phase2 pack")

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
	benchmarkConfig, err := selectBenchmark(researchanalysis.GoldFirstPhase2HardPackPhase, loaded.Config)
	if err != nil {
		return err
	}
	resolvedSummaryDir := strings.TrimSpace(*summaryDir)
	if resolvedSummaryDir == "" {
		resolvedSummaryDir = loaded.ResolvePath(benchmarkConfig.SummaryDir)
	} else {
		resolvedSummaryDir = loaded.ResolvePath(resolvedSummaryDir)
	}

	artifacts, err := researchanalysis.AnalyzeGoldFirstPhase2HardPack(commandContext(), researchanalysis.GoldFirstPhase2HardPackOptions{
		SummaryDir:         resolvedSummaryDir,
		CasesFile:          strings.TrimSpace(*casesFile),
		ReportMarkdownPath: loaded.ResolvePath(*reportMD),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research analysis completed\nanalysis: gold-first-phase2-hard-pack\nsummary_dir: %s\ncases_file: %s\nmatched_summary_files: %d\nmatched_runs: %d\nmodel_rows: %d\nreport_md: %s\n",
		filepath.ToSlash(artifacts.SummaryDir),
		artifacts.CasesFile,
		len(artifacts.SummaryFiles),
		len(artifacts.RunRows),
		len(artifacts.ModelRows),
		filepath.ToSlash(artifacts.ReportMarkdownPath),
	)
	return nil
}
