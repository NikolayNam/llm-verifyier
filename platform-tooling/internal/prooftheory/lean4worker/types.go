package lean4worker

import "time"

const (
	DefaultProjectName    = "CollabSphereLean"
	DefaultToolchain      = "leanprover/lean4:v4.22.0"
	DefaultMathlibRev     = "v4.22.0"
	DefaultProjectDir     = "lean-project"
	DefaultResultDir      = "result"
	DefaultRunTimeout     = 60 * time.Second
	DefaultLakeBinary     = "lake"
	DefaultGeneratedDir   = ".generated"
	DefaultTheoremsDir    = "theorems"
	HistoricalTheoremsDir = "historical"
	LegacyCasesDir        = "cases"
)

type Job struct {
	Name      string
	Statement string
	Payload   []byte
}

type ModulePayload struct {
	Imports []string `json:"imports,omitempty"`
	Helpers []string `json:"helpers,omitempty"`
	Proof   string   `json:"proof,omitempty"`
}

type Options struct {
	ProjectDir string
	ResultRoot string
	RunID      string
	LakeBinary string
	Timeout    time.Duration
}

type Result struct {
	OK          bool   `json:"ok"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exit_code"`
	ArtifactDir string `json:"artifact_dir"`
	ModulePath  string `json:"module_path"`
	ReportPath  string `json:"report_path"`
}

type Report struct {
	GeneratedAtUTC string `json:"generated_at_utc"`
	ProjectDir     string `json:"project_dir"`
	LakeBinary     string `json:"lake_binary"`
	JobName        string `json:"job_name"`
	Statement      string `json:"statement"`
	PayloadPresent bool   `json:"payload_present"`
	Result         Result `json:"result"`
}

const (
	HypothesisGenerationModeEnumerator = "enumerator"
	HypothesisGenerationModeCurated    = "curated_backlog"
	HypothesisGenerationModeProposed   = "model_proposed"

	DerivabilityStatusDerivable    = "derivable"
	DerivabilityStatusNotDerivable = "not_derivable"
)

type HypothesisCase struct {
	TheoremPackID      string
	TheoremID          string
	CaseID             string
	GenerationMode     string
	Category           string
	DerivabilityStatus string
	Interesting        bool
	Minimal            bool
	Difficulty         string
	Assumptions        []string
	Goal               string
	LogicFragment      string
	LeanStatement      string
	Comment            string
}

type HypothesisGenerationConfig struct {
	Mode            string   `json:"mode"`
	Atoms           []string `json:"atoms,omitempty"`
	MaxFormulaDepth int      `json:"max_formula_depth,omitempty"`
	MaxAssumptions  int      `json:"max_assumptions,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Filters         []string `json:"filters,omitempty"`
	SourceFile      string   `json:"source_file,omitempty"`
}
