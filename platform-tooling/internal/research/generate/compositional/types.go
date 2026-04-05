package compositional

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SurfaceBridgeImportFinalResearch = "bridge-import-final-research"
	SurfaceBridgeOnlyAuthoring       = "bridge-only-authoring"
	SurfaceGoldFinalCompositionOnly  = "gold-final-composition-only"

	ModePhase1    = "phase1"
	ModePhase2    = "phase2"
	ModeCanonical = "canonical"

	DefaultProjectFolder             = "hilbert-ai-verification-benchmark-v2-held-out"
	DefaultTransportName             = "local-compatible"
	DefaultTransportProvider         = "compatible"
	DefaultBaseURL                   = "http://localhost:11434"
	DefaultInterruptPolicy           = "drop_if_no_results"
	defaultBridgeBenchmarkKey        = "bridge_only_authoring"
	defaultGoldBenchmarkKey          = "gold_final_composition_only"
	BridgeImportFinalPhase1CaseCount = 20
)

var defaultModels = []string{
	"gpt-oss:20b",
	"gpt-oss:120b-cloud",
	"glm-5:cloud",
	"deepseek-v3.1:671b-cloud",
}

var promptExperimentNames = map[string]string{
	"hilbert-ai-verification-bridge-only-v1.0":                          "bridge-only-authoring-v1",
	"hilbert-ai-verification-bridge-only-skeleton-v1.0":                 "bridge-only-authoring-skeleton-v1",
	"hilbert-ai-verification-bridge-only-canonical-vars-v1.0":           "bridge-only-authoring-canonical-v1",
	"hilbert-ai-verification-gold-final-only-v1.0":                      "gold-final-composition-only-v1",
	"hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0": "gold-final-composition-only-explicit-import-refs-v1",
	"hilbert-ai-verification-benchmark-v1.3":                            "bridge-import-final-research-v1",
}

var promptLocalConfigPaths = map[string]string{
	"hilbert-ai-verification-bridge-only-skeleton-v1.0":                 "research/config/compositional/bridge/only-authoring.skeleton.yaml",
	"hilbert-ai-verification-bridge-only-canonical-vars-v1.0":           "research/config/compositional/bridge/only-authoring.canonical.yaml",
	"hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0": "research/config/compositional/gold/final-composition-only.explicit-import-refs.yaml",
}

var bridgeOnlyCSVHeader = []string{
	"case_id",
	"chain_id",
	"chain_protocol",
	"stage_id",
	"stage_order",
	"stage_role",
	"category",
	"label",
	"difficulty",
	"case_family",
	"chain_depth",
	"provenance_mode",
	"atom_renaming_id",
	"trusted_reuse",
	"transition_group",
	"import_arity",
	"reuse_shape",
	"bridge_depth",
	"symbol_overlap",
	"lemma_surface_style",
	"notes",
	"import_stage_ids_json",
	"assumptions_json",
	"imported_lemmas_json",
	"goal",
	"expected_behavior",
	"comment",
}

var goldFinalCSVHeader = []string{
	"case_id",
	"chain_id",
	"chain_protocol",
	"stage_id",
	"stage_order",
	"stage_role",
	"category",
	"label",
	"difficulty",
	"case_family",
	"chain_depth",
	"provenance_mode",
	"atom_renaming_id",
	"trusted_reuse",
	"transition_group",
	"import_arity",
	"reuse_shape",
	"bridge_depth",
	"symbol_overlap",
	"lemma_surface_style",
	"negative_twin_hardness",
	"notes",
	"import_stage_ids_json",
	"assumptions_json",
	"imported_lemmas_json",
	"goal",
	"expected_behavior",
	"comment",
}

var bridgeFinalCSVHeader = []string{
	"case_id",
	"chain_id",
	"chain_protocol",
	"stage_id",
	"stage_order",
	"stage_role",
	"category",
	"label",
	"difficulty",
	"case_family",
	"chain_depth",
	"provenance_mode",
	"atom_renaming_id",
	"trusted_reuse",
	"import_stage_ids_json",
	"assumptions_json",
	"imported_lemmas_json",
	"goal",
	"expected_behavior",
	"comment",
}

