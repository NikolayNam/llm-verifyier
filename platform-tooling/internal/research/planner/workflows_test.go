package planner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestBuildPhase2PlanCreatesDirectAndNDChildren(t *testing.T) {
	loaded := loadWorkflowTestConfig(t)
	plan, err := BuildPhase2Plan(loaded, WorkflowPlanOptions{
		RunID:   "20260401T120000Z",
		Command: "phase2 run",
	})
	if err != nil {
		t.Fatalf("BuildPhase2Plan() error = %v", err)
	}
	if plan.Phase != "phase2" {
		t.Fatalf("plan.Phase = %q, want phase2", plan.Phase)
	}
	if plan.Direct == nil || plan.ND == nil {
		t.Fatalf("paired plan must include direct and nd children")
	}
	if plan.Direct.PromptVersion != loaded.Config.Benchmarks.Phase1.PromptVersion {
		t.Fatalf("direct prompt version = %q, want %q", plan.Direct.PromptVersion, loaded.Config.Benchmarks.Phase1.PromptVersion)
	}
	if plan.ND.PromptVersion != loaded.Config.Benchmarks.ND.PromptVersion {
		t.Fatalf("nd prompt version = %q, want %q", plan.ND.PromptVersion, loaded.Config.Benchmarks.ND.PromptVersion)
	}
	if plan.Direct.RequestTimeoutAbortThreshold != loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold {
		t.Fatalf("direct threshold = %d, want %d", plan.Direct.RequestTimeoutAbortThreshold, loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold)
	}
	if plan.ND.RequestTimeoutAbortThreshold != loaded.Config.Benchmarks.ND.RequestTimeoutAbortThreshold {
		t.Fatalf("nd threshold = %d, want %d", plan.ND.RequestTimeoutAbortThreshold, loaded.Config.Benchmarks.ND.RequestTimeoutAbortThreshold)
	}
	if plan.Direct.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
		t.Fatalf("direct interrupt policy = %q, want %q", plan.Direct.InterruptPolicy, researchconfig.InterruptPolicyDropIfNoResults)
	}
	if plan.ND.InterruptPolicy != researchconfig.InterruptPolicyDropIfNoResults {
		t.Fatalf("nd interrupt policy = %q, want %q", plan.ND.InterruptPolicy, researchconfig.InterruptPolicyDropIfNoResults)
	}
	if !strings.Contains(plan.Direct.SummaryPath, "/result_research/direct/") {
		t.Fatalf("direct summary path = %q, want direct summary root", plan.Direct.SummaryPath)
	}
	if !strings.Contains(plan.ND.SummaryPath, "/result_research/nd/") {
		t.Fatalf("nd summary path = %q, want nd summary root", plan.ND.SummaryPath)
	}
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		t.Fatalf("MarshalIndentedJSON() error = %v", err)
	}
	if !strings.Contains(string(raw), "\"phase\": \"phase2\"") {
		t.Fatalf("paired manifest missing phase2 field: %s", string(raw))
	}
}

func TestBuildPhase3PlanUsesLeanDefaults(t *testing.T) {
	loaded := loadWorkflowTestConfig(t)
	plan := BuildPhase3Plan(loaded, Phase3PlanOptions{
		RunID:   "20260401T120000Z",
		Command: "phase3 run",
	})
	if plan.Phase != "phase3" {
		t.Fatalf("plan.Phase = %q, want phase3", plan.Phase)
	}
	if plan.JobName != "identity_proved" {
		t.Fatalf("plan.JobName = %q, want identity_proved", plan.JobName)
	}
	if plan.JobStatement != "P -> P" {
		t.Fatalf("plan.JobStatement = %q, want P -> P", plan.JobStatement)
	}
	if !strings.HasSuffix(plan.JobPayloadPath, "/research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/payloads/identity-proof.json") {
		t.Fatalf("plan.JobPayloadPath = %q", plan.JobPayloadPath)
	}
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		t.Fatalf("MarshalIndentedJSON() error = %v", err)
	}
	if !strings.Contains(string(raw), "\"phase\": \"phase3\"") {
		t.Fatalf("phase3 manifest missing phase field: %s", string(raw))
	}
}

func loadWorkflowTestConfig(t *testing.T) researchconfig.Loaded {
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
    models: [gpt-oss:20b]
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 10
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
    request_timeout_abort_threshold: 12
    interrupt_policy: drop_if_no_results
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
	return loaded
}
