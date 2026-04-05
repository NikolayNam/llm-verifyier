package planner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	lean4worker "github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	researchlean4 "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/lean4"
)

const WorkflowManifestSchemaVersion = "researchctl.workflow-manifest/v1"

type WorkflowPlanOptions struct {
	RunID                        string
	Command                      string
	RequireSecrets               bool
	RequestTimeoutAbortThreshold *int
	InterruptPolicy              *string
}

type PairedBenchmarkPlan struct {
	SchemaVersion     string         `json:"schema_version"`
	Command           string         `json:"command"`
	Phase             string         `json:"phase"`
	RunID             string         `json:"run_id"`
	CreatedAtUTC      string         `json:"created_at_utc"`
	ConfigFingerprint string         `json:"config_fingerprint"`
	ManifestPath      string         `json:"manifest_path"`
	ProjectFolder     string         `json:"project_folder"`
	InputFile         string         `json:"input_file"`
	InputPath         string         `json:"input_path"`
	PairReportPath    string         `json:"pair_report_path"`
	PairModelCatalog  string         `json:"pair_model_catalog_path"`
	Direct            *BenchmarkPlan `json:"direct"`
	ND                *BenchmarkPlan `json:"nd"`
}

type Phase3Plan struct {
	SchemaVersion       string `json:"schema_version"`
	Command             string `json:"command"`
	Phase               string `json:"phase"`
	RunID               string `json:"run_id"`
	CreatedAtUTC        string `json:"created_at_utc"`
	ConfigFingerprint   string `json:"config_fingerprint"`
	ManifestPath        string `json:"manifest_path"`
	WorkspaceDir        string `json:"workspace_dir"`
	JobName             string `json:"job_name"`
	JobStatement        string `json:"job_statement"`
	JobPayloadPath      string `json:"job_payload_path"`
	TheoremPackPath     string `json:"theorem_pack_path"`
	GenerationConfig    string `json:"generation_config_path"`
	DefaultPayloadPath  string `json:"default_payload_path"`
	ExportOutputPath    string `json:"export_output_path"`
	BenchmarkProjectDir string `json:"benchmark_project_folder"`
}

type Phase3PlanOptions struct {
	RunID          string
	Command        string
	JobName        string
	JobStatement   string
	JobPayloadFile string
}

func (p *PairedBenchmarkPlan) MarshalIndentedJSON() ([]byte, error) {
	type plain PairedBenchmarkPlan
	return json.MarshalIndent((*plain)(p), "", "  ")
}

func (p *PairedBenchmarkPlan) AllJobs() []BenchmarkJob {
	if p == nil {
		return nil
	}
	jobs := make([]BenchmarkJob, 0, 8)
	if p.Direct != nil {
		jobs = append(jobs, p.Direct.Jobs...)
	}
	if p.ND != nil {
		jobs = append(jobs, p.ND.Jobs...)
	}
	return jobs
}

func (p *Phase3Plan) MarshalIndentedJSON() ([]byte, error) {
	type plain Phase3Plan
	return json.MarshalIndent((*plain)(p), "", "  ")
}

func BuildPhase2Plan(loaded researchconfig.Loaded, opts WorkflowPlanOptions) (*PairedBenchmarkPlan, error) {
	runID := normalizeWorkflowRunID(opts.RunID)
	base := loaded.Config.Benchmarks.ND
	directBenchmark := cloneBenchmark(base)
	directBenchmark.PromptVersion = loaded.Config.Benchmarks.Phase1.PromptVersion
	directBenchmark.RequestTimeoutAbortThreshold = loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold
	directBenchmark.SummaryDir = filepath.ToSlash(filepath.Join("research", "artifacts", "result_research", "direct"))
	directBenchmark.ReportDir = loaded.Config.Reports.BenchmarkDir

	directPlan, err := BuildBenchmarkPlan(loaded, "phase2-direct", directBenchmark, BenchmarkPlanOptions{
		RunID:                        runID + "_direct",
		Command:                      opts.Command,
		RequireSecrets:               opts.RequireSecrets,
		RequestTimeoutAbortThreshold: opts.RequestTimeoutAbortThreshold,
		InterruptPolicy:              opts.InterruptPolicy,
	})
	if err != nil {
		return nil, err
	}
	ndPlan, err := BuildBenchmarkPlan(loaded, "phase2-nd", base, BenchmarkPlanOptions{
		RunID:                        runID + "_nd",
		Command:                      opts.Command,
		RequireSecrets:               opts.RequireSecrets,
		RequestTimeoutAbortThreshold: opts.RequestTimeoutAbortThreshold,
		InterruptPolicy:              opts.InterruptPolicy,
	})
	if err != nil {
		return nil, err
	}
	return buildPairedPlan(loaded, "phase2", runID, opts.Command, directPlan, ndPlan), nil
}

