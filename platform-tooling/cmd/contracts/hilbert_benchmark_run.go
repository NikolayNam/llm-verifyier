package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/llmclient/compatible"
	openai "github.com/NikolayNam/collabsphere/platform-tooling/internal/llmclient/openai"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/benchmarkcases"
	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

func runHilbertBenchmark(ctx context.Context, opts benchmarkRunOptions, promptProfile benchmarkPromptProfile, model benchmarkModel, verifier benchmarkVerifier, now func() time.Time) (*benchmarkSummary, error) {
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
	if err := validateBenchmarkSelectedCaseDependencies(cases); err != nil {
		return nil, fmt.Errorf("invalid benchmark dependency graph for selected slice: %w", err)
	}
	cases = orderBenchmarkCasesForExecution(cases)

	rawRunDir := filepath.Join(opts.RawBaseDir, opts.RunID)
	if err := os.MkdirAll(rawRunDir, 0o755); err != nil {
		return nil, fmt.Errorf("create raw output dir: %w", err)
	}

	rows := make([]benchmarkResultRow, 0, len(cases))
	trustedArtifacts := make([]benchmarkTrustedCertificateObject, 0, len(cases))
	trustedArtifactCatalog := make([]benchmarkTrustedCertificateCatalogRow, 0, len(cases))
	buckets := make(map[string]int)
	var totalLatencyMS int64
	var maxLatencyMS int64
	latencySamples := 0
	consecutiveRequestTimeouts := 0
	for idx, tc := range cases {
		resolution, err := resolveBenchmarkPromptCase(tc, trustedArtifacts)
		if err != nil {
			return nil, err
		}
		promptCase := resolution.ResolvedCase
		userPrompt := buildBenchmarkUserPrompt(promptCase)
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
			SamplingSurface:          benchmarkSamplingResolution(opts).Surface,
			RequestedSamplingProfile: benchmarkSamplingResolution(opts).RequestedProfile,
			EffectiveSamplingProfile: benchmarkSamplingResolution(opts).EffectiveProfile,
			RequestedSamplingJSON:    benchmarkSamplingResolution(opts).Requested.JSON(),
			EffectiveSamplingJSON:    benchmarkSamplingResolution(opts).Effective.JSON(),
			UnsupportedSamplingJSON:  benchmarkSamplingResolution(opts).Unsupported.JSON(),
			GoogleThinkingMode:       opts.GoogleThinkingMode,
			PromptVersion:            opts.PromptVersion,
			PromptHash:               promptHash,
			Surface:                  strings.TrimSpace(opts.Surface),
			TheoremPackID:            tc.TheoremPackID,
			TheoremID:                tc.TheoremID,
			CaseID:                   tc.CaseID,
			ChainID:                  tc.ChainID,
			StageID:                  tc.StageID,
			Category:                 tc.Category,
			CaseFamily:               tc.CaseFamily,
			ChainDepth:               strconv.Itoa(tc.ChainDepth),
			ProvenanceMode:           tc.ProvenanceMode,
			AtomRenamingID:           tc.AtomRenamingID,
			SourceFamily:             tc.SourceFamily,
			TargetFamily:             tc.TargetFamily,
			TransitionGroup:          tc.TransitionGroup,
			ImportArity:              tc.ImportArity,
			ReuseShape:               tc.ReuseShape,
			BridgeDepth:              tc.BridgeDepth,
			SymbolOverlap:            tc.SymbolOverlap,
			LemmaSurfaceStyle:        tc.LemmaSurfaceStyle,
			NegativeTwinHardness:     tc.NegativeTwinHardness,
			CaseNotes:                tc.Notes,
			ExpectedLabel:            tc.Label,
			SchemaStatus:             "not_run",
			ParseStatus:              "not_run",
			KernelStatus:             "not_run",
			ContractStatus:           "not_run",
			LatencyMS:                strconv.FormatInt(latencyMS, 10),
			RequestedImportsJSON:     benchmarkJSONString(tc.ImportedLemmas),
			EffectiveImportsJSON:     benchmarkJSONString(promptCase.ImportedLemmas),
			ResolvedImportRefsJSON:   benchmarkJSONString(resolution.ResolvedArtifacts),
			Notes:                    benchmarkResolvedImportNote(tc, resolution),
		}

		rawPath := filepath.Join(rawRunDir, tc.CaseID+".txt")
		if modelErr != nil {
			rawOutput = "LLM_REQUEST_ERROR: " + strings.TrimSpace(modelErr.Error())
			row.RawOutputKind = "other"
			row.ScoreBucket = "request_failure"
			row.Notes = appendBenchmarkNote(row.Notes, sanitizeNote(modelErr.Error()))
			if isBenchmarkRequestTimeoutError(modelErr) {
				consecutiveRequestTimeouts++
			} else {
				consecutiveRequestTimeouts = 0
			}
		} else {
			consecutiveRequestTimeouts = 0
		}
		row.OutputLength = strconv.Itoa(len(strings.TrimSpace(rawOutput)))
		row.ProofStepsCount = strconv.Itoa(benchmarkProofStepsCount(rawOutput))

		if err := os.WriteFile(rawPath, []byte(rawOutput), 0o644); err != nil {
			return nil, fmt.Errorf("write raw output %q: %w", rawPath, err)
		}

		if modelErr == nil {
			row.RawOutputKind = classifyBenchmarkRawOutput(rawOutput)
			switch row.RawOutputKind {
			case "not_derivable":
				if tc.Label == "not_entailed" {
					row.ScoreBucket = "pass"
				} else {
					row.ScoreBucket = "false_refusal"
				}
			case "invalid_json", "other":
				row.ScoreBucket = "format_failure"
			case "certificate":
				jsonPath := filepath.Join(rawRunDir, tc.CaseID+".json")
				if err := os.WriteFile(jsonPath, []byte(strings.TrimSpace(rawOutput)), 0o644); err != nil {
					return nil, fmt.Errorf("write candidate certificate %q: %w", jsonPath, err)
				}
				verifyResult, err := verifier.Verify(ctx, jsonPath)
				if err != nil {
					return nil, err
				}
				if err := populateVerificationStatuses(&row, tc, verifyResult); err != nil {
					return nil, fmt.Errorf("case %s: %w", tc.CaseID, err)
				}
				if strings.TrimSpace(row.ScoreBucket) == "pass" {
					if err := populateBenchmarkContractStatus(&row, rawOutput); err != nil {
						return nil, fmt.Errorf("case %s: %w", tc.CaseID, err)
					}
				}
				if strings.TrimSpace(row.ScoreBucket) == "pass" && benchmarkCaseUsesCompositionalChainProtocol(tc) {
					certificateObjectPath := benchmarkImportedCertificateObjectPath(opts.ResultsPath, opts.RunID, tc.CaseID)
					object, err := writeBenchmarkTrustedCertificateObject(certificateObjectPath, opts, promptCase, row, rawOutput)
					if err != nil {
						return nil, err
					}
					row.CertificateObjectPath = object.CertificateObjectPath
					trustedArtifacts = append(trustedArtifacts, object)
					trustedArtifactCatalog = append(trustedArtifactCatalog, benchmarkTrustedCertificateCatalogRow{
						RunID:                 object.RunID,
						CaseID:                object.CaseID,
						ChainID:               object.ChainID,
						StageID:               object.StageID,
						Category:              object.Category,
						CaseFamily:            object.CaseFamily,
						ChainDepth:            strconv.Itoa(object.ChainDepth),
						Goal:                  object.Goal,
						TrustedForReuse:       strconv.FormatBool(object.TrustedForReuse),
						TrustedReuseReason:    object.TrustedReuseReason,
						CertificateObjectPath: object.CertificateObjectPath,
					})
				}
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
			if err := appendBenchmarkShortCircuitedTimeoutRows(&rows, buckets, cases[idx+1:], opts, promptProfile, rawRunDir, now, opts.RequestTimeoutAbortThreshold, modelErr.Error()); err != nil {
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
	importCatalogPath := ""
	if len(trustedArtifactCatalog) > 0 {
		importCatalogPath = benchmarkImportedCertificateCatalogPath(opts.ResultsPath, opts.RunID)
		if err := appendBenchmarkTrustedCertificateCatalog(importCatalogPath, trustedArtifactCatalog); err != nil {
			return nil, err
		}
	}
	chainSummaryPath := ""
	if benchmarkCasesHaveChainMetadata(cases) {
		chainSummaryPath = benchmarkChainSummaryPath(opts.ResultsPath, opts.RunID)
		if err := appendBenchmarkChainSummary(chainSummaryPath, opts.RunID, cases, rows); err != nil {
			return nil, err
		}
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
		SamplingSurface:          benchmarkSamplingResolution(opts).Surface,
		RequestedSamplingProfile: benchmarkSamplingResolution(opts).RequestedProfile,
		EffectiveSamplingProfile: benchmarkSamplingResolution(opts).EffectiveProfile,
		RequestedSamplingJSON:    benchmarkSamplingResolution(opts).Requested.JSON(),
		EffectiveSamplingJSON:    benchmarkSamplingResolution(opts).Effective.JSON(),
		UnsupportedSamplingJSON:  benchmarkSamplingResolution(opts).Unsupported.JSON(),
		GoogleThinkingMode:       opts.GoogleThinkingMode,
		PromptVersion:            opts.PromptVersion,
		ProjectFolder:            opts.ProjectFolder,
		Surface:                  strings.TrimSpace(opts.Surface),
		TheoremPackID:            benchmarkTheoremPackID(cases, opts.CasesPath),
		CasesFile:                opts.CasesFile,
		CaseSelector:             opts.CaseSelector,
		ResultsPath:              normalizeBenchmarkPath(opts.ResultsPath),
		SummaryPath:              normalizeBenchmarkPath(opts.SummaryPath),
		ChainSummaryPath:         normalizeBenchmarkPath(chainSummaryPath),
		ImportCatalogPath:        normalizeBenchmarkPath(importCatalogPath),
		RawRunDir:                normalizeBenchmarkPath(rawRunDir),
		CaseCount:                len(rows),
		AvgLatencyMS:             avgLatencyMS,
		MaxLatencyMS:             maxLatencyMS,
		RunElapsedS:              int64(runElapsed / time.Second),
		Buckets:                  buckets,
		CategoryStats:            categoryStats,
	}, nil
}

func isBenchmarkRequestTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(text, "context deadline exceeded"),
		strings.Contains(text, "client.timeout exceeded"),
		strings.Contains(text, "timeout"):
		return true
	default:
		return false
	}
}

func appendBenchmarkShortCircuitedTimeoutRows(rows *[]benchmarkResultRow, buckets map[string]int, remaining []benchmarkCase, opts benchmarkRunOptions, promptProfile benchmarkPromptProfile, rawRunDir string, now func() time.Time, threshold int, lastErr string) error {
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
			PromptHash:               benchmarkPromptHash(promptProfile.SystemPrompt, buildBenchmarkUserPrompt(tc)),
			Surface:                  strings.TrimSpace(opts.Surface),
			TheoremPackID:            tc.TheoremPackID,
			TheoremID:                tc.TheoremID,
			CaseID:                   tc.CaseID,
			ChainID:                  tc.ChainID,
			StageID:                  tc.StageID,
			Category:                 tc.Category,
			CaseFamily:               tc.CaseFamily,
			ChainDepth:               strconv.Itoa(tc.ChainDepth),
			ProvenanceMode:           tc.ProvenanceMode,
			AtomRenamingID:           tc.AtomRenamingID,
			SourceFamily:             tc.SourceFamily,
			TargetFamily:             tc.TargetFamily,
			TransitionGroup:          tc.TransitionGroup,
			ImportArity:              tc.ImportArity,
			ReuseShape:               tc.ReuseShape,
			BridgeDepth:              tc.BridgeDepth,
			SymbolOverlap:            tc.SymbolOverlap,
			LemmaSurfaceStyle:        tc.LemmaSurfaceStyle,
			NegativeTwinHardness:     tc.NegativeTwinHardness,
			CaseNotes:                tc.Notes,
			ExpectedLabel:            tc.Label,
			RawOutputKind:            "other",
			SchemaStatus:             "not_run",
			ParseStatus:              "not_run",
			KernelStatus:             "not_run",
			ContractStatus:           "not_run",
			ScoreBucket:              "request_failure",
			OutputLength:             strconv.Itoa(len(strings.TrimSpace(syntheticRaw))),
			ProofStepsCount:          "0",
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

func benchmarkTimeoutAbortNote(threshold int, lastErr string) string {
	return fmt.Sprintf("benchmark short-circuited remaining cases after %d consecutive request timeouts; last observed error: %s", threshold, strings.TrimSpace(lastErr))
}

func benchmarkProofStepsCount(rawOutput string) int {
	trimmed := strings.TrimSpace(rawOutput)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return 0
	}
	var payload struct {
		Steps []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return 0
	}
	return len(payload.Steps)
}

func benchmarkJSONString(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func benchmarkImportedCertificateObjectPath(resultsPath, runID, caseID string) string {
	if strings.TrimSpace(resultsPath) == "" || strings.TrimSpace(runID) == "" || strings.TrimSpace(caseID) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(resultsPath), "imported_certificates", strings.TrimSpace(runID), strings.TrimSpace(caseID)+".json")
}

func benchmarkImportedCertificateCatalogPath(resultsPath, runID string) string {
	if strings.TrimSpace(resultsPath) == "" || strings.TrimSpace(runID) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(resultsPath), fmt.Sprintf("imported_certificate_catalog_%s.csv", strings.TrimSpace(runID)))
}

