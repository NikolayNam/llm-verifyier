package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestBuildPhase1PlanCreatesStableJobsAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: formal-mode
    models: [gpt-oss:20b, gpt-oss:120b-cloud]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 2
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildPhase1Plan(loaded, BenchmarkPlanOptions{
		RunID:   "20260401T120000Z",
		Command: "researchctl phase1 run",
	})
	if err != nil {
		t.Fatalf("BuildPhase1Plan() error = %v", err)
	}
	if len(plan.Jobs) != 4 {
		t.Fatalf("len(plan.Jobs) = %d, want 4", len(plan.Jobs))
	}
	seen := map[string]struct{}{}
	for _, job := range plan.Jobs {
		if job.SpecHash == "" {
			t.Fatalf("job %q has empty SpecHash", job.Key)
		}
		if _, ok := seen[job.Key]; ok {
			t.Fatalf("duplicate job key %q", job.Key)
		}
		seen[job.Key] = struct{}{}
		if job.SummaryPath != plan.SummaryPath {
			t.Fatalf("job %q summary_path = %q, want %q", job.Key, job.SummaryPath, plan.SummaryPath)
		}
		if filepath.Base(job.ResultsPath) == "result_.csv" {
			t.Fatalf("job %q results_path = %q, want concrete run-id suffix", job.Key, job.ResultsPath)
		}
		if job.ExperimentName != "formal-mode" {
			t.Fatalf("job %q ExperimentName = %q, want formal-mode", job.Key, job.ExperimentName)
		}
		if job.ExperimentID == "" || !strings.Contains(job.ExperimentID, "formal-mode-temp0-seed42-topp1-r") {
			t.Fatalf("job %q ExperimentID = %q, want formal-mode sampling prefix", job.Key, job.ExperimentID)
		}
		if job.RequestedSamplingProfile != "temp0_seed42_topp1" {
			t.Fatalf("job %q RequestedSamplingProfile = %q, want temp0_seed42_topp1", job.Key, job.RequestedSamplingProfile)
		}
		if job.EffectiveSamplingProfile != "temp0_seed42_topp1" {
			t.Fatalf("job %q EffectiveSamplingProfile = %q, want temp0_seed42_topp1", job.Key, job.EffectiveSamplingProfile)
		}
	}
	if filepath.Base(plan.ManifestPath) != "phase1_20260401T120000Z.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
}

func TestBenchmarkPlanMarshalIndentedJSON(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    models: [gpt-oss:20b]
    experiment_name: formal-mode
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildPhase1Plan(loaded, BenchmarkPlanOptions{
		RunID:   "20260401T120000Z",
		Command: "researchctl phase1 plan",
	})
	if err != nil {
		t.Fatalf("BuildPhase1Plan() error = %v", err)
	}

	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		t.Fatalf("MarshalIndentedJSON() error = %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "\"phase\": \"phase1\"") {
		t.Fatalf("manifest json missing phase field: %s", text)
	}
	if !strings.Contains(text, "\"jobs\": [") {
		t.Fatalf("manifest json missing jobs array: %s", text)
	}
	if !strings.Contains(text, "\"requested_sampling_family\"") {
		t.Fatalf("manifest json missing requested_sampling_family: %s", text)
	}
	if !strings.Contains(text, "\"experiment_id\"") {
		t.Fatalf("manifest json missing experiment_id: %s", text)
	}
}

func TestBuildPhase1PlanUsesConfiguredRequestTimeoutAbortThresholdWhenCLIUnset(t *testing.T) {
	loaded := loadBenchmarkPlannerConfig(t, 10, 12)
	plan, err := BuildPhase1Plan(loaded, BenchmarkPlanOptions{
		RunID:   "20260401T120000Z",
		Command: "researchctl phase1 plan",
	})
	if err != nil {
		t.Fatalf("BuildPhase1Plan() error = %v", err)
	}
	if plan.RequestTimeoutAbortThreshold != 10 {
		t.Fatalf("plan.RequestTimeoutAbortThreshold = %d, want 10", plan.RequestTimeoutAbortThreshold)
	}
	if plan.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
		t.Fatalf("plan.InterruptPolicy = %q, want %q", plan.InterruptPolicy, researchconfig.InterruptPolicyDropIfNoResults)
	}
	for _, job := range plan.Jobs {
		if job.RequestTimeoutAbortThreshold != 10 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 10", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want %q", job.Key, job.InterruptPolicy, researchconfig.InterruptPolicyDropIfNoResults)
		}
	}
}

