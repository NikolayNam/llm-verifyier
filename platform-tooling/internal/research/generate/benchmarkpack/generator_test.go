package benchmarkpack

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestGenerateDirectExpandedPackKeepsBalancedLabels(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultDirectProjectFolder, "cases", "phase1-expanded-100.csv")
	localConfigPath := filepath.Join(root, "research", "config", "local.phase1-expanded-100.yaml")

	result, err := Generate(GenerationOptions{
		Surface:            SurfaceDirect,
		CaseCount:          100,
		OutputPath:         outputPath,
		BenchmarkInputFile: "cases/phase1-expanded-100.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    localConfigPath,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.CaseCount != 100 {
		t.Fatalf("result.CaseCount = %d, want 100", result.CaseCount)
	}

	records := readCSVRecords(t, outputPath)
	if len(records) != 101 {
		t.Fatalf("len(records) = %d, want 101 including header", len(records))
	}
	header := headerIndex(records[0])
	labelCounts := map[string]int{}
	difficultyCounts := map[string]int{}
	seenSourceCaseIDs := map[string]int{}
	if _, ok := header["source_case_id"]; !ok {
		t.Fatalf("expanded direct header missing source_case_id: %#v", records[0])
	}
	for _, record := range records[1:] {
		labelCounts[cell(record, header, "label")]++
		difficultyCounts[cell(record, header, "difficulty")]++
		sourceCaseID := cell(record, header, "source_case_id")
		if sourceCaseID == "" {
			t.Fatalf("expanded direct row missing source_case_id: %#v", record)
		}
		seenSourceCaseIDs[sourceCaseID]++
	}
	if labelCounts["entailed"] != 50 || labelCounts["not_entailed"] != 50 {
		t.Fatalf("label counts = %#v, want entailed=50 and not_entailed=50", labelCounts)
	}
	if len(seenSourceCaseIDs) < 10 {
		t.Fatalf("source_case_id coverage = %d, want broad lineage coverage across direct templates", len(seenSourceCaseIDs))
	}
	for _, difficulty := range []string{"easy", "medium", "hard"} {
		if difficultyCounts[difficulty] == 0 {
			t.Fatalf("difficulty %q missing from expanded direct pack", difficulty)
		}
	}
}

func TestGenerateNDExpandedPackKeepsBalancedLabels(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "research", "artifacts", DefaultNDProjectFolder, "theorems", "pilot_shared_20260327-expanded-100.csv")
	localConfigPath := filepath.Join(root, "research", "config", "local.nd-expanded-100.yaml")

	result, err := Generate(GenerationOptions{
		Surface:            SurfaceND,
		CaseCount:          100,
		OutputPath:         outputPath,
		BenchmarkInputFile: "theorems/pilot_shared_20260327-expanded-100.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    localConfigPath,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.CaseCount != 100 {
		t.Fatalf("result.CaseCount = %d, want 100", result.CaseCount)
	}

	records := readCSVRecords(t, outputPath)
	if len(records) != 101 {
		t.Fatalf("len(records) = %d, want 101 including header", len(records))
	}
	header := headerIndex(records[0])
	labelCounts := map[string]int{}
	packIDs := map[string]int{}
	ids := map[string]struct{}{}
	for _, record := range records[1:] {
		labelCounts[cell(record, header, "label")]++
		packIDs[cell(record, header, "theorem_pack_id")]++
		caseID := cell(record, header, "case_id")
		if _, ok := ids[caseID]; ok {
			t.Fatalf("duplicate case_id %q in nd expanded pack", caseID)
		}
		ids[caseID] = struct{}{}
		if cell(record, header, "logic_fragment") != "implicational-prop-v1" {
			t.Fatalf("logic_fragment = %q, want implicational-prop-v1", cell(record, header, "logic_fragment"))
		}
	}
	if labelCounts["entailed"] != 50 || labelCounts["not_entailed"] != 50 {
		t.Fatalf("label counts = %#v, want entailed=50 and not_entailed=50", labelCounts)
	}
	if len(packIDs) != 1 {
		t.Fatalf("packIDs = %#v, want exactly one theorem_pack_id", packIDs)
	}
}

func TestGenerateExpandedLocalConfigsLoadThroughResearchConfig(t *testing.T) {
	root := t.TempDir()
	writeDefaultResearchConfig(t, root)

	directConfigPath := filepath.Join(root, "research", "config", "local.phase1-expanded-100.yaml")
	directOutputPath := filepath.Join(root, "research", "artifacts", DefaultDirectProjectFolder, "cases", "phase1-expanded-100.csv")
	if _, err := Generate(GenerationOptions{
		Surface:            SurfaceDirect,
		CaseCount:          100,
		OutputPath:         directOutputPath,
		BenchmarkInputFile: "cases/phase1-expanded-100.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    directConfigPath,
	}); err != nil {
		t.Fatalf("Generate(direct) error = %v", err)
	}
	loadedDirect, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "local.phase1-expanded-100.yaml"))
	if err != nil {
		t.Fatalf("Load(direct expanded config) error = %v", err)
	}
	if loadedDirect.Config.Benchmarks.Phase1.InputFile != "cases/phase1-expanded-100.csv" {
		t.Fatalf("phase1 input_file = %q, want generated selector", loadedDirect.Config.Benchmarks.Phase1.InputFile)
	}

	ndConfigPath := filepath.Join(root, "research", "config", "local.nd-expanded-100.yaml")
	ndOutputPath := filepath.Join(root, "research", "artifacts", DefaultNDProjectFolder, "theorems", "pilot_shared_20260327-expanded-100.csv")
	if _, err := Generate(GenerationOptions{
		Surface:            SurfaceND,
		CaseCount:          100,
		OutputPath:         ndOutputPath,
		BenchmarkInputFile: "theorems/pilot_shared_20260327-expanded-100.csv",
		EmitLocalConfig:    true,
		LocalConfigPath:    ndConfigPath,
	}); err != nil {
		t.Fatalf("Generate(nd) error = %v", err)
	}
	loadedND, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "local.nd-expanded-100.yaml"))
	if err != nil {
		t.Fatalf("Load(nd expanded config) error = %v", err)
	}
	if loadedND.Config.Benchmarks.ND.InputFile != "theorems/pilot_shared_20260327-expanded-100.csv" {
		t.Fatalf("nd input_file = %q, want generated selector", loadedND.Config.Benchmarks.ND.InputFile)
	}
}

func TestGenerateRejectsOddExpandedCaseCount(t *testing.T) {
	root := t.TempDir()
	directOutputPath := filepath.Join(root, "research", "artifacts", DefaultDirectProjectFolder, "cases", "phase1-expanded-99.csv")
	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceDirect,
		CaseCount:  99,
		OutputPath: directOutputPath,
	}); err == nil {
		t.Fatalf("Generate(direct odd case-count) error = nil, want refusal")
	}

	ndOutputPath := filepath.Join(root, "research", "artifacts", DefaultNDProjectFolder, "theorems", "pilot_shared_20260327-expanded-99.csv")
	if _, err := Generate(GenerationOptions{
		Surface:    SurfaceND,
		CaseCount:  99,
		OutputPath: ndOutputPath,
	}); err == nil {
		t.Fatalf("Generate(nd odd case-count) error = nil, want refusal")
	}
}

func writeDefaultResearchConfig(t *testing.T, root string) {
	t.Helper()
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
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
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
	idx, ok := header[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return record[idx]
}
