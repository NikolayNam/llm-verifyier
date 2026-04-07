package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	openai "github.com/NikolayNam/collabsphere/platform-tooling/internal/llmclient/openai"
)

const (
	benchmarkPromptVersionV1                                = "hilbert-ai-verification-benchmark-v1"
	benchmarkPromptVersionV11                               = "hilbert-ai-verification-benchmark-v1.1"
	benchmarkPromptVersionV12                               = "hilbert-ai-verification-benchmark-v1.2"
	benchmarkPromptVersionV13                               = "hilbert-ai-verification-benchmark-v1.3"
	benchmarkPromptVersionBridgeOnlyV1                      = "hilbert-ai-verification-bridge-only-v1.0"
	benchmarkPromptVersionBridgeOnlySkeletonV1              = "hilbert-ai-verification-bridge-only-skeleton-v1.0"
	benchmarkPromptVersionBridgeOnlyCanonicalVarsV1         = "hilbert-ai-verification-bridge-only-canonical-vars-v1.0"
	benchmarkPromptVersionGoldFinalOnlyV1                   = "hilbert-ai-verification-gold-final-only-v1.0"
	benchmarkPromptVersionGoldFinalOnlyExplicitImportRefsV1 = "hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0"
	benchmarkPromptVersionDefault                           = benchmarkPromptVersionV11
	benchmarkHypothesis                                     = "A minimal Hilbert-style certificate kernel can serve as a useful trust boundary for a formalizable subset of AI-assisted reasoning outputs."
	benchmarkCertificateVersion                             = "1.0.0"
	benchmarkLegacyDefaultCases                             = "cases.csv"
	benchmarkV2HeldOutProjectFolder                         = "hilbert-ai-verification-benchmark-v2-held-out"
	benchmarkNDProjectFolder                                = "hilbert-ai-verification-benchmark-nd-v1"
	benchmarkNDTheoremPack                                  = "theorems/pilot_shared_20260327.csv"
)

type benchmarkPromptProfile struct {
	Version      string
	SystemPrompt string
}

var benchmarkV2HeldOutHardAuditCaseIDs = []string{
	"V2E01",
	"V2E06",
	"V2E08",
	"V2E07",
	"V2E12",
	"V2N07",
	"V2N09",
}

type benchmarkCase struct {
	TheoremPackID        string
	TheoremID            string
	CaseID               string
	SourceCaseID         string
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
	ExpectedBehavior     string
	TrustedReuse         *bool
	ImportStageIDs       []string
	SourceFamily         string
	TargetFamily         string
	TransitionGroup      string
	ImportArity          string
	ReuseShape           string
	BridgeDepth          string
	SymbolOverlap        string
	LemmaSurfaceStyle    string
	NegativeTwinHardness string
	Notes                string
	Assumptions          []string
	ImportedLemmas       []string
	Goal                 string
	Comment              string
}

type benchmarkImportedArtifactRef struct {
	RunID                 string `json:"run_id,omitempty"`
	CaseID                string `json:"case_id,omitempty"`
	ChainID               string `json:"chain_id,omitempty"`
	StageID               string `json:"stage_id,omitempty"`
	Goal                  string `json:"goal,omitempty"`
	CertificateObjectPath string `json:"certificate_object_file,omitempty"`
	TrustedForReuse       bool   `json:"trusted_for_reuse"`
}

type benchmarkTrustedCertificateObject struct {
	SchemaVersion         string          `json:"schema_version"`
	RunID                 string          `json:"run_id"`
	ProjectFolder         string          `json:"project_folder"`
	Surface               string          `json:"surface"`
	ExperimentName        string          `json:"experiment_name,omitempty"`
	ExperimentID          string          `json:"experiment_id,omitempty"`
	PromptVersion         string          `json:"prompt_version"`
	CertificateVersion    string          `json:"certificate_version"`
	CaseID                string          `json:"case_id"`
	TheoremID             string          `json:"theorem_id,omitempty"`
	ChainID               string          `json:"chain_id,omitempty"`
	StageID               string          `json:"stage_id,omitempty"`
	Category              string          `json:"category,omitempty"`
	CaseFamily            string          `json:"case_family,omitempty"`
	ChainDepth            int             `json:"chain_depth,omitempty"`
	AtomRenamingID        string          `json:"atom_renaming_id,omitempty"`
	Goal                  string          `json:"goal"`
	Assumptions           []string        `json:"assumptions,omitempty"`
	ImportedLemmas        []string        `json:"imported_lemmas,omitempty"`
	ExpectedBehavior      string          `json:"expected_behavior,omitempty"`
	RawOutputKind         string          `json:"raw_output_kind"`
	ScoreBucket           string          `json:"score_bucket"`
	SchemaStatus          string          `json:"schema_status"`
	ParseStatus           string          `json:"parse_status"`
	KernelStatus          string          `json:"kernel_status"`
	CLIExitCode           string          `json:"cli_exit_code,omitempty"`
	TrustedForReuse       bool            `json:"trusted_for_reuse"`
	TrustedReuseReason    string          `json:"trusted_reuse_reason,omitempty"`
	CertificateObjectPath string          `json:"certificate_object_file,omitempty"`
	CertificateJSON       json.RawMessage `json:"certificate_json"`
}

