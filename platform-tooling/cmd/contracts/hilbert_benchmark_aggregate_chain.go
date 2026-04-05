package main

import (
	"slices"
	"strings"
	"time"
)

// Chain/compositional aggregation is intentionally isolated from summary and
// observed-slice logic so compositional protocol analysis can be read on its own.
func aggregateBenchmarkChainSummaryRowsImpl(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainSummaryAggregate {
	groups := make(map[string]*benchmarkChainSummaryAggregate)
	runMembership := make(map[string]map[string]struct{})
	observedStages := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkChainSummaryLoadedRows(rows) {
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkChainSummaryAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				DisplayName:    row.DisplayName,
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
			}
			groups[key] = group
			runMembership[key] = make(map[string]struct{})
			observedStages[key] = make(map[string]struct{})
		}
		if _, ok := runMembership[key][row.RunID]; !ok {
			runMembership[key][row.RunID] = struct{}{}
			group.Runs++
		}
		for _, stageID := range benchmarkLoadedChainStageIDs(row) {
			observedStages[key][stageID] = struct{}{}
		}
		group.ChainsTotal++
		if row.ImportStagePass {
			group.ImportStagePassCount++
		}
		if row.FinalCompositionPassGold {
			group.FinalCompositionPassGoldCount++
		}
		if row.FinalCompositionPassModel {
			group.FinalCompositionPassModelCount++
		}
		if row.ConditionalFinalPassGold {
			group.ConditionalFinalPassGoldCount++
		}
		if row.ConditionalFinalPassModel {
			group.ConditionalFinalPassModelCount++
		}
		if row.NegativeTwinPass {
			group.NegativeTwinPassCount++
		}
		if row.AllStagesPass {
			group.AllStagesPassCount++
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	result := make([]benchmarkChainSummaryAggregate, 0, len(groups))
	for key, group := range groups {
		if stageSet, ok := observedStages[key]; ok {
			group.ObservedStageIDs = benchmarkSortedChainStageKeys(stageSet)
		}
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkChainSummaryAggregate) int {
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ExperimentID, b.ExperimentID)
	})
	return result
}

func aggregateBenchmarkChainStageSummaryRowsImpl(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainStageAggregate {
	groups := make(map[string]*benchmarkChainStageAggregate)
	for _, row := range dedupeBenchmarkChainSummaryLoadedRows(rows) {
		for _, stageID := range benchmarkLoadedChainStageIDs(row) {
			key := strings.Join([]string{
				strings.TrimSpace(row.Model),
				strings.TrimSpace(row.LLMModel),
				strings.TrimSpace(row.ExperimentID),
				strings.TrimSpace(row.PromptVersion),
				strings.TrimSpace(row.Surface),
				stageID,
			}, "\x00")
			group, ok := groups[key]
			if !ok {
				group = &benchmarkChainStageAggregate{
					Model:          row.Model,
					LLMModel:       row.LLMModel,
					ExperimentName: row.ExperimentName,
					ExperimentID:   row.ExperimentID,
					DisplayName:    row.DisplayName,
					PromptVersion:  row.PromptVersion,
					Surface:        row.Surface,
					StageID:        stageID,
					StageRole:      strings.TrimSpace(row.StageRoles[stageID]),
				}
				groups[key] = group
			}
			group.Chains++
			if row.StagePasses[stageID] {
				group.PassCount++
			}
			timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
			if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
				group.LatestRunID = row.RunID
				group.LatestTimestampUTC = row.TimestampUTC
				group.latestTimestamp = timestamp
			}
		}
	}
	result := make([]benchmarkChainStageAggregate, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkChainStageAggregate) int {
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.StageID, b.StageID); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ExperimentID, b.ExperimentID)
	})
	return result
}

func aggregateBenchmarkChainFailuresImpl(rows []benchmarkChainSummaryLoadedRow) []benchmarkChainFailureAggregate {
	groups := make(map[string]*benchmarkChainFailureAggregate)
	for _, row := range dedupeBenchmarkChainSummaryLoadedRows(rows) {
		failureStage := strings.TrimSpace(row.FailureStage)
		failureStageRole := strings.TrimSpace(row.FailureStageRole)
		failureType := strings.TrimSpace(row.FailureType)
		failureClass := strings.TrimSpace(row.FailureClass)
		if failureStage == "" && failureType == "" && failureClass == "" {
			continue
		}
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
			failureStage,
			failureStageRole,
			failureType,
			failureClass,
			strings.TrimSpace(row.FailureDetail),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkChainFailureAggregate{
				Model:            row.Model,
				LLMModel:         row.LLMModel,
				ExperimentName:   row.ExperimentName,
				ExperimentID:     row.ExperimentID,
				DisplayName:      row.DisplayName,
				PromptVersion:    row.PromptVersion,
				Surface:          row.Surface,
				FailureStage:     failureStage,
				FailureStageRole: failureStageRole,
				FailureType:      failureType,
				FailureClass:     failureClass,
			}
			groups[key] = group
		}
		group.Chains++
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	result := make([]benchmarkChainFailureAggregate, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkChainFailureAggregate) int {
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.FailureClass, b.FailureClass); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.FailureStage, b.FailureStage); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ExperimentID, b.ExperimentID)
	})
	return result
}

