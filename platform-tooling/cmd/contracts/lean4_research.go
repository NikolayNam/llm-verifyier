package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

const (
	lean4ResearchProjectFolderDefault = "hilbert-ai-verification-lean4-worker-v1"
	lean4ResearchHypothesis           = "Go orchestration plus a Lean4/mathlib worker may provide a mechanized proof/specification sidecar and generate formal case hypotheses without replacing the Hilbert kernel as the approved trust boundary."
)

func runLean4ResearchJobCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	runID := defaultRunID()
	defaultProjectFolder := firstNonEmptyEnv("LEAN4_RESEARCH_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = lean4ResearchProjectFolderDefault
	}
	defaultTimeout := lean4worker.DefaultRunTimeout
	if rawTimeout := firstNonEmptyEnv("LEAN4_JOB_TIMEOUT"); rawTimeout != "" {
		parsedTimeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return fmt.Errorf("parse LEAN4_JOB_TIMEOUT: %w", err)
		}
		defaultTimeout = parsedTimeout
	}

	fs := flag.NewFlagSet("run-lean4-research-job", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Lean4 research project folder under research/artifacts")
	projectDir := fs.String("project-dir", firstNonEmptyEnv("LEAN4_RESEARCH_PROJECT_DIR"), "Lean4 project directory that contains lakefile.lean")
	resultRoot := fs.String("result-root", firstNonEmptyEnv("LEAN4_RESEARCH_RESULT_ROOT"), "Lean4 result artifact root")
	lakeBinary := fs.String("lake-binary", firstNonEmptyEnv("LEAN4_LAKE_BINARY"), "Lean4 lake executable")
	timeout := fs.Duration("timeout", defaultTimeout, "Lean4 job timeout")
	name := fs.String("name", firstNonEmptyEnv("LEAN4_JOB_NAME"), "Lean4 job name")
	statement := fs.String("statement", firstNonEmptyEnv("LEAN4_JOB_STATEMENT"), "Lean theorem statement")
	payloadFile := fs.String("payload-file", firstNonEmptyEnv("LEAN4_JOB_PAYLOAD_FILE"), "Optional JSON payload file with imports/helpers/proof")
	runIDFlag := fs.String("run-id", firstNonEmptyEnv("LEAN4_RUN_ID"), "Stable Lean4 run identifier")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if strings.TrimSpace(*runIDFlag) == "" {
		*runIDFlag = runID
	}

	artifactRoot := benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*projectFolder))
	if strings.TrimSpace(*projectDir) == "" {
		*projectDir = filepath.Join(artifactRoot, lean4worker.DefaultProjectDir)
	}
	if strings.TrimSpace(*resultRoot) == "" {
		*resultRoot = filepath.Join(artifactRoot, lean4worker.DefaultResultDir)
	}
	if strings.TrimSpace(*lakeBinary) == "" {
		*lakeBinary = lean4worker.DefaultLakeBinary
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("lean4 job name is required")
	}
	if strings.TrimSpace(*statement) == "" {
		return fmt.Errorf("lean4 job statement is required")
	}

	var payload []byte
	if strings.TrimSpace(*payloadFile) != "" {
		payload, err = os.ReadFile(*payloadFile)
		if err != nil {
			return fmt.Errorf("read lean4 payload file: %w", err)
		}
	}

	result, err := lean4worker.RunJob(context.Background(), lean4worker.Job{
		Name:      strings.TrimSpace(*name),
		Statement: strings.TrimSpace(*statement),
		Payload:   payload,
	}, lean4worker.Options{
		ProjectDir: *projectDir,
		ResultRoot: *resultRoot,
		RunID:      *runIDFlag,
		LakeBinary: *lakeBinary,
		Timeout:    *timeout,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "lean4 research job completed\nrun_id: %s\nproject_folder: %s\nproject_dir: %s\nresult_root: %s\njob_name: %s\nstatement: %s\nhypothesis: %s\nlake_binary: %s\nok: %t\nexit_code: %d\nartifact_dir: %s\nmodule_path: %s\nreport_path: %s\n", *runIDFlag, strings.TrimSpace(*projectFolder), filepath.ToSlash(*projectDir), filepath.ToSlash(*resultRoot), strings.TrimSpace(*name), strings.TrimSpace(*statement), lean4ResearchHypothesis, strings.TrimSpace(*lakeBinary), result.OK, result.ExitCode, result.ArtifactDir, result.ModulePath, result.ReportPath)
	if strings.TrimSpace(result.Stderr) != "" {
		_, _ = fmt.Fprintf(stdout, "stderr: %s\n", sanitizeNote(result.Stderr))
	}
	return nil
}

