package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/benchmarkcases"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

const ManifestSchemaVersion = "researchctl.manifest/v1"

type BenchmarkPlan struct {
	SchemaVersion                string         `json:"schema_version"`
	Command                      string         `json:"command"`
	Phase                        string         `json:"phase"`
	RunID                        string         `json:"run_id"`
	CreatedAtUTC                 string         `json:"created_at_utc"`
	ConfigFingerprint            string         `json:"config_fingerprint"`
	ProjectFolder                string         `json:"project_folder"`
	InputKind                    string         `json:"input_kind"`
	InputFile                    string         `json:"input_file"`
	InputPath                    string         `json:"input_path"`
	PromptVersion                string         `json:"prompt_version"`
	SummaryPath                  string         `json:"summary_path"`
	ReportPath                   string         `json:"report_path"`
	ModelCatalogPath             string         `json:"model_catalog_path"`
	ManifestPath                 string         `json:"manifest_path"`
	RepeatCount                  int            `json:"repeat_count"`
	RequestTimeoutAbortThreshold int            `json:"request_timeout_abort_threshold,omitempty"`
	InterruptPolicy              string         `json:"interrupt_policy,omitempty"`
	Jobs                         []BenchmarkJob `json:"jobs"`
}

type BenchmarkJob struct {
	Key                          string                   `json:"key"`
	SpecHash                     string                   `json:"spec_hash"`
	Command                      string                   `json:"command"`
	Phase                        string                   `json:"phase"`
	RunID                        string                   `json:"run_id"`
	Family                       string                   `json:"family"`
	Model                        string                   `json:"model"`
	Repeat                       int                      `json:"repeat"`
	Selector                     string                   `json:"selector"`
	Provider                     string                   `json:"provider"`
	BaseURL                      string                   `json:"base_url"`
	APIKeyEnv                    string                   `json:"api_key_env,omitempty"`
	Timeout                      string                   `json:"timeout"`
	ProjectFolder                string                   `json:"project_folder"`
	InputKind                    string                   `json:"input_kind"`
	InputFile                    string                   `json:"input_file"`
	InputPath                    string                   `json:"input_path"`
	PromptVersion                string                   `json:"prompt_version"`
	ExperimentName               string                   `json:"experiment_name,omitempty"`
	ExperimentID                 string                   `json:"experiment_id,omitempty"`
	SamplingSurface              string                   `json:"sampling_surface,omitempty"`
	RequestedSampling            *researchsampling.Params `json:"requested_sampling_family,omitempty"`
	EffectiveSampling            *researchsampling.Params `json:"effective_sampling_by_surface,omitempty"`
	UnsupportedSampling          *researchsampling.Params `json:"unsupported_or_undocumented,omitempty"`
	RequestedSamplingProfile     string                   `json:"requested_sampling_profile,omitempty"`
	EffectiveSamplingProfile     string                   `json:"effective_sampling_profile,omitempty"`
	RequestedTemperature         *float64                 `json:"-"`
	RequestedSeed                *int64                   `json:"-"`
	RequestedTopP                *float64                 `json:"-"`
	ResultsPath                  string                   `json:"results_path"`
	SummaryPath                  string                   `json:"summary_path"`
	RawDir                       string                   `json:"raw_dir"`
	RequestTimeoutAbortThreshold int                      `json:"request_timeout_abort_threshold,omitempty"`
	InterruptPolicy              string                   `json:"interrupt_policy,omitempty"`
}

type BenchmarkPlanOptions struct {
	RunID                        string
	Command                      string
	RequireSecrets               bool
	RequestTimeoutAbortThreshold *int
	InterruptPolicy              *string
}

func BuildPhase1Plan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "phase1", loaded.Config.Benchmarks.Phase1, opts)
}

func BuildNDPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "nd", loaded.Config.Benchmarks.ND, opts)
}

func BuildCompositionalAssumptionImportPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-assumption-import", loaded.Config.Benchmarks.CompositionalAssumptionImport, opts)
}

func BuildBridgeImportFinalResearchPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "bridge-import-final-research", loaded.Config.Benchmarks.BridgeImportFinalResearch, opts)
}

// This lane removes imported-final closure so failures can be read as
// bridge-authoring / formalization failures rather than downstream reuse noise.
func BuildBridgeOnlyAuthoringPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "bridge-only-authoring", loaded.Config.Benchmarks.BridgeOnlyAuthoring, opts)
}

// This lane does the opposite split: bridge authoring is fixed away and only
// final closure over static gold imports remains under test.
func BuildGoldFinalCompositionOnlyPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "gold-final-composition-only", loaded.Config.Benchmarks.GoldFinalCompositionOnly, opts)
}

func BuildCompositionalDepthLadderPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-depth-ladder", loaded.Config.Benchmarks.CompositionalDepthLadder, opts)
}

func BuildCompositionalBranchingPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-branching", loaded.Config.Benchmarks.CompositionalBranching, opts)
}

func BuildCompositionalMixedFamilyReusePlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-reuse", loaded.Config.Benchmarks.CompositionalMixedFamilyReuse, opts)
}

func BuildCompositionalMixedFamilyGoldFirstPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-gold-first", loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirst, opts)
}

func BuildCompositionalMixedFamilyGoldFirstPhase2Plan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-gold-first-phase2", loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2, opts)
}

func BuildCompositionalMixedFamilyGoldFirstHardPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-gold-first-hard", loaded.Config.Benchmarks.CompositionalMixedFamilyGoldFirstHard, opts)
}

func BuildCompositionalMixedFamilySemiGoldStablePlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-semi-gold-stable", loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldStable, opts)
}

func BuildCompositionalMixedFamilySemiGoldFrontierPlan(loaded researchconfig.Loaded, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	return BuildBenchmarkPlan(loaded, "compositional-mixed-family-semi-gold-frontier", loaded.Config.Benchmarks.CompositionalMixedFamilySemiGoldFrontier, opts)
}

func BuildBenchmarkPlan(loaded researchconfig.Loaded, phase string, benchmark researchconfig.Benchmark, opts BenchmarkPlanOptions) (*BenchmarkPlan, error) {
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = defaultTimestampRunID()
	}
	requestTimeoutAbortThreshold := benchmark.RequestTimeoutAbortThreshold
	if opts.RequestTimeoutAbortThreshold != nil {
		requestTimeoutAbortThreshold = *opts.RequestTimeoutAbortThreshold
	}
	interruptPolicy := researchconfig.NormalizeInterruptPolicy(benchmark.InterruptPolicy)
	if opts.InterruptPolicy != nil {
		interruptPolicy = researchconfig.NormalizeInterruptPolicy(*opts.InterruptPolicy)
	}
	inputPath := strings.TrimSpace(benchmark.InputPath)
	if inputPath == "" {
		inputPath = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ArtifactRoot), benchmark.ProjectFolder, filepath.FromSlash(benchmark.InputFile)))
	} else {
		inputPath = filepath.ToSlash(loaded.ResolvePath(inputPath))
	}
	if benchmarkPhaseUsesChainDependencyValidation(phase) {
		if _, err := os.Stat(inputPath); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat benchmark input: %w", err)
			}
		} else {
			chainCases, err := benchmarkcases.LoadChainCasesCSV(inputPath)
			if err != nil {
				return nil, err
			}
			if err := benchmarkcases.ValidateCompositionalDependencies(chainCases); err != nil {
				return nil, fmt.Errorf("invalid benchmark dependency graph for %s: %w", phase, err)
			}
		}
	}

	summaryPath := filepath.ToSlash(filepath.Join(loaded.ResolvePath(benchmark.SummaryDir), fmt.Sprintf("%s_%s.csv", phase, runID)))
	reportPath := filepath.ToSlash(filepath.Join(loaded.ResolvePath(benchmark.ReportDir), fmt.Sprintf("%s_%s.md", phase, runID)))
	manifestPath := filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("%s_%s.json", phase, runID)))
	modelCatalogPath := strings.TrimSpace(benchmark.ModelCatalog)
	if modelCatalogPath == "" {
		modelCatalogPath = filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ManifestRoot), fmt.Sprintf("%s_%s_model_catalog.csv", phase, runID)))
	} else {
		modelCatalogPath = filepath.ToSlash(loaded.ResolvePath(modelCatalogPath))
	}

	jobs := make([]BenchmarkJob, 0, len(benchmark.Families)*benchmark.Repeats)
	experimentDate := time.Now()
	for _, familyName := range benchmark.Families {
		family := loaded.Config.Families[familyName]
		transport, err := loaded.ResolveTransport(family.Transport, opts.RequireSecrets)
		if err != nil {
			return nil, err
		}
		requestedSampling := researchsampling.FromConfig(family.Sampling)
		resolvedSampling := researchsampling.Resolve(transport.Provider, requestedSampling)
		experimentName := researchsampling.ResolveExperimentName(family.ExperimentName, resolvedSampling.RequestedProfile, resolvedSampling.EffectiveProfile)
		for _, model := range family.Models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			for repeat := 1; repeat <= benchmark.Repeats; repeat++ {
				jobKey := sanitizeToken(fmt.Sprintf("%s_%s_r%02d", familyName, model, repeat))
				jobRunID := fmt.Sprintf("%s_%s", runID, jobKey)
				experimentID := researchsampling.BuildExperimentID(experimentName, repeat, experimentDate, resolvedSampling.RequestedProfile)
				resultsPath := filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ArtifactRoot), benchmark.ProjectFolder, "result", fmt.Sprintf("result_%s.csv", jobRunID)))
				rawDir := filepath.ToSlash(filepath.Join(loaded.ResolvePath(loaded.Config.Paths.ArtifactRoot), benchmark.ProjectFolder, "raw"))
				job := BenchmarkJob{
					Key:                          jobKey,
					Command:                      opts.Command,
					Phase:                        phase,
					RunID:                        jobRunID,
					Family:                       familyName,
					Model:                        model,
					Repeat:                       repeat,
					Selector:                     benchmark.InputFile,
					Provider:                     transport.Provider,
					BaseURL:                      transport.BaseURL,
					APIKeyEnv:                    transport.APIKeyEnv,
					Timeout:                      transport.Timeout.String(),
					ProjectFolder:                benchmark.ProjectFolder,
					InputKind:                    benchmark.InputKind,
					InputFile:                    benchmark.InputFile,
					InputPath:                    inputPath,
					PromptVersion:                benchmark.PromptVersion,
					ExperimentName:               experimentName,
					ExperimentID:                 experimentID,
					SamplingSurface:              resolvedSampling.Surface,
					RequestedSampling:            resolvedSampling.Requested,
					EffectiveSampling:            resolvedSampling.Effective,
					UnsupportedSampling:          resolvedSampling.Unsupported,
					RequestedSamplingProfile:     resolvedSampling.RequestedProfile,
					EffectiveSamplingProfile:     resolvedSampling.EffectiveProfile,
					RequestedTemperature:         cloneFloat64Ptr(family.Sampling, func(s *researchconfig.SamplingConfig) *float64 { return s.Temperature }),
					RequestedSeed:                cloneInt64Ptr(family.Sampling, func(s *researchconfig.SamplingConfig) *int64 { return s.Seed }),
					RequestedTopP:                cloneFloat64Ptr(family.Sampling, func(s *researchconfig.SamplingConfig) *float64 { return s.TopP }),
					ResultsPath:                  resultsPath,
					SummaryPath:                  summaryPath,
					RawDir:                       rawDir,
					RequestTimeoutAbortThreshold: requestTimeoutAbortThreshold,
					InterruptPolicy:              interruptPolicy,
				}
				job.SpecHash = fingerprintJob(job)
				jobs = append(jobs, job)
			}
		}
	}
	if len(jobs) == 0 {
		return nil, fmt.Errorf("research benchmark %q produced no jobs", phase)
	}

	return &BenchmarkPlan{
		SchemaVersion:                ManifestSchemaVersion,
		Command:                      opts.Command,
		Phase:                        phase,
		RunID:                        runID,
		CreatedAtUTC:                 time.Now().UTC().Format(time.RFC3339),
		ConfigFingerprint:            loaded.Fingerprint,
		ProjectFolder:                benchmark.ProjectFolder,
		InputKind:                    benchmark.InputKind,
		InputFile:                    benchmark.InputFile,
		InputPath:                    inputPath,
		PromptVersion:                benchmark.PromptVersion,
		SummaryPath:                  summaryPath,
		ReportPath:                   reportPath,
		ModelCatalogPath:             modelCatalogPath,
		ManifestPath:                 manifestPath,
		RepeatCount:                  benchmark.Repeats,
		RequestTimeoutAbortThreshold: requestTimeoutAbortThreshold,
		InterruptPolicy:              interruptPolicy,
		Jobs:                         jobs,
	}, nil
}

