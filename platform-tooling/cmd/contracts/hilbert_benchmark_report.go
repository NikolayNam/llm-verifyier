package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

func renderBenchmarkMarkdownReport(summaryPaths []string, projectFolder, casesFile, modelCatalogPath string, reports []benchmarkCasePackReport, generatedAt time.Time) string {
	var b strings.Builder
	packLabel := "Case packs"
	packItemLabel := "Case Pack"
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		packLabel = "Theorem packs"
		packItemLabel = "Theorem Pack"
	}
	b.WriteString("# Hilbert Benchmark Model Comparison Report\n\n")
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
	b.WriteString(fmt.Sprintf("- %s: `%d`\n", packLabel, len(reports)))
	b.WriteString("\n")
	if len(reports) == 0 {
		b.WriteString("No benchmark runs matched the selected filter.\n")
		return b.String()
	}

	for _, report := range reports {
		b.WriteString(fmt.Sprintf("## %s: `%s`\n\n", packItemLabel, report.CasesFile))
		if packID := benchmarkReportTheoremPackID(report.Rows); packID != "" {
			b.WriteString(fmt.Sprintf("- Theorem pack id: `%s`\n", packID))
		}
		b.WriteString(fmt.Sprintf("- Runs: `%d`\n", report.TotalRuns))
		if strings.TrimSpace(report.Hypothesis) != "" {
			b.WriteString(fmt.Sprintf("- Hypothesis: %s\n", report.Hypothesis))
		}
		b.WriteString("\n")
		renderBenchmarkAggregateSection(&b, "Aggregate", "", report.Aggregates)
		renderBenchmarkChainSummarySections(&b, report.ChainSummary, report.CasesFile)
		if report.Observed != nil {
			renderBenchmarkObservedSections(&b, report.ProjectFolder, report.Observed)
		}
		if report.Comparative != nil {
			renderBenchmarkAggregateSection(&b, "Engineering-best-performance (v1.2)", "Valid engineering runs recorded under `hilbert-ai-verification-benchmark-v1.2`.", report.Comparative.EngineeringBestPerformance)
			renderBenchmarkAggregateSection(&b, "Clean baseline (v1.3)", "Valid comparative runs recorded under `hilbert-ai-verification-benchmark-v1.3`.", report.Comparative.CleanBaseline)
			renderBenchmarkAggregateSection(&b, "<=20b cohort", "Primary `v1.3` comparative slice for models classified as `<=20b`.", report.Comparative.LE20B)
			renderBenchmarkAggregateSection(&b, ">20b cohort", "Primary `v1.3` comparative slice for models classified as `>20b`.", report.Comparative.GT20B)
			renderBenchmarkAggregateSection(&b, "Unknown-size appendix", "Valid `v1.3` runs kept outside the size-based primary conclusion because the catalog marks them as `unknown` or `appendix`.", report.Comparative.UnknownSizeAppendix)
			renderBenchmarkAppendixSection(&b, "Insufficient evidence / invalid runs", "Catalog entries without a valid `v2/v1.3` run and operationally invalid runs remain here as appendix evidence only.", report.Comparative.Appendix)
		}
	}
	return b.String()
}

func renderBenchmarkAggregateSection(b *strings.Builder, title, description string, aggregates []benchmarkRunSummaryAggregate) {
	if strings.TrimSpace(title) != "" {
		b.WriteString(fmt.Sprintf("### %s\n\n", title))
	}
	if strings.TrimSpace(description) != "" {
		b.WriteString(description + "\n\n")
	}
	if len(aggregates) == 0 {
		b.WriteString("No matching runs.\n\n")
		return
	}
	renderBenchmarkSectionContext(
		b,
		benchmarkCollectDistinct(aggregates, func(item benchmarkRunSummaryAggregate) string { return item.PromptVersion }),
		benchmarkCollectDistinct(aggregates, func(item benchmarkRunSummaryAggregate) string { return item.CertificateVersion }),
		benchmarkLatestTimestampFrom(aggregates, func(item benchmarkRunSummaryAggregate) string { return item.LatestTimestampUTC }),
	)
	b.WriteString("| Model | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Avg Run Elapsed (s) | Latest Run (UTC) |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, aggregate := range aggregates {
		b.WriteString(renderBenchmarkAggregateMarkdownRow(aggregate))
	}
	b.WriteString("\n")
}

func renderBenchmarkAggregateMarkdownRow(aggregate benchmarkRunSummaryAggregate) string {
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
	return fmt.Sprintf("| `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %s | %d | %s | `%s` |\n",
		benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
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
		formatBenchmarkFloat(avgRunElapsedSeconds),
		displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
	)
}

func renderBenchmarkObservedSections(b *strings.Builder, projectFolder string, data *benchmarkObservedData) {
	renderBenchmarkObservedSectionsImpl(b, projectFolder, data)
}

func renderBenchmarkFocusedCategorySections(b *strings.Builder, projectFolder string, data *benchmarkObservedData) {
	renderBenchmarkFocusedCategorySectionsImpl(b, projectFolder, data)
}

func renderBenchmarkNDPipelineTraceSection(b *strings.Builder, aggregates []benchmarkNDPipelineTraceAggregate) {
	renderBenchmarkNDPipelineTraceSectionImpl(b, aggregates)
}

func renderBenchmarkResultSliceSection(b *strings.Builder, title, description string, aggregates []benchmarkResultSliceAggregate) {
	renderBenchmarkResultSliceSectionImpl(b, title, description, aggregates)
}

func renderBenchmarkResultSliceMarkdownRow(aggregate benchmarkResultSliceAggregate) string {
	return renderBenchmarkResultSliceMarkdownRowImpl(aggregate, true)
}

func renderBenchmarkHardCaseAuditSection(b *strings.Builder, audits []benchmarkHardCaseAudit) {
	renderBenchmarkHardCaseAuditSectionImpl(b, audits)
}

func renderBenchmarkPerCaseHardestFailuresSection(b *strings.Builder, aggregates []benchmarkCaseFailureAggregate) {
	renderBenchmarkPerCaseHardestFailuresSectionImpl(b, aggregates)
}

func renderBenchmarkObservedSummary(stdout io.Writer, data *benchmarkObservedData) {
	renderBenchmarkObservedSummaryImpl(stdout, data)
}

func printBenchmarkResultSliceSummary(stdout io.Writer, label string, aggregates []benchmarkResultSliceAggregate) {
	printBenchmarkResultSliceSummaryImpl(stdout, label, aggregates)
}

func printBenchmarkNDPipelineTraceSummary(stdout io.Writer, aggregates []benchmarkNDPipelineTraceAggregate) {
	printBenchmarkNDPipelineTraceSummaryImpl(stdout, aggregates)
}

