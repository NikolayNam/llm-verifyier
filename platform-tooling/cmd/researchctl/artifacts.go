package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/artifactkey"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

type artifactCompactionPlan struct {
	WorkspaceRoot      string
	ArtifactRoot       string
	ManifestRoot       string
	SummaryFiles       []string
	ManifestFiles      []string
	BenchmarkJobs      map[string]benchmarkArtifactRecord
	ExactMappings      map[string]string
	OrderedMappings    []artifactMappingRecord
	RewriteAliasMap    map[string]string
	OrderedRewriteMap  []artifactMappingRecord
	ResultFiles        []string
	ImportCatalogFiles []string
	MigrationManifest  string
}

type benchmarkArtifactRecord struct {
	OuterRunID                  string
	WorkerRunID                 string
	JobKey                      string
	Phase                       string
	ArtifactKey                 string
	ResultPath                  string
	RawDir                      string
	ChainSummaryPath            string
	ImportCatalogPath           string
	ImportedCertificatesDirPath string
}

type reportArtifactRecord struct {
	Kind string
	Phase string
	RunID string
	Path string
}

type artifactMigrationManifest struct {
	SchemaVersion string                  `json:"schema_version"`
	CreatedAtUTC  string                  `json:"created_at_utc"`
	WorkspaceRoot string                  `json:"workspace_root"`
	Apply         bool                    `json:"apply"`
	RenamedPaths  []artifactMappingRecord `json:"renamed_paths"`
}

type artifactMappingRecord struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func runArtifactsSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("artifacts requires: compact")
	}
	switch args[0] {
	case "compact":
		return runArtifactsCompactCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown artifacts subcommand %q", args[0])
	}
}

func runArtifactsCompactCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("artifacts-compact", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	summaryDir := fs.String("summary-dir", "", "Optional benchmark summary directory to limit benchmark discovery")
	apply := fs.Bool("apply", false, "Apply in-place renames and metadata rewrites; default is dry-run")

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

	plan, err := buildArtifactCompactionPlan(context.Background(), loaded, store, strings.TrimSpace(*summaryDir))
	if err != nil {
		return err
	}
	if err := validateArtifactCompactionPlan(plan); err != nil {
		return err
	}

	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	_, _ = fmt.Fprintf(stdout, "artifact compaction %s ready\nworkspace_root: %s\nbenchmark_jobs: %d\nsummary_files: %d\nmanifest_files: %d\nrenamed_paths: %d\nmigration_manifest: %s\n",
		mode,
		filepath.ToSlash(plan.WorkspaceRoot),
		len(plan.BenchmarkJobs),
		len(plan.SummaryFiles),
		len(plan.ManifestFiles),
		len(plan.ExactMappings),
		filepath.ToSlash(plan.MigrationManifest),
	)
	for _, entry := range sampleArtifactMappings(plan.OrderedMappings, 12) {
		_, _ = fmt.Fprintf(stdout, "rename: %s -> %s\n", entry.Source, entry.Target)
	}
	if !*apply {
		return nil
	}

	if err := applyArtifactCompactionPlan(context.Background(), loaded, store, plan); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "artifact compaction applied\nrenamed_paths: %d\nmigration_manifest: %s\n", len(plan.ExactMappings), filepath.ToSlash(plan.MigrationManifest))
	return nil
}

func buildArtifactCompactionPlan(ctx context.Context, loaded researchconfig.Loaded, store *state.Store, summaryDirOverride string) (*artifactCompactionPlan, error) {
	summaryFiles, err := discoverArtifactSummaryFiles(loaded, summaryDirOverride)
	if err != nil {
		return nil, err
	}
	summaryRecords, err := discoverBenchmarkSummaryRecords(summaryFiles)
	if err != nil {
		return nil, err
	}
	manifestRoot := loaded.ResolvePath(loaded.Config.Paths.ManifestRoot)
	manifestFiles, err := discoverJSONFiles(manifestRoot)
	if err != nil {
		return nil, err
	}
	stateJobs, err := store.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	manifestJobs, err := discoverBenchmarkManifestJobs(manifestFiles)
	if err != nil {
		return nil, err
	}
	benchmarkJobs := mergeBenchmarkArtifactRecords(loaded.WorkspaceRoot, summaryRecords, stateJobs, manifestJobs)
	reportFiles, err := discoverManifestReportArtifacts(loaded, manifestFiles)
	if err != nil {
		return nil, err
	}
	exactMappings := make(map[string]string)

	for _, record := range benchmarkJobs {
		addArtifactMapping(exactMappings, record.ResultPath, benchmarkResultTargetPath(record.ResultPath, record.ArtifactKey))
		addArtifactMapping(exactMappings, record.RawDir, benchmarkRawTargetPath(record.RawDir, record.ArtifactKey))
		addArtifactMapping(exactMappings, record.ChainSummaryPath, benchmarkChainTargetPath(record.ChainSummaryPath, record.ArtifactKey))
		addArtifactMapping(exactMappings, record.ImportCatalogPath, benchmarkImportCatalogTargetPath(record.ImportCatalogPath, record.ArtifactKey))
		addArtifactMapping(exactMappings, record.ImportedCertificatesDirPath, benchmarkImportedCertificatesTargetPath(record.ImportedCertificatesDirPath, record.ArtifactKey))
	}
	for _, record := range reportFiles {
		addArtifactMapping(exactMappings, record.Path, compactReportTargetPath(loaded, record))
	}

	generatedPacks, err := store.ListGeneratedPacks(ctx)
	if err != nil {
		return nil, err
	}
	for _, pack := range generatedPacks {
		key := artifactkey.GeneratedPack(pack.RunID, pack.PackKey)
		addArtifactMapping(exactMappings, pack.OutputPath, compactArtifactOutputPath(loaded, pack.OutputPath, "pack", key))
	}

	exports, err := store.ListExports(ctx)
	if err != nil {
		return nil, err
	}
	for _, export := range exports {
		key := artifactkey.Export(export.RunID, export.ExportKey)
		addArtifactMapping(exactMappings, export.OutputPath, compactArtifactOutputPath(loaded, export.OutputPath, "export", key))
	}

	aliasMap := buildArtifactAliasMapping(loaded.WorkspaceRoot, exactMappings)
	orderedMappings := orderedArtifactMappings(exactMappings)
	orderedAliasMappings := orderedArtifactMappings(aliasMap)
	resultFiles := collectBenchmarkResultFiles(benchmarkJobs, exactMappings, orderedMappings)
	importCatalogFiles := collectBenchmarkImportCatalogFiles(benchmarkJobs, exactMappings, orderedMappings)
	return &artifactCompactionPlan{
		WorkspaceRoot:      loaded.WorkspaceRoot,
		ArtifactRoot:       loaded.ResolvePath(loaded.Config.Paths.ArtifactRoot),
		ManifestRoot:       manifestRoot,
		SummaryFiles:       summaryFiles,
		ManifestFiles:      manifestFiles,
		BenchmarkJobs:      benchmarkJobs,
		ExactMappings:      exactMappings,
		OrderedMappings:    orderedMappings,
		RewriteAliasMap:    aliasMap,
		OrderedRewriteMap:  orderedAliasMappings,
		ResultFiles:        resultFiles,
		ImportCatalogFiles: importCatalogFiles,
		MigrationManifest: filepath.ToSlash(filepath.Join(
			loaded.ResolvePath(loaded.Config.Paths.ManifestRoot),
			fmt.Sprintf("artifact_compaction_%s.json", time.Now().Local().Format("20060102T150405-0700")),
		)),
	}, nil
}

