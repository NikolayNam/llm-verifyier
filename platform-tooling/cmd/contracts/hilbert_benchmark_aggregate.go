package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// This file stays focused on summary/comparative aggregation and pack assembly.
// Observed result-row forensics and chain/compositional protocol aggregation are
// implemented in dedicated files.

func aggregateBenchmarkResultRows(rows []benchmarkResultRow) ([]benchmarkResultSliceAggregate, error) {
	return aggregateBenchmarkResultRowsImpl(rows)
}

func aggregateBenchmarkChainSummaryRows(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainSummaryAggregate {
	return aggregateBenchmarkChainSummaryRowsImpl(rows)
}

func aggregateBenchmarkChainStageSummaryRows(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainStageAggregate {
	return aggregateBenchmarkChainStageSummaryRowsImpl(rows)
}

func aggregateBenchmarkChainFailures(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainFailureAggregate {
	return aggregateBenchmarkChainFailuresImpl(rows)
}

func applyBenchmarkChainSummaryCatalog(aggregates []benchmarkChainSummaryAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkChainSummaryCatalogImpl(aggregates, catalog)
}

func applyBenchmarkChainStageCatalog(aggregates []benchmarkChainStageAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkChainStageCatalogImpl(aggregates, catalog)
}

func applyBenchmarkChainFailureCatalog(aggregates []benchmarkChainFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkChainFailureCatalogImpl(aggregates, catalog)
}

func aggregateBenchmarkChainMetadataRows(rows []benchmarkChainSummaryLoadedRow, selector func(benchmarkChainSummaryLoadedRow) string) []benchmarkChainMetadataAggregate {
	return aggregateBenchmarkChainMetadataRowsImpl(rows, selector)
}

func applyBenchmarkChainMetadataCatalog(aggregates []benchmarkChainMetadataAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkChainMetadataCatalogImpl(aggregates, catalog)
}

func aggregateBenchmarkFGFailureRows(rows []benchmarkResultRow) []benchmarkFGFailureAggregate {
	return aggregateBenchmarkFGFailureRowsImpl(rows)
}

func applyBenchmarkFGFailureCatalog(aggregates []benchmarkFGFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkFGFailureCatalogImpl(aggregates, catalog)
}

func aggregateBenchmarkFGCaseFailures(rows []benchmarkResultRow) []benchmarkFGCaseFailureAggregate {
	return aggregateBenchmarkFGCaseFailuresImpl(rows)
}

func applyBenchmarkFGCaseFailureCatalog(aggregates []benchmarkFGCaseFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkFGCaseFailureCatalogImpl(aggregates, catalog)
}

func applyBenchmarkModelCatalogToResultAggregates(aggregates []benchmarkResultSliceAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkModelCatalogToResultAggregatesImpl(aggregates, catalog)
}

func buildBenchmarkNDPipelineTraceAggregates(rows []benchmarkResultRow) []benchmarkNDPipelineTraceAggregate {
	return buildBenchmarkNDPipelineTraceAggregatesImpl(rows)
}

func benchmarkNDPipelineStage(row benchmarkResultRow) string {
	return benchmarkNDPipelineStageImpl(row)
}

func applyBenchmarkModelCatalogToNDPipelineTraceAggregates(aggregates []benchmarkNDPipelineTraceAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkModelCatalogToNDPipelineTraceAggregatesImpl(aggregates, catalog)
}

func loadBenchmarkCaseIndex(path string) (map[string]benchmarkCase, error) {
	return loadBenchmarkCaseIndexImpl(path)
}

func benchmarkObservedMetadataPath(rows []benchmarkRunSummaryRow) string {
	return benchmarkObservedMetadataPathImpl(rows)
}

func buildBenchmarkObservedData(projectFolder string, rows []benchmarkRunSummaryRow, modelCatalog []benchmarkModelCatalogRow) (*benchmarkObservedData, error) {
	return buildBenchmarkObservedDataImpl(projectFolder, rows, modelCatalog)
}

func buildBenchmarkCaseFailureAggregates(caseIndex map[string]benchmarkCase, rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow) ([]benchmarkCaseFailureAggregate, error) {
	return buildBenchmarkCaseFailureAggregatesImpl(caseIndex, rows, summaryByRun)
}

func buildBenchmarkHardCaseAudits(caseIDs []string, caseIndex map[string]benchmarkCase, rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow, catalog map[string]benchmarkModelCatalogRow) ([]benchmarkHardCaseAudit, []string, error) {
	return buildBenchmarkHardCaseAuditsImpl(caseIDs, caseIndex, rows, summaryByRun, catalog)
}

func aggregateBenchmarkHardCaseRows(rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow) ([]benchmarkHardCaseAuditAggregate, error) {
	return aggregateBenchmarkHardCaseRowsImpl(rows, summaryByRun)
}

func applyBenchmarkModelCatalogToHardCaseAggregates(aggregates []benchmarkHardCaseAuditAggregate, catalog map[string]benchmarkModelCatalogRow) {
	applyBenchmarkModelCatalogToHardCaseAggregatesImpl(aggregates, catalog)
}

func sortedBenchmarkResultSliceKeys[T any](groups map[string][]T) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func loadBenchmarkModelCatalog(path string) ([]benchmarkModelCatalogRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open benchmark model catalog: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read benchmark model catalog csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("benchmark model catalog csv must contain a header")
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	required := []string{"llm_model", "display_name", "size_bucket", "size_label", "tier", "comparison_status", "notes"}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, fmt.Errorf("benchmark model catalog csv is missing required column %q", name)
		}
	}

	rows := make([]benchmarkModelCatalogRow, 0, max(0, len(records)-1))
	for _, record := range records[1:] {
		if isCSVRecordEmpty(record) {
			continue
		}
		row := benchmarkModelCatalogRow{
			LLMModel:         strings.TrimSpace(csvCell(record, header, "llm_model")),
			DisplayName:      strings.TrimSpace(csvCell(record, header, "display_name")),
			SizeBucket:       strings.TrimSpace(csvCell(record, header, "size_bucket")),
			SizeLabel:        strings.TrimSpace(csvCell(record, header, "size_label")),
			Tier:             strings.TrimSpace(csvCell(record, header, "tier")),
			ComparisonStatus: strings.TrimSpace(csvCell(record, header, "comparison_status")),
			Notes:            strings.TrimSpace(csvCell(record, header, "notes")),
		}
		if row.LLMModel == "" {
			return nil, fmt.Errorf("benchmark model catalog row has empty llm_model")
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func aggregateBenchmarkRunSummary(rows []benchmarkRunSummaryRow) ([]benchmarkRunSummaryAggregate, error) {
	groups := make(map[string]*benchmarkRunSummaryAggregate)
	for _, row := range rows {
		key := row.Model + "\x00" + row.LLMModel + "\x00" + row.ExperimentID + "\x00" + row.PromptVersion + "\x00" + row.CertificateVersion
		group, ok := groups[key]
		if !ok {
			group = &benchmarkRunSummaryAggregate{
				Model:              row.Model,
				LLMModel:           row.LLMModel,
				ExperimentName:     row.ExperimentName,
				ExperimentID:       row.ExperimentID,
				DisplayName:        benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
				PromptVersion:      row.PromptVersion,
				CertificateVersion: row.CertificateVersion,
			}
			groups[key] = group
		}

		group.Runs++

		casesTotal, err := parseBenchmarkSummaryInt(row.RunID, "cases_total", row.CasesTotal)
		if err != nil {
			return nil, err
		}
		passCount, err := parseBenchmarkSummaryInt(row.RunID, "pass_count", row.PassCount)
		if err != nil {
			return nil, err
		}
		falseRefusalCount, err := parseBenchmarkSummaryInt(row.RunID, "false_refusal_count", row.FalseRefusalCount)
		if err != nil {
			return nil, err
		}
		falseAcceptCount, err := parseBenchmarkSummaryInt(row.RunID, "false_accept_count", row.FalseAcceptCount)
		if err != nil {
			return nil, err
		}
		requestFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "request_failure_count", row.RequestFailureCount)
		if err != nil {
			return nil, err
		}
		schemaFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "schema_failure_count", row.SchemaFailureCount)
		if err != nil {
			return nil, err
		}
		parseFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "parse_failure_count", row.ParseFailureCount)
		if err != nil {
			return nil, err
		}
		kernelFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "kernel_failure_count", row.KernelFailureCount)
		if err != nil {
			return nil, err
		}
		contractFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "contract_failure_count", row.ContractFailureCount)
		if err != nil {
			return nil, err
		}
		formatFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "format_failure_count", row.FormatFailureCount)
		if err != nil {
			return nil, err
		}
		avgLatencyMS, avgLatencyPresent, err := parseBenchmarkSummaryFloat(row.RunID, "avg_latency_ms", row.AvgLatencyMS)
		if err != nil {
			return nil, err
		}
		maxLatencyMS, maxLatencyPresent, err := parseBenchmarkSummaryInt64(row.RunID, "max_latency_ms", row.MaxLatencyMS)
		if err != nil {
			return nil, err
		}
		runElapsedSeconds, runElapsedPresent, err := parseBenchmarkSummaryInt64(row.RunID, "run_elapsed_seconds", row.RunElapsedSeconds)
		if err != nil {
			return nil, err
		}

		group.CasesTotal += casesTotal
		group.PassCount += passCount
		group.FalseRefusalCount += falseRefusalCount
		group.FalseAcceptCount += falseAcceptCount
		group.RequestFailureCount += requestFailureCount
		group.SchemaFailureCount += schemaFailureCount
		group.ParseFailureCount += parseFailureCount
		group.KernelFailureCount += kernelFailureCount
		group.ContractFailureCount += contractFailureCount
		group.FormatFailureCount += formatFailureCount
		if avgLatencyPresent {
			group.LatencyCasesTotal += casesTotal
			group.LatencySumMS += avgLatencyMS * float64(casesTotal)
		}
		if maxLatencyPresent && maxLatencyMS > group.MaxLatencyMS {
			group.MaxLatencyMS = maxLatencyMS
		}
		if runElapsedPresent {
			group.TimedRuns++
			group.RunElapsedTotalS += runElapsedSeconds
			if runElapsedSeconds > group.RunElapsedMaxS {
				group.RunElapsedMaxS = runElapsedSeconds
			}
		}

		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err != nil {
			return nil, fmt.Errorf("benchmark summary row %q has invalid timestamp_utc %q: %w", row.RunID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}

	aggregates := make([]benchmarkRunSummaryAggregate, 0, len(groups))
	for _, group := range groups {
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkRunSummaryAggregate) int {
		if cmp := strings.Compare(a.Model, b.Model); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.ExperimentID, b.ExperimentID); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.CertificateVersion, b.CertificateVersion)
	})
	return aggregates, nil
}

