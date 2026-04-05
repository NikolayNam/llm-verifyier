package analysis

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	GoldFirstPhase2Phase = "compositional-mixed-family-gold-first-phase2"

	defaultGoldFinalCollapseThreshold    = 0.60
	defaultNegativeTwinCollapseThreshold = 0.90
)

type GoldFirstPhase2Options struct {
	Phase                  string
	SummaryDir             string
	ValidityCSVPath        string
	ValidityMarkdownPath   string
	SubgroupCSVPath        string
	SubgroupMarkdownPath   string
	GoldCollapseThreshold  float64
	NegativeCollapseThresh float64
}

type GoldFirstPhase2Artifacts struct {
	Phase                string
	SummaryDir           string
	SummaryFiles         []string
	ValidityCSVPath      string
	ValidityMarkdownPath string
	SubgroupCSVPath      string
	SubgroupMarkdownPath string
	ValidityRows         []RunValidityRow
	SubgroupRows         []SubgroupRow
}

type RunValidityRow struct {
	SummaryRunID            string
	SummaryFile             string
	JobRows                 int
	ModelCount              int
	Models                  string
	TotalCases              int
	RequestFailureCount     int
	SchemaFailureCount      int
	KernelFailureCount      int
	ParseFailureCount       int
	FormatFailureCount      int
	FalseAcceptCount        int
	FalseRefusalCount       int
	RequestFailureRate      float64
	SchemaFailureRate       float64
	KernelFailureRate       float64
	GoldFinalPassRate       float64
	NegativeTwinPassRate    float64
	DominantSummaryFailure  string
	DominantChainFailure    string
	GoldFinalCollapsed      bool
	NegativeTwinCollapsed   bool
	Classification          string
	ClassificationReason    string
	ChainCount              int
	ReasoningEvidenceStatus string
}

type SubgroupRow struct {
	AnalysisScope       string
	AxisName            string
	AxisValue           string
	LabelScope          string
	LLMModel            string
	SourceSummaryRuns   string
	TotalCases          int
	PassCount           int
	PassRate            float64
	FailureCount        int
	DominantFailureType string
}

type summaryJobRow struct {
	RunID               string
	LLMModel            string
	CasesFile           string
	CasesTotal          int
	RequestFailureCount int
	SchemaFailureCount  int
	KernelFailureCount  int
	ParseFailureCount   int
	FormatFailureCount  int
	FalseAcceptCount    int
	FalseRefusalCount   int
	ResultFile          string
	ChainResultFile     string
}

type resultRow struct {
	RunID                string
	LLMModel             string
	CaseID               string
	StageID              string
	ExpectedLabel        string
	ScoreBucket          string
	TransitionGroup      string
	ImportArity          string
	ReuseShape           string
	BridgeDepth          string
	SymbolOverlap        string
	NegativeTwinHardness string
}

type chainRow struct {
	RunID                    string
	FinalCompositionPassGold bool
	NegativeTwinPass         bool
	FailureType              string
}

type summaryAggregate struct {
	JobRows                int
	Models                 []string
	TotalCases             int
	RequestFailureCount    int
	SchemaFailureCount     int
	KernelFailureCount     int
	ParseFailureCount      int
	FormatFailureCount     int
	FalseAcceptCount       int
	FalseRefusalCount      int
	DominantSummaryFailure string
	ResultFiles            []string
	ChainFiles             []string
}

type chainAggregate struct {
	ChainCount            int
	GoldFinalPassRate     float64
	NegativeTwinPassRate  float64
	DominantChainFailure  string
	GoldFinalCollapsed    bool
	NegativeTwinCollapsed bool
}

type subgroupAggregate struct {
	AnalysisScope     string
	AxisName          string
	AxisValue         string
	LabelScope        string
	LLMModel          string
	SourceSummaryRuns map[string]struct{}
	TotalCases        int
	PassCount         int
	FailureCounts     map[string]int
}