func renderBenchmarkAppendixSection(b *strings.Builder, title, description string, entries []benchmarkComparativeAppendixEntry) {
	b.WriteString(fmt.Sprintf("### %s\n\n", title))
	if strings.TrimSpace(description) != "" {
		b.WriteString(description + "\n\n")
	}
	if len(entries) == 0 {
		b.WriteString("No appendix entries.\n\n")
		return
	}
	renderBenchmarkSectionContext(
		b,
		benchmarkCollectDistinct(entries, func(item benchmarkComparativeAppendixEntry) string { return item.PromptVersion }),
		nil,
		benchmarkLatestTimestampFrom(entries, func(item benchmarkComparativeAppendixEntry) string {
			if item.latestTimestamp.IsZero() {
				return ""
			}
			return item.latestTimestamp.UTC().Format(time.RFC3339)
		}),
	)
	b.WriteString("| Model | Tier | Comparison Status | Status | Reason | Runs | Cases | Pass | Request Failure | Latest Run (UTC) |\n")
	b.WriteString("| --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | --- |\n")
	for _, entry := range entries {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %s | %d | %d | %d | %d | `%s` |\n",
			benchmarkReportModelLabel(entry.DisplayName, entry.LLMModel, ""),
			valueOrNA(entry.Tier),
			valueOrNA(entry.ComparisonStatus),
			valueOrNA(entry.Status),
			valueOrNA(entry.Reason),
			entry.Runs,
			entry.CasesTotal,
			entry.PassCount,
			entry.RequestFailureCount,
			func() string {
				if entry.latestTimestamp.IsZero() {
					return "n/a"
				}
				return displayBenchmarkTimestamp(entry.latestTimestamp.UTC().Format(time.RFC3339))
			}(),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkResearchMarkdownReport(summaryPaths []string, projectFolder, casesFile, modelCatalogPath string, reports []benchmarkCasePackReport, generatedAt time.Time) (string, error) {
	var b strings.Builder
	packLabel := "Case packs"
	researchItemLabel := "Case Pack"
	if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
		packLabel = "Theorem packs"
		researchItemLabel = "Theorem Pack"
	}

	b.WriteString("# Hilbert Benchmark Research Report\n\n")
	b.WriteString("This report is generated from recorded summary rows. It is intended to provide a research-facing interpretation layer above the benchmark summary and comparison tables.\n\n")
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
	b.WriteString(fmt.Sprintf("- %s: `%d`\n", packLabel, len(reports)))
	b.WriteString("\n")
	if len(reports) == 0 {
		b.WriteString("No benchmark runs matched the selected filter.\n")
		return b.String(), nil
	}

	for _, report := range reports {
		if err := renderBenchmarkResearchPackSection(&b, researchItemLabel, report); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func renderBenchmarkResearchPackSection(b *strings.Builder, itemLabel string, report benchmarkCasePackReport) error {
	totalFalseAccept := 0
	for _, row := range report.Rows {
		totalFalseAccept += mustParseBenchmarkSummaryInt(row.FalseAcceptCount)
	}
	hypotheses := distinctBenchmarkHypotheses(report.Rows)

	b.WriteString(fmt.Sprintf("## Research Summary: `%s`\n\n", report.CasesFile))
	b.WriteString(fmt.Sprintf("- %s: `%s`\n", itemLabel, report.CasesFile))
	if packID := benchmarkReportTheoremPackID(report.Rows); packID != "" {
		b.WriteString(fmt.Sprintf("- Theorem pack id: `%s`\n", packID))
	}
	b.WriteString(fmt.Sprintf("- Recorded runs: `%d`\n", report.TotalRuns))
	switch len(hypotheses) {
	case 1:
		b.WriteString(fmt.Sprintf("- Hypothesis: %s\n", hypotheses[0]))
	case 2:
		b.WriteString("- Hypotheses:\n")
		for _, hypothesis := range hypotheses {
			b.WriteString(fmt.Sprintf("  - %s\n", hypothesis))
		}
	default:
		if strings.TrimSpace(report.Hypothesis) != "" {
			b.WriteString(fmt.Sprintf("- Hypothesis family: %s\n", report.Hypothesis))
		}
	}
	if totalFalseAccept == 0 {
		b.WriteString("- Conservative boundary status: `preserved` (`false_accept = 0` across the selected summary rows)\n")
	} else {
		b.WriteString(fmt.Sprintf("- Conservative boundary status: `violated` (`false_accept = %d` across the selected summary rows)\n", totalFalseAccept))
	}
	b.WriteString("\n")

	b.WriteString("### Findings\n\n")
	pairs := buildBenchmarkResearchPairComparisons(report.Rows)
	if len(pairs) > 0 {
		for _, pair := range pairs {
			finding, err := renderBenchmarkResearchPairFinding(pair)
			if err != nil {
				return err
			}
			b.WriteString(finding)
		}
		b.WriteString("\n")
	} else {
		bestAggregate, ok := benchmarkBestAggregate(report.Aggregates)
		if ok {
			passRate := benchmarkAggregatePassRate(bestAggregate)
			b.WriteString(fmt.Sprintf("- Best recorded slice in this pack: `%s` under `%s` with `%d/%d` verified pass (`%.2f%%`).\n", bestAggregate.LLMModel, bestAggregate.PromptVersion, bestAggregate.PassCount, bestAggregate.CasesTotal, passRate))
		}
		if report.Comparative != nil {
			renderBenchmarkResearchComparativeFindings(b, report.Comparative)
		} else {
			b.WriteString("- No model-catalog-driven cohort analysis was available for this pack, so the report is limited to recorded run-level evidence.\n")
		}
		b.WriteString("\n")
	}

	renderBenchmarkAggregateSection(b, "Aggregate", "Summary-row aggregate across all recorded runs in this pack. Use this as top-layer context only; interpret it together with the observed slices below.", report.Aggregates)
	renderBenchmarkChainSummarySections(b, report.ChainSummary, report.CasesFile)
	renderBenchmarkObservedSections(b, report.ProjectFolder, report.Observed)

	b.WriteString("### Caveats\n\n")
	if report.Comparative != nil && len(report.Comparative.Appendix) > 0 {
		invalidRuns := 0
		missingEvidence := 0
		for _, entry := range report.Comparative.Appendix {
			switch strings.TrimSpace(entry.Status) {
			case "invalid-run":
				invalidRuns++
			default:
				missingEvidence++
			}
		}
		if invalidRuns > 0 {
			b.WriteString(fmt.Sprintf("- `%d` appendix entr%s correspond to operationally invalid or non-comparative runs; these remain part of the audit trail but are excluded from the main comparative claim.\n", invalidRuns, pluralSuffix(invalidRuns, "y", "ies")))
		}
		if missingEvidence > 0 {
			b.WriteString(fmt.Sprintf("- `%d` appendix entr%s correspond to missing or intentionally non-primary evidence and should not be read as negative proof-quality results.\n", missingEvidence, pluralSuffix(missingEvidence, "y", "ies")))
		}
	} else {
		b.WriteString("- No appendix exclusions were recorded for this pack.\n")
	}
	if len(pairs) > 0 {
		b.WriteString("- Paired findings compare the latest available direct-Hilbert `v1.3` run against the latest available `ND -> Hilbert` run for the same `llm_model` within this pack. They do not by themselves establish a broader generalization claim.\n")
	} else {
		b.WriteString("- This interpretation is summary-driven and does not re-judge individual raw proof objects beyond the statuses already recorded in the summary layer.\n")
	}
	b.WriteString("\n")
	return nil
}

func renderBenchmarkChainSummarySections(b *strings.Builder, data *benchmarkChainSummaryData, casesFile string) {
	if data == nil {
		return
	}
	b.WriteString("### Chain Summary\n\n")
	if len(data.Warnings) > 0 {
		for _, warning := range data.Warnings {
			b.WriteString(fmt.Sprintf("- Warning: %s\n", warning))
		}
		b.WriteString("\n")
	}
	if len(data.Aggregates) == 0 {
		b.WriteString("No compositional chain summary rows were available for this pack.\n\n")
		return
	}
	columns := benchmarkChainSummaryColumns(data)
	promptVersions := benchmarkCollectDistinct(data.Aggregates, func(item benchmarkChainSummaryAggregate) string { return item.PromptVersion })
	latestTimestamp := benchmarkLatestTimestampFrom(data.Aggregates, func(item benchmarkChainSummaryAggregate) string { return item.LatestTimestampUTC })
	if columns.ShowSurface {
		renderBenchmarkSectionContext(b, promptVersions, nil, latestTimestamp)
	} else {
		renderBenchmarkSectionContextWithSurface(
			b,
			promptVersions,
			nil,
			benchmarkCollectDistinct(data.Aggregates, func(item benchmarkChainSummaryAggregate) string { return item.Surface }),
			latestTimestamp,
		)
	}
	b.WriteString(benchmarkChainSummaryDescription(columns, casesFile) + "\n\n")
	b.WriteString(benchmarkChainSummaryMarkdownHeader(columns))
	b.WriteString(benchmarkChainSummaryMarkdownDivider(columns))
	for _, aggregate := range data.Aggregates {
		b.WriteString(renderBenchmarkChainSummaryMarkdownRow(aggregate, columns))
	}
	b.WriteString("\n")
	if len(data.StageAggregates) > 0 {
		renderBenchmarkChainStageGroupSections(b, data.StageAggregates, casesFile)
		b.WriteString("#### Stage protocol breakdown\n\n")
		b.WriteString("Per-stage pass rates derived from `stage_sequence_json`, `stage_roles_json`, and `stage_passes_json`. For non-starter protocols, this section is the authoritative stage-level view.\n\n")
		renderBenchmarkChainStageAggregateTable(b, data.StageAggregates)
		b.WriteString("\n")
	}
	renderBenchmarkChainMetadataSection(b, "#### Per-transition-group", "Grouped by annotated `transition_group` when that metadata is present in this lane.", "Transition Group", data.TransitionGroups, columns)
	renderBenchmarkChainMetadataSection(b, "#### Per-import-arity", "Grouped by annotated `import_arity` when that metadata is present in this lane.", "Import Arity", data.ImportArities, columns)
	renderBenchmarkChainMetadataSection(b, "#### Per-reuse-shape", "Grouped by annotated `reuse_shape` when that metadata is present in this lane.", "Reuse Shape", data.ReuseShapes, columns)
	renderBenchmarkChainMetadataSection(b, "#### Per-bridge-depth", "Grouped by annotated `bridge_depth` when that metadata is present in this lane.", "Bridge Depth", data.BridgeDepths, columns)
	renderBenchmarkChainMetadataSection(b, "#### Per-symbol-overlap", "Grouped by annotated `symbol_overlap` when that metadata is present in this lane.", "Symbol Overlap", data.SymbolOverlaps, columns)
	renderBenchmarkChainMetadataSection(b, "#### Per-negative-twin-hardness", "Grouped by annotated `negative_twin_hardness` when that metadata is present in this lane.", "Negative Twin Hardness", data.NegativeTwinHardnesses, columns)
	renderBenchmarkFGFailureMixSection(b, data.FGFailureMix)
	renderBenchmarkHardestFGFailuresSection(b, data.HardestFGFailures)
	if len(data.Failures) > 0 {
		failureSurfaces := benchmarkCollectDistinct(data.Failures, func(item benchmarkChainFailureAggregate) string { return item.Surface })
		showSurfaceColumn := len(failureSurfaces) > 1
		b.WriteString("#### Chain failure breakdown\n\n")
		b.WriteString("`failure_class` distinguishes failures in the bridge/import -> final transition into `semantic_transport_failure`, `import_binding_failure`, `certificate_assembly_failure`, and `final_proof_closure_failure`. Negative-control failures remain explicit rather than being folded into that taxonomy.\n\n")
		if !showSurfaceColumn && len(failureSurfaces) == 1 {
			b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", failureSurfaces[0]))
		}
		if showSurfaceColumn {
			b.WriteString("| Model | Surface | Failure Stage | Stage Role | Failure Class | Failure Type | Chains | Latest Run (UTC) |\n")
			b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | --- |\n")
		} else {
			b.WriteString("| Model | Failure Stage | Stage Role | Failure Class | Failure Type | Chains | Latest Run (UTC) |\n")
			b.WriteString("| --- | --- | --- | --- | --- | ---: | --- |\n")
		}
		for _, aggregate := range data.Failures {
			if showSurfaceColumn {
				b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %d | `%s` |\n",
					benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
					valueOrNA(aggregate.Surface),
					valueOrNA(aggregate.FailureStage),
					valueOrNA(aggregate.FailureStageRole),
					valueOrNA(aggregate.FailureClass),
					valueOrNA(aggregate.FailureType),
					aggregate.Chains,
					displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
				))
				continue
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %d | `%s` |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				valueOrNA(aggregate.FailureStage),
				valueOrNA(aggregate.FailureStageRole),
				valueOrNA(aggregate.FailureClass),
				valueOrNA(aggregate.FailureType),
				aggregate.Chains,
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			))
		}
		b.WriteString("\n")
	}
}