func writeBenchmarkTrustedCertificateObject(path string, opts benchmarkRunOptions, tc benchmarkCase, row benchmarkResultRow, rawOutput string) (benchmarkTrustedCertificateObject, error) {
	rawOutput = strings.TrimSpace(rawOutput)
	if strings.TrimSpace(path) == "" {
		return benchmarkTrustedCertificateObject{}, fmt.Errorf("trusted certificate object path is required")
	}
	if rawOutput == "" || !json.Valid([]byte(rawOutput)) {
		return benchmarkTrustedCertificateObject{}, fmt.Errorf("trusted certificate object requires valid certificate json for case %s", tc.CaseID)
	}
	object := benchmarkTrustedCertificateObject{
		SchemaVersion:         "research.imported-certificate/v1",
		RunID:                 opts.RunID,
		ProjectFolder:         opts.ProjectFolder,
		Surface:               strings.TrimSpace(opts.Surface),
		ExperimentName:        opts.ExperimentName,
		ExperimentID:          opts.ExperimentID,
		PromptVersion:         opts.PromptVersion,
		CertificateVersion:    benchmarkCertificateVersion,
		CaseID:                tc.CaseID,
		TheoremID:             firstNonEmptyBenchmarkCell(tc.TheoremID, tc.CaseID),
		ChainID:               tc.ChainID,
		StageID:               tc.StageID,
		Category:              tc.Category,
		CaseFamily:            tc.CaseFamily,
		ChainDepth:            tc.ChainDepth,
		AtomRenamingID:        tc.AtomRenamingID,
		Goal:                  tc.Goal,
		Assumptions:           append([]string(nil), tc.Assumptions...),
		ImportedLemmas:        append([]string(nil), tc.ImportedLemmas...),
		ExpectedBehavior:      tc.ExpectedBehavior,
		RawOutputKind:         row.RawOutputKind,
		ScoreBucket:           row.ScoreBucket,
		SchemaStatus:          row.SchemaStatus,
		ParseStatus:           row.ParseStatus,
		KernelStatus:          row.KernelStatus,
		CLIExitCode:           row.CLIExitCode,
		TrustedForReuse:       benchmarkCaseTrustedReuseEligible(tc) && strings.TrimSpace(row.ScoreBucket) == "pass",
		TrustedReuseReason:    benchmarkTrustedReuseReason(tc, row.ScoreBucket),
		CertificateObjectPath: normalizeBenchmarkPath(path),
		CertificateJSON:       json.RawMessage(rawOutput),
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return benchmarkTrustedCertificateObject{}, fmt.Errorf("create imported certificate object dir: %w", err)
	}
	payload, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return benchmarkTrustedCertificateObject{}, fmt.Errorf("marshal imported certificate object: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return benchmarkTrustedCertificateObject{}, fmt.Errorf("write imported certificate object %q: %w", path, err)
	}
	return object, nil
}