func applyBenchmarkChainSummaryCatalogImpl(aggregates []benchmarkChainSummaryAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func applyBenchmarkChainStageCatalogImpl(aggregates []benchmarkChainStageAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func applyBenchmarkChainFailureCatalogImpl(aggregates []benchmarkChainFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func aggregateBenchmarkChainMetadataRowsImpl(rows []benchmarkChainSummaryLoadedRow, selector func(benchmarkChainSummaryLoadedRow) string) []benchmarkChainMetadataAggregate {
	groups := make(map[string]*benchmarkChainMetadataAggregate)
	for _, row := range dedupeBenchmarkChainSummaryLoadedRows(rows) {
		value := strings.TrimSpace(selector(row))
		if value == "" {
			value = "unknown"
		}
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
			value,
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkChainMetadataAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				DisplayName:    row.DisplayName,
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
				GroupValue:     value,
			}
			groups[key] = group
		}
		group.ChainsTotal++
		group.Runs++
		if row.AllStagesPass {
			group.AllStagesPassCount++
		}
		if row.FinalCompositionPassGold {
			group.GoldFinalPassCount++
		}
		if row.NegativeTwinPass {
			group.NegativeTwinPassCount++
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	result := make([]benchmarkChainMetadataAggregate, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkChainMetadataAggregate) int {
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.GroupValue, b.GroupValue); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ExperimentID, b.ExperimentID)
	})
	return result
}

func applyBenchmarkChainMetadataCatalogImpl(aggregates []benchmarkChainMetadataAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func aggregateBenchmarkFGFailureRowsImpl(rows []benchmarkResultRow) []benchmarkFGFailureAggregate {
	groups := make(map[string]*benchmarkFGFailureAggregate)
	for _, row := range dedupeBenchmarkResultRows(rows) {
		if strings.TrimSpace(row.StageID) != "fg" {
			continue
		}
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
			strings.TrimSpace(row.ScoreBucket),
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkFGFailureAggregate{
				Model:          row.Model,
				LLMModel:       row.LLMModel,
				ExperimentName: row.ExperimentName,
				ExperimentID:   row.ExperimentID,
				DisplayName:    benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
				PromptVersion:  row.PromptVersion,
				Surface:        row.Surface,
				FailureType:    strings.TrimSpace(row.ScoreBucket),
			}
			groups[key] = group
		}
		group.Cases++
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	result := make([]benchmarkFGFailureAggregate, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkFGFailureAggregate) int {
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.FailureType, b.FailureType); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ExperimentID, b.ExperimentID)
	})
	return result
}

func applyBenchmarkFGFailureCatalogImpl(aggregates []benchmarkFGFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func aggregateBenchmarkFGCaseFailuresImpl(rows []benchmarkResultRow) []benchmarkFGCaseFailureAggregate {
	groups := make(map[string]*benchmarkFGCaseFailureAggregate)
	runMembership := make(map[string]map[string]struct{})
	for _, row := range dedupeBenchmarkResultRows(rows) {
		if strings.TrimSpace(row.StageID) != "fg" {
			continue
		}
		caseID := strings.TrimSpace(firstNonEmptyBenchmarkCell(row.CaseID, row.TheoremID))
		if caseID == "" {
			continue
		}
		key := strings.Join([]string{
			strings.TrimSpace(row.Model),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.Surface),
			caseID,
		}, "\x00")
		group, ok := groups[key]
		if !ok {
			group = &benchmarkFGCaseFailureAggregate{
				Model:                row.Model,
				LLMModel:             row.LLMModel,
				ExperimentName:       row.ExperimentName,
				ExperimentID:         row.ExperimentID,
				DisplayName:          benchmarkDisplayNameWithExperiment(row.Model, row.ExperimentID),
				PromptVersion:        row.PromptVersion,
				Surface:              row.Surface,
				CaseID:               caseID,
				ChainID:              strings.TrimSpace(row.ChainID),
				TransitionGroup:      strings.TrimSpace(row.TransitionGroup),
				ImportArity:          strings.TrimSpace(row.ImportArity),
				ReuseShape:           strings.TrimSpace(row.ReuseShape),
				BridgeDepth:          strings.TrimSpace(row.BridgeDepth),
				SymbolOverlap:        strings.TrimSpace(row.SymbolOverlap),
				NegativeTwinHardness: strings.TrimSpace(row.NegativeTwinHardness),
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
		default:
			group.FailureCount++
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
		if err == nil && (group.LatestRunID == "" || timestamp.After(group.latestTimestamp)) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
		}
	}
	result := make([]benchmarkFGCaseFailureAggregate, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkFGCaseFailureAggregate) int {
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
		case a.CasesTotal > b.CasesTotal:
			return -1
		case a.CasesTotal < b.CasesTotal:
			return 1
		}
		if cmp := strings.Compare(a.PromptVersion, b.PromptVersion); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.LLMModel, b.LLMModel); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.CaseID, b.CaseID)
	})
	return result
}