func renderBenchmarkChainMetadataSection(b *strings.Builder, title, description, valueHeader string, aggregates []benchmarkChainMetadataAggregate, columns benchmarkChainSummaryColumnSet) {
	if len(aggregates) == 0 {
		return
	}
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkChainMetadataAggregate) string { return item.Surface })
	showSurfaceColumn := columns.ShowSurface && len(surfaces) > 1
	b.WriteString(title + "\n\n")
	if strings.TrimSpace(description) != "" {
		b.WriteString(description + "\n\n")
	}
	if !showSurfaceColumn && len(surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", surfaces[0]))
	}
	headers := []string{"Model", valueHeader, "Runs", "Chains"}
	dividers := []string{"---", "---", "---:", "---:"}
	if showSurfaceColumn {
		headers = []string{"Model", "Surface", valueHeader, "Runs", "Chains"}
		dividers = []string{"---", "---", "---", "---:", "---:"}
	}
	if columns.ShowGoldFinal {
		headers = append(headers, "Gold Final")
		dividers = append(dividers, "---:")
	}
	if columns.ShowNegativeTwin {
		headers = append(headers, "Negative Twin")
		dividers = append(dividers, "---:")
	}
	headers = append(headers, "All Stages", "Latest Run (UTC)")
	dividers = append(dividers, "---:", "---")
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	b.WriteString("| " + strings.Join(dividers, " | ") + " |\n")
	for _, aggregate := range aggregates {
		cells := []string{
			fmt.Sprintf("`%s`", benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model)),
			fmt.Sprintf("`%s`", valueOrNA(aggregate.GroupValue)),
			fmt.Sprintf("%d", aggregate.Runs),
			fmt.Sprintf("%d", aggregate.ChainsTotal),
		}
		if showSurfaceColumn {
			cells = []string{
				fmt.Sprintf("`%s`", benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model)),
				fmt.Sprintf("`%s`", valueOrNA(aggregate.Surface)),
				fmt.Sprintf("`%s`", valueOrNA(aggregate.GroupValue)),
				fmt.Sprintf("%d", aggregate.Runs),
				fmt.Sprintf("%d", aggregate.ChainsTotal),
			}
		}
		if columns.ShowGoldFinal {
			cells = append(cells, benchmarkCountRateMarkdown(aggregate.GoldFinalPassCount, aggregate.ChainsTotal))
		}
		if columns.ShowNegativeTwin {
			cells = append(cells, benchmarkCountRateMarkdown(aggregate.NegativeTwinPassCount, aggregate.ChainsTotal))
		}
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.AllStagesPassCount, aggregate.ChainsTotal), fmt.Sprintf("`%s`", displayBenchmarkTimestamp(aggregate.LatestTimestampUTC)))
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	b.WriteString("\n")
}

