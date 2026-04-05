package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const SchemaVersion = "researchctl.config/v1"

type Config struct {
	SchemaVersion string               `yaml:"schema_version" json:"schema_version"`
	EnvFile       string               `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	Paths         PathsConfig          `yaml:"paths" json:"paths"`
	State         StateConfig          `yaml:"state" json:"state"`
	Transports    map[string]Transport `yaml:"transports" json:"transports"`
	Families      map[string]Family    `yaml:"families" json:"families"`
	Benchmarks    BenchmarkSuite       `yaml:"benchmarks" json:"benchmarks"`
	Generate      GenerationSuite      `yaml:"generate" json:"generate"`
	Reports       ReportsConfig        `yaml:"reports" json:"reports"`
	DB            DBConfig             `yaml:"db" json:"db"`
}

type PathsConfig struct {
	ArtifactRoot string `yaml:"artifact_root" json:"artifact_root"`
	ReportRoot   string `yaml:"report_root" json:"report_root"`
	TempRoot     string `yaml:"temp_root" json:"temp_root"`
	ManifestRoot string `yaml:"manifest_root" json:"manifest_root"`
}

type StateConfig struct {
	DBPath string `yaml:"db_path" json:"db_path"`
}

type Transport struct {
	Provider  string `yaml:"provider" json:"provider"`
	BaseURL   string `yaml:"base_url" json:"base_url"`
	APIKeyEnv string `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"`
	Timeout   string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type SamplingConfig struct {
	Temperature *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	Seed        *int64   `yaml:"seed,omitempty" json:"seed,omitempty"`
	TopP        *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty"`
}

type Family struct {
	Transport      string          `yaml:"transport" json:"transport"`
	Models         []string        `yaml:"models" json:"models"`
	Enabled        *bool           `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ExperimentName string          `yaml:"experiment_name,omitempty" json:"experiment_name,omitempty"`
	Sampling       *SamplingConfig `yaml:"sampling,omitempty" json:"sampling,omitempty"`
}

type BenchmarkSuite struct {
	Phase1                                   Benchmark `yaml:"phase1" json:"phase1"`
	ND                                       Benchmark `yaml:"nd" json:"nd"`
	CompositionalAssumptionImport            Benchmark `yaml:"compositional_assumption_import" json:"compositional_assumption_import"`
	BridgeImportFinalResearch                Benchmark `yaml:"bridge_import_final_research" json:"bridge_import_final_research"`
	BridgeOnlyAuthoring                      Benchmark `yaml:"bridge_only_authoring" json:"bridge_only_authoring"`
	GoldFinalCompositionOnly                 Benchmark `yaml:"gold_final_composition_only" json:"gold_final_composition_only"`
	CompositionalDepthLadder                 Benchmark `yaml:"compositional_depth_ladder" json:"compositional_depth_ladder"`
	CompositionalBranching                   Benchmark `yaml:"compositional_branching" json:"compositional_branching"`
	CompositionalMixedFamilyReuse            Benchmark `yaml:"compositional_mixed_family_reuse" json:"compositional_mixed_family_reuse"`
	CompositionalMixedFamilyGoldFirst        Benchmark `yaml:"compositional_mixed_family_gold_first" json:"compositional_mixed_family_gold_first"`
	CompositionalMixedFamilyGoldFirstPhase2  Benchmark `yaml:"compositional_mixed_family_gold_first_phase2" json:"compositional_mixed_family_gold_first_phase2"`
	CompositionalMixedFamilyGoldFirstHard    Benchmark `yaml:"compositional_mixed_family_gold_first_hard" json:"compositional_mixed_family_gold_first_hard"`
	CompositionalMixedFamilySemiGoldStable   Benchmark `yaml:"compositional_mixed_family_semi_gold_stable" json:"compositional_mixed_family_semi_gold_stable"`
	CompositionalMixedFamilySemiGoldFrontier Benchmark `yaml:"compositional_mixed_family_semi_gold_frontier" json:"compositional_mixed_family_semi_gold_frontier"`
}

type Benchmark struct {
	ProjectFolder                string   `yaml:"project_folder" json:"project_folder"`
	InputKind                    string   `yaml:"input_kind" json:"input_kind"`
	InputFile                    string   `yaml:"input_file" json:"input_file"`
	InputPath                    string   `yaml:"input_path,omitempty" json:"input_path,omitempty"`
	PromptVersion                string   `yaml:"prompt_version" json:"prompt_version"`
	Families                     []string `yaml:"families" json:"families"`
	ParallelJobs                 int      `yaml:"jobs,omitempty" json:"jobs,omitempty"`
	Repeats                      int      `yaml:"repeats" json:"repeats"`
	RequestTimeoutAbortThreshold int      `yaml:"request_timeout_abort_threshold,omitempty" json:"request_timeout_abort_threshold,omitempty"`
	InterruptPolicy              string   `yaml:"interrupt_policy,omitempty" json:"interrupt_policy,omitempty"`
	SummaryDir                   string   `yaml:"summary_dir" json:"summary_dir"`
	ReportDir                    string   `yaml:"report_dir" json:"report_dir"`
	ModelCatalog                 string   `yaml:"model_catalog,omitempty" json:"model_catalog,omitempty"`
}

type GenerationSuite struct {
	Lean4 Lean4Config `yaml:"lean4" json:"lean4"`
}

type Lean4Config struct {
	ProjectFolder          string `yaml:"project_folder" json:"project_folder"`
	BenchmarkProjectFolder string `yaml:"benchmark_project_folder" json:"benchmark_project_folder"`
	WorkspaceRoot          string `yaml:"workspace_root" json:"workspace_root"`
	LakeBinary             string `yaml:"lake_binary" json:"lake_binary"`
	Timeout                string `yaml:"timeout" json:"timeout"`
	ExportOutputFile       string `yaml:"export_output_file" json:"export_output_file"`
	JobName                string `yaml:"job_name,omitempty" json:"job_name,omitempty"`
	JobStatement           string `yaml:"job_statement,omitempty" json:"job_statement,omitempty"`
	JobPayloadFile         string `yaml:"job_payload_file,omitempty" json:"job_payload_file,omitempty"`
}

type ReportsConfig struct {
	BenchmarkDir string `yaml:"benchmark_dir" json:"benchmark_dir"`
	NDDir        string `yaml:"nd_dir" json:"nd_dir"`
	ResearchDir  string `yaml:"research_dir" json:"research_dir"`
	MetaDir      string `yaml:"meta_dir" json:"meta_dir"`
	CasesDir     string `yaml:"cases_dir" json:"cases_dir"`
}

type DBConfig struct {
	ProjectFolder string `yaml:"project_folder" json:"project_folder"`
}

type Loaded struct {
	WorkspaceRoot    string
	ConfigPath       string
	LocalConfigPath  string
	LocalConfigFound bool
	Config           Config
	MergedYAML       []byte
	ResolvedJSON     []byte
	Fingerprint      string
}

type ResolvedTransport struct {
	Name      string
	Provider  string
	BaseURL   string
	APIKeyEnv string
	APIKey    string
	Timeout   time.Duration
}

func Load(workspaceRoot, configPath, localConfigPath string) (Loaded, error) {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return Loaded{}, fmt.Errorf("workspace root is required")
	}
	configPath = resolvePath(workspaceRoot, firstNonEmpty(configPath, filepath.Join("research", "config", "default.yaml")))
	localConfigPath = resolvePath(workspaceRoot, firstNonEmpty(localConfigPath, filepath.Join("research", "config", "workflow", "default-local.yaml")))

	defaultMap, err := loadYAMLMap(configPath)
	if err != nil {
		return Loaded{}, err
	}
	localFound := fileExists(localConfigPath)
	if localFound {
		localMap, err := loadYAMLMap(localConfigPath)
		if err != nil {
			return Loaded{}, err
		}
		defaultMap = deepMergeMaps(defaultMap, localMap)
	}

	mergedYAML, err := yaml.Marshal(defaultMap)
	if err != nil {
		return Loaded{}, fmt.Errorf("marshal merged research config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(mergedYAML, &cfg); err != nil {
		return Loaded{}, fmt.Errorf("decode merged research config: %w", err)
	}
	applyDefaults(&cfg)
	if cfg.EnvFile != "" {
		if err := loadEnvFile(resolvePath(workspaceRoot, cfg.EnvFile)); err != nil {
			return Loaded{}, err
		}
	}
	if err := validateConfig(cfg); err != nil {
		return Loaded{}, err
	}

	resolvedJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Loaded{}, fmt.Errorf("marshal resolved research config json: %w", err)
	}

	sum := sha256.Sum256(resolvedJSON)
	return Loaded{
		WorkspaceRoot:    workspaceRoot,
		ConfigPath:       configPath,
		LocalConfigPath:  localConfigPath,
		LocalConfigFound: localFound,
		Config:           cfg,
		MergedYAML:       mergedYAML,
		ResolvedJSON:     resolvedJSON,
		Fingerprint:      hex.EncodeToString(sum[:]),
	}, nil
}

func (l Loaded) ResolvePath(path string) string {
	return resolvePath(l.WorkspaceRoot, path)
}

func (l Loaded) RelativePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	rel, err := filepath.Rel(l.WorkspaceRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func (l Loaded) ResolveTransport(name string, requireSecret bool) (ResolvedTransport, error) {
	transport, ok := l.Config.Transports[name]
	if !ok {
		return ResolvedTransport{}, fmt.Errorf("research transport %q is not defined", name)
	}
	provider := strings.ToLower(strings.TrimSpace(firstNonEmpty(transport.Provider, "compatible")))
	switch provider {
	case "compatible", "openai", "google", "mistral":
	default:
		return ResolvedTransport{}, fmt.Errorf("research transport %q uses unsupported provider %q", name, provider)
	}
	timeoutText := firstNonEmpty(strings.TrimSpace(transport.Timeout), "60s")
	timeout, err := time.ParseDuration(timeoutText)
	if err != nil {
		return ResolvedTransport{}, fmt.Errorf("research transport %q has invalid timeout %q: %w", name, transport.Timeout, err)
	}
	baseURL := strings.TrimSpace(transport.BaseURL)
	if baseURL == "" {
		return ResolvedTransport{}, fmt.Errorf("research transport %q requires base_url", name)
	}
	apiKeyEnv := strings.TrimSpace(transport.APIKeyEnv)
	apiKey := ""
	if apiKeyEnv != "" {
		apiKey = strings.TrimSpace(os.Getenv(apiKeyEnv))
	}
	if requireSecret && provider != "compatible" && apiKey == "" {
		if apiKeyEnv == "" {
			return ResolvedTransport{}, fmt.Errorf("research transport %q requires api_key_env for provider=%s", name, provider)
		}
		return ResolvedTransport{}, fmt.Errorf("required API key env %q is empty for research transport %q", apiKeyEnv, name)
	}
	return ResolvedTransport{
		Name:      name,
		Provider:  provider,
		BaseURL:   baseURL,
		APIKeyEnv: apiKeyEnv,
		APIKey:    apiKey,
		Timeout:   timeout,
	}, nil
}

func resolvePath(workspaceRoot, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(workspaceRoot, filepath.FromSlash(path))
}

func applyDefaults(cfg *Config) {
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = SchemaVersion
	}
	cfg.Paths.ArtifactRoot = firstNonEmpty(cfg.Paths.ArtifactRoot, "research/artifacts")
	cfg.Paths.ReportRoot = firstNonEmpty(cfg.Paths.ReportRoot, "research/result_research_report_v1")
	cfg.Paths.TempRoot = firstNonEmpty(cfg.Paths.TempRoot, ".tmp/research")
	cfg.Paths.ManifestRoot = firstNonEmpty(cfg.Paths.ManifestRoot, "research/artifacts/result_research/manifests")
	cfg.State.DBPath = firstNonEmpty(cfg.State.DBPath, "research/artifacts/result_research/research_db/research_db.sqlite")

	if cfg.Transports == nil {
		cfg.Transports = map[string]Transport{}
	}
	if cfg.Families == nil {
		cfg.Families = map[string]Family{}
	}

	applyBenchmarkDefaults(&cfg.Benchmarks.Phase1, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: DefaultInterruptPolicy,
		SummaryDir:      "research/artifacts/result_research/waves",
		ReportDir:       "research/result_research_report_v1/direct",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.ND, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-nd-v1",
		InputKind:       "theorems",
		InputFile:       "theorems/pilot_shared_20260327.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-nd-v1.1",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: DefaultInterruptPolicy,
		SummaryDir:      "research/artifacts/result_research/nd",
		ReportDir:       "research/result_research_report_v1/nd",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalAssumptionImport, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-assumption-import-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: DefaultInterruptPolicy,
		SummaryDir:      "research/artifacts/result_research/compositional",
		ReportDir:       "research/result_research_report_v1/compositional/assumption-import",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.BridgeImportFinalResearch, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/bridge-import-final-research-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/bridge-import-final-research",
		ReportDir:       "research/result_research_report_v1/compositional/bridge-import-final-research",
	})
	if len(cfg.Benchmarks.BridgeImportFinalResearch.Families) == 0 {
		if len(cfg.Benchmarks.CompositionalAssumptionImport.Families) > 0 {
			cfg.Benchmarks.BridgeImportFinalResearch.Families = append([]string(nil), cfg.Benchmarks.CompositionalAssumptionImport.Families...)
		} else if len(cfg.Benchmarks.Phase1.Families) > 0 {
			cfg.Benchmarks.BridgeImportFinalResearch.Families = append([]string(nil), cfg.Benchmarks.Phase1.Families...)
		} else if len(cfg.Benchmarks.ND.Families) > 0 {
			cfg.Benchmarks.BridgeImportFinalResearch.Families = append([]string(nil), cfg.Benchmarks.ND.Families...)
		} else {
			cfg.Benchmarks.BridgeImportFinalResearch.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.BridgeOnlyAuthoring, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/bridge-only-authoring-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-bridge-only-v1.0",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/bridge-only-authoring",
		ReportDir:       "research/result_research_report_v1/compositional/bridge-only-authoring",
	})
	if len(cfg.Benchmarks.BridgeOnlyAuthoring.Families) == 0 {
		if len(cfg.Benchmarks.BridgeImportFinalResearch.Families) > 0 {
			cfg.Benchmarks.BridgeOnlyAuthoring.Families = append([]string(nil), cfg.Benchmarks.BridgeImportFinalResearch.Families...)
		} else if len(cfg.Benchmarks.CompositionalAssumptionImport.Families) > 0 {
			cfg.Benchmarks.BridgeOnlyAuthoring.Families = append([]string(nil), cfg.Benchmarks.CompositionalAssumptionImport.Families...)
		} else if len(cfg.Benchmarks.Phase1.Families) > 0 {
			cfg.Benchmarks.BridgeOnlyAuthoring.Families = append([]string(nil), cfg.Benchmarks.Phase1.Families...)
		} else {
			cfg.Benchmarks.BridgeOnlyAuthoring.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.GoldFinalCompositionOnly, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/gold-final-composition-only-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-gold-final-only-v1.0",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/gold-final-composition-only",
		ReportDir:       "research/result_research_report_v1/compositional/gold-final-composition-only",
	})
	if len(cfg.Benchmarks.GoldFinalCompositionOnly.Families) == 0 {
		if len(cfg.Benchmarks.BridgeImportFinalResearch.Families) > 0 {
			cfg.Benchmarks.GoldFinalCompositionOnly.Families = append([]string(nil), cfg.Benchmarks.BridgeImportFinalResearch.Families...)
		} else if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families) > 0 {
			cfg.Benchmarks.GoldFinalCompositionOnly.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families...)
		} else if len(cfg.Benchmarks.Phase1.Families) > 0 {
			cfg.Benchmarks.GoldFinalCompositionOnly.Families = append([]string(nil), cfg.Benchmarks.Phase1.Families...)
		} else {
			cfg.Benchmarks.GoldFinalCompositionOnly.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalDepthLadder, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-depth-ladder-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-depth-ladder",
		ReportDir:       "research/result_research_report_v1/compositional/depth-ladder",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalBranching, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-branching-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-branching",
		ReportDir:       "research/result_research_report_v1/compositional/branching",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilyReuse, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-reuse-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-reuse",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-reuse",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilyGoldFirst, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-gold-first-phase1.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		Families:        []string{"local-compatible"},
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-gold-first",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-gold-first",
	})
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-gold-first-phase2.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-gold-first-phase2",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-gold-first-phase2",
	})
	if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families) == 0 {
		if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families...)
		} else {
			cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-gold-first-hard.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-gold-first-hard",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-gold-first-hard",
	})
	if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families) == 0 {
		if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families...)
		} else if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirst.Families...)
		} else {
			cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-semi-gold-stable.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-semi-gold-stable",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-semi-gold-stable",
	})
	if len(cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families) == 0 {
		if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families...)
		} else if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2.Families...)
		} else {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families = []string{"local-compatible"}
		}
	}
	applyBenchmarkDefaults(&cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier, Benchmark{
		ProjectFolder:   "hilbert-ai-verification-benchmark-v2-held-out",
		InputKind:       "cases",
		InputFile:       "cases/compositional-mixed-family-semi-gold-frontier.csv",
		PromptVersion:   "hilbert-ai-verification-benchmark-v1.3",
		ParallelJobs:    1,
		Repeats:         1,
		InterruptPolicy: InterruptPolicyDropIfNoResults,
		SummaryDir:      "research/artifacts/result_research/compositional-mixed-family-semi-gold-frontier",
		ReportDir:       "research/result_research_report_v1/compositional/mixed-family-semi-gold-frontier",
	})
	if len(cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.Families) == 0 {
		if len(cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilySemiGoldStable.Families...)
		} else if len(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families) > 0 {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.Families = append([]string(nil), cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard.Families...)
		} else {
			cfg.Benchmarks.CompositionalMixedFamilySemiGoldFrontier.Families = []string{"local-compatible"}
		}
	}

	cfg.Generate.Lean4.ProjectFolder = firstNonEmpty(cfg.Generate.Lean4.ProjectFolder, "hilbert-ai-verification-lean4-worker-v1")
	cfg.Generate.Lean4.BenchmarkProjectFolder = firstNonEmpty(cfg.Generate.Lean4.BenchmarkProjectFolder, "hilbert-ai-verification-benchmark-nd-v1")
	cfg.Generate.Lean4.WorkspaceRoot = firstNonEmpty(cfg.Generate.Lean4.WorkspaceRoot, ".tmp/research/lean4")
	cfg.Generate.Lean4.LakeBinary = firstNonEmpty(cfg.Generate.Lean4.LakeBinary, "lake")
	cfg.Generate.Lean4.Timeout = firstNonEmpty(cfg.Generate.Lean4.Timeout, "60s")
	cfg.Generate.Lean4.ExportOutputFile = firstNonEmpty(cfg.Generate.Lean4.ExportOutputFile, "theorems/lean4-generated.csv")
	cfg.Generate.Lean4.JobName = firstNonEmpty(cfg.Generate.Lean4.JobName, "identity_proved")
	cfg.Generate.Lean4.JobStatement = firstNonEmpty(cfg.Generate.Lean4.JobStatement, "P -> P")
	cfg.Generate.Lean4.JobPayloadFile = firstNonEmpty(cfg.Generate.Lean4.JobPayloadFile, filepath.ToSlash(filepath.Join("research", "artifacts", cfg.Generate.Lean4.ProjectFolder, "theorems", "payloads", "identity-proof.json")))

	cfg.Reports.BenchmarkDir = firstNonEmpty(cfg.Reports.BenchmarkDir, "research/result_research_report_v1/direct")
	cfg.Reports.NDDir = firstNonEmpty(cfg.Reports.NDDir, "research/result_research_report_v1/nd")
	cfg.Reports.ResearchDir = firstNonEmpty(cfg.Reports.ResearchDir, "research/result_research_report_v1/pair")
	cfg.Reports.MetaDir = firstNonEmpty(cfg.Reports.MetaDir, "research/result_research_report_v1/summary/meta")
	cfg.Reports.CasesDir = firstNonEmpty(cfg.Reports.CasesDir, "research/result_research_report_v1/summary/cases")
	cfg.DB.ProjectFolder = firstNonEmpty(cfg.DB.ProjectFolder, "hilbert-ai-verification-benchmark-nd-v1")
}

func applyBenchmarkDefaults(target *Benchmark, defaults Benchmark) {
	target.ProjectFolder = firstNonEmpty(target.ProjectFolder, defaults.ProjectFolder)
	target.InputKind = firstNonEmpty(target.InputKind, defaults.InputKind)
	target.InputFile = firstNonEmpty(target.InputFile, defaults.InputFile)
	target.PromptVersion = firstNonEmpty(target.PromptVersion, defaults.PromptVersion)
	if len(target.Families) == 0 {
		target.Families = append([]string(nil), defaults.Families...)
	}
	if target.ParallelJobs <= 0 {
		target.ParallelJobs = defaults.ParallelJobs
	}
	if target.Repeats <= 0 {
		target.Repeats = defaults.Repeats
	}
	if target.RequestTimeoutAbortThreshold < 0 {
		target.RequestTimeoutAbortThreshold = defaults.RequestTimeoutAbortThreshold
	}
	target.InterruptPolicy = NormalizeInterruptPolicy(firstNonEmpty(target.InterruptPolicy, defaults.InterruptPolicy))
	target.SummaryDir = firstNonEmpty(target.SummaryDir, defaults.SummaryDir)
	target.ReportDir = firstNonEmpty(target.ReportDir, defaults.ReportDir)
}

func validateConfig(cfg Config) error {
	if cfg.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported research config schema_version %q", cfg.SchemaVersion)
	}
	if strings.TrimSpace(cfg.Paths.ArtifactRoot) == "" || strings.TrimSpace(cfg.Paths.ReportRoot) == "" || strings.TrimSpace(cfg.Paths.TempRoot) == "" || strings.TrimSpace(cfg.Paths.ManifestRoot) == "" {
		return fmt.Errorf("research config paths.* must not be empty")
	}
	if strings.TrimSpace(cfg.State.DBPath) == "" {
		return fmt.Errorf("research config state.db_path must not be empty")
	}
	for name, transport := range cfg.Transports {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("research config transport name must not be blank")
		}
		provider := strings.ToLower(strings.TrimSpace(firstNonEmpty(transport.Provider, "compatible")))
		if provider != "compatible" && provider != "openai" && provider != "google" && provider != "mistral" {
			return fmt.Errorf("research transport %q uses unsupported provider %q", name, provider)
		}
		if strings.TrimSpace(transport.BaseURL) == "" {
			return fmt.Errorf("research transport %q requires base_url", name)
		}
		if provider != "compatible" && strings.TrimSpace(transport.APIKeyEnv) == "" {
			return fmt.Errorf("research transport %q requires api_key_env for provider=%s", name, provider)
		}
		if strings.TrimSpace(firstNonEmpty(transport.Timeout, "60s")) != "" {
			if _, err := time.ParseDuration(firstNonEmpty(transport.Timeout, "60s")); err != nil {
				return fmt.Errorf("research transport %q has invalid timeout %q: %w", name, transport.Timeout, err)
			}
		}
	}
	for name, family := range cfg.Families {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("research config family name must not be blank")
		}
		if family.Enabled != nil && !*family.Enabled {
			continue
		}
		if strings.TrimSpace(family.Transport) == "" {
			return fmt.Errorf("research family %q requires transport", name)
		}
		if _, ok := cfg.Transports[family.Transport]; !ok {
			return fmt.Errorf("research family %q references unknown transport %q", name, family.Transport)
		}
		if len(family.Models) == 0 {
			return fmt.Errorf("research family %q requires at least one model", name)
		}
		if strings.TrimSpace(family.ExperimentName) == "" && family.Sampling != nil {
			return fmt.Errorf("research family %q with sampling overrides requires experiment_name for honest provenance", name)
		}
		if err := validateSamplingConfig(family.Sampling, fmt.Sprintf("research family %q sampling", name)); err != nil {
			return err
		}
	}
	if err := validateBenchmark(cfg.Benchmarks.Phase1, "phase1", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.ND, "nd", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalAssumptionImport, "compositional_assumption_import", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.BridgeImportFinalResearch, "bridge_import_final_research", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.BridgeOnlyAuthoring, "bridge_only_authoring", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.GoldFinalCompositionOnly, "gold_final_composition_only", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalDepthLadder, "compositional_depth_ladder", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalBranching, "compositional_branching", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalMixedFamilyReuse, "compositional_mixed_family_reuse", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalMixedFamilyGoldFirst, "compositional_mixed_family_gold_first", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstPhase2, "compositional_mixed_family_gold_first_phase2", cfg.Families); err != nil {
		return err
	}
	if err := validateBenchmark(cfg.Benchmarks.CompositionalMixedFamilyGoldFirstHard, "compositional_mixed_family_gold_first_hard", cfg.Families); err != nil {
		return err
	}
	if _, err := time.ParseDuration(cfg.Generate.Lean4.Timeout); err != nil {
		return fmt.Errorf("research config generate.lean4.timeout %q is invalid: %w", cfg.Generate.Lean4.Timeout, err)
	}
	if strings.TrimSpace(cfg.Generate.Lean4.ProjectFolder) == "" || strings.TrimSpace(cfg.Generate.Lean4.BenchmarkProjectFolder) == "" || strings.TrimSpace(cfg.Generate.Lean4.WorkspaceRoot) == "" || strings.TrimSpace(cfg.Generate.Lean4.LakeBinary) == "" {
		return fmt.Errorf("research config generate.lean4.* must not be empty")
	}
	if strings.TrimSpace(cfg.Generate.Lean4.JobName) == "" || strings.TrimSpace(cfg.Generate.Lean4.JobStatement) == "" || strings.TrimSpace(cfg.Generate.Lean4.JobPayloadFile) == "" {
		return fmt.Errorf("research config generate.lean4.job_name, job_statement, and job_payload_file must not be empty")
	}
	if strings.TrimSpace(cfg.DB.ProjectFolder) == "" {
		return fmt.Errorf("research config db.project_folder must not be empty")
	}
	return nil
}

func validateBenchmark(benchmark Benchmark, name string, families map[string]Family) error {
	if strings.TrimSpace(benchmark.ProjectFolder) == "" {
		return fmt.Errorf("research benchmark %q requires project_folder", name)
	}
	inputKind := strings.TrimSpace(benchmark.InputKind)
	if inputKind != "cases" && inputKind != "theorems" {
		return fmt.Errorf("research benchmark %q input_kind must be one of cases|theorems", name)
	}
	if strings.TrimSpace(benchmark.InputFile) == "" && strings.TrimSpace(benchmark.InputPath) == "" {
		return fmt.Errorf("research benchmark %q requires input_file or input_path", name)
	}
	if strings.TrimSpace(benchmark.PromptVersion) == "" {
		return fmt.Errorf("research benchmark %q requires prompt_version", name)
	}
	if benchmark.Repeats <= 0 {
		return fmt.Errorf("research benchmark %q requires repeats > 0", name)
	}
	if benchmark.ParallelJobs <= 0 {
		return fmt.Errorf("research benchmark %q requires jobs > 0", name)
	}
	if benchmark.RequestTimeoutAbortThreshold < 0 {
		return fmt.Errorf("research benchmark %q request_timeout_abort_threshold must be >= 0", name)
	}
	if err := ValidateInterruptPolicy(benchmark.InterruptPolicy); err != nil {
		return fmt.Errorf("research benchmark %q %w", name, err)
	}
	if len(benchmark.Families) == 0 {
		return fmt.Errorf("research benchmark %q requires at least one family", name)
	}
	for _, familyName := range benchmark.Families {
		familyName = strings.TrimSpace(familyName)
		family, ok := families[familyName]
		if !ok {
			return fmt.Errorf("research benchmark %q references unknown family %q", name, familyName)
		}
		if family.Enabled != nil && !*family.Enabled {
			return fmt.Errorf("research benchmark %q references disabled family %q", name, familyName)
		}
	}
	if strings.TrimSpace(benchmark.SummaryDir) == "" || strings.TrimSpace(benchmark.ReportDir) == "" {
		return fmt.Errorf("research benchmark %q requires summary_dir and report_dir", name)
	}
	return nil
}

func validateSamplingConfig(sampling *SamplingConfig, scope string) error {
	if sampling == nil {
		return nil
	}
	if sampling.Temperature != nil && *sampling.Temperature < 0 {
		return fmt.Errorf("%s temperature must be >= 0", scope)
	}
	if sampling.Seed != nil && *sampling.Seed < 0 {
		return fmt.Errorf("%s seed must be >= 0", scope)
	}
	if sampling.TopP != nil {
		if *sampling.TopP <= 0 || *sampling.TopP > 1 {
			return fmt.Errorf("%s top_p must satisfy 0 < top_p <= 1", scope)
		}
	}
	return nil
}

func loadYAMLMap(path string) (map[string]any, error) {
	return loadYAMLMapRecursive(filepath.Clean(path), map[string]struct{}{})
}

func loadYAMLMapRecursive(path string, stack map[string]struct{}) (map[string]any, error) {
	if _, seen := stack[path]; seen {
		return nil, fmt.Errorf("research config extends cycle detected at %s", filepath.ToSlash(path))
	}
	nextStack := make(map[string]struct{}, len(stack)+1)
	for key := range stack {
		nextStack[key] = struct{}{}
	}
	nextStack[path] = struct{}{}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read research config %s: %w", path, err)
	}
	out := map[string]any{}
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode research config %s: %w", path, err)
	}
	extendsPaths, err := decodeExtendsPaths(path, out["extends"])
	if err != nil {
		return nil, err
	}
	delete(out, "extends")
	if len(extendsPaths) == 0 {
		return out, nil
	}
	merged := map[string]any{}
	for _, extendPath := range extendsPaths {
		basePath := resolveExtendsPath(path, extendPath)
		base, err := loadYAMLMapRecursive(basePath, nextStack)
		if err != nil {
			return nil, err
		}
		merged = deepMergeMaps(merged, base)
	}
	return deepMergeMaps(merged, out), nil
}

func decodeExtendsPaths(path string, raw any) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil
		}
		return []string{value}, nil
	case []any:
		paths := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("research config %s uses non-string extends entry", filepath.ToSlash(path))
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			paths = append(paths, text)
		}
		return paths, nil
	default:
		return nil, fmt.Errorf("research config %s uses invalid extends field", filepath.ToSlash(path))
	}
}

func resolveExtendsPath(path, extendPath string) string {
	if filepath.IsAbs(extendPath) {
		return filepath.Clean(extendPath)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), extendPath))
}

func deepMergeMaps(base, overlay map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		current, ok := out[key]
		if ok {
			left, leftOK := current.(map[string]any)
			right, rightOK := value.(map[string]any)
			if leftOK && rightOK {
				out[key] = deepMergeMaps(left, right)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func loadEnvFile(path string) error {
	if !fileExists(path) {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read research env_file %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid env_file line %q in %s", line, path)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("invalid env_file line %q in %s", line, path)
		}
		if _, present := os.LookupEnv(key); present {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env var %q from %s: %w", key, path, err)
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