func AnalyzeGoldFirstPhase2(_ context.Context, opts GoldFirstPhase2Options) (*GoldFirstPhase2Artifacts, error) {
	phase := strings.TrimSpace(opts.Phase)
	if phase == "" {
		phase = GoldFirstPhase2Phase
	}
	summaryDir := filepath.Clean(strings.TrimSpace(opts.SummaryDir))
	if summaryDir == "" {
		return nil, fmt.Errorf("summary dir is required")
	}
	if strings.TrimSpace(opts.ValidityCSVPath) == "" || strings.TrimSpace(opts.ValidityMarkdownPath) == "" || strings.TrimSpace(opts.SubgroupCSVPath) == "" || strings.TrimSpace(opts.SubgroupMarkdownPath) == "" {
		return nil, fmt.Errorf("all analysis output paths are required")
	}
	goldCollapse := opts.GoldCollapseThreshold
	if goldCollapse <= 0 {
		goldCollapse = defaultGoldFinalCollapseThreshold
	}
	negativeCollapse := opts.NegativeCollapseThresh
	if negativeCollapse <= 0 {
		negativeCollapse = defaultNegativeTwinCollapseThreshold
	}

	summaryFiles, err := discoverCSVFiles(summaryDir)
	if err != nil {
		return nil, err
	}
	if len(summaryFiles) == 0 {
		return nil, fmt.Errorf("no summary csv files found in %s", filepath.ToSlash(summaryDir))
	}

	validityRows := make([]RunValidityRow, 0, len(summaryFiles))
	validResultFiles := map[string]string{}
	for _, summaryFile := range summaryFiles {
		summaryRows, err := readSummaryJobRows(summaryFile)
		if err != nil {
			return nil, err
		}
		if len(summaryRows) == 0 {
			continue
		}
		runID := summaryRunIDFromPath(summaryFile, phase)
		summaryAgg := aggregateSummaryRows(summaryRows)
		chainAgg, err := aggregateChainRows(summaryAgg.ChainFiles, goldCollapse, negativeCollapse)
		if err != nil {
			return nil, err
		}
		classification, reason, evidenceStatus := classifyRun(summaryAgg, chainAgg)
		validityRows = append(validityRows, RunValidityRow{
			SummaryRunID:            runID,
			SummaryFile:             filepath.ToSlash(summaryFile),
			JobRows:                 summaryAgg.JobRows,
			ModelCount:              len(summaryAgg.Models),
			Models:                  strings.Join(summaryAgg.Models, ", "),
			TotalCases:              summaryAgg.TotalCases,
			RequestFailureCount:     summaryAgg.RequestFailureCount,
			SchemaFailureCount:      summaryAgg.SchemaFailureCount,
			KernelFailureCount:      summaryAgg.KernelFailureCount,
			ParseFailureCount:       summaryAgg.ParseFailureCount,
			FormatFailureCount:      summaryAgg.FormatFailureCount,
			FalseAcceptCount:        summaryAgg.FalseAcceptCount,
			FalseRefusalCount:       summaryAgg.FalseRefusalCount,
			RequestFailureRate:      safeRate(summaryAgg.RequestFailureCount, summaryAgg.TotalCases),
			SchemaFailureRate:       safeRate(summaryAgg.SchemaFailureCount, summaryAgg.TotalCases),
			KernelFailureRate:       safeRate(summaryAgg.KernelFailureCount, summaryAgg.TotalCases),
			GoldFinalPassRate:       chainAgg.GoldFinalPassRate,
			NegativeTwinPassRate:    chainAgg.NegativeTwinPassRate,
			DominantSummaryFailure:  summaryAgg.DominantSummaryFailure,
			DominantChainFailure:    chainAgg.DominantChainFailure,
			GoldFinalCollapsed:      chainAgg.GoldFinalCollapsed,
			NegativeTwinCollapsed:   chainAgg.NegativeTwinCollapsed,
			Classification:          classification,
			ClassificationReason:    reason,
			ChainCount:              chainAgg.ChainCount,
			ReasoningEvidenceStatus: evidenceStatus,
		})
		if classification == "reasoning-valid" {
			for _, resultFile := range summaryAgg.ResultFiles {
				validResultFiles[resultFile] = runID
			}
		}
	}

	sort.Slice(validityRows, func(i, j int) bool {
		return validityRows[i].SummaryRunID < validityRows[j].SummaryRunID
	})

	subgroupRows, err := aggregateSubgroups(validResultFiles)
	if err != nil {
		return nil, err
	}

	artifacts := &GoldFirstPhase2Artifacts{
		Phase:                phase,
		SummaryDir:           filepath.ToSlash(summaryDir),
		SummaryFiles:         summaryFiles,
		ValidityCSVPath:      filepath.ToSlash(opts.ValidityCSVPath),
		ValidityMarkdownPath: filepath.ToSlash(opts.ValidityMarkdownPath),
		SubgroupCSVPath:      filepath.ToSlash(opts.SubgroupCSVPath),
		SubgroupMarkdownPath: filepath.ToSlash(opts.SubgroupMarkdownPath),
		ValidityRows:         validityRows,
		SubgroupRows:         subgroupRows,
	}
	if err := writeRunValidityCSV(artifacts.ValidityCSVPath, validityRows); err != nil {
		return nil, err
	}
	if err := writeRunValidityMarkdown(artifacts.ValidityMarkdownPath, validityRows, goldCollapse, negativeCollapse); err != nil {
		return nil, err
	}
	if err := writeSubgroupCSV(artifacts.SubgroupCSVPath, subgroupRows); err != nil {
		return nil, err
	}
	if err := writeSubgroupMarkdown(artifacts.SubgroupMarkdownPath, validityRows, subgroupRows); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func discoverCSVFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read summary dir %s: %w", filepath.ToSlash(dir), err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".csv") {
			continue
		}
		files = append(files, filepath.ToSlash(filepath.Join(dir, name)))
	}
	sort.Strings(files)
	return files, nil
}