func BuildPhase3ComparePlan(loaded researchconfig.Loaded, opts WorkflowPlanOptions, sourceFile string) (*PairedBenchmarkPlan, error) {
	runID := normalizeWorkflowRunID(opts.RunID)
	inputFile := strings.TrimSpace(sourceFile)
	if inputFile == "" {
		inputFile = loaded.Config.Generate.Lean4.ExportOutputFile
	}
	base := loaded.Config.Benchmarks.ND
	base.ProjectFolder = loaded.Config.Generate.Lean4.BenchmarkProjectFolder
	base.InputFile = filepath.ToSlash(inputFile)
	base.InputPath = filepath.ToSlash(resolveBenchmarkInputPath(loaded, inputFile))

	directBenchmark := cloneBenchmark(base)
	directBenchmark.PromptVersion = loaded.Config.Benchmarks.Phase1.PromptVersion
	directBenchmark.RequestTimeoutAbortThreshold = loaded.Config.Benchmarks.Phase1.RequestTimeoutAbortThreshold
	directBenchmark.SummaryDir = filepath.ToSlash(filepath.Join("research", "artifacts", "result_research", "direct"))
	directBenchmark.ReportDir = loaded.Config.Reports.BenchmarkDir

	directPlan, err := BuildBenchmarkPlan(loaded, "phase3-direct", directBenchmark, BenchmarkPlanOptions{
		RunID:                        runID + "_direct",
		Command:                      opts.Command,
		RequireSecrets:               opts.RequireSecrets,
		RequestTimeoutAbortThreshold: opts.RequestTimeoutAbortThreshold,
		InterruptPolicy:              opts.InterruptPolicy,
	})
	if err != nil {
		return nil, err
	}
	ndPlan, err := BuildBenchmarkPlan(loaded, "phase3-nd", base, BenchmarkPlanOptions{
		RunID:                        runID + "_nd",
		Command:                      opts.Command,
		RequireSecrets:               opts.RequireSecrets,
		RequestTimeoutAbortThreshold: opts.RequestTimeoutAbortThreshold,
		InterruptPolicy:              opts.InterruptPolicy,
	})
	if err != nil {
		return nil, err
	}
	return buildPairedPlan(loaded, "phase3-compare", runID, opts.Command, directPlan, ndPlan), nil
}

func BuildPhase3Plan(loaded researchconfig.Loaded, opts Phase3PlanOptions) *Phase3Plan {
	runID := normalizeWorkflowRunID(opts.RunID)
	jobName := firstNonEmpty(strings.TrimSpace(opts.JobName), loaded.Config.Generate.Lean4.JobName)
	jobStatement := firstNonEmpty(strings.TrimSpace(opts.JobStatement), loaded.Config.Generate.Lean4.JobStatement)
	jobPayloadFile := firstNonEmpty(strings.TrimSpace(opts.JobPayloadFile), loaded.Config.Generate.Lean4.JobPayloadFile)
	artifactRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder)))
	theoremsRoot := filepath.Join(artifactRoot, lean4worker.DefaultTheoremsDir)
	return &Phase3Plan{
		SchemaVersion:       WorkflowManifestSchemaVersion,
		Command:             firstNonEmpty(strings.TrimSpace(opts.Command), "phase3 run"),
		Phase:               "phase3",
		RunID:               runID,
		CreatedAtUTC:        time.Now().UTC().Format(time.RFC3339),
		ConfigFingerprint:   loaded.Fingerprint,
		ManifestPath:        filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("phase3_%s.json", runID))),
		WorkspaceDir:        filepath.ToSlash(researchlean4.WorkspaceDir(loaded, runID)),
		JobName:             jobName,
		JobStatement:        jobStatement,
		JobPayloadPath:      filepath.ToSlash(loaded.ResolvePath(jobPayloadFile)),
		TheoremPackPath:     filepath.ToSlash(filepath.Join(theoremsRoot, "theorems.csv")),
		GenerationConfig:    filepath.ToSlash(filepath.Join(theoremsRoot, "generation-config.json")),
		DefaultPayloadPath:  filepath.ToSlash(filepath.Join(theoremsRoot, "payloads", "identity-proof.json")),
		ExportOutputPath:    filepath.ToSlash(filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.BenchmarkProjectFolder))), filepath.FromSlash(loaded.Config.Generate.Lean4.ExportOutputFile))),
		BenchmarkProjectDir: loaded.Config.Generate.Lean4.BenchmarkProjectFolder,
	}
}

func buildPairedPlan(loaded researchconfig.Loaded, phase, runID, command string, directPlan, ndPlan *BenchmarkPlan) *PairedBenchmarkPlan {
	return &PairedBenchmarkPlan{
		SchemaVersion:     WorkflowManifestSchemaVersion,
		Command:           command,
		Phase:             phase,
		RunID:             runID,
		CreatedAtUTC:      time.Now().UTC().Format(time.RFC3339),
		ConfigFingerprint: loaded.Fingerprint,
		ManifestPath:      filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("%s_%s.json", phase, runID))),
		ProjectFolder:     directPlan.ProjectFolder,
		InputFile:         directPlan.InputFile,
		InputPath:         directPlan.InputPath,
		PairReportPath:    filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Reports.ResearchDir), fmt.Sprintf("%s_pair_%s.md", phase, runID))),
		PairModelCatalog:  filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("%s_pair_%s_model_catalog.csv", phase, runID))),
		Direct:            directPlan,
		ND:                ndPlan,
	}
}

func cloneBenchmark(in researchconfig.Benchmark) researchconfig.Benchmark {
	out := in
	if len(in.Families) > 0 {
		out.Families = append([]string(nil), in.Families...)
	}
	return out
}

func normalizeWorkflowRunID(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID != "" {
		return runID
	}
	return defaultTimestampRunID()
}

func resolveBenchmarkInputPath(loaded researchconfig.Loaded, inputFile string) string {
	inputFile = strings.TrimSpace(inputFile)
	if inputFile == "" {
		return ""
	}
	if filepath.IsAbs(filepath.FromSlash(inputFile)) {
		return inputFile
	}
	if strings.HasPrefix(inputFile, "research/") || strings.HasPrefix(inputFile, ".tmp/") {
		return loaded.ResolvePath(inputFile)
	}
	return filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.BenchmarkProjectFolder))), filepath.FromSlash(inputFile))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
