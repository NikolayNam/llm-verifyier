package main

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type benchmarkMetaTotals struct {
	SummaryRows          int
	DistinctRunIDs       int
	DistinctLLMModels    int
	DistinctPrompts      int
	DistinctSurfaces     int
	DistinctCasePacks    int
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatencyCasesTotal    int
	LatencySumMS         float64
	MaxLatencyMS         int64
	TimedRuns            int
	RunElapsedTotalS     int64
	MaxRunElapsedS       int64
}

type benchmarkMetaSurfaceAggregate struct {
	Surface              string
	Runs                 int
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatencyCasesTotal    int
	LatencySumMS         float64
	MaxLatencyMS         int64
	LatestRunID          string
	latestTimestamp      time.Time
}

type benchmarkMetaPackSummary struct {
	CasesFile  string
	Rows       []benchmarkRunSummaryRow
	Aggregates []benchmarkRunSummaryAggregate
	Hypotheses []string
	Surfaces   []string
	TotalRuns  int
	BestSlice  benchmarkRunSummaryAggregate
	HasBest    bool
}

func renderBenchmarkMetaMarkdownReport(summaryPaths []string, projectFolder, casesFile, modelCatalogPath string, rows []benchmarkRunSummaryRow, modelCatalog []benchmarkModelCatalogRow, generatedAt time.Time) (string, error) {
	var b strings.Builder
	packLabel := "Case packs"
	packItemLabel := "Case Pack"
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		packLabel = "Theorem packs"
		packItemLabel = "Theorem Pack"
	}

	b.WriteString("# Hilbert Benchmark Meta Report\n\n")
	b.WriteString("This report stays on the summary layer only. It intentionally excludes observed result-row slices, hard-case audits, and chain sidecars so that multiple summary CSVs can be scanned quickly.\n\n")
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
	if strings.TrimSpace(modelCatalogPath) != "" {
		b.WriteString(fmt.Sprintf("- Model catalog: `%s`\n", displayBenchmarkPath(modelCatalogPath)))
	}
	b.WriteString("\n")

	if len(rows) == 0 {
		b.WriteString("No benchmark summary rows matched the selected filter.\n")
		return b.String(), nil
	}

	aggregates, err := aggregateBenchmarkRunSummary(rows)
	if err != nil {
		return "", err
	}
	catalogByModel := indexBenchmarkModelCatalog(modelCatalog)
	applyBenchmarkModelCatalog(aggregates, catalogByModel)

	totals, err := buildBenchmarkMetaTotals(rows)
	if err != nil {
		return "", err
	}
	packs, err := buildBenchmarkMetaPackSummaries(rows, catalogByModel)
	if err != nil {
		return "", err
	}
	surfaceAggregates, err := buildBenchmarkMetaSurfaceAggregates(rows)
	if err != nil {
		return "", err
	}

	b.WriteString("## Overview\n\n")
	renderBenchmarkMetaScopeSection(&b, totals)
	renderBenchmarkMetaTotalsSection(&b, totals)

	b.WriteString("## Top recorded slices\n\n")
	b.WriteString("Highest-pass summary aggregates across the selected summary rows. This is the fastest way to see which model/prompt slices are currently leading without opening the heavier per-pack report.\n\n")
	renderBenchmarkMetaTopSlicesTable(&b, aggregates, 12)

	if len(surfaceAggregates) > 1 {
		b.WriteString("## Surface summary\n\n")
		b.WriteString("Summary-row totals grouped only by benchmark surface.\n\n")
		renderBenchmarkMetaSurfaceTable(&b, surfaceAggregates)
	}

	b.WriteString("## Pack overview\n\n")
	b.WriteString(fmt.Sprintf("One line per %s, with the strongest recorded slice for that pack.\n\n", strings.ToLower(packItemLabel)))
	renderBenchmarkMetaPackOverviewTable(&b, packs)

	if pairs := buildBenchmarkResearchPairComparisons(rows); len(pairs) > 0 {
		b.WriteString("## Direct vs ND pair snapshot\n\n")
		for _, pair := range pairs {
			finding, err := renderBenchmarkResearchPairFinding(pair)
			if err != nil {
				return "", err
			}
			b.WriteString(finding)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Per-pack details\n\n")
	for _, pack := range packs {
		if err := renderBenchmarkMetaPackDetailSection(&b, packItemLabel, pack); err != nil {
			return "", err
		}
	}

	b.WriteString("## Notes\n\n")
	b.WriteString(fmt.Sprintf("- %s: `%d`\n", packLabel, len(packs)))
	b.WriteString("- This report is intentionally summary-driven. If a pack needs failure forensics, use `report benchmark` or `report research` on the same summary sources.\n")
	return b.String(), nil
}

func buildBenchmarkMetaTotals(rows []benchmarkRunSummaryRow) (benchmarkMetaTotals, error) {
	var totals benchmarkMetaTotals
	runIDs := make(map[string]struct{}, len(rows))
	llmModels := make(map[string]struct{}, len(rows))
	prompts := make(map[string]struct{}, len(rows))
	surfaces := make(map[string]struct{}, len(rows))
	packs := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		stats, err := benchmarkResearchStatsFromRow(row)
		if err != nil {
			return totals, err
		}
		totals.SummaryRows++
		if value := strings.TrimSpace(row.RunID); value != "" {
			runIDs[value] = struct{}{}
		}
		if value := strings.TrimSpace(row.LLMModel); value != "" {
			llmModels[value] = struct{}{}
		}
		if value := strings.TrimSpace(row.PromptVersion); value != "" {
			prompts[value] = struct{}{}
		}
		if value := strings.TrimSpace(row.Surface); value != "" {
			surfaces[value] = struct{}{}
		}
		if value := strings.TrimSpace(benchmarkRunSummaryInputFile(row, "", strings.TrimSpace(row.ProjectFolder))); value != "" {
			packs[value] = struct{}{}
		}
		totals.CasesTotal += stats.CasesTotal
		totals.PassCount += stats.PassCount
		totals.FalseRefusalCount += stats.FalseRefusalCount
		totals.FalseAcceptCount += stats.FalseAcceptCount
		totals.RequestFailureCount += stats.RequestFailureCount
		totals.SchemaFailureCount += stats.SchemaFailureCount
		totals.ParseFailureCount += stats.ParseFailureCount
		totals.KernelFailureCount += stats.KernelFailureCount
		totals.ContractFailureCount += stats.ContractFailureCount
		totals.FormatFailureCount += stats.FormatFailureCount
		totals.LatencyCasesTotal += stats.CasesTotal
		totals.LatencySumMS += stats.AvgLatencyMS * float64(stats.CasesTotal)
		if stats.MaxLatencyMS > totals.MaxLatencyMS {
			totals.MaxLatencyMS = stats.MaxLatencyMS
		}
		totals.TimedRuns++
		totals.RunElapsedTotalS += stats.RunElapsedSeconds
		if stats.RunElapsedSeconds > totals.MaxRunElapsedS {
			totals.MaxRunElapsedS = stats.RunElapsedSeconds
		}
	}
	totals.DistinctRunIDs = len(runIDs)
	totals.DistinctLLMModels = len(llmModels)
	totals.DistinctPrompts = len(prompts)
	totals.DistinctSurfaces = len(surfaces)
	totals.DistinctCasePacks = len(packs)
	return totals, nil
}

