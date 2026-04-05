package lean4worker

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateHypothesisArtifacts(t *testing.T) {
	root := t.TempDir()
	theoremsPath, configPath, payloadPath, err := GenerateHypothesisArtifacts(root, DefaultHypothesisGenerationConfig())
	if err != nil {
		t.Fatalf("GenerateHypothesisArtifacts() error = %v", err)
	}
	if _, err := os.Stat(theoremsPath); err != nil {
		t.Fatalf("Stat(theoremsPath) error = %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Stat(configPath) error = %v", err)
	}
	if _, err := os.Stat(payloadPath); err != nil {
		t.Fatalf("Stat(payloadPath) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, LegacyCasesDir, legacyHypothesesFileName)); err != nil {
		t.Fatalf("Stat(legacy hypotheses alias) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, LegacyCasesDir, generationConfigFileName)); err != nil {
		t.Fatalf("Stat(legacy generation config alias) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, LegacyCasesDir, "payloads", defaultPayloadTemplateName)); err != nil {
		t.Fatalf("Stat(legacy payload alias) error = %v", err)
	}

	file, err := os.Open(theoremsPath)
	if err != nil {
		t.Fatalf("Open(theoremsPath) error = %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll(theoremsPath) error = %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("len(records) = %d, want at least header + 1 row", len(records))
	}
	if got := records[0][0]; got != "theorem_pack_id" {
		t.Fatalf("header[0] = %q, want theorem_pack_id", got)
	}
	if got := records[0][1]; got != "theorem_id" {
		t.Fatalf("header[1] = %q, want theorem_id", got)
	}
	if got := records[0][2]; got != "case_id" {
		t.Fatalf("header[2] = %q, want case_id", got)
	}
	if got := records[0][3]; got != "generation_mode" {
		t.Fatalf("header[3] = %q, want generation_mode", got)
	}
	if got := filepath.Base(theoremsPath); got != canonicalTheoremsFileName {
		t.Fatalf("theorem file = %q, want %q", got, canonicalTheoremsFileName)
	}
	if _, err := os.Stat(filepath.Join(root, DefaultTheoremsDir, HistoricalTheoremsDir)); err != nil {
		t.Fatalf("Stat(historical theorem dir) error = %v", err)
	}
	if got := filepath.Base(payloadPath); got != "implicational-default.json" {
		t.Fatalf("payload file = %q, want implicational-default.json", got)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(configPath) error = %v", err)
	}
	var cfg HypothesisGenerationConfig
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		t.Fatalf("Unmarshal(configPath) error = %v", err)
	}
	if cfg.Mode != HypothesisGenerationModeEnumerator {
		t.Fatalf("cfg.Mode = %q, want %q", cfg.Mode, HypothesisGenerationModeEnumerator)
	}
}

func TestGenerateHypothesisCases_CuratedMode(t *testing.T) {
	cases, cfg, err := GenerateHypothesisCases(HypothesisGenerationConfig{
		Mode:    HypothesisGenerationModeCurated,
		Limit:   8,
		Filters: []string{"non_duplicate"},
	})
	if err != nil {
		t.Fatalf("GenerateHypothesisCases(curated) error = %v", err)
	}
	if cfg.Mode != HypothesisGenerationModeCurated {
		t.Fatalf("cfg.Mode = %q, want %q", cfg.Mode, HypothesisGenerationModeCurated)
	}
	if len(cases) == 0 {
		t.Fatalf("len(cases) = 0, want > 0")
	}
	if cases[0].GenerationMode != HypothesisGenerationModeCurated {
		t.Fatalf("cases[0].GenerationMode = %q, want %q", cases[0].GenerationMode, HypothesisGenerationModeCurated)
	}
}

func TestGenerateHypothesisCases_ModelProposedMode(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "model-proposed.csv")
	if err := writeHypothesisSeedCSV(sourcePath, DefaultModelProposedSeeds()); err != nil {
		t.Fatalf("writeHypothesisSeedCSV() error = %v", err)
	}

	cases, cfg, err := GenerateHypothesisCases(HypothesisGenerationConfig{
		Mode:       HypothesisGenerationModeProposed,
		SourceFile: sourcePath,
		Limit:      8,
		Filters:    []string{"non_duplicate"},
	})
	if err != nil {
		t.Fatalf("GenerateHypothesisCases(model_proposed) error = %v", err)
	}
	if cfg.Mode != HypothesisGenerationModeProposed {
		t.Fatalf("cfg.Mode = %q, want %q", cfg.Mode, HypothesisGenerationModeProposed)
	}
	if len(cases) == 0 {
		t.Fatalf("len(cases) = 0, want > 0")
	}
	if cases[0].GenerationMode != HypothesisGenerationModeProposed {
		t.Fatalf("cases[0].GenerationMode = %q, want %q", cases[0].GenerationMode, HypothesisGenerationModeProposed)
	}
}