func validateArtifactCompactionPlan(plan *artifactCompactionPlan) error {
	if plan == nil {
		return fmt.Errorf("artifact compaction plan is required")
	}
	targets := make(map[string]string, len(plan.ExactMappings))
	for source, target := range plan.ExactMappings {
		if !pathWithinWorkspace(plan.WorkspaceRoot, source) {
			return fmt.Errorf("artifact compaction source escapes workspace: %s", filepath.ToSlash(source))
		}
		if !pathWithinWorkspace(plan.WorkspaceRoot, target) {
			return fmt.Errorf("artifact compaction target escapes workspace: %s", filepath.ToSlash(target))
		}
		info, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("artifact compaction source missing %s: %w", filepath.ToSlash(source), err)
		}
		if other, ok := targets[target]; ok && other != source {
			return fmt.Errorf("artifact compaction target collision: %s and %s both map to %s", filepath.ToSlash(other), filepath.ToSlash(source), filepath.ToSlash(target))
		}
		targets[target] = source
		if existing, err := os.Stat(target); err == nil {
			if filepath.Clean(source) != filepath.Clean(target) {
				return fmt.Errorf("artifact compaction target already exists: %s", filepath.ToSlash(target))
			}
			if info.IsDir() != existing.IsDir() {
				return fmt.Errorf("artifact compaction target type mismatch: %s", filepath.ToSlash(target))
			}
		}
	}
	return nil
}

func applyArtifactCompactionPlan(ctx context.Context, loaded researchconfig.Loaded, store *state.Store, plan *artifactCompactionPlan) error {
	for _, mapping := range plan.OrderedMappings {
		if err := os.MkdirAll(filepath.Dir(mapping.Target), 0o755); err != nil {
			return fmt.Errorf("create artifact target dir %s: %w", filepath.ToSlash(filepath.Dir(mapping.Target)), err)
		}
		if err := os.Rename(mapping.Source, mapping.Target); err != nil {
			return fmt.Errorf("rename artifact %s -> %s: %w", filepath.ToSlash(mapping.Source), filepath.ToSlash(mapping.Target), err)
		}
	}

	for _, summaryPath := range plan.SummaryFiles {
		if _, err := rewriteCSVPathColumns(summaryPath, []string{"result_file", "chain_result_file", "import_catalog_file", "raw_dir"}, plan.RewriteAliasMap, plan.OrderedRewriteMap); err != nil {
			return err
		}
	}
	for _, resultPath := range plan.ResultFiles {
		if _, err := rewriteCSVPathColumns(resultPath, []string{"certificate_object_file"}, plan.RewriteAliasMap, plan.OrderedRewriteMap); err != nil {
			return err
		}
	}
	for _, catalogPath := range plan.ImportCatalogFiles {
		if _, err := rewriteCSVPathColumns(catalogPath, []string{"certificate_object_file"}, plan.RewriteAliasMap, plan.OrderedRewriteMap); err != nil {
			return err
		}
	}
	for _, manifestPath := range plan.ManifestFiles {
		if _, err := rewriteManifestJSON(manifestPath, plan.BenchmarkJobs, plan.RewriteAliasMap, plan.OrderedRewriteMap); err != nil {
			return err
		}
	}

	if err := store.RewritePaths(ctx, plan.ExactMappings); err != nil {
		return err
	}
	if err := rewriteBenchmarkStateRows(ctx, store, plan); err != nil {
		return err
	}
	if err := writeArtifactMigrationManifest(plan.MigrationManifest, plan.WorkspaceRoot, plan.ExactMappings); err != nil {
		return err
	}
	return nil
}