func runGenerateLean4HypothesisCasesCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	defaultProjectFolder := firstNonEmptyEnv("LEAN4_RESEARCH_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = lean4ResearchProjectFolderDefault
	}
	defaultConfig := lean4worker.DefaultHypothesisGenerationConfig()
	if rawMode := firstNonEmptyEnv("LEAN4_HYPOTHESIS_MODE"); rawMode != "" {
		defaultConfig.Mode = rawMode
	}
	if rawAtoms := firstNonEmptyEnv("LEAN4_ENUMERATOR_ATOMS"); rawAtoms != "" {
		defaultConfig.Atoms = splitCSV(rawAtoms)
	}
	if rawDepth := firstNonEmptyEnv("LEAN4_ENUMERATOR_MAX_FORMULA_DEPTH"); rawDepth != "" {
		parsedDepth, err := strconv.Atoi(rawDepth)
		if err != nil {
			return fmt.Errorf("parse LEAN4_ENUMERATOR_MAX_FORMULA_DEPTH: %w", err)
		}
		defaultConfig.MaxFormulaDepth = parsedDepth
	}
	if rawAssumptions := firstNonEmptyEnv("LEAN4_ENUMERATOR_MAX_ASSUMPTIONS"); rawAssumptions != "" {
		parsedAssumptions, err := strconv.Atoi(rawAssumptions)
		if err != nil {
			return fmt.Errorf("parse LEAN4_ENUMERATOR_MAX_ASSUMPTIONS: %w", err)
		}
		defaultConfig.MaxAssumptions = parsedAssumptions
	}
	if rawLimit := firstNonEmptyEnv("LEAN4_ENUMERATOR_LIMIT"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return fmt.Errorf("parse LEAN4_ENUMERATOR_LIMIT: %w", err)
		}
		defaultConfig.Limit = parsedLimit
	}
	if rawFilters := firstNonEmptyEnv("LEAN4_ENUMERATOR_FILTERS"); rawFilters != "" {
		defaultConfig.Filters = splitCSV(rawFilters)
	}
	if rawSource := firstNonEmptyEnv("LEAN4_HYPOTHESIS_SOURCE_FILE"); rawSource != "" {
		defaultConfig.SourceFile = rawSource
	}
	if rawSource := firstNonEmptyEnv("LEAN4_THEOREMS_FILE"); rawSource != "" {
		defaultConfig.SourceFile = rawSource
	}

	fs := flag.NewFlagSet("generate-lean4-hypothesis-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Lean4 research project folder under research/artifacts")
	artifactRoot := fs.String("artifact-root", firstNonEmptyEnv("LEAN4_RESEARCH_ARTIFACT_ROOT"), "Lean4 artifact family root")
	projectDir := fs.String("project-dir", firstNonEmptyEnv("LEAN4_RESEARCH_PROJECT_DIR"), "Lean4 project directory")
	dbPath := fs.String("db-path", firstNonEmptyEnv("RESEARCH_DB_PATH"), "SQLite research db path")
	mode := fs.String("mode", defaultConfig.Mode, "Hypothesis generation mode")
	atoms := fs.String("atoms", strings.Join(defaultConfig.Atoms, ","), "Comma-separated atomic propositions for the enumerator")
	maxFormulaDepth := fs.Int("max-formula-depth", defaultConfig.MaxFormulaDepth, "Maximum implicational formula depth")
	maxAssumptions := fs.Int("max-assumptions", defaultConfig.MaxAssumptions, "Maximum assumptions per generated case")
	limit := fs.Int("limit", defaultConfig.Limit, "Maximum number of generated cases")
	filters := fs.String("filters", strings.Join(defaultConfig.Filters, ","), "Comma-separated enumerator filters")
	sourceFile := fs.String("source-file", defaultConfig.SourceFile, "Optional source theorem CSV for curated_backlog or model_proposed modes")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	if strings.TrimSpace(*artifactRoot) == "" {
		*artifactRoot = benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*projectFolder))
	}
	if strings.TrimSpace(*projectDir) == "" {
		*projectDir = filepath.Join(*artifactRoot, lean4worker.DefaultProjectDir)
	}
	if strings.TrimSpace(*dbPath) == "" {
		*dbPath = filepath.Join(workspaceRoot, filepath.FromSlash(researchdb.DefaultDBRelativePath))
	} else {
		*dbPath = resolveBenchmarkArgPath(workspaceRoot, *dbPath)
	}
	if err := lean4worker.EnsureProjectScaffold(*projectDir); err != nil {
		return err
	}
	generationConfig := lean4worker.HypothesisGenerationConfig{
		Mode:            strings.TrimSpace(*mode),
		Atoms:           splitCSV(*atoms),
		MaxFormulaDepth: *maxFormulaDepth,
		MaxAssumptions:  *maxAssumptions,
		Limit:           *limit,
		Filters:         splitCSV(*filters),
		SourceFile:      strings.TrimSpace(*sourceFile),
	}
	cases, normalized, err := lean4worker.GenerateHypothesisCases(generationConfig)
	if err != nil {
		return err
	}
	createdAt := time.Now().UTC()
	theoremPackID, cases := lean4worker.FinalizeHypothesisCases(cases, normalized.Mode, createdAt)
	resolvedSourceFile := strings.TrimSpace(normalized.SourceFile)
	if resolvedSourceFile == "" {
		resolvedSourceFile = lean4worker.DefaultHypothesisSourcePath(*artifactRoot, normalized.Mode)
	}
	if resolvedSourceFile != "" {
		resolvedSourceFile = resolveBenchmarkArgPath(workspaceRoot, resolvedSourceFile)
	}
	normalizedForArtifacts := normalized
	if strings.TrimSpace(normalizedForArtifacts.SourceFile) == "" && resolvedSourceFile != "" {
		normalizedForArtifacts.SourceFile = filepath.ToSlash(resolvedSourceFile)
	}
	artifactPaths, err := lean4worker.WriteHypothesisArtifacts(*artifactRoot, normalizedForArtifacts, cases)
	if err != nil {
		return err
	}

	db, err := researchdb.Open(context.Background(), *dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	persisted, err := researchdb.PersistHypothesisGeneration(context.Background(), db, researchdb.PersistHypothesisGenerationInput{
		WorkspaceRoot:         workspaceRoot,
		ProjectFolder:         strings.TrimSpace(*projectFolder),
		ArtifactRoot:          *artifactRoot,
		SourceFilePath:        resolvedSourceFile,
		HistoricalTheoremPath: artifactPaths.HistoricalPath,
		Config:                normalizedForArtifacts,
		Cases:                 cases,
		GenerationConfigPath:  artifactPaths.GenerationConfigPath,
		DefaultPayloadPath:    artifactPaths.DefaultPayloadPath,
		CreatedAt:             createdAt,
	})
	if err != nil {
		return err
	}

	if err := researchdb.CompleteGenerationJob(context.Background(), db, persisted.GenerationJobID, time.Now().UTC()); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "lean4 theorem backlog generated\nproject_folder: %s\nartifact_root: %s\nproject_dir: %s\ndb_path: %s\ngeneration_job_id: %d\nhypothesis_set_id: %d\ntheorem_pack_id: %s\nhypothesis: %s\ngeneration_mode: %s\ntheorems_file: %s\nhistorical_theorem_file: %s\ngeneration_config: %s\ndefault_payload: %s\n", strings.TrimSpace(*projectFolder), filepath.ToSlash(*artifactRoot), filepath.ToSlash(*projectDir), filepath.ToSlash(*dbPath), persisted.GenerationJobID, persisted.HypothesisSetID, theoremPackID, lean4ResearchHypothesis, normalized.Mode, filepath.ToSlash(artifactPaths.TheoremsPath), filepath.ToSlash(artifactPaths.HistoricalPath), filepath.ToSlash(artifactPaths.GenerationConfigPath), filepath.ToSlash(artifactPaths.DefaultPayloadPath))
	return nil
}