type benchmarkTrustedCertificateCatalogRow struct {
	RunID                 string
	CaseID                string
	ChainID               string
	StageID               string
	Category              string
	CaseFamily            string
	ChainDepth            string
	Goal                  string
	TrustedForReuse       string
	TrustedReuseReason    string
	CertificateObjectPath string
}

type benchmarkPromptCaseResolution struct {
	ResolvedCase      benchmarkCase
	ResolvedArtifacts []benchmarkImportedArtifactRef
}

type benchmarkResultRow struct {
	RunID                    string
	TimestampUTC             string
	Provider                 string
	Model                    string
	LLMModel                 string
	ExperimentName           string
	ExperimentID             string
	SamplingSurface          string
	RequestedSamplingProfile string
	EffectiveSamplingProfile string
	RequestedSamplingJSON    string
	EffectiveSamplingJSON    string
	UnsupportedSamplingJSON  string
	GoogleThinkingMode       string
	PromptVersion            string
	PromptHash               string
	Surface                  string
	NDProofStatus            string
	LoweringStatus           string
	PipelineStage            string
	PipelineDetail           string
	TheoremPackID            string
	TheoremID                string
	CaseID                   string
	ChainID                  string
	StageID                  string
	Category                 string
	CaseFamily               string
	ChainDepth               string
	ProvenanceMode           string
	AtomRenamingID           string
	SourceFamily             string
	TargetFamily             string
	TransitionGroup          string
	ImportArity              string
	ReuseShape               string
	BridgeDepth              string
	SymbolOverlap            string
	LemmaSurfaceStyle        string
	NegativeTwinHardness     string
	CaseNotes                string
	ExpectedLabel            string
	RawOutputKind            string
	SchemaStatus             string
	ParseStatus              string
	KernelStatus             string
	ContractStatus           string
	CLIExitCode              string
	ScoreBucket              string
	LatencyMS                string
	OutputLength             string
	ProofStepsCount          string
	RequestedImportsJSON     string
	EffectiveImportsJSON     string
	ResolvedImportRefsJSON   string
	CertificateObjectPath    string
	Notes                    string
}

type benchmarkSummary struct {
	RunID                    string
	Provider                 string
	LLMModel                 string
	ExperimentName           string
	ExperimentID             string
	SamplingSurface          string
	RequestedSamplingProfile string
	EffectiveSamplingProfile string
	RequestedSamplingJSON    string
	EffectiveSamplingJSON    string
	UnsupportedSamplingJSON  string
	GoogleThinkingMode       string
	PromptVersion            string
	ProjectFolder            string
	Surface                  string
	TheoremPackID            string
	CasesFile                string
	CaseSelector             string
	ResultsPath              string
	SummaryPath              string
	ChainSummaryPath         string
	ImportCatalogPath        string
	RawRunDir                string
	CaseCount                int
	AvgLatencyMS             float64
	MaxLatencyMS             int64
	RunElapsedS              int64
	Buckets                  map[string]int
	CategoryStats            map[string]map[string]int
}

type benchmarkRunOptions struct {
	ProjectFolder                string
	CasesFile                    string
	CaseSelector                 string
	CasesPath                    string
	ResultsPath                  string
	SummaryPath                  string
	RawBaseDir                   string
	RunID                        string
	ArtifactKey                  string
	Provider                     string
	ProviderLabel                string
	LLMModel                     string
	ExperimentName               string
	ExperimentID                 string
	RequestedTemperature         *float64
	RequestedSeed                *int64
	RequestedTopP                *float64
	GoogleThinkingMode           string
	PromptVersion                string
	Surface                      string
	ModelLabel                   string
	CaseID                       string
	Limit                        int
	RequestTimeoutAbortThreshold int
}

type benchmarkModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, time.Duration, error)
}

type benchmarkVerifier interface {
	Verify(ctx context.Context, certificatePath string) (*benchmarkVerifyResult, error)
}

type benchmarkVerifyResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type benchmarkRunSummaryRow struct {
	RunID                    string
	TimestampUTC             string
	Provider                 string
	Model                    string
	LLMModel                 string
	ExperimentName           string
	ExperimentID             string
	SamplingSurface          string
	RequestedSamplingProfile string
	EffectiveSamplingProfile string
	RequestedSamplingJSON    string
	EffectiveSamplingJSON    string
	UnsupportedSamplingJSON  string
	GoogleThinkingMode       string
	PromptVersion            string
	CertificateVersion       string
	ProjectFolder            string
	Surface                  string
	TheoremPackID            string
	CasesFile                string
	TheoremsFile             string
	CaseSelector             string
	Hypothesis               string
	CasesPath                string
	TheoremsPath             string
	ResultsPath              string
	ChainSummaryPath         string
	ImportCatalogPath        string
	RawDir                   string
	CasesTotal               string
	PassCount                string
	FalseRefusalCount        string
	FalseAcceptCount         string
	RequestFailureCount      string
	SchemaFailureCount       string
	ParseFailureCount        string
	KernelFailureCount       string
	ContractFailureCount     string
	FormatFailureCount       string
	AvgLatencyMS             string
	MaxLatencyMS             string
	RunElapsedSeconds        string
	CategoryCount            string
}

type benchmarkRunSummaryAggregate struct {
	Model                string
	LLMModel             string
	ExperimentName       string
	ExperimentID         string
	DisplayName          string
	SizeBucket           string
	SizeLabel            string
	Tier                 string
	ComparisonStatus     string
	PromptVersion        string
	CertificateVersion   string
	Runs                 int
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatencyCasesTotal    int
	LatencySumMS         float64
	MaxLatencyMS         int64
	TimedRuns            int
	RunElapsedTotalS     int64
	RunElapsedMaxS       int64
	LatestRunID          string
	LatestTimestampUTC   string
	latestTimestamp      time.Time
}

type benchmarkResultSliceAggregate struct {
	Model                string
	LLMModel             string
	ExperimentName       string
	ExperimentID         string
	DisplayName          string
	SizeBucket           string
	SizeLabel            string
	Tier                 string
	ComparisonStatus     string
	PromptVersion        string
	Surface              string
	Runs                 int
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatencyCasesTotal    int
	LatencySumMS         float64
	MaxLatencyMS         float64
	LatestRunID          string
	LatestTimestampUTC   string
	latestTimestamp      time.Time
}

type benchmarkHardCaseAuditAggregate struct {
	Model                string
	LLMModel             string
	ExperimentName       string
	ExperimentID         string
	DisplayName          string
	SizeBucket           string
	SizeLabel            string
	Tier                 string
	ComparisonStatus     string
	PromptVersion        string
	Surface              string
	Runs                 int
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatestRunID          string
	LatestTimestampUTC   string
	LatestResultPath     string
	LatestRawDir         string
	latestTimestamp      time.Time
}

type benchmarkModelCatalogRow struct {
	LLMModel         string
	DisplayName      string
	SizeBucket       string
	SizeLabel        string
	Tier             string
	ComparisonStatus string
	Notes            string
}

type benchmarkComparativeAppendixEntry struct {
	Model               string
	LLMModel            string
	ExperimentName      string
	ExperimentID        string
	DisplayName         string
	SizeBucket          string
	SizeLabel           string
	Tier                string
	ComparisonStatus    string
	PromptVersion       string
	Status              string
	Reason              string
	Runs                int
	CasesTotal          int
	PassCount           int
	RequestFailureCount int
	LatestRunID         string
	latestTimestamp     time.Time
}

type benchmarkComparativeSlices struct {
	EngineeringBestPerformance []benchmarkRunSummaryAggregate
	CleanBaseline              []benchmarkRunSummaryAggregate
	LE20B                      []benchmarkRunSummaryAggregate
	GT20B                      []benchmarkRunSummaryAggregate
	UnknownSizeAppendix        []benchmarkRunSummaryAggregate
	Appendix                   []benchmarkComparativeAppendixEntry
}

