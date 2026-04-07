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

	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

func runHilbertBenchmarkCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := firstNonEmptyEnv("HILBERT_BENCHMARK_THEOREMS_FILE", "HILBERT_BENCHMARK_CASES_FILE")
	if defaultCasesFile == "" {
		defaultCasesFile = defaultBenchmarkInputFile(defaultProjectFolder)
	}

	defaultProvider := firstNonEmptyEnv("HILBERT_BENCHMARK_LLM_PROVIDER", "LLM_PROVIDER")
	if defaultProvider == "" {
		defaultProvider = "compatible"
	}

	runID := defaultRunID()
	fs := flag.NewFlagSet("run-hilbert-benchmark", flag.ContinueOnError)
	fs.SetOutput(stderr)

	provider := fs.String("provider", defaultProvider, "LLM provider: compatible, openai, google, or mistral")
	providerLabel := fs.String("provider-label", firstNonEmptyEnv("HILBERT_BENCHMARK_PROVIDER_LABEL"), "Display-only provider label for outputs and paths")
	googleThinkingMode := fs.String("google-thinking-mode", firstNonEmptyEnv("HILBERT_BENCHMARK_GOOGLE_THINKING_MODE"), "Google thinking mode: default|minimal|low|high|budget0")
	googleRequestsPerMinute := fs.Int("google-requests-per-minute", parseIntEnv(firstNonEmptyEnv("HILBERT_BENCHMARK_GOOGLE_REQUESTS_PER_MINUTE")), "Optional Google requests-per-minute throttle (0 disables pacing)")
	mistralRequestsPerSecond := fs.Int("mistral-requests-per-second", parseIntEnv(firstNonEmptyEnv("HILBERT_BENCHMARK_MISTRAL_REQUESTS_PER_SECOND")), "Optional Mistral requests-per-second throttle (0 disables pacing)")
	baseURL := fs.String("base-url", firstNonEmptyEnv("HILBERT_BENCHMARK_BASE_URL", "OPENAI_BASE_URL", "LLM_BASE_URL"), "LLM base URL")
	apiKey := fs.String("api-key", firstNonEmptyEnv("HILBERT_BENCHMARK_API_KEY", "OPENAI_API_KEY", "LLM_API_KEY"), "LLM API key")
	model := fs.String("model", firstNonEmptyEnv("HILBERT_BENCHMARK_MODEL", "OPENAI_MODEL", "LLM_MODEL"), "LLM model identifier")
	canonicalModel := fs.String("canonical-model", firstNonEmptyEnv("HILBERT_BENCHMARK_CANONICAL_MODEL"), "Canonical llm_model identifier to preserve in benchmark outputs")
	timeout := fs.Duration("timeout", 60*time.Second, "LLM request timeout")
	requestTimeoutAbortThreshold := fs.Int("request-timeout-abort-threshold", parseIntEnv(firstNonEmptyEnv("HILBERT_BENCHMARK_REQUEST_TIMEOUT_ABORT_THRESHOLD")), "After N consecutive request timeouts, stop new LLM requests and mark remaining cases as request_failure (0 disables)")
	experimentName := fs.String("experiment-name", firstNonEmptyEnv("HILBERT_BENCHMARK_EXPERIMENT_NAME"), "Human-readable experimental setup label used to derive experiment_id")
	experimentID := fs.String("experiment-id", firstNonEmptyEnv("HILBERT_BENCHMARK_EXPERIMENT_ID"), "Explicit experiment identifier override")
	temperature := fs.String("temperature", firstNonEmptyEnv("HILBERT_BENCHMARK_TEMPERATURE"), "Requested sampling temperature")
	seed := fs.String("seed", firstNonEmptyEnv("HILBERT_BENCHMARK_SEED"), "Requested sampling seed")
	topP := fs.String("top-p", firstNonEmptyEnv("HILBERT_BENCHMARK_TOP_P"), "Requested sampling top_p")
	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input CSV relative to the benchmark project folder")
	casesPath := fs.String("cases", firstNonEmptyEnv("HILBERT_BENCHMARK_THEOREMS", "HILBERT_BENCHMARK_CASES"), "Path to benchmark input CSV")
	resultsPath := fs.String("results", firstNonEmptyEnv("HILBERT_BENCHMARK_RESULTS"), "Path to benchmark run result csv")
	summaryPath := fs.String("summary", firstNonEmptyEnv("HILBERT_BENCHMARK_SUMMARY"), "Path to benchmark run summary csv")
	rawBaseDir := fs.String("raw-dir", firstNonEmptyEnv("HILBERT_BENCHMARK_RAW_DIR"), "Directory for raw model outputs")
	artifactKey := fs.String("artifact-key", firstNonEmptyEnv("HILBERT_BENCHMARK_ARTIFACT_KEY"), "Short filesystem artifact key for result/raw/sidecar paths")
	caseID := fs.String("case-id", "", "Run only one case_id")
	limit := fs.Int("limit", 0, "Maximum number of cases to run")
	runIDFlag := fs.String("run-id", runID, "Stable run identifier")
	promptVersion := fs.String("prompt-version", firstNonEmptyEnv("HILBERT_BENCHMARK_PROMPT_VERSION"), "Benchmark prompt profile version")
	hilbertcheckPath := fs.String("hilbertcheck-path", firstNonEmptyEnv("HILBERT_BENCHMARK_HILBERTCHECK"), "Optional path to a prebuilt hilbertcheck binary")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if *requestTimeoutAbortThreshold < 0 {
		return fmt.Errorf("request-timeout-abort-threshold must be >= 0")
	}
	requestedTemperature, err := parseOptionalBenchmarkFloat64(*temperature, "temperature")
	if err != nil {
		return err
	}
	requestedSeed, err := parseOptionalBenchmarkInt64(*seed, "seed")
	if err != nil {
		return err
	}
	requestedTopP, err := parseOptionalBenchmarkFloat64(*topP, "top-p")
	if err != nil {
		return err
	}
	if strings.TrimSpace(*promptVersion) == "" {
		*promptVersion = benchmarkPromptVersionDefault
	}
	promptProfile, err := resolveBenchmarkPromptProfile(*promptVersion)
	if err != nil {
		return err
	}
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*projectFolder))
	if strings.TrimSpace(*casesPath) != "" {
		*casesPath = resolveBenchmarkArgPath(workspaceRoot, *casesPath)
	}
	if strings.TrimSpace(*casesPath) == "" {
		*casesPath = filepath.Join(benchmarkRoot, strings.TrimSpace(*casesFile))
	}
	resolvedProjectFolder := inferBenchmarkProjectFolder(*casesPath, workspaceRoot, strings.TrimSpace(*projectFolder))
	resolvedCasesFile := inferBenchmarkCasesFile(*casesPath, workspaceRoot, resolvedProjectFolder, strings.TrimSpace(*casesFile))
	resolvedCaseSelector := deriveBenchmarkCaseSelector(strings.TrimSpace(*caseID), *limit)
	resolvedBenchmarkRoot := benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)
	if strings.TrimSpace(*resultsPath) == "" {
		*resultsPath = defaultBenchmarkResultsPathForArtifact(resolvedBenchmarkRoot, *runIDFlag, strings.TrimSpace(*artifactKey))
	} else {
		*resultsPath = resolveBenchmarkArgPath(workspaceRoot, *resultsPath)
	}
	if strings.TrimSpace(*summaryPath) == "" {
		*summaryPath = defaultBenchmarkSummaryPath(resolvedBenchmarkRoot)
	} else {
		*summaryPath = resolveBenchmarkArgPath(workspaceRoot, *summaryPath)
	}
	if strings.TrimSpace(*rawBaseDir) == "" {
		*rawBaseDir = defaultBenchmarkRawDir(resolvedBenchmarkRoot)
	} else {
		*rawBaseDir = resolveBenchmarkArgPath(workspaceRoot, *rawBaseDir)
	}

	resolvedGoogleThinkingMode := ""
	if normalizeBenchmarkProvider(*provider) == "google" {
		resolvedGoogleThinkingMode, err = normalizeGoogleThinkingMode(*googleThinkingMode)
		if err != nil {
			return err
		}
	}

	requestedSampling := researchsampling.NewParams(requestedTemperature, requestedSeed, requestedTopP)
	samplingResolution := researchsampling.Resolve(normalizeBenchmarkProvider(*provider), requestedSampling)
	resolvedExperimentName := researchsampling.ResolveExperimentName(strings.TrimSpace(*experimentName), samplingResolution.RequestedProfile, samplingResolution.EffectiveProfile)
	resolvedExperimentID := strings.TrimSpace(*experimentID)
	if resolvedExperimentID == "" {
		resolvedExperimentID = researchsampling.BuildExperimentID(resolvedExperimentName, researchsampling.SequenceFromRunID(*runIDFlag), time.Now(), samplingResolution.RequestedProfile)
	}

	modelClient, modelLabel, runtimeLLMModel, err := newBenchmarkModel(*provider, strings.TrimSpace(*baseURL), strings.TrimSpace(*apiKey), strings.TrimSpace(*model), resolvedGoogleThinkingMode, *googleRequestsPerMinute, *mistralRequestsPerSecond, *timeout, samplingResolution.Effective)
	if err != nil {
		return err
	}
	resolvedCanonicalModel := resolveCanonicalBenchmarkModel(*canonicalModel, runtimeLLMModel)
	displayModelLabel := benchmarkDisplayModelLabel(modelLabel, *providerLabel, resolvedCanonicalModel)

	tempDir, err := os.MkdirTemp("", "hilbert-benchmark-*")
	if err != nil {
		return fmt.Errorf("create benchmark temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	verifier, err := resolveHilbertVerifier(context.Background(), workspaceRoot, tempDir, strings.TrimSpace(*hilbertcheckPath))
	if err != nil {
		return err
	}

	summary, err := runHilbertBenchmark(context.Background(), benchmarkRunOptions{
		ProjectFolder:                resolvedProjectFolder,
		CasesFile:                    resolvedCasesFile,
		CaseSelector:                 resolvedCaseSelector,
		CasesPath:                    *casesPath,
		ResultsPath:                  *resultsPath,
		SummaryPath:                  *summaryPath,
		RawBaseDir:                   *rawBaseDir,
		RunID:                        *runIDFlag,
		ArtifactKey:                  strings.TrimSpace(*artifactKey),
		Provider:                     normalizeBenchmarkProvider(*provider),
		ProviderLabel:                strings.TrimSpace(*providerLabel),
		LLMModel:                     resolvedCanonicalModel,
		ExperimentName:               resolvedExperimentName,
		ExperimentID:                 resolvedExperimentID,
		RequestedTemperature:         requestedTemperature,
		RequestedSeed:                requestedSeed,
		RequestedTopP:                requestedTopP,
		GoogleThinkingMode:           resolvedGoogleThinkingMode,
		PromptVersion:                promptProfile.Version,
		Surface:                      "direct",
		ModelLabel:                   displayModelLabel,
		CaseID:                       strings.TrimSpace(*caseID),
		Limit:                        *limit,
		RequestTimeoutAbortThreshold: *requestTimeoutAbortThreshold,
	}, promptProfile, modelClient, verifier, time.Now)
	if err != nil {
		return err
	}
	if err := appendBenchmarkRunSummary(summary.SummaryPath, benchmarkRunSummaryRow{
		RunID:                    summary.RunID,
		TimestampUTC:             time.Now().UTC().Format(time.RFC3339),
		Provider:                 summary.Provider,
		Model:                    displayModelLabel,
		LLMModel:                 summary.LLMModel,
		ExperimentName:           summary.ExperimentName,
		ExperimentID:             summary.ExperimentID,
		SamplingSurface:          summary.SamplingSurface,
		RequestedSamplingProfile: summary.RequestedSamplingProfile,
		EffectiveSamplingProfile: summary.EffectiveSamplingProfile,
		RequestedSamplingJSON:    summary.RequestedSamplingJSON,
		EffectiveSamplingJSON:    summary.EffectiveSamplingJSON,
		UnsupportedSamplingJSON:  summary.UnsupportedSamplingJSON,
		GoogleThinkingMode:       summary.GoogleThinkingMode,
		PromptVersion:            summary.PromptVersion,
		CertificateVersion:       benchmarkCertificateVersion,
		ProjectFolder:            summary.ProjectFolder,
		Surface:                  summary.Surface,
		TheoremPackID:            summary.TheoremPackID,
		CasesFile:                summary.CasesFile,
		CaseSelector:             summary.CaseSelector,
		Hypothesis:               benchmarkHypothesis,
		CasesPath:                *casesPath,
		ResultsPath:              summary.ResultsPath,
		ChainSummaryPath:         summary.ChainSummaryPath,
		ImportCatalogPath:        summary.ImportCatalogPath,
		RawDir:                   summary.RawRunDir,
		CasesTotal:               strconv.Itoa(summary.CaseCount),
		PassCount:                strconv.Itoa(summary.Buckets["pass"]),
		FalseRefusalCount:        strconv.Itoa(summary.Buckets["false_refusal"]),
		FalseAcceptCount:         strconv.Itoa(summary.Buckets["false_accept"]),
		RequestFailureCount:      strconv.Itoa(summary.Buckets["request_failure"]),
		SchemaFailureCount:       strconv.Itoa(summary.Buckets["schema_failure"]),
		ParseFailureCount:        strconv.Itoa(summary.Buckets["parse_failure"]),
		KernelFailureCount:       strconv.Itoa(summary.Buckets["kernel_failure"]),
		ContractFailureCount:     strconv.Itoa(summary.Buckets["contract_failure"]),
		FormatFailureCount:       strconv.Itoa(summary.Buckets["format_failure"]),
		AvgLatencyMS:             formatBenchmarkFloat(summary.AvgLatencyMS),
		MaxLatencyMS:             strconv.FormatInt(summary.MaxLatencyMS, 10),
		RunElapsedSeconds:        strconv.FormatInt(summary.RunElapsedS, 10),
		CategoryCount:            strconv.Itoa(len(summary.CategoryStats)),
	}); err != nil {
		return err
	}

	inputFileLabel := benchmarkInputFileFieldName(summary.ProjectFolder)
	inputCountLabel := benchmarkInputCountLabel(summary.ProjectFolder)
	theoremPackLine := ""
	if strings.TrimSpace(summary.TheoremPackID) != "" {
		theoremPackLine = fmt.Sprintf("theorem_pack_id: %s\n", summary.TheoremPackID)
	}
	googleThinkingLine := ""
	if strings.TrimSpace(summary.GoogleThinkingMode) != "" {
		googleThinkingLine = fmt.Sprintf("google_thinking_mode: %s\n", summary.GoogleThinkingMode)
	}
	chainSummaryLine := ""
	if strings.TrimSpace(summary.ChainSummaryPath) != "" {
		chainSummaryLine = fmt.Sprintf("chain_result_file: %s\n", summary.ChainSummaryPath)
	}
	importCatalogLine := ""
	if strings.TrimSpace(summary.ImportCatalogPath) != "" {
		importCatalogLine = fmt.Sprintf("import_catalog_file: %s\n", summary.ImportCatalogPath)
	}
	artifactKeyLine := ""
	if strings.TrimSpace(*artifactKey) != "" {
		artifactKeyLine = fmt.Sprintf("artifact_key: %s\n", strings.TrimSpace(*artifactKey))
	}
	_, _ = fmt.Fprintf(stdout, "hilbert benchmark completed\nrun_id: %s\n%sprovider: %s\nllm_model: %s\nruntime_model: %s\nproject_folder: %s\n%s%s: %s\ncase_selector: %s\nprompt_version: %s\n%shypothesis: %s\ncertificate_version: %s\n%s: %d\navg_latency_ms: %s\nmax_latency_ms: %d\nrun_elapsed_seconds: %d\nresult_file: %s\nresult_summary: %s\n%s%sraw: %s\n", summary.RunID, artifactKeyLine, summary.Provider, summary.LLMModel, runtimeLLMModel, summary.ProjectFolder, theoremPackLine, inputFileLabel, summary.CasesFile, summary.CaseSelector, summary.PromptVersion, googleThinkingLine, benchmarkHypothesis, benchmarkCertificateVersion, inputCountLabel, summary.CaseCount, formatBenchmarkFloat(summary.AvgLatencyMS), summary.MaxLatencyMS, summary.RunElapsedS, summary.ResultsPath, summary.SummaryPath, chainSummaryLine, importCatalogLine, summary.RawRunDir)
	for _, bucket := range orderedBenchmarkBuckets() {
		if count := summary.Buckets[bucket]; count > 0 {
			_, _ = fmt.Fprintf(stdout, "%s: %d\n", bucket, count)
		}
	}
	for _, category := range orderedCategoryNames(summary.CategoryStats) {
		stats := summary.CategoryStats[category]
		if stats["cases_total"] == 0 {
			continue
		}
		var parts []string
		parts = append(parts, fmt.Sprintf("cases_total=%d", stats["cases_total"]))
		for _, bucket := range orderedCategorySummaryBuckets() {
			if count := stats[bucket]; count > 0 {
				parts = append(parts, fmt.Sprintf("%s=%d", bucket, count))
			}
		}
		_, _ = fmt.Fprintf(stdout, "category[%s]: %s\n", category, strings.Join(parts, " "))
	}
	return nil
}

func runHilbertBenchmarkSummaryCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := firstNonEmptyEnv("HILBERT_BENCHMARK_THEOREMS_FILE", "HILBERT_BENCHMARK_CASES_FILE")
	if defaultCasesFile == "" {
		defaultCasesFile = defaultBenchmarkInputFile(defaultProjectFolder)
	}

	fs := flag.NewFlagSet("hilbert-benchmark-summary", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input CSV relative to the benchmark project folder")
	casesPath := fs.String("cases", firstNonEmptyEnv("HILBERT_BENCHMARK_THEOREMS", "HILBERT_BENCHMARK_CASES"), "Path to benchmark input CSV used to derive the benchmark root")
	summaryPath := fs.String("summary", firstNonEmptyEnv("HILBERT_BENCHMARK_SUMMARY"), "Path to benchmark run summary csv")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, strings.TrimSpace(*projectFolder))
	if strings.TrimSpace(*casesPath) != "" {
		*casesPath = resolveBenchmarkArgPath(workspaceRoot, *casesPath)
	}
	if strings.TrimSpace(*casesPath) == "" && strings.TrimSpace(*casesFile) != "*" {
		*casesPath = filepath.Join(benchmarkRoot, strings.TrimSpace(*casesFile))
	}
	resolvedProjectFolder := strings.TrimSpace(*projectFolder)
	if strings.TrimSpace(*casesPath) != "" {
		resolvedProjectFolder = inferBenchmarkProjectFolder(*casesPath, workspaceRoot, resolvedProjectFolder)
	}
	resolvedCasesFile := normalizeBenchmarkPath(strings.TrimSpace(*casesFile))
	if strings.TrimSpace(*casesPath) != "" && resolvedCasesFile != "*" {
		resolvedCasesFile = inferBenchmarkCasesFile(*casesPath, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)
	}
	benchmarkRoot = benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)
	summaryPaths := []string{strings.TrimSpace(*summaryPath)}
	if strings.TrimSpace(*summaryPath) == "" {
		summaryPaths = defaultBenchmarkSummaryPaths(benchmarkRoot)
		*summaryPath = summaryPaths[0]
	} else {
		*summaryPath = resolveBenchmarkArgPath(workspaceRoot, *summaryPath)
		summaryPaths = []string{*summaryPath}
	}

	rows, err := loadBenchmarkRunSummaryRowsFromPaths(summaryPaths)
	if err != nil {
		return err
	}
	rows = filterBenchmarkRunSummaryRows(rows, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)
	aggregates, err := aggregateBenchmarkRunSummary(rows)
	if err != nil {
		return err
	}

	inputFileLabel := benchmarkInputFileFieldName(resolvedProjectFolder)
	_, _ = fmt.Fprintf(stdout, "hilbert benchmark historical summary\nsummary_file: %s\nproject_folder: %s\n%s: %s\ngroups: %d\ntotal_runs: %d\n", *summaryPath, resolvedProjectFolder, inputFileLabel, resolvedCasesFile, len(aggregates), len(rows))
	for _, aggregate := range aggregates {
		passRate := 0.0
		if aggregate.CasesTotal > 0 {
			passRate = 100 * float64(aggregate.PassCount) / float64(aggregate.CasesTotal)
		}
		avgLatencyMS := 0.0
		if aggregate.LatencyCasesTotal > 0 {
			avgLatencyMS = aggregate.LatencySumMS / float64(aggregate.LatencyCasesTotal)
		}
		avgRunElapsedSeconds := 0.0
		if aggregate.TimedRuns > 0 {
			avgRunElapsedSeconds = float64(aggregate.RunElapsedTotalS) / float64(aggregate.TimedRuns)
		}
		_, _ = fmt.Fprintf(
			stdout,
			"group[model=%s llm_model=%s prompt_version=%s certificate_version=%s]: runs=%d cases_total=%d pass=%d pass_rate=%.2f%% false_refusal=%d false_accept=%d request_failure=%d schema_failure=%d parse_failure=%d kernel_failure=%d contract_failure=%d format_failure=%d avg_latency_ms=%s max_latency_ms=%d run_elapsed_seconds_total=%d avg_run_elapsed_seconds=%s max_run_elapsed_seconds=%d latest_run_id=%s latest_timestamp=%s\n",
			aggregate.Model,
			aggregate.LLMModel,
			aggregate.PromptVersion,
			aggregate.CertificateVersion,
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.PassCount,
			passRate,
			aggregate.FalseRefusalCount,
			aggregate.FalseAcceptCount,
			aggregate.RequestFailureCount,
			aggregate.SchemaFailureCount,
			aggregate.ParseFailureCount,
			aggregate.KernelFailureCount,
			aggregate.ContractFailureCount,
			aggregate.FormatFailureCount,
			formatBenchmarkFloat(avgLatencyMS),
			aggregate.MaxLatencyMS,
			aggregate.RunElapsedTotalS,
			formatBenchmarkFloat(avgRunElapsedSeconds),
			aggregate.RunElapsedMaxS,
			aggregate.LatestRunID,
			aggregate.LatestTimestampUTC,
		)
	}
	if benchmarkProjectSupportsObservedSlices(resolvedProjectFolder) {
		observed, err := buildBenchmarkObservedData(resolvedProjectFolder, rows, nil)
		if err != nil {
			return err
		}
		renderBenchmarkObservedSummary(stdout, observed)
	}
	return nil
}

func runHilbertBenchmarkReportCommand(args []string, stdout, stderr io.Writer) error {
	return runHilbertBenchmarkReportCommandImpl(args, stdout, stderr)
}

func runHilbertBenchmarkResearchReportCommand(args []string, stdout, stderr io.Writer) error {
	return runHilbertBenchmarkResearchReportCommandImpl(args, stdout, stderr)
}

func runHilbertBenchmarkMetaReportCommand(args []string, stdout, stderr io.Writer) error {
	return runHilbertBenchmarkMetaReportCommandImpl(args, stdout, stderr)
}

func runHilbertBenchmarkCaseReportCommand(args []string, stdout, stderr io.Writer) error {
	return runHilbertBenchmarkCaseReportCommandImpl(args, stdout, stderr)
}