func readSummaryJobRows(path string) ([]summaryJobRow, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}
	rows := make([]summaryJobRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, summaryJobRow{
			RunID:               record["run_id"],
			LLMModel:            record["llm_model"],
			CasesFile:           filepath.ToSlash(strings.TrimSpace(record["cases_file"])),
			CasesTotal:          parseInt(record["cases_total"]),
			RequestFailureCount: parseInt(record["request_failure_count"]),
			SchemaFailureCount:  parseInt(record["schema_failure_count"]),
			KernelFailureCount:  parseInt(record["kernel_failure_count"]),
			ParseFailureCount:   parseInt(record["parse_failure_count"]),
			FormatFailureCount:  parseInt(record["format_failure_count"]),
			FalseAcceptCount:    parseInt(record["false_accept_count"]),
			FalseRefusalCount:   parseInt(record["false_refusal_count"]),
			ResultFile:          filepath.ToSlash(strings.TrimSpace(record["result_file"])),
			ChainResultFile:     filepath.ToSlash(strings.TrimSpace(record["chain_result_file"])),
		})
	}
	return rows, nil
}

func readResultRows(path string) ([]resultRow, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}
	rows := make([]resultRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, resultRow{
			RunID:                record["run_id"],
			LLMModel:             record["llm_model"],
			CaseID:               record["case_id"],
			StageID:              record["stage_id"],
			ExpectedLabel:        record["expected_label"],
			ScoreBucket:          record["score_bucket"],
			TransitionGroup:      record["transition_group"],
			ImportArity:          record["import_arity"],
			ReuseShape:           record["reuse_shape"],
			BridgeDepth:          record["bridge_depth"],
			SymbolOverlap:        record["symbol_overlap"],
			NegativeTwinHardness: record["negative_twin_hardness"],
		})
	}
	return rows, nil
}

func readChainRows(path string) ([]chainRow, error) {
	records, err := readCSVRecords(path)
	if err != nil {
		return nil, err
	}
	rows := make([]chainRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, chainRow{
			RunID:                    record["run_id"],
			FinalCompositionPassGold: parseBool(record["final_composition_pass_gold"]),
			NegativeTwinPass:         parseBool(record["negative_twin_pass"]),
			FailureType:              strings.TrimSpace(record["failure_type"]),
		})
	}
	return rows, nil
}

func readCSVRecords(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	raw, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv %s: %w", filepath.ToSlash(path), err)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	headers := raw[0]
	records := make([]map[string]string, 0, max(0, len(raw)-1))
	for _, row := range raw[1:] {
		record := make(map[string]string, len(headers))
		for idx, header := range headers {
			value := ""
			if idx < len(row) {
				value = row[idx]
			}
			record[strings.TrimSpace(header)] = strings.TrimSpace(value)
		}
		records = append(records, record)
	}
	return records, nil
}