type benchmarkCasePackReport struct {
	CasesFile     string
	Hypothesis    string
	Rows          []benchmarkRunSummaryRow
	Aggregates    []benchmarkRunSummaryAggregate
	Comparative   *benchmarkComparativeSlices
	Observed      *benchmarkObservedData
	ChainSummary  *benchmarkChainSummaryData
	TotalRuns     int
	GeneratedAt   time.Time
	ProjectFolder string
}

type benchmarkChainSummaryData struct {
	Warnings               []string
	Aggregates             []benchmarkChainSummaryAggregate
	StageAggregates        []benchmarkChainStageAggregate
	Failures               []benchmarkChainFailureAggregate
	TransitionGroups       []benchmarkChainMetadataAggregate
	ImportArities          []benchmarkChainMetadataAggregate
	ReuseShapes            []benchmarkChainMetadataAggregate
	BridgeDepths           []benchmarkChainMetadataAggregate
	SymbolOverlaps         []benchmarkChainMetadataAggregate
	NegativeTwinHardnesses []benchmarkChainMetadataAggregate
	FGFailureMix           []benchmarkFGFailureAggregate
	HardestFGFailures      []benchmarkFGCaseFailureAggregate
}

type benchmarkChainSummaryLoadedRow struct {
	RunID                     string
	TimestampUTC              string
	Model                     string
	LLMModel                  string
	ExperimentName            string
	ExperimentID              string
	DisplayName               string
	PromptVersion             string
	Surface                   string
	ChainID                   string
	ChainProtocol             string
	CaseFamily                string
	ChainDepth                int
	AtomRenamingID            string
	SourceFamily              string
	TargetFamily              string
	TransitionGroup           string
	ImportArity               string
	ReuseShape                string
	BridgeDepth               string
	SymbolOverlap             string
	LemmaSurfaceStyle         string
	NegativeTwinHardness      string
	CaseNotes                 string
	Stage1Pass                bool
	Stage2Pass                bool
	Stage3GoldPass            bool
	Stage3ModelPass           bool
	NegativeTwinPass          bool
	AllStagesPass             bool
	ImportStagePass           bool
	FinalCompositionPassGold  bool
	FinalCompositionPassModel bool
	ConditionalFinalPassGold  bool
	ConditionalFinalPassModel bool
	StageSequence             []string
	StageRoles                map[string]string
	StagePasses               map[string]bool
	StageFailures             map[string]string
	TrustedReuseStages        []string
	GoldImportStages          []string
	ModelImportStages         []string
	FailureStage              string
	FailureStageRole          string
	FailureType               string
	FailureClass              string
	FailureDetail             string
	Notes                     string
}

type benchmarkChainSummaryAggregate struct {
	Model                          string
	LLMModel                       string
	ExperimentName                 string
	ExperimentID                   string
	DisplayName                    string
	PromptVersion                  string
	Surface                        string
	Runs                           int
	ChainsTotal                    int
	ImportStagePassCount           int
	FinalCompositionPassGoldCount  int
	FinalCompositionPassModelCount int
	ConditionalFinalPassGoldCount  int
	ConditionalFinalPassModelCount int
	NegativeTwinPassCount          int
	AllStagesPassCount             int
	ObservedStageIDs               []string
	LatestRunID                    string
	LatestTimestampUTC             string
	latestTimestamp                time.Time
}

type benchmarkChainMetadataAggregate struct {
	Model                 string
	LLMModel              string
	ExperimentName        string
	ExperimentID          string
	DisplayName           string
	PromptVersion         string
	Surface               string
	GroupValue            string
	Runs                  int
	ChainsTotal           int
	GoldFinalPassCount    int
	NegativeTwinPassCount int
	AllStagesPassCount    int
	LatestRunID           string
	LatestTimestampUTC    string
	latestTimestamp       time.Time
}

type benchmarkChainStageAggregate struct {
	Model              string
	LLMModel           string
	ExperimentName     string
	ExperimentID       string
	DisplayName        string
	PromptVersion      string
	Surface            string
	StageID            string
	StageRole          string
	Chains             int
	PassCount          int
	LatestRunID        string
	LatestTimestampUTC string
	latestTimestamp    time.Time
}

type benchmarkChainFailureAggregate struct {
	Model              string
	LLMModel           string
	ExperimentName     string
	ExperimentID       string
	DisplayName        string
	PromptVersion      string
	Surface            string
	FailureStage       string
	FailureStageRole   string
	FailureType        string
	FailureClass       string
	Chains             int
	LatestRunID        string
	LatestTimestampUTC string
	latestTimestamp    time.Time
}

