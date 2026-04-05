package main

import (
	"fmt"
	"io"
	"strings"
)

// Observed-data rendering is kept separate from the top-level benchmark report
// so that summary-first report entrypoints remain easier to scan.
func renderBenchmarkObservedSectionsImpl(b *strings.Builder, projectFolder string, data *benchmarkObservedData) {
	if data == nil {
		return
	}
	if len(data.Warnings) > 0 {
		b.WriteString("### Observed-data warnings\n\n")
		for _, warning := range data.Warnings {
			b.WriteString(fmt.Sprintf("- %s\n", warning))
		}
		b.WriteString("\n")
	}
	renderBenchmarkResultSliceSection(b, "Entailed-only", "Deduplicated result-row slice filtered to `expected_label = entailed`.", data.Entailed)
	renderBenchmarkResultSliceSection(b, "Not-entailed-only", "Deduplicated result-row slice filtered to `expected_label = not_entailed`.", data.NotEntailed)
	renderBenchmarkFocusedCategorySections(b, projectFolder, data)
	b.WriteString("### Per-category\n\n")
	if len(data.ByCategory) == 0 {
		b.WriteString("No category-level observed slices were available.\n\n")
	} else {
		for _, category := range sortedBenchmarkResultSliceKeys(data.ByCategory) {
			renderBenchmarkResultSliceSection(b, fmt.Sprintf("Category: `%s`", category), "", data.ByCategory[category])
		}
	}
	renderBenchmarkNDPipelineTraceSection(b, data.NDPipelineTrace)
	renderBenchmarkPerCaseHardestFailuresSection(b, data.HardestFailures)
	renderBenchmarkHardCaseAuditSection(b, data.HardCases)
}

func renderBenchmarkFocusedCategorySectionsImpl(b *strings.Builder, projectFolder string, data *benchmarkObservedData) {
	if data == nil || strings.TrimSpace(projectFolder) != benchmarkNDProjectFolder {
		return
	}
	if theoremSynthesis := data.ByCategory["theorem_synthesis"]; len(theoremSynthesis) > 0 {
		renderBenchmarkResultSliceSection(b, "Theorem-synthesis focus", "Dedicated slice for theorem-construction cases. Interpret this separately from the mixed aggregate before making any broad ND claim.", theoremSynthesis)
	}

	assumptionImport := data.ByCategory["assumption_import"]
	singleMP := data.ByCategory["single_mp"]
	if len(assumptionImport) == 0 && len(singleMP) == 0 {
		return
	}
	b.WriteString("### Foundational-category audit\n\n")
	b.WriteString("These two categories isolate the cleanest local-inference authoring cases and should be checked separately from theorem-synthesis behavior.\n\n")
	if len(assumptionImport) > 0 {
		renderBenchmarkResultSliceSection(b, "Audit: `assumption_import`", "", assumptionImport)
	}
	if len(singleMP) > 0 {
		renderBenchmarkResultSliceSection(b, "Audit: `single_mp`", "", singleMP)
	}
}

