package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

func benchmarkProjectSupportsObservedSlices(projectFolder string) bool {
	switch strings.TrimSpace(projectFolder) {
	case benchmarkV2HeldOutProjectFolder, benchmarkNDProjectFolder:
		return true
	default:
		return false
	}
}

func benchmarkObservedHardAuditCaseIDs(projectFolder string) []string {
	switch strings.TrimSpace(projectFolder) {
	case benchmarkV2HeldOutProjectFolder:
		return benchmarkV2HeldOutHardAuditCaseIDs
	default:
		return nil
	}
}

func resolveBenchmarkLocalPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(trimmed, "/mnt/") {
		parts := strings.SplitN(strings.TrimPrefix(trimmed, "/mnt/"), "/", 2)
		if len(parts) >= 1 && len(parts[0]) == 1 {
			driveLetter := strings.ToUpper(parts[0])
			suffix := ""
			if len(parts) == 2 {
				suffix = strings.ReplaceAll(parts[1], "/", `\`)
			}
			if suffix == "" {
				return driveLetter + `:\`
			}
			return driveLetter + `:\` + suffix
		}
	}
	return filepath.Clean(trimmed)
}

func benchmarkPassRate(passCount, casesTotal int) float64 {
	if casesTotal <= 0 {
		return 0
	}
	return 100 * float64(passCount) / float64(casesTotal)
}

func pluralSuffix(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func valueOrNA(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return value
}

func displayBenchmarkTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "n/a"
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return timestamp.UTC().Format("2006-01-02 15:04:05 UTC")
}

func benchmarkCollectDistinct[T any](items []T, selector func(T) string) []string {
	seen := make(map[string]struct{}, len(items))
	values := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(selector(item))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}

func benchmarkLatestTimestampFrom[T any](items []T, selector func(T) string) string {
	var latest time.Time
	latestRaw := ""
	for _, item := range items {
		raw := strings.TrimSpace(selector(item))
		if raw == "" {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			if latestRaw == "" {
				latestRaw = raw
			}
			continue
		}
		if latestRaw == "" || timestamp.After(latest) {
			latest = timestamp
			latestRaw = raw
		}
	}
	return latestRaw
}

func benchmarkSingleDistinctValue[T any](items []T, selector func(T) string) (string, bool) {
	values := benchmarkCollectDistinct(items, selector)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func renderBenchmarkSectionContext(b *strings.Builder, promptVersions, certificateVersions []string, latestTimestampUTC string) {
	renderBenchmarkSectionContextWithSurface(b, promptVersions, certificateVersions, nil, latestTimestampUTC)
}

func renderBenchmarkSectionContextWithSurface(b *strings.Builder, promptVersions, certificateVersions, surfaces []string, latestTimestampUTC string) {
	wrote := false
	if len(promptVersions) == 1 {
		b.WriteString(fmt.Sprintf("- Prompt version: `%s`\n", promptVersions[0]))
		wrote = true
	} else if len(promptVersions) > 1 {
		b.WriteString(fmt.Sprintf("- Prompt versions: `%s`\n", strings.Join(promptVersions, "`, `")))
		wrote = true
	}
	if len(certificateVersions) == 1 {
		b.WriteString(fmt.Sprintf("- Certificate version: `%s`\n", certificateVersions[0]))
		wrote = true
	} else if len(certificateVersions) > 1 {
		b.WriteString(fmt.Sprintf("- Certificate versions: `%s`\n", strings.Join(certificateVersions, "`, `")))
		wrote = true
	}
	if len(surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n", surfaces[0]))
		wrote = true
	} else if len(surfaces) > 1 {
		b.WriteString(fmt.Sprintf("- Surfaces: `%s`\n", strings.Join(surfaces, "`, `")))
		wrote = true
	}
	if strings.TrimSpace(latestTimestampUTC) != "" {
		b.WriteString(fmt.Sprintf("- Latest run (UTC): `%s`\n", displayBenchmarkTimestamp(latestTimestampUTC)))
		wrote = true
	}
	if wrote {
		b.WriteString("\n")
	}
}

func benchmarkReportModelLabel(displayName, llmModel, model string) string {
	for _, value := range []string{displayName, llmModel, model} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "n/a"
}

func parseBenchmarkSummaryInt(runID, field, value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("benchmark summary row %q field %q must be an integer, got %q: %w", runID, field, value, err)
	}
	return parsed, nil
}

func parseBenchmarkSummaryInt64(runID, field, value string) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("benchmark summary row %q field %q must be an integer, got %q: %w", runID, field, value, err)
	}
	return parsed, true, nil
}

func parseBenchmarkSummaryFloat(runID, field, value string) (float64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false, fmt.Errorf("benchmark summary row %q field %q must be a float, got %q: %w", runID, field, value, err)
	}
	return parsed, true, nil
}

func parseBenchmarkResultFloat(runID, caseID, field, value string) (float64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false, fmt.Errorf("benchmark result row %q/%q field %q must be a float, got %q: %w", runID, caseID, field, value, err)
	}
	return parsed, true, nil
}

func readCSV(path string) ([][]string, map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open csv %q: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("read csv %q: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("csv %q must contain a header", path)
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	return records[1:], header, nil
}

func defaultBenchmarkResultsPath(benchmarkRoot, runID string) string {
	return defaultBenchmarkResultsPathForArtifact(benchmarkRoot, runID, "")
}

func defaultBenchmarkResultsPathForArtifact(benchmarkRoot, runID, artifactKey string) string {
	return filepath.Join(benchmarkRoot, "result", "result_"+benchmarkArtifactRunKey(runID, artifactKey)+".csv")
}

func benchmarkArtifactsRoot(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, "research", "artifacts")
}

func benchmarkProjectRoot(workspaceRoot, projectFolder string) string {
	return filepath.Join(benchmarkArtifactsRoot(workspaceRoot), strings.TrimSpace(projectFolder))
}

func defaultBenchmarkInputFile(projectFolder string) string {
	switch strings.TrimSpace(projectFolder) {
	case benchmarkNDProjectFolder:
		return benchmarkNDTheoremPack
	default:
		return benchmarkLegacyDefaultCases
	}
}

func canonicalBenchmarkInputFile(projectFolder, inputFile string) string {
	projectFolder = strings.TrimSpace(projectFolder)
	inputFile = normalizeBenchmarkPath(strings.TrimSpace(inputFile))
	switch projectFolder {
	case benchmarkNDProjectFolder:
		switch inputFile {
		case "", benchmarkLegacyDefaultCases:
			return benchmarkNDTheoremPack
		}
	}
	return inputFile
}

func defaultBenchmarkModelCatalogPath(benchmarkRoot string) string {
	return filepath.Join(benchmarkRoot, "model-catalog.csv")
}

func benchmarkProjectFolderFromRoot(benchmarkRoot string) string {
	return strings.TrimSpace(filepath.Base(filepath.Clean(benchmarkRoot)))
}

func benchmarkProjectUsesTheoremSummaryV2(projectFolder string) bool {
	switch strings.TrimSpace(projectFolder) {
	case benchmarkNDProjectFolder:
		return true
	default:
		return false
	}
}

func benchmarkInputFileFieldName(projectFolder string) string {
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		return "theorems_file"
	}
	return "cases_file"
}

func benchmarkInputPathFieldName(projectFolder string) string {
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		return "theorems_path"
	}
	return "cases_path"
}

func benchmarkInputCountLabel(projectFolder string) string {
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		return "theorems"
	}
	return "cases"
}

func benchmarkInputFilterLabel(projectFolder string) string {
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		return "Theorems file"
	}
	return "Cases file"
}

func defaultBenchmarkLegacySummaryPath(benchmarkRoot string) string {
	return filepath.Join(benchmarkRoot, "result", "result_summary.csv")
}

func defaultBenchmarkSummaryPath(benchmarkRoot string) string {
	if benchmarkProjectUsesTheoremSummaryV2(benchmarkProjectFolderFromRoot(benchmarkRoot)) {
		return filepath.Join(benchmarkRoot, "result", "result_summary_v2.csv")
	}
	return defaultBenchmarkLegacySummaryPath(benchmarkRoot)
}

func defaultBenchmarkSummaryPaths(benchmarkRoot string) []string {
	primary := defaultBenchmarkSummaryPath(benchmarkRoot)
	if !benchmarkProjectUsesTheoremSummaryV2(benchmarkProjectFolderFromRoot(benchmarkRoot)) {
		return []string{primary}
	}
	legacy := defaultBenchmarkLegacySummaryPath(benchmarkRoot)
	if filepath.Clean(legacy) == filepath.Clean(primary) {
		return []string{primary}
	}
	return []string{primary, legacy}
}

func parseBenchmarkSummaryArgPaths(workspaceRoot, raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	segments := []string{raw}
	if strings.Contains(raw, ",") {
		segments = strings.Split(raw, ",")
	} else if runtime.GOOS == "windows" && strings.Contains(raw, ";") {
		segments = strings.Split(raw, ";")
	}

	paths := make([]string, 0, len(segments))
	seen := make(map[string]struct{}, len(segments))
	for _, segment := range segments {
		resolved := resolveBenchmarkArgPath(workspaceRoot, strings.TrimSpace(segment))
		if resolved == "" {
			continue
		}
		key := filepath.Clean(resolved)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, resolved)
	}
	return paths
}

func defaultBenchmarkRawDir(benchmarkRoot string) string {
	return filepath.Join(benchmarkRoot, "raw")
}

func benchmarkArtifactRunKey(runID, artifactKey string) string {
	if key := strings.TrimSpace(artifactKey); key != "" {
		return key
	}
	return strings.TrimSpace(runID)
}

func benchmarkRunSummaryHeaderForPath(path, projectFolder string) ([]string, bool, error) {
	canonical := benchmarkRunSummaryHeader()
	base := filepath.Base(filepath.Clean(path))
	if strings.EqualFold(base, "result_summary_v2.csv") {
		canonical = benchmarkRunSummaryHeaderV2()
	}
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) && !strings.EqualFold(base, "result_summary.csv") {
		canonical = benchmarkRunSummaryHeaderV2()
	}
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 {
		file, err := os.Open(path)
		if err != nil {
			return nil, false, fmt.Errorf("open benchmark summary header: %w", err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		header, err := reader.Read()
		if err != nil {
			return nil, false, fmt.Errorf("read benchmark summary header: %w", err)
		}
		if benchmarkHeaderHasColumns(header, canonical) {
			return header, false, nil
		}
		return canonical, true, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("stat benchmark summary: %w", err)
	}
	return canonical, true, nil
}

func serializeBenchmarkRunSummaryRow(row benchmarkRunSummaryRow, header []string) []string {
	theoremsFile := strings.TrimSpace(row.TheoremsFile)
	theoremsPath := strings.TrimSpace(row.TheoremsPath)
	theoremPackID := strings.TrimSpace(row.TheoremPackID)
	if benchmarkProjectUsesTheoremSummaryV2(row.ProjectFolder) {
		if theoremsFile == "" {
			theoremsFile = row.CasesFile
		}
		if theoremsPath == "" {
			theoremsPath = row.CasesPath
		}
		if theoremPackID == "" {
			theoremPackID = deriveBenchmarkTheoremPackID(firstNonEmptyBenchmarkCell(theoremsPath, theoremsFile))
		}
	}
	values := map[string]string{
		"run_id":                     row.RunID,
		"timestamp_utc":              row.TimestampUTC,
		"provider":                   row.Provider,
		"model":                      row.Model,
		"llm_model":                  row.LLMModel,
		"experiment_name":            row.ExperimentName,
		"experiment_id":              row.ExperimentID,
		"sampling_surface":           row.SamplingSurface,
		"requested_sampling_profile": row.RequestedSamplingProfile,
		"effective_sampling_profile": row.EffectiveSamplingProfile,
		"requested_sampling_json":    row.RequestedSamplingJSON,
		"effective_sampling_json":    row.EffectiveSamplingJSON,
		"unsupported_sampling_json":  row.UnsupportedSamplingJSON,
		"google_thinking_mode":       row.GoogleThinkingMode,
		"prompt_version":             row.PromptVersion,
		"certificate_version":        row.CertificateVersion,
		"project_folder":             row.ProjectFolder,
		"surface":                    row.Surface,
		"theorem_pack_id":            theoremPackID,
		"cases_file":                 row.CasesFile,
		"theorems_file":              theoremsFile,
		"case_selector":              row.CaseSelector,
		"hypothesis":                 row.Hypothesis,
		"cases_path":                 normalizeBenchmarkPath(row.CasesPath),
		"theorems_path":              normalizeBenchmarkPath(theoremsPath),
		"result_file":                normalizeBenchmarkPath(row.ResultsPath),
		"chain_result_file":          normalizeBenchmarkPath(row.ChainSummaryPath),
		"import_catalog_file":        normalizeBenchmarkPath(row.ImportCatalogPath),
		"raw_dir":                    normalizeBenchmarkPath(row.RawDir),
		"cases_total":                row.CasesTotal,
		"pass_count":                 row.PassCount,
		"false_refusal_count":        row.FalseRefusalCount,
		"false_accept_count":         row.FalseAcceptCount,
		"request_failure_count":      row.RequestFailureCount,
		"schema_failure_count":       row.SchemaFailureCount,
		"parse_failure_count":        row.ParseFailureCount,
		"kernel_failure_count":       row.KernelFailureCount,
		"contract_failure_count":     row.ContractFailureCount,
		"format_failure_count":       row.FormatFailureCount,
		"avg_latency_ms":             row.AvgLatencyMS,
		"max_latency_ms":             row.MaxLatencyMS,
		"run_elapsed_seconds":        row.RunElapsedSeconds,
		"category_count":             row.CategoryCount,
	}

	record := make([]string, 0, len(header))
	for _, name := range header {
		record = append(record, values[name])
	}
	return record
}

func csvCell(record []string, header map[string]int, name string) string {
	idx, ok := header[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func isCSVRecordEmpty(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func sanitizeNote(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

func benchmarkHeaderHasColumns(header, canonical []string) bool {
	index := make(map[string]struct{}, len(header))
	for _, name := range header {
		index[strings.TrimSpace(name)] = struct{}{}
	}
	for _, name := range canonical {
		if _, ok := index[strings.TrimSpace(name)]; !ok {
			return false
		}
	}
	return true
}

func firstNonEmptyBenchmarkCell(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func deriveBenchmarkTheoremPackID(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(base), "theorems_") {
		base = strings.TrimPrefix(strings.ToLower(base), "theorems_")
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func benchmarkTheoremPackID(cases []benchmarkCase, casesPath string) string {
	for _, tc := range cases {
		if value := strings.TrimSpace(tc.TheoremPackID); value != "" {
			return value
		}
	}
	return deriveBenchmarkTheoremPackID(casesPath)
}

func benchmarkSamplingResolution(opts benchmarkRunOptions) researchsampling.Resolution {
	requested := researchsampling.NewParams(opts.RequestedTemperature, opts.RequestedSeed, opts.RequestedTopP)
	return researchsampling.Resolve(opts.Provider, requested)
}

func benchmarkSamplingTemperature(params *researchsampling.Params) *float64 {
	if params == nil {
		return nil
	}
	return params.Temperature
}

func benchmarkSamplingSeed(params *researchsampling.Params) *int64 {
	if params == nil {
		return nil
	}
	return params.Seed
}

func benchmarkSamplingTopP(params *researchsampling.Params) *float64 {
	if params == nil {
		return nil
	}
	return params.TopP
}

func benchmarkSamplingTemperatureOrDefault(value *float64, fallback float64) *float64 {
	if value != nil {
		return value
	}
	resolved := fallback
	return &resolved
}

func benchmarkDisplayNameWithExperiment(base, experimentID string) string {
	base = strings.TrimSpace(base)
	experimentID = strings.TrimSpace(experimentID)
	switch {
	case base == "" && experimentID == "":
		return ""
	case experimentID == "":
		return base
	case base == "":
		return experimentID
	default:
		return base + " [exp: " + experimentID + "]"
	}
}

func benchmarkComparisonKeyLabel(row benchmarkRunSummaryRow) string {
	base := strings.TrimSpace(row.LLMModel)
	if base == "" {
		base = strings.TrimSpace(row.Model)
	}
	return benchmarkDisplayNameWithExperiment(base, row.ExperimentID)
}

func parseOptionalBenchmarkFloat64(raw, label string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("%s must be a float: %w", label, err)
	}
	return &value, nil
}

func parseOptionalBenchmarkInt64(raw, label string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s must be an integer: %w", label, err)
	}
	return &value, nil
}

func formatBenchmarkFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func inferBenchmarkProjectFolder(casesPath, workspaceRoot, fallback string) string {
	artifactsRoot := benchmarkArtifactsRoot(workspaceRoot)
	rel, err := filepath.Rel(artifactsRoot, filepath.Clean(casesPath))
	if err == nil && rel != "." && rel != "" && !strings.HasPrefix(rel, "..") {
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return parts[0]
		}
	}
	return strings.TrimSpace(fallback)
}

func inferBenchmarkCasesFile(casesPath, workspaceRoot, projectFolder, fallback string) string {
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, projectFolder)
	rel, err := filepath.Rel(benchmarkRoot, filepath.Clean(casesPath))
	if err == nil && rel != "." && rel != "" && !strings.HasPrefix(rel, "..") {
		return canonicalBenchmarkInputFile(projectFolder, rel)
	}
	if strings.TrimSpace(fallback) != "" {
		return canonicalBenchmarkInputFile(projectFolder, fallback)
	}
	return canonicalBenchmarkInputFile(projectFolder, casesPath)
}

func deriveBenchmarkCaseSelector(caseID string, limit int) string {
	caseID = strings.TrimSpace(caseID)
	switch {
	case caseID == "" && limit <= 0:
		return "all"
	case caseID != "" && limit > 0:
		return fmt.Sprintf("case_id=%s;limit=%d", caseID, limit)
	case caseID != "":
		return fmt.Sprintf("case_id=%s", caseID)
	default:
		return fmt.Sprintf("limit=%d", limit)
	}
}

func normalizeBenchmarkPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.ToSlash(path)
}

func displayBenchmarkPath(path string) string {
	clean := normalizeBenchmarkPath(resolveBenchmarkLocalPath(path))
	if strings.TrimSpace(clean) == "" {
		return ""
	}
	workspaceRoot, err := findWorkspaceRoot()
	if err == nil && strings.TrimSpace(workspaceRoot) != "" {
		rel, relErr := filepath.Rel(workspaceRoot, filepath.FromSlash(clean))
		if relErr == nil && rel != "." && rel != "" && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return clean
}

func resolveBenchmarkArgPath(workspaceRoot, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(workspaceRoot, filepath.FromSlash(trimmed))
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func asExitError(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*target = exitErr
	return true
}