type CaseRow struct {
	CaseID               string
	ChainID              string
	ChainProtocol        string
	StageID              string
	StageOrder           int
	StageRole            string
	Category             string
	Label                string
	Difficulty           string
	CaseFamily           string
	ChainDepth           int
	ProvenanceMode       string
	AtomRenamingID       string
	TrustedReuse         *bool
	TransitionGroup      string
	ImportArity          string
	ReuseShape           string
	BridgeDepth          string
	SymbolOverlap        string
	LemmaSurfaceStyle    string
	NegativeTwinHardness string
	Notes                string
	ImportStageIDs       []string
	Assumptions          []string
	ImportedLemmas       []string
	Goal                 string
	ExpectedBehavior     string
	Comment              string
}

type SurfaceProfile struct {
	Surface                             string
	Mode                                string
	BenchmarkKey                        string
	ProjectFolder                       string
	DefaultOutputFile                   string
	DefaultLocalConfigPath              string
	DefaultPromptVersion                string
	DefaultExperimentName               string
	DefaultFamilyName                   string
	DefaultRepeats                      int
	DefaultRequestTimeoutAbortThreshold int
	DefaultInterruptPolicy              string
	SummaryDir                          string
	ReportDir                           string
	CSVHeader                           []string
	generateRows                        func() []CaseRow
}

type GenerationOptions struct {
	Surface                      string
	Mode                         string
	CaseCount                    int
	OutputPath                   string
	BenchmarkInputFile           string
	LocalConfigPath              string
	EmitLocalConfig              bool
	Overwrite                    bool
	TransportName                string
	TransportProvider            string
	BaseURL                      string
	APIKeyEnv                    string
	FamilyName                   string
	ExperimentName               string
	PromptVersion                string
	InterruptPolicy              string
	Models                       []string
	Repeats                      int
	RequestTimeoutAbortThreshold int
}

type Result struct {
	Profile                      SurfaceProfile
	OutputPath                   string
	BenchmarkInputFile           string
	LocalConfigPath              string
	PromptVersion                string
	ExperimentName               string
	FamilyName                   string
	TransportName                string
	TransportProvider            string
	BaseURL                      string
	APIKeyEnv                    string
	Models                       []string
	Repeats                      int
	RequestTimeoutAbortThreshold int
	InterruptPolicy              string
	CaseCount                    int
	ChainCount                   int
}

type localConfigFile struct {
	Transports map[string]localTransport `yaml:"transports,omitempty"`
	Families   map[string]localFamily    `yaml:"families,omitempty"`
	Benchmarks map[string]localBenchmark `yaml:"benchmarks,omitempty"`
}