func TestBuildPhase1PlanAllowsRequestTimeoutAbortThresholdOverride(t *testing.T) {
	tests := []struct {
		name     string
		override int
		want     int
	}{
		{name: "disable explicitly", override: 0, want: 0},
		{name: "raise threshold", override: 20, want: 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loaded := loadBenchmarkPlannerConfig(t, 10, 12)
			override := tc.override
			plan, err := BuildPhase1Plan(loaded, BenchmarkPlanOptions{
				RunID:                        "20260401T120000Z",
				Command:                      "researchctl phase1 plan",
				RequestTimeoutAbortThreshold: &override,
				InterruptPolicy:              optionalStringPlannerPtr("keep"),
			})
			if err != nil {
				t.Fatalf("BuildPhase1Plan() error = %v", err)
			}
			if plan.RequestTimeoutAbortThreshold != tc.want {
				t.Fatalf("plan.RequestTimeoutAbortThreshold = %d, want %d", plan.RequestTimeoutAbortThreshold, tc.want)
			}
			for _, job := range plan.Jobs {
				if job.RequestTimeoutAbortThreshold != tc.want {
					t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want %d", job.Key, job.RequestTimeoutAbortThreshold, tc.want)
				}
				if job.InterruptPolicy != researchconfig.InterruptPolicyKeep {
					t.Fatalf("job %q InterruptPolicy = %q, want %q", job.Key, job.InterruptPolicy, researchconfig.InterruptPolicyKeep)
				}
			}
		})
	}
}

func TestBuildBenchmarkPlanSeparatesRequestedAndEffectiveSamplingBySurface(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
paths:
  artifact_root: research/artifacts
  report_root: research/result_research_report_v1
  temp_root: .tmp/research
  manifest_root: research/artifacts/result_research/manifests
state:
  db_path: research/artifacts/result_research/research_db/research_db.sqlite
transports:
  google-public:
    provider: google
    base_url: https://generativelanguage.googleapis.com/v1beta
    api_key_env: GEMINI_API_KEY
families:
  google-family:
    transport: google-public
    experiment_name: formal-mode
    models: [gemini-2.5-flash]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [google-family]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildPhase1Plan(loaded, BenchmarkPlanOptions{
		RunID:          "20260402T070000Z",
		Command:        "researchctl phase1 plan",
		RequireSecrets: false,
	})
	if err != nil {
		t.Fatalf("BuildPhase1Plan() error = %v", err)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("len(plan.Jobs) = %d, want 1", len(plan.Jobs))
	}
	job := plan.Jobs[0]
	if job.SamplingSurface != "google_generate_content" {
		t.Fatalf("job.SamplingSurface = %q, want google_generate_content", job.SamplingSurface)
	}
	if job.RequestedSamplingProfile != "temp0_seed42_topp1" {
		t.Fatalf("job.RequestedSamplingProfile = %q, want temp0_seed42_topp1", job.RequestedSamplingProfile)
	}
	if job.EffectiveSamplingProfile != "temp0" {
		t.Fatalf("job.EffectiveSamplingProfile = %q, want temp0", job.EffectiveSamplingProfile)
	}
	if job.RequestedSampling == nil || job.RequestedSampling.Seed == nil || *job.RequestedSampling.Seed != 42 {
		t.Fatalf("job.RequestedSampling = %#v, want seed=42", job.RequestedSampling)
	}
	if job.EffectiveSampling == nil || job.EffectiveSampling.Seed != nil || job.EffectiveSampling.TopP != nil {
		t.Fatalf("job.EffectiveSampling = %#v, want temperature-only", job.EffectiveSampling)
	}
	if job.UnsupportedSampling == nil || job.UnsupportedSampling.Seed == nil || *job.UnsupportedSampling.Seed != 42 {
		t.Fatalf("job.UnsupportedSampling = %#v, want seed in unsupported set", job.UnsupportedSampling)
	}
	if job.UnsupportedSampling.TopP == nil || *job.UnsupportedSampling.TopP != 1 {
		t.Fatalf("job.UnsupportedSampling = %#v, want top_p in unsupported set", job.UnsupportedSampling)
	}
}

func TestBuildCompositionalAssumptionImportPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  compositional-main:
    transport: local-compatible
    experiment_name: compositional-assumption-import-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 2
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [compositional-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalAssumptionImportPlan(loaded, BenchmarkPlanOptions{
		RunID:   "compositional-20260403",
		Command: "researchctl compositional-assumption-import run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalAssumptionImportPlan() error = %v", err)
	}
	if plan.Phase != "compositional-assumption-import" {
		t.Fatalf("plan.Phase = %q, want compositional-assumption-import", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-assumption-import_compositional-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional/") {
		t.Fatalf("summary path = %q, want compositional summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/assumption-import/") {
		t.Fatalf("report path = %q, want compositional report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 2 {
		t.Fatalf("len(plan.Jobs) = %d, want 2", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-assumption-import" {
			t.Fatalf("job %q phase = %q, want compositional-assumption-import", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-assumption-import-phase1.csv" {
			t.Fatalf("job %q selector = %q, want compositional case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 5 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 5", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildBridgeImportFinalResearchPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: local-compatible-default
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  bridge-main:
    transport: local-compatible
    experiment_name: bridge-import-final-research-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [bridge-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [bridge-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [bridge-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  bridge_import_final_research:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/bridge-import-final-research-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [bridge-main]
    repeats: 2
    request_timeout_abort_threshold: 6
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/bridge-import-final-research
    report_dir: research/result_research_report_v1/compositional/bridge-import-final-research
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildBridgeImportFinalResearchPlan(loaded, BenchmarkPlanOptions{
		RunID:   "bridge-20260404",
		Command: "researchctl bridge-import-final-research run",
	})
	if err != nil {
		t.Fatalf("BuildBridgeImportFinalResearchPlan() error = %v", err)
	}
	if plan.Phase != "bridge-import-final-research" {
		t.Fatalf("plan.Phase = %q, want bridge-import-final-research", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "bridge-import-final-research_bridge-20260404.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/bridge-import-final-research/") {
		t.Fatalf("summary path = %q, want bridge-import-final-research summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/bridge-import-final-research/") {
		t.Fatalf("report path = %q, want bridge-import-final-research report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 2 {
		t.Fatalf("len(plan.Jobs) = %d, want 2", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "bridge-import-final-research" {
			t.Fatalf("job %q phase = %q, want bridge-import-final-research", job.Key, job.Phase)
		}
		if job.Selector != "cases/bridge-import-final-research-phase1.csv" {
			t.Fatalf("job %q selector = %q, want bridge-import-final-research case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 6 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 6", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildBridgeOnlyAuthoringPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: local-compatible-default
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  bridge-main:
    transport: local-compatible
    experiment_name: bridge-import-final-research-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [bridge-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [bridge-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  bridge_import_final_research:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/bridge-import-final-research-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [bridge-main]
    repeats: 2
    request_timeout_abort_threshold: 6
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/bridge-import-final-research
    report_dir: research/result_research_report_v1/compositional/bridge-import-final-research
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildBridgeOnlyAuthoringPlan(loaded, BenchmarkPlanOptions{
		RunID:   "bridge-only-20260404",
		Command: "researchctl bridge-only-authoring run",
	})
	if err != nil {
		t.Fatalf("BuildBridgeOnlyAuthoringPlan() error = %v", err)
	}
	if plan.Phase != "bridge-only-authoring" {
		t.Fatalf("plan.Phase = %q, want bridge-only-authoring", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "bridge-only-authoring_bridge-only-20260404.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/bridge-only-authoring/") {
		t.Fatalf("summary path = %q, want bridge-only-authoring summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/bridge-only-authoring/") {
		t.Fatalf("report path = %q, want bridge-only-authoring report dir", plan.ReportPath)
	}
	if plan.PromptVersion != "hilbert-ai-verification-bridge-only-v1.0" {
		t.Fatalf("plan.PromptVersion = %q, want bridge-only prompt", plan.PromptVersion)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("len(plan.Jobs) = %d, want 1", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "bridge-only-authoring" {
			t.Fatalf("job %q phase = %q, want bridge-only-authoring", job.Key, job.Phase)
		}
		if job.Selector != "cases/bridge-only-authoring-phase1.csv" {
			t.Fatalf("job %q selector = %q, want bridge-only-authoring case pack", job.Key, job.Selector)
		}
	}
}

func TestBuildGoldFinalCompositionOnlyPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: local-compatible-default
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  gold-main:
    transport: local-compatible
    experiment_name: gold-final-composition-only-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [gold-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [gold-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [gold-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildGoldFinalCompositionOnlyPlan(loaded, BenchmarkPlanOptions{
		RunID:   "gold-final-only-20260404",
		Command: "researchctl gold-final-composition-only run",
	})
	if err != nil {
		t.Fatalf("BuildGoldFinalCompositionOnlyPlan() error = %v", err)
	}
	if plan.Phase != "gold-final-composition-only" {
		t.Fatalf("plan.Phase = %q, want gold-final-composition-only", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "gold-final-composition-only_gold-final-only-20260404.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/gold-final-composition-only/") {
		t.Fatalf("summary path = %q, want gold-final-composition-only summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/gold-final-composition-only/") {
		t.Fatalf("report path = %q, want gold-final-composition-only report dir", plan.ReportPath)
	}
	if plan.PromptVersion != "hilbert-ai-verification-gold-final-only-v1.0" {
		t.Fatalf("plan.PromptVersion = %q, want gold-final-only prompt", plan.PromptVersion)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("len(plan.Jobs) = %d, want 1", len(plan.Jobs))
	}
	job := plan.Jobs[0]
	if job.Phase != "gold-final-composition-only" {
		t.Fatalf("job phase = %q, want gold-final-composition-only", job.Phase)
	}
	if job.Selector != "cases/gold-final-composition-only-phase1.csv" {
		t.Fatalf("job selector = %q, want gold-final-composition-only case pack", job.Selector)
	}
}

func TestBuildCompositionalDepthLadderPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  depth-ladder-main:
    transport: local-compatible
    experiment_name: compositional-depth-ladder-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 3
    request_timeout_abort_threshold: 6
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [depth-ladder-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalDepthLadderPlan(loaded, BenchmarkPlanOptions{
		RunID:   "depth-ladder-20260403",
		Command: "researchctl compositional-depth-ladder run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalDepthLadderPlan() error = %v", err)
	}
	if plan.Phase != "compositional-depth-ladder" {
		t.Fatalf("plan.Phase = %q, want compositional-depth-ladder", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-depth-ladder_depth-ladder-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-depth-ladder/") {
		t.Fatalf("summary path = %q, want compositional-depth-ladder summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/depth-ladder/") {
		t.Fatalf("report path = %q, want compositional-depth-ladder report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 3 {
		t.Fatalf("len(plan.Jobs) = %d, want 3", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-depth-ladder" {
			t.Fatalf("job %q phase = %q, want compositional-depth-ladder", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-depth-ladder-phase1.csv" {
			t.Fatalf("job %q selector = %q, want compositional depth ladder case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 6 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 6", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalBranchingPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  branching-main:
    transport: local-compatible
    experiment_name: compositional-branching-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 4
    request_timeout_abort_threshold: 8
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [branching-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalBranchingPlan(loaded, BenchmarkPlanOptions{
		RunID:   "branching-20260403",
		Command: "researchctl compositional-branching run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalBranchingPlan() error = %v", err)
	}
	if plan.Phase != "compositional-branching" {
		t.Fatalf("plan.Phase = %q, want compositional-branching", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-branching_branching-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-branching/") {
		t.Fatalf("summary path = %q, want compositional-branching summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/branching/") {
		t.Fatalf("report path = %q, want compositional-branching report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 4 {
		t.Fatalf("len(plan.Jobs) = %d, want 4", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-branching" {
			t.Fatalf("job %q phase = %q, want compositional-branching", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-branching-phase1.csv" {
			t.Fatalf("job %q selector = %q, want compositional branching case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 8 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 8", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilyReusePlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  mixed-family-main:
    transport: local-compatible
    experiment_name: compositional-mixed-family-reuse-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 4
    request_timeout_abort_threshold: 9
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalMixedFamilyReusePlan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-20260403",
		Command: "researchctl compositional-mixed-family-reuse run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilyReusePlan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-reuse" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-reuse", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-reuse_mixed-family-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-reuse/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-reuse summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-reuse/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-reuse report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 4 {
		t.Fatalf("len(plan.Jobs) = %d, want 4", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-reuse" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-reuse", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-reuse-phase1.csv" {
			t.Fatalf("job %q selector = %q, want compositional mixed-family reuse case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 9 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 9", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilyGoldFirstPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  mixed-family-gold-first-main:
    transport: local-compatible
    experiment_name: compositional-mixed-family-gold-first-v1
    models: [gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-main]
    repeats: 3
    request_timeout_abort_threshold: 7
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalMixedFamilyGoldFirstPlan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-gold-first-20260403",
		Command: "researchctl compositional-mixed-family-gold-first run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilyGoldFirstPlan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-gold-first" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-gold-first", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-gold-first_mixed-family-gold-first-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-gold-first/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-gold-first summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-gold-first/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-gold-first report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 3 {
		t.Fatalf("len(plan.Jobs) = %d, want 3", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-gold-first" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-gold-first", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-gold-first-phase1.csv" {
			t.Fatalf("job %q selector = %q, want compositional mixed-family gold-first case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 7 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 7", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilyGoldFirstPhase2PlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: compositional-mixed-family-gold-first-phase2-v1
    models: [gpt-oss:120b-cloud, glm-5:cloud, deepseek-v3.1:671b-cloud, gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  mixed-family-gold-first-phase2-main:
    transport: local-compatible
    experiment_name: compositional-mixed-family-gold-first-phase2-v1
    models: [gpt-oss:120b-cloud, glm-5:cloud, deepseek-v3.1:671b-cloud, gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-phase2-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [mixed-family-gold-first-phase2-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-phase2-main]
    repeats: 3
    request_timeout_abort_threshold: 7
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
  compositional_mixed_family_gold_first_phase2:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase2.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-phase2-main]
    repeats: 3
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first-phase2
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first-phase2
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalMixedFamilyGoldFirstPhase2Plan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-gold-first-phase2-20260403",
		Command: "researchctl compositional-mixed-family-gold-first-phase2 run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilyGoldFirstPhase2Plan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-gold-first-phase2" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-gold-first-phase2", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-gold-first-phase2_mixed-family-gold-first-phase2-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-gold-first-phase2/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-gold-first-phase2 summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-gold-first-phase2/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-gold-first-phase2 report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 12 {
		t.Fatalf("len(plan.Jobs) = %d, want 12", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-gold-first-phase2" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-gold-first-phase2", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-gold-first-phase2.csv" {
			t.Fatalf("job %q selector = %q, want phase2 case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 5 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 5", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilyGoldFirstHardPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: compositional-mixed-family-gold-first-hard-v1
    models: [gpt-oss:120b-cloud, glm-5:cloud, deepseek-v3.1:671b-cloud, gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  mixed-family-gold-first-hard-main:
    transport: local-compatible
    experiment_name: compositional-mixed-family-gold-first-hard-v1
    models: [gpt-oss:120b-cloud, glm-5:cloud, deepseek-v3.1:671b-cloud, gpt-oss:20b]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-hard-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [mixed-family-gold-first-hard-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-hard-main]
    repeats: 3
    request_timeout_abort_threshold: 7
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
  compositional_mixed_family_gold_first_phase2:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase2.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-hard-main]
    repeats: 3
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first-phase2
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first-phase2
  compositional_mixed_family_gold_first_hard:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-hard.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-gold-first-hard-main]
    repeats: 3
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first-hard
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first-hard
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	plan, err := BuildCompositionalMixedFamilyGoldFirstHardPlan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-gold-first-hard-20260403",
		Command: "researchctl compositional-mixed-family-gold-first-hard run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilyGoldFirstHardPlan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-gold-first-hard" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-gold-first-hard", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-gold-first-hard_mixed-family-gold-first-hard-20260403.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-gold-first-hard/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-gold-first-hard summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-gold-first-hard/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-gold-first-hard report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 12 {
		t.Fatalf("len(plan.Jobs) = %d, want 12", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-gold-first-hard" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-gold-first-hard", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-gold-first-hard.csv" {
			t.Fatalf("job %q selector = %q, want hard case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 5 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 5", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilySemiGoldStablePlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	loaded := loadSemiGoldPlannerConfig(t)
	plan, err := BuildCompositionalMixedFamilySemiGoldStablePlan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-semi-gold-stable-20260404",
		Command: "researchctl compositional-mixed-family-semi-gold-stable run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilySemiGoldStablePlan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-semi-gold-stable" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-semi-gold-stable", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-semi-gold-stable_mixed-family-semi-gold-stable-20260404.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-semi-gold-stable/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-semi-gold-stable summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-semi-gold-stable/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-semi-gold-stable report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 12 {
		t.Fatalf("len(plan.Jobs) = %d, want 12", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-semi-gold-stable" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-semi-gold-stable", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-semi-gold-stable.csv" {
			t.Fatalf("job %q selector = %q, want stable semi-gold case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 5 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 5", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func TestBuildCompositionalMixedFamilySemiGoldFrontierPlanUsesDedicatedSurfaceAndPaths(t *testing.T) {
	loaded := loadSemiGoldPlannerConfig(t)
	plan, err := BuildCompositionalMixedFamilySemiGoldFrontierPlan(loaded, BenchmarkPlanOptions{
		RunID:   "mixed-family-semi-gold-frontier-20260404",
		Command: "researchctl compositional-mixed-family-semi-gold-frontier run",
	})
	if err != nil {
		t.Fatalf("BuildCompositionalMixedFamilySemiGoldFrontierPlan() error = %v", err)
	}
	if plan.Phase != "compositional-mixed-family-semi-gold-frontier" {
		t.Fatalf("plan.Phase = %q, want compositional-mixed-family-semi-gold-frontier", plan.Phase)
	}
	if filepath.Base(plan.ManifestPath) != "compositional-mixed-family-semi-gold-frontier_mixed-family-semi-gold-frontier-20260404.json" {
		t.Fatalf("manifest basename = %q", filepath.Base(plan.ManifestPath))
	}
	if !strings.Contains(filepath.ToSlash(plan.SummaryPath), "/result_research/compositional-mixed-family-semi-gold-frontier/") {
		t.Fatalf("summary path = %q, want compositional-mixed-family-semi-gold-frontier summary dir", plan.SummaryPath)
	}
	if !strings.Contains(filepath.ToSlash(plan.ReportPath), "/result_research_report_v1/compositional/mixed-family-semi-gold-frontier/") {
		t.Fatalf("report path = %q, want compositional-mixed-family-semi-gold-frontier report dir", plan.ReportPath)
	}
	if len(plan.Jobs) != 12 {
		t.Fatalf("len(plan.Jobs) = %d, want 12", len(plan.Jobs))
	}
	for _, job := range plan.Jobs {
		if job.Phase != "compositional-mixed-family-semi-gold-frontier" {
			t.Fatalf("job %q phase = %q, want compositional-mixed-family-semi-gold-frontier", job.Key, job.Phase)
		}
		if job.Selector != "cases/compositional-mixed-family-semi-gold-frontier.csv" {
			t.Fatalf("job %q selector = %q, want frontier semi-gold case pack", job.Key, job.Selector)
		}
		if job.RequestTimeoutAbortThreshold != 5 {
			t.Fatalf("job %q RequestTimeoutAbortThreshold = %d, want 5", job.Key, job.RequestTimeoutAbortThreshold)
		}
		if job.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
			t.Fatalf("job %q InterruptPolicy = %q, want drop_if_no_results", job.Key, job.InterruptPolicy)
		}
	}
}

func loadSemiGoldPlannerConfig(t *testing.T) researchconfig.Loaded {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    experiment_name: compositional-mixed-family-semi-gold-v1
    models: [deepseek-v3.1:671b-cloud, gpt-oss:120b-cloud, glm-5:cloud, gpt-oss:20b-cloud]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
  mixed-family-semi-gold-main:
    transport: local-compatible
    experiment_name: compositional-mixed-family-semi-gold-v1
    models: [deepseek-v3.1:671b-cloud, gpt-oss:120b-cloud, glm-5:cloud, gpt-oss:20b-cloud]
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-semi-gold-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [mixed-family-semi-gold-main]
    repeats: 1
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_mixed_family_semi_gold_stable:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-semi-gold-stable.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-semi-gold-main]
    repeats: 3
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-semi-gold-stable
    report_dir: research/result_research_report_v1/compositional/mixed-family-semi-gold-stable
  compositional_mixed_family_semi_gold_frontier:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-semi-gold-frontier.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [mixed-family-semi-gold-main]
    repeats: 3
    request_timeout_abort_threshold: 5
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/compositional-mixed-family-semi-gold-frontier
    report_dir: research/result_research_report_v1/compositional/mixed-family-semi-gold-frontier
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
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}
	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return loaded
}

func loadBenchmarkPlannerConfig(t *testing.T, phase1Threshold, ndThreshold int) researchconfig.Loaded {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := fmt.Sprintf(`schema_version: researchctl.config/v1
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
    timeout: 60s
families:
  local-compatible:
    transport: local-compatible
    models: [gpt-oss:20b]
    experiment_name: formal-mode
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: %d
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: %d
    interrupt_policy: drop_if_no_results
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
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
`, phase1Threshold, ndThreshold)
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}
	loaded, err := researchconfig.Load(root, filepath.Join("research", "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return loaded
}

func optionalStringPlannerPtr(value string) *string {
	return &value
}