func discoverArtifactSummaryFiles(loaded researchconfig.Loaded, summaryDirOverride string) ([]string, error) {
	dirs := []string{}
	if strings.TrimSpace(summaryDirOverride) != "" {
		dirs = append(dirs, loaded.ResolvePath(summaryDirOverride))
	} else {
		for _, dir := range benchmarkSummaryDirs(loaded.Config) {
			dirs = append(dirs, loaded.ResolvePath(dir))
		}
	}
	seen := make(map[string]struct{}, len(dirs))
	files := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		matches, err := discoverCSVFiles(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		files = append(files, matches...)
	}
	slices.Sort(files)
	return files, nil
}

func benchmarkSummaryDirs(cfg researchconfig.Config) []string {
	return []string{
		cfg.Benchmarks.Phase1.SummaryDir,
		cfg.Benchmarks.ND.SummaryDir,
		cfg.Benchmarks.CompositionalAssumptionImport.SummaryDir,
		cfg.Benchmarks.BridgeImportFinalResearch.SummaryDir,
		cfg.Benchmarks.BridgeOnlyAuthoring.SummaryDir,
		cfg.Benchmarks.GoldFinalCompositionOnly.SummaryDir,
		cfg.Benchmarks.CompositionalDepthLadder.SummaryDir,
		cfg.Benchmarks.CompositionalBranching.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilyReuse.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.SummaryDir,
		cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.SummaryDir,
	}
}

