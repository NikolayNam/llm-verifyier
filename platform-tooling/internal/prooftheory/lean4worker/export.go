package lean4worker

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type BenchmarkCaseExportConfig struct {
	ModeFilter      string
	InterestingOnly bool
	MinimalOnly     bool
	Limit           int
}

type BenchmarkCase struct {
	TheoremPackID string
	TheoremID     string
	CaseID        string
	Category      string
	Label         string
	Difficulty    string
	Assumptions   []string
	Goal          string
	LogicFragment string
	Comment       string
}

func LoadHypothesisCasesCSV(path string) ([]HypothesisCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open lean4 hypotheses csv: %w", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read lean4 hypotheses csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("lean4 hypotheses csv is empty")
	}

	columns := make(map[string]int, len(records[0]))
	for idx, header := range records[0] {
		columns[strings.TrimSpace(header)] = idx
	}
	required := []string{
		"generation_mode",
		"category",
		"derivability_status",
		"interesting",
		"minimal",
		"difficulty",
		"assumptions_json",
		"goal",
		"logic_fragment",
		"comment",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			return nil, fmt.Errorf("lean4 hypotheses csv is missing required column %q", column)
		}
	}
	if _, ok := columns["case_id"]; !ok {
		if _, ok := columns["theorem_id"]; !ok {
			return nil, fmt.Errorf("lean4 hypotheses csv must contain either %q or %q", "case_id", "theorem_id")
		}
	}

	cases := make([]HypothesisCase, 0, len(records)-1)
	for rowIndex, record := range records[1:] {
		interesting, err := strconv.ParseBool(valueAt(record, columns, "interesting"))
		if err != nil {
			return nil, fmt.Errorf("parse interesting at row %d: %w", rowIndex+2, err)
		}
		minimal, err := strconv.ParseBool(valueAt(record, columns, "minimal"))
		if err != nil {
			return nil, fmt.Errorf("parse minimal at row %d: %w", rowIndex+2, err)
		}
		var assumptions []string
		if err := unmarshalJSONStringArray(valueAt(record, columns, "assumptions_json"), &assumptions); err != nil {
			return nil, fmt.Errorf("decode assumptions_json at row %d: %w", rowIndex+2, err)
		}
		theoremID := firstNonEmpty(valueAt(record, columns, "theorem_id"), valueAt(record, columns, "case_id"))
		caseID := firstNonEmpty(valueAt(record, columns, "case_id"), theoremID)
		cases = append(cases, HypothesisCase{
			TheoremPackID:      firstNonEmpty(valueAt(record, columns, "theorem_pack_id"), deriveTheoremPackIDFromPath(path)),
			TheoremID:          theoremID,
			CaseID:             caseID,
			GenerationMode:     valueAt(record, columns, "generation_mode"),
			Category:           valueAt(record, columns, "category"),
			DerivabilityStatus: valueAt(record, columns, "derivability_status"),
			Interesting:        interesting,
			Minimal:            minimal,
			Difficulty:         valueAt(record, columns, "difficulty"),
			Assumptions:        assumptions,
			Goal:               valueAt(record, columns, "goal"),
			LogicFragment:      valueAt(record, columns, "logic_fragment"),
			LeanStatement:      valueAt(record, columns, "lean_statement"),
			Comment:            valueAt(record, columns, "comment"),
		})
	}
	return cases, nil
}

func ExportBenchmarkCasesCSV(sourcePath, outputPath string, config BenchmarkCaseExportConfig) (int, error) {
	hypotheses, err := LoadHypothesisCasesCSV(sourcePath)
	if err != nil {
		return 0, err
	}
	exported, _, err := ExportBenchmarkCases(outputPath, hypotheses, config)
	if err != nil {
		return 0, err
	}
	return exported, nil
}

