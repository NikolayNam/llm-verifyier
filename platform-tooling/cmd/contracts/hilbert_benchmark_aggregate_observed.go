package main

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type benchmarkPackMetadata struct {
	InputPath        string
	CaseIndex        map[string]benchmarkCase
	HasChainMetadata bool
}

func loadBenchmarkPackMetadata(rows []benchmarkRunSummaryRow) (*benchmarkPackMetadata, error) {
	metadataPath := benchmarkObservedMetadataPath(rows)
	if strings.TrimSpace(metadataPath) == "" {
		return nil, nil
	}
	caseIndex, err := loadBenchmarkCaseIndex(metadataPath)
	if err != nil {
		return nil, err
	}
	hasChainMetadata := false
	for _, tc := range caseIndex {
		if benchmarkCaseUsesCompositionalChainProtocol(tc) {
			hasChainMetadata = true
			break
		}
	}
	return &benchmarkPackMetadata{
		InputPath:        metadataPath,
		CaseIndex:        caseIndex,
		HasChainMetadata: hasChainMetadata,
	}, nil
}

// Observed-data aggregation stays separate from summary aggregation because it
// pulls in result-row forensics, case metadata, and hard-case audits.
func aggregateBenchmarkResultRowsImpl(rows []benchmarkResultRow) ([]benchmarkResultSliceAggregate, error) {
	groups := make(map[string]*benchmarkResultSliceAggregate)
	runMembership := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkResultRows(rows) {
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkResultSliceAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
				DisplayName:    benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
			}
			groups[key] = group
			runMembership[key] = make(map[string]struct{})
		}
		if _, ok := runMembership[key][row.RunID]; !ok {
			runMembership[key][row.RunID] = struct{}{}
			group.Runs++
		}
		group.CasesTotal++
		switch strings.TrimSpace(row.ScoreBucket) {
		case "pass":
			group.PassCount++
		case "false_refusal":
			group.FalseRefusalCount++
		case "false_accept":
			group.FalseAcceptCount++
		case "request_failure":
			group.RequestFailureCount++
		case "schema_failure":
			group.SchemaFailureCount++
		case "parse_failure":
			group.ParseFailureCount++
		case "kernel_failure":
			group.KernelFailureCount++
		case "contract_failure":
			group.ContractFailureCount++
		case "format_failure":
			group.FormatFailureCount++
		}
		latencyMS, latencyPresent, err := parseBenchmarkResultFloat(row.RunID, row.CaseID, "latency_ms", row.LatencyMS)
		if err != nil {
			return nil, err
		}
		if latencyPresent {
			group.LatencyCasesTotal++
			group.LatencySumMS += latencyMS
			if latencyMS > group.MaxLatencyMS {
				group.MaxLatencyMS = latencyMS
			}
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err != nil {
			return nil, fmt.Errorf("benchmark result row %q/%q has invalid timestamp_utc %q: %w", row.RunID, row.CaseID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}

	aggregates := make([]benchmarkResultSliceAggregate, 0, len(groups))
	for _, group := range groups {
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkResultSliceAggregate) int {
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
		return strings.Compare(a.Surface, b.Surface)
	})
	return aggregates, nil
}

func applyBenchmarkModelCatalogToResultAggregatesImpl(aggregates []benchmarkResultSliceAggregate, catalog map[string]benchmarkModelCatalogRow) {
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

func buildBenchmarkNDPipelineTraceAggregatesImpl(rows []benchmarkResultRow) []benchmarkNDPipelineTraceAggregate {
	groups := make(map[string]*benchmarkNDPipelineTraceAggregate)
	runMembership := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkResultRows(rows) {
		stage := benchmarkNDPipelineStage(row)
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkNDPipelineTraceAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				DisplayName:    benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
			}
			groups[key] = group
			runMembership[key] = make(map[string]struct{})
		}
		if _, ok := runMembership[key][row.RunID]; !ok {
			runMembership[key][row.RunID] = struct{}{}
			group.Runs++
		}
		group.CasesTotal++
		switch stage {
		case "nd_proof_object_failure":
			group.NDProofObjectFailures++
		case "lowering_failure":
			group.LoweringFailures++
		case "hilbert_artifact_failure":
			group.HilbertArtifactFailures++
		case "refusal_before_nd_proof":
			group.OutputRefusals++
		case "format_before_nd_proof":
			group.OutputFormatFailures++
		case "request_failure":
			group.RequestFailures++
		case "":
			// Explicitly successful or otherwise neutral rows still contribute to
			// total cases, but they are not unknown pipeline failures.
		default:
			group.UnknownStageRows++
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	aggregates := make([]benchmarkNDPipelineTraceAggregate, 0, len(groups))
	for _, group := range groups {
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkNDPipelineTraceAggregate) int {
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
		return strings.Compare(a.Surface, b.Surface)
	})
	return aggregates
}

