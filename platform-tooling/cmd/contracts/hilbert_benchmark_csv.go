package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func appendBenchmarkResults(path string, rows []benchmarkResultRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create benchmark results dir: %w", err)
	}

	header := benchmarkLegacyResultsHeader()
	for _, row := range rows {
		if strings.TrimSpace(row.Category) != "" {
			header = benchmarkCategoryResultsHeader()
			break
		}
	}
	rows = dedupeBenchmarkResultRows(rows)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open benchmark results: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write benchmark results header: %w", err)
	}

	for _, row := range rows {
		if err := writer.Write(serializeBenchmarkResultRow(row, header)); err != nil {
			return fmt.Errorf("write benchmark results row for case %s: %w", row.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush benchmark results csv: %w", err)
	}
	return nil
}

func benchmarkResultDedupKey(row benchmarkResultRow) string {
	return strings.Join([]string{
		strings.TrimSpace(row.RunID),
		strings.TrimSpace(row.Provider),
		strings.TrimSpace(row.Model),
		strings.TrimSpace(row.LLMModel),
		strings.TrimSpace(row.ExperimentID),
		strings.TrimSpace(row.RequestedSamplingProfile),
		strings.TrimSpace(row.EffectiveSamplingProfile),
		strings.TrimSpace(row.PromptVersion),
		strings.TrimSpace(row.PromptHash),
		strings.TrimSpace(row.Surface),
		strings.TrimSpace(row.TheoremPackID),
		strings.TrimSpace(firstNonEmptyBenchmarkCell(row.TheoremID, row.CaseID)),
		strings.TrimSpace(row.CaseID),
		strings.TrimSpace(row.Category),
		strings.TrimSpace(row.ExpectedLabel),
	}, "\x00")
}

func dedupeBenchmarkResultRows(rows []benchmarkResultRow) []benchmarkResultRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]benchmarkResultRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := benchmarkResultDedupKey(row)
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func assertVerifierPrefix(stderr, prefix string) error {
	if !strings.HasPrefix(strings.TrimSpace(stderr), prefix) {
		return fmt.Errorf("hilbertcheck stderr %q does not start with %q", strings.TrimSpace(stderr), prefix)
	}
	return nil
}

func orderedBenchmarkBuckets() []string {
	return []string{
		"pass",
		"false_refusal",
		"false_accept",
		"request_failure",
		"schema_failure",
		"parse_failure",
		"kernel_failure",
		"contract_failure",
		"format_failure",
	}
}

func orderedCategorySummaryBuckets() []string {
	return []string{
		"pass",
		"false_refusal",
		"false_accept",
		"request_failure",
		"schema_failure",
		"parse_failure",
		"kernel_failure",
		"contract_failure",
		"format_failure",
	}
}