func benchmarkTrustedReuseReason(tc benchmarkCase, scoreBucket string) string {
	scoreBucket = strings.TrimSpace(scoreBucket)
	switch {
	case scoreBucket != "pass":
		return "eligible only after verifier-confirmed pass"
	case !benchmarkCaseTrustedReuseEligible(tc):
		return "stage is not marked reusable in the current chain protocol"
	default:
		return "verifier-confirmed reusable local lemma artifact"
	}
}

func appendBenchmarkTrustedCertificateCatalog(path string, rows []benchmarkTrustedCertificateCatalogRow) error {
	if strings.TrimSpace(path) == "" || len(rows) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create imported certificate catalog dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open imported certificate catalog: %w", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	header := []string{
		"run_id", "case_id", "chain_id", "stage_id", "category", "case_family", "chain_depth", "goal",
		"trusted_for_reuse", "trusted_reuse_reason", "certificate_object_file",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write imported certificate catalog header: %w", err)
	}
	for _, row := range rows {
		record := []string{
			row.RunID, row.CaseID, row.ChainID, row.StageID, row.Category, row.CaseFamily, row.ChainDepth, row.Goal,
			row.TrustedForReuse, row.TrustedReuseReason, row.CertificateObjectPath,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write imported certificate catalog row for case %s: %w", row.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush imported certificate catalog csv: %w", err)
	}
	return nil
}