type localTransport struct {
	Provider  string `yaml:"provider,omitempty"`
	BaseURL   string `yaml:"base_url,omitempty"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

type localFamily struct {
	Transport      string         `yaml:"transport,omitempty"`
	ExperimentName string         `yaml:"experiment_name,omitempty"`
	Models         []string       `yaml:"models,omitempty"`
	Sampling       *localSampling `yaml:"sampling,omitempty"`
}

type localSampling struct {
	Temperature int `yaml:"temperature,omitempty"`
	Seed        int `yaml:"seed,omitempty"`
	TopP        int `yaml:"top_p,omitempty"`
}

type localBenchmark struct {
	InputFile                    string   `yaml:"input_file,omitempty"`
	PromptVersion                string   `yaml:"prompt_version,omitempty"`
	Families                     []string `yaml:"families,omitempty"`
	Repeats                      int      `yaml:"repeats,omitempty"`
	RequestTimeoutAbortThreshold int      `yaml:"request_timeout_abort_threshold,omitempty"`
	InterruptPolicy              string   `yaml:"interrupt_policy,omitempty"`
	SummaryDir                   string   `yaml:"summary_dir,omitempty"`
	ReportDir                    string   `yaml:"report_dir,omitempty"`
}

func SupportedSurfaces() []string {
	return []string{
		SurfaceBridgeImportFinalResearch,
		SurfaceBridgeOnlyAuthoring,
		SurfaceGoldFinalCompositionOnly,
	}
}

func SupportedModes(surface string) []string {
	switch normalizeSurface(surface) {
	case SurfaceBridgeImportFinalResearch:
		return []string{ModePhase1}
	case SurfaceBridgeOnlyAuthoring:
		return []string{ModePhase1, ModeCanonical, ModePhase2}
	case SurfaceGoldFinalCompositionOnly:
		return []string{ModePhase1, ModePhase2}
	default:
		return nil
	}
}

func ResolveProfile(surface, mode string) (SurfaceProfile, error) {
	surface = normalizeSurface(surface)
	mode = normalizeMode(mode)
	if mode == "" {
		mode = defaultModeForSurface(surface)
	}
	profile, ok := compositionalProfiles()[profileKey(surface, mode)]
	if !ok {
		return SurfaceProfile{}, fmt.Errorf("unsupported compositional generator target surface=%q mode=%q", surface, mode)
	}
	return profile, nil
}

// Generate emits benchmark-compatible CSV rows and, optionally, a matching local
// YAML override so operators can go straight from case synthesis to plan/run.
func Generate(opts GenerationOptions) (Result, error) {
	profile, err := ResolveProfile(opts.Surface, opts.Mode)
	if err != nil {
		return Result{}, err
	}
	outputPath := strings.TrimSpace(opts.OutputPath)
	if outputPath == "" {
		return Result{}, fmt.Errorf("compositional cases output path is required")
	}
	benchmarkInputFile := strings.TrimSpace(opts.BenchmarkInputFile)
	if benchmarkInputFile == "" {
		benchmarkInputFile = profile.DefaultOutputFile
	}
	if err := ensureWritableTarget(outputPath, opts.Overwrite); err != nil {
		return Result{}, err
	}
	promptVersion := firstNonEmpty(strings.TrimSpace(opts.PromptVersion), profile.DefaultPromptVersion)
	experimentName := resolveExperimentName(profile, promptVersion, strings.TrimSpace(opts.ExperimentName))
	familyName := firstNonEmpty(strings.TrimSpace(opts.FamilyName), profile.DefaultFamilyName)
	transportName := firstNonEmpty(strings.TrimSpace(opts.TransportName), DefaultTransportName)
	transportProvider := firstNonEmpty(strings.TrimSpace(opts.TransportProvider), DefaultTransportProvider)
	baseURL := firstNonEmpty(strings.TrimSpace(opts.BaseURL), DefaultBaseURL)
	models := append([]string(nil), opts.Models...)
	if len(models) == 0 {
		models = append(models, defaultModels...)
	}
	repeats := opts.Repeats
	if repeats <= 0 {
		repeats = profile.DefaultRepeats
	}
	requestTimeoutAbortThreshold := opts.RequestTimeoutAbortThreshold
	if requestTimeoutAbortThreshold <= 0 {
		requestTimeoutAbortThreshold = profile.DefaultRequestTimeoutAbortThreshold
	}
	interruptPolicy := firstNonEmpty(strings.TrimSpace(opts.InterruptPolicy), profile.DefaultInterruptPolicy)
	rows, err := generateRowsForProfile(profile, opts.CaseCount)
	if err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		return Result{}, fmt.Errorf("compositional generator profile %q/%q produced no rows", profile.Surface, profile.Mode)
	}
	if err := writeCasesCSV(outputPath, profile.CSVHeader, rows); err != nil {
		return Result{}, err
	}
	localConfigPath := strings.TrimSpace(opts.LocalConfigPath)
	if opts.EmitLocalConfig {
		if localConfigPath == "" {
			localConfigPath = DefaultLocalConfigPath(profile, promptVersion)
		}
		if err := ensureWritableTarget(localConfigPath, opts.Overwrite); err != nil {
			return Result{}, err
		}
		if err := writeLocalConfig(localConfigPath, profile, benchmarkInputFile, promptVersion, familyName, experimentName, transportName, transportProvider, baseURL, strings.TrimSpace(opts.APIKeyEnv), models, repeats, requestTimeoutAbortThreshold, interruptPolicy); err != nil {
			return Result{}, err
		}
	}
	return Result{
		Profile:                      profile,
		OutputPath:                   filepath.ToSlash(outputPath),
		BenchmarkInputFile:           filepath.ToSlash(benchmarkInputFile),
		LocalConfigPath:              filepath.ToSlash(localConfigPath),
		PromptVersion:                promptVersion,
		ExperimentName:               experimentName,
		FamilyName:                   familyName,
		TransportName:                transportName,
		TransportProvider:            transportProvider,
		BaseURL:                      baseURL,
		APIKeyEnv:                    strings.TrimSpace(opts.APIKeyEnv),
		Models:                       models,
		Repeats:                      repeats,
		RequestTimeoutAbortThreshold: requestTimeoutAbortThreshold,
		InterruptPolicy:              interruptPolicy,
		CaseCount:                    len(rows),
		ChainCount:                   countUniqueChains(rows),
	}, nil
}

func generateRowsForProfile(profile SurfaceProfile, caseCount int) ([]CaseRow, error) {
	if caseCount < 0 {
		return nil, fmt.Errorf("case_count must be >= 0")
	}
	if profile.Surface == SurfaceBridgeImportFinalResearch {
		return generateBridgeImportFinalRows(caseCount)
	}
	if caseCount > 0 {
		return nil, fmt.Errorf("--case-count is only supported for %q; %q/%q uses a fixed canonical pack", SurfaceBridgeImportFinalResearch, profile.Surface, profile.Mode)
	}
	return profile.generateRows(), nil
}

func DefaultLocalConfigPath(profile SurfaceProfile, promptVersion string) string {
	promptVersion = strings.TrimSpace(promptVersion)
	if override, ok := promptLocalConfigPaths[promptVersion]; ok {
		return override
	}
	return profile.DefaultLocalConfigPath
}

func countUniqueChains(rows []CaseRow) int {
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ChainID) == "" {
			continue
		}
		seen[row.ChainID] = struct{}{}
	}
	return len(seen)
}

func resolveExperimentName(profile SurfaceProfile, promptVersion, override string) string {
	if override = strings.TrimSpace(override); override != "" {
		return override
	}
	promptVersion = strings.TrimSpace(promptVersion)
	if promptVersion == "" || promptVersion == profile.DefaultPromptVersion {
		return profile.DefaultExperimentName
	}
	if value, ok := promptExperimentNames[promptVersion]; ok {
		return value
	}
	return profile.DefaultExperimentName + "-" + sanitizeSlug(promptVersion)
}

func ensureWritableTarget(path string, overwrite bool) error {
	path = strings.TrimSpace(path)
	if path == "" || overwrite {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file %q without --force", filepath.ToSlash(path))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat target %q: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeCasesCSV(path string, header []string, rows []CaseRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create compositional cases dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create compositional cases csv: %w", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write compositional cases header: %w", err)
	}
	for _, row := range rows {
		record, err := rowRecord(header, row)
		if err != nil {
			return fmt.Errorf("encode compositional case %s: %w", row.CaseID, err)
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write compositional case %s: %w", row.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush compositional cases csv: %w", err)
	}
	return nil
}

func writeLocalConfig(path string, profile SurfaceProfile, inputFile, promptVersion, familyName, experimentName, transportName, transportProvider, baseURL, apiKeyEnv string, models []string, repeats, requestTimeoutAbortThreshold int, interruptPolicy string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create compositional local config dir: %w", err)
	}
	payload := localConfigFile{
		Transports: map[string]localTransport{
			transportName: {
				Provider:  transportProvider,
				BaseURL:   baseURL,
				APIKeyEnv: apiKeyEnv,
			},
		},
		Families: map[string]localFamily{
			familyName: {
				Transport:      transportName,
				ExperimentName: experimentName,
				Models:         append([]string(nil), models...),
				Sampling: &localSampling{
					Temperature: 0,
					Seed:        42,
					TopP:        1,
				},
			},
		},
		Benchmarks: map[string]localBenchmark{
			profile.BenchmarkKey: {
				InputFile:                    inputFile,
				PromptVersion:                promptVersion,
				Families:                     []string{familyName},
				Repeats:                      repeats,
				RequestTimeoutAbortThreshold: requestTimeoutAbortThreshold,
				InterruptPolicy:              interruptPolicy,
				SummaryDir:                   profile.SummaryDir,
				ReportDir:                    profile.ReportDir,
			},
		},
	}
	raw, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal compositional local config yaml: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write compositional local config %q: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func rowRecord(header []string, row CaseRow) ([]string, error) {
	record := make([]string, 0, len(header))
	for _, column := range header {
		value, err := rowValue(column, row)
		if err != nil {
			return nil, err
		}
		record = append(record, value)
	}
	return record, nil
}

func rowValue(column string, row CaseRow) (string, error) {
	switch column {
	case "case_id":
		return row.CaseID, nil
	case "chain_id":
		return row.ChainID, nil
	case "chain_protocol":
		return row.ChainProtocol, nil
	case "stage_id":
		return row.StageID, nil
	case "stage_order":
		return strconv.Itoa(row.StageOrder), nil
	case "stage_role":
		return row.StageRole, nil
	case "category":
		return row.Category, nil
	case "label":
		return row.Label, nil
	case "difficulty":
		return row.Difficulty, nil
	case "case_family":
		return row.CaseFamily, nil
	case "chain_depth":
		return strconv.Itoa(row.ChainDepth), nil
	case "provenance_mode":
		return row.ProvenanceMode, nil
	case "atom_renaming_id":
		return row.AtomRenamingID, nil
	case "trusted_reuse":
		return optionalBoolString(row.TrustedReuse), nil
	case "transition_group":
		return row.TransitionGroup, nil
	case "import_arity":
		return row.ImportArity, nil
	case "reuse_shape":
		return row.ReuseShape, nil
	case "bridge_depth":
		return row.BridgeDepth, nil
	case "symbol_overlap":
		return row.SymbolOverlap, nil
	case "lemma_surface_style":
		return row.LemmaSurfaceStyle, nil
	case "negative_twin_hardness":
		return row.NegativeTwinHardness, nil
	case "notes":
		return row.Notes, nil
	case "import_stage_ids_json":
		return marshalJSONString(row.ImportStageIDs)
	case "assumptions_json":
		return marshalJSONString(row.Assumptions)
	case "imported_lemmas_json":
		return marshalJSONString(row.ImportedLemmas)
	case "goal":
		return row.Goal, nil
	case "expected_behavior":
		return row.ExpectedBehavior, nil
	case "comment":
		return row.Comment, nil
	default:
		return "", fmt.Errorf("unsupported compositional csv column %q", column)
	}
}

func marshalJSONString(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func optionalBoolString(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}

func compositionalProfiles() map[string]SurfaceProfile {
	return map[string]SurfaceProfile{
		profileKey(SurfaceBridgeImportFinalResearch, ModePhase1): {
			Surface:                             SurfaceBridgeImportFinalResearch,
			Mode:                                ModePhase1,
			BenchmarkKey:                        "bridge_import_final_research",
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/bridge-import-final-research-phase1.csv",
			DefaultLocalConfigPath:              "research/config/compositional/bridge/import-final-research.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-benchmark-v1.3",
			DefaultExperimentName:               "bridge-import-final-research-v1",
			DefaultFamilyName:                   "bridge-import-final-research-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/bridge-import-final-research",
			ReportDir:                           "research/result_research_report_v1/compositional/bridge-import-final-research",
			CSVHeader:                           bridgeFinalCSVHeader,
			generateRows:                        generateBridgeImportFinalPhase1Rows,
		},
		profileKey(SurfaceBridgeOnlyAuthoring, ModePhase1): {
			Surface:                             SurfaceBridgeOnlyAuthoring,
			Mode:                                ModePhase1,
			BenchmarkKey:                        defaultBridgeBenchmarkKey,
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/bridge-only-authoring-phase1.csv",
			DefaultLocalConfigPath:              "research/config/compositional/bridge/only-authoring.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-bridge-only-v1.0",
			DefaultExperimentName:               "bridge-only-authoring-v1",
			DefaultFamilyName:                   "bridge-only-authoring-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/bridge-only-authoring",
			ReportDir:                           "research/result_research_report_v1/compositional/bridge-only-authoring",
			CSVHeader:                           bridgeOnlyCSVHeader,
			generateRows:                        generateBridgeOnlyPhase1Rows,
		},
		profileKey(SurfaceBridgeOnlyAuthoring, ModeCanonical): {
			Surface:                             SurfaceBridgeOnlyAuthoring,
			Mode:                                ModeCanonical,
			BenchmarkKey:                        defaultBridgeBenchmarkKey,
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/bridge-only-authoring-canonical-phase1.csv",
			DefaultLocalConfigPath:              "research/config/compositional/bridge/only-authoring.canonical.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-bridge-only-canonical-vars-v1.0",
			DefaultExperimentName:               "bridge-only-authoring-canonical-v1",
			DefaultFamilyName:                   "bridge-only-authoring-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/bridge-only-authoring",
			ReportDir:                           "research/result_research_report_v1/compositional/bridge-only-authoring",
			CSVHeader:                           bridgeOnlyCSVHeader,
			generateRows:                        generateBridgeOnlyCanonicalRows,
		},
		profileKey(SurfaceBridgeOnlyAuthoring, ModePhase2): {
			Surface:                             SurfaceBridgeOnlyAuthoring,
			Mode:                                ModePhase2,
			BenchmarkKey:                        defaultBridgeBenchmarkKey,
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/bridge-only-authoring-phase2.csv",
			DefaultLocalConfigPath:              "research/config/compositional/bridge/only-authoring.phase2.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-bridge-only-v1.0",
			DefaultExperimentName:               "bridge-only-authoring-v2",
			DefaultFamilyName:                   "bridge-only-authoring-phase2-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/bridge-only-authoring",
			ReportDir:                           "research/result_research_report_v1/compositional/bridge-only-authoring",
			CSVHeader:                           bridgeOnlyCSVHeader,
			generateRows:                        generateBridgeOnlyPhase2Rows,
		},
		profileKey(SurfaceGoldFinalCompositionOnly, ModePhase1): {
			Surface:                             SurfaceGoldFinalCompositionOnly,
			Mode:                                ModePhase1,
			BenchmarkKey:                        defaultGoldBenchmarkKey,
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/gold-final-composition-only-phase1.csv",
			DefaultLocalConfigPath:              "research/config/compositional/gold/final-composition-only.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-gold-final-only-v1.0",
			DefaultExperimentName:               "gold-final-composition-only-v1",
			DefaultFamilyName:                   "gold-final-composition-only-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/gold-final-composition-only",
			ReportDir:                           "research/result_research_report_v1/compositional/gold-final-composition-only",
			CSVHeader:                           goldFinalCSVHeader,
			generateRows:                        generateGoldFinalPhase1Rows,
		},
		profileKey(SurfaceGoldFinalCompositionOnly, ModePhase2): {
			Surface:                             SurfaceGoldFinalCompositionOnly,
			Mode:                                ModePhase2,
			BenchmarkKey:                        defaultGoldBenchmarkKey,
			ProjectFolder:                       DefaultProjectFolder,
			DefaultOutputFile:                   "cases/gold-final-composition-only-phase2.csv",
			DefaultLocalConfigPath:              "research/config/compositional/gold/final-composition-only.phase2.yaml",
			DefaultPromptVersion:                "hilbert-ai-verification-gold-final-only-v1.0",
			DefaultExperimentName:               "gold-final-composition-only-v2",
			DefaultFamilyName:                   "gold-final-composition-only-phase2-main",
			DefaultRepeats:                      5,
			DefaultRequestTimeoutAbortThreshold: 5,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/gold-final-composition-only",
			ReportDir:                           "research/result_research_report_v1/compositional/gold-final-composition-only",
			CSVHeader:                           goldFinalCSVHeader,
			generateRows:                        generateGoldFinalPhase2Rows,
		},
	}
}

func profileKey(surface, mode string) string {
	return normalizeSurface(surface) + "::" + normalizeMode(mode)
}

func defaultModeForSurface(surface string) string {
	switch normalizeSurface(surface) {
	case SurfaceBridgeOnlyAuthoring, SurfaceBridgeImportFinalResearch, SurfaceGoldFinalCompositionOnly:
		return ModePhase1
	default:
		return ""
	}
}

func normalizeSurface(surface string) string {
	surface = strings.TrimSpace(strings.ToLower(surface))
	return strings.ReplaceAll(surface, "_", "-")
}

func normalizeMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	return strings.ReplaceAll(mode, "_", "-")
}

func sanitizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-", ":", "-", "/", "-", "\\", "-")
	raw = replacer.Replace(raw)
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
	})
	raw = strings.Join(parts, "-")
	raw = strings.Trim(raw, "-")
	raw = strings.ReplaceAll(raw, "--", "-")
	if raw == "" {
		return "generated"
	}
	return raw
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func boolPtr(value bool) *bool {
	return &value
}

func phase2ChainID(prefix string, chain int) string {
	return fmt.Sprintf("%s%d", prefix, chain)
}

func phase2AtomID(chain int) string {
	return fmt.Sprintf("%02d", chain-200)
}