func orderedCategoryNames(stats map[string]map[string]int) []string {
	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func benchmarkLegacyResultsHeader() []string {
	return []string{
		"run_id",
		"timestamp_utc",
		"provider",
		"model",
		"llm_model",
		"experiment_name",
		"experiment_id",
		"sampling_surface",
		"requested_sampling_profile",
		"effective_sampling_profile",
		"requested_sampling_json",
		"effective_sampling_json",
		"unsupported_sampling_json",
		"google_thinking_mode",
		"prompt_version",
		"prompt_hash",
		"surface",
		"nd_proof_status",
		"lowering_status",
		"pipeline_stage",
		"pipeline_detail",
		"case_id",
		"expected_label",
		"raw_output_kind",
		"schema_status",
		"parse_status",
		"kernel_status",
		"contract_status",
		"cli_exit_code",
		"score_bucket",
		"latency_ms",
		"notes",
	}
}

func benchmarkCategoryResultsHeader() []string {
	return []string{
		"run_id",
		"timestamp_utc",
		"provider",
		"model",
		"llm_model",
		"experiment_name",
		"experiment_id",
		"sampling_surface",
		"requested_sampling_profile",
		"effective_sampling_profile",
		"requested_sampling_json",
		"effective_sampling_json",
		"unsupported_sampling_json",
		"google_thinking_mode",
		"prompt_version",
		"prompt_hash",
		"surface",
		"nd_proof_status",
		"lowering_status",
		"pipeline_stage",
		"pipeline_detail",
		"theorem_pack_id",
		"theorem_id",
		"case_id",
		"chain_id",
		"stage_id",
		"category",
		"case_family",
		"chain_depth",
		"provenance_mode",
		"atom_renaming_id",
		"source_family",
		"target_family",
		"transition_group",
		"import_arity",
		"reuse_shape",
		"bridge_depth",
		"symbol_overlap",
		"lemma_surface_style",
		"negative_twin_hardness",
		"case_notes",
		"expected_label",
		"raw_output_kind",
		"schema_status",
		"parse_status",
		"kernel_status",
		"contract_status",
		"cli_exit_code",
		"score_bucket",
		"latency_ms",
		"output_length",
		"proof_steps_count",
		"requested_imports_json",
		"effective_imports_json",
		"resolved_import_refs_json",
		"certificate_object_file",
		"notes",
	}
}

func benchmarkRunSummaryHeader() []string {
	return []string{
		"run_id",
		"timestamp_utc",
		"provider",
		"model",
		"llm_model",
		"experiment_name",
		"experiment_id",
		"sampling_surface",
		"requested_sampling_profile",
		"effective_sampling_profile",
		"requested_sampling_json",
		"effective_sampling_json",
		"unsupported_sampling_json",
		"google_thinking_mode",
		"prompt_version",
		"certificate_version",
		"project_folder",
		"surface",
		"cases_file",
		"case_selector",
		"hypothesis",
		"cases_path",
		"result_file",
		"chain_result_file",
		"import_catalog_file",
		"raw_dir",
		"cases_total",
		"pass_count",
		"false_refusal_count",
		"false_accept_count",
		"request_failure_count",
		"schema_failure_count",
		"parse_failure_count",
		"kernel_failure_count",
		"contract_failure_count",
		"format_failure_count",
		"avg_latency_ms",
		"max_latency_ms",
		"run_elapsed_seconds",
		"category_count",
	}
}

func benchmarkRunSummaryHeaderV2() []string {
	return []string{
		"run_id",
		"timestamp_utc",
		"provider",
		"model",
		"llm_model",
		"experiment_name",
		"experiment_id",
		"sampling_surface",
		"requested_sampling_profile",
		"effective_sampling_profile",
		"requested_sampling_json",
		"effective_sampling_json",
		"unsupported_sampling_json",
		"google_thinking_mode",
		"prompt_version",
		"certificate_version",
		"project_folder",
		"surface",
		"theorem_pack_id",
		"theorems_file",
		"case_selector",
		"hypothesis",
		"theorems_path",
		"result_file",
		"chain_result_file",
		"import_catalog_file",
		"raw_dir",
		"cases_total",
		"pass_count",
		"false_refusal_count",
		"false_accept_count",
		"request_failure_count",
		"schema_failure_count",
		"parse_failure_count",
		"kernel_failure_count",
		"contract_failure_count",
		"format_failure_count",
		"avg_latency_ms",
		"max_latency_ms",
		"run_elapsed_seconds",
		"category_count",
	}
}

func benchmarkResultsHeader(path string, rows []benchmarkResultRow) ([]string, bool, error) {
	canonical := benchmarkLegacyResultsHeader()
	for _, row := range rows {
		if strings.TrimSpace(row.Category) != "" {
			canonical = benchmarkCategoryResultsHeader()
			break
		}
	}
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 {
		file, err := os.Open(path)
		if err != nil {
			return nil, false, fmt.Errorf("open benchmark results header: %w", err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		header, err := reader.Read()
		if err != nil {
			return nil, false, fmt.Errorf("read benchmark results header: %w", err)
		}
		if benchmarkHeaderHasColumns(header, canonical) {
			return header, false, nil
		}
		return canonical, true, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("stat benchmark results: %w", err)
	}
	return canonical, true, nil
}

func serializeBenchmarkResultRow(row benchmarkResultRow, header []string) []string {
	values := make(map[string]string, len(header))
	values["run_id"] = row.RunID
	values["timestamp_utc"] = row.TimestampUTC
	values["provider"] = row.Provider
	values["model"] = row.Model
	values["llm_model"] = row.LLMModel
	values["experiment_name"] = row.ExperimentName
	values["experiment_id"] = row.ExperimentID
	values["sampling_surface"] = row.SamplingSurface
	values["requested_sampling_profile"] = row.RequestedSamplingProfile
	values["effective_sampling_profile"] = row.EffectiveSamplingProfile
	values["requested_sampling_json"] = row.RequestedSamplingJSON
	values["effective_sampling_json"] = row.EffectiveSamplingJSON
	values["unsupported_sampling_json"] = row.UnsupportedSamplingJSON
	values["google_thinking_mode"] = row.GoogleThinkingMode
	values["prompt_version"] = row.PromptVersion
	values["prompt_hash"] = row.PromptHash
	values["surface"] = row.Surface
	values["nd_proof_status"] = row.NDProofStatus
	values["lowering_status"] = row.LoweringStatus
	values["pipeline_stage"] = row.PipelineStage
	values["pipeline_detail"] = row.PipelineDetail
	values["theorem_pack_id"] = row.TheoremPackID
	values["theorem_id"] = firstNonEmptyBenchmarkCell(row.TheoremID, row.CaseID)
	values["case_id"] = row.CaseID
	values["chain_id"] = row.ChainID
	values["stage_id"] = row.StageID
	values["category"] = row.Category
	values["case_family"] = row.CaseFamily
	values["chain_depth"] = row.ChainDepth
	values["provenance_mode"] = row.ProvenanceMode
	values["atom_renaming_id"] = row.AtomRenamingID
	values["source_family"] = row.SourceFamily
	values["target_family"] = row.TargetFamily
	values["transition_group"] = row.TransitionGroup
	values["import_arity"] = row.ImportArity
	values["reuse_shape"] = row.ReuseShape
	values["bridge_depth"] = row.BridgeDepth
	values["symbol_overlap"] = row.SymbolOverlap
	values["lemma_surface_style"] = row.LemmaSurfaceStyle
	values["negative_twin_hardness"] = row.NegativeTwinHardness
	values["case_notes"] = row.CaseNotes
	values["expected_label"] = row.ExpectedLabel
	values["raw_output_kind"] = row.RawOutputKind
	values["schema_status"] = row.SchemaStatus
	values["parse_status"] = row.ParseStatus
	values["kernel_status"] = row.KernelStatus
	values["contract_status"] = row.ContractStatus
	values["cli_exit_code"] = row.CLIExitCode
	values["score_bucket"] = row.ScoreBucket
	values["latency_ms"] = row.LatencyMS
	values["output_length"] = row.OutputLength
	values["proof_steps_count"] = row.ProofStepsCount
	values["requested_imports_json"] = row.RequestedImportsJSON
	values["effective_imports_json"] = row.EffectiveImportsJSON
	values["resolved_import_refs_json"] = row.ResolvedImportRefsJSON
	values["certificate_object_file"] = row.CertificateObjectPath
	values["notes"] = row.Notes

	record := make([]string, 0, len(header))
	for _, name := range header {
		record = append(record, values[name])
	}
	return record
}

func appendBenchmarkRunSummary(path string, row benchmarkRunSummaryRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create benchmark summary dir: %w", err)
	}
	releaseSummaryLock, err := acquireBenchmarkSummaryLock(path)
	if err != nil {
		return err
	}
	defer releaseSummaryLock()
	if strings.TrimSpace(row.Provider) == "" {
		row.Provider = deriveBenchmarkProvider(row.Model)
	}

	header, _, err := benchmarkRunSummaryHeaderForPath(path, row.ProjectFolder)
	if err != nil {
		return err
	}

	rows := make([]benchmarkRunSummaryRow, 0, 8)
	if info, statErr := os.Stat(path); statErr == nil && info.Size() > 0 {
		rows, err = loadBenchmarkRunSummaryRows(path)
		if err != nil {
			return err
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("stat benchmark summary: %w", statErr)
	}
	rows = dedupeBenchmarkRunSummaryRows(append(rows, row))

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open benchmark summary: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write benchmark summary header: %w", err)
	}
	for _, existing := range rows {
		if err := writer.Write(serializeBenchmarkRunSummaryRow(existing, header)); err != nil {
			return fmt.Errorf("write benchmark summary row for run %s: %w", existing.RunID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush benchmark summary csv: %w", err)
	}
	return nil
}

func acquireBenchmarkSummaryLock(path string) (func(), error) {
	lockPath := path + ".lock"
	deadline := time.Now().Add(2 * time.Minute)
	staleAfter := 10 * time.Minute
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintf(lockFile, "pid=%d\nacquired_at_utc=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
			_ = lockFile.Close()
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}
		if err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("acquire benchmark summary lock %s: %w", lockPath, err)
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > staleAfter {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout acquiring benchmark summary lock %s", lockPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func benchmarkRunSummaryDedupKey(row benchmarkRunSummaryRow) string {
	return strings.Join([]string{
		strings.TrimSpace(row.ProjectFolder),
		strings.TrimSpace(row.RunID),
		strings.TrimSpace(row.Provider),
		strings.TrimSpace(row.Model),
		strings.TrimSpace(row.LLMModel),
		strings.TrimSpace(row.ExperimentID),
		strings.TrimSpace(row.RequestedSamplingProfile),
		strings.TrimSpace(row.EffectiveSamplingProfile),
		strings.TrimSpace(row.PromptVersion),
		strings.TrimSpace(row.Surface),
		normalizeBenchmarkPath(strings.TrimSpace(row.ResultsPath)),
	}, "\x00")
}

func dedupeBenchmarkRunSummaryRows(rows []benchmarkRunSummaryRow) []benchmarkRunSummaryRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]benchmarkRunSummaryRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := benchmarkRunSummaryDedupKey(row)
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func loadBenchmarkRunSummaryRows(path string) ([]benchmarkRunSummaryRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open benchmark summary: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read benchmark summary csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("benchmark summary csv must contain a header")
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	required := []string{
		"run_id",
		"timestamp_utc",
		"prompt_version",
		"certificate_version",
		"cases_total",
		"pass_count",
		"false_refusal_count",
		"false_accept_count",
		"request_failure_count",
		"schema_failure_count",
		"parse_failure_count",
		"kernel_failure_count",
		"format_failure_count",
	}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, fmt.Errorf("benchmark summary csv is missing required column %q", name)
		}
	}

	rows := make([]benchmarkRunSummaryRow, 0, max(0, len(records)-1))
	for _, record := range records[1:] {
		if isCSVRecordEmpty(record) {
			continue
		}
		rows = append(rows, benchmarkRunSummaryRow{
			RunID:                    csvCell(record, header, "run_id"),
			TimestampUTC:             csvCell(record, header, "timestamp_utc"),
			Provider:                 csvCell(record, header, "provider"),
			Model:                    csvCell(record, header, "model"),
			LLMModel:                 csvCell(record, header, "llm_model"),
			ExperimentName:           csvCell(record, header, "experiment_name"),
			ExperimentID:             csvCell(record, header, "experiment_id"),
			SamplingSurface:          csvCell(record, header, "sampling_surface"),
			RequestedSamplingProfile: csvCell(record, header, "requested_sampling_profile"),
			EffectiveSamplingProfile: csvCell(record, header, "effective_sampling_profile"),
			RequestedSamplingJSON:    csvCell(record, header, "requested_sampling_json"),
			EffectiveSamplingJSON:    csvCell(record, header, "effective_sampling_json"),
			UnsupportedSamplingJSON:  csvCell(record, header, "unsupported_sampling_json"),
			GoogleThinkingMode:       csvCell(record, header, "google_thinking_mode"),
			PromptVersion:            csvCell(record, header, "prompt_version"),
			CertificateVersion:       csvCell(record, header, "certificate_version"),
			ProjectFolder:            csvCell(record, header, "project_folder"),
			Surface:                  csvCell(record, header, "surface"),
			TheoremPackID:            csvCell(record, header, "theorem_pack_id"),
			CasesFile:                csvCell(record, header, "cases_file"),
			TheoremsFile:             csvCell(record, header, "theorems_file"),
			CaseSelector:             csvCell(record, header, "case_selector"),
			Hypothesis:               csvCell(record, header, "hypothesis"),
			CasesPath:                csvCell(record, header, "cases_path"),
			TheoremsPath:             csvCell(record, header, "theorems_path"),
			ResultsPath:              csvCell(record, header, "result_file"),
			ChainSummaryPath:         csvCell(record, header, "chain_result_file"),
			ImportCatalogPath:        csvCell(record, header, "import_catalog_file"),
			RawDir:                   csvCell(record, header, "raw_dir"),
			CasesTotal:               csvCell(record, header, "cases_total"),
			PassCount:                csvCell(record, header, "pass_count"),
			FalseRefusalCount:        csvCell(record, header, "false_refusal_count"),
			FalseAcceptCount:         csvCell(record, header, "false_accept_count"),
			RequestFailureCount:      csvCell(record, header, "request_failure_count"),
			SchemaFailureCount:       csvCell(record, header, "schema_failure_count"),
			ParseFailureCount:        csvCell(record, header, "parse_failure_count"),
			KernelFailureCount:       csvCell(record, header, "kernel_failure_count"),
			ContractFailureCount:     csvCell(record, header, "contract_failure_count"),
			FormatFailureCount:       csvCell(record, header, "format_failure_count"),
			AvgLatencyMS:             csvCell(record, header, "avg_latency_ms"),
			MaxLatencyMS:             csvCell(record, header, "max_latency_ms"),
			RunElapsedSeconds:        csvCell(record, header, "run_elapsed_seconds"),
			CategoryCount:            csvCell(record, header, "category_count"),
		})
		last := &rows[len(rows)-1]
		if strings.TrimSpace(last.Provider) == "" {
			last.Provider = deriveBenchmarkProvider(last.Model)
		}
		if strings.TrimSpace(last.CasesFile) == "" {
			last.CasesFile = last.TheoremsFile
		}
		if strings.TrimSpace(last.CasesPath) == "" {
			last.CasesPath = last.TheoremsPath
		}
		if strings.TrimSpace(last.TheoremsFile) == "" && benchmarkProjectUsesTheoremSummaryV2(last.ProjectFolder) {
			last.TheoremsFile = last.CasesFile
		}
		if strings.TrimSpace(last.TheoremsPath) == "" && benchmarkProjectUsesTheoremSummaryV2(last.ProjectFolder) {
			last.TheoremsPath = last.CasesPath
		}
		if strings.TrimSpace(last.TheoremPackID) == "" {
			last.TheoremPackID = deriveBenchmarkTheoremPackID(firstNonEmptyBenchmarkCell(last.TheoremsPath, last.TheoremsFile, last.CasesPath, last.CasesFile))
		}
	}
	return rows, nil
}

func loadBenchmarkRunSummaryRowsFromPaths(paths []string) ([]benchmarkRunSummaryRow, error) {
	normalized := make([]string, 0, len(paths))
	seenPaths := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		key := filepath.Clean(path)
		if _, ok := seenPaths[key]; ok {
			continue
		}
		seenPaths[key] = struct{}{}
		normalized = append(normalized, path)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("at least one benchmark summary path is required")
	}

	rows := make([]benchmarkRunSummaryRow, 0)
	seenRows := make(map[string]struct{})
	foundAny := false
	for _, path := range normalized {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat benchmark summary: %w", err)
		}
		foundAny = true
		loaded, err := loadBenchmarkRunSummaryRows(path)
		if err != nil {
			return nil, err
		}
		for _, row := range loaded {
			key := strings.Join([]string{
				strings.TrimSpace(row.ProjectFolder),
				strings.TrimSpace(row.RunID),
				strings.TrimSpace(row.PromptVersion),
				strings.TrimSpace(row.Model),
				strings.TrimSpace(row.LLMModel),
				strings.TrimSpace(row.ExperimentID),
				normalizeBenchmarkPath(strings.TrimSpace(row.ResultsPath)),
			}, "|")
			if _, ok := seenRows[key]; ok {
				continue
			}
			seenRows[key] = struct{}{}
			rows = append(rows, row)
		}
	}
	if !foundAny {
		return loadBenchmarkRunSummaryRows(normalized[0])
	}
	return rows, nil
}