func resolveBenchmarkPromptProfile(version string) (benchmarkPromptProfile, error) {
	version = strings.TrimSpace(version)
	profiles, err := loadBenchmarkPromptProfiles()
	if err != nil {
		return benchmarkPromptProfile{}, err
	}
	profile, ok := profiles[version]
	if !ok {
		return benchmarkPromptProfile{}, fmt.Errorf("unsupported benchmark prompt version %q", version)
	}
	return profile, nil
}

func newBenchmarkModel(provider, baseURL, apiKey, model, googleThinkingMode string, googleRequestsPerMinute, mistralRequestsPerSecond int, timeout time.Duration, effectiveSampling *researchsampling.Params) (benchmarkModel, string, string, error) {
	switch normalizeBenchmarkProvider(provider) {
	case "", "compatible":
		client, err := compatible.NewClient(baseURL, apiKey, model, timeout)
		if err != nil {
			return nil, "", "", err
		}
		resolvedModel := strings.TrimSpace(model)
		if resolvedModel == "" {
			resolvedModel = "llama3.2"
		}
		return &llmPromptRunner{
			client:      client,
			model:       resolvedModel,
			temperature: benchmarkSamplingTemperature(effectiveSampling),
			seed:        benchmarkSamplingSeed(effectiveSampling),
			topP:        benchmarkSamplingTopP(effectiveSampling),
		}, "compatible:" + resolvedModel, resolvedModel, nil
	case "openai":
		client, err := openai.NewHTTPClient(baseURL, apiKey, model, timeout)
		if err != nil {
			return nil, "", "", err
		}
		resolvedModel := strings.TrimSpace(model)
		if resolvedModel == "" {
			resolvedModel = "gpt-4.1-mini"
		}
		return &llmPromptRunner{
			client:      client,
			model:       resolvedModel,
			temperature: benchmarkSamplingTemperature(effectiveSampling),
			seed:        benchmarkSamplingSeed(effectiveSampling),
			topP:        benchmarkSamplingTopP(effectiveSampling),
		}, "openai:" + resolvedModel, resolvedModel, nil
	case "google":
		return newGoogleBenchmarkModel(baseURL, apiKey, model, googleThinkingMode, googleRequestsPerMinute, timeout, effectiveSampling)
	case "mistral":
		return newMistralBenchmarkModel(baseURL, apiKey, model, mistralRequestsPerSecond, timeout, effectiveSampling)
	default:
		return nil, "", "", fmt.Errorf("unsupported benchmark provider %q", provider)
	}
}

func buildHilbertCLIVerifier(ctx context.Context, workspaceRoot, tempDir string) (benchmarkVerifier, error) {
	toolingRoot := filepath.Join(workspaceRoot, "platform-tooling")
	binaryName := "hilbertcheck"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)
	cmd := exec.CommandContext(ctx, "go", "-C", toolingRoot, "build", "-o", binaryPath, "./cmd/hilbertcheck")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("build hilbertcheck binary: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return &hilbertCLIVerifier{binaryPath: binaryPath}, nil
}