func renderBenchmarkNDPipelineTraceSectionImpl(b *strings.Builder, aggregates []benchmarkNDPipelineTraceAggregate) {
	if len(aggregates) == 0 {
		return
	}
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkNDPipelineTraceAggregate) string { return item.Surface })
	showSurfaceColumn := len(surfaces) > 1
	b.WriteString("### ND pipeline trace\n\n")
	b.WriteString("Failure routing across the ND pipeline. This separates ND proof-object failures, lowering failures, and Hilbert-side artifact failures after lowering. Legacy rows without explicit pipeline metadata are classified from `raw_output_kind` and recorded status fields, so treat those counts as inferred rather than first-class stage telemetry.\n\n")
	renderBenchmarkSectionContextWithSurface(
		b,
		benchmarkCollectDistinct(aggregates, func(item benchmarkNDPipelineTraceAggregate) string { return item.PromptVersion }),
		nil,
		func() []string {
			if showSurfaceColumn {
				return nil
			}
			return surfaces
		}(),
		benchmarkLatestTimestampFrom(aggregates, func(item benchmarkNDPipelineTraceAggregate) string { return item.LatestTimestampUTC }),
	)
	if showSurfaceColumn {
		b.WriteString("| Model | Surface | Runs | Cases | ND Proof Object Failure | Lowering Failure | Hilbert Artifact Failure | Refusal Before ND Proof | Format Before ND Proof | Request Failure | Unknown / Legacy Stage |\n")
		b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	} else {
		b.WriteString("| Model | Runs | Cases | ND Proof Object Failure | Lowering Failure | Hilbert Artifact Failure | Refusal Before ND Proof | Format Before ND Proof | Request Failure | Unknown / Legacy Stage |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	}
	for _, aggregate := range aggregates {
		if showSurfaceColumn {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				valueOrNA(aggregate.Surface),
				aggregate.Runs,
				aggregate.CasesTotal,
				aggregate.NDProofObjectFailures,
				aggregate.LoweringFailures,
				aggregate.HilbertArtifactFailures,
				aggregate.OutputRefusals,
				aggregate.OutputFormatFailures,
				aggregate.RequestFailures,
				aggregate.UnknownStageRows,
			))
			continue
		}
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.NDProofObjectFailures,
			aggregate.LoweringFailures,
			aggregate.HilbertArtifactFailures,
			aggregate.OutputRefusals,
			aggregate.OutputFormatFailures,
			aggregate.RequestFailures,
			aggregate.UnknownStageRows,
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkResultSliceSectionImpl(b *strings.Builder, title, description string, aggregates []benchmarkResultSliceAggregate) {
	b.WriteString(fmt.Sprintf("### %s\n\n", title))
	if strings.TrimSpace(description) != "" {
		b.WriteString(description + "\n\n")
	}
	if len(aggregates) == 0 {
		b.WriteString("No matching observed result rows.\n\n")
		return
	}
	surfaces := benchmarkCollectDistinct(aggregates, func(item benchmarkResultSliceAggregate) string { return item.Surface })
	showSurfaceColumn := len(surfaces) > 1
	renderBenchmarkSectionContextWithSurface(
		b,
		benchmarkCollectDistinct(aggregates, func(item benchmarkResultSliceAggregate) string { return item.PromptVersion }),
		nil,
		func() []string {
			if showSurfaceColumn {
				return nil
			}
			return surfaces
		}(),
		benchmarkLatestTimestampFrom(aggregates, func(item benchmarkResultSliceAggregate) string { return item.LatestTimestampUTC }),
	)
	if showSurfaceColumn {
		b.WriteString("| Model | Surface | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Latest Run (UTC) |\n")
		b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	} else {
		b.WriteString("| Model | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Latest Run (UTC) |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	}
	for _, aggregate := range aggregates {
		b.WriteString(renderBenchmarkResultSliceMarkdownRowImpl(aggregate, showSurfaceColumn))
	}
	b.WriteString("\n")
}

func renderBenchmarkResultSliceMarkdownRowImpl(aggregate benchmarkResultSliceAggregate, showSurfaceColumn bool) string {
	passRate := benchmarkPassRate(aggregate.PassCount, aggregate.CasesTotal)
	avgLatencyMS := 0.0
	if aggregate.LatencyCasesTotal > 0 {
		avgLatencyMS = aggregate.LatencySumMS / float64(aggregate.LatencyCasesTotal)
	}
	if showSurfaceColumn {
		return fmt.Sprintf("| `%s` | `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %s | %s | `%s` |\n",
			benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
			valueOrNA(aggregate.Surface),
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
			formatBenchmarkFloat(aggregate.MaxLatencyMS),
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
		)
	}
	return fmt.Sprintf("| `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %s | %s | `%s` |\n",
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
		formatBenchmarkFloat(aggregate.MaxLatencyMS),
		displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
	)
}

func renderBenchmarkHardCaseAuditSectionImpl(b *strings.Builder, audits []benchmarkHardCaseAudit) {
	b.WriteString("### Hard-case audit targets\n\n")
	if len(audits) == 0 {
		b.WriteString("No hard-case audit targets were available.\n\n")
		return
	}
	for _, audit := range audits {
		caseID := valueOrNA(firstNonEmptyBenchmarkCell(audit.TargetCaseID, audit.Case.CaseID))
		lineageFallback := strings.TrimSpace(audit.TargetCaseID) != "" && strings.TrimSpace(audit.TargetCaseID) != strings.TrimSpace(audit.Case.CaseID)
		label := valueOrNA(audit.Case.Label)
		category := valueOrNA(audit.Case.Category)
		difficulty := valueOrNA(audit.Case.Difficulty)
		goal := valueOrNA(audit.Case.Goal)
		b.WriteString(fmt.Sprintf("#### `%s` — `%s` / `%s` / `%s`\n\n", caseID, label, category, difficulty))
		if lineageFallback {
			b.WriteString(fmt.Sprintf("- Representative descendant case: `%s`\n", valueOrNA(audit.Case.CaseID)))
		}
		if len(audit.ObservedCaseIDs) > 0 {
			b.WriteString(fmt.Sprintf("- Observed descendant case ids: `%s`\n", strings.Join(audit.ObservedCaseIDs, "`, `")))
		}
		if lineageFallback {
			b.WriteString(fmt.Sprintf("- Representative descendant goal: `%s`\n", goal))
		} else {
			b.WriteString(fmt.Sprintf("- Goal: `%s`\n", goal))
		}
		if len(audit.Case.Assumptions) > 0 {
			if lineageFallback {
				b.WriteString(fmt.Sprintf("- Representative descendant assumptions: `%s`\n", strings.Join(audit.Case.Assumptions, "`, `")))
			} else {
				b.WriteString(fmt.Sprintf("- Assumptions: `%s`\n", strings.Join(audit.Case.Assumptions, "`, `")))
			}
		}
		if strings.TrimSpace(audit.Case.Comment) != "" {
			if lineageFallback {
				b.WriteString(fmt.Sprintf("- Representative descendant comment: %s\n", audit.Case.Comment))
			} else {
				b.WriteString(fmt.Sprintf("- Comment: %s\n", audit.Case.Comment))
			}
		}
		b.WriteString("\n")
		if len(audit.Aggregates) == 0 {
			b.WriteString("No recorded observations for this hard-case target.\n\n")
			continue
		}
		renderBenchmarkSectionContext(
			b,
			benchmarkCollectDistinct(audit.Aggregates, func(item benchmarkHardCaseAuditAggregate) string { return item.PromptVersion }),
			nil,
			benchmarkLatestTimestampFrom(audit.Aggregates, func(item benchmarkHardCaseAuditAggregate) string { return item.LatestTimestampUTC }),
		)
		surfaces := benchmarkCollectDistinct(audit.Aggregates, func(item benchmarkHardCaseAuditAggregate) string { return item.Surface })
		showSurfaceColumn := len(surfaces) > 1
		if !showSurfaceColumn && len(surfaces) == 1 {
			b.WriteString(fmt.Sprintf("- Surface: `%s`\n\n", surfaces[0]))
		}
		if showSurfaceColumn {
			b.WriteString("| Model | Surface | Runs | Pass | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Latest Run (UTC) | Latest Result | Latest Raw |\n")
			b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |\n")
		} else {
			b.WriteString("| Model | Runs | Pass | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Latest Run (UTC) | Latest Result | Latest Raw |\n")
			b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |\n")
		}
		for _, aggregate := range audit.Aggregates {
			if showSurfaceColumn {
				b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | `%s` | `%s` | `%s` |\n",
					benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
					valueOrNA(aggregate.Surface),
					aggregate.Runs,
					aggregate.PassCount,
					aggregate.FalseRefusalCount,
					aggregate.FalseAcceptCount,
					aggregate.RequestFailureCount,
					aggregate.SchemaFailureCount,
					aggregate.ParseFailureCount,
					aggregate.KernelFailureCount,
					aggregate.ContractFailureCount,
					aggregate.FormatFailureCount,
					displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
					valueOrNA(aggregate.LatestResultPath),
					valueOrNA(aggregate.LatestRawDir),
				))
				continue
			}
			b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | `%s` | `%s` | `%s` |\n",
				benchmarkReportModelLabel(aggregate.DisplayName, aggregate.LLMModel, aggregate.Model),
				aggregate.Runs,
				aggregate.PassCount,
				aggregate.FalseRefusalCount,
				aggregate.FalseAcceptCount,
				aggregate.RequestFailureCount,
				aggregate.SchemaFailureCount,
				aggregate.ParseFailureCount,
				aggregate.KernelFailureCount,
				aggregate.ContractFailureCount,
				aggregate.FormatFailureCount,
				displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
				valueOrNA(aggregate.LatestResultPath),
				valueOrNA(aggregate.LatestRawDir),
			))
		}
		b.WriteString("\n")
	}
}