func renderBenchmarkFGFailureMixSection(b *strings.Builder, aggregates []benchmarkFGFailureAggregate) {
	if len(aggregates) == 0 {
		return
	}
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkFGFailureAggregate) string { return item.Surface })
	showSurfaceColumn := len(surfaces) > 1
	b.WriteString("#### FG failure-type mix\n\n")
	b.WriteString("Failure mix restricted to `fg` rows. This isolates final gold-import collapse from negative-control behavior.\n\n")
	if !showSurfaceColumn && len(surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", surfaces[0]))
	}
	if showSurfaceColumn {
		b.WriteString("| Model | Surface | Failure Type | Cases | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | ---: | --- |\n")
	} else {
		b.WriteString("| Model | Failure Type | Cases | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | ---: | --- |\n")
	}
	for _, aggregate := range aggregates {
		if showSurfaceColumn {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %d | `%s` |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				valueOrNA(aggregate.Surface),
				valueOrNA(aggregate.FailureType),
				aggregate.Cases,
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			))
			continue
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | `%s` |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			valueOrNA(aggregate.FailureType),
			aggregate.Cases,
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkHardestFGFailuresSection(b *strings.Builder, aggregates []benchmarkFGCaseFailureAggregate) {
	if len(aggregates) == 0 {
		return
	}
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkFGCaseFailureAggregate) string { return item.Surface })
	showSurfaceColumn := len(surfaces) > 1
	b.WriteString("#### Hardest FG failures\n\n")
	b.WriteString("Ranked within `fg` rows by lowest pass rate and then by failure count.\n\n")
	if !showSurfaceColumn && len(surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", surfaces[0]))
	}
	if showSurfaceColumn {
		b.WriteString("| Model | Surface | Case ID | Chain ID | Transition Group | Import Arity | Reuse Shape | Bridge Depth | Symbol Overlap | Negative Twin Hardness | Runs | Pass | Failures | Schema Failure | Kernel Failure | Request Failure | Format Failure | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	} else {
		b.WriteString("| Model | Case ID | Chain ID | Transition Group | Import Arity | Reuse Shape | Bridge Depth | Symbol Overlap | Negative Twin Hardness | Runs | Pass | Failures | Schema Failure | Kernel Failure | Request Failure | Format Failure | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	}
	for _, aggregate := range aggregates {
		if showSurfaceColumn {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %d | %d | %d | %d | %d | %d | %d | `%s` |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				valueOrNA(aggregate.Surface),
				valueOrNA(aggregate.CaseID),
				valueOrNA(aggregate.ChainID),
				valueOrNA(aggregate.TransitionGroup),
				valueOrNA(aggregate.ImportArity),
				valueOrNA(aggregate.ReuseShape),
				valueOrNA(aggregate.BridgeDepth),
				valueOrNA(aggregate.SymbolOverlap),
				valueOrNA(aggregate.NegativeTwinHardness),
				aggregate.Runs,
				aggregate.PassCount,
				aggregate.FailureCount,
				aggregate.SchemaFailureCount,
				aggregate.KernelFailureCount,
				aggregate.RequestFailureCount,
				aggregate.FormatFailureCount,
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			))
			continue
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %d | %d | %d | %d | %d | %d | %d | `%s` |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			valueOrNA(aggregate.CaseID),
			valueOrNA(aggregate.ChainID),
			valueOrNA(aggregate.TransitionGroup),
			valueOrNA(aggregate.ImportArity),
			valueOrNA(aggregate.ReuseShape),
			valueOrNA(aggregate.BridgeDepth),
			valueOrNA(aggregate.SymbolOverlap),
			valueOrNA(aggregate.NegativeTwinHardness),
			aggregate.Runs,
			aggregate.PassCount,
			aggregate.FailureCount,
			aggregate.SchemaFailureCount,
			aggregate.KernelFailureCount,
			aggregate.RequestFailureCount,
			aggregate.FormatFailureCount,
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkChainStageGroupSections(b *strings.Builder, aggregates []benchmarkChainStageAggregate, casesFile string) {
	type chainStageGroupSection struct {
		Title        string
		Description  string
		EmptyMessage string
		Rows         []benchmarkChainStageAggregate
	}

	grouped := map[string][]benchmarkChainStageAggregate{
		"starter":  nil,
		"import":   nil,
		"negative": nil,
		"other":    nil,
	}
	for _, aggregate := range aggregates {
		grouped[benchmarkChainStageGroup(aggregate.StageID, aggregate.StageRole)] = append(grouped[benchmarkChainStageGroup(aggregate.StageID, aggregate.StageRole)], aggregate)
	}
	laneKind := benchmarkChainSummaryLaneKind(casesFile)

	starterDescription := "Local prerequisite or seed-like stages. If this section stays clean while later composition stages collapse, the bottleneck is downstream imported-lemma composition rather than starter-stage authoring."
	importDescription := "Stages that consume imported lemmas or attempt final closure. Read this together with the prerequisite-stage section so early bridge or reuse failures are not misdiagnosed as a pure final-composition bottleneck."
	negativeDescription := "Negative twins / refusal-discipline stages. When this section stays clean while final stages fail, the failure mode is composition-specific rather than a general negative-control collapse."
	switch laneKind {
	case "bridge-import-final-research":
		starterDescription = "Bridge-stage prerequisite authoring lives here. If these rows fail, later `fg` or `fm` collapse may be downstream of an earlier semantic transport problem rather than a pure final-stage bottleneck."
		importDescription = "These rows cover gold-final and model-final closure after the bridge stage. Do not interpret them in isolation; check whether the bridge-stage rows already failed first."
	case "bridge-only-authoring":
		starterDescription = "This split lane intentionally isolates bridge-stage authoring only. Treat this section as the primary evidence surface for whether locally derivable bridge lemmas can be rendered into verifier-acceptable certificates."
		importDescription = "This split lane intentionally omits imported-final closure stages."
		negativeDescription = "This split lane intentionally omits negative-control rows."
	case "gold-final-composition-only":
		starterDescription = "This split lane intentionally omits bridge-stage authoring rows."
		importDescription = "This split lane isolates final composition over static gold-backed imported lemmas. This is the primary evidence surface for whether final closure works once prerequisite resources are fixed and trusted."
	}

	sections := []chainStageGroupSection{
		{
			Title:        "Starter-stage breakdown",
			Description:  starterDescription,
			EmptyMessage: "No starter-stage rows are present in this pack.",
			Rows:         grouped["starter"],
		},
		{
			Title:        "Import-stage breakdown",
			Description:  importDescription,
			EmptyMessage: "No import-stage rows are present in this pack.",
			Rows:         grouped["import"],
		},
		{
			Title:        "Negative-control breakdown",
			Description:  negativeDescription,
			EmptyMessage: "No negative-control rows are present in this pack.",
			Rows:         grouped["negative"],
		},
	}
	if len(grouped["other"]) > 0 {
		sections = append(sections, chainStageGroupSection{
			Title:        "Other stage breakdown",
			Description:  "Stage rows that do not cleanly map into starter, import, or negative-control groups.",
			EmptyMessage: "",
			Rows:         grouped["other"],
		})
	}

	for _, section := range sections {
		b.WriteString(fmt.Sprintf("#### %s\n\n", section.Title))
		b.WriteString(section.Description + "\n\n")
		if len(section.Rows) == 0 {
			b.WriteString(section.EmptyMessage + "\n\n")
			continue
		}
		renderBenchmarkChainStageAggregateTable(b, section.Rows)
		b.WriteString("\n")
	}
}

func renderBenchmarkChainStageAggregateTable(b *strings.Builder, aggregates []benchmarkChainStageAggregate) {
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkChainStageAggregate) string { return item.Surface })
	showSurfaceColumn := len(surfaces) > 1
	if !showSurfaceColumn && len(surfaces) == 1 {
		b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", surfaces[0]))
	}
	if showSurfaceColumn {
		b.WriteString("| Model | Surface | Stage ID | Stage Role | Chains | Pass | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | --- | ---: | ---: | --- |\n")
	} else {
		b.WriteString("| Model | Stage ID | Stage Role | Chains | Pass | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | --- | ---: | ---: | --- |\n")
	}
	for _, aggregate := range aggregates {
		if showSurfaceColumn {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %d | %s | `%s` |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				valueOrNA(aggregate.Surface),
				valueOrNA(aggregate.StageID),
				valueOrNA(aggregate.StageRole),
				aggregate.Chains,
				benchmarkCountRateMarkdown(aggregate.PassCount, aggregate.Chains),
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			))
			continue
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %d | %s | `%s` |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			valueOrNA(aggregate.StageID),
			valueOrNA(aggregate.StageRole),
			aggregate.Chains,
			benchmarkCountRateMarkdown(aggregate.PassCount, aggregate.Chains),
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
		))
	}
}