func applyBenchmarkModelCatalog(aggregates []benchmarkRunSummaryAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
		aggregates[idx].SizeBucket = entry.SizeBucket
		aggregates[idx].SizeLabel = entry.SizeLabel
		aggregates[idx].Tier = entry.Tier
		aggregates[idx].ComparisonStatus = entry.ComparisonStatus
	}
}

func filterBenchmarkRunSummaryRows(rows []benchmarkRunSummaryRow, workspaceRoot, projectFolder, casesFile string) []benchmarkRunSummaryRow {
	projectFolder = strings.TrimSpace(projectFolder)
	casesFile = canonicalBenchmarkInputFile(projectFolder, casesFile)
	filtered := make([]benchmarkRunSummaryRow, 0, len(rows))
	for _, row := range rows {
		rowProjectFolder := strings.TrimSpace(row.ProjectFolder)
		if rowProjectFolder == "" {
			rowProjectFolder = inferBenchmarkProjectFolder(row.CasesPath, workspaceRoot, "")
		}
		if projectFolder != "" && rowProjectFolder != projectFolder {
			continue
		}

		rowCasesFile := benchmarkRunSummaryInputFile(row, workspaceRoot, rowProjectFolder)
		if rowCasesFile == "" {
			rowCasesFile = defaultBenchmarkInputFile(rowProjectFolder)
		}
		if casesFile != "" && casesFile != "*" && rowCasesFile != casesFile {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func buildBenchmarkCasePackReports(rows []benchmarkRunSummaryRow, projectFolder string, modelCatalog []benchmarkModelCatalogRow) ([]benchmarkCasePackReport, error) {
	catalogByModel := indexBenchmarkModelCatalog(modelCatalog)
	byCasesFile := make(map[string][]benchmarkRunSummaryRow)
	for _, row := range rows {
		key := benchmarkRunSummaryInputFile(row, "", strings.TrimSpace(row.ProjectFolder))
		if key == "" {
			key = defaultBenchmarkInputFile(strings.TrimSpace(row.ProjectFolder))
		}
		byCasesFile[key] = append(byCasesFile[key], row)
	}

	caseFiles := make([]string, 0, len(byCasesFile))
	for casesFile := range byCasesFile {
		caseFiles = append(caseFiles, casesFile)
	}
	slices.Sort(caseFiles)

	reports := make([]benchmarkCasePackReport, 0, len(caseFiles))
	for _, casesFile := range caseFiles {
		packRows := byCasesFile[casesFile]
		aggregates, err := aggregateBenchmarkRunSummary(packRows)
		if err != nil {
			return nil, err
		}
		applyBenchmarkModelCatalog(aggregates, catalogByModel)
		hypothesis := firstNonEmptyHypothesis(packRows)
		var comparative *benchmarkComparativeSlices
		if len(catalogByModel) > 0 {
			comparative, err = buildBenchmarkComparativeSlices(packRows, catalogByModel)
			if err != nil {
				return nil, err
			}
		}
		var observed *benchmarkObservedData
		if benchmarkProjectSupportsObservedSlices(projectFolder) {
			observed, err = buildBenchmarkObservedData(projectFolder, packRows, modelCatalog)
			if err != nil {
				return nil, err
			}
		}
		chainSummary, err := buildBenchmarkChainSummaryData(packRows, modelCatalog)
		if err != nil {
			return nil, err
		}
		reports = append(reports, benchmarkCasePackReport{
			CasesFile:     casesFile,
			Hypothesis:    hypothesis,
			Rows:          packRows,
			Aggregates:    aggregates,
			Comparative:   comparative,
			Observed:      observed,
			ChainSummary:  chainSummary,
			TotalRuns:     len(packRows),
			GeneratedAt:   time.Now().UTC(),
			ProjectFolder: projectFolder,
		})
	}
	return reports, nil
}

func benchmarkRunSummaryInputFile(row benchmarkRunSummaryRow, workspaceRoot, projectFolder string) string {
	projectFolder = strings.TrimSpace(projectFolder)
	if projectFolder == "" {
		projectFolder = strings.TrimSpace(row.ProjectFolder)
	}

	if value := strings.TrimSpace(row.TheoremsFile); value != "" {
		return canonicalBenchmarkInputFile(projectFolder, value)
	}
	if value := strings.TrimSpace(row.CasesFile); value != "" {
		return canonicalBenchmarkInputFile(projectFolder, value)
	}

	inputPath := strings.TrimSpace(row.TheoremsPath)
	if inputPath == "" {
		inputPath = strings.TrimSpace(row.CasesPath)
	}
	if inputPath == "" {
		return ""
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		return canonicalBenchmarkInputFile(projectFolder, inputPath)
	}
	return inferBenchmarkCasesFile(inputPath, workspaceRoot, projectFolder, "")
}

func buildBenchmarkChainSummaryData(rows []benchmarkRunSummaryRow, modelCatalog []benchmarkModelCatalogRow) (*benchmarkChainSummaryData, error) {
	return buildBenchmarkChainSummaryDataImpl(rows, modelCatalog)
}

func firstNonEmptyHypothesis(rows []benchmarkRunSummaryRow) string {
	for _, row := range rows {
		if value := strings.TrimSpace(row.Hypothesis); value != "" {
			return value
		}
	}
	return ""
}

func indexBenchmarkModelCatalog(rows []benchmarkModelCatalogRow) map[string]benchmarkModelCatalogRow {
	index := make(map[string]benchmarkModelCatalogRow, len(rows))
	for _, row := range rows {
		index[strings.TrimSpace(row.LLMModel)] = row
	}
	return index
}

func buildBenchmarkComparativeSlices(rows []benchmarkRunSummaryRow, catalog map[string]benchmarkModelCatalogRow) (*benchmarkComparativeSlices, error) {
	engineeringRows := make([]benchmarkRunSummaryRow, 0)
	cleanRows := make([]benchmarkRunSummaryRow, 0)
	le20Rows := make([]benchmarkRunSummaryRow, 0)
	gt20Rows := make([]benchmarkRunSummaryRow, 0)
	unknownRows := make([]benchmarkRunSummaryRow, 0)
	appendixEntries := make([]benchmarkComparativeAppendixEntry, 0)
	invalidEntries := make(map[string]*benchmarkComparativeAppendixEntry)

	validV13ByModel := make(map[string]bool)
	anyV13ByModel := make(map[string]bool)
	validV12ByModel := make(map[string]bool)

	for _, row := range rows {
		entry, ok := catalog[strings.TrimSpace(row.LLMModel)]
		if !ok {
			if strings.TrimSpace(row.PromptVersion) == benchmarkPromptVersionV13 {
				appendBenchmarkInvalidEntry(invalidEntries, row, benchmarkComparativeAppendixEntry{
					Model:            row.Model,
					LLMModel:         row.LLMModel,
					ExperimentName:   row.ExperimentName,
					ExperimentID:     row.ExperimentID,
					DisplayName:      benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
					PromptVersion:    row.PromptVersion,
					Status:           "uncatalogued",
					Reason:           "llm_model is not present in model-catalog.csv",
					ComparisonStatus: "uncatalogued",
				})
			}
			continue
		}

		isValid, reason, err := isBenchmarkComparativeRunValid(row)
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(row.PromptVersion) == benchmarkPromptVersionV13 {
			anyV13ByModel[entry.LLMModel] = true
		}

		if !isValid {
			appendBenchmarkInvalidEntry(invalidEntries, row, benchmarkComparativeAppendixEntry{
				Model:            row.Model,
				LLMModel:         row.LLMModel,
				ExperimentName:   row.ExperimentName,
				ExperimentID:     row.ExperimentID,
				DisplayName:      benchmarkDisplayNameWithExperiment(entry.DisplayName, row.ExperimentID),
				SizeBucket:       entry.SizeBucket,
				SizeLabel:        entry.SizeLabel,
				Tier:             entry.Tier,
				ComparisonStatus: entry.ComparisonStatus,
				PromptVersion:    row.PromptVersion,
				Status:           "invalid-run",
				Reason:           reason,
			})
			continue
		}

		switch strings.TrimSpace(row.PromptVersion) {
		case benchmarkPromptVersionV12:
			engineeringRows = append(engineeringRows, row)
			validV12ByModel[entry.LLMModel] = true
		case benchmarkPromptVersionV13:
			cleanRows = append(cleanRows, row)
			validV13ByModel[entry.LLMModel] = true
			switch {
			case strings.TrimSpace(entry.ComparisonStatus) == "primary" && strings.TrimSpace(entry.SizeBucket) == "<=20b":
				le20Rows = append(le20Rows, row)
			case strings.TrimSpace(entry.ComparisonStatus) == "primary" && strings.TrimSpace(entry.SizeBucket) == ">20b":
				gt20Rows = append(gt20Rows, row)
			case strings.TrimSpace(entry.SizeBucket) == "unknown" || strings.TrimSpace(entry.ComparisonStatus) == "appendix":
				unknownRows = append(unknownRows, row)
			}
		}
	}

	for _, entry := range catalog {
		if validV13ByModel[entry.LLMModel] {
			continue
		}
		if anyBenchmarkAppendixEntryForModel(invalidEntries, entry.LLMModel, benchmarkPromptVersionV13) {
			continue
		}
		status := strings.TrimSpace(entry.ComparisonStatus)
		reason := "no valid v2/v1.3 run recorded for this case pack"
		switch {
		case anyV13ByModel[entry.LLMModel]:
			reason = "only invalid v2/v1.3 runs were recorded for this case pack"
		case validV12ByModel[entry.LLMModel]:
			reason = "only v1.2 engineering runs were recorded; no valid v2/v1.3 baseline run exists for this case pack"
		case status == "insufficient-evidence":
			reason = "catalog marks the model as insufficient-evidence until a valid v2/v1.3 run exists"
		case status == "appendix":
			reason = "model is intentionally kept in the appendix and has no valid v2/v1.3 run for this case pack"
		}
		entryStatus := "missing-v1.3-evidence"
		if status == "insufficient-evidence" || status == "appendix" {
			entryStatus = status
		}
		appendixEntries = append(appendixEntries, benchmarkComparativeAppendixEntry{
			LLMModel:         entry.LLMModel,
			DisplayName:      entry.DisplayName,
			SizeBucket:       entry.SizeBucket,
			SizeLabel:        entry.SizeLabel,
			Tier:             entry.Tier,
			ComparisonStatus: entry.ComparisonStatus,
			PromptVersion:    benchmarkPromptVersionV13,
			Status:           entryStatus,
			Reason:           reason,
		})
	}

	invalidKeys := make([]string, 0, len(invalidEntries))
	for key := range invalidEntries {
		invalidKeys = append(invalidKeys, key)
	}
	slices.Sort(invalidKeys)
	for _, key := range invalidKeys {
		appendixEntries = append(appendixEntries, *invalidEntries[key])
	}
	slices.SortFunc(appendixEntries, func(a, b benchmarkComparativeAppendixEntry) int {
		if cmp := strings.Compare(a.Status, b.Status); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.ExperimentID, b.ExperimentID); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.PromptVersion, b.PromptVersion)
	})

	engineeringAggregates, err := aggregateBenchmarkRunSummary(engineeringRows)
	if err != nil {
		return nil, err
	}
	cleanAggregates, err := aggregateBenchmarkRunSummary(cleanRows)
	if err != nil {
		return nil, err
	}
	le20Aggregates, err := aggregateBenchmarkRunSummary(le20Rows)
	if err != nil {
		return nil, err
	}
	gt20Aggregates, err := aggregateBenchmarkRunSummary(gt20Rows)
	if err != nil {
		return nil, err
	}
	unknownAggregates, err := aggregateBenchmarkRunSummary(unknownRows)
	if err != nil {
		return nil, err
	}
	applyBenchmarkModelCatalog(engineeringAggregates, catalog)
	applyBenchmarkModelCatalog(cleanAggregates, catalog)
	applyBenchmarkModelCatalog(le20Aggregates, catalog)
	applyBenchmarkModelCatalog(gt20Aggregates, catalog)
	applyBenchmarkModelCatalog(unknownAggregates, catalog)

	return &benchmarkComparativeSlices{
		EngineeringBestPerformance: engineeringAggregates,
		CleanBaseline:              cleanAggregates,
		LE20B:                      le20Aggregates,
		GT20B:                      gt20Aggregates,
		UnknownSizeAppendix:        unknownAggregates,
		Appendix:                   appendixEntries,
	}, nil
}

func anyBenchmarkAppendixEntryForModel(entries map[string]*benchmarkComparativeAppendixEntry, llmModel, promptVersion string) bool {
	for _, entry := range entries {
		if entry.LLMModel == llmModel && strings.TrimSpace(entry.PromptVersion) == strings.TrimSpace(promptVersion) {
			return true
		}
	}
	return false
}

func appendBenchmarkInvalidEntry(groups map[string]*benchmarkComparativeAppendixEntry, row benchmarkRunSummaryRow, seed benchmarkComparativeAppendixEntry) {
	key := strings.TrimSpace(seed.Model) + "\x00" + strings.TrimSpace(seed.LLMModel) + "\x00" + strings.TrimSpace(seed.ExperimentID) + "\x00" + strings.TrimSpace(seed.PromptVersion) + "\x00" + strings.TrimSpace(seed.Status)
	group, ok := groups[key]
	if !ok {
		group = &benchmarkComparativeAppendixEntry{
			Model:            seed.Model,
			LLMModel:         seed.LLMModel,
			ExperimentName:   seed.ExperimentName,
			ExperimentID:     seed.ExperimentID,
			DisplayName:      seed.DisplayName,
			SizeBucket:       seed.SizeBucket,
			SizeLabel:        seed.SizeLabel,
			Tier:             seed.Tier,
			ComparisonStatus: seed.ComparisonStatus,
			PromptVersion:    seed.PromptVersion,
			Status:           seed.Status,
			Reason:           seed.Reason,
		}
		groups[key] = group
	} else if !strings.Contains(group.Reason, seed.Reason) {
		group.Reason = group.Reason + "; " + seed.Reason
	}

	group.Runs++
	group.CasesTotal += mustParseBenchmarkSummaryInt(row.CasesTotal)
	group.PassCount += mustParseBenchmarkSummaryInt(row.PassCount)
	group.RequestFailureCount += mustParseBenchmarkSummaryInt(row.RequestFailureCount)
	timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
	if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
		group.LatestRunID = row.RunID
		group.latestTimestamp = timestamp
	}
}

func mustParseBenchmarkSummaryInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func statusOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func isBenchmarkComparativeRunValid(row benchmarkRunSummaryRow) (bool, string, error) {
	casesTotal, err := parseBenchmarkSummaryInt(row.RunID, "cases_total", row.CasesTotal)
	if err != nil {
		return false, "", err
	}
	requestFailureCount, err := parseBenchmarkSummaryInt(row.RunID, "request_failure_count", row.RequestFailureCount)
	if err != nil {
		return false, "", err
	}
	if casesTotal > 0 && requestFailureCount >= casesTotal {
		reason := inspectBenchmarkOperationalFailure(row.ResultsPath)
		if strings.TrimSpace(reason) == "" {
			reason = "all cases failed at the request layer before a meaningful proof-quality comparison could be made"
		}
		return false, reason, nil
	}
	return true, "", nil
}

func inspectBenchmarkOperationalFailure(path string) string {
	resolvedPath := resolveBenchmarkLocalPath(path)
	if strings.TrimSpace(resolvedPath) == "" {
		return ""
	}
	file, err := os.Open(resolvedPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return ""
	}
	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	scoreIdx, okScore := header["score_bucket"]
	notesIdx, okNotes := header["notes"]
	if !okScore || !okNotes {
		return ""
	}

	allRequestFailures := true
	combinedNotes := make([]string, 0, len(records)-1)
	for _, record := range records[1:] {
		if isCSVRecordEmpty(record) {
			continue
		}
		if cellAt(record, scoreIdx) != "request_failure" {
			allRequestFailures = false
			break
		}
		if note := strings.TrimSpace(cellAt(record, notesIdx)); note != "" {
			combinedNotes = append(combinedNotes, note)
		}
	}
	if !allRequestFailures {
		return ""
	}
	for _, note := range combinedNotes {
		lower := strings.ToLower(note)
		switch {
		case strings.Contains(lower, "not found on the target endpoint"):
			return "model not found on the target endpoint"
		case strings.Contains(lower, "model") && strings.Contains(lower, "not found"):
			return "model not found on the target endpoint"
		case strings.Contains(lower, "transport"):
			return note
		}
	}
	if len(combinedNotes) > 0 {
		return combinedNotes[0]
	}
	return ""
}

func cellAt(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return record[idx]
}