func discoverCSVFiles(root string) ([]string, error) {
	var files []string
	if strings.TrimSpace(root) == "" {
		return files, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".csv") {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func discoverJSONFiles(root string) ([]string, error) {
	var files []string
	if strings.TrimSpace(root) == "" {
		return files, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".json") {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

type manifestBenchmarkJobRecord struct {
	OuterRunID  string
	WorkerRunID string
	JobKey      string
	Phase       string
	ResultPath  string
	RawDir      string
	ArtifactKey string
}

func discoverBenchmarkManifestJobs(manifestFiles []string) ([]manifestBenchmarkJobRecord, error) {
	var jobs []manifestBenchmarkJobRecord
	for _, manifestPath := range manifestFiles {
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		var payload any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		collectBenchmarkManifestJobs(payload, &jobs)
	}
	return jobs, nil
}

func discoverManifestReportArtifacts(loaded researchconfig.Loaded, manifestFiles []string) ([]reportArtifactRecord, error) {
	records := make([]reportArtifactRecord, 0, len(manifestFiles))
	for _, manifestPath := range manifestFiles {
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		var payload any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode manifest %s: %w", filepath.ToSlash(manifestPath), err)
		}
		collectManifestReportArtifacts(loaded, payload, "", "", &records)
	}
	return records, nil
}

func collectManifestReportArtifacts(loaded researchconfig.Loaded, value any, inheritedPhase, inheritedRunID string, records *[]reportArtifactRecord) {
	switch typed := value.(type) {
	case map[string]any:
		phase := firstNonEmptyText(strings.TrimSpace(stringField(typed, "phase")), inheritedPhase)
		runID := firstNonEmptyText(strings.TrimSpace(stringField(typed, "run_id")), inheritedRunID)
		if reportPath := resolvePreferredExistingPath(loaded.WorkspaceRoot, stringField(typed, "report_path")); reportPath != "" && phase != "" && runID != "" {
			*records = append(*records, reportArtifactRecord{
				Kind:  "benchmark",
				Phase: phase,
				RunID: runID,
				Path:  reportPath,
			})
		}
		if pairReportPath := resolvePreferredExistingPath(loaded.WorkspaceRoot, stringField(typed, "pair_report_path")); pairReportPath != "" && phase != "" && runID != "" {
			*records = append(*records, reportArtifactRecord{
				Kind:  "pair",
				Phase: phase,
				RunID: runID,
				Path:  pairReportPath,
			})
		}
		for _, nested := range typed {
			collectManifestReportArtifacts(loaded, nested, phase, runID, records)
		}
	case []any:
		for _, item := range typed {
			collectManifestReportArtifacts(loaded, item, inheritedPhase, inheritedRunID, records)
		}
	}
}

func discoverBenchmarkSummaryRecords(summaryFiles []string) ([]benchmarkArtifactRecord, error) {
	var records []benchmarkArtifactRecord
	for _, summaryPath := range summaryFiles {
		raw, err := os.ReadFile(summaryPath)
		if err != nil {
			return nil, fmt.Errorf("read summary %s: %w", filepath.ToSlash(summaryPath), err)
		}
		reader := csv.NewReader(strings.NewReader(string(raw)))
		rows, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("decode summary %s: %w", filepath.ToSlash(summaryPath), err)
		}
		if len(rows) < 2 {
			continue
		}
		header := make(map[string]int, len(rows[0]))
		for idx, name := range rows[0] {
			header[strings.TrimSpace(name)] = idx
		}
		runIDColumn, ok := header["run_id"]
		if !ok {
			continue
		}
		summaryBase := strings.TrimSuffix(filepath.Base(summaryPath), filepath.Ext(summaryPath))
		for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
			row := rows[rowIdx]
			if runIDColumn >= len(row) {
				continue
			}
			workerRunID := strings.TrimSpace(row[runIDColumn])
			if workerRunID == "" {
				continue
			}
			outerRunID, jobKey := splitBenchmarkWorkerRunID(workerRunID)
			record := benchmarkArtifactRecord{
				OuterRunID:        outerRunID,
				WorkerRunID:       workerRunID,
				JobKey:            jobKey,
				Phase:             deriveBenchmarkSummaryPhase(summaryBase, outerRunID),
				ResultPath:        csvPathField(row, header, "result_file"),
				RawDir:            csvPathField(row, header, "raw_dir"),
				ChainSummaryPath:  csvPathField(row, header, "chain_result_file"),
				ImportCatalogPath: csvPathField(row, header, "import_catalog_file"),
			}
			record.ArtifactKey = extractBenchmarkArtifactKey(
				record.ResultPath,
				record.ChainSummaryPath,
				record.ImportCatalogPath,
				record.RawDir,
			)
			if record.ArtifactKey == "" && record.Phase != "" && record.OuterRunID != "" && record.JobKey != "" {
				record.ArtifactKey = artifactkey.BenchmarkJob(record.Phase, record.OuterRunID, record.JobKey)
			}
			records = append(records, record)
		}
	}
	return records, nil
}

func collectBenchmarkManifestJobs(value any, jobs *[]manifestBenchmarkJobRecord) {
	switch typed := value.(type) {
	case map[string]any:
		phase := strings.TrimSpace(stringField(typed, "phase"))
		outerRunID := strings.TrimSpace(stringField(typed, "run_id"))
		if rawJobs, ok := typed["jobs"].([]any); ok && phase != "" && outerRunID != "" {
			for _, item := range rawJobs {
				jobMap, ok := item.(map[string]any)
				if !ok {
					continue
				}
				jobKey := strings.TrimSpace(stringField(jobMap, "key"))
				workerRunID := strings.TrimSpace(stringField(jobMap, "run_id"))
				if jobKey == "" || workerRunID == "" {
					continue
				}
				*jobs = append(*jobs, manifestBenchmarkJobRecord{
					OuterRunID:  outerRunID,
					WorkerRunID: workerRunID,
					JobKey:      jobKey,
					Phase:       phase,
					ResultPath:  normalizeMaybePath(stringField(jobMap, "results_path")),
					RawDir:      normalizeMaybePath(stringField(jobMap, "raw_dir")),
					ArtifactKey: strings.TrimSpace(stringField(jobMap, "artifact_key")),
				})
			}
		}
		for _, nested := range typed {
			collectBenchmarkManifestJobs(nested, jobs)
		}
	case []any:
		for _, item := range typed {
			collectBenchmarkManifestJobs(item, jobs)
		}
	}
}

func mergeBenchmarkArtifactRecords(workspaceRoot string, summaryRecords []benchmarkArtifactRecord, stateJobs []state.Job, manifestJobs []manifestBenchmarkJobRecord) map[string]benchmarkArtifactRecord {
	records := make(map[string]benchmarkArtifactRecord)
	for _, record := range summaryRecords {
		if strings.TrimSpace(record.WorkerRunID) == "" {
			continue
		}
		record.ImportedCertificatesDirPath = existingBenchmarkPath(benchmarkLegacyImportedCertificatesDir(record.ResultPath, record.WorkerRunID))
		records[record.WorkerRunID] = record
	}
	for _, job := range stateJobs {
		workerRunID := benchmarkWorkerRunID(job.RunID, job.JobKey, "")
		if workerRunID == "" {
			continue
		}
		record := benchmarkArtifactRecord{
			OuterRunID:  strings.TrimSpace(job.RunID),
			WorkerRunID: workerRunID,
			JobKey:      strings.TrimSpace(job.JobKey),
			Phase:       strings.TrimSpace(job.Phase),
			ArtifactKey: artifactkey.BenchmarkJob(job.Phase, job.RunID, job.JobKey),
			ResultPath:  normalizeMaybePath(job.ResultPath),
		}
		record.RawDir = benchmarkActualRawDir(job.RawDir, workerRunID, record.ArtifactKey)
		if record.ChainSummaryPath == "" {
			record.ChainSummaryPath = existingBenchmarkPath(benchmarkLegacyChainPath(record.ResultPath, workerRunID))
		}
		if record.ImportCatalogPath == "" {
			record.ImportCatalogPath = existingBenchmarkPath(benchmarkLegacyImportCatalogPath(record.ResultPath, workerRunID))
		}
		if record.ImportedCertificatesDirPath == "" {
			record.ImportedCertificatesDirPath = existingBenchmarkPath(benchmarkLegacyImportedCertificatesDir(record.ResultPath, workerRunID))
		}
		records[record.WorkerRunID] = record
	}
	for _, job := range manifestJobs {
		record := records[job.WorkerRunID]
		if record.WorkerRunID == "" {
			record = benchmarkArtifactRecord{
				OuterRunID:  strings.TrimSpace(job.OuterRunID),
				WorkerRunID: strings.TrimSpace(job.WorkerRunID),
				JobKey:      strings.TrimSpace(job.JobKey),
				Phase:       strings.TrimSpace(job.Phase),
				ArtifactKey: strings.TrimSpace(job.ArtifactKey),
			}
			if record.ArtifactKey == "" {
				record.ArtifactKey = artifactkey.BenchmarkJob(record.Phase, record.OuterRunID, record.JobKey)
			}
		}
		if record.ResultPath == "" {
			record.ResultPath = normalizeMaybePath(job.ResultPath)
		}
		if record.RawDir == "" {
			record.RawDir = benchmarkActualRawDir(job.RawDir, job.WorkerRunID, record.ArtifactKey)
		}
		if record.ChainSummaryPath == "" {
			record.ChainSummaryPath = existingBenchmarkPath(benchmarkLegacyChainPath(record.ResultPath, job.WorkerRunID))
		}
		if record.ImportCatalogPath == "" {
			record.ImportCatalogPath = existingBenchmarkPath(benchmarkLegacyImportCatalogPath(record.ResultPath, job.WorkerRunID))
		}
		if record.ImportedCertificatesDirPath == "" {
			record.ImportedCertificatesDirPath = existingBenchmarkPath(benchmarkLegacyImportedCertificatesDir(record.ResultPath, job.WorkerRunID))
		}
		records[record.WorkerRunID] = record
	}
	return resolveBenchmarkArtifactRecords(workspaceRoot, records)
}

func resolveBenchmarkArtifactRecords(workspaceRoot string, records map[string]benchmarkArtifactRecord) map[string]benchmarkArtifactRecord {
	for workerRunID, record := range records {
		if record.ArtifactKey == "" && record.Phase != "" && record.OuterRunID != "" && record.JobKey != "" {
			record.ArtifactKey = artifactkey.BenchmarkJob(record.Phase, record.OuterRunID, record.JobKey)
		}
		record.ResultPath = resolveBenchmarkResultPath(workspaceRoot, record.ResultPath, record.WorkerRunID)
		record.RawDir = resolveBenchmarkRawDir(workspaceRoot, record.RawDir, record.WorkerRunID, record.ArtifactKey)
		record.ChainSummaryPath = resolvePreferredExistingPath(
			workspaceRoot,
			record.ChainSummaryPath,
			benchmarkLegacyChainPath(record.ResultPath, record.WorkerRunID),
		)
		record.ImportCatalogPath = resolvePreferredExistingPath(
			workspaceRoot,
			record.ImportCatalogPath,
			benchmarkLegacyImportCatalogPath(record.ResultPath, record.WorkerRunID),
		)
		record.ImportedCertificatesDirPath = resolvePreferredExistingPath(
			workspaceRoot,
			record.ImportedCertificatesDirPath,
			benchmarkLegacyImportedCertificatesDir(record.ResultPath, record.WorkerRunID),
		)
		records[workerRunID] = record
	}
	return records
}

func compactReportTargetPath(loaded researchconfig.Loaded, record reportArtifactRecord) string {
	sourcePath := normalizeMaybePath(record.Path)
	if sourcePath == "" {
		return ""
	}
	reportDir := filepath.ToSlash(filepath.Dir(sourcePath))
	var basename string
	switch strings.TrimSpace(record.Kind) {
	case "pair":
		basename = planner.PairReportFilename(loaded, record.Phase, reportDir, record.RunID)
	default:
		basename = planner.BenchmarkReportFilename(loaded, record.Phase, reportDir, record.RunID)
	}
	if strings.TrimSpace(basename) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(reportDir, basename))
}