func buildBenchmarkMetaSurfaceAggregates(rows []benchmarkRunSummaryRow) ([]benchmarkMetaSurfaceAggregate, error) {
	groups := make(map[string]*benchmarkMetaSurfaceAggregate)
	for _, row := range rows {
		surface := strings.TrimSpace(row.Surface)
		if surface == "" {
			surface = "unknown"
		}
		group, ok := groups[surface]
		if !ok {
			group = &benchmarkMetaSurfaceAggregate{Surface: surface}
			groups[surface] = group
		}
		stats, err := benchmarkResearchStatsFromRow(row)
		if err != nil {
			return nil, err
		}
		group.Runs++
		group.CasesTotal += stats.CasesTotal
		group.PassCount += stats.PassCount
		group.FalseRefusalCount += stats.FalseRefusalCount
		group.FalseAcceptCount += stats.FalseAcceptCount
		group.RequestFailureCount += stats.RequestFailureCount
		group.SchemaFailureCount += stats.SchemaFailureCount
		group.ParseFailureCount += stats.ParseFailureCount
		group.KernelFailureCount += stats.KernelFailureCount
		group.ContractFailureCount += stats.ContractFailureCount
		group.FormatFailureCount += stats.FormatFailureCount
		group.LatencyCasesTotal += stats.CasesTotal
		group.LatencySumMS += stats.AvgLatencyMS * float64(stats.CasesTotal)
		if stats.MaxLatencyMS > group.MaxLatencyMS {
			group.MaxLatencyMS = stats.MaxLatencyMS
		}
		timestamp, err := time.Parse(time.RFC3339, row.TimestampUTC)
		if err != nil {
			return nil, fmt.Errorf("benchmark summary row %q has invalid timestamp_utc %q: %w", row.RunID, row.TimestampUTC, err)
		}
		if group.LatestRunID == "" || timestamp.After(group.latestTimestamp) {
			group.LatestRunID = row.RunID
			group.latestTimestamp = timestamp
		}
	}
	aggregates := make([]benchmarkMetaSurfaceAggregate, 0, len(groups))
	for _, group := range groups {
		aggregates = append(aggregates, *group)
	}
	slices.SortFunc(aggregates, func(a, b benchmarkMetaSurfaceAggregate) int {
		aRate := benchmarkPassRate(a.PassCount, a.CasesTotal)
		bRate := benchmarkPassRate(b.PassCount, b.CasesTotal)
		switch {
		case aRate > bRate:
			return -1
		case aRate < bRate:
			return 1
		case a.CasesTotal > b.CasesTotal:
			return -1
		case a.CasesTotal < b.CasesTotal:
			return 1
		default:
			return strings.Compare(a.Surface, b.Surface)
		}
	})
	return aggregates, nil
}

