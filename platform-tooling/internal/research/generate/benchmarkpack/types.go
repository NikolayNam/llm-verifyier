package benchmarkpack

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SurfaceDirect = "direct"
	SurfaceND     = "nd"

	BenchmarkKeyDirect = "phase1"
	BenchmarkKeyND     = "nd"

	DefaultDirectProjectFolder = "hilbert-ai-verification-benchmark-v2-held-out"
	DefaultNDProjectFolder     = "hilbert-ai-verification-benchmark-nd-v1"

	DirectBaselineCaseCount = 24
	NDBaselineCaseCount     = 8

	DefaultTransportName      = "local-compatible"
	DefaultTransportProvider  = "compatible"
	DefaultBaseURL            = "http://localhost:11434"
	DefaultInterruptPolicy    = "drop_if_no_results"
	DefaultExperimentName     = "formal-mode"
	DefaultDirectPrompt       = "hilbert-ai-verification-benchmark-v1.3"
	DefaultNDPrompt           = "hilbert-ai-verification-benchmark-nd-v1.1"
	DefaultGeneratedSamplingT = 0
	DefaultGeneratedSamplingP = 1
	DefaultDirectRepeats      = 10
	DefaultNDRepeats          = 10
	DefaultAbortThreshold     = 10
)

var defaultGeneratedModels = []string{
	"gpt-oss:20b",
	"gpt-oss:120b-cloud",
	"glm-5:cloud",
	"deepseek-v3.1:671b-cloud",
}

var directCSVHeader = []string{
	"case_id",
	"source_case_id",
	"category",
	"label",
	"difficulty",
	"assumptions_json",
	"goal",
	"comment",
}

var ndCSVHeader = []string{
	"theorem_pack_id",
	"theorem_id",
	"case_id",
	"category",
	"label",
	"difficulty",
	"assumptions_json",
	"goal",
	"logic_fragment",
	"comment",
}

type Row struct {
	TheoremPackID string
	TheoremID     string
	CaseID        string
	SourceCaseID  string
	Category      string
	Label         string
	Difficulty    string
	Assumptions   []string
	Goal          string
	LogicFragment string
	Comment       string
}

type SurfaceProfile struct {
	Surface                             string
	BenchmarkKey                        string
	ProjectFolder                       string
	DefaultOutputFile                   string
	DefaultLocalConfigPath              string
	DefaultPromptVersion                string
	DefaultRepeats                      int
	DefaultRequestTimeoutAbortThreshold int
	DefaultInterruptPolicy              string
	SummaryDir                          string
	ReportDir                           string
	CSVHeader                           []string
}

type GenerationOptions struct {
	Surface                      string
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
	return []string{SurfaceDirect, SurfaceND}
}

func ResolveProfile(surface string) (SurfaceProfile, error) {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return SurfaceProfile{
			Surface:                             SurfaceDirect,
			BenchmarkKey:                        BenchmarkKeyDirect,
			ProjectFolder:                       DefaultDirectProjectFolder,
			DefaultOutputFile:                   "cases.csv",
			DefaultLocalConfigPath:              filepath.ToSlash(filepath.Join("research", "config", "direct", "generated.yaml")),
			DefaultPromptVersion:                DefaultDirectPrompt,
			DefaultRepeats:                      DefaultDirectRepeats,
			DefaultRequestTimeoutAbortThreshold: DefaultAbortThreshold,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/waves",
			ReportDir:                           "research/result_research_report_v1/direct",
			CSVHeader:                           directCSVHeader,
		}, nil
	case SurfaceND:
		return SurfaceProfile{
			Surface:                             SurfaceND,
			BenchmarkKey:                        BenchmarkKeyND,
			ProjectFolder:                       DefaultNDProjectFolder,
			DefaultOutputFile:                   "theorems/pilot_shared_20260327.csv",
			DefaultLocalConfigPath:              filepath.ToSlash(filepath.Join("research", "config", "nd", "generated.yaml")),
			DefaultPromptVersion:                DefaultNDPrompt,
			DefaultRepeats:                      DefaultNDRepeats,
			DefaultRequestTimeoutAbortThreshold: DefaultAbortThreshold,
			DefaultInterruptPolicy:              DefaultInterruptPolicy,
			SummaryDir:                          "research/artifacts/result_research/nd",
			ReportDir:                           "research/result_research_report_v1/nd",
			CSVHeader:                           ndCSVHeader,
		}, nil
	default:
		return SurfaceProfile{}, fmt.Errorf("unsupported benchmark generator surface %q", surface)
	}
}