func benchmarkChainStageGroup(stageID, stageRole string) string {
	stageID = strings.ToLower(strings.TrimSpace(stageID))
	stageRole = strings.ToLower(strings.TrimSpace(stageRole))
	switch {
	case stageID == "neg" || strings.Contains(stageRole, "negative_control"):
		return "negative"
	case strings.Contains(stageRole, "import"), strings.Contains(stageRole, "final_"), strings.Contains(stageRole, "intermediate_"),
		stageID == "fg", stageID == "fm", stageID == "bg", stageID == "bm", stageID == "d2g", stageID == "d2m", stageID == "d3g", stageID == "d3m":
		return "import"
	case strings.Contains(stageRole, "seed"), strings.Contains(stageRole, "source_family"), strings.Contains(stageRole, "branch_"),
		strings.Contains(stageRole, "bridge"), strings.Contains(stageRole, "merge_rule"), strings.Contains(stageRole, "resource"),
		stageID == "s1", stageID == "s2", stageID == "s3", stageID == "r1", stageID == "r2", stageID == "m1", stageID == "m2":
		return "starter"
	default:
		return "other"
	}
}

type benchmarkChainSummaryColumnSet struct {
	ShowSurface           bool
	ShowPrerequisiteStage bool
	ShowGoldFinal         bool
	ShowModelFinal        bool
	ShowConditionalGold   bool
	ShowConditionalModel  bool
	ShowNegativeTwin      bool
}

func renderBenchmarkChainSummaryMarkdownRow(aggregate benchmarkChainSummaryAggregate, columns benchmarkChainSummaryColumnSet) string {
	cells := []string{
		fmt.Sprintf("`%s`", benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model)),
		fmt.Sprintf("%d", aggregate.Runs),
		fmt.Sprintf("%d", aggregate.ChainsTotal),
	}
	if columns.ShowSurface {
		cells = []string{
			fmt.Sprintf("`%s`", benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model)),
			fmt.Sprintf("`%s`", valueOrNA(aggregate.Surface)),
			fmt.Sprintf("%d", aggregate.Runs),
			fmt.Sprintf("%d", aggregate.ChainsTotal),
		}
	}
	if columns.ShowPrerequisiteStage {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.ImportStagePassCount, aggregate.ChainsTotal))
	}
	if columns.ShowGoldFinal {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.FinalCompositionPassGoldCount, aggregate.ChainsTotal))
	}
	if columns.ShowModelFinal {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.FinalCompositionPassModelCount, aggregate.ChainsTotal))
	}
	if columns.ShowConditionalGold {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.ConditionalFinalPassGoldCount, aggregate.ChainsTotal))
	}
	if columns.ShowConditionalModel {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.ConditionalFinalPassModelCount, aggregate.ChainsTotal))
	}
	if columns.ShowNegativeTwin {
		cells = append(cells, benchmarkCountRateMarkdown(aggregate.NegativeTwinPassCount, aggregate.ChainsTotal))
	}
	cells = append(cells, benchmarkCountRateMarkdown(aggregate.AllStagesPassCount, aggregate.ChainsTotal), fmt.Sprintf("`%s`", displayBenchmarkTimestamp(aggregate.LatestTimestampUTC)))
	return "| " + strings.Join(cells, " | ") + " |\n"
}