func resolveHilbertVerifier(ctx context.Context, workspaceRoot, tempDir, existingBinaryPath string) (benchmarkVerifier, error) {
	if strings.TrimSpace(existingBinaryPath) == "" {
		return buildHilbertCLIVerifier(ctx, workspaceRoot, tempDir)
	}
	resolvedPath := resolveBenchmarkArgPath(workspaceRoot, existingBinaryPath)
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("stat hilbertcheck binary %q: %w", resolvedPath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("hilbertcheck path %q is a directory", resolvedPath)
	}
	return &hilbertCLIVerifier{binaryPath: resolvedPath}, nil
}

func loadBenchmarkCases(path string) ([]benchmarkCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open benchmark cases: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read benchmark cases csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("benchmark cases csv must contain a header and at least one row")
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	required := []string{"label", "difficulty", "assumptions_json", "goal", "comment"}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, fmt.Errorf("benchmark cases csv is missing required column %q", name)
		}
	}
	if _, ok := header["case_id"]; !ok {
		if _, ok := header["theorem_id"]; !ok {
			return nil, fmt.Errorf("benchmark cases csv must contain either %q or %q", "case_id", "theorem_id")
		}
	}

	cases := make([]benchmarkCase, 0, len(records)-1)
	defaultPackID := deriveBenchmarkTheoremPackID(path)
	for rowIdx, record := range records[1:] {
		assumptions := make([]string, 0)
		if raw := strings.TrimSpace(record[header["assumptions_json"]]); raw != "" {
			if err := json.Unmarshal([]byte(raw), &assumptions); err != nil {
				return nil, fmt.Errorf("benchmark case row %d assumptions_json: %w", rowIdx+2, err)
			}
		}
		importedLemmas := make([]string, 0)
		if _, ok := header["imported_lemmas_json"]; ok {
			if raw := strings.TrimSpace(record[header["imported_lemmas_json"]]); raw != "" {
				if err := json.Unmarshal([]byte(raw), &importedLemmas); err != nil {
					return nil, fmt.Errorf("benchmark case row %d imported_lemmas_json: %w", rowIdx+2, err)
				}
			}
		}
		importStageIDs := make([]string, 0)
		if _, ok := header["import_stage_ids_json"]; ok {
			if raw := strings.TrimSpace(record[header["import_stage_ids_json"]]); raw != "" {
				if err := json.Unmarshal([]byte(raw), &importStageIDs); err != nil {
					return nil, fmt.Errorf("benchmark case row %d import_stage_ids_json: %w", rowIdx+2, err)
				}
			}
		}
		trustedReuse, err := parseOptionalBenchmarkBool(header, record, "trusted_reuse")
		if err != nil {
			return nil, fmt.Errorf("benchmark case row %d trusted_reuse: %w", rowIdx+2, err)
		}
		theoremID := csvCell(record, header, "theorem_id")
		caseID := csvCell(record, header, "case_id")
		if theoremID == "" {
			theoremID = caseID
		}
		if caseID == "" {
			caseID = theoremID
		}
		if theoremID != "" && caseID != "" && theoremID != caseID && csvCell(record, header, "case_id") != "" && csvCell(record, header, "theorem_id") != "" {
			return nil, fmt.Errorf("benchmark case row %d has mismatched case_id %q and theorem_id %q", rowIdx+2, caseID, theoremID)
		}
		cases = append(cases, benchmarkCase{
			TheoremPackID:        firstNonEmptyBenchmarkCell(csvCell(record, header, "theorem_pack_id"), defaultPackID),
			TheoremID:            theoremID,
			CaseID:               caseID,
			SourceCaseID:         csvCell(record, header, "source_case_id"),
			ChainID:              csvCell(record, header, "chain_id"),
			ChainProtocol:        csvCell(record, header, "chain_protocol"),
			StageID:              csvCell(record, header, "stage_id"),
			StageOrder:           parseIntLoose(csvCell(record, header, "stage_order")),
			StageRole:            csvCell(record, header, "stage_role"),
			Category:             csvCell(record, header, "category"),
			Label:                strings.TrimSpace(record[header["label"]]),
			Difficulty:           strings.TrimSpace(record[header["difficulty"]]),
			CaseFamily:           csvCell(record, header, "case_family"),
			ChainDepth:           parseIntLoose(csvCell(record, header, "chain_depth")),
			ProvenanceMode:       csvCell(record, header, "provenance_mode"),
			AtomRenamingID:       csvCell(record, header, "atom_renaming_id"),
			SourceFamily:         csvCell(record, header, "source_family"),
			TargetFamily:         csvCell(record, header, "target_family"),
			TransitionGroup:      csvCell(record, header, "transition_group"),
			ImportArity:          csvCell(record, header, "import_arity"),
			ReuseShape:           csvCell(record, header, "reuse_shape"),
			BridgeDepth:          csvCell(record, header, "bridge_depth"),
			SymbolOverlap:        csvCell(record, header, "symbol_overlap"),
			LemmaSurfaceStyle:    csvCell(record, header, "lemma_surface_style"),
			NegativeTwinHardness: csvCell(record, header, "negative_twin_hardness"),
			Notes:                csvCell(record, header, "notes"),
			ExpectedBehavior:     csvCell(record, header, "expected_behavior"),
			TrustedReuse:         trustedReuse,
			ImportStageIDs:       importStageIDs,
			Assumptions:          assumptions,
			ImportedLemmas:       importedLemmas,
			Goal:                 strings.TrimSpace(record[header["goal"]]),
			Comment:              strings.TrimSpace(record[header["comment"]]),
		})
	}
	return cases, nil
}