func benchmarkNDPipelineStageImpl(row benchmarkResultRow) string {
	if pipelineStage := strings.TrimSpace(row.PipelineStage); pipelineStage != "" {
		switch pipelineStage {
		case "nd_proof_object":
			return "nd_proof_object_failure"
		case "lowering":
			return "lowering_failure"
		case "hilbert_artifact":
			if strings.TrimSpace(row.SchemaStatus) == "fail" || strings.TrimSpace(row.ParseStatus) == "fail" || strings.TrimSpace(row.KernelStatus) == "reject" {
				return "hilbert_artifact_failure"
			}
			return ""
		case "llm_output":
			switch strings.TrimSpace(row.ScoreBucket) {
			case "false_refusal":
				return "refusal_before_nd_proof"
			case "format_failure":
				return "format_before_nd_proof"
			case "request_failure":
				return "request_failure"
			default:
				return ""
			}
		}
	}
	switch strings.TrimSpace(row.ScoreBucket) {
	case "request_failure":
		return "request_failure"
	case "false_refusal":
		return "refusal_before_nd_proof"
	case "format_failure":
		if strings.TrimSpace(row.RawOutputKind) == "other" || strings.TrimSpace(row.RawOutputKind) == "invalid_json" {
			return "format_before_nd_proof"
		}
	}
	switch strings.TrimSpace(row.RawOutputKind) {
	case "not_derivable":
		// Legacy ND rows recorded an early refusal as a final raw output kind even
		// when the benchmark score bucket remained "pass" for a correct negative.
		return "refusal_before_nd_proof"
	case "proof_object":
		if strings.TrimSpace(row.SchemaStatus) == "fail" {
			return "nd_proof_object_failure"
		}
		if strings.TrimSpace(row.ParseStatus) == "fail" || strings.TrimSpace(row.KernelStatus) == "reject" {
			return "hilbert_artifact_failure"
		}
		if strings.TrimSpace(row.KernelStatus) == "not_run" && strings.TrimSpace(row.ScoreBucket) == "kernel_failure" {
			return "lowering_failure"
		}
	}
	if status := strings.TrimSpace(row.NDProofStatus); status != "" && status != "not_run" && status != "pass" {
		return "nd_proof_object_failure"
	}
	if status := strings.TrimSpace(row.LoweringStatus); status != "" && status != "not_run" && status != "pass" {
		return "lowering_failure"
	}
	if status := strings.TrimSpace(row.SchemaStatus); status == "fail" {
		return "hilbert_artifact_failure"
	}
	if status := strings.TrimSpace(row.ParseStatus); status == "fail" {
		return "hilbert_artifact_failure"
	}
	if status := strings.TrimSpace(row.KernelStatus); status == "reject" {
		return "hilbert_artifact_failure"
	}
	return "unknown"
}

func applyBenchmarkModelCatalogToNDPipelineTraceAggregatesImpl(aggregates []benchmarkNDPipelineTraceAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
		aggregates[idx].SizeBucket = entry.SizeBucket
		aggregates[idx].SizeLabel = entry.SizeLabel
	}
}

func loadBenchmarkCaseIndexImpl(path string) (map[string]benchmarkCase, error) {
	cases, err := loadBenchmarkCases(path)
	if err != nil {
		return nil, err
	}
	index := make(map[string]benchmarkCase, len(cases))
	for _, tc := range cases {
		index[tc.CaseID] = tc
	}
	return index, nil
}