func buildBenchmarkMetaPackSummaries(rows []benchmarkRunSummaryRow, catalog map[string]benchmarkModelCatalogRow) ([]benchmarkMetaPackSummary, error) {
	byPack := make(map[string][]benchmarkRunSummaryRow)
	for _, row := range rows {
		pack := benchmarkRunSummaryInputFile(row, "", strings.TrimSpace(row.ProjectFolder))
		if pack == "" {
			pack = defaultBenchmarkInputFile(strings.TrimSpace(row.ProjectFolder))
		}
		byPack[pack] = append(byPack[pack], row)
	}
	packKeys := make([]string, 0, len(byPack))
	for pack := range byPack {
		packKeys = append(packKeys, pack)
	}
	slices.Sort(packKeys)

	summaries := make([]benchmarkMetaPackSummary, 0, len(packKeys))
	for _, pack := range packKeys {
		packRows := byPack[pack]
		aggregates, err := aggregateBenchmarkRunSummary(packRows)
		if err != nil {
			return nil, err
		}
		applyBenchmarkModelCatalog(aggregates, catalog)
		bestSlice, hasBest := benchmarkBestAggregate(aggregates)
		summary := benchmarkMetaPackSummary{
			CasesFile:  pack,
			Rows:       packRows,
			Aggregates: sortBenchmarkAggregatesByMetaPriority(aggregates),
			Hypotheses: distinctBenchmarkHypotheses(packRows),
			Surfaces:   distinctBenchmarkPackSurfaces(packRows),
			TotalRuns:  len(packRows),
			BestSlice:  bestSlice,
			HasBest:    hasBest,
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func renderBenchmarkMetaScopeSection(b *strings.Builder, totals benchmarkMetaTotals) {
	b.WriteString("### Scope\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("| --- | ---: |\n")
	b.WriteString(fmt.Sprintf("| Recorded summary rows | %d |\n", totals.SummaryRows))
	b.WriteString(fmt.Sprintf("| Distinct run ids | %d |\n", totals.DistinctRunIDs))
	b.WriteString(fmt.Sprintf("| Distinct `llm_model` | %d |\n", totals.DistinctLLMModels))
	b.WriteString(fmt.Sprintf("| Distinct prompt versions | %d |\n", totals.DistinctPrompts))
	b.WriteString(fmt.Sprintf("| Distinct surfaces | %d |\n", totals.DistinctSurfaces))
	b.WriteString(fmt.Sprintf("| Distinct packs | %d |\n", totals.DistinctCasePacks))
	b.WriteString("\n")
}

func renderBenchmarkMetaTotalsSection(b *strings.Builder, totals benchmarkMetaTotals) {
	passRate := benchmarkPassRate(totals.PassCount, totals.CasesTotal)
	avgLatencyMS := 0.0
	if totals.LatencyCasesTotal > 0 {
		avgLatencyMS = totals.LatencySumMS / float64(totals.LatencyCasesTotal)
	}
	avgRunElapsedS := 0.0
	if totals.TimedRuns > 0 {
		avgRunElapsedS = float64(totals.RunElapsedTotalS) / float64(totals.TimedRuns)
	}
	b.WriteString("### Outcome totals\n\n")
	b.WriteString("| Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Avg Run Elapsed (s) | Max Run Elapsed (s) |\n")
	b.WriteString("| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	b.WriteString(fmt.Sprintf("| %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %s | %d | %s | %d |\n\n",
		totals.CasesTotal,
		totals.PassCount,
		passRate,
		totals.FalseRefusalCount,
		totals.FalseAcceptCount,
		totals.RequestFailureCount,
		totals.SchemaFailureCount,
		totals.ParseFailureCount,
		totals.KernelFailureCount,
		totals.ContractFailureCount,
		totals.FormatFailureCount,
		formatBenchmarkFloat(avgLatencyMS),
		totals.MaxLatencyMS,
		formatBenchmarkFloat(avgRunElapsedS),
		totals.MaxRunElapsedS,
	))
}

func renderBenchmarkMetaTopSlicesTable(b *strings.Builder, aggregates []benchmarkRunSummaryAggregate, maxRows int) {
	if len(aggregates) == 0 {
		b.WriteString("No matching summary aggregates.\n\n")
		return
	}
	ordered := sortBenchmarkAggregatesByMetaPriority(aggregates)
	limit := len(ordered)
	if maxRows > 0 && limit > maxRows {
		limit = maxRows
	}
	renderBenchmarkSectionContext(
		b,
		benchmarkCollectDistinct(ordered[:limit], func(item benchmarkRunSummaryAggregate) string { return item.PromptVersion }),
		benchmarkCollectDistinct(ordered[:limit], func(item benchmarkRunSummaryAggregate) string { return item.CertificateVersion }),
		benchmarkLatestTimestampFrom(ordered[:limit], func(item benchmarkRunSummaryAggregate) string { return item.LatestTimestampUTC }),
	)
	b.WriteString("| Model | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema | Parse | Kernel | Contract | Format | Latest Run (UTC) |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, aggregate := range ordered[:limit] {
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | `%s` |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.PassCount,
			benchmarkAggregatePassRate(aggregate),
			aggregate.FalseRefusalCount,
			aggregate.FalseAcceptCount,
			aggregate.RequestFailureCount,
			aggregate.SchemaFailureCount,
			aggregate.ParseFailureCount,
			aggregate.KernelFailureCount,
			aggregate.ContractFailureCount,
			aggregate.FormatFailureCount,
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
		))
	}
	b.WriteString("\n")
	if limit < len(ordered) {
		b.WriteString(fmt.Sprintf("- Showing the top `%d` slices out of `%d` aggregated summary rows.\n\n", limit, len(ordered)))
	}
}

func renderBenchmarkMetaSurfaceTable(b *strings.Builder, aggregates []benchmarkMetaSurfaceAggregate) {
	if len(aggregates) == 0 {
		b.WriteString("No surface-level summary rows were available.\n\n")
		return
	}
	b.WriteString("| Surface | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema | Parse | Kernel | Contract | Format | Avg Latency (ms) | Max Latency (ms) | Latest Run (UTC) |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, aggregate := range aggregates {
		avgLatencyMS := 0.0
		if aggregate.LatencyCasesTotal > 0 {
			avgLatencyMS = aggregate.LatencySumMS / float64(aggregate.LatencyCasesTotal)
		}
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %s | %d | `%s` |\n",
			valueOrNA(aggregate.Surface),
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.PassCount,
			benchmarkPassRate(aggregate.PassCount, aggregate.CasesTotal),
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
			func() string {
				if aggregate.latestTimestamp.IsZero() {
					return "n/a"
				}
				return displayBenchmarkTimestamp(aggregate.latestTimestamp.UTC().Format(time.RFC3339))
			}(),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkMetaPackOverviewTable(b *strings.Builder, packs []benchmarkMetaPackSummary) {
	if len(packs) == 0 {
		b.WriteString("No pack-level summary rows were available.\n\n")
		return
	}
	showSurfaceColumn := false
	for _, pack := range packs {
		if len(pack.Surfaces) > 1 {
			showSurfaceColumn = true
			break
		}
	}
	if !showSurfaceColumn {
		if _, ok := benchmarkSingleDistinctValue(packs, func(item benchmarkMetaPackSummary) string {
			if len(item.Surfaces) == 1 {
				return item.Surfaces[0]
			}
			return ""
		}); !ok {
			showSurfaceColumn = true
		}
	}
	if showSurfaceColumn {
		b.WriteString("| Pack | Summary Rows | Surfaces | Best Slice | Pass Rate | Runs | Cases | Hypotheses |\n")
		b.WriteString("| --- | ---: | --- | --- | ---: | ---: | ---: | ---: |\n")
	} else {
		b.WriteString("| Pack | Summary Rows | Best Slice | Pass Rate | Runs | Cases | Hypotheses |\n")
		b.WriteString("| --- | ---: | --- | ---: | ---: | ---: | ---: |\n")
	}
	for _, pack := range packs {
		bestSlice := "n/a"
		bestRate := "n/a"
		bestRuns := 0
		bestCases := 0
		if pack.HasBest {
			bestSlice = fmt.Sprintf("`%s` / `%s`", benchmarkReportModelLabel(pack.BestSlice.DisplayName, pack.BestSlice.LLMModel, pack.BestSlice.Model), pack.BestSlice.PromptVersion)
			bestRate = fmt.Sprintf("%.2f%%", benchmarkAggregatePassRate(pack.BestSlice))
			bestRuns = pack.BestSlice.Runs
			bestCases = pack.BestSlice.CasesTotal
		}
		if showSurfaceColumn {
			b.WriteString(fmt.Sprintf("| `%s` | %d | `%s` | %s | %s | %d | %d | %d |\n",
				pack.CasesFile,
				pack.TotalRuns,
				strings.Join(pack.Surfaces, "`, `"),
				bestSlice,
				bestRate,
				bestRuns,
				bestCases,
				len(pack.Hypotheses),
			))
			continue
		}
		b.WriteString(fmt.Sprintf("| `%s` | %d | %s | %s | %d | %d | %d |\n",
			pack.CasesFile,
			pack.TotalRuns,
			bestSlice,
			bestRate,
			bestRuns,
			bestCases,
			len(pack.Hypotheses),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkMetaPackDetailSection(b *strings.Builder, packItemLabel string, pack benchmarkMetaPackSummary) error {
	b.WriteString(fmt.Sprintf("### %s: `%s`\n\n", packItemLabel, pack.CasesFile))
	b.WriteString(fmt.Sprintf("- Recorded summary rows: `%d`\n", pack.TotalRuns))
	if len(pack.Surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n", pack.Surfaces[0]))
	} else if len(pack.Surfaces) > 1 {
		b.WriteString(fmt.Sprintf("- Surfaces: `%s`\n", strings.Join(pack.Surfaces, "`, `")))
	}
	switch len(pack.Hypotheses) {
	case 0:
	case 1:
		b.WriteString(fmt.Sprintf("- Hypothesis: %s\n", pack.Hypotheses[0]))
	default:
		b.WriteString("- Hypotheses:\n")
		for _, hypothesis := range pack.Hypotheses {
			b.WriteString(fmt.Sprintf("  - %s\n", hypothesis))
		}
	}
	if pack.HasBest {
		b.WriteString(fmt.Sprintf("- Best slice: `%s` under `%s` with `%d/%d` verified pass (`%.2f%%`).\n",
			benchmarkReportModelLabel(pack.BestSlice.DisplayName, pack.BestSlice.LLMModel, pack.BestSlice.Model),
			pack.BestSlice.PromptVersion,
			pack.BestSlice.PassCount,
			pack.BestSlice.CasesTotal,
			benchmarkAggregatePassRate(pack.BestSlice),
		))
	}
	b.WriteString("\n")
	renderBenchmarkMetaTopSlicesTable(b, pack.Aggregates, 0)
	return nil
}

func sortBenchmarkAggregatesByMetaPriority(aggregates []benchmarkRunSummaryAggregate) []benchmarkRunSummaryAggregate {
	ordered := append([]benchmarkRunSummaryAggregate(nil), aggregates...)
	slices.SortFunc(ordered, func(a, b benchmarkRunSummaryAggregate) int {
		aRate := benchmarkAggregatePassRate(a)
		bRate := benchmarkAggregatePassRate(b)
		switch {
		case aRate > bRate:
			return -1
		case aRate < bRate:
			return 1
		case a.PassCount > b.PassCount:
			return -1
		case a.PassCount < b.PassCount:
			return 1
		case a.CasesTotal > b.CasesTotal:
			return -1
		case a.CasesTotal < b.CasesTotal:
			return 1
		case a.LatestRunID > b.LatestRunID:
			return -1
		case a.LatestRunID < b.LatestRunID:
			return 1
		default:
			return strings.Compare(a.LLMModel, b.LLMModel)
		}
	})
	return ordered
}

func distinctBenchmarkPackSurfaces(rows []benchmarkRunSummaryRow) []string {
	seen := make(map[string]struct{}, len(rows))
	surfaces := make([]string, 0, len(rows))
	for _, row := range rows {
		surface := strings.TrimSpace(row.Surface)
		if surface == "" {
			continue
		}
		if _, ok := seen[surface]; ok {
			continue
		}
		seen[surface] = struct{}{}
		surfaces = append(surfaces, surface)
	}
	slices.Sort(surfaces)
	return surfaces
}
