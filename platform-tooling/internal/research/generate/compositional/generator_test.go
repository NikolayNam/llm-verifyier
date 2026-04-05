package compositional

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestGenerateBridgeOnlyPhase2WritesBalancedPackAndLocalConfig(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-bridge-only-phase2.csv")
	localConfigPath := filepath.Join(root, "research", "config", "local.generated.bridge-only.phase2.yaml")

	result, err := Generate(GenerationOptions{
		Surface:            SurfaceBridgeOnlyAuthoring,
		Mode:               ModePhase2,
		OutputPath:         outputPath,
		BenchmarkInputFile: "cases/generated-bridge-only-phase2.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    localConfigPath,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.CaseCount != 150 {
		t.Fatalf("result.CaseCount = %d, want 150", result.CaseCount)
	}
	if result.ChainCount != 75 {
		t.Fatalf("result.ChainCount = %d, want 75", result.ChainCount)
	}
	if result.PromptVersion != "hilbert-ai-verification-bridge-only-v1.0" {
		t.Fatalf("result.PromptVersion = %q, want bridge-only prompt", result.PromptVersion)
	}

	records := readCSVRecords(t, outputPath)
	if len(records) != 151 {
		t.Fatalf("len(records) = %d, want 151 including header", len(records))
	}
	header := headerIndex(records[0])
	stageCounts := map[string]int{}
	difficultyCounts := map[string]int{}
	for _, record := range records[1:] {
		stageCounts[cell(record, header, "stage_id")]++
		difficultyCounts[cell(record, header, "difficulty")]++
		if cell(record, header, "category") != "bridge_only_authoring" {
			t.Fatalf("category = %q, want bridge_only_authoring", cell(record, header, "category"))
		}
	}
	if stageCounts["b1"] != 75 || stageCounts["b2"] != 75 {
		t.Fatalf("stage counts = %#v, want b1=75 and b2=75", stageCounts)
	}
	if difficultyCounts["easy"] != 50 || difficultyCounts["medium"] != 50 || difficultyCounts["hard"] != 50 {
		t.Fatalf("difficulty counts = %#v, want 50/50/50", difficultyCounts)
	}

	rawLocalConfig, err := os.ReadFile(localConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(local config) error = %v", err)
	}
	text := string(rawLocalConfig)
	if !strings.Contains(text, "bridge_only_authoring:") {
		t.Fatalf("local config missing bridge_only_authoring benchmark override: %s", text)
	}
	if !strings.Contains(text, "cases/generated-bridge-only-phase2.csv") {
		t.Fatalf("local config missing generated input_file selector: %s", text)
	}
}

func TestGenerateGoldFinalPhase2KeepsBalancedEntailedAndNegativeSlices(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-gold-final-phase2.csv")

	result, err := Generate(GenerationOptions{
		Surface:    SurfaceGoldFinalCompositionOnly,
		Mode:       ModePhase2,
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.CaseCount != 150 {
		t.Fatalf("result.CaseCount = %d, want 150", result.CaseCount)
	}

	records := readCSVRecords(t, outputPath)
	header := headerIndex(records[0])
	labelCounts := map[string]int{}
	difficultyLabelCounts := map[string]int{}
	for _, record := range records[1:] {
		label := cell(record, header, "label")
		difficulty := cell(record, header, "difficulty")
		labelCounts[label]++
		difficultyLabelCounts[difficulty+"::"+label]++
	}
	if labelCounts["entailed"] != 75 || labelCounts["not_entailed"] != 75 {
		t.Fatalf("label counts = %#v, want entailed=75 and not_entailed=75", labelCounts)
	}
	for _, key := range []string{
		"easy::entailed",
		"easy::not_entailed",
		"medium::entailed",
		"medium::not_entailed",
		"hard::entailed",
		"hard::not_entailed",
	} {
		if difficultyLabelCounts[key] != 25 {
			t.Fatalf("difficulty-label counts[%q] = %d, want 25", key, difficultyLabelCounts[key])
		}
	}
}

func TestGenerateBridgeImportFinalPhase1CaseCountKeepsCanonicalBaseline(t *testing.T) {
	rows, err := generateBridgeImportFinalRows(BridgeImportFinalPhase1CaseCount)
	if err != nil {
		t.Fatalf("generateBridgeImportFinalRows() error = %v", err)
	}
	if len(rows) != BridgeImportFinalPhase1CaseCount {
		t.Fatalf("len(rows) = %d, want %d", len(rows), BridgeImportFinalPhase1CaseCount)
	}
	if rows[0].CaseID != "BIF01-B1" {
		t.Fatalf("rows[0].CaseID = %q, want canonical BIF01-B1", rows[0].CaseID)
	}
	if rows[len(rows)-1].CaseID != "BIF04-NEG" {
		t.Fatalf("rows[last].CaseID = %q, want canonical BIF04-NEG", rows[len(rows)-1].CaseID)
	}
}

func TestGenerateBridgeImportFinalExpandedPackFromCaseCount(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-bridge-final-100.csv")

	result, err := Generate(GenerationOptions{
		Surface:    SurfaceBridgeImportFinalResearch,
		Mode:       ModePhase1,
		CaseCount:  100,
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.CaseCount != 100 {
		t.Fatalf("result.CaseCount = %d, want 100", result.CaseCount)
	}
	if result.ChainCount != 20 {
		t.Fatalf("result.ChainCount = %d, want 20", result.ChainCount)
	}

	records := readCSVRecords(t, outputPath)
	if len(records) != 101 {
		t.Fatalf("len(records) = %d, want 101 including header", len(records))
	}
	header := headerIndex(records[0])
	stageCounts := map[string]int{}
	difficultyCounts := map[string]int{}
	for _, record := range records[1:] {
		stageCounts[cell(record, header, "stage_id")]++
		difficultyCounts[cell(record, header, "difficulty")]++
		if !strings.Contains(cell(record, header, "chain_protocol"), "expanded") {
			t.Fatalf("chain_protocol = %q, want expanded protocol marker", cell(record, header, "chain_protocol"))
		}
	}
	for _, stageID := range []string{"b1", "b2", "fg", "fm", "neg"} {
		if stageCounts[stageID] != 20 {
			t.Fatalf("stageCounts[%q] = %d, want 20", stageID, stageCounts[stageID])
		}
	}
	if difficultyCounts["easy"] != 40 || difficultyCounts["medium"] != 30 || difficultyCounts["hard"] != 30 {
		t.Fatalf("difficulty counts = %#v, want easy=40, medium=30, hard=30", difficultyCounts)
	}
}

func TestGenerateBridgeImportFinalRejectsInvalidCaseCount(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-bridge-final-invalid.csv")

	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceBridgeImportFinalResearch,
		Mode:       ModePhase1,
		CaseCount:  99,
		OutputPath: outputPath,
	}); err == nil {
		t.Fatalf("Generate() error = nil, want invalid case-count refusal")
	}
}

func TestGenerateRejectsCaseCountForFixedSplitSurface(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-bridge-only-invalid.csv")

	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceBridgeOnlyAuthoring,
		Mode:       ModePhase1,
		CaseCount:  10,
		OutputPath: outputPath,
	}); err == nil {
		t.Fatalf("Generate() error = nil, want fixed-surface case-count refusal")
	}
}

func TestGenerateRefusesOverwriteWithoutForce(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-bridge-final.csv")

	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceBridgeImportFinalResearch,
		Mode:       ModePhase1,
		OutputPath: outputPath,
	}); err != nil {
		t.Fatalf("first Generate() error = %v", err)
	}
	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceBridgeImportFinalResearch,
		Mode:       ModePhase1,
		OutputPath: outputPath,
	}); err == nil {
		t.Fatalf("second Generate() error = nil, want overwrite refusal")
	}
}

