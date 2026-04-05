package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesLocalConfigAndEnvFile(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	defaultConfig := `schema_version: researchctl.config/v1
env_file: research/config/test.env
paths:
  artifact_root: research/artifacts
state:
  db_path: research/artifacts/result_research/research_db/research_db.sqlite
transports:
  local-compatible:
    provider: compatible
    base_url: http://localhost:11434
  openai-public:
    provider: openai
    base_url: https://api.openai.com/v1
    api_key_env: TEST_OPENAI_API_KEY
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
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
  nd:
    project_folder: hilbert-ai-verification-benchmark-nd-v1
    input_kind: theorems
    input_file: theorems/pilot_shared_20260327.csv
    prompt_version: hilbert-ai-verification-benchmark-nd-v1.1
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/nd
    report_dir: research/result_research_report_v1/nd
  compositional_assumption_import:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-assumption-import-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional
    report_dir: research/result_research_report_v1/compositional/assumption-import
  bridge_import_final_research:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/bridge-import-final-research-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/bridge-import-final-research
    report_dir: research/result_research_report_v1/compositional/bridge-import-final-research
  bridge_only_authoring:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/bridge-only-authoring-phase1.csv
    prompt_version: hilbert-ai-verification-bridge-only-v1.0
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/bridge-only-authoring
    report_dir: research/result_research_report_v1/compositional/bridge-only-authoring
  gold_final_composition_only:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/gold-final-composition-only-phase1.csv
    prompt_version: hilbert-ai-verification-gold-final-only-v1.0
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/gold-final-composition-only
    report_dir: research/result_research_report_v1/compositional/gold-final-composition-only
  compositional_depth_ladder:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-depth-ladder-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional-depth-ladder
    report_dir: research/result_research_report_v1/compositional/depth-ladder
  compositional_branching:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-branching-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional-branching
    report_dir: research/result_research_report_v1/compositional/branching
  compositional_mixed_family_reuse:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-reuse-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional-mixed-family-reuse
    report_dir: research/result_research_report_v1/compositional/mixed-family-reuse
  compositional_mixed_family_gold_first:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase1.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first
  compositional_mixed_family_gold_first_phase2:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-phase2.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
    summary_dir: research/artifacts/result_research/compositional-mixed-family-gold-first-phase2
    report_dir: research/result_research_report_v1/compositional/mixed-family-gold-first-phase2
  compositional_mixed_family_gold_first_hard:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases/compositional-mixed-family-gold-first-hard.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    request_timeout_abort_threshold: 0
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
  meta_dir: research/result_research_report_v1/summary/meta
  cases_dir: research/result_research_report_v1/summary/cases
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`
	localConfig := `benchmarks:
  phase1:
    request_timeout_abort_threshold: 10
  nd:
    request_timeout_abort_threshold: 12
  compositional_assumption_import:
    request_timeout_abort_threshold: 7
  bridge_import_final_research:
    request_timeout_abort_threshold: 8
  bridge_only_authoring:
    request_timeout_abort_threshold: 9
  gold_final_composition_only:
    request_timeout_abort_threshold: 10
  compositional_depth_ladder:
    request_timeout_abort_threshold: 11
  compositional_branching:
    request_timeout_abort_threshold: 13
  compositional_mixed_family_reuse:
    request_timeout_abort_threshold: 15
  compositional_mixed_family_gold_first:
    request_timeout_abort_threshold: 17
  compositional_mixed_family_gold_first_phase2:
    request_timeout_abort_threshold: 19
  compositional_mixed_family_gold_first_hard:
    request_timeout_abort_threshold: 21
transports:
  local-compatible:
    base_url: http://127.0.0.1:11435
families:
  local-compatible:
    models: [gpt-oss:120b-cloud]
    experiment_name: balanced-mode
    sampling:
      top_p: 0.9
`
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(defaultConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}
	workflowDir := filepath.Join(configDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workflowDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "default-local.yaml"), []byte(localConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default-local.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "test.env"), []byte("TEST_OPENAI_API_KEY=test-secret\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(test.env) error = %v", err)
	}

	loaded, err := Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "workflow", "default-local.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.LocalConfigFound {
		t.Fatalf("Load() LocalConfigFound = false, want true")
	}
	if got := loaded.Config.Transports["local-compatible"].BaseURL; got != "http://127.0.0.1:11435" {
		t.Fatalf("local-compatible base_url = %q, want override", got)
	}
	models := loaded.Config.Families["local-compatible"].Models
	if len(models) != 1 || models[0] != "gpt-oss:120b-cloud" {
		t.Fatalf("local-compatible models = %#v, want local override", models)
	}
	family := loaded.Config.Families["local-compatible"]
	if family.ExperimentName != "balanced-mode" {
		t.Fatalf("local-compatible experiment_name = %q, want balanced-mode", family.ExperimentName)
	}
	if family.Sampling == nil || family.Sampling.TopP == nil || *family.Sampling.TopP != 0.9 {
		t.Fatalf("local-compatible sampling.top_p = %#v, want 0.9", family.Sampling)
	}
	if family.Sampling.Temperature == nil || *family.Sampling.Temperature != 0 {
		t.Fatalf("local-compatible sampling.temperature = %#v, want inherited 0", family.Sampling)
	}
	if family.Sampling.Seed == nil || *family.Sampling.Seed != 42 {
		t.Fatalf("local-compatible sampling.seed = %#v, want inherited 42", family.Sampling)
	}
	if got := loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold; got != 10 {
		t.Fatalf("phase1 request_timeout_abort_threshold = %d, want 10", got)
	}
	if got := loaded.Config.Benchmarks.ND.RequestTimeoutAbortThreshold; got != 12 {
		t.Fatalf("nd request_timeout_abort_threshold = %d, want 12", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalAssumptionImport.RequestTimeoutAbortThreshold; got != 7 {
		t.Fatalf("compositional_assumption_import request_timeout_abort_threshold = %d, want 7", got)
	}
	if got := loaded.Config.Benchmarks.BridgeImportFinalResearch.RequestTimeoutAbortThreshold; got != 8 {
		t.Fatalf("bridge_import_final_research request_timeout_abort_threshold = %d, want 8", got)
	}
	if got := loaded.Config.Benchmarks.BridgeOnlyAuthoring.RequestTimeoutAbortThreshold; got != 9 {
		t.Fatalf("bridge_only_authoring request_timeout_abort_threshold = %d, want 9", got)
	}
	if got := loaded.Config.Benchmarks.GoldFinalCompositionOnly.RequestTimeoutAbortThreshold; got != 10 {
		t.Fatalf("gold_final_composition_only request_timeout_abort_threshold = %d, want 10", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalDepthLadder.RequestTimeoutAbortThreshold; got != 11 {
		t.Fatalf("compositional_depth_ladder request_timeout_abort_threshold = %d, want 11", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalBranching.RequestTimeoutAbortThreshold; got != 13 {
		t.Fatalf("compositional_branching request_timeout_abort_threshold = %d, want 13", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyReuse.RequestTimeoutAbortThreshold; got != 15 {
		t.Fatalf("compositional_mixed_family_reuse request_timeout_abort_threshold = %d, want 15", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirst.RequestTimeoutAbortThreshold; got != 17 {
		t.Fatalf("compositional_mixed_family_gold_first request_timeout_abort_threshold = %d, want 17", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.RequestTimeoutAbortThreshold; got != 19 {
		t.Fatalf("compositional_mixed_family_gold_first_phase2 request_timeout_abort_threshold = %d, want 19", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstHard.RequestTimeoutAbortThreshold; got != 21 {
		t.Fatalf("compositional_mixed_family_gold_first_hard request_timeout_abort_threshold = %d, want 21", got)
	}
	if got := loaded.Config.Benchmarks.Phase1.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("phase1 interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.ND.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("nd interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalAssumptionImport.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_assumption_import interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.BridgeImportFinalResearch.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("bridge_import_final_research interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.BridgeOnlyAuthoring.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("bridge_only_authoring interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.GoldFinalCompositionOnly.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("gold_final_composition_only interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalDepthLadder.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_depth_ladder interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalBranching.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_branching interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyReuse.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_mixed_family_reuse interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirst.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_mixed_family_gold_first interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_mixed_family_gold_first_phase2 interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstHard.InterruptPolicy; got != InterruptPolicyDropIfNoResults {
		t.Fatalf("compositional_mixed_family_gold_first_hard interrupt_policy = %q, want %q", got, InterruptPolicyDropIfNoResults)
	}
	if got := loaded.Config.Benchmarks.CompositionalAssumptionImport.SummaryDir; got != "research/artifacts/result_research/compositional" {
		t.Fatalf("compositional_assumption_import summary_dir = %q, want compositional dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalAssumptionImport.ReportDir; got != "research/result_research_report_v1/compositional/assumption-import" {
		t.Fatalf("compositional_assumption_import report_dir = %q, want compositional/assumption-import dir", got)
	}
	if got := loaded.Config.Benchmarks.BridgeImportFinalResearch.SummaryDir; got != "research/artifacts/result_research/bridge-import-final-research" {
		t.Fatalf("bridge_import_final_research summary_dir = %q, want bridge-import-final-research dir", got)
	}
	if got := loaded.Config.Benchmarks.BridgeImportFinalResearch.ReportDir; got != "research/result_research_report_v1/compositional/bridge-import-final-research" {
		t.Fatalf("bridge_import_final_research report_dir = %q, want compositional/bridge-import-final-research dir", got)
	}
	if got := loaded.Config.Benchmarks.BridgeOnlyAuthoring.SummaryDir; got != "research/artifacts/result_research/bridge-only-authoring" {
		t.Fatalf("bridge_only_authoring summary_dir = %q, want bridge-only-authoring dir", got)
	}
	if got := loaded.Config.Benchmarks.BridgeOnlyAuthoring.ReportDir; got != "research/result_research_report_v1/compositional/bridge-only-authoring" {
		t.Fatalf("bridge_only_authoring report_dir = %q, want compositional/bridge-only-authoring dir", got)
	}
	if got := loaded.Config.Benchmarks.GoldFinalCompositionOnly.SummaryDir; got != "research/artifacts/result_research/gold-final-composition-only" {
		t.Fatalf("gold_final_composition_only summary_dir = %q, want gold-final-composition-only dir", got)
	}
	if got := loaded.Config.Benchmarks.GoldFinalCompositionOnly.ReportDir; got != "research/result_research_report_v1/compositional/gold-final-composition-only" {
		t.Fatalf("gold_final_composition_only report_dir = %q, want compositional/gold-final-composition-only dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalDepthLadder.SummaryDir; got != "research/artifacts/result_research/compositional-depth-ladder" {
		t.Fatalf("compositional_depth_ladder summary_dir = %q, want compositional-depth-ladder dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalDepthLadder.ReportDir; got != "research/result_research_report_v1/compositional/depth-ladder" {
		t.Fatalf("compositional_depth_ladder report_dir = %q, want compositional/depth-ladder dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalBranching.SummaryDir; got != "research/artifacts/result_research/compositional-branching" {
		t.Fatalf("compositional_branching summary_dir = %q, want compositional-branching dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalBranching.ReportDir; got != "research/result_research_report_v1/compositional/branching" {
		t.Fatalf("compositional_branching report_dir = %q, want compositional/branching dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyReuse.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-reuse" {
		t.Fatalf("compositional_mixed_family_reuse summary_dir = %q, want compositional-mixed-family-reuse dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyReuse.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-reuse" {
		t.Fatalf("compositional_mixed_family_reuse report_dir = %q, want compositional/mixed-family-reuse dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirst.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-gold-first" {
		t.Fatalf("compositional_mixed_family_gold_first summary_dir = %q, want compositional-mixed-family-gold-first dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirst.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-gold-first" {
		t.Fatalf("compositional_mixed_family_gold_first report_dir = %q, want compositional/mixed-family-gold-first dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-gold-first-phase2" {
		t.Fatalf("compositional_mixed_family_gold_first_phase2 summary_dir = %q, want compositional-mixed-family-gold-first-phase2 dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-gold-first-phase2" {
		t.Fatalf("compositional_mixed_family_gold_first_phase2 report_dir = %q, want compositional/mixed-family-gold-first-phase2 dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstHard.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-gold-first-hard" {
		t.Fatalf("compositional_mixed_family_gold_first_hard summary_dir = %q, want compositional-mixed-family-gold-first-hard dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstHard.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-gold-first-hard" {
		t.Fatalf("compositional_mixed_family_gold_first_hard report_dir = %q, want compositional/mixed-family-gold-first-hard dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldStable.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-semi-gold-stable" {
		t.Fatalf("compositional_mixed_family_semi_gold_stable summary_dir = %q, want compositional-mixed-family-semi-gold-stable dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldStable.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-semi-gold-stable" {
		t.Fatalf("compositional_mixed_family_semi_gold_stable report_dir = %q, want compositional/mixed-family-semi-gold-stable dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.SummaryDir; got != "research/artifacts/result_research/compositional-mixed-family-semi-gold-frontier" {
		t.Fatalf("compositional_mixed_family_semi_gold_frontier summary_dir = %q, want compositional-mixed-family-semi-gold-frontier dir", got)
	}
	if got := loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.ReportDir; got != "research/result_research_report_v1/compositional/mixed-family-semi-gold-frontier" {
		t.Fatalf("compositional_mixed_family_semi_gold_frontier report_dir = %q, want compositional/mixed-family-semi-gold-frontier dir", got)
	}
	if got := loaded.Config.Reports.MetaDir; got != "research/result_research_report_v1/summary/meta" {
		t.Fatalf("reports.meta_dir = %q, want summary/meta dir", got)
	}
	if got := loaded.Config.Reports.CasesDir; got != "research/result_research_report_v1/summary/cases" {
		t.Fatalf("reports.cases_dir = %q, want summary/cases dir", got)
	}
	transport, err := loaded.ResolveTransport("openai-public", true)
	if err != nil {
		t.Fatalf("ResolveTransport() error = %v", err)
	}
	if transport.APIKey != "test-secret" {
		t.Fatalf("ResolveTransport() APIKey = %q, want env file value", transport.APIKey)
	}
}

func TestLoadSupportsRelativeExtendsCompatibilityWrappers(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	directDir := filepath.Join(configDir, "direct")
	if err := os.MkdirAll(directDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(directDir) error = %v", err)
	}

	defaultConfig := `schema_version: researchctl.config/v1
paths:
  artifact_root: research/artifacts
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
reports:
  benchmark_dir: research/result_research_report_v1/direct
  nd_dir: research/result_research_report_v1/nd
  research_dir: research/result_research_report_v1/pair
  meta_dir: research/result_research_report_v1/summary/meta
  cases_dir: research/result_research_report_v1/summary/cases
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`
	groupedLocal := `transports:
  phase1-compatible:
    provider: compatible
    base_url: http://localhost:11435

families:
  phase1-matrix:
    transport: phase1-compatible
    models: [glm-5:cloud]
    experiment_name: grouped-layout

benchmarks:
  phase1:
    families: [phase1-matrix]
    request_timeout_abort_threshold: 8
`
	compatibilityWrapper := `extends: direct/phase1.yaml

benchmarks:
  phase1:
    repeats: 4
`

	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(defaultConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(directDir, "phase1.yaml"), []byte(groupedLocal), 0o644); err != nil {
		t.Fatalf("WriteFile(direct/phase1.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "local.phase1.yaml"), []byte(compatibilityWrapper), 0o644); err != nil {
		t.Fatalf("WriteFile(local.phase1.yaml) error = %v", err)
	}

	loaded, err := Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "local.phase1.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Config.Transports["phase1-compatible"].BaseURL; got != "http://localhost:11435" {
		t.Fatalf("phase1-compatible base_url = %q, want grouped override", got)
	}
	if got := loaded.Config.Benchmarks.Phase1.Repeats; got != 4 {
		t.Fatalf("phase1 repeats = %d, want wrapper override", got)
	}
	if got := loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold; got != 8 {
		t.Fatalf("phase1 request_timeout_abort_threshold = %d, want grouped inherited value", got)
	}
	models := loaded.Config.Families["phase1-matrix"].Models
	if len(models) != 1 || models[0] != "glm-5:cloud" {
		t.Fatalf("phase1-matrix models = %#v, want grouped inherited family", models)
	}
}

func TestLoadBenchmarkParallelJobsFromLocalConfig(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	workflowDir := filepath.Join(configDir, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workflowDir) error = %v", err)
	}

	defaultConfig := `schema_version: researchctl.config/v1
paths:
  artifact_root: research/artifacts
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
reports:
  benchmark_dir: research/result_research_report_v1/direct
  nd_dir: research/result_research_report_v1/nd
  research_dir: research/result_research_report_v1/pair
  meta_dir: research/result_research_report_v1/summary/meta
  cases_dir: research/result_research_report_v1/summary/cases
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`
	localConfig := `benchmarks:
  phase1:
    jobs: 3
`

	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(defaultConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "default-local.yaml"), []byte(localConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default-local.yaml) error = %v", err)
	}

	loaded, err := Load(root, filepath.Join("research", "config", "default.yaml"), filepath.Join("research", "config", "workflow", "default-local.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Config.Benchmarks.Phase1.ParallelJobs; got != 3 {
		t.Fatalf("phase1 jobs = %d, want 3", got)
	}
	if got := loaded.Config.Benchmarks.ND.ParallelJobs; got != 1 {
		t.Fatalf("nd jobs = %d, want default 1", got)
	}
}