func loadBenchmarkChainSummaryRowsFromSummaryRows(rows []benchmarkRunSummaryRow, allowDerivedPaths bool) ([]benchmarkChainSummaryLoadedRow, []string) {
	summaryRows := dedupeBenchmarkRunSummaryRows(rows)
	loaded := make([]benchmarkChainSummaryLoadedRow, 0, len(summaryRows)*4)
	warnings := make([]string, 0)
	seenPaths := make(map[string]struct{}, len(summaryRows))
	for _, row := range summaryRows {
		resolvedPath := benchmarkSummaryChainSummaryPathWithFallback(row, allowDerivedPaths)
		if strings.TrimSpace(resolvedPath) == "" {
			continue
		}
		cleanPath := filepath.Clean(resolvedPath)
		if _, ok := seenPaths[cleanPath]; ok {
			continue
		}
		seenPaths[cleanPath] = struct{}{}
		if _, err := os.Stat(cleanPath); err != nil {
			warnings = append(warnings, fmt.Sprintf("chain summary file missing for run `%s`: `%s`", row.RunID, displayBenchmarkPath(cleanPath)))
			continue
		}
		chainRows, err := loadBenchmarkChainSummaryRows(cleanPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to load chain summary for run `%s`: %s", row.RunID, err.Error()))
			continue
		}
		for _, chainRow := range chainRows {
			loaded = append(loaded, benchmarkChainSummaryLoadedRow{
				RunID:                     chainRow.RunID,
				TimestampUTC:              row.TimestampUTC,
				Model:                     row.Model,
				LLMModel:                  row.LLMModel,
				ExperimentName:            row.ExperimentName,
				ExperimentID:              row.ExperimentID,
				DisplayName:               benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
				PromptVersion:             row.PromptVersion,
				Surface:                   row.Surface,
				ChainID:                   chainRow.ChainID,
				ChainProtocol:             chainRow.ChainProtocol,
				CaseFamily:                chainRow.CaseFamily,
				ChainDepth:                parseIntLoose(chainRow.ChainDepth),
				AtomRenamingID:            chainRow.AtomRenamingID,
				SourceFamily:              chainRow.SourceFamily,
				TargetFamily:              chainRow.TargetFamily,
				TransitionGroup:           chainRow.TransitionGroup,
				ImportArity:               chainRow.ImportArity,
				ReuseShape:                chainRow.ReuseShape,
				BridgeDepth:               chainRow.BridgeDepth,
				SymbolOverlap:             chainRow.SymbolOverlap,
				LemmaSurfaceStyle:         chainRow.LemmaSurfaceStyle,
				NegativeTwinHardness:      chainRow.NegativeTwinHardness,
				CaseNotes:                 chainRow.CaseNotes,
				Stage1Pass:                parseBenchmarkBoolLoose(chainRow.Stage1Pass),
				Stage2Pass:                parseBenchmarkBoolLoose(chainRow.Stage2Pass),
				Stage3GoldPass:            parseBenchmarkBoolLoose(chainRow.Stage3GoldPass),
				Stage3ModelPass:           parseBenchmarkBoolLoose(chainRow.Stage3ModelPass),
				NegativeTwinPass:          parseBenchmarkBoolLoose(chainRow.NegativeTwinPass),
				AllStagesPass:             parseBenchmarkBoolLoose(chainRow.AllStagesPass),
				ImportStagePass:           parseBenchmarkBoolLoose(chainRow.ImportStagePass),
				FinalCompositionPassGold:  parseBenchmarkBoolLoose(chainRow.FinalCompositionPassGold),
				FinalCompositionPassModel: parseBenchmarkBoolLoose(chainRow.FinalCompositionPassModel),
				ConditionalFinalPassGold:  parseBenchmarkBoolLoose(chainRow.ConditionalFinalPassGold),
				ConditionalFinalPassModel: parseBenchmarkBoolLoose(chainRow.ConditionalFinalPassModel),
				StageSequence:             parseBenchmarkJSONStringSlice(chainRow.StageSequenceJSON),
				StageRoles:                parseBenchmarkJSONStringMap(chainRow.StageRolesJSON),
				StagePasses:               parseBenchmarkJSONBoolMap(chainRow.StagePassesJSON),
				StageFailures:             parseBenchmarkJSONStringMap(chainRow.StageFailuresJSON),
				TrustedReuseStages:        parseBenchmarkJSONStringSlice(chainRow.TrustedReuseStagesJSON),
				GoldImportStages:          parseBenchmarkJSONStringSlice(chainRow.GoldImportStagesJSON),
				ModelImportStages:         parseBenchmarkJSONStringSlice(chainRow.ModelImportStagesJSON),
				FailureStage:              chainRow.FailureStage,
				FailureStageRole:          chainRow.FailureStageRole,
				FailureType:               chainRow.FailureType,
				FailureClass:              chainRow.FailureClass,
				FailureDetail:             chainRow.FailureDetail,
				Notes:                     chainRow.Notes,
			})
		}
	}
	return dedupeBenchmarkChainSummaryLoadedRows(loaded), warnings
}

