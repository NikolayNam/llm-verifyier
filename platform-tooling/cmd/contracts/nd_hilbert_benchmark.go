package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/naturaldeduction"
	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

const (
	ndBenchmarkPromptVersionV1      = "hilbert-ai-verification-benchmark-nd-v1"
	ndBenchmarkPromptVersionV11     = "hilbert-ai-verification-benchmark-nd-v1.1"
	ndBenchmarkPromptVersionDefault = ndBenchmarkPromptVersionV11
	ndBenchmarkHypothesis           = "A Natural Deduction front-end with deterministic lowering into Hilbert certificates may improve end-to-end verified performance while preserving the Hilbert kernel as the final trust boundary."
)

type ndBenchmarkPromptProfile struct {
	Version      string
	SystemPrompt string
}

func runNaturalDeductionBenchmarkCommand(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = benchmarkNDProjectFolder
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
	fs := flag.NewFlagSet("run-nd-hilbert-benchmark", flag.ContinueOnError)
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
	requestTimeoutAbortThreshold := fs.Int("request-timeout-abort-threshold", parseIntEnv(firstNonEmptyEnv("HILBERT_BENCHMARK_REQUEST_TIMEOUT_ABORT_THRESHOLD")), "After N consecutive request timeouts, stop new LLM requests and mark remaining cases as request_failure (0 disables)")
	runIDFlag := fs.String("run-id", runID, "Stable run identifier")
	promptVersion := fs.String("prompt-version", firstNonEmptyEnv("HILBERT_BENCHMARK_PROMPT_VERSION"), "Benchmark prompt profile version")
	hilbertcheckPath := fs.String("hilbertcheck-path", firstNonEmptyEnv("HILBERT_BENCHMARK_HILBERTCHECK"), "Optional path to a prebuilt hilbertcheck binary")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}
	if strings.TrimSpace(*promptVersion) == "" {
		*promptVersion = ndBenchmarkPromptVersionDefault
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
	if *requestTimeoutAbortThreshold < 0 {
		return fmt.Errorf("request-timeout-abort-threshold must be >= 0")
	}
	promptProfile, err := resolveNDBenchmarkPromptProfile(*promptVersion)
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

	tempDir, err := os.MkdirTemp("", "nd-hilbert-benchmark-*")
	if err != nil {
		return fmt.Errorf("create benchmark temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	verifier, err := resolveHilbertVerifier(context.Background(), workspaceRoot, tempDir, strings.TrimSpace(*hilbertcheckPath))
	if err != nil {
		return err
	}

	summary, err := runNaturalDeductionBenchmark(context.Background(), benchmarkRunOptions{
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
		Surface:                      "nd",
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
		Hypothesis:               ndBenchmarkHypothesis,
		CasesPath:                *casesPath,
		ResultsPath:              summary.ResultsPath,
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
	artifactKeyLine := ""
	if strings.TrimSpace(*artifactKey) != "" {
		artifactKeyLine = fmt.Sprintf("artifact_key: %s\n", strings.TrimSpace(*artifactKey))
	}
	_, _ = fmt.Fprintf(stdout, "nd->hilbert benchmark completed\nrun_id: %s\n%sprovider: %s\nllm_model: %s\nruntime_model: %s\nproject_folder: %s\n%s%s: %s\ncase_selector: %s\nprompt_version: %s\n%shypothesis: %s\ncertificate_version: %s\n%s: %d\navg_latency_ms: %s\nmax_latency_ms: %d\nrun_elapsed_seconds: %d\nresult_file: %s\nresult_summary: %s\nraw: %s\n", summary.RunID, artifactKeyLine, summary.Provider, summary.LLMModel, runtimeLLMModel, summary.ProjectFolder, theoremPackLine, inputFileLabel, summary.CasesFile, summary.CaseSelector, summary.PromptVersion, googleThinkingLine, ndBenchmarkHypothesis, benchmarkCertificateVersion, inputCountLabel, summary.CaseCount, formatBenchmarkFloat(summary.AvgLatencyMS), summary.MaxLatencyMS, summary.RunElapsedS, summary.ResultsPath, summary.SummaryPath, summary.RawRunDir)
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

func runNaturalDeductionBenchmark(ctx context.Context, opts benchmarkRunOptions, promptProfile ndBenchmarkPromptProfile, model benchmarkModel, verifier benchmarkVerifier, now func() time.Time) (*benchmarkSummary, error) {
	if strings.TrimSpace(opts.CasesPath) == "" {
		return nil, fmt.Errorf("benchmark cases path is required")
	}
	if strings.TrimSpace(opts.ResultsPath) == "" {
		return nil, fmt.Errorf("benchmark results path is required")
	}
	if strings.TrimSpace(opts.RawBaseDir) == "" {
		return nil, fmt.Errorf("benchmark raw dir is required")
	}
	if strings.TrimSpace(opts.RunID) == "" {
		return nil, fmt.Errorf("benchmark run id is required")
	}
	if strings.TrimSpace(promptProfile.Version) == "" || strings.TrimSpace(promptProfile.SystemPrompt) == "" {
		return nil, fmt.Errorf("benchmark prompt profile is required")
	}
	if now == nil {
		now = time.Now
	}
	runStartedAt := now()

	cases, err := loadBenchmarkCases(opts.CasesPath)
	if err != nil {
		return nil, err
	}
	cases = filterBenchmarkCases(cases, opts.CaseID, opts.Limit)
	if len(cases) == 0 {
		return nil, fmt.Errorf("no benchmark cases selected")
	}

	rawRunDir := filepath.Join(opts.RawBaseDir, opts.RunID)
	if err := os.MkdirAll(rawRunDir, 0o755); err != nil {
		return nil, fmt.Errorf("create raw output dir: %w", err)
	}

	rows := make([]benchmarkResultRow, 0, len(cases))
	buckets := make(map[string]int)
	var totalLatencyMS int64
	var maxLatencyMS int64
	latencySamples := 0
	consecutiveRequestTimeouts := 0
	samplingResolution := benchmarkSamplingResolution(opts)

	for idx, tc := range cases {
		userPrompt := buildNDBenchmarkUserPrompt(tc)
		promptHash := benchmarkPromptHash(promptProfile.SystemPrompt, userPrompt)
		rawOutput, latency, modelErr := model.Generate(ctx, promptProfile.SystemPrompt, userPrompt)
		latencyMS := latency.Milliseconds()
		totalLatencyMS += latencyMS
		latencySamples++
		if latencyMS > maxLatencyMS {
			maxLatencyMS = latencyMS
		}

		row := benchmarkResultRow{
			RunID:                    opts.RunID,
			TimestampUTC:             now().UTC().Format(time.RFC3339),
			Provider:                 benchmarkDisplayProvider(opts.Provider, opts.ProviderLabel),
			Model:                    opts.ModelLabel,
			LLMModel:                 opts.LLMModel,
			ExperimentName:           opts.ExperimentName,
			ExperimentID:             opts.ExperimentID,
			SamplingSurface:          samplingResolution.Surface,
			RequestedSamplingProfile: samplingResolution.RequestedProfile,
			EffectiveSamplingProfile: samplingResolution.EffectiveProfile,
			RequestedSamplingJSON:    samplingResolution.Requested.JSON(),
			EffectiveSamplingJSON:    samplingResolution.Effective.JSON(),
			UnsupportedSamplingJSON:  samplingResolution.Unsupported.JSON(),
			GoogleThinkingMode:       opts.GoogleThinkingMode,
			PromptVersion:            opts.PromptVersion,
			PromptHash:               promptHash,
			Surface:                  strings.TrimSpace(opts.Surface),
			NDProofStatus:            "not_run",
			LoweringStatus:           "not_run",
			TheoremPackID:            tc.TheoremPackID,
			TheoremID:                tc.TheoremID,
			CaseID:                   tc.CaseID,
			Category:                 tc.Category,
			ExpectedLabel:            tc.Label,
			SchemaStatus:             "not_run",
			ParseStatus:              "not_run",
			KernelStatus:             "not_run",
			LatencyMS:                strconv.FormatInt(latencyMS, 10),
		}

		rawPath := filepath.Join(rawRunDir, tc.CaseID+".txt")
		if modelErr != nil {
			rawOutput = "LLM_REQUEST_ERROR: " + strings.TrimSpace(modelErr.Error())
			row.RawOutputKind = "other"
			row.ScoreBucket = "request_failure"
			row.PipelineStage = "request"
			row.PipelineDetail = "request_failure"
			row.Notes = sanitizeNote(modelErr.Error())
			if isBenchmarkRequestTimeoutError(modelErr) {
				consecutiveRequestTimeouts++
			} else {
				consecutiveRequestTimeouts = 0
			}
		} else {
			consecutiveRequestTimeouts = 0
		}
		if err := os.WriteFile(rawPath, []byte(rawOutput), 0o644); err != nil {
			return nil, fmt.Errorf("write raw output %q: %w", rawPath, err)
		}

		if modelErr == nil {
			row.RawOutputKind = classifyNDBenchmarkRawOutput(rawOutput)
			switch row.RawOutputKind {
			case "not_derivable":
				row.PipelineStage = "llm_output"
				row.PipelineDetail = "not_derivable"
				if tc.Label == "not_entailed" {
					row.ScoreBucket = "pass"
				} else {
					row.ScoreBucket = "false_refusal"
				}
			case "invalid_json", "other":
				row.PipelineStage = "llm_output"
				row.PipelineDetail = row.RawOutputKind
				row.ScoreBucket = "format_failure"
			case "proof_object":
				proofJSONPath := filepath.Join(rawRunDir, tc.CaseID+".nd.json")
				trimmed := strings.TrimSpace(rawOutput)
				if err := os.WriteFile(proofJSONPath, []byte(trimmed), 0o644); err != nil {
					return nil, fmt.Errorf("write candidate ND proof %q: %w", proofJSONPath, err)
				}

				proof, err := naturaldeduction.DecodeJSON([]byte(trimmed))
				if err != nil {
					populateNDBenchmarkValidationStatuses(&row, err)
					break
				}
				row.NDProofStatus = "pass"
				row.SchemaStatus = "pass"
				row.ParseStatus = "pass"

				cert, err := naturaldeduction.LowerToCertificate(proof, naturaldeduction.LoweringOptions{})
				if err != nil {
					populateNDBenchmarkLoweringStatuses(&row, err)
					break
				}
				row.LoweringStatus = "pass"

				certBytes, err := json.Marshal(cert)
				if err != nil {
					return nil, fmt.Errorf("marshal lowered certificate for case %s: %w", tc.CaseID, err)
				}
				certJSONPath := filepath.Join(rawRunDir, tc.CaseID+".json")
				if err := os.WriteFile(certJSONPath, certBytes, 0o644); err != nil {
					return nil, fmt.Errorf("write lowered certificate %q: %w", certJSONPath, err)
				}

				verifyResult, err := verifier.Verify(ctx, certJSONPath)
				if err != nil {
					return nil, err
				}
				if err := populateVerificationStatuses(&row, tc, verifyResult); err != nil {
					return nil, fmt.Errorf("case %s: %w", tc.CaseID, err)
				}
				row.PipelineStage = "hilbert_artifact"
				row.PipelineDetail = row.ScoreBucket
			default:
				return nil, fmt.Errorf("case %s: unexpected raw output kind %q", tc.CaseID, row.RawOutputKind)
			}
		}

		buckets[row.ScoreBucket]++
		rows = append(rows, row)
		if modelErr != nil &&
			opts.RequestTimeoutAbortThreshold > 0 &&
			consecutiveRequestTimeouts >= opts.RequestTimeoutAbortThreshold &&
			idx+1 < len(cases) {
			if err := appendNDBenchmarkShortCircuitedTimeoutRows(&rows, buckets, cases[idx+1:], opts, promptProfile, rawRunDir, now, opts.RequestTimeoutAbortThreshold, modelErr.Error()); err != nil {
				return nil, err
			}
			break
		}
	}

	categoryStats := make(map[string]map[string]int)
	for _, row := range rows {
		if strings.TrimSpace(row.Category) == "" {
			continue
		}
		if _, ok := categoryStats[row.Category]; !ok {
			categoryStats[row.Category] = make(map[string]int)
		}
		categoryStats[row.Category]["cases_total"]++
		categoryStats[row.Category][row.ScoreBucket]++
	}

	if err := appendBenchmarkResults(opts.ResultsPath, rows); err != nil {
		return nil, err
	}
	runElapsed := now().Sub(runStartedAt)
	if runElapsed < 0 {
		runElapsed = 0
	}
	avgLatencyMS := 0.0
	if latencySamples > 0 {
		avgLatencyMS = float64(totalLatencyMS) / float64(latencySamples)
	}

	return &benchmarkSummary{
		RunID:                    opts.RunID,
		Provider:                 benchmarkDisplayProvider(opts.Provider, opts.ProviderLabel),
		LLMModel:                 opts.LLMModel,
		ExperimentName:           opts.ExperimentName,
		ExperimentID:             opts.ExperimentID,
		SamplingSurface:          samplingResolution.Surface,
		RequestedSamplingProfile: samplingResolution.RequestedProfile,
		EffectiveSamplingProfile: samplingResolution.EffectiveProfile,
		RequestedSamplingJSON:    samplingResolution.Requested.JSON(),
		EffectiveSamplingJSON:    samplingResolution.Effective.JSON(),
		UnsupportedSamplingJSON:  samplingResolution.Unsupported.JSON(),
		GoogleThinkingMode:       opts.GoogleThinkingMode,
		PromptVersion:            opts.PromptVersion,
		ProjectFolder:            opts.ProjectFolder,
		Surface:                  strings.TrimSpace(opts.Surface),
		TheoremPackID:            benchmarkTheoremPackID(cases, opts.CasesPath),
		CasesFile:                opts.CasesFile,
		CaseSelector:             opts.CaseSelector,
		ResultsPath:              normalizeBenchmarkPath(opts.ResultsPath),
		SummaryPath:              normalizeBenchmarkPath(opts.SummaryPath),
		RawRunDir:                normalizeBenchmarkPath(rawRunDir),
		CaseCount:                len(rows),
		AvgLatencyMS:             avgLatencyMS,
		MaxLatencyMS:             maxLatencyMS,
		RunElapsedS:              int64(runElapsed / time.Second),
		Buckets:                  buckets,
		CategoryStats:            categoryStats,
	}, nil
}

func appendNDBenchmarkShortCircuitedTimeoutRows(rows *[]benchmarkResultRow, buckets map[string]int, remaining []benchmarkCase, opts benchmarkRunOptions, promptProfile ndBenchmarkPromptProfile, rawRunDir string, now func() time.Time, threshold int, lastErr string) error {
	if len(remaining) == 0 {
		return nil
	}
	if rows == nil {
		return fmt.Errorf("benchmark rows accumulator is nil")
	}
	syntheticRaw := "LLM_REQUEST_ERROR: " + benchmarkTimeoutAbortNote(threshold, lastErr)
	note := sanitizeNote(strings.TrimPrefix(syntheticRaw, "LLM_REQUEST_ERROR: "))
	samplingResolution := benchmarkSamplingResolution(opts)
	for _, tc := range remaining {
		row := benchmarkResultRow{
			RunID:                    opts.RunID,
			TimestampUTC:             now().UTC().Format(time.RFC3339),
			Provider:                 benchmarkDisplayProvider(opts.Provider, opts.ProviderLabel),
			Model:                    opts.ModelLabel,
			LLMModel:                 opts.LLMModel,
			ExperimentName:           opts.ExperimentName,
			ExperimentID:             opts.ExperimentID,
			SamplingSurface:          samplingResolution.Surface,
			RequestedSamplingProfile: samplingResolution.RequestedProfile,
			EffectiveSamplingProfile: samplingResolution.EffectiveProfile,
			RequestedSamplingJSON:    samplingResolution.Requested.JSON(),
			EffectiveSamplingJSON:    samplingResolution.Effective.JSON(),
			UnsupportedSamplingJSON:  samplingResolution.Unsupported.JSON(),
			GoogleThinkingMode:       opts.GoogleThinkingMode,
			PromptVersion:            opts.PromptVersion,
			PromptHash:               benchmarkPromptHash(promptProfile.SystemPrompt, buildNDBenchmarkUserPrompt(tc)),
			Surface:                  strings.TrimSpace(opts.Surface),
			NDProofStatus:            "not_run",
			LoweringStatus:           "not_run",
			PipelineStage:            "request",
			PipelineDetail:           "request_failure",
			TheoremPackID:            tc.TheoremPackID,
			TheoremID:                tc.TheoremID,
			CaseID:                   tc.CaseID,
			Category:                 tc.Category,
			ExpectedLabel:            tc.Label,
			RawOutputKind:            "other",
			SchemaStatus:             "not_run",
			ParseStatus:              "not_run",
			KernelStatus:             "not_run",
			ScoreBucket:              "request_failure",
			Notes:                    note,
		}
		rawPath := filepath.Join(rawRunDir, tc.CaseID+".txt")
		if err := os.WriteFile(rawPath, []byte(syntheticRaw), 0o644); err != nil {
			return fmt.Errorf("write short-circuited raw output %q: %w", rawPath, err)
		}
		buckets[row.ScoreBucket]++
		*rows = append(*rows, row)
	}
	return nil
}

func resolveNDBenchmarkPromptProfile(version string) (ndBenchmarkPromptProfile, error) {
	version = strings.TrimSpace(version)
	profiles, err := loadNDBenchmarkPromptProfiles()
	if err != nil {
		return ndBenchmarkPromptProfile{}, err
	}
	profile, ok := profiles[version]
	if !ok {
		return ndBenchmarkPromptProfile{}, fmt.Errorf("unsupported ND benchmark prompt version %q", version)
	}
	return profile, nil
}

func buildNDBenchmarkUserPrompt(tc benchmarkCase) string {
	var b strings.Builder
	if strings.TrimSpace(tc.TheoremPackID) != "" {
		b.WriteString("Theorem Pack ID: ")
		b.WriteString(tc.TheoremPackID)
		b.WriteString("\n")
	}
	if strings.TrimSpace(tc.TheoremID) != "" {
		b.WriteString("Theorem ID: ")
		b.WriteString(tc.TheoremID)
	} else {
		b.WriteString("Case ID: ")
		b.WriteString(tc.CaseID)
	}
	b.WriteString("\n\nAssumptions:\n")
	if len(tc.Assumptions) == 0 {
		b.WriteString("(none)\n")
	} else {
		for idx, assumption := range tc.Assumptions {
			_, _ = fmt.Fprintf(&b, "%d. %s\n", idx+1, assumption)
		}
	}
	b.WriteString("\nGoal:\n")
	b.WriteString(tc.Goal)
	b.WriteString("\n\nReturn only:\n- a valid natural-deduction-v1 JSON proof object for this goal from these assumptions; or\n- NOT_DERIVABLE\n")
	return b.String()
}

func classifyNDBenchmarkRawOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "NOT_DERIVABLE":
		return "not_derivable"
	case strings.HasPrefix(trimmed, "{"):
		if json.Valid([]byte(trimmed)) {
			return "proof_object"
		}
		return "invalid_json"
	default:
		return "other"
	}
}

func populateNDBenchmarkValidationStatuses(row *benchmarkResultRow, err error) {
	row.Notes = sanitizeNote(err.Error())
	row.NDProofStatus = "fail"
	row.LoweringStatus = "not_run"
	row.PipelineStage = "nd_proof_object"
	switch naturaldeduction.ClassOf(err) {
	case naturaldeduction.ErrorClassParse:
		row.SchemaStatus = "pass"
		row.ParseStatus = "fail"
		row.ScoreBucket = "parse_failure"
		row.PipelineDetail = string(naturaldeduction.ErrorClassParse)
	case naturaldeduction.ErrorClassSchema, naturaldeduction.ErrorClassValidation:
		row.SchemaStatus = "fail"
		row.ScoreBucket = "schema_failure"
		row.PipelineDetail = string(naturaldeduction.ClassOf(err))
	default:
		row.ScoreBucket = "format_failure"
		row.PipelineDetail = string(naturaldeduction.ErrorClassInternal)
	}
}

func populateNDBenchmarkLoweringStatuses(row *benchmarkResultRow, err error) {
	row.Notes = sanitizeNote(err.Error())
	row.NDProofStatus = "pass"
	row.LoweringStatus = "fail"
	row.PipelineStage = "lowering"
	class := naturaldeduction.ClassOf(err)
	if class == "" {
		class = naturaldeduction.ErrorClassInternal
	}
	row.PipelineDetail = string(class)
	switch class {
	case naturaldeduction.ErrorClassParse:
		row.ScoreBucket = "parse_failure"
	case naturaldeduction.ErrorClassSchema:
		row.ScoreBucket = "schema_failure"
	case naturaldeduction.ErrorClassValidation:
		row.ScoreBucket = "kernel_failure"
	default:
		row.ScoreBucket = "format_failure"
	}
}
