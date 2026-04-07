package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/artifactkey"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

func TestRunArtifactsCompactCommandMigratesBenchmarkAndGeneratedArtifacts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("COLLABSPHERE_WORKSPACE_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "platform-tooling"), 0o755); err != nil {
		t.Fatalf("MkdirAll(platform-tooling) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "research"), 0o755); err != nil {
		t.Fatalf("MkdirAll(research) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	writeTestFile(t, filepath.Join(root, "research", "config", "default.yaml"), []byte(`schema_version: researchctl.config/v1
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
    input_file: theorems/pilot.csv
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
  meta_dir: research/result_research_report_v1/summary/meta
  cases_dir: research/result_research_report_v1/summary/cases
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`))

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	outerRunID := "phase1_20260406T100000+0300"
	jobKey := "local-compatible-gpt-oss-20b-r01"
	workerRunID := outerRunID + "_" + jobKey
	oldResultPath := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out", "result", "result_"+workerRunID+".csv")
	oldRawBase := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out", "raw")
	oldRawActual := filepath.Join(oldRawBase, workerRunID)
	oldChainPath := filepath.Join(filepath.Dir(oldResultPath), "chain_result_"+workerRunID+".csv")
	oldImportCatalogPath := filepath.Join(filepath.Dir(oldResultPath), "imported_certificate_catalog_"+workerRunID+".csv")
	oldImportedDir := filepath.Join(filepath.Dir(oldResultPath), "imported_certificates", workerRunID)
	oldImportedObject := filepath.Join(oldImportedDir, "CI01-S1.json")
	summaryPath := filepath.Join(root, "research", "artifacts", "result_research", "waves", outerRunID+".csv")
	manifestPath := filepath.Join(root, "research", "artifacts", "result_research", "manifests", outerRunID+".json")
	oldGenericReportPath := filepath.Join(root, "research", "result_research_report", "direct", "phase1_phase1_20260406T100000+0300.md")
	dedicatedRunID := "bridge-import-final-research-phase1-20260404"
	oldDedicatedReportPath := filepath.Join(root, "research", "result_research_report", "bridge-import-final-research", "bridge-import-final-research_"+dedicatedRunID+".md")
	dedicatedManifestPath := filepath.Join(root, "research", "artifacts", "result_research", "manifests", "bridge-import-final-research_"+dedicatedRunID+".json")

	writeTestFile(t, oldResultPath, []byte(strings.Join([]string{
		"run_id,case_id,certificate_object_file",
		workerRunID + ",CI01-S1," + filepath.ToSlash(oldImportedObject),
	}, "\n")))
	writeTestFile(t, oldChainPath, []byte("run_id,chain_id\n"+workerRunID+",CI01\n"))
	writeTestFile(t, oldImportCatalogPath, []byte(strings.Join([]string{
		"run_id,case_id,certificate_object_file",
		workerRunID + ",CI01-S1," + filepath.ToSlash(oldImportedObject),
	}, "\n")))
	writeTestFile(t, oldImportedObject, []byte(`{"certificate_object_file":"`+filepath.ToSlash(oldImportedObject)+`"}`))
	writeTestFile(t, filepath.Join(oldRawActual, "CI01-S1.txt"), []byte("raw output"))
	writeTestFile(t, oldGenericReportPath, []byte("# phase1 report\n"))
	writeTestFile(t, oldDedicatedReportPath, []byte("# bridge report\n"))
	writeTestFile(t, summaryPath, []byte(strings.Join([]string{
		"run_id,result_file,chain_result_file,import_catalog_file,raw_dir",
		workerRunID + "," + filepath.ToSlash(oldResultPath) + "," + filepath.ToSlash(oldChainPath) + "," + filepath.ToSlash(oldImportCatalogPath) + "," + filepath.ToSlash(oldRawActual),
	}, "\n")))

	writeJSONFile(t, manifestPath, map[string]any{
		"schema_version": "researchctl.manifest/v1",
		"phase":          "phase1",
		"run_id":         outerRunID,
		"report_path":    filepath.ToSlash(oldGenericReportPath),
		"jobs": []map[string]any{
			{
				"key":          jobKey,
				"run_id":       workerRunID,
				"results_path": filepath.ToSlash(oldResultPath),
				"raw_dir":      filepath.ToSlash(oldRawBase),
			},
		},
	})
	writeJSONFile(t, dedicatedManifestPath, map[string]any{
		"schema_version": "researchctl.manifest/v1",
		"phase":          "bridge-import-final-research",
		"run_id":         dedicatedRunID,
		"report_path":    filepath.ToSlash(oldDedicatedReportPath),
	})

	longPackOutput := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-v2-held-out", "cases", "phase1-expanded-100-very-long-benchmark-pack.csv")
	writeTestFile(t, longPackOutput, []byte("case_id,goal\n"))
	packManifestPath := filepath.Join(root, "research", "artifacts", "result_research", "manifests", "generate-benchmark-phase1_20260406T100500+0300.json")
	writeJSONFile(t, packManifestPath, map[string]any{
		"schema_version": "researchctl.generate-benchmark/v1",
		"run_id":         "20260406T100500+0300",
		"output_path":    filepath.ToSlash(longPackOutput),
	})

	longExportOutput := filepath.Join(root, "research", "artifacts", "hilbert-ai-verification-benchmark-nd-v1", "theorems", "lean-benchmark-export-very-long.csv")
	writeTestFile(t, longExportOutput, []byte("theorem_id,goal\n"))

	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:        outerRunID,
		Command:      "phase1 run",
		Phase:        "phase1",
		Status:       "completed",
		ManifestPath: manifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{"report_path": filepath.ToSlash(oldGenericReportPath)}),
	}); err != nil {
		t.Fatalf("UpsertRun() error = %v", err)
	}
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:        dedicatedRunID,
		Command:      "bridge-import-final-research run",
		Phase:        "bridge-import-final-research",
		Status:       "completed",
		ManifestPath: dedicatedManifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{"report_path": filepath.ToSlash(oldDedicatedReportPath)}),
	}); err != nil {
		t.Fatalf("UpsertRun(dedicated) error = %v", err)
	}
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:   "20260406T100500+0300",
		Command: "generate benchmark cases",
		Phase:   "generate-benchmark",
		Status:  "completed",
	}); err != nil {
		t.Fatalf("UpsertRun(generate) error = %v", err)
	}
	if err := store.UpsertRun(context.Background(), state.Run{
		RunID:   "20260406T101000+0300",
		Command: "export lean-to-benchmark",
		Phase:   "lean-export",
		Status:  "completed",
	}); err != nil {
		t.Fatalf("UpsertRun(export) error = %v", err)
	}
	if err := store.UpsertJob(context.Background(), state.Job{
		RunID:        outerRunID,
		JobKey:       jobKey,
		Phase:        "phase1",
		Family:       "local-compatible",
		Model:        "gpt-oss:20b",
		Selector:     "cases.csv",
		Status:       "completed",
		ResultPath:   oldResultPath,
		SummaryPath:  summaryPath,
		RawDir:       oldRawBase,
		MetadataJSON: state.MetadataJSON(map[string]any{"worker_run_id": workerRunID}),
	}); err != nil {
		t.Fatalf("UpsertJob() error = %v", err)
	}
	if err := store.RecordArtifact(context.Background(), state.Artifact{RunID: outerRunID, JobKey: jobKey, Role: "benchmark-result", AbsolutePath: oldResultPath}); err != nil {
		t.Fatalf("RecordArtifact(result) error = %v", err)
	}
	if err := store.RecordArtifact(context.Background(), state.Artifact{RunID: outerRunID, JobKey: jobKey, Role: "benchmark-raw-dir", AbsolutePath: oldRawBase}); err != nil {
		t.Fatalf("RecordArtifact(raw) error = %v", err)
	}
	if err := store.RecordArtifact(context.Background(), state.Artifact{RunID: outerRunID, Role: "benchmark-report", AbsolutePath: oldGenericReportPath}); err != nil {
		t.Fatalf("RecordArtifact(report) error = %v", err)
	}
	if err := store.RecordArtifact(context.Background(), state.Artifact{RunID: dedicatedRunID, Role: "benchmark-report", AbsolutePath: oldDedicatedReportPath}); err != nil {
		t.Fatalf("RecordArtifact(dedicated report) error = %v", err)
	}
	if err := store.RecordGeneratedPack(context.Background(), state.GeneratedPack{
		RunID:         "20260406T100500+0300",
		PackKey:       "phase1-expanded-100-very-long-benchmark-pack",
		ProjectFolder: "hilbert-ai-verification-benchmark-v2-held-out",
		SourceKind:    "benchmark/phase1",
		OutputPath:    longPackOutput,
		ManifestPath:  packManifestPath,
	}); err != nil {
		t.Fatalf("RecordGeneratedPack() error = %v", err)
	}
	if err := store.RecordExport(context.Background(), state.Export{
		RunID:                  "20260406T101000+0300",
		ExportKey:              "20260406T101000+0300",
		ProjectFolder:          "hilbert-ai-verification-lean4-worker-v1",
		BenchmarkProjectFolder: "hilbert-ai-verification-benchmark-nd-v1",
		SourcePackID:           "pack-001",
		OutputPath:             longExportOutput,
	}); err != nil {
		t.Fatalf("RecordExport() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}

	var dryRunOut strings.Builder
	if err := runArtifactsCompactCommand([]string{"--config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml"))}, &dryRunOut, &dryRunOut); err != nil {
		t.Fatalf("runArtifactsCompactCommand(dry-run) error = %v", err)
	}
	if !strings.Contains(dryRunOut.String(), "renamed_paths:") {
		t.Fatalf("dry-run output missing renamed_paths:\n%s", dryRunOut.String())
	}

	var applyOut strings.Builder
	if err := runArtifactsCompactCommand([]string{"--config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "--apply"}, &applyOut, &applyOut); err != nil {
		t.Fatalf("runArtifactsCompactCommand(apply) error = %v", err)
	}

	expectedBenchmarkArtifactKey := artifactkey.BenchmarkJob("phase1", outerRunID, jobKey)
	newResultPath := filepath.Join(filepath.Dir(oldResultPath), "result_"+expectedBenchmarkArtifactKey+".csv")
	newRawDir := filepath.Join(oldRawBase, expectedBenchmarkArtifactKey)
	newImportCatalogPath := filepath.Join(filepath.Dir(oldImportCatalogPath), "imported_certificate_catalog_"+expectedBenchmarkArtifactKey+".csv")
	newImportedObject := filepath.Join(filepath.Dir(oldImportedDir), expectedBenchmarkArtifactKey, "CI01-S1.json")

	if _, err := os.Stat(newResultPath); err != nil {
		t.Fatalf("Stat(newResultPath) error = %v", err)
	}
	if _, err := os.Stat(newRawDir); err != nil {
		t.Fatalf("Stat(newRawDir) error = %v", err)
	}
	if _, err := os.Stat(newImportCatalogPath); err != nil {
		t.Fatalf("Stat(newImportCatalogPath) error = %v", err)
	}
	if _, err := os.Stat(newImportedObject); err != nil {
		t.Fatalf("Stat(newImportedObject) error = %v", err)
	}
	if _, err := os.Stat(oldResultPath); !os.IsNotExist(err) {
		t.Fatalf("old result path still exists: %v", err)
	}
	newGenericReportPath := filepath.Join(root, "research", "result_research_report", "direct", "phase1_20260406T100000+0300.md")
	newDedicatedReportPath := filepath.Join(root, "research", "result_research_report", "bridge-import-final-research", dedicatedRunID+".md")
	if _, err := os.Stat(newGenericReportPath); err != nil {
		t.Fatalf("Stat(newGenericReportPath) error = %v", err)
	}
	if _, err := os.Stat(newDedicatedReportPath); err != nil {
		t.Fatalf("Stat(newDedicatedReportPath) error = %v", err)
	}
	if _, err := os.Stat(oldGenericReportPath); !os.IsNotExist(err) {
		t.Fatalf("old generic report path still exists: %v", err)
	}
	if _, err := os.Stat(oldDedicatedReportPath); !os.IsNotExist(err) {
		t.Fatalf("old dedicated report path still exists: %v", err)
	}

	summaryText := string(readTestFile(t, summaryPath))
	if !strings.Contains(summaryText, filepath.ToSlash(newResultPath)) || !strings.Contains(summaryText, filepath.ToSlash(newRawDir)) {
		t.Fatalf("summary file not rewritten:\n%s", summaryText)
	}
	resultText := string(readTestFile(t, newResultPath))
	if !strings.Contains(resultText, filepath.ToSlash(newImportedObject)) {
		t.Fatalf("result file not rewritten:\n%s", resultText)
	}
	manifestText := string(readTestFile(t, manifestPath))
	if !strings.Contains(manifestText, expectedBenchmarkArtifactKey) || !strings.Contains(manifestText, filepath.ToSlash(newResultPath)) || !strings.Contains(manifestText, filepath.ToSlash(newGenericReportPath)) {
		t.Fatalf("benchmark manifest not rewritten:\n%s", manifestText)
	}
	dedicatedManifestText := string(readTestFile(t, dedicatedManifestPath))
	if !strings.Contains(dedicatedManifestText, filepath.ToSlash(newDedicatedReportPath)) {
		t.Fatalf("dedicated benchmark manifest not rewritten:\n%s", dedicatedManifestText)
	}

	expectedPackArtifactKey := artifactkey.GeneratedPack("20260406T100500+0300", "phase1-expanded-100-very-long-benchmark-pack")
	newPackOutput := filepath.Join(filepath.Dir(longPackOutput), "pack_"+expectedPackArtifactKey+".csv")
	if _, err := os.Stat(newPackOutput); err != nil {
		t.Fatalf("Stat(newPackOutput) error = %v", err)
	}
	packManifestText := string(readTestFile(t, packManifestPath))
	if !strings.Contains(packManifestText, filepath.ToSlash(newPackOutput)) {
		t.Fatalf("pack manifest not rewritten:\n%s", packManifestText)
	}

	expectedExportArtifactKey := artifactkey.Export("20260406T101000+0300", "20260406T101000+0300")
	newExportOutput := filepath.Join(filepath.Dir(longExportOutput), "export_"+expectedExportArtifactKey+".csv")
	if _, err := os.Stat(newExportOutput); err != nil {
		t.Fatalf("Stat(newExportOutput) error = %v", err)
	}

	store, err = state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		t.Fatalf("state.Open(after) error = %v", err)
	}
	defer store.Close()
	jobs, err := store.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if got := jobs[0].RawDir; filepath.ToSlash(got) != filepath.ToSlash(newRawDir) {
		t.Fatalf("jobs[0].RawDir = %q, want %q", got, newRawDir)
	}
	packs, err := store.ListGeneratedPacks(context.Background())
	if err != nil {
		t.Fatalf("ListGeneratedPacks() error = %v", err)
	}
	if filepath.ToSlash(packs[0].OutputPath) != filepath.ToSlash(newPackOutput) {
		t.Fatalf("packs[0].OutputPath = %q, want %q", packs[0].OutputPath, newPackOutput)
	}
	exports, err := store.ListExports(context.Background())
	if err != nil {
		t.Fatalf("ListExports() error = %v", err)
	}
	if filepath.ToSlash(exports[0].OutputPath) != filepath.ToSlash(newExportOutput) {
		t.Fatalf("exports[0].OutputPath = %q, want %q", exports[0].OutputPath, newExportOutput)
	}
	runs, err := store.ListRuns(context.Background())
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	runMetadata := map[string]string{}
	for _, run := range runs {
		runMetadata[run.RunID] = run.MetadataJSON
	}
	if !strings.Contains(runMetadata[outerRunID], filepath.ToSlash(newGenericReportPath)) {
		t.Fatalf("outer run metadata not rewritten: %s", runMetadata[outerRunID])
	}
	if !strings.Contains(runMetadata[dedicatedRunID], filepath.ToSlash(newDedicatedReportPath)) {
		t.Fatalf("dedicated run metadata not rewritten: %s", runMetadata[dedicatedRunID])
	}
	if !strings.Contains(applyOut.String(), "artifact compaction applied") {
		t.Fatalf("apply output missing success marker:\n%s", applyOut.String())
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	writeTestFile(t, path, raw)
}

func writeTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return raw
}