func benchmarkSummaryChainSummaryPath(row benchmarkRunSummaryRow) string {
	return benchmarkSummaryChainSummaryPathWithFallback(row, true)
}

func benchmarkSummaryChainSummaryPathWithFallback(row benchmarkRunSummaryRow, allowDerivedPaths bool) string {
	if value := resolveBenchmarkLocalPath(row.ChainSummaryPath); strings.TrimSpace(value) != "" {
		return value
	}
	if !allowDerivedPaths {
		return ""
	}
	resultsPath := resolveBenchmarkLocalPath(row.ResultsPath)
	if strings.TrimSpace(resultsPath) == "" || strings.TrimSpace(row.RunID) == "" {
		return ""
	}
	return benchmarkChainSummaryPath(resultsPath, row.RunID)
}

func loadBenchmarkChainSummaryRows(path string) ([]benchmarkChainSummaryRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open benchmark chain summary: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read benchmark chain summary csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("benchmark chain summary csv must contain a header")
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	required := []string{"run_id", "chain_id", "case_family", "chain_depth", "stage1_pass", "stage2_pass", "stage3_gold_pass", "stage3_model_pass", "negative_twin_pass", "all_stages_pass", "import_stage_pass", "final_composition_pass_gold", "final_composition_pass_model", "conditional_final_pass_gold", "conditional_final_pass_model", "failure_stage", "failure_type"}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, fmt.Errorf("benchmark chain summary csv is missing required column %q", name)
		}
	}

	rows := make([]benchmarkChainSummaryRow, 0, max(0, len(records)-1))
	for _, record := range records[1:] {
		if isCSVRecordEmpty(record) {
			continue
		}
		rows = append(rows, benchmarkChainSummaryRow{
			RunID:                     csvCell(record, header, "run_id"),
			ChainID:                   csvCell(record, header, "chain_id"),
			ChainProtocol:             csvCell(record, header, "chain_protocol"),
			CaseFamily:                csvCell(record, header, "case_family"),
			ChainDepth:                csvCell(record, header, "chain_depth"),
			AtomRenamingID:            csvCell(record, header, "atom_renaming_id"),
			SourceFamily:              csvCell(record, header, "source_family"),
			TargetFamily:              csvCell(record, header, "target_family"),
			TransitionGroup:           csvCell(record, header, "transition_group"),
			ImportArity:               csvCell(record, header, "import_arity"),
			ReuseShape:                csvCell(record, header, "reuse_shape"),
			BridgeDepth:               csvCell(record, header, "bridge_depth"),
			SymbolOverlap:             csvCell(record, header, "symbol_overlap"),
			LemmaSurfaceStyle:         csvCell(record, header, "lemma_surface_style"),
			NegativeTwinHardness:      csvCell(record, header, "negative_twin_hardness"),
			CaseNotes:                 csvCell(record, header, "case_notes"),
			Stage1Pass:                csvCell(record, header, "stage1_pass"),
			Stage2Pass:                csvCell(record, header, "stage2_pass"),
			Stage3GoldPass:            csvCell(record, header, "stage3_gold_pass"),
			Stage3ModelPass:           csvCell(record, header, "stage3_model_pass"),
			NegativeTwinPass:          csvCell(record, header, "negative_twin_pass"),
			AllStagesPass:             csvCell(record, header, "all_stages_pass"),
			ImportStagePass:           csvCell(record, header, "import_stage_pass"),
			FinalCompositionPassGold:  csvCell(record, header, "final_composition_pass_gold"),
			FinalCompositionPassModel: csvCell(record, header, "final_composition_pass_model"),
			ConditionalFinalPassGold:  csvCell(record, header, "conditional_final_pass_gold"),
			ConditionalFinalPassModel: csvCell(record, header, "conditional_final_pass_model"),
			StageSequenceJSON:         csvCell(record, header, "stage_sequence_json"),
			StageRolesJSON:            csvCell(record, header, "stage_roles_json"),
			StagePassesJSON:           csvCell(record, header, "stage_passes_json"),
			StageFailuresJSON:         csvCell(record, header, "stage_failures_json"),
			TrustedReuseStagesJSON:    csvCell(record, header, "trusted_reuse_stages_json"),
			GoldImportStagesJSON:      csvCell(record, header, "gold_import_stages_json"),
			ModelImportStagesJSON:     csvCell(record, header, "model_import_stages_json"),
			FailureStage:              csvCell(record, header, "failure_stage"),
			FailureStageRole:          csvCell(record, header, "failure_stage_role"),
			FailureType:               csvCell(record, header, "failure_type"),
			FailureClass:              csvCell(record, header, "failure_class"),
			FailureDetail:             csvCell(record, header, "failure_detail"),
			Notes:                     csvCell(record, header, "notes"),
		})
	}
	return dedupeBenchmarkChainSummaryFileRows(rows), nil
}

func parseBenchmarkJSONStringSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func parseBenchmarkJSONBoolMap(raw string) map[string]bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	values := make(map[string]bool)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func parseBenchmarkJSONStringMap(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	values := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func benchmarkLoadedChainStageIDs(row benchmarkChainSummaryLoadedRow) []string {
	stageIDs := make([]string, 0, len(row.StageSequence)+len(row.StagePasses)+len(row.StageRoles))
	seen := make(map[string]struct{}, len(row.StageSequence)+len(row.StagePasses)+len(row.StageRoles))
	for _, stageID := range row.StageSequence {
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		if _, ok := seen[stageID]; ok {
			continue
		}
		seen[stageID] = struct{}{}
		stageIDs = append(stageIDs, stageID)
	}
	for stageID := range row.StagePasses {
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		if _, ok := seen[stageID]; ok {
			continue
		}
		seen[stageID] = struct{}{}
		stageIDs = append(stageIDs, stageID)
	}
	for stageID := range row.StageRoles {
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		if _, ok := seen[stageID]; ok {
			continue
		}
		seen[stageID] = struct{}{}
		stageIDs = append(stageIDs, stageID)
	}
	return stageIDs
}

func benchmarkSortedChainStageKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	stageIDs := make([]string, 0, len(values))
	for stageID := range values {
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		stageIDs = append(stageIDs, stageID)
	}
	slices.Sort(stageIDs)
	return stageIDs
}

