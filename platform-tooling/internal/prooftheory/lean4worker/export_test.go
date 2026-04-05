package lean4worker

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestExportBenchmarkCasesCSV(t *testing.T) {
	root := t.TempDir()
	sourcePath, _, _, err := GenerateHypothesisArtifacts(root, DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisArtifacts() error = %v", err)
	}

	outputPath := filepath.Join(root, "benchmark", "theorems", "lean4-generated.csv")
	exported, err := ExportBenchmarkCasesCSV(sourcePath, outputPath, BenchmarkCaseExportConfig{
		ModeFilter:      HypothesisGenerationModeEnumerator,
		InterestingOnly: true,
		MinimalOnly:     true,
		Limit:           5,
	})
	if err != nil {
		t.Fatalf("ExportBenchmarkCasesCSV() error = %v", err)
	}
	if exported != 5 {
		t.Fatalf("exported = %d, want 5", exported)
	}

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Open(outputPath) error = %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll(outputPath) error = %v", err)
	}
	if len(records) != 6 {
		t.Fatalf("len(records) = %d, want 6", len(records))
	}
	if got := records[0][0]; got != "theorem_pack_id" {
		t.Fatalf("header[0] = %q, want theorem_pack_id", got)
	}
	if got := records[0][1]; got != "theorem_id" {
		t.Fatalf("header[1] = %q, want theorem_id", got)
	}
	if got := records[0][4]; got != "label" {
		t.Fatalf("header[4] = %q, want label", got)
	}
	if got := records[1][4]; got != "entailed" && got != "not_entailed" {
		t.Fatalf("row[1][4] = %q, want entailed or not_entailed", got)
	}
	if _, err := os.Stat(filepath.Join(root, "benchmark", "theorems", HistoricalTheoremsDir)); err != nil {
		t.Fatalf("Stat(benchmark historical theorem dir) error = %v", err)
	}
}

func TestLoadHypothesisCasesCSV(t *testing.T) {
	root := t.TempDir()
	sourcePath, _, _, err := GenerateHypothesisArtifacts(root, DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisArtifacts() error = %v", err)
	}
	cases, err := LoadHypothesisCasesCSV(sourcePath)
	if err != nil {
		t.Fatalf("LoadHypothesisCasesCSV() error = %v", err)
	}
	if len(cases) == 0 {
		t.Fatalf("len(cases) = 0, want > 0")
	}
	if cases[0].GenerationMode != HypothesisGenerationModeEnumerator {
		t.Fatalf("cases[0].GenerationMode = %q, want %q", cases[0].GenerationMode, HypothesisGenerationModeEnumerator)
	}
}