func runExportLean4HypothesisCasesCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	defaultProjectFolder := firstNonEmptyEnv("LEAN4_RESEARCH_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = lean4ResearchProjectFolderDefault
	}
	defaultBenchmarkProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultBenchmarkProjectFolder == "" {
		defaultBenchmarkProjectFolder = "hilbert-ai-verification-benchmark-nd-v1"
	}

	fs := flag.NewFlagSet("export-lean4-hypothesis-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Lean4 research project folder under research/artifacts")
	dbPath := fs.String("db-path", firstNonEmptyEnv("RESEARCH_DB_PATH"), "SQLite research db path")
	sourceFile := fs.String("source-file", firstNonEmptyEnv("LEAN4_THEOREMS_FILE", "LEAN4_HYPOTHESIS_SOURCE_FILE"), "Lean4 theorem backlog CSV to export")
	hypothesisSetID := fs.Int64("hypothesis-set-id", int64(parseIntEnv(firstNonEmptyEnv("LEAN4_HYPOTHESIS_SET_ID"))), "Explicit hypothesis set id to export")
	theoremPackID := fs.String("theorem-pack-id", firstNonEmptyEnv("LEAN4_THEOREM_PACK_ID"), "Explicit theorem_pack_id to export")
	benchmarkProjectFolder := fs.String("benchmark-project-folder", defaultBenchmarkProjectFolder, "Benchmark project folder under research/artifacts")
	outputFile := fs.String("output-file", firstNonEmptyEnv("LEAN4_EXPORT_OUTPUT_FILE"), "Benchmark-relative or absolute output CSV path")
	mode := fs.String("mode", firstNonEmptyEnv("LEAN4_EXPORT_MODE"), "Optional generation_mode filter (enumerator|curated_backlog|model_proposed|*)")
	interestingOnly := fs.Bool("interesting-only", parseBoolEnv(firstNonEmptyEnv("LEAN4_EXPORT_INTERESTING_ONLY")), "Export only interesting hypotheses")
	minimalOnly := fs.Bool("minimal-only", parseBoolEnv(firstNonEmptyEnv("LEAN4_EXPORT_MINIMAL_ONLY")), "Export only minimal hypotheses")
	limit := fs.Int("limit", parseIntEnv(firstNonEmptyEnv("LEAN4_EXPORT_LIMIT")), "Maximum number of exported cases (0 = no limit)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	if strings.TrimSpace(*outputFile) == "" {
		*outputFile = filepath.Join(benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*benchmarkProjectFolder)), "theorems", "lean4-generated.csv")
	} else if !filepath.IsAbs(*outputFile) {
		*outputFile = filepath.Join(benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*benchmarkProjectFolder)), filepath.FromSlash(strings.TrimSpace(*outputFile)))
	}
	if strings.TrimSpace(*dbPath) == "" {
		*dbPath = filepath.Join(workspaceRoot, filepath.FromSlash(researchdb.DefaultDBRelativePath))
	} else {
		*dbPath = resolveBenchmarkArgPath(workspaceRoot, *dbPath)
	}
	if strings.TrimSpace(*sourceFile) != "" {
		*sourceFile = resolveBenchmarkArgPath(workspaceRoot, *sourceFile)
	}

	db, err := researchdb.Open(context.Background(), *dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var (
		hypotheses  []lean4worker.HypothesisCase
		resolvedSet *researchdb.ResolvedHypothesisSet
		snapshotID  int64
	)
	if strings.TrimSpace(*sourceFile) != "" {
		hypotheses, err = lean4worker.LoadHypothesisCasesCSV(*sourceFile)
		if err != nil {
			return err
		}
		if packID := firstNonEmptyHypothesisPackID(hypotheses); packID != "" {
			resolved, resolveErr := researchdb.ResolveHypothesisSet(context.Background(), db, researchdb.HypothesisSetSelector{
				ProjectFolder: strings.TrimSpace(*projectFolder),
				TheoremPackID: packID,
			})
			if resolveErr == nil {
				resolvedSet = &resolved
			}
		}
	} else {
		resolved, err := researchdb.ResolveHypothesisSet(context.Background(), db, researchdb.HypothesisSetSelector{
			ProjectFolder:   strings.TrimSpace(*projectFolder),
			HypothesisSetID: *hypothesisSetID,
			TheoremPackID:   strings.TrimSpace(*theoremPackID),
		})
		if err != nil {
			return err
		}
		resolvedSet = &resolved
		hypotheses, err = researchdb.LoadHypothesisCasesBySet(context.Background(), db, resolved.HypothesisSetID)
		if err != nil {
			return err
		}
	}

	exportConfig := lean4worker.BenchmarkCaseExportConfig{
		ModeFilter:      strings.TrimSpace(*mode),
		InterestingOnly: *interestingOnly,
		MinimalOnly:     *minimalOnly,
		Limit:           *limit,
	}
	if resolvedSet != nil {
		snapshotID, err = researchdb.StartExportSnapshot(context.Background(), db, researchdb.ExportSnapshotInput{
			WorkspaceRoot:          workspaceRoot,
			ResearchID:             resolvedSet.ResearchID,
			HypothesisSetID:        resolvedSet.HypothesisSetID,
			TheoremPackID:          resolvedSet.TheoremPackID,
			BenchmarkProjectFolder: strings.TrimSpace(*benchmarkProjectFolder),
			ModeFilter:             strings.TrimSpace(*mode),
			InterestingOnly:        *interestingOnly,
			MinimalOnly:            *minimalOnly,
			LimitApplied:           *limit,
			OutputPath:             *outputFile,
			CreatedAt:              time.Now().UTC(),
		})
		if err != nil {
			return err
		}
	}
	exported, historicalPath, err := lean4worker.ExportBenchmarkCases(*outputFile, hypotheses, exportConfig)
	if err != nil {
		if snapshotID > 0 {
			_ = researchdb.FailExportSnapshot(context.Background(), db, snapshotID, err.Error(), time.Now().UTC())
		}
		return err
	}
	if snapshotID > 0 {
		if err := researchdb.CompleteExportSnapshot(context.Background(), db, snapshotID, exported, workspaceRoot, historicalPath, time.Now().UTC()); err != nil {
			return err
		}
	}

	modeValue := strings.TrimSpace(*mode)
	if modeValue == "" {
		modeValue = "*"
	}
	sourceValue := strings.TrimSpace(*sourceFile)
	if sourceValue == "" {
		sourceValue = "db"
	}
	resolvedSetID := int64(0)
	resolvedPackID := strings.TrimSpace(*theoremPackID)
	if resolvedSet != nil {
		resolvedSetID = resolvedSet.HypothesisSetID
		resolvedPackID = resolvedSet.TheoremPackID
	}
	_, _ = fmt.Fprintf(stdout, "lean4 theorem backlog exported\nproject_folder: %s\ndb_path: %s\nsource_file: %s\nhypothesis_set_id: %d\ntheorem_pack_id: %s\nbenchmark_project_folder: %s\noutput_file: %s\nhypothesis: %s\nmode_filter: %s\ninteresting_only: %t\nminimal_only: %t\nexported_theorems: %d\n", strings.TrimSpace(*projectFolder), filepath.ToSlash(*dbPath), filepath.ToSlash(sourceValue), resolvedSetID, resolvedPackID, strings.TrimSpace(*benchmarkProjectFolder), filepath.ToSlash(*outputFile), lean4ResearchHypothesis, modeValue, *interestingOnly, *minimalOnly, exported)
	return nil
}

func firstNonEmptyHypothesisPackID(cases []lean4worker.HypothesisCase) string {
	for _, hypothesis := range cases {
		if value := strings.TrimSpace(hypothesis.TheoremPackID); value != "" {
			return value
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func parseBoolEnv(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseIntEnv(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}