func TestGeneratedLocalConfigLoadsThroughResearchConfig(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}
	defaultConfig := `schema_version: researchctl.config/v1
paths:
  artifact_root: research/artifacts
  report_root: research/result_research_report_v1
  temp_root: .tmp/research
  manifest_root: research/artifacts/result_research/manifests
state:
  db_path: research/artifacts/result_research/research_db/research_db.sqlite
transports:
  local-compatible:
    provider: compatible
    base_url: http://localhost:11434
families:
  local-compatible:
    transport: local-compatible
    models: [gpt-oss:20b]
    experiment_name: formal-mode
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
generate:
  lean4:
    project_folder: hilbert-ai-verification-lean4-worker-v1
    benchmark_project_folder: hilbert-ai-verification-benchmark-nd-v1
    workspace_root: .tmp/research/lean4
    lake_binary: lake
    timeout: 60s
    export_output_file: theorems/lean4-generated.csv
reports:
  benchmark_dir: research/result_research_report_v1/direct
  nd_dir: research/result_research_report_v1/nd
  research_dir: research/result_research_report_v1/pair
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(defaultConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	localConfigPath := filepath.Join(configDir, "local.generated.gold.yaml")
	outputPath := filepath.Join(root, "research", "artifacts", DefaultProjectFolder, "cases", "generated-gold-phase1.csv")
	if _, err := Generate(GenerationOptions{
		Surface:            SurfaceGoldFinalCompositionOnly,
		Mode:               ModePhase1,
		OutputPath:         outputPath,
		BenchmarkInputFile: "cases/generated-gold-phase1.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    localConfigPath,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "local.generated.gold.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Config.Benchmarks.GoldFinalCompositionOnly.InputFile != "cases/generated-gold-phase1.csv" {
		t.Fatalf("gold_final_composition_only input_file = %q, want generated selector", loaded.Config.Benchmarks.GoldFinalCompositionOnly.InputFile)
	}
	if loaded.Config.Benchmarks.GoldFinalCompositionOnly.PromptVersion != "hilbert-ai-verification-gold-final-only-v1.0" {
		t.Fatalf("gold_final_composition_only prompt_version = %q", loaded.Config.Benchmarks.GoldFinalCompositionOnly.PromptVersion)
	}
	if len(loaded.Config.Benchmarks.GoldFinalCompositionOnly.Families) != 1 {
		t.Fatalf("gold_final_composition_only families = %#v, want one generated family", loaded.Config.Benchmarks.GoldFinalCompositionOnly.Families)
	}
}

func readCSVRecords(t *testing.T, path string) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll(%s) error = %v", path, err)
	}
	return records
}

func headerIndex(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[name] = i
	}
	return index
}

func cell(record []string, header map[string]int, name string) string {
	if idx, ok := header[name]; ok && idx < len(record) {
		return record[idx]
	}
	return ""
}
