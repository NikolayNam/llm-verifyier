package main

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type benchmarkCaseReportAggregate struct {
	PackKey              string
	Case                 benchmarkCase
	Observations         int
	Runs                 int
	DistinctModels       int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatestRunID          string
	LatestTimestampUTC   string
	LatestResultPath     string
	LatestRawDir         string
	latestTimestamp      time.Time
}

type benchmarkCasePackOverview struct {
	PackKey              string
	DistinctCases        int
	Observations         int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
}

type benchmarkCaseReportData struct {
	Warnings          []string
	Aggregates        []benchmarkCaseReportAggregate
	Hardest           []benchmarkCaseReportAggregate
	PackOverviews     []benchmarkCasePackOverview
	DistinctPackCount int
	DistinctCaseCount int
	TotalObservations int
}

func renderBenchmarkCaseMarkdownReport(summaryPaths []string, projectFolder, casesFile string, rows []benchmarkRunSummaryRow, generatedAt time.Time) (string, error) {
	var b strings.Builder
	b.WriteString("# Hilbert Benchmark Case Report\n\n")
	b.WriteString("This report aggregates result rows at the case level. Unlike the summary-only meta report, it shows one consolidated row per selected case in the chosen pack slice.\n\n")
	b.WriteString(fmt.Sprintf("- Generated at (UTC): `%s`\n", generatedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Project folder: `%s`\n", projectFolder))
	b.WriteString(fmt.Sprintf("- %s filter: `%s`\n", benchmarkInputFilterLabel(projectFolder), casesFile))
	if len(summaryPaths) == 1 {
		b.WriteString(fmt.Sprintf("- Summary source: `%s`\n", displayBenchmarkPath(summaryPaths[0])))
	} else {
		b.WriteString(fmt.Sprintf("- Summary sources: `%d`\n", len(summaryPaths)))
		for _, path := range summaryPaths {
			b.WriteString(fmt.Sprintf("  - `%s`\n", displayBenchmarkPath(path)))
		}
	}
	b.WriteString("\n")

	if len(rows) == 0 {
		b.WriteString("No benchmark summary rows matched the selected filter.\n")
		return b.String(), nil
	}

	data, err := buildBenchmarkCaseReportData(rows)
	if err != nil {
		return "", err
	}
	if len(data.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range data.Warnings {
			b.WriteString(fmt.Sprintf("- %s\n", warning))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Overview\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("| --- | ---: |\n")
	b.WriteString(fmt.Sprintf("| Summary rows | %d |\n", len(rows)))
	b.WriteString(fmt.Sprintf("| Distinct packs | %d |\n", data.DistinctPackCount))
	b.WriteString(fmt.Sprintf("| Distinct cases | %d |\n", data.DistinctCaseCount))
	b.WriteString(fmt.Sprintf("| Result-row observations | %d |\n", data.TotalObservations))
	b.WriteString("\n")

	if len(data.PackOverviews) > 0 {
		b.WriteString("## Pack overview\n\n")
		b.WriteString("| Pack | Cases | Observations | Pass | Pass Rate | False Refusal | False Accept | Schema | Parse | Kernel | Contract | Format |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		for _, overview := range data.PackOverviews {
			b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d |\n",
				overview.PackKey,
				overview.DistinctCases,
				overview.Observations,
				overview.PassCount,
				benchmarkPassRate(overview.PassCount, overview.Observations),
				overview.FalseRefusalCount,
				overview.FalseAcceptCount,
				overview.SchemaFailureCount,
				overview.ParseFailureCount,
				overview.KernelFailureCount,
				overview.ContractFailureCount,
				overview.FormatFailureCount,
			))
		}
		b.WriteString("\n")
	}

	if len(data.Hardest) > 0 {
		b.WriteString("## Hardest observed cases\n\n")
		b.WriteString("Lowest-pass case aggregates across the selected observations. Use this as the fastest entrypoint before opening the full per-pack tables below.\n\n")
		b.WriteString("| Pack | Case | Category | Label | Difficulty | Goal | Observations | Pass | Pass Rate | Dominant non-pass | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | --- | --- |\n")
		for _, aggregate := range data.Hardest {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %d | %d | %.2f%% | `%s` | `%s` |\n",
				aggregate.PackKey,
				aggregate.Case.CaseID,
				valueOrNA(aggregate.Case.Category),
				valueOrNA(aggregate.Case.Label),
				valueOrNA(aggregate.Case.Difficulty),
				valueOrNA(aggregate.Case.Goal),
				aggregate.Observations,
				aggregate.PassCount,
				benchmarkPassRate(aggregate.PassCount, aggregate.Observations),
				benchmarkDominantCaseFailure(aggregate),
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Per-pack case tables\n\n")
	grouped := make(map[string][]benchmarkCaseReportAggregate)
	packOrder := make([]string, 0)
	for _, aggregate := range data.Aggregates {
		if _, ok := grouped[aggregate.PackKey]; !ok {
			packOrder = append(packOrder, aggregate.PackKey)
		}
		grouped[aggregate.PackKey] = append(grouped[aggregate.PackKey], aggregate)
	}
	slices.Sort(packOrder)
	for _, packKey := range packOrder {
		b.WriteString(fmt.Sprintf("### Pack: `%s`\n\n", packKey))
		b.WriteString("| Case | Category | Label | Difficulty | Goal | Observations | Runs | Models | Pass | Pass Rate | False Refusal | False Accept | Request | Schema | Parse | Kernel | Contract | Format | Dominant non-pass |\n")
		b.WriteString("| --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
		for _, aggregate := range grouped[packKey] {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %d | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | `%s` |\n",
				aggregate.Case.CaseID,
				valueOrNA(aggregate.Case.Category),
				valueOrNA(aggregate.Case.Label),
				valueOrNA(aggregate.Case.Difficulty),
				valueOrNA(aggregate.Case.Goal),
				aggregate.Observations,
				aggregate.Runs,
				aggregate.DistinctModels,
				aggregate.PassCount,
				benchmarkPassRate(aggregate.PassCount, aggregate.Observations),
				aggregate.FalseRefusalCount,
				aggregate.FalseAcceptCount,
				aggregate.RequestFailureCount,
				aggregate.SchemaFailureCount,
				aggregate.ParseFailureCount,
				aggregate.KernelFailureCount,
				aggregate.ContractFailureCount,
				aggregate.FormatFailureCount,
				benchmarkDominantCaseFailure(aggregate),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Notes\n\n")
	b.WriteString("- This report is result-row driven. It is heavier than `report meta`, but much easier to scan by `case_id` than the full forensic benchmark report.\n")
	b.WriteString("- Aggregation key is `pack + case_id`, so the same `case_id` from different packs is not merged accidentally.\n")
	return b.String(), nil
}

func buildBenchmarkCaseReportData(rows []benchmarkRunSummaryRow) (*benchmarkCaseReportData, error) {
	resultRows, summaryByRun, warnings := loadBenchmarkResultRowsFromSummaryRows(rows)
	data := &benchmarkCaseReportData{Warnings: warnings}
	if len(resultRows) == 0 {
		return data, nil
	}

	caseIndexes, metadataWarnings := loadBenchmarkCaseIndexesByPack(rows)
	data.Warnings = append(data.Warnings, metadataWarnings...)

	type key struct {
		pack   string
		caseID string
	}
	groups := make(map[key]*benchmarkCaseReportAggregate)
	runMembership := make(map[key]map[string]struct{})
	modelMembership := make(map[key]map[string]struct{})
	distinctCases := make(map[key]struct{})

	for _, row := range dedupeBenchmarkResultRows(resultRows) {
		caseID := strings.TrimSpace(firstNonEmptyBenchmarkCell(row.CaseID, row.TheoremID))
		if caseID == "" {
			continue
		}
		summary, ok := summaryByRun[strings.TrimSpace(row.RunID)]
		packKey := "unknown"
		if ok {
			if value := strings.TrimSpace(benchmarkRunSummaryInputFile(summary, "", strings.TrimSpace(summary.ProjectFolder))); value != "" {
				packKey = value
			}
		}
		groupKey := key{pack: packKey, caseID: caseID}
		group, exists := groups[groupKey]
		if !exists {
			tc, ok := caseIndexes[packKey][caseID]
			if !ok {
				tc = benchmarkCase{
					TheoremPackID: strings.TrimSpace(row.TheoremPackID),
					TheoremID:     strings.TrimSpace(row.TheoremID),
					CaseID:        caseID,
					Category:      strings.TrimSpace(row.Category),
					Label:         strings.TrimSpace(row.ExpectedLabel),
				}
			}
			group = &benchmarkCaseReportAggregate{
				PackKey: packKey,
				Case:    tc,
			}
			groups[groupKey] = group
			runMembership[groupKey] = make(map[string]struct{})
			modelMembership[groupKey] = make(map[string]struct{})
		}
		if _, ok := runMembership[groupKey][row.RunID]; !ok {
			runMembership[groupKey][row.RunID] = struct{}{}
			group.Runs++
		}
		modelKey := strings.TrimSpace(firstNonEmptyBenchmarkCell(row.LLMModel, row.Model))
		if modelKey != "" {
			if _, ok := modelMembership[groupKey][modelKey]; !ok {
				modelMembership[groupKey][modelKey] = struct{}{}
				group.DistinctModels++
			}
		}
		group.Observations++
		data.TotalObservations++
		distinctCases[groupKey] = struct{}{}

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
			return nil, fmt.Errorf("benchmark case report row %q/%q has invalid timestamp_utc %q: %w", row.RunID, row.CaseID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.LatestTimestampUTC = row.TimestampUTC
			group.latestTimestamp = timestamp
			if ok {
				group.LatestResultPath = displayBenchmarkPath(summary.ResultsPath)
				group.LatestRawDir = displayBenchmarkPath(summary.RawDir)
			}
		}
	}

	data.DistinctCaseCount = len(distinctCases)
	packSeen := make(map[string]struct{})
	for _, group := range groups {
		data.Aggregates = append(data.Aggregates, *group)
		packSeen[group.PackKey] = struct{}{}
	}
	data.DistinctPackCount = len(packSeen)
	slices.SortFunc(data.Aggregates, func(a, b benchmarkCaseReportAggregate) int {
		switch cmp := strings.Compare(a.PackKey, b.PackKey); {
		case cmp < 0:
			return -1
		case cmp > 0:
			return 1
		}
		switch cmp := strings.Compare(strings.TrimSpace(a.Case.Category), strings.TrimSpace(b.Case.Category)); {
		case cmp < 0:
			return -1
		case cmp > 0:
			return 1
		}
		return strings.Compare(strings.TrimSpace(a.Case.CaseID), strings.TrimSpace(b.Case.CaseID))
	})

	data.PackOverviews = buildBenchmarkCasePackOverviews(data.Aggregates)
	data.Hardest = buildBenchmarkHardestCaseAggregates(data.Aggregates, 20)
	return data, nil
}

func loadBenchmarkCaseIndexesByPack(rows []benchmarkRunSummaryRow) (map[string]map[string]benchmarkCase, []string) {
	result := make(map[string]map[string]benchmarkCase)
	warnings := make([]string, 0)
	for _, row := range dedupeBenchmarkRunSummaryRows(rows) {
		packKey := strings.TrimSpace(benchmarkRunSummaryInputFile(row, "", strings.TrimSpace(row.ProjectFolder)))
		if packKey == "" {
			continue
		}
		if _, ok := result[packKey]; ok {
			continue
		}
		metadataPath := ""
		if value := strings.TrimSpace(row.TheoremsPath); value != "" {
			metadataPath = resolveBenchmarkLocalPath(value)
		} else if value := strings.TrimSpace(row.CasesPath); value != "" {
			metadataPath = resolveBenchmarkLocalPath(value)
		}
		if strings.TrimSpace(metadataPath) == "" {
			warnings = append(warnings, fmt.Sprintf("benchmark input metadata path is unavailable for pack `%s`; goal/comment columns may be blank", packKey))
			continue
		}
		caseIndex, err := loadBenchmarkCaseIndex(metadataPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to load case metadata for pack `%s`: %s", packKey, err.Error()))
			continue
		}
		result[packKey] = caseIndex
	}
	return result, warnings
}

func buildBenchmarkCasePackOverviews(aggregates []benchmarkCaseReportAggregate) []benchmarkCasePackOverview {
	groups := make(map[string]*benchmarkCasePackOverview)
	for _, aggregate := range aggregates {
		group, ok := groups[aggregate.PackKey]
		if !ok {
			group = &benchmarkCasePackOverview{PackKey: aggregate.PackKey}
			groups[aggregate.PackKey] = group
		}
		group.DistinctCases++
		group.Observations += aggregate.Observations
		group.PassCount += aggregate.PassCount
		group.FalseRefusalCount += aggregate.FalseRefusalCount
		group.FalseAcceptCount += aggregate.FalseAcceptCount
		group.SchemaFailureCount += aggregate.SchemaFailureCount
		group.ParseFailureCount += aggregate.ParseFailureCount
		group.KernelFailureCount += aggregate.KernelFailureCount
		group.ContractFailureCount += aggregate.ContractFailureCount
		group.FormatFailureCount += aggregate.FormatFailureCount
	}
	result := make([]benchmarkCasePackOverview, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b benchmarkCasePackOverview) int {
		return strings.Compare(a.PackKey, b.PackKey)
	})
	return result
}

func buildBenchmarkHardestCaseAggregates(aggregates []benchmarkCaseReportAggregate, limit int) []benchmarkCaseReportAggregate {
	ordered := append([]benchmarkCaseReportAggregate(nil), aggregates...)
	slices.SortFunc(ordered, func(a, b benchmarkCaseReportAggregate) int {
		aRate := benchmarkPassRate(a.PassCount, a.Observations)
		bRate := benchmarkPassRate(b.PassCount, b.Observations)
		switch {
		case aRate < bRate:
			return -1
		case aRate > bRate:
			return 1
		case a.FalseAcceptCount > b.FalseAcceptCount:
			return -1
		case a.FalseAcceptCount < b.FalseAcceptCount:
			return 1
		case a.Observations > b.Observations:
			return -1
		case a.Observations < b.Observations:
			return 1
		}
		switch cmp := strings.Compare(a.PackKey, b.PackKey); {
		case cmp < 0:
			return -1
		case cmp > 0:
			return 1
		default:
			return strings.Compare(a.Case.CaseID, b.Case.CaseID)
		}
	})
	if limit > 0 && len(ordered) > limit {
		return ordered[:limit]
	}
	return ordered
}

func benchmarkDominantCaseFailure(aggregate benchmarkCaseReportAggregate) string {
	type failureCount struct {
		label string
		count int
	}
	counts := []failureCount{
		{label: "false_accept", count: aggregate.FalseAcceptCount},
		{label: "false_refusal", count: aggregate.FalseRefusalCount},
		{label: "kernel_failure", count: aggregate.KernelFailureCount},
		{label: "parse_failure", count: aggregate.ParseFailureCount},
		{label: "schema_failure", count: aggregate.SchemaFailureCount},
		{label: "contract_failure", count: aggregate.ContractFailureCount},
		{label: "request_failure", count: aggregate.RequestFailureCount},
		{label: "format_failure", count: aggregate.FormatFailureCount},
	}
	best := failureCount{label: "pass_only", count: 0}
	for _, item := range counts {
		if item.count > best.count {
			best = item
		}
	}
	if best.count == 0 {
		return "pass_only"
	}
	return fmt.Sprintf("%s (%d)", best.label, best.count)
}