func applyBenchmarkFGCaseFailureCatalogImpl(aggregates []benchmarkFGCaseFailureAggregate, catalog map[string]benchmarkModelCatalogRow) {
	for idx := range aggregates {
		entry, ok := catalog[strings.TrimSpace(aggregates[idx].LLMModel)]
		if !ok {
			continue
		}
		aggregates[idx].DisplayName = benchmarkDisplayNameWithExperiment(entry.DisplayName, aggregates[idx].ExperimentID)
	}
}

func buildBenchmarkChainSummaryDataImpl(rows []benchmarkRunSummaryRow, modelCatalog []benchmarkModelCatalogRow) (*benchmarkChainSummaryData, error) {
	metadata, metadataErr := loadBenchmarkPackMetadata(rows)
	explicitChainSummary := false
	for _, row := range dedupeBenchmarkRunSummaryRows(rows) {
		if strings.TrimSpace(resolveBenchmarkLocalPath(row.ChainSummaryPath)) != "" {
			explicitChainSummary = true
			break
		}
	}
	packUsesChainSummary := explicitChainSummary || (metadataErr == nil && metadata != nil && metadata.HasChainMetadata)
	if !packUsesChainSummary {
		return nil, nil
	}
	loaded, warnings := loadBenchmarkChainSummaryRowsFromSummaryRows(rows, metadataErr == nil && metadata != nil && metadata.HasChainMetadata)
	if len(loaded) == 0 && len(warnings) == 0 {
		return nil, nil
	}
	catalog := indexBenchmarkModelCatalog(modelCatalog)
	aggregates := aggregateBenchmarkChainSummaryRows(loaded)
	stageAggregates := aggregateBenchmarkChainStageSummaryRows(loaded)
	failures := aggregateBenchmarkChainFailures(loaded)
	transitionGroups := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.TransitionGroup })
	importArities := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.ImportArity })
	reuseShapes := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.ReuseShape })
	bridgeDepths := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.BridgeDepth })
	symbolOverlaps := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.SymbolOverlap })
	negativeTwinHardnesses := aggregateBenchmarkChainMetadataRows(loaded, func(row benchmarkChainSummaryLoadedRow) string { return row.NegativeTwinHardness })
	resultRows, _, resultWarnings := loadBenchmarkResultRowsFromSummaryRows(rows)
	warnings = append(warnings, resultWarnings...)
	fgFailureMix := aggregateBenchmarkFGFailureRows(resultRows)
	hardestFGFailures := aggregateBenchmarkFGCaseFailures(resultRows)
	applyBenchmarkChainSummaryCatalog(aggregates, catalog)
	applyBenchmarkChainStageCatalog(stageAggregates, catalog)
	applyBenchmarkChainFailureCatalog(failures, catalog)
	applyBenchmarkChainMetadataCatalog(transitionGroups, catalog)
	applyBenchmarkChainMetadataCatalog(importArities, catalog)
	applyBenchmarkChainMetadataCatalog(reuseShapes, catalog)
	applyBenchmarkChainMetadataCatalog(bridgeDepths, catalog)
	applyBenchmarkChainMetadataCatalog(symbolOverlaps, catalog)
	applyBenchmarkChainMetadataCatalog(negativeTwinHardnesses, catalog)
	applyBenchmarkFGFailureCatalog(fgFailureMix, catalog)
	applyBenchmarkFGCaseFailureCatalog(hardestFGFailures, catalog)
	return &benchmarkChainSummaryData{
		Warnings:               warnings,
		Aggregates:             aggregates,
		StageAggregates:        stageAggregates,
		Failures:               failures,
		TransitionGroups:       transitionGroups,
		ImportArities:          importArities,
		ReuseShapes:            reuseShapes,
		BridgeDepths:           bridgeDepths,
		SymbolOverlaps:         symbolOverlaps,
		NegativeTwinHardnesses: negativeTwinHardnesses,
		FGFailureMix:           fgFailureMix,
		HardestFGFailures:      hardestFGFailures,
	}, nil
}