func benchmarkObservedMetadataPathImpl(rows []benchmarkRunSummaryRow) string {
	for _, row := range rows {
		if value := strings.TrimSpace(row.TheoremsPath); value != "" {
			return resolveBenchmarkLocalPath(value)
		}
		if value := strings.TrimSpace(row.CasesPath); value != "" {
			return resolveBenchmarkLocalPath(value)
		}
	}
	return ""
}

func buildBenchmarkObservedDataImpl(projectFolder string, rows []benchmarkRunSummaryRow, modelCatalog []benchmarkModelCatalogRow) (*benchmarkObservedData, error) {
	resultRows, summaryByRun, warnings := loadBenchmarkResultRowsFromSummaryRows(rows)
	data := &benchmarkObservedData{
		Warnings:   warnings,
		ByCategory: make(map[string][]benchmarkResultSliceAggregate),
		HardCases:  make([]benchmarkHardCaseAudit, 0),
	}
	if len(resultRows) == 0 {
		return data, nil
	}

	catalogByModel := indexBenchmarkModelCatalog(modelCatalog)

	aggregate, err := aggregateBenchmarkResultRows(resultRows)
	if err != nil {
		return nil, err
	}
	data.Aggregate = aggregate
	applyBenchmarkModelCatalogToResultAggregates(data.Aggregate, catalogByModel)
	data.NDPipelineTrace = buildBenchmarkNDPipelineTraceAggregates(resultRows)
	applyBenchmarkModelCatalogToNDPipelineTraceAggregates(data.NDPipelineTrace, catalogByModel)

	entailedRows := make([]benchmarkResultRow, 0, len(resultRows))
	notEntailedRows := make([]benchmarkResultRow, 0, len(resultRows))
	byCategoryRows := make(map[string][]benchmarkResultRow)
	for _, row := range resultRows {
		switch strings.TrimSpace(row.ExpectedLabel) {
		case "entailed":
			entailedRows = append(entailedRows, row)
		case "not_entailed":
			notEntailedRows = append(notEntailedRows, row)
		}
		category := strings.TrimSpace(row.Category)
		if category != "" {
			byCategoryRows[category] = append(byCategoryRows[category], row)
		}
	}

	if data.Entailed, err = aggregateBenchmarkResultRows(entailedRows); err != nil {
		return nil, err
	}
	applyBenchmarkModelCatalogToResultAggregates(data.Entailed, catalogByModel)

	if data.NotEntailed, err = aggregateBenchmarkResultRows(notEntailedRows); err != nil {
		return nil, err
	}
	applyBenchmarkModelCatalogToResultAggregates(data.NotEntailed, catalogByModel)

	for _, category := range sortedBenchmarkResultSliceKeys(byCategoryRows) {
		aggregates, err := aggregateBenchmarkResultRows(byCategoryRows[category])
		if err != nil {
			return nil, err
		}
		applyBenchmarkModelCatalogToResultAggregates(aggregates, catalogByModel)
		data.ByCategory[category] = aggregates
	}

	metadata, err := loadBenchmarkPackMetadata(rows)
	if metadata == nil {
		data.Warnings = append(data.Warnings, "benchmark input metadata path is unavailable; hard-case audit was skipped")
		return data, nil
	}
	if err != nil {
		data.Warnings = append(data.Warnings, fmt.Sprintf("failed to load case metadata for hard-case audit: %s", err.Error()))
		return data, nil
	}
	caseIndex := metadata.CaseIndex
	hardestFailures, err := buildBenchmarkCaseFailureAggregates(caseIndex, resultRows, summaryByRun)
	if err != nil {
		return nil, err
	}
	data.HardestFailures = hardestFailures
	hardAuditIDs := benchmarkObservedHardAuditCaseIDs(projectFolder)
	if len(hardAuditIDs) > 0 {
		hardCases, warnings, err := buildBenchmarkHardCaseAudits(hardAuditIDs, caseIndex, resultRows, summaryByRun, catalogByModel)
		if err != nil {
			return nil, err
		}
		data.HardCases = hardCases
		data.Warnings = append(data.Warnings, warnings...)
	}
	return data, nil
}