func benchmarkChainSummaryColumns(data *benchmarkChainSummaryData) benchmarkChainSummaryColumnSet {
	var columns benchmarkChainSummaryColumnSet
	if data == nil {
		return columns
	}
	columns.ShowSurface = len(benchmarkCollectDistinct(data.Aggregates, func(item benchmarkChainSummaryAggregate) string { return item.Surface })) > 1
	for _, aggregate := range data.StageAggregates {
		if benchmarkChainStageShowsPrerequisite(aggregate.StageID, aggregate.StageRole) {
			columns.ShowPrerequisiteStage = true
		}
		if benchmarkChainStageShowsGoldFinal(aggregate.StageID, aggregate.StageRole) {
			columns.ShowGoldFinal = true
		}
		if benchmarkChainStageShowsModelFinal(aggregate.StageID, aggregate.StageRole) {
			columns.ShowModelFinal = true
		}
		if benchmarkChainStageShowsNegative(aggregate.StageID, aggregate.StageRole) {
			columns.ShowNegativeTwin = true
		}
	}
	if !columns.ShowPrerequisiteStage || !columns.ShowGoldFinal || !columns.ShowModelFinal || !columns.ShowNegativeTwin {
		for _, aggregate := range data.Aggregates {
			for _, stageID := range aggregate.ObservedStageIDs {
				if benchmarkChainStageShowsPrerequisite(stageID, "") {
					columns.ShowPrerequisiteStage = true
				}
				if benchmarkChainStageShowsGoldFinal(stageID, "") {
					columns.ShowGoldFinal = true
				}
				if benchmarkChainStageShowsModelFinal(stageID, "") {
					columns.ShowModelFinal = true
				}
				if benchmarkChainStageShowsNegative(stageID, "") {
					columns.ShowNegativeTwin = true
				}
			}
		}
	}
	columns.ShowConditionalGold = columns.ShowPrerequisiteStage && columns.ShowGoldFinal
	columns.ShowConditionalModel = columns.ShowPrerequisiteStage && columns.ShowModelFinal
	return columns
}

func benchmarkChainSummaryDescription(columns benchmarkChainSummaryColumnSet, casesFile string) string {
	switch benchmarkChainSummaryLaneKind(casesFile) {
	case "bridge-import-final-research":
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This baseline lane mixes bridge-stage authoring, gold-final closure, model-final closure, and negative controls. Failures can arise before final composition, so `fg` and `fm` must be interpreted together with the prerequisite-stage column and the stage protocol breakdown."
	case "bridge-only-authoring":
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This split lane isolates bridge-stage authoring only. No final-composition or negative-control stages are expected here; the key question is whether locally derivable bridge lemmas can be rendered as verifier-acceptable certificates."
	case "gold-final-composition-only":
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This split lane isolates final composition over static gold-backed imported lemmas. Bridge-stage authoring is intentionally omitted, so this surface should be read only as evidence about final closure given fixed trusted resources."
	}
	switch {
	case columns.ShowPrerequisiteStage && columns.ShowGoldFinal && columns.ShowModelFinal:
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This layer measures whether prerequisite stages actually compose into stable gold-import and model-import outcomes. Use the stage protocol breakdown as the authoritative stage-level view."
	case columns.ShowPrerequisiteStage && columns.ShowGoldFinal:
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This lane records prerequisite-stage completion, gold-final composition, and any available negative-control discipline. No model-final stage is present in this pack."
	case columns.ShowGoldFinal:
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This lane records gold-final composition and any available negative-control discipline. Prerequisite-stage rows are intentionally absent from this pack."
	default:
		return "Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. Use the stage protocol breakdown below as the authoritative stage-level view for this pack."
	}
}

func benchmarkChainSummaryMarkdownHeader(columns benchmarkChainSummaryColumnSet) string {
	headers := []string{"Model", "Runs", "Chains"}
	if columns.ShowSurface {
		headers = []string{"Model", "Surface", "Runs", "Chains"}
	}
	if columns.ShowPrerequisiteStage {
		headers = append(headers, "Bridge / Reuse Stage")
	}
	if columns.ShowGoldFinal {
		headers = append(headers, "Gold Final")
	}
	if columns.ShowModelFinal {
		headers = append(headers, "Model Final")
	}
	if columns.ShowConditionalGold {
		headers = append(headers, "Conditional Gold")
	}
	if columns.ShowConditionalModel {
		headers = append(headers, "Conditional Model")
	}
	if columns.ShowNegativeTwin {
		headers = append(headers, "Negative Twin")
	}
	headers = append(headers, "All Stages", "Latest Run (UTC)")
	return "| " + strings.Join(headers, " | ") + " |\n"
}

func benchmarkChainSummaryMarkdownDivider(columns benchmarkChainSummaryColumnSet) string {
	parts := []string{"---", "---:", "---:"}
	if columns.ShowSurface {
		parts = []string{"---", "---", "---:", "---:"}
	}
	if columns.ShowPrerequisiteStage {
		parts = append(parts, "---:")
	}
	if columns.ShowGoldFinal {
		parts = append(parts, "---:")
	}
	if columns.ShowModelFinal {
		parts = append(parts, "---:")
	}
	if columns.ShowConditionalGold {
		parts = append(parts, "---:")
	}
	if columns.ShowConditionalModel {
		parts = append(parts, "---:")
	}
	if columns.ShowNegativeTwin {
		parts = append(parts, "---:")
	}
	parts = append(parts, "---:", "---")
	return "| " + strings.Join(parts, " | ") + " |\n"
}

func benchmarkChainSummaryLaneKind(casesFile string) string {
	normalized := strings.ToLower(normalizeBenchmarkPath(casesFile))
	switch {
	case strings.HasSuffix(normalized, "/bridge-import-final-research-phase1.csv"), strings.HasSuffix(normalized, "bridge-import-final-research-phase1.csv"):
		return "bridge-import-final-research"
	case strings.HasSuffix(normalized, "/bridge-only-authoring-phase1.csv"), strings.HasSuffix(normalized, "bridge-only-authoring-phase1.csv"),
		strings.HasSuffix(normalized, "/bridge-only-authoring-canonical-phase1.csv"), strings.HasSuffix(normalized, "bridge-only-authoring-canonical-phase1.csv"),
		strings.HasSuffix(normalized, "/bridge-only-authoring-phase2.csv"), strings.HasSuffix(normalized, "bridge-only-authoring-phase2.csv"):
		// Phase1, canonical-ablation, and expanded phase2 packs should all render
		// with the same bridge-only narrative and stage semantics.
		return "bridge-only-authoring"
	case strings.HasSuffix(normalized, "/gold-final-composition-only-phase1.csv"), strings.HasSuffix(normalized, "gold-final-composition-only-phase1.csv"),
		strings.HasSuffix(normalized, "/gold-final-composition-only-phase2.csv"), strings.HasSuffix(normalized, "gold-final-composition-only-phase2.csv"):
		return "gold-final-composition-only"
	default:
		return ""
	}
}

func benchmarkChainStageShowsPrerequisite(stageID, stageRole string) bool {
	return benchmarkChainStageGroup(stageID, stageRole) == "starter"
}

func benchmarkChainStageShowsGoldFinal(stageID, stageRole string) bool {
	stageID = strings.ToLower(strings.TrimSpace(stageID))
	stageRole = strings.ToLower(strings.TrimSpace(stageRole))
	switch stageID {
	case "fg", "bg", "d2g", "d3g", "s3g":
		return true
	}
	return strings.Contains(stageRole, "final_gold") || strings.Contains(stageRole, "intermediate_gold") || strings.Contains(stageRole, "semi_gold")
}

func benchmarkChainStageShowsModelFinal(stageID, stageRole string) bool {
	stageID = strings.ToLower(strings.TrimSpace(stageID))
	stageRole = strings.ToLower(strings.TrimSpace(stageRole))
	switch stageID {
	case "fm", "bm", "d2m", "d3m", "s3m":
		return true
	}
	return strings.Contains(stageRole, "final_model") || strings.Contains(stageRole, "intermediate_model")
}