func dedupeBenchmarkChainSummaryFileRows(rows []benchmarkChainSummaryRow) []benchmarkChainSummaryRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]benchmarkChainSummaryRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.Join([]string{
			strings.TrimSpace(row.RunID),
			strings.TrimSpace(row.ChainID),
			strings.TrimSpace(row.CaseFamily),
			strings.TrimSpace(row.AtomRenamingID),
		}, "\x00")
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func dedupeBenchmarkChainSummaryLoadedRows(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainSummaryLoadedRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]benchmarkChainSummaryLoadedRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.Join([]string{
			strings.TrimSpace(row.RunID),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.ChainID),
			strings.TrimSpace(row.CaseFamily),
			strings.TrimSpace(row.AtomRenamingID),
		}, "\x00")
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func loadBenchmarkResultRows(path string) ([]benchmarkResultRow, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	rows := make([]benchmarkResultRow, 0, len(records))
	for _, record := range records {
		theoremID := firstNonEmptyBenchmarkCell(csvCell(record, header, "theorem_id"), csvCell(record, header, "case_id"))
		rows = append(rows, benchmarkResultRow{
			RunID:                    csvCell(record, header, "run_id"),
			TimestampUTC:             csvCell(record, header, "timestamp_utc"),
			Provider:                 csvCell(record, header, "provider"),
			Model:                    csvCell(record, header, "model"),
			LLMModel:                 firstNonEmptyBenchmarkCell(csvCell(record, header, "llm_model"), csvCell(record, header, "model")),
			ExperimentName:           csvCell(record, header, "experiment_name"),
			ExperimentID:             csvCell(record, header, "experiment_id"),
			SamplingSurface:          csvCell(record, header, "sampling_surface"),
			RequestedSamplingProfile: csvCell(record, header, "requested_sampling_profile"),
			EffectiveSamplingProfile: csvCell(record, header, "effective_sampling_profile"),
			RequestedSamplingJSON:    csvCell(record, header, "requested_sampling_json"),
			EffectiveSamplingJSON:    csvCell(record, header, "effective_sampling_json"),
			UnsupportedSamplingJSON:  csvCell(record, header, "unsupported_sampling_json"),
			GoogleThinkingMode:       csvCell(record, header, "google_thinking_mode"),
			PromptVersion:            csvCell(record, header, "prompt_version"),
			PromptHash:               csvCell(record, header, "prompt_hash"),
			Surface:                  csvCell(record, header, "surface"),
			NDProofStatus:            csvCell(record, header, "nd_proof_status"),
			LoweringStatus:           csvCell(record, header, "lowering_status"),
			PipelineStage:            csvCell(record, header, "pipeline_stage"),
			PipelineDetail:           csvCell(record, header, "pipeline_detail"),
			TheoremPackID:            csvCell(record, header, "theorem_pack_id"),
			TheoremID:                theoremID,
			CaseID:                   firstNonEmptyBenchmarkCell(csvCell(record, header, "case_id"), theoremID),
			ChainID:                  csvCell(record, header, "chain_id"),
			StageID:                  csvCell(record, header, "stage_id"),
			Category:                 csvCell(record, header, "category"),
			CaseFamily:               csvCell(record, header, "case_family"),
			ChainDepth:               csvCell(record, header, "chain_depth"),
			ProvenanceMode:           csvCell(record, header, "provenance_mode"),
			AtomRenamingID:           csvCell(record, header, "atom_renaming_id"),
			SourceFamily:             csvCell(record, header, "source_family"),
			TargetFamily:             csvCell(record, header, "target_family"),
			TransitionGroup:          csvCell(record, header, "transition_group"),
			ImportArity:              csvCell(record, header, "import_arity"),
			ReuseShape:               csvCell(record, header, "reuse_shape"),
			BridgeDepth:              csvCell(record, header, "bridge_depth"),
			SymbolOverlap:            csvCell(record, header, "symbol_overlap"),
			LemmaSurfaceStyle:        csvCell(record, header, "lemma_surface_style"),
			NegativeTwinHardness:     csvCell(record, header, "negative_twin_hardness"),
			CaseNotes:                csvCell(record, header, "case_notes"),
			ExpectedLabel:            firstNonEmptyBenchmarkCell(csvCell(record, header, "expected_label"), csvCell(record, header, "label")),
			RawOutputKind:            csvCell(record, header, "raw_output_kind"),
			SchemaStatus:             csvCell(record, header, "schema_status"),
			ParseStatus:              csvCell(record, header, "parse_status"),
			KernelStatus:             csvCell(record, header, "kernel_status"),
			ContractStatus:           firstNonEmptyBenchmarkCell(csvCell(record, header, "contract_status"), "not_run"),
			CLIExitCode:              csvCell(record, header, "cli_exit_code"),
			ScoreBucket:              csvCell(record, header, "score_bucket"),
			LatencyMS:                csvCell(record, header, "latency_ms"),
			OutputLength:             csvCell(record, header, "output_length"),
			ProofStepsCount:          csvCell(record, header, "proof_steps_count"),
			RequestedImportsJSON:     csvCell(record, header, "requested_imports_json"),
			EffectiveImportsJSON:     csvCell(record, header, "effective_imports_json"),
			ResolvedImportRefsJSON:   csvCell(record, header, "resolved_import_refs_json"),
			CertificateObjectPath:    csvCell(record, header, "certificate_object_file"),
			Notes:                    csvCell(record, header, "notes"),
		})
		last := &rows[len(rows)-1]
		if strings.TrimSpace(last.Provider) == "" {
			last.Provider = deriveBenchmarkProvider(last.Model)
		}
	}
	return dedupeBenchmarkResultRows(rows), nil
}

