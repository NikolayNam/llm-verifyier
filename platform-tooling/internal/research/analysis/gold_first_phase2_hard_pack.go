package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	GoldFirstPhase2HardPackPhase     = "compositional-mixed-family-gold-first-hard"
	GoldFirstPhase2HardPackCasesFile = "cases/compositional-mixed-family-gold-first-hard-phase2.csv"
)

var hardPackStrongModels = []string{
	"gpt-oss:120b-cloud",
	"glm-5:cloud",
	"deepseek-v3.1:671b-cloud",
}

type GoldFirstPhase2HardPackOptions struct {
	SummaryDir             string
	CasesFile              string
	ReportMarkdownPath     string
	GoldCollapseThreshold  float64
	NegativeCollapseThresh float64
}

type GoldFirstPhase2HardPackArtifacts struct {
	SummaryDir         string
	CasesFile          string
	SummaryFiles       []string
	ReportMarkdownPath string
	RunRows            []GoldFirstPhase2HardPackRunRow
	ModelRows          []GoldFirstPhase2HardPackModelRow
	EntailedChains     int
	NegativeTwinChains int
	TransitionGroups   []string
}

type GoldFirstPhase2HardPackRunRow struct {
	SummaryRunID           string
	SummaryFile            string
	JobRows                int
	Models                 string
	TotalCases             int
	RequestFailureRate     float64
	SchemaFailureRate      float64
	KernelFailureRate      float64
	GoldFinalPassRate      float64
	NegativeTwinPassRate   float64
	DominantSummaryFailure string
	DominantChainFailure   string
	Classification         string
	ClassificationReason   string
}

type GoldFirstPhase2HardPackModelRow struct {
	LLMModel               string
	JobRows                int
	TotalCases             int
	RequestFailureRate     float64
	SchemaFailureRate      float64
	KernelFailureRate      float64
	GoldFinalPassRate      float64
	NegativeTwinPassRate   float64
	DominantSummaryFailure string
	DominantChainFailure   string
}