func benchmarkWorkerRunID(outerRunID, jobKey, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	if strings.TrimSpace(outerRunID) == "" || strings.TrimSpace(jobKey) == "" {
		return ""
	}
	return strings.TrimSpace(outerRunID) + "_" + strings.TrimSpace(jobKey)
}

func splitBenchmarkWorkerRunID(workerRunID string) (string, string) {
	workerRunID = strings.TrimSpace(workerRunID)
	lastUnderscore := strings.LastIndex(workerRunID, "_")
	if lastUnderscore <= 0 || lastUnderscore >= len(workerRunID)-1 {
		return "", workerRunID
	}
	return workerRunID[:lastUnderscore], workerRunID[lastUnderscore+1:]
}

func deriveBenchmarkSummaryPhase(summaryBase, outerRunID string) string {
	summaryBase = strings.TrimSpace(summaryBase)
	outerRunID = strings.TrimSpace(outerRunID)
	if summaryBase == "" {
		return ""
	}
	if outerRunID != "" {
		suffix := "_" + outerRunID
		if strings.HasSuffix(summaryBase, suffix) {
			return strings.TrimSuffix(summaryBase, suffix)
		}
	}
	if idx := strings.Index(outerRunID, "_"); idx > 0 {
		return outerRunID[:idx]
	}
	return summaryBase
}

func csvPathField(row []string, header map[string]int, column string) string {
	idx, ok := header[column]
	if !ok || idx >= len(row) {
		return ""
	}
	return normalizeMaybePath(row[idx])
}

func extractBenchmarkArtifactKey(paths ...string) string {
	for _, path := range paths {
		if key := extractArtifactKeyFromPath(path); key != "" {
			return key
		}
	}
	return ""
}

func extractArtifactKeyFromPath(path string) string {
	path = normalizeMaybePath(path)
	if path == "" {
		return ""
	}
	base := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	for _, prefix := range []string{"result_", "chain_result_", "imported_certificate_catalog_"} {
		if strings.HasPrefix(base, prefix) {
			base = strings.TrimPrefix(base, prefix)
			break
		}
	}
	if strings.HasPrefix(base, "bj_") {
		return base
	}
	return ""
}

func benchmarkActualRawDir(rawDir, workerRunID, artifactKey string) string {
	rawDir = normalizeMaybePath(rawDir)
	if rawDir == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(rawDir))
	switch base {
	case strings.TrimSpace(workerRunID), strings.TrimSpace(artifactKey):
		return rawDir
	default:
		return filepath.ToSlash(filepath.Join(rawDir, strings.TrimSpace(workerRunID)))
	}
}

func benchmarkLegacyResultPath(resultsPath, workerRunID string) string {
	resultsPath = normalizeMaybePath(resultsPath)
	if resultsPath == "" || strings.TrimSpace(workerRunID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(resultsPath), fmt.Sprintf("result_%s%s", strings.TrimSpace(workerRunID), filepath.Ext(resultsPath))))
}

func benchmarkLegacyRawDir(rawDir, workerRunID, artifactKey string) string {
	rawDir = normalizeMaybePath(rawDir)
	if rawDir == "" || strings.TrimSpace(workerRunID) == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(rawDir))
	if strings.TrimSpace(artifactKey) != "" && base == strings.TrimSpace(artifactKey) {
		return filepath.ToSlash(filepath.Join(filepath.Dir(rawDir), strings.TrimSpace(workerRunID)))
	}
	if base == strings.TrimSpace(workerRunID) {
		return rawDir
	}
	return benchmarkActualRawDir(rawDir, workerRunID, artifactKey)
}