func loadBenchmarkResultRowsFromSummaryRows(rows []benchmarkRunSummaryRow) ([]benchmarkResultRow, map[string]benchmarkRunSummaryRow, []string) {
	summaryRows := dedupeBenchmarkRunSummaryRows(rows)
	summaryByRun := make(map[string]benchmarkRunSummaryRow, len(summaryRows))
	for _, row := range summaryRows {
		summaryByRun[strings.TrimSpace(row.RunID)] = row
	}

	loaded := make([]benchmarkResultRow, 0, len(summaryRows)*8)
	warnings := make([]string, 0)
	seenPaths := make(map[string]struct{}, len(summaryRows))
	for _, row := range summaryRows {
		resolvedPath := resolveBenchmarkLocalPath(row.ResultsPath)
		if strings.TrimSpace(resolvedPath) == "" {
			warnings = append(warnings, fmt.Sprintf("run `%s` has no result_file path", row.RunID))
			continue
		}
		cleanPath := filepath.Clean(resolvedPath)
		if _, ok := seenPaths[cleanPath]; ok {
			continue
		}
		seenPaths[cleanPath] = struct{}{}
		if _, err := os.Stat(cleanPath); err != nil {
			warnings = append(warnings, fmt.Sprintf("result file missing for run `%s`: `%s`", row.RunID, displayBenchmarkPath(cleanPath)))
			continue
		}
		resultRows, err := loadBenchmarkResultRows(cleanPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to load result file for run `%s`: %s", row.RunID, err.Error()))
			continue
		}
		loaded = append(loaded, resultRows...)
	}
	return dedupeBenchmarkResultRows(loaded), summaryByRun, warnings
}