func aggregateSummaryRows(rows []summaryJobRow) summaryAggregate {
	modelSet := map[string]struct{}{}
	resultSet := map[string]struct{}{}
	chainSet := map[string]struct{}{}
	failureCounts := map[string]int{}
	agg := summaryAggregate{JobRows: len(rows)}
	for _, row := range rows {
		if model := strings.TrimSpace(row.LLMModel); model != "" {
			modelSet[model] = struct{}{}
		}
		agg.TotalCases += row.CasesTotal
		agg.RequestFailureCount += row.RequestFailureCount
		agg.SchemaFailureCount += row.SchemaFailureCount
		agg.KernelFailureCount += row.KernelFailureCount
		agg.ParseFailureCount += row.ParseFailureCount
		agg.FormatFailureCount += row.FormatFailureCount
		agg.FalseAcceptCount += row.FalseAcceptCount
		agg.FalseRefusalCount += row.FalseRefusalCount
		if row.RequestFailureCount > 0 {
			failureCounts["request_failure"] += row.RequestFailureCount
		}
		if row.SchemaFailureCount > 0 {
			failureCounts["schema_failure"] += row.SchemaFailureCount
		}
		if row.KernelFailureCount > 0 {
			failureCounts["kernel_failure"] += row.KernelFailureCount
		}
		if row.ParseFailureCount > 0 {
			failureCounts["parse_failure"] += row.ParseFailureCount
		}
		if row.FormatFailureCount > 0 {
			failureCounts["format_failure"] += row.FormatFailureCount
		}
		if row.FalseAcceptCount > 0 {
			failureCounts["false_accept"] += row.FalseAcceptCount
		}
		if row.FalseRefusalCount > 0 {
			failureCounts["false_refusal"] += row.FalseRefusalCount
		}
		if path := strings.TrimSpace(row.ResultFile); path != "" {
			resultSet[path] = struct{}{}
		}
		if path := strings.TrimSpace(row.ChainResultFile); path != "" {
			chainSet[path] = struct{}{}
		}
	}
	agg.Models = sortedKeys(modelSet)
	agg.ResultFiles = sortedKeys(resultSet)
	agg.ChainFiles = sortedKeys(chainSet)
	agg.DominantSummaryFailure = dominantFailure(failureCounts, "pass_only")
	return agg
}

func aggregateChainRows(paths []string, goldCollapse, negativeCollapse float64) (chainAggregate, error) {
	var goldPass, negativePass int
	failureCounts := map[string]int{}
	totalChains := 0
	for _, path := range paths {
		rows, err := readChainRows(path)
		if err != nil {
			return chainAggregate{}, err
		}
		for _, row := range rows {
			totalChains++
			if row.FinalCompositionPassGold {
				goldPass++
			}
			if row.NegativeTwinPass {
				negativePass++
			}
			if failure := strings.TrimSpace(row.FailureType); failure != "" {
				failureCounts[failure]++
			}
		}
	}
	goldRate := safeRate(goldPass, totalChains)
	negativeRate := safeRate(negativePass, totalChains)
	return chainAggregate{
		ChainCount:            totalChains,
		GoldFinalPassRate:     goldRate,
		NegativeTwinPassRate:  negativeRate,
		DominantChainFailure:  dominantFailure(failureCounts, "pass_only"),
		GoldFinalCollapsed:    totalChains > 0 && goldRate < goldCollapse,
		NegativeTwinCollapsed: totalChains > 0 && negativeRate < negativeCollapse,
	}, nil
}

func classifyRun(summary summaryAggregate, chain chainAggregate) (string, string, string) {
	if summary.TotalCases == 0 {
		return "ambiguous-needs-review", "summary file has zero total cases", "insufficient-artifacts"
	}
	requestRate := safeRate(summary.RequestFailureCount, summary.TotalCases)
	if summary.FalseAcceptCount > 0 {
		return "ambiguous-needs-review", "false_accept observed; requires manual review before reliability labeling", "manual-review"
	}
	if requestRate >= 0.20 {
		return "execution-invalidated", "request_failure_rate >= 0.20", "execution-invalidated"
	}
	if chain.GoldFinalCollapsed && chain.NegativeTwinCollapsed {
		return "execution-invalidated", "gold final and negative twin both collapsed", "execution-invalidated"
	}
	if requestRate <= 0.05 && !chain.NegativeTwinCollapsed && isReasoningFailureFamily(summary.DominantSummaryFailure) {
		return "reasoning-valid", "low request failure; negative twin stable; dominant failures stay in reasoning-visible buckets", "claim-bearing"
	}
	return "ambiguous-needs-review", "does not satisfy either clean execution-invalidated or reasoning-valid rule", "manual-review"
}