func resolvePreferredExistingPath(workspaceRoot, primary string, fallbacks ...string) string {
	primary = normalizeMaybePath(primary)
	candidates := make([]string, 0, 2+len(fallbacks)*2)
	for _, candidate := range append([]string{primary}, fallbacks...) {
		candidate = normalizeMaybePath(candidate)
		if candidate == "" {
			continue
		}
		if rebased := rebaseToWorkspacePath(workspaceRoot, candidate); rebased != "" && rebased != candidate {
			candidates = append(candidates, rebased)
		}
		candidates = append(candidates, candidate)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if rebased := rebaseToWorkspacePath(workspaceRoot, primary); rebased != "" {
		return rebased
	}
	return primary
}

func resolveBenchmarkResultPath(workspaceRoot, path, workerRunID string) string {
	return resolvePreferredExistingPath(workspaceRoot, path, benchmarkLegacyResultPath(path, workerRunID))
}

func resolveBenchmarkRawDir(workspaceRoot, path, workerRunID, artifactKey string) string {
	return resolvePreferredExistingPath(workspaceRoot, path, benchmarkLegacyRawDir(path, workerRunID, artifactKey))
}

func rebaseToWorkspacePath(workspaceRoot, path string) string {
	path = normalizeMaybePath(path)
	workspaceRoot = normalizeMaybePath(workspaceRoot)
	if path == "" || workspaceRoot == "" {
		return ""
	}
	isAbsoluteLike := filepath.IsAbs(filepath.FromSlash(path)) || strings.HasPrefix(path, "/")
	if !isAbsoluteLike {
		return filepath.ToSlash(filepath.Join(workspaceRoot, filepath.FromSlash(path)))
	}
	if strings.HasPrefix(path, workspaceRoot+"/") || path == workspaceRoot {
		return path
	}
	for _, marker := range []string{"research/", "platform-tooling/", "docs/"} {
		if idx := strings.Index(path, marker); idx >= 0 {
			return filepath.ToSlash(filepath.Join(workspaceRoot, filepath.FromSlash(path[idx:])))
		}
	}
	return path
}

func benchmarkResultTargetPath(path, artifactID string) string {
	path = normalizeMaybePath(path)
	if path == "" || strings.TrimSpace(artifactID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), fmt.Sprintf("result_%s%s", strings.TrimSpace(artifactID), filepath.Ext(path))))
}

func benchmarkRawTargetPath(path, artifactID string) string {
	path = normalizeMaybePath(path)
	if path == "" || strings.TrimSpace(artifactID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), strings.TrimSpace(artifactID)))
}

func benchmarkChainTargetPath(path, artifactID string) string {
	path = normalizeMaybePath(path)
	if path == "" || strings.TrimSpace(artifactID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), fmt.Sprintf("chain_result_%s%s", strings.TrimSpace(artifactID), filepath.Ext(path))))
}

func benchmarkImportCatalogTargetPath(path, artifactID string) string {
	path = normalizeMaybePath(path)
	if path == "" || strings.TrimSpace(artifactID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), fmt.Sprintf("imported_certificate_catalog_%s%s", strings.TrimSpace(artifactID), filepath.Ext(path))))
}

func benchmarkImportedCertificatesTargetPath(path, artifactID string) string {
	path = normalizeMaybePath(path)
	if path == "" || strings.TrimSpace(artifactID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), strings.TrimSpace(artifactID)))
}

func benchmarkLegacyChainPath(resultsPath, workerRunID string) string {
	resultsPath = normalizeMaybePath(resultsPath)
	if resultsPath == "" || strings.TrimSpace(workerRunID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(resultsPath), fmt.Sprintf("chain_result_%s.csv", strings.TrimSpace(workerRunID))))
}

func benchmarkLegacyImportCatalogPath(resultsPath, workerRunID string) string {
	resultsPath = normalizeMaybePath(resultsPath)
	if resultsPath == "" || strings.TrimSpace(workerRunID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(resultsPath), fmt.Sprintf("imported_certificate_catalog_%s.csv", strings.TrimSpace(workerRunID))))
}

func benchmarkLegacyImportedCertificatesDir(resultsPath, workerRunID string) string {
	resultsPath = normalizeMaybePath(resultsPath)
	if resultsPath == "" || strings.TrimSpace(workerRunID) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(resultsPath), "imported_certificates", strings.TrimSpace(workerRunID)))
}