func ExportBenchmarkCases(outputPath string, hypotheses []HypothesisCase, config BenchmarkCaseExportConfig) (int, string, error) {
	filtered := make([]BenchmarkCase, 0, len(hypotheses))
	for _, hypothesis := range hypotheses {
		if strings.TrimSpace(config.ModeFilter) != "" && config.ModeFilter != "*" && hypothesis.GenerationMode != config.ModeFilter {
			continue
		}
		if config.InterestingOnly && !hypothesis.Interesting {
			continue
		}
		if config.MinimalOnly && !hypothesis.Minimal {
			continue
		}
		filtered = append(filtered, BenchmarkCase{
			TheoremPackID: strings.TrimSpace(firstNonEmpty(hypothesis.TheoremPackID, deriveTheoremPackIDFromPath(outputPath))),
			TheoremID:     strings.TrimSpace(firstNonEmpty(hypothesis.TheoremID, hypothesis.CaseID)),
			CaseID:        strings.TrimSpace(hypothesis.CaseID),
			Category:      strings.TrimSpace(hypothesis.Category),
			Label:         benchmarkLabelFromDerivability(hypothesis.DerivabilityStatus),
			Difficulty:    strings.TrimSpace(hypothesis.Difficulty),
			Assumptions:   slices.Clone(hypothesis.Assumptions),
			Goal:          strings.TrimSpace(hypothesis.Goal),
			LogicFragment: strings.TrimSpace(hypothesis.LogicFragment),
			Comment:       benchmarkComment(hypothesis),
		})
		if config.Limit > 0 && len(filtered) >= config.Limit {
			break
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return 0, "", fmt.Errorf("create benchmark cases dir: %w", err)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, "", fmt.Errorf("create benchmark cases csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"theorem_pack_id",
		"theorem_id",
		"case_id",
		"category",
		"label",
		"difficulty",
		"assumptions_json",
		"goal",
		"logic_fragment",
		"comment",
	}); err != nil {
		return 0, "", fmt.Errorf("write benchmark cases header: %w", err)
	}
	for _, benchmarkCase := range filtered {
		assumptionsJSON, err := marshalJSONNoEscape(benchmarkCase.Assumptions)
		if err != nil {
			return 0, "", fmt.Errorf("marshal benchmark assumptions for %s: %w", benchmarkCase.CaseID, err)
		}
		if err := writer.Write([]string{
			benchmarkCase.TheoremPackID,
			benchmarkCase.TheoremID,
			benchmarkCase.CaseID,
			benchmarkCase.Category,
			benchmarkCase.Label,
			benchmarkCase.Difficulty,
			string(assumptionsJSON),
			benchmarkCase.Goal,
			benchmarkCase.LogicFragment,
			benchmarkCase.Comment,
		}); err != nil {
			return 0, "", fmt.Errorf("write benchmark case row %s: %w", benchmarkCase.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return 0, "", fmt.Errorf("flush benchmark cases csv: %w", err)
	}
	historicalPath := ""
	if strings.EqualFold(filepath.Base(filepath.Dir(outputPath)), DefaultTheoremsDir) {
		packID := deriveTheoremPackIDFromCases(filtered, outputPath)
		if err := writeHistoricalTheoremSnapshot(filepath.Dir(outputPath), packID, outputPath); err != nil {
			return 0, "", err
		}
		historicalPath = historicalTheoremSnapshotPath(filepath.Dir(outputPath), packID)
	}
	return len(filtered), historicalPath, nil
}

func benchmarkLabelFromDerivability(status string) string {
	switch strings.TrimSpace(status) {
	case DerivabilityStatusDerivable:
		return "entailed"
	default:
		return "not_entailed"
	}
}

func benchmarkComment(hypothesis HypothesisCase) string {
	base := strings.TrimSpace(hypothesis.Comment)
	suffix := fmt.Sprintf("lean4_source_mode=%s derivability_status=%s interesting=%t minimal=%t", hypothesis.GenerationMode, hypothesis.DerivabilityStatus, hypothesis.Interesting, hypothesis.Minimal)
	if base == "" {
		return suffix
	}
	return base + " | " + suffix
}

func unmarshalJSONStringArray(raw string, target *[]string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		*target = nil
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

func deriveTheoremPackIDFromCases(cases []BenchmarkCase, fallbackPath string) string {
	for _, tc := range cases {
		if value := strings.TrimSpace(tc.TheoremPackID); value != "" {
			return value
		}
	}
	return deriveTheoremPackIDFromPath(fallbackPath)
}

func deriveTheoremPackIDFromPath(path string) string {
	base := strings.TrimSpace(path)
	base = strings.TrimSuffix(filepath.Base(base), filepath.Ext(base))
	if base == "" {
		return "unknown_pack"
	}
	if strings.HasPrefix(strings.ToLower(base), "theorems_") {
		base = strings.TrimPrefix(strings.ToLower(base), "theorems_")
	}
	return sanitizeTheoremPackID(base)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