// Generate emits direct-Hilbert and ND benchmark-compatible CSV packs plus an
// optional local YAML override that can be passed directly to researchctl.
func Generate(opts GenerationOptions) (Result, error) {
	profile, err := ResolveProfile(opts.Surface)
	if err != nil {
		return Result{}, err
	}
	outputPath := strings.TrimSpace(opts.OutputPath)
	if outputPath == "" {
		return Result{}, fmt.Errorf("benchmark cases output path is required")
	}
	benchmarkInputFile := strings.TrimSpace(opts.BenchmarkInputFile)
	if benchmarkInputFile == "" {
		benchmarkInputFile = profile.DefaultOutputFile
	}
	if err := ensureWritableTarget(outputPath, opts.Overwrite); err != nil {
		return Result{}, err
	}
	rows, err := generateRows(profile.Surface, opts.CaseCount)
	if err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		return Result{}, fmt.Errorf("benchmark generator %q produced no rows", profile.Surface)
	}
	if err := writeCasesCSV(outputPath, profile.CSVHeader, rows); err != nil {
		return Result{}, err
	}

	promptVersion := firstNonEmpty(strings.TrimSpace(opts.PromptVersion), profile.DefaultPromptVersion)
	transportName := firstNonEmpty(strings.TrimSpace(opts.TransportName), DefaultTransportName)
	transportProvider := firstNonEmpty(strings.TrimSpace(opts.TransportProvider), DefaultTransportProvider)
	baseURL := firstNonEmpty(strings.TrimSpace(opts.BaseURL), DefaultBaseURL)
	experimentName := firstNonEmpty(strings.TrimSpace(opts.ExperimentName), DefaultExperimentName)
	models := append([]string(nil), opts.Models...)
	if len(models) == 0 {
		models = append(models, defaultGeneratedModels...)
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
	familyName := firstNonEmpty(strings.TrimSpace(opts.FamilyName), defaultFamilyName(profile.Surface, len(rows)))

	localConfigPath := strings.TrimSpace(opts.LocalConfigPath)
	if opts.EmitLocalConfig {
		if localConfigPath == "" {
			localConfigPath = defaultLocalConfigPath(profile.Surface, len(rows))
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
	}, nil
}

func normalizeSurface(surface string) string {
	switch strings.ToLower(strings.TrimSpace(surface)) {
	case "phase1", "direct":
		return SurfaceDirect
	case "nd":
		return SurfaceND
	default:
		return strings.ToLower(strings.TrimSpace(surface))
	}
}

func defaultExpandedOutputFile(surface string, count int) string {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return filepath.ToSlash(filepath.Join("cases", fmt.Sprintf("phase1-expanded-%d.csv", count)))
	case SurfaceND:
		return filepath.ToSlash(filepath.Join("theorems", fmt.Sprintf("pilot_shared_20260327-expanded-%d.csv", count)))
	default:
		return ""
	}
}

func defaultLocalConfigPath(surface string, count int) string {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return filepath.ToSlash(filepath.Join("research", "config", "direct", fmt.Sprintf("phase1-expanded-%d.yaml", count)))
	case SurfaceND:
		return filepath.ToSlash(filepath.Join("research", "config", "nd", fmt.Sprintf("expanded-%d.yaml", count)))
	default:
		return ""
	}
}

func DefaultOutputFileSelector(surface string, count int) string {
	return defaultExpandedOutputFile(surface, count)
}

func DefaultExpandedLocalConfigPath(surface string, count int) string {
	return defaultLocalConfigPath(surface, count)
}

func BaselineCaseCount(surface string) int {
	return baselineCaseCount(surface)
}

func defaultFamilyName(surface string, count int) string {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return fmt.Sprintf("phase1-expanded-%d-main", count)
	case SurfaceND:
		return fmt.Sprintf("nd-expanded-%d-main", count)
	default:
		return "generated-benchmark-main"
	}
}

func baselineCaseCount(surface string) int {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return DirectBaselineCaseCount
	case SurfaceND:
		return NDBaselineCaseCount
	default:
		return 0
	}
}

func generateRows(surface string, caseCount int) ([]Row, error) {
	switch normalizeSurface(surface) {
	case SurfaceDirect:
		return generateDirectRows(caseCount)
	case SurfaceND:
		return generateNDRows(caseCount)
	default:
		return nil, fmt.Errorf("unsupported benchmark generator surface %q", surface)
	}
}

func ensureWritableTarget(path string, overwrite bool) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing file %q; pass --force to replace it", filepath.ToSlash(path))
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir for %q: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func writeCasesCSV(path string, header []string, rows []Row) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create benchmark csv %q: %w", filepath.ToSlash(path), err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write benchmark csv header %q: %w", filepath.ToSlash(path), err)
	}
	for _, row := range rows {
		record, err := rowToRecord(header, row)
		if err != nil {
			return err
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write benchmark csv row %q/%q: %w", filepath.ToSlash(path), row.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush benchmark csv %q: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func rowToRecord(header []string, row Row) ([]string, error) {
	assumptions, err := yamlJSONArray(row.Assumptions)
	if err != nil {
		return nil, fmt.Errorf("marshal assumptions for %s: %w", row.CaseID, err)
	}
	values := map[string]string{
		"theorem_pack_id":  row.TheoremPackID,
		"theorem_id":       row.TheoremID,
		"case_id":          row.CaseID,
		"source_case_id":   row.SourceCaseID,
		"category":         row.Category,
		"label":            row.Label,
		"difficulty":       row.Difficulty,
		"assumptions_json": assumptions,
		"goal":             row.Goal,
		"logic_fragment":   row.LogicFragment,
		"comment":          row.Comment,
	}
	record := make([]string, len(header))
	for i, name := range header {
		record[i] = values[name]
	}
	return record, nil
}

func yamlJSONArray(values []string) (string, error) {
	if len(values) == 0 {
		return "[]", nil
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ",") + "]", nil
}

func writeLocalConfig(path string, profile SurfaceProfile, benchmarkInputFile, promptVersion, familyName, experimentName, transportName, transportProvider, baseURL, apiKeyEnv string, models []string, repeats, requestTimeoutAbortThreshold int, interruptPolicy string) error {
	cfg := localConfigFile{
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
				Models:         models,
				Sampling: &localSampling{
					Temperature: DefaultGeneratedSamplingT,
					Seed:        42,
					TopP:        DefaultGeneratedSamplingP,
				},
			},
		},
		Benchmarks: map[string]localBenchmark{
			profile.BenchmarkKey: {
				InputFile:                    benchmarkInputFile,
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
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal local benchmark config %q: %w", filepath.ToSlash(path), err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write local benchmark config %q: %w", filepath.ToSlash(path), err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