func benchmarkChainStageShowsNegative(stageID, stageRole string) bool {
	stageID = strings.ToLower(strings.TrimSpace(stageID))
	stageRole = strings.ToLower(strings.TrimSpace(stageRole))
	return stageID == "neg" || strings.Contains(stageRole, "negative_control")
}

func benchmarkCountRateMarkdown(count, total int) string {
	if total <= 0 {
		return "0/0 (n/a)"
	}
	return fmt.Sprintf("%d/%d (%.2f%%)", count, total, 100*float64(count)/float64(total))
}

func renderBenchmarkResearchComparativeFindings(b *strings.Builder, slices *benchmarkComparativeSlices) {
	if len(slices.CleanBaseline) > 0 {
		bestClean, ok := benchmarkBestAggregate(slices.CleanBaseline)
		if ok {
			b.WriteString(fmt.Sprintf("- Best clean-baseline (`v1.3`) result: `%s` with `%d/%d` verified pass (`%.2f%%`).\n", bestClean.LLMModel, bestClean.PassCount, bestClean.CasesTotal, benchmarkAggregatePassRate(bestClean)))
		}
	}
	if len(slices.LE20B) > 0 || len(slices.GT20B) > 0 {
		le20Best, le20OK := benchmarkBestAggregate(slices.LE20B)
		gt20Best, gt20OK := benchmarkBestAggregate(slices.GT20B)
		switch {
		case le20OK && gt20OK:
			b.WriteString(fmt.Sprintf("- Cohort snapshot: best `<=20b` result is `%s` at `%.2f%%`, while best `>20b` result is `%s` at `%.2f%%`.\n", le20Best.LLMModel, benchmarkAggregatePassRate(le20Best), gt20Best.LLMModel, benchmarkAggregatePassRate(gt20Best)))
		case le20OK:
			b.WriteString(fmt.Sprintf("- Cohort snapshot: only the `<=20b` cohort has valid comparative evidence here, led by `%s` at `%.2f%%`.\n", le20Best.LLMModel, benchmarkAggregatePassRate(le20Best)))
		case gt20OK:
			b.WriteString(fmt.Sprintf("- Cohort snapshot: only the `>20b` cohort has valid comparative evidence here, led by `%s` at `%.2f%%`.\n", gt20Best.LLMModel, benchmarkAggregatePassRate(gt20Best)))
		}
	}
	if len(slices.CleanBaseline) == 0 && len(slices.EngineeringBestPerformance) > 0 {
		bestEngineering, ok := benchmarkBestAggregate(slices.EngineeringBestPerformance)
		if ok {
			b.WriteString(fmt.Sprintf("- Only engineering-best-performance (`v1.2`) evidence was available here; the strongest such row was `%s` at `%.2f%%`.\n", bestEngineering.LLMModel, benchmarkAggregatePassRate(bestEngineering)))
		}
	}
}

func distinctBenchmarkHypotheses(rows []benchmarkRunSummaryRow) []string {
	seen := make(map[string]struct{}, len(rows))
	hypotheses := make([]string, 0, len(rows))
	for _, row := range rows {
		hypothesis := strings.TrimSpace(row.Hypothesis)
		if hypothesis == "" {
			continue
		}
		if _, ok := seen[hypothesis]; ok {
			continue
		}
		seen[hypothesis] = struct{}{}
		hypotheses = append(hypotheses, hypothesis)
	}
	slices.Sort(hypotheses)
	return hypotheses
}

func benchmarkReportTheoremPackID(rows []benchmarkRunSummaryRow) string {
	for _, row := range rows {
		if value := strings.TrimSpace(row.TheoremPackID); value != "" {
			return value
		}
	}
	return ""
}

func buildBenchmarkResearchPairComparisons(rows []benchmarkRunSummaryRow) []benchmarkResearchPairComparison {
	directByModel := latestBenchmarkRowsByPrompt(rows, benchmarkPromptVersionV13)
	ndByModel := latestBenchmarkRowsByPromptMatcher(rows, isNDBenchmarkPromptVersion)

	models := make([]string, 0)
	for model := range directByModel {
		if _, ok := ndByModel[model]; ok {
			models = append(models, model)
		}
	}
	slices.Sort(models)

	comparisons := make([]benchmarkResearchPairComparison, 0, len(models))
	for _, model := range models {
		comparisons = append(comparisons, benchmarkResearchPairComparison{
			LLMModel: benchmarkComparisonKeyLabel(directByModel[model]),
			Direct:   directByModel[model],
			ND:       ndByModel[model],
		})
	}
	return comparisons
}

func latestBenchmarkRowsByPrompt(rows []benchmarkRunSummaryRow, promptVersion string) map[string]benchmarkRunSummaryRow {
	return latestBenchmarkRowsByPromptMatcher(rows, func(version string) bool {
		return strings.TrimSpace(version) == strings.TrimSpace(promptVersion)
	})
}

func latestBenchmarkRowsByPromptMatcher(rows []benchmarkRunSummaryRow, match func(string) bool) map[string]benchmarkRunSummaryRow {
	latest := make(map[string]benchmarkRunSummaryRow)
	for _, row := range rows {
		if !match(strings.TrimSpace(row.PromptVersion)) {
			continue
		}
		key := strings.TrimSpace(row.LLMModel)
		if key == "" {
			key = strings.TrimSpace(row.Model)
		}
		if experimentID := strings.TrimSpace(row.ExperimentID); experimentID != "" {
			key = key + "@" + experimentID
		}
		current, ok := latest[key]
		if !ok || benchmarkRowIsLater(row, current) {
			latest[key] = row
		}
	}
	return latest
}

func isNDBenchmarkPromptVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	return version == ndBenchmarkPromptVersionV1 || strings.HasPrefix(version, ndBenchmarkPromptVersionV1+".")
}

func benchmarkRowIsLater(candidate, current benchmarkRunSummaryRow) bool {
	candidateTS, candidateErr := time.Parse(time.RFC3339, strings.TrimSpace(candidate.TimestampUTC))
	currentTS, currentErr := time.Parse(time.RFC3339, strings.TrimSpace(current.TimestampUTC))
	switch {
	case candidateErr == nil && currentErr == nil:
		if !candidateTS.Equal(currentTS) {
			return candidateTS.After(currentTS)
		}
	case candidateErr == nil && currentErr != nil:
		return true
	case candidateErr != nil && currentErr == nil:
		return false
	}
	return strings.TrimSpace(candidate.RunID) > strings.TrimSpace(current.RunID)
}