func filterBenchmarkCases(cases []benchmarkCase, caseID string, limit int) []benchmarkCase {
	filtered := make([]benchmarkCase, 0, len(cases))
	for _, tc := range cases {
		if caseID != "" && tc.CaseID != caseID {
			continue
		}
		filtered = append(filtered, tc)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func parseIntLoose(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func parseBenchmarkBoolLoose(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "t", "yes", "y":
		return true
	default:
		return false
	}
}

func parseOptionalBenchmarkBool(header map[string]int, record []string, name string) (*bool, error) {
	if _, ok := header[name]; !ok {
		return nil, nil
	}
	raw := csvCell(record, header, name)
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "1", "true", "t", "yes", "y":
		resolved := true
		return &resolved, nil
	case "0", "false", "f", "no", "n":
		resolved := false
		return &resolved, nil
	default:
		return nil, fmt.Errorf("invalid boolean value %q", raw)
	}
}

func buildBenchmarkUserPrompt(tc benchmarkCase) string {
	var b strings.Builder
	availableAssumptions := append([]string{}, tc.Assumptions...)
	availableAssumptions = append(availableAssumptions, tc.ImportedLemmas...)
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
	if len(tc.ImportedLemmas) > 0 {
		b.WriteString("\nImported lemmas:\n")
		for idx, lemma := range tc.ImportedLemmas {
			_, _ = fmt.Fprintf(&b, "%d. %s\n", idx+1, lemma)
		}
		b.WriteString("\nAvailable assumptions for certificate construction:\n")
		for idx, assumption := range availableAssumptions {
			_, _ = fmt.Fprintf(&b, "%d. %s\n", idx+1, assumption)
		}
	}
	b.WriteString("\nGoal:\n")
	b.WriteString(tc.Goal)
	b.WriteString("\n\nReturn only:\n- a valid certificate JSON for this goal from these assumptions; or\n- NOT_DERIVABLE\n")
	return b.String()
}

func validateBenchmarkSelectedCaseDependencies(cases []benchmarkCase) error {
	sharedCases := make([]benchmarkcases.ChainCase, 0, len(cases))
	for idx, tc := range cases {
		stageOrder := tc.StageOrder
		if stageOrder <= 0 {
			stageOrder = idx + 1
		}
		sharedCases = append(sharedCases, benchmarkcases.ChainCase{
			CaseID:         tc.CaseID,
			ChainID:        tc.ChainID,
			ChainProtocol:  tc.ChainProtocol,
			StageID:        tc.StageID,
			StageOrder:     stageOrder,
			StageRole:      tc.StageRole,
			ProvenanceMode: tc.ProvenanceMode,
			ImportStageIDs: tc.ImportStageIDs,
		})
	}
	return benchmarkcases.ValidateCompositionalDependencies(sharedCases)
}

func orderBenchmarkCasesForExecution(cases []benchmarkCase) []benchmarkCase {
	if len(cases) <= 1 {
		return cases
	}
	ordered := append([]benchmarkCase(nil), cases...)
	slices.SortStableFunc(ordered, func(left, right benchmarkCase) int {
		leftChain := strings.TrimSpace(left.ChainID)
		rightChain := strings.TrimSpace(right.ChainID)
		switch {
		case leftChain == "" && rightChain != "":
			return -1
		case leftChain != "" && rightChain == "":
			return 1
		case leftChain != rightChain:
			return strings.Compare(leftChain, rightChain)
		}
		leftOrder := left.StageOrder
		rightOrder := right.StageOrder
		switch {
		case leftOrder <= 0 && rightOrder > 0:
			return 1
		case leftOrder > 0 && rightOrder <= 0:
			return -1
		case leftOrder != rightOrder:
			return leftOrder - rightOrder
		}
		return strings.Compare(strings.TrimSpace(left.CaseID), strings.TrimSpace(right.CaseID))
	})
	return ordered
}

func resolveBenchmarkPromptCase(tc benchmarkCase, priorArtifacts []benchmarkTrustedCertificateObject) (benchmarkPromptCaseResolution, error) {
	resolved := benchmarkPromptCaseResolution{ResolvedCase: tc}
	mode := strings.ToLower(strings.TrimSpace(tc.ProvenanceMode))
	if mode != "model" && mode != "semi_gold" {
		return resolved, nil
	}
	artifacts, err := benchmarkResolvedModelImportedArtifacts(tc, priorArtifacts)
	if err != nil {
		return benchmarkPromptCaseResolution{}, err
	}
	resolved.ResolvedArtifacts = artifacts
	switch mode {
	case "semi_gold":
		resolved.ResolvedCase.ImportedLemmas = benchmarkMergeImportedLemmas(tc.ImportedLemmas, benchmarkImportedArtifactGoals(resolved.ResolvedArtifacts))
	default:
		resolved.ResolvedCase.ImportedLemmas = benchmarkImportedArtifactGoals(resolved.ResolvedArtifacts)
	}
	return resolved, nil
}

func benchmarkResolvedModelImportedArtifacts(tc benchmarkCase, priorArtifacts []benchmarkTrustedCertificateObject) ([]benchmarkImportedArtifactRef, error) {
	chainID := strings.TrimSpace(tc.ChainID)
	if chainID == "" || len(priorArtifacts) == 0 {
		if benchmarkCaseUsesCompositionalChainProtocol(tc) {
			requested := benchmarkTrimmedStageIDs(tc.ImportStageIDs)
			if len(requested) > 0 {
				return nil, benchmarkUnmaterializedImportError(tc, requested)
			}
		}
		return nil, nil
	}
	requestedStageIDs := benchmarkTrimmedStageIDs(tc.ImportStageIDs)
	if benchmarkCaseUsesCompositionalChainProtocol(tc) && len(requestedStageIDs) == 0 {
		return nil, fmt.Errorf("missing_import_stage: chain_id=%s case_id=%s provenance_mode=%s requires explicit import_stage_ids_json", strings.TrimSpace(tc.ChainID), strings.TrimSpace(tc.CaseID), strings.ToLower(strings.TrimSpace(tc.ProvenanceMode)))
	}
	requestedStageIndex := make(map[string]struct{}, len(requestedStageIDs))
	for _, stageID := range requestedStageIDs {
		requestedStageIndex[strings.ToLower(stageID)] = struct{}{}
	}
	refs := make([]benchmarkImportedArtifactRef, 0, 4)
	seenGoals := make(map[string]struct{}, 4)
	resolvedStageIDs := make(map[string]struct{}, len(requestedStageIDs))
	for _, artifact := range priorArtifacts {
		if !artifact.TrustedForReuse {
			continue
		}
		if strings.TrimSpace(artifact.ChainID) != chainID {
			continue
		}
		stageID := strings.ToLower(strings.TrimSpace(artifact.StageID))
		if len(requestedStageIndex) > 0 {
			if _, ok := requestedStageIndex[stageID]; !ok {
				continue
			}
		}
		formula := strings.TrimSpace(artifact.Goal)
		if formula == "" {
			continue
		}
		if _, ok := seenGoals[formula]; ok {
			continue
		}
		seenGoals[formula] = struct{}{}
		resolvedStageIDs[stageID] = struct{}{}
		refs = append(refs, benchmarkImportedArtifactRef{
			RunID:                 artifact.RunID,
			CaseID:                artifact.CaseID,
			ChainID:               artifact.ChainID,
			StageID:               artifact.StageID,
			Goal:                  formula,
			CertificateObjectPath: artifact.CertificateObjectPath,
			TrustedForReuse:       artifact.TrustedForReuse,
		})
	}
	missingStageIDs := make([]string, 0, len(requestedStageIDs))
	for _, stageID := range requestedStageIDs {
		if _, ok := resolvedStageIDs[strings.ToLower(stageID)]; !ok {
			missingStageIDs = append(missingStageIDs, stageID)
		}
	}
	if len(missingStageIDs) > 0 {
		return nil, benchmarkUnmaterializedImportError(tc, missingStageIDs)
	}
	return refs, nil
}

func benchmarkUnmaterializedImportError(tc benchmarkCase, missingStageIDs []string) error {
	return fmt.Errorf("unmaterialized_import_artifact: chain_id=%s case_id=%s stage_id=%s requested prerequisite stage(s) did not materialize trusted artifacts before prompt construction", strings.TrimSpace(tc.ChainID), strings.TrimSpace(tc.CaseID), strings.Join(benchmarkTrimmedStageIDs(missingStageIDs), ","))
}

func benchmarkImportedArtifactGoals(refs []benchmarkImportedArtifactRef) []string {
	if len(refs) == 0 {
		return nil
	}
	goals := make([]string, 0, len(refs))
	for _, ref := range refs {
		goal := strings.TrimSpace(ref.Goal)
		if goal == "" {
			continue
		}
		goals = append(goals, goal)
	}
	return goals
}

func benchmarkMergeImportedLemmas(staticImports, modelImports []string) []string {
	if len(staticImports) == 0 && len(modelImports) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(staticImports)+len(modelImports))
	merged := make([]string, 0, len(staticImports)+len(modelImports))
	for _, lemma := range staticImports {
		lemma = strings.TrimSpace(lemma)
		if lemma == "" {
			continue
		}
		if _, ok := seen[lemma]; ok {
			continue
		}
		seen[lemma] = struct{}{}
		merged = append(merged, lemma)
	}
	for _, lemma := range modelImports {
		lemma = strings.TrimSpace(lemma)
		if lemma == "" {
			continue
		}
		if _, ok := seen[lemma]; ok {
			continue
		}
		seen[lemma] = struct{}{}
		merged = append(merged, lemma)
	}
	return merged
}