func buildBenchmarkCaseFailureAggregatesImpl(caseIndex map[string]benchmarkCase, rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow) ([]benchmarkCaseFailureAggregate, error) {
	groups := make(map[string]*benchmarkCaseFailureAggregate)
	runMembership := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkResultRows(rows) {
		caseID := strings.TrimSpace(firstNonEmptyBenchmarkCell(row.CaseID, row.TheoremID))
		if caseID == "" {
			continue
		}
		group, ok := groups[caseID]
		if !ok {
			tc, ok := caseIndex[caseID]
			if !ok {
				tc = benchmarkCase{
					TheoremPackID: strings.TrimSpace(row.TheoremPackID),
					TheoremID:     strings.TrimSpace(row.TheoremID),
					CaseID:        caseID,
					Category:      strings.TrimSpace(row.Category),
					Label:         strings.TrimSpace(row.ExpectedLabel),
				}
			}
			group = &benchmarkCaseFailureAggregate{Case: tc}
			groups[caseID] = group
			runMembership[caseID] = make(map[string]struct{})
		}
		if _, ok := runMembership[caseID][row.RunID]; !ok {
			runMembership[caseID][row.RunID] = struct{}{}
			group.Runs++
		}
		group.CasesTotal++
		if strings.TrimSpace(row.ScoreBucket) == "pass" {
			group.PassCount++
		} else {
			group.FailureCount++
		}
		switch strings.TrimSpace(row.ScoreBucket) {
		case "false_refusal":
			group.FalseRefusalCount++
		case "false_accept":
			group.FalseAcceptCount++
		case "request_failure":
			group.RequestFailureCount++
		case "schema_failure":
			group.SchemaFailureCount++
		case "parse_failure":
			group.ParseFailureCount++
		case "kernel_failure":
			group.KernelFailureCount++
		case "contract_failure":
			group.ContractFailureCount++
		case "format_failure":
			group.FormatFailureCount++
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err != nil {
			return nil, fmt.Errorf("benchmark per-case row %q/%q has invalid timestamp_utc %q: %w", row.RunID, row.CaseID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
			if summary, ok := summaryByRun[strings.TrimSpace(row.RunID)]; ok {
				group.LatestResultPath = displayBenchmarkPath(summary.ResultsPath)
				group.LatestRawDir = displayBenchmarkPath(summary.RawDir)
			}
		}
	}

	aggregates := make([]benchmarkCaseFailureAggregate, 0, len(groups))
	for _, group := range groups {
		if group.FailureCount == 0 {
			continue
		}
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkCaseFailureAggregate) int {
		aPassRate := benchmarkPassRate(a.PassCount, a.CasesTotal)
		bPassRate := benchmarkPassRate(b.PassCount, b.CasesTotal)
		switch {
		case aPassRate < bPassRate:
			return -1
		case aPassRate > bPassRate:
			return 1
		}
		switch {
		case a.FalseAcceptCount > b.FalseAcceptCount:
			return -1
		case a.FalseAcceptCount < b.FalseAcceptCount:
			return 1
		}
		switch {
		case a.FailureCount > b.FailureCount:
			return -1
		case a.FailureCount < b.FailureCount:
			return 1
		}
		return strings.Compare(a.Case.CaseID, b.Case.CaseID)
	})
	return aggregates, nil
}

func buildBenchmarkHardCaseAuditsImpl(caseIDs []string, caseIndex map[string]benchmarkCase, rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow, catalog map[string]benchmarkModelCatalogRow) ([]benchmarkHardCaseAudit, []string, error) {
	audits := make([]benchmarkHardCaseAudit, 0, len(caseIDs))
	warnings := make([]string, 0)
	lineageIndex := make(map[string][]benchmarkCase)
	for _, tc := range caseIndex {
		sourceCaseID := strings.TrimSpace(tc.SourceCaseID)
		if sourceCaseID == "" || sourceCaseID == strings.TrimSpace(tc.CaseID) {
			continue
		}
		lineageIndex[sourceCaseID] = append(lineageIndex[sourceCaseID], tc)
	}
	for sourceCaseID := range lineageIndex {
		slices.SortFunc(lineageIndex[sourceCaseID], func(a, b benchmarkCase) int {
			return strings.Compare(strings.TrimSpace(a.CaseID), strings.TrimSpace(b.CaseID))
		})
	}
	for _, caseID := range caseIDs {
		tc, ok := caseIndex[caseID]
		targetCaseID := strings.TrimSpace(caseID)
		filtered := make([]benchmarkResultRow, 0, 8)
		observedCaseIDs := make([]string, 0)
		if ok {
			for _, row := range rows {
				if strings.TrimSpace(row.CaseID) == targetCaseID {
					filtered = append(filtered, row)
				}
			}
		} else if descendants := lineageIndex[targetCaseID]; len(descendants) > 0 {
			tc = descendants[0]
			descendantCaseIDs := make(map[string]struct{}, len(descendants))
			for _, descendant := range descendants {
				descendantCaseID := strings.TrimSpace(descendant.CaseID)
				if descendantCaseID == "" {
					continue
				}
				descendantCaseIDs[descendantCaseID] = struct{}{}
				observedCaseIDs = append(observedCaseIDs, descendantCaseID)
			}
			for _, row := range rows {
				if _, ok := descendantCaseIDs[strings.TrimSpace(row.CaseID)]; ok {
					filtered = append(filtered, row)
				}
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("hard-case audit target `%s` is missing from the selected case pack", targetCaseID))
			audits = append(audits, benchmarkHardCaseAudit{
				TargetCaseID: targetCaseID,
				Case:         benchmarkCase{CaseID: targetCaseID},
			})
			continue
		}
		aggregates, err := aggregateBenchmarkHardCaseRows(filtered, summaryByRun)
		if err != nil {
			return nil, warnings, err
		}
		applyBenchmarkModelCatalogToHardCaseAggregates(aggregates, catalog)
		audits = append(audits, benchmarkHardCaseAudit{
			TargetCaseID:    targetCaseID,
			Case:            tc,
			ObservedCaseIDs: observedCaseIDs,
			Aggregates:      aggregates,
		})
	}
	return audits, warnings, nil
}

func aggregateBenchmarkHardCaseRowsImpl(rows []benchmarkResultRow, summaryByRun map[string]benchmarkRunSummaryRow) ([]benchmarkHardCaseAuditAggregate, error) {
	groups := make(map[string]*benchmarkHardCaseAuditAggregate)
	runMembership := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkResultRows(rows) {
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkHardCaseAuditAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
				DisplayName:    benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
			}
			groups[key] = group
			runMembership[key] = make(map[string]struct{})
		}
		if _, ok := runMembership[key][row.RunID]; !ok {
			runMembership[key][row.RunID] = struct{}{}
			group.Runs++
		}
		group.CasesTotal++
		switch strings.TrimSpace(row.ScoreBucket) {
		case "pass":
			group.PassCount++
		case "false_refusal":
			group.FalseRefusalCount++
		case "false_accept":
			group.FalseAcceptCount++
		case "request_failure":
			group.RequestFailureCount++
		case "schema_failure":
			group.SchemaFailureCount++
		case "parse_failure":
			group.ParseFailureCount++
		case "kernel_failure":
			group.KernelFailureCount++
		case "contract_failure":
			group.ContractFailureCount++
		case "format_failure":
			group.FormatFailureCount++
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err != nil {
			return nil, fmt.Errorf("benchmark hard-case row %q/%q has invalid timestamp_utc %q: %w", row.RunID, row.CaseID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
			if summary, ok := summaryByRun[strings.TrimSpace(row.RunID)]; ok {
				group.LatestResultPath = displayBenchmarkPath(summary.ResultsPath)
				group.LatestRawDir = displayBenchmarkPath(summary.RawDir)
			}
		}
	}

	aggregates := make([]benchmarkHardCaseAuditAggregate, 0, len(groups))
	for _, group := range groups {
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkHardCaseAuditAggregate) int {
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
		return strings.Compare(a.Surface, b.Surface)
	})
	return aggregates, nil
}

func applyBenchmarkModelCatalogToHardCaseAggregatesImpl(aggregates []benchmarkHardCaseAuditAggregate, catalog map[string]benchmarkModelCatalogRow) {
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