func renderBenchmarkResearchPairFinding(pair benchmarkResearchPairComparison) (string, error) {
	directStats, err := benchmarkResearchStatsFromRow(pair.Direct)
	if err != nil {
		return "", err
	}
	ndStats, err := benchmarkResearchStatsFromRow(pair.ND)
	if err != nil {
		return "", err
	}

	directRate := benchmarkPassRate(directStats.PassCount, directStats.CasesTotal)
	ndRate := benchmarkPassRate(ndStats.PassCount, ndStats.CasesTotal)
	delta := ndRate - directRate

	var lead string
	switch {
	case delta > 0.0001:
		lead = fmt.Sprintf("`ND -> Hilbert` improved verified pass rate from `%d/%d` (`%.2f%%`) to `%d/%d` (`%.2f%%`, `+%.2fpp`).", directStats.PassCount, directStats.CasesTotal, directRate, ndStats.PassCount, ndStats.CasesTotal, ndRate, delta)
	case delta < -0.0001:
		lead = fmt.Sprintf("`ND -> Hilbert` regressed verified pass rate from `%d/%d` (`%.2f%%`) to `%d/%d` (`%.2f%%`, `%.2fpp`).", directStats.PassCount, directStats.CasesTotal, directRate, ndStats.PassCount, ndStats.CasesTotal, ndRate, delta)
	default:
		lead = fmt.Sprintf("`ND -> Hilbert` matched direct Hilbert on verified pass rate at `%d/%d` (`%.2f%%`).", ndStats.PassCount, ndStats.CasesTotal, ndRate)
	}

	boundary := "Conservative-boundary status stayed intact with `false_accept = 0` in both runs."
	if directStats.FalseAcceptCount > 0 || ndStats.FalseAcceptCount > 0 {
		boundary = fmt.Sprintf("Conservative-boundary status changed: direct `false_accept = %d`, ND `false_accept = %d`.", directStats.FalseAcceptCount, ndStats.FalseAcceptCount)
	}

	latency := ""
	if directStats.AvgLatencyMS > 0 || ndStats.AvgLatencyMS > 0 {
		latency = fmt.Sprintf(" Average latency moved from `%s ms` to `%s ms`.", formatBenchmarkFloat(directStats.AvgLatencyMS), formatBenchmarkFloat(ndStats.AvgLatencyMS))
	}

	directFailures := benchmarkResearchDominantFailures(directStats)
	ndFailures := benchmarkResearchDominantFailures(ndStats)
	failures := ""
	if directFailures != "" || ndFailures != "" {
		failures = fmt.Sprintf(" Dominant direct failures: %s. Dominant ND failures: %s.", valueOrNA(directFailures), valueOrNA(ndFailures))
	}

	return fmt.Sprintf("- `%s`: %s %s%s%s\n", pair.LLMModel, lead, boundary, latency, failures), nil
}

func benchmarkResearchStatsFromRow(row benchmarkRunSummaryRow) (benchmarkResearchRowStats, error) {
	var stats benchmarkResearchRowStats
	var err error
	if stats.CasesTotal, err = parseBenchmarkSummaryInt(row.RunID, "cases_total", row.CasesTotal); err != nil {
		return stats, err
	}
	if stats.PassCount, err = parseBenchmarkSummaryInt(row.RunID, "pass_count", row.PassCount); err != nil {
		return stats, err
	}
	if stats.FalseRefusalCount, err = parseBenchmarkSummaryInt(row.RunID, "false_refusal_count", row.FalseRefusalCount); err != nil {
		return stats, err
	}
	if stats.FalseAcceptCount, err = parseBenchmarkSummaryInt(row.RunID, "false_accept_count", row.FalseAcceptCount); err != nil {
		return stats, err
	}
	if stats.RequestFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "request_failure_count", row.RequestFailureCount); err != nil {
		return stats, err
	}
	if stats.SchemaFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "schema_failure_count", row.SchemaFailureCount); err != nil {
		return stats, err
	}
	if stats.ParseFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "parse_failure_count", row.ParseFailureCount); err != nil {
		return stats, err
	}
	if stats.KernelFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "kernel_failure_count", row.KernelFailureCount); err != nil {
		return stats, err
	}
	if stats.ContractFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "contract_failure_count", row.ContractFailureCount); err != nil {
		return stats, err
	}
	if stats.FormatFailureCount, err = parseBenchmarkSummaryInt(row.RunID, "format_failure_count", row.FormatFailureCount); err != nil {
		return stats, err
	}
	if stats.AvgLatencyMS, _, err = parseBenchmarkSummaryFloat(row.RunID, "avg_latency_ms", row.AvgLatencyMS); err != nil {
		return stats, err
	}
	if stats.MaxLatencyMS, _, err = parseBenchmarkSummaryInt64(row.RunID, "max_latency_ms", row.MaxLatencyMS); err != nil {
		return stats, err
	}
	if stats.RunElapsedSeconds, _, err = parseBenchmarkSummaryInt64(row.RunID, "run_elapsed_seconds", row.RunElapsedSeconds); err != nil {
		return stats, err
	}
	return stats, nil
}

func benchmarkResearchDominantFailures(stats benchmarkResearchRowStats) string {
	type failureBucket struct {
		Label string
		Count int
	}
	buckets := []failureBucket{
		{Label: "false_accept", Count: stats.FalseAcceptCount},
		{Label: "false_refusal", Count: stats.FalseRefusalCount},
		{Label: "request_failure", Count: stats.RequestFailureCount},
		{Label: "schema_failure", Count: stats.SchemaFailureCount},
		{Label: "parse_failure", Count: stats.ParseFailureCount},
		{Label: "kernel_failure", Count: stats.KernelFailureCount},
		{Label: "contract_failure", Count: stats.ContractFailureCount},
		{Label: "format_failure", Count: stats.FormatFailureCount},
	}
	nonZero := make([]failureBucket, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.Count > 0 {
			nonZero = append(nonZero, bucket)
		}
	}
	if len(nonZero) == 0 {
		return "none"
	}
	slices.SortFunc(nonZero, func(a, b failureBucket) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Label, b.Label)
	})
	if len(nonZero) > 2 {
		nonZero = nonZero[:2]
	}
	parts := make([]string, 0, len(nonZero))
	for _, bucket := range nonZero {
		parts = append(parts, fmt.Sprintf("`%s=%d`", bucket.Label, bucket.Count))
	}
	return strings.Join(parts, ", ")
}

func benchmarkBestAggregate(aggregates []benchmarkRunSummaryAggregate) (benchmarkRunSummaryAggregate, bool) {
	if len(aggregates) == 0 {
		return benchmarkRunSummaryAggregate{}, false
	}
	best := aggregates[0]
	for _, aggregate := range aggregates[1:] {
		if benchmarkAggregateBetter(aggregate, best) {
			best = aggregate
		}
	}
	return best, true
}

func benchmarkAggregateBetter(candidate, current benchmarkRunSummaryAggregate) bool {
	candidateRate := benchmarkAggregatePassRate(candidate)
	currentRate := benchmarkAggregatePassRate(current)
	switch {
	case candidateRate > currentRate:
		return true
	case candidateRate < currentRate:
		return false
	case candidate.PassCount > current.PassCount:
		return true
	case candidate.PassCount < current.PassCount:
		return false
	default:
		return candidate.LatestRunID > current.LatestRunID
	}
}

func benchmarkAggregatePassRate(aggregate benchmarkRunSummaryAggregate) float64 {
	return benchmarkPassRate(aggregate.PassCount, aggregate.CasesTotal)
}