func benchmarkTrustedReuseEligibleStage(stageID string) bool {
	switch strings.ToLower(strings.TrimSpace(stageID)) {
	case "s1", "s2", "m1", "m2":
		return true
	default:
		return false
	}
}

func benchmarkCaseTrustedReuseEligible(tc benchmarkCase) bool {
	if tc.TrustedReuse != nil {
		return *tc.TrustedReuse
	}
	return benchmarkTrustedReuseEligibleStage(tc.StageID)
}

func benchmarkResolvedImportNote(source benchmarkCase, resolved benchmarkPromptCaseResolution) string {
	switch strings.ToLower(strings.TrimSpace(source.ProvenanceMode)) {
	case "semi_gold":
		return fmt.Sprintf("static_gold_imports=%d; requested_model_imports=%d; effective_imports=%d; resolved_model_import_artifacts=%d", len(source.ImportedLemmas), len(source.ImportStageIDs), len(resolved.ResolvedCase.ImportedLemmas), len(resolved.ResolvedArtifacts))
	case "model":
		return fmt.Sprintf("requested_model_imports=%d; resolved_model_imports=%d; resolved_import_artifacts=%d", len(source.ImportStageIDs), len(resolved.ResolvedCase.ImportedLemmas), len(resolved.ResolvedArtifacts))
	default:
		return ""
	}
}