type benchmarkFGFailureAggregate struct {
	Model              string
	LLMModel           string
	ExperimentName     string
	ExperimentID       string
	DisplayName        string
	PromptVersion      string
	Surface            string
	FailureType        string
	Cases              int
	LatestRunID        string
	LatestTimestampUTC string
	latestTimestamp    time.Time
}

type benchmarkFGCaseFailureAggregate struct {
	Model                string
	LLMModel             string
	ExperimentName       string
	ExperimentID         string
	DisplayName          string
	PromptVersion        string
	Surface              string
	CaseID               string
	ChainID              string
	TransitionGroup      string
	ImportArity          string
	ReuseShape           string
	BridgeDepth          string
	SymbolOverlap        string
	NegativeTwinHardness string
	Runs                 int
	CasesTotal           int
	PassCount            int
	FailureCount         int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatestRunID          string
	LatestTimestampUTC   string
	latestTimestamp      time.Time
}

type benchmarkHardCaseAudit struct {
	TargetCaseID    string
	Case            benchmarkCase
	ObservedCaseIDs []string
	Aggregates      []benchmarkHardCaseAuditAggregate
}

type benchmarkCaseFailureAggregate struct {
	Case                 benchmarkCase
	Runs                 int
	CasesTotal           int
	PassCount            int
	FailureCount         int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	LatestRunID          string
	LatestTimestampUTC   string
	LatestResultPath     string
	LatestRawDir         string
	latestTimestamp      time.Time
}

type benchmarkNDPipelineTraceAggregate struct {
	Model                   string
	LLMModel                string
	ExperimentName          string
	ExperimentID            string
	DisplayName             string
	SizeBucket              string
	SizeLabel               string
	Tier                    string
	ComparisonStatus        string
	PromptVersion           string
	Surface                 string
	Runs                    int
	CasesTotal              int
	NDProofObjectFailures   int
	LoweringFailures        int
	HilbertArtifactFailures int
	OutputRefusals          int
	OutputFormatFailures    int
	RequestFailures         int
	UnknownStageRows        int
	LatestRunID             string
	LatestTimestampUTC      string
	latestTimestamp         time.Time
}

type benchmarkObservedData struct {
	Warnings        []string
	Aggregate       []benchmarkResultSliceAggregate
	Entailed        []benchmarkResultSliceAggregate
	NotEntailed     []benchmarkResultSliceAggregate
	ByCategory      map[string][]benchmarkResultSliceAggregate
	NDPipelineTrace []benchmarkNDPipelineTraceAggregate
	HardestFailures []benchmarkCaseFailureAggregate
	HardCases       []benchmarkHardCaseAudit
}

type benchmarkResearchPairComparison struct {
	LLMModel string
	Direct   benchmarkRunSummaryRow
	ND       benchmarkRunSummaryRow
}

type benchmarkResearchRowStats struct {
	CasesTotal           int
	PassCount            int
	FalseRefusalCount    int
	FalseAcceptCount     int
	RequestFailureCount  int
	SchemaFailureCount   int
	ParseFailureCount    int
	KernelFailureCount   int
	ContractFailureCount int
	FormatFailureCount   int
	AvgLatencyMS         float64
	MaxLatencyMS         int64
	RunElapsedSeconds    int64
}

type llmPromptRunner struct {
	client      openai.Client
	model       string
	temperature *float64
	seed        *int64
	topP        *float64
}

func (r *llmPromptRunner) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, time.Duration, error) {
	start := time.Now()
	resp, err := r.client.CreateResponse(ctx, &openai.ResponseRequest{
		Model: r.model,
		Messages: []openai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: r.temperature,
		Seed:        r.seed,
		TopP:        r.topP,
	})
	latency := time.Since(start)
	if err != nil {
		return "", latency, err
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return "", latency, fmt.Errorf("llm benchmark: empty model response")
	}
	return resp.Choices[0].Message.Content, latency, nil
}

type hilbertCLIVerifier struct {
	binaryPath string
}

func (v *hilbertCLIVerifier) Verify(ctx context.Context, certificatePath string) (*benchmarkVerifyResult, error) {
	cmd := exec.CommandContext(ctx, v.binaryPath, certificatePath)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !asExitError(err, &exitErr) {
			return nil, fmt.Errorf("run hilbertcheck: %w", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return &benchmarkVerifyResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}