func existingBenchmarkPath(path string) string {
	path = normalizeMaybePath(path)
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func addArtifactMapping(mappings map[string]string, source, target string) {
	source = normalizeMaybePath(source)
	target = normalizeMaybePath(target)
	if source == "" || target == "" || source == target {
		return
	}
	if _, err := os.Stat(source); err != nil {
		return
	}
	if existing, ok := mappings[source]; ok && existing != target {
		return
	}
	mappings[source] = target
}

func buildArtifactAliasMapping(workspaceRoot string, mappings map[string]string) map[string]string {
	aliases := make(map[string]string, len(mappings)*2)
	for source, target := range mappings {
		source = normalizeMaybePath(source)
		target = normalizeMaybePath(target)
		if source == "" || target == "" {
			continue
		}
		aliases[source] = target
		if relSource, err := filepath.Rel(workspaceRoot, filepath.FromSlash(source)); err == nil {
			if relTarget, err := filepath.Rel(workspaceRoot, filepath.FromSlash(target)); err == nil {
				aliases[filepath.ToSlash(relSource)] = target
				aliases[filepath.ToSlash(relTarget)] = target
			}
		}
	}
	return aliases
}

func collectBenchmarkResultFiles(records map[string]benchmarkArtifactRecord, mappings map[string]string, ordered []artifactMappingRecord) []string {
	seen := make(map[string]struct{}, len(records))
	files := make([]string, 0, len(records))
	for _, record := range records {
		path := remapPathValue(record.ResultPath, mappings, ordered)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			source := normalizeMaybePath(record.ResultPath)
			if source == "" {
				continue
			}
			if _, sourceErr := os.Stat(source); sourceErr != nil {
				continue
			}
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}
	slices.Sort(files)
	return files
}

func collectBenchmarkImportCatalogFiles(records map[string]benchmarkArtifactRecord, mappings map[string]string, ordered []artifactMappingRecord) []string {
	seen := make(map[string]struct{}, len(records))
	files := make([]string, 0, len(records))
	for _, record := range records {
		path := remapPathValue(record.ImportCatalogPath, mappings, ordered)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			source := normalizeMaybePath(record.ImportCatalogPath)
			if source == "" {
				continue
			}
			if _, sourceErr := os.Stat(source); sourceErr != nil {
				continue
			}
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}
	slices.Sort(files)
	return files
}

func orderedArtifactMappings(mappings map[string]string) []artifactMappingRecord {
	ordered := make([]artifactMappingRecord, 0, len(mappings))
	for source, target := range mappings {
		ordered = append(ordered, artifactMappingRecord{Source: source, Target: target})
	}
	slices.SortFunc(ordered, func(a, b artifactMappingRecord) int {
		switch {
		case len(a.Source) > len(b.Source):
			return -1
		case len(a.Source) < len(b.Source):
			return 1
		default:
			return strings.Compare(a.Source, b.Source)
		}
	})
	return ordered
}

func sampleArtifactMappings(ordered []artifactMappingRecord, limit int) []artifactMappingRecord {
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}

func rewriteCSVPathColumns(path string, columns []string, mappings map[string]string, ordered []artifactMappingRecord) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read csv %s: %w", filepath.ToSlash(path), err)
	}
	reader := csv.NewReader(strings.NewReader(string(raw)))
	records, err := reader.ReadAll()
	if err != nil {
		return false, fmt.Errorf("decode csv %s: %w", filepath.ToSlash(path), err)
	}
	if len(records) == 0 {
		return false, nil
	}
	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	changed := false
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		for _, column := range columns {
			colIdx, ok := header[column]
			if !ok || colIdx >= len(records[rowIdx]) {
				continue
			}
			rewritten := remapPathValue(records[rowIdx][colIdx], mappings, ordered)
			if rewritten != "" && rewritten != records[rowIdx][colIdx] {
				records[rowIdx][colIdx] = rewritten
				changed = true
			}
		}
	}
	if !changed {
		return false, nil
	}
	var b strings.Builder
	writer := csv.NewWriter(&b)
	if err := writer.WriteAll(records); err != nil {
		return false, fmt.Errorf("encode csv %s: %w", filepath.ToSlash(path), err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return false, fmt.Errorf("write csv %s: %w", filepath.ToSlash(path), err)
	}
	return true, nil
}

func rewriteManifestJSON(path string, benchmarkJobs map[string]benchmarkArtifactRecord, mappings map[string]string, ordered []artifactMappingRecord) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read manifest %s: %w", filepath.ToSlash(path), err)
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, fmt.Errorf("decode manifest %s: %w", filepath.ToSlash(path), err)
	}
	changed := rewriteManifestValue(&payload, benchmarkJobs, mappings, ordered)
	if !changed {
		return false, nil
	}
	rewritten, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode manifest %s: %w", filepath.ToSlash(path), err)
	}
	if err := os.WriteFile(path, rewritten, 0o644); err != nil {
		return false, fmt.Errorf("write manifest %s: %w", filepath.ToSlash(path), err)
	}
	return true, nil
}

func rewriteManifestValue(value *any, benchmarkJobs map[string]benchmarkArtifactRecord, mappings map[string]string, ordered []artifactMappingRecord) bool {
	switch typed := (*value).(type) {
	case map[string]any:
		changed := false
		phase := strings.TrimSpace(stringField(typed, "phase"))
		if rawJobs, ok := typed["jobs"].([]any); ok && phase != "" {
			for idx, item := range rawJobs {
				jobMap, ok := item.(map[string]any)
				if !ok {
					continue
				}
				workerRunID := strings.TrimSpace(stringField(jobMap, "run_id"))
				record, ok := benchmarkJobs[workerRunID]
				if !ok {
					continue
				}
				jobChanged := false
				targetResult := benchmarkResultTargetPath(record.ResultPath, record.ArtifactKey)
				if targetResult != "" && normalizeMaybePath(stringField(jobMap, "results_path")) != targetResult {
					jobMap["results_path"] = targetResult
					jobChanged = true
				}
				targetRawDir := benchmarkRawTargetPath(record.RawDir, record.ArtifactKey)
				if targetRawDir != "" && normalizeMaybePath(stringField(jobMap, "raw_dir")) != targetRawDir {
					jobMap["raw_dir"] = targetRawDir
					jobChanged = true
				}
				if strings.TrimSpace(stringField(jobMap, "artifact_key")) != record.ArtifactKey {
					jobMap["artifact_key"] = record.ArtifactKey
					jobChanged = true
				}
				if jobChanged {
					rawJobs[idx] = jobMap
					changed = true
				}
			}
			typed["jobs"] = rawJobs
		}
		for key, nested := range typed {
			nestedValue := nested
			if rewriteManifestValue(&nestedValue, benchmarkJobs, mappings, ordered) {
				typed[key] = nestedValue
				changed = true
			}
		}
		*value = typed
		return changed
	case []any:
		changed := false
		for idx, item := range typed {
			nested := item
			if rewriteManifestValue(&nested, benchmarkJobs, mappings, ordered) {
				typed[idx] = nested
				changed = true
			}
		}
		*value = typed
		return changed
	case string:
		rewritten := remapPathValue(typed, mappings, ordered)
		if rewritten != typed {
			*value = rewritten
			return true
		}
	}
	return false
}