func appendBenchmarkNote(existing, addition string) string {
	existing = strings.TrimSpace(existing)
	addition = strings.TrimSpace(addition)
	switch {
	case existing == "":
		return addition
	case addition == "":
		return existing
	default:
		return existing + "; " + addition
	}
}

func classifyBenchmarkRawOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "NOT_DERIVABLE":
		return "not_derivable"
	case strings.HasPrefix(trimmed, "{"):
		if json.Valid([]byte(trimmed)) {
			return "certificate"
		}
		return "invalid_json"
	default:
		return "other"
	}
}

func populateBenchmarkContractStatus(row *benchmarkResultRow, rawOutput string) error {
	if row == nil {
		return fmt.Errorf("benchmark result row is nil")
	}
	cert, err := certificates.DecodeJSON([]byte(strings.TrimSpace(rawOutput)))
	if err != nil {
		return fmt.Errorf("decode accepted certificate for contract validation: %w", err)
	}
	if err := validateBenchmarkCertificateContract(cert); err != nil {
		row.ContractStatus = "fail"
		row.ScoreBucket = "contract_failure"
		row.Notes = appendBenchmarkNote(row.Notes, sanitizeNote(err.Error()))
		return nil
	}
	row.ContractStatus = "pass"
	return nil
}

func validateBenchmarkCertificateContract(cert *certificates.Certificate) error {
	if cert == nil {
		return fmt.Errorf("benchmark contract certificate is nil")
	}
	if strings.TrimSpace(cert.CertificateVersion) != certificates.FormatVersionV1 {
		return fmt.Errorf("benchmark contract requires certificate_version=%q, got %q", certificates.FormatVersionV1, strings.TrimSpace(cert.CertificateVersion))
	}
	if strings.TrimSpace(cert.Context.Domain) != "hilbert-benchmark-v1" {
		return fmt.Errorf("benchmark contract requires context.domain=%q, got %q", "hilbert-benchmark-v1", strings.TrimSpace(cert.Context.Domain))
	}
	if strings.TrimSpace(cert.Context.Generator) != "llm-benchmark-v1" {
		return fmt.Errorf("benchmark contract requires context.generator=%q, got %q", "llm-benchmark-v1", strings.TrimSpace(cert.Context.Generator))
	}
	if strings.TrimSpace(cert.Context.RulePack) != hilbert.RulePackID {
		return fmt.Errorf("benchmark contract requires context.rule_pack=%q, got %q", hilbert.RulePackID, strings.TrimSpace(cert.Context.RulePack))
	}
	if strings.TrimSpace(cert.Context.Syntax) != hilbert.SyntaxID {
		return fmt.Errorf("benchmark contract requires context.syntax=%q, got %q", hilbert.SyntaxID, strings.TrimSpace(cert.Context.Syntax))
	}
	return nil
}

func populateVerificationStatuses(row *benchmarkResultRow, tc benchmarkCase, result *benchmarkVerifyResult) error {
	if row == nil {
		return fmt.Errorf("benchmark result row is nil")
	}
	if result == nil {
		return fmt.Errorf("verification result is nil")
	}

	row.CLIExitCode = strconv.Itoa(result.ExitCode)
	switch result.ExitCode {
	case 0:
		row.SchemaStatus = "pass"
		row.ParseStatus = "pass"
		row.KernelStatus = "accept"
		if tc.Label == "entailed" {
			row.ScoreBucket = "pass"
		} else {
			row.ScoreBucket = "false_accept"
		}
	case 1:
		if err := assertVerifierPrefix(result.Stderr, "schema_error:"); err != nil {
			return err
		}
		row.SchemaStatus = "fail"
		row.ScoreBucket = "schema_failure"
	case 2:
		if err := assertVerifierPrefix(result.Stderr, "parse_error:"); err != nil {
			return err
		}
		row.SchemaStatus = "pass"
		row.ParseStatus = "fail"
		row.ScoreBucket = "parse_failure"
	case 3:
		if err := assertVerifierPrefix(result.Stderr, "kernel_validation_error:"); err != nil {
			return err
		}
		row.SchemaStatus = "pass"
		row.ParseStatus = "pass"
		row.KernelStatus = "reject"
		row.ScoreBucket = "kernel_failure"
	case 4:
		if err := assertVerifierPrefix(result.Stderr, "internal_error:"); err != nil {
			return err
		}
		return fmt.Errorf("hilbertcheck returned internal_error for case %s", tc.CaseID)
	case 64:
		return fmt.Errorf("hilbertcheck returned usage error for case %s", tc.CaseID)
	default:
		return fmt.Errorf("hilbertcheck returned unexpected exit code %d for case %s", result.ExitCode, tc.CaseID)
	}

	row.Notes = sanitizeNote(strings.TrimSpace(result.Stderr))
	return nil
}