func AnalyzeGoldFirstPhase2HardPack(_ context.Context, opts GoldFirstPhase2HardPackOptions) (*GoldFirstPhase2HardPackArtifacts, error) {
	summaryDir := filepath.Clean(strings.TrimSpace(opts.SummaryDir))
	if summaryDir == "" {
		return nil, fmt.Errorf("summary dir is required")
	}
	casesFile := normalizeAnalysisPath(strings.TrimSpace(opts.CasesFile))
	if casesFile == "" {
		casesFile = GoldFirstPhase2HardPackCasesFile
	}
	reportPath := filepath.Clean(strings.TrimSpace(opts.ReportMarkdownPath))
	if reportPath == "" {
		return nil, fmt.Errorf("report markdown path is required")
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

	runRows := make([]GoldFirstPhase2HardPackRunRow, 0, len(summaryFiles))
	modelGroups := map[string][]summaryJobRow{}
	resultFiles := map[string]struct{}{}
	transitionGroups := map[string]struct{}{}
	entailedChains := map[string]struct{}{}
	negativeChains := map[string]struct{}{}
	matchedFiles := make([]string, 0, len(summaryFiles))

	for _, summaryFile := range summaryFiles {
		rows, err := readSummaryJobRows(summaryFile)
		if err != nil {
			return nil, err
		}
		filtered := filterSummaryRowsByCasesFile(rows, casesFile)
		if len(filtered) == 0 {
			continue
		}
		matchedFiles = append(matchedFiles, summaryFile)
		summaryAgg := aggregateSummaryRows(filtered)
		chainAgg, err := aggregateChainRows(summaryAgg.ChainFiles, goldCollapse, negativeCollapse)
		if err != nil {
			return nil, err
		}
		classification, reason, _ := classifyRun(summaryAgg, chainAgg)
		runID := summaryRunIDFromPath(summaryFile, GoldFirstPhase2HardPackPhase)
		runRows = append(runRows, GoldFirstPhase2HardPackRunRow{
			SummaryRunID:           runID,
			SummaryFile:            filepath.ToSlash(summaryFile),
			JobRows:                summaryAgg.JobRows,
			Models:                 strings.Join(summaryAgg.Models, ", "),
			TotalCases:             summaryAgg.TotalCases,
			RequestFailureRate:     safeRate(summaryAgg.RequestFailureCount, summaryAgg.TotalCases),
			SchemaFailureRate:      safeRate(summaryAgg.SchemaFailureCount, summaryAgg.TotalCases),
			KernelFailureRate:      safeRate(summaryAgg.KernelFailureCount, summaryAgg.TotalCases),
			GoldFinalPassRate:      chainAgg.GoldFinalPassRate,
			NegativeTwinPassRate:   chainAgg.NegativeTwinPassRate,
			DominantSummaryFailure: summaryAgg.DominantSummaryFailure,
			DominantChainFailure:   chainAgg.DominantChainFailure,
			Classification:         classification,
			ClassificationReason:   reason,
		})
		for _, row := range filtered {
			modelGroups[row.LLMModel] = append(modelGroups[row.LLMModel], row)
		}
		for _, resultFile := range summaryAgg.ResultFiles {
			if strings.TrimSpace(resultFile) == "" {
				continue
			}
			resultFiles[resultFile] = struct{}{}
		}
	}
	if len(runRows) == 0 {
		return nil, fmt.Errorf("no summary rows matched cases_file %q in %s", casesFile, filepath.ToSlash(summaryDir))
	}
	for resultFile := range resultFiles {
		rows, err := readResultRows(resultFile)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			switch strings.TrimSpace(row.StageID) {
			case "fg":
				if strings.TrimSpace(row.CaseID) != "" {
					entailedChains[row.CaseID] = struct{}{}
				}
				if value := strings.TrimSpace(row.TransitionGroup); value != "" {
					transitionGroups[value] = struct{}{}
				}
			case "neg":
				if strings.TrimSpace(row.CaseID) != "" {
					negativeChains[row.CaseID] = struct{}{}
				}
			}
		}
	}

	modelRows := make([]GoldFirstPhase2HardPackModelRow, 0, len(modelGroups))
	for model, rows := range modelGroups {
		summaryAgg := aggregateSummaryRows(rows)
		chainAgg, err := aggregateChainRows(summaryAgg.ChainFiles, goldCollapse, negativeCollapse)
		if err != nil {
			return nil, err
		}
		modelRows = append(modelRows, GoldFirstPhase2HardPackModelRow{
			LLMModel:               model,
			JobRows:                summaryAgg.JobRows,
			TotalCases:             summaryAgg.TotalCases,
			RequestFailureRate:     safeRate(summaryAgg.RequestFailureCount, summaryAgg.TotalCases),
			SchemaFailureRate:      safeRate(summaryAgg.SchemaFailureCount, summaryAgg.TotalCases),
			KernelFailureRate:      safeRate(summaryAgg.KernelFailureCount, summaryAgg.TotalCases),
			GoldFinalPassRate:      chainAgg.GoldFinalPassRate,
			NegativeTwinPassRate:   chainAgg.NegativeTwinPassRate,
			DominantSummaryFailure: summaryAgg.DominantSummaryFailure,
			DominantChainFailure:   chainAgg.DominantChainFailure,
		})
	}

	sort.Slice(runRows, func(i, j int) bool {
		return runRows[i].SummaryRunID < runRows[j].SummaryRunID
	})
	sort.Slice(modelRows, func(i, j int) bool {
		if modelRows[i].GoldFinalPassRate != modelRows[j].GoldFinalPassRate {
			return modelRows[i].GoldFinalPassRate > modelRows[j].GoldFinalPassRate
		}
		return modelRows[i].LLMModel < modelRows[j].LLMModel
	})

	artifacts := &GoldFirstPhase2HardPackArtifacts{
		SummaryDir:         filepath.ToSlash(summaryDir),
		CasesFile:          casesFile,
		SummaryFiles:       matchedFiles,
		ReportMarkdownPath: filepath.ToSlash(reportPath),
		RunRows:            runRows,
		ModelRows:          modelRows,
		EntailedChains:     len(entailedChains),
		NegativeTwinChains: len(negativeChains),
		TransitionGroups:   sortedKeys(transitionGroups),
	}
	if err := writeGoldFirstPhase2HardPackMarkdown(reportPath, artifacts); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func filterSummaryRowsByCasesFile(rows []summaryJobRow, casesFile string) []summaryJobRow {
	filtered := make([]summaryJobRow, 0, len(rows))
	for _, row := range rows {
		if sameAnalysisPath(row.CasesFile, casesFile) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func sameAnalysisPath(left, right string) bool {
	left = normalizeAnalysisPath(left)
	right = normalizeAnalysisPath(right)
	if left == "" || right == "" {
		return false
	}
	return left == right || strings.HasSuffix(left, "/"+right) || strings.HasSuffix(right, "/"+left)
}

func normalizeAnalysisPath(value string) string {
	return filepath.ToSlash(strings.TrimSpace(value))
}

func writeGoldFirstPhase2HardPackMarkdown(path string, artifacts *GoldFirstPhase2HardPackArtifacts) error {
	if artifacts == nil {
		return fmt.Errorf("hard-pack artifacts are required")
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Gold-First Phase2 Hard Pack Report\n\n")
	b.WriteString("Status: generated artifact  \n")
	b.WriteString("Scope: hard-transition micro-pack derived from the heaviest `gold-first phase2` transition groups\n\n")
	b.WriteString(fmt.Sprintf("- cases_file: `%s`\n", artifacts.CasesFile))
	b.WriteString(fmt.Sprintf("- summary_dir: `%s`\n", artifacts.SummaryDir))
	b.WriteString(fmt.Sprintf("- matched summary runs: `%d`\n", len(artifacts.RunRows)))
	b.WriteString(fmt.Sprintf("- entailed chains: `%d`\n", artifacts.EntailedChains))
	b.WriteString(fmt.Sprintf("- hard negative twins: `%d`\n", artifacts.NegativeTwinChains))
	if len(artifacts.TransitionGroups) > 0 {
		b.WriteString(fmt.Sprintf("- transition groups: `%s`\n", strings.Join(artifacts.TransitionGroups, "`, `")))
	}
	b.WriteString("\n")

	b.WriteString("## Matched Runs\n\n")
	b.WriteString("| Summary run | Classification | Request failure | Schema failure | Kernel failure | Gold Final | Negative Twin | Notes |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, row := range artifacts.RunRows {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
			row.SummaryRunID,
			row.Classification,
			formatRate(row.RequestFailureRate),
			formatRate(row.SchemaFailureRate),
			formatRate(row.KernelFailureRate),
			formatRate(row.GoldFinalPassRate),
			formatRate(row.NegativeTwinPassRate),
			row.ClassificationReason,
		))
	}
	b.WriteString("\n")

	b.WriteString("## Model Summary\n\n")
	b.WriteString("| Model | Jobs | Cases | Gold Final | Negative Twin | Request failure | Schema failure | Dominant summary failure | Dominant chain failure |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |\n")
	for _, row := range artifacts.ModelRows {
		b.WriteString(fmt.Sprintf("| `%s` | `%d` | `%d` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n",
			row.LLMModel,
			row.JobRows,
			row.TotalCases,
			formatRate(row.GoldFinalPassRate),
			formatRate(row.NegativeTwinPassRate),
			formatRate(row.RequestFailureRate),
			formatRate(row.SchemaFailureRate),
			row.DominantSummaryFailure,
			row.DominantChainFailure,
		))
	}
	b.WriteString("\n")

	writeGoldFirstPhase2HardPackCriteria(&b, artifacts.ModelRows)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeGoldFirstPhase2HardPackCriteria(b *strings.Builder, rows []GoldFirstPhase2HardPackModelRow) {
	if len(rows) == 0 {
		return
	}
	b.WriteString("## Success Criteria\n\n")
	best := rows[0]
	status := "pass"
	if best.GoldFinalPassRate < 0.60 || best.NegativeTwinPassRate < 0.95 {
		status = "fail"
	}
	b.WriteString(fmt.Sprintf("- Best model: `%s` -> Gold Final `%s`, Negative Twin `%s` => `%s`\n",
		best.LLMModel, formatRate(best.GoldFinalPassRate), formatRate(best.NegativeTwinPassRate), status))
	if len(rows) > 1 {
		second := rows[1]
		secondStatus := "pass"
		if second.GoldFinalPassRate < 0.45 {
			secondStatus = "fail"
		}
		b.WriteString(fmt.Sprintf("- Second model: `%s` -> Gold Final `%s` => `%s`\n",
			second.LLMModel, formatRate(second.GoldFinalPassRate), secondStatus))
	}
	if weakestStrong, ok := weakestStrongModel(rows); ok {
		b.WriteString(fmt.Sprintf("- Weakest strong model among `gpt-oss:120b-cloud`, `glm-5:cloud`, `deepseek-v3.1:671b-cloud`: `%s` with Gold Final `%s`.\n",
			weakestStrong.LLMModel, formatRate(weakestStrong.GoldFinalPassRate)))
		b.WriteString("- The `above random/trivial level` criterion still requires manual interpretation; this report surfaces the exact rate rather than inventing a new threshold.\n")
	}
	b.WriteString("\n")

	b.WriteString("## Kill Criteria\n\n")
	bestKill := "not triggered"
	if best.GoldFinalPassRate < 0.40 {
		bestKill = "triggered"
	}
	b.WriteString(fmt.Sprintf("- Best-model collapse below `0.40`: `%s` (`%s` at `%s`).\n",
		bestKill, best.LLMModel, formatRate(best.GoldFinalPassRate)))
	negativeBoundary := "not triggered"
	for _, row := range rows {
		if row.NegativeTwinPassRate < 0.95 {
			negativeBoundary = "triggered"
			break
		}
	}
	b.WriteString(fmt.Sprintf("- Hard negative twins breaking the refusal boundary (`Negative Twin < 0.95` on any model): `%s`.\n", negativeBoundary))
	b.WriteString("- `Easy-case phenomenon` is only partially testable here; this pack is already restricted to the heaviest `GF-MF-D/E` slice, so the remaining question is whether the strongest models still sustain signal under that narrowed distribution.\n")
	b.WriteString("\n")

	b.WriteString("## Notes\n\n")
	b.WriteString("- This report is filtered by `cases_file`, so it does not mix the new hard-phase2 slice with the older full hard pack or the earlier hard-micro pack.\n")
	b.WriteString("- The pack intentionally combines the hardest `GF-MF-D` and `GF-MF-E` phase2 groups: `GF-MF-D` contributes deeper bridge stress, while `GF-MF-E` contributes lower symbol overlap and `import_arity >= 2` adversarial reuse.\n")
}

func weakestStrongModel(rows []GoldFirstPhase2HardPackModelRow) (GoldFirstPhase2HardPackModelRow, bool) {
	filtered := make([]GoldFirstPhase2HardPackModelRow, 0, len(hardPackStrongModels))
	for _, target := range hardPackStrongModels {
		for _, row := range rows {
			if row.LLMModel == target {
				filtered = append(filtered, row)
				break
			}
		}
	}
	if len(filtered) == 0 {
		return GoldFirstPhase2HardPackModelRow{}, false
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].GoldFinalPassRate != filtered[j].GoldFinalPassRate {
			return filtered[i].GoldFinalPassRate < filtered[j].GoldFinalPassRate
		}
		return filtered[i].LLMModel < filtered[j].LLMModel
	})
	return filtered[0], true
}