func benchmarkPhaseUsesChainDependencyValidation(phase string) bool {
	switch strings.TrimSpace(phase) {
	case "bridge-import-final-research",
		"bridge-only-authoring",
		"gold-final-composition-only",
		"compositional-assumption-import",
		"compositional-depth-ladder",
		"compositional-branching",
		"compositional-mixed-family-reuse",
		"compositional-mixed-family-gold-first",
		"compositional-mixed-family-gold-first-phase2",
		"compositional-mixed-family-gold-first-hard",
		"compositional-mixed-family-semi-gold-stable",
		"compositional-mixed-family-semi-gold-frontier":
		return true
	default:
		return false
	}
}

func (p *BenchmarkPlan) MarshalIndentedJSON() ([]byte, error) {
	type plain BenchmarkPlan
	return json.MarshalIndent((*plain)(p), "", "  ")
}

func fingerprintJob(job BenchmarkJob) string {
	raw, _ := json.Marshal(job)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func sanitizeToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "job"
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "job"
	}
	return result
}

func cloneFloat64Ptr[T any](source *T, selector func(*T) *float64) *float64 {
	if source == nil {
		return nil
	}
	value := selector(source)
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Ptr[T any](source *T, selector func(*T) *int64) *int64 {
	if source == nil {
		return nil
	}
	value := selector(source)
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