func rewriteBenchmarkStateRows(ctx context.Context, store *state.Store, plan *artifactCompactionPlan) error {
	runs, err := store.ListRuns(ctx)
	if err != nil {
		return err
	}
	for _, run := range runs {
		rewrittenMetadata := rewriteMappedTextLocal(run.MetadataJSON, plan.OrderedRewriteMap)
		if rewrittenMetadata == run.MetadataJSON {
			continue
		}
		run.MetadataJSON = rewrittenMetadata
		if err := store.UpsertRun(ctx, run); err != nil {
			return err
		}
	}
	jobs, err := store.ListJobs(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		workerRunID := benchmarkWorkerRunID(job.RunID, job.JobKey, "")
		record, ok := plan.BenchmarkJobs[workerRunID]
		if !ok {
			continue
		}
		job.ResultPath = remapPathValue(job.ResultPath, plan.ExactMappings, plan.OrderedMappings)
		job.RawDir = benchmarkRawTargetPath(record.RawDir, record.ArtifactKey)
		job.MetadataJSON = rewriteMappedTextLocal(job.MetadataJSON, plan.OrderedRewriteMap)
		if err := store.UpsertJob(ctx, job); err != nil {
			return err
		}
	}
	artifacts, err := store.ListArtifacts(ctx)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		newArtifact := artifact
		if artifact.Role == "benchmark-raw-dir" {
			workerRunID := benchmarkWorkerRunID(artifact.RunID, artifact.JobKey, "")
			if record, ok := plan.BenchmarkJobs[workerRunID]; ok {
				newArtifact.AbsolutePath = benchmarkRawTargetPath(record.RawDir, record.ArtifactKey)
			}
		} else {
			newArtifact.AbsolutePath = remapPathValue(artifact.AbsolutePath, plan.ExactMappings, plan.OrderedMappings)
		}
		newArtifact.RelativePath = ""
		newArtifact.MetadataJSON = rewriteMappedTextLocal(artifact.MetadataJSON, plan.OrderedRewriteMap)
		if normalizeMaybePath(newArtifact.AbsolutePath) == normalizeMaybePath(artifact.AbsolutePath) && newArtifact.MetadataJSON == artifact.MetadataJSON {
			continue
		}
		if err := store.DeleteArtifact(ctx, artifact.RunID, artifact.Role, artifact.AbsolutePath); err != nil {
			return err
		}
		if err := store.RecordArtifact(ctx, newArtifact); err != nil {
			return err
		}
	}
	return nil
}

func writeArtifactMigrationManifest(path, workspaceRoot string, mappings map[string]string) error {
	records := orderedArtifactMappings(mappings)
	payload := artifactMigrationManifest{
		SchemaVersion: "researchctl.artifact-compaction/v1",
		CreatedAtUTC:  time.Now().UTC().Format(time.RFC3339),
		WorkspaceRoot: filepath.ToSlash(workspaceRoot),
		Apply:         true,
		RenamedPaths:  records,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal artifact compaction manifest: %w", err)
	}
	return writeManifestFile(path, raw)
}

func remapPathValue(value string, mappings map[string]string, ordered []artifactMappingRecord) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	normalized := normalizeMaybePath(value)
	if target, ok := mappings[normalized]; ok {
		return filepath.ToSlash(target)
	}
	if len(ordered) == 0 {
		ordered = orderedArtifactMappings(mappings)
	}
	for _, mapping := range ordered {
		source := normalizeMaybePath(mapping.Source)
		target := normalizeMaybePath(mapping.Target)
		if strings.HasPrefix(normalized, source+"/") {
			suffix := strings.TrimPrefix(normalized, source+"/")
			return filepath.ToSlash(filepath.Join(target, filepath.FromSlash(suffix)))
		}
		if !filepath.IsAbs(filepath.FromSlash(source)) {
			if normalized == source || strings.HasSuffix(normalized, "/"+source) {
				return target
			}
			marker := "/" + source + "/"
			if idx := strings.Index(normalized, marker); idx >= 0 {
				suffix := normalized[idx+len(marker):]
				return filepath.ToSlash(filepath.Join(target, filepath.FromSlash(suffix)))
			}
		}
	}
	return normalizeMaybePath(value)
}

func rewriteMappedTextLocal(value string, ordered []artifactMappingRecord) string {
	rewritten := value
	for _, mapping := range ordered {
		rewritten = strings.ReplaceAll(rewritten, mapping.Source, mapping.Target)
		rewritten = strings.ReplaceAll(rewritten, strings.ReplaceAll(mapping.Source, "/", `\`), strings.ReplaceAll(mapping.Target, "/", `\`))
	}
	return rewritten
}

func normalizeMaybePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
}

func pathWithinWorkspace(workspaceRoot, path string) bool {
	workspaceRoot = normalizeMaybePath(workspaceRoot)
	path = normalizeMaybePath(path)
	if workspaceRoot == "" || path == "" {
		return false
	}
	if path == workspaceRoot {
		return true
	}
	return strings.HasPrefix(path, workspaceRoot+"/")
}

func stringField(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	text, _ := raw.(string)
	return text
}