func aggregateSubgroups(validResultFiles map[string]string) ([]SubgroupRow, error) {
	if len(validResultFiles) == 0 {
		return nil, nil
	}
	groupMap := map[string]*subgroupAggregate{}
	for resultFile, summaryRunID := range validResultFiles {
		rows, err := readResultRows(resultFile)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			switch strings.TrimSpace(row.StageID) {
			case "fg":
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "fg", "transition_group", row.TransitionGroup, row.ScoreBucket, summaryRunID)
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "fg", "import_arity", row.ImportArity, row.ScoreBucket, summaryRunID)
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "fg", "reuse_shape", row.ReuseShape, row.ScoreBucket, summaryRunID)
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "fg", "bridge_depth", row.BridgeDepth, row.ScoreBucket, summaryRunID)
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "fg", "symbol_overlap", row.SymbolOverlap, row.ScoreBucket, summaryRunID)
			case "neg":
				accumulateSubgroup(groupMap, "reasoning-valid-runs", row.LLMModel, "neg", "negative_twin_hardness", row.NegativeTwinHardness, row.ScoreBucket, summaryRunID)
			}
		}
	}
	rows := make([]SubgroupRow, 0, len(groupMap))
	for _, agg := range groupMap {
		rows = append(rows, SubgroupRow{
			AnalysisScope:       agg.AnalysisScope,
			AxisName:            agg.AxisName,
			AxisValue:           agg.AxisValue,
			LabelScope:          agg.LabelScope,
			LLMModel:            agg.LLMModel,
			SourceSummaryRuns:   strings.Join(sortedKeys(agg.SourceSummaryRuns), ", "),
			TotalCases:          agg.TotalCases,
			PassCount:           agg.PassCount,
			PassRate:            safeRate(agg.PassCount, agg.TotalCases),
			FailureCount:        agg.TotalCases - agg.PassCount,
			DominantFailureType: dominantFailure(agg.FailureCounts, "pass_only"),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LLMModel != rows[j].LLMModel {
			return rows[i].LLMModel < rows[j].LLMModel
		}
		if rows[i].LabelScope != rows[j].LabelScope {
			return rows[i].LabelScope < rows[j].LabelScope
		}
		if rows[i].AxisName != rows[j].AxisName {
			return rows[i].AxisName < rows[j].AxisName
		}
		return rows[i].AxisValue < rows[j].AxisValue
	})
	return rows, nil
}

func accumulateSubgroup(groups map[string]*subgroupAggregate, scope, model, labelScope, axisName, axisValue, scoreBucket, summaryRunID string) {
	axisValue = strings.TrimSpace(axisValue)
	if axisValue == "" {
		return
	}
	key := strings.Join([]string{scope, model, labelScope, axisName, axisValue}, "|")
	agg, ok := groups[key]
	if !ok {
		agg = &subgroupAggregate{
			AnalysisScope:     scope,
			AxisName:          axisName,
			AxisValue:         axisValue,
			LabelScope:        labelScope,
			LLMModel:          strings.TrimSpace(model),
			SourceSummaryRuns: map[string]struct{}{},
			FailureCounts:     map[string]int{},
		}
		groups[key] = agg
	}
	agg.TotalCases++
	if strings.TrimSpace(scoreBucket) == "pass" {
		agg.PassCount++
	} else {
		agg.FailureCounts[strings.TrimSpace(scoreBucket)]++
	}
	if strings.TrimSpace(summaryRunID) != "" {
		agg.SourceSummaryRuns[strings.TrimSpace(summaryRunID)] = struct{}{}
	}
}