func renderBenchmarkPerCaseHardestFailuresSectionImpl(b *strings.Builder, aggregates []benchmarkCaseFailureAggregate) {
	b.WriteString("### Per-case hardest failures\n\n")
	b.WriteString("Ranked by lowest pass rate, then unsafe acceptance burden, then total failures.\n\n")
	if len(aggregates) == 0 {
		b.WriteString("No per-case failures were observed.\n\n")
		return
	}
	b.WriteString("| Case ID | Category | Label | Difficulty | Runs | Observations | Pass | Pass Rate | Failures | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Contract Failure | Format Failure | Latest Run (UTC) | Latest Result | Latest Raw |\n")
	b.WriteString("| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |\n")
	for _, aggregate := range aggregates {
		passRate := benchmarkPassRate(aggregate.PassCount, aggregate.CasesTotal)
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %d | %d | %d | %.2f%% | %d | %d | %d | %d | %d | %d | %d | %d | %d | `%s` | `%s` | `%s` |\n",
			valueOrNA(aggregate.Case.CaseID),
			valueOrNA(aggregate.Case.Category),
			valueOrNA(aggregate.Case.Label),
			valueOrNA(aggregate.Case.Difficulty),
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.PassCount,
			passRate,
			aggregate.FailureCount,
			aggregate.FalseRefusalCount,
			aggregate.FalseAcceptCount,
			aggregate.RequestFailureCount,
			aggregate.SchemaFailureCount,
			aggregate.ParseFailureCount,
			aggregate.KernelFailureCount,
			aggregate.ContractFailureCount,
			aggregate.FormatFailureCount,
			displayBenchmarkTimestamp(aggregate.LatestTimestampUTC),
			valueOrNA(aggregate.LatestResultPath),
			valueOrNA(aggregate.LatestRawDir),
		))
	}
	b.WriteString("\n")
}