func writeRunValidityCSV(path string, rows []RunValidityRow) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create run validity csv %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"summary_run_id", "summary_file", "job_rows", "model_count", "models", "total_cases",
		"request_failure_count", "schema_failure_count", "kernel_failure_count", "parse_failure_count",
		"format_failure_count", "false_accept_count", "false_refusal_count",
		"request_failure_rate", "schema_failure_rate", "kernel_failure_rate",
		"chain_count", "gold_final_pass_rate", "negative_twin_pass_rate",
		"dominant_summary_failure", "dominant_chain_failure",
		"gold_final_collapsed", "negative_twin_collapsed",
		"classification", "classification_reason", "reasoning_evidence_status",
	}); err != nil {
		return fmt.Errorf("write run validity header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.SummaryRunID,
			filepath.ToSlash(row.SummaryFile),
			strconv.Itoa(row.JobRows),
			strconv.Itoa(row.ModelCount),
			row.Models,
			strconv.Itoa(row.TotalCases),
			strconv.Itoa(row.RequestFailureCount),
			strconv.Itoa(row.SchemaFailureCount),
			strconv.Itoa(row.KernelFailureCount),
			strconv.Itoa(row.ParseFailureCount),
			strconv.Itoa(row.FormatFailureCount),
			strconv.Itoa(row.FalseAcceptCount),
			strconv.Itoa(row.FalseRefusalCount),
			formatRate(row.RequestFailureRate),
			formatRate(row.SchemaFailureRate),
			formatRate(row.KernelFailureRate),
			strconv.Itoa(row.ChainCount),
			formatRate(row.GoldFinalPassRate),
			formatRate(row.NegativeTwinPassRate),
			row.DominantSummaryFailure,
			row.DominantChainFailure,
			strconv.FormatBool(row.GoldFinalCollapsed),
			strconv.FormatBool(row.NegativeTwinCollapsed),
			row.Classification,
			row.ClassificationReason,
			row.ReasoningEvidenceStatus,
		}); err != nil {
			return fmt.Errorf("write run validity row %q: %w", row.SummaryRunID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush run validity csv %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeRunValidityMarkdown(path string, rows []RunValidityRow, goldCollapse, negativeCollapse float64) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Gold-First Phase2 Run Validity\n\n")
	b.WriteString("Status: generated artifact  \n")
	b.WriteString("Scope: post-hoc reliability classification for existing `gold-first phase2` benchmark runs  \n")
	b.WriteString(fmt.Sprintf("Gold Final collapse threshold: `< %.2f`  \n", goldCollapse))
	b.WriteString(fmt.Sprintf("Negative Twin collapse threshold: `< %.2f`\n\n", negativeCollapse))
	b.WriteString("## Run Classification\n\n")
	b.WriteString("| Summary run | Classification | Request failure | Gold Final | Negative Twin | Dominant summary failure | Dominant chain failure | Notes |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | --- | --- | --- |\n")
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
			row.SummaryRunID, row.Classification, formatRate(row.RequestFailureRate), formatRate(row.GoldFinalPassRate),
			formatRate(row.NegativeTwinPassRate), row.DominantSummaryFailure, row.DominantChainFailure, row.ClassificationReason))
	}
	b.WriteString("\n## Notes\n\n")
	b.WriteString("- This batch is artifact-first and does not automatically inspect provider logs or transport traces.\n")
	b.WriteString("- `ambiguous-needs-review` means the current artifact-only rule-set is insufficient for a clean label.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSubgroupCSV(path string, rows []SubgroupRow) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create subgroup analysis csv %s: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"analysis_scope", "llm_model", "label_scope", "axis_name", "axis_value", "source_summary_runs", "total_cases", "pass_count", "pass_rate", "failure_count", "dominant_failure_type"}); err != nil {
		return fmt.Errorf("write subgroup analysis header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.AnalysisScope,
			row.LLMModel,
			row.LabelScope,
			row.AxisName,
			row.AxisValue,
			row.SourceSummaryRuns,
			strconv.Itoa(row.TotalCases),
			strconv.Itoa(row.PassCount),
			formatRate(row.PassRate),
			strconv.Itoa(row.FailureCount),
			row.DominantFailureType,
		}); err != nil {
			return fmt.Errorf("write subgroup row %s/%s/%s: %w", row.LLMModel, row.LabelScope, row.AxisName, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush subgroup csv %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeSubgroupMarkdown(path string, validityRows []RunValidityRow, subgroupRows []SubgroupRow) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Gold-First Phase2 Subgroup Analysis\n\n")
	b.WriteString("Status: generated artifact  \n")
	b.WriteString("Scope: subgroup analysis over `reasoning-valid` `gold-first phase2` runs only\n\n")

	validRuns := make([]string, 0, len(validityRows))
	for _, row := range validityRows {
		if row.Classification == "reasoning-valid" {
			validRuns = append(validRuns, row.SummaryRunID)
		}
	}
	sort.Strings(validRuns)
	b.WriteString("## Reasoning-Valid Runs Used\n\n")
	if len(validRuns) == 0 {
		b.WriteString("No `reasoning-valid` runs were available, so subgroup tables are empty.\n")
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	for _, runID := range validRuns {
		b.WriteString(fmt.Sprintf("- `%s`\n", runID))
	}
	b.WriteString("\n")

	writeSubgroupSection(&b, subgroupRows, "Gold Final pass rate by transition group", "fg", "transition_group")
	writeSubgroupSection(&b, subgroupRows, "Gold Final pass rate by import arity", "fg", "import_arity")
	writeSubgroupSection(&b, subgroupRows, "Gold Final pass rate by reuse shape", "fg", "reuse_shape")
	writeSubgroupSection(&b, subgroupRows, "Gold Final pass rate by bridge depth", "fg", "bridge_depth")
	writeSubgroupSection(&b, subgroupRows, "Gold Final pass rate by symbol overlap", "fg", "symbol_overlap")
	writeSubgroupSection(&b, subgroupRows, "Negative Twin pass rate by hardness", "neg", "negative_twin_hardness")

	hardest := hardestFGFailures(subgroupRows, 8)
	if len(hardest) > 0 {
		b.WriteString("## Hardest FG failures\n\n")
		b.WriteString("| Model | Axis | Value | Pass rate | Dominant failure |\n")
		b.WriteString("| --- | --- | --- | ---: | --- |\n")
		for _, row := range hardest {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` |\n", row.LLMModel, row.AxisName, row.AxisValue, formatRate(row.PassRate), row.DominantFailureType))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Notes\n\n")
	b.WriteString("- `FG failure-type mix` is represented through `dominant_failure_type` inside each subgroup row.\n")
	b.WriteString("- This batch aggregates only over `reasoning-valid` runs and excludes `execution-invalidated` runs from reasoning evidence.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSubgroupSection(b *strings.Builder, rows []SubgroupRow, title, labelScope, axisName string) {
	filtered := filterSubgroupRows(rows, labelScope, axisName)
	if len(filtered) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("## %s\n\n", title))
	b.WriteString("| Model | Value | Total | Pass | Pass rate | Dominant failure |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | --- |\n")
	for _, row := range filtered {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%d` | `%d` | `%s` | `%s` |\n", row.LLMModel, row.AxisValue, row.TotalCases, row.PassCount, formatRate(row.PassRate), row.DominantFailureType))
	}
	b.WriteString("\n")
}

func filterSubgroupRows(rows []SubgroupRow, labelScope, axisName string) []SubgroupRow {
	filtered := make([]SubgroupRow, 0, len(rows))
	for _, row := range rows {
		if row.LabelScope == labelScope && row.AxisName == axisName {
			filtered = append(filtered, row)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].LLMModel != filtered[j].LLMModel {
			return filtered[i].LLMModel < filtered[j].LLMModel
		}
		return filtered[i].AxisValue < filtered[j].AxisValue
	})
	return filtered
}

func hardestFGFailures(rows []SubgroupRow, limit int) []SubgroupRow {
	filtered := make([]SubgroupRow, 0, len(rows))
	for _, row := range rows {
		if row.LabelScope != "fg" || row.PassRate >= 1 {
			continue
		}
		filtered = append(filtered, row)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].PassRate != filtered[j].PassRate {
			return filtered[i].PassRate < filtered[j].PassRate
		}
		if filtered[i].LLMModel != filtered[j].LLMModel {
			return filtered[i].LLMModel < filtered[j].LLMModel
		}
		if filtered[i].AxisName != filtered[j].AxisName {
			return filtered[i].AxisName < filtered[j].AxisName
		}
		return filtered[i].AxisValue < filtered[j].AxisValue
	})
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func ensureParentDir(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func safeRate(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func formatRate(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func summaryRunIDFromPath(path, phase string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	prefix := strings.TrimSpace(phase) + "_"
	if strings.HasPrefix(base, prefix) {
		return strings.TrimPrefix(base, prefix)
	}
	return base
}

func dominantFailure(counts map[string]int, fallback string) string {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for name, count := range counts {
		if strings.TrimSpace(name) == "" || count <= 0 {
			continue
		}
		pairs = append(pairs, pair{name: name, count: count})
	}
	if len(pairs) == 0 {
		return fallback
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].name < pairs[j].name
	})
	return pairs[0].name
}

func isReasoningFailureFamily(name string) bool {
	switch strings.TrimSpace(name) {
	case "", "pass_only":
		return true
	case "schema_failure", "kernel_failure", "false_refusal", "parse_failure", "format_failure":
		return true
	default:
		return false
	}
}

func parseInt(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}

func parseBool(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return value
}

func sortedKeys[T any](set map[string]T) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, filepath.ToSlash(strings.TrimSpace(key)))
	}
	sort.Strings(keys)
	return keys
}