func renderBenchmarkObservedSummaryImpl(stdout io.Writer, data *benchmarkObservedData) {
	if data == nil {
		return
	}
	for _, warning := range data.Warnings {
		_, _ = fmt.Fprintf(stdout, "observed_warning: %s\n", warning)
	}
	printBenchmarkResultSliceSummary(stdout, "aggregate", data.Aggregate)
	printBenchmarkResultSliceSummary(stdout, "entailed-only", data.Entailed)
	printBenchmarkResultSliceSummary(stdout, "not-entailed-only", data.NotEntailed)
	if theoremSynthesis := data.ByCategory["theorem_synthesis"]; len(theoremSynthesis) > 0 {
		printBenchmarkResultSliceSummary(stdout, "theorem_synthesis-focus", theoremSynthesis)
	}
	if assumptionImport := data.ByCategory["assumption_import"]; len(assumptionImport) > 0 {
		printBenchmarkResultSliceSummary(stdout, "audit=assumption_import", assumptionImport)
	}
	if singleMP := data.ByCategory["single_mp"]; len(singleMP) > 0 {
		printBenchmarkResultSliceSummary(stdout, "audit=single_mp", singleMP)
	}
	for _, category := range sortedBenchmarkResultSliceKeys(data.ByCategory) {
		printBenchmarkResultSliceSummary(stdout, "category="+category, data.ByCategory[category])
	}
	printBenchmarkNDPipelineTraceSummary(stdout, data.NDPipelineTrace)
}

func printBenchmarkResultSliceSummaryImpl(stdout io.Writer, label string, aggregates []benchmarkResultSliceAggregate) {
	_, _ = fmt.Fprintf(stdout, "slice[%s]: groups=%d\n", label, len(aggregates))
	for _, aggregate := range aggregates {
		passRate := benchmarkPassRate(aggregate.PassCount, aggregate.CasesTotal)
		avgLatencyMS := 0.0
		if aggregate.LatencyCasesTotal > 0 {
			avgLatencyMS = aggregate.LatencySumMS / float64(aggregate.LatencyCasesTotal)
		}
		_, _ = fmt.Fprintf(
			stdout,
			"slice_group[%s][model=%s llm_model=%s prompt_version=%s surface=%s]: runs=%d cases_total=%d pass=%d pass_rate=%.2f%% false_refusal=%d false_accept=%d request_failure=%d schema_failure=%d parse_failure=%d kernel_failure=%d format_failure=%d avg_latency_ms=%s max_latency_ms=%s latest_run_id=%s latest_timestamp=%s\n",
			label,
			aggregate.Model,
			aggregate.LLMModel,
			aggregate.PromptVersion,
			aggregate.Surface,
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
			aggregate.FormatFailureCount,
			formatBenchmarkFloat(avgLatencyMS),
			formatBenchmarkFloat(aggregate.MaxLatencyMS),
			aggregate.LatestRunID,
			aggregate.LatestTimestampUTC,
		)
	}
}

func printBenchmarkNDPipelineTraceSummaryImpl(stdout io.Writer, aggregates []benchmarkNDPipelineTraceAggregate) {
	if len(aggregates) == 0 {
		return
	}
	_, _ = fmt.Fprintf(stdout, "nd_pipeline_trace: groups=%d\n", len(aggregates))
	for _, aggregate := range aggregates {
		_, _ = fmt.Fprintf(
			stdout,
			"nd_trace[model=%s llm_model=%s prompt_version=%s surface=%s]: runs=%d cases_total=%d nd_proof_object_failure=%d lowering_failure=%d hilbert_artifact_failure=%d refusal_before_nd_proof=%d format_before_nd_proof=%d request_failure=%d unknown_stage=%d latest_run_id=%s latest_timestamp=%s\n",
			aggregate.Model,
			aggregate.LLMModel,
			aggregate.PromptVersion,
			aggregate.Surface,
			aggregate.Runs,
			aggregate.CasesTotal,
			aggregate.NDProofObjectFailures,
			aggregate.LoweringFailures,
			aggregate.HilbertArtifactFailures,
			aggregate.OutputRefusals,
			aggregate.OutputFormatFailures,
			aggregate.RequestFailures,
			aggregate.UnknownStageRows,
			aggregate.LatestRunID,
			aggregate.LatestTimestampUTC,
		)
	}
}
