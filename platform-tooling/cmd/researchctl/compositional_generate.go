package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/compositional"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

type compositionalGenerationManifest struct {
	SchemaVersion string   `json:"schema_version"`
	RunID         string   `json:"run_id"`
	Command       string   `json:"command"`
	Surface       string   `json:"surface"`
	Mode          string   `json:"mode"`
	ProjectFolder string   `json:"project_folder"`
	BenchmarkKey  string   `json:"benchmark_key"`
	OutputPath    string   `json:"output_path"`
	InputFile     string   `json:"input_file"`
	LocalConfig   string   `json:"local_config,omitempty"`
	PromptVersion string   `json:"prompt_version"`
	Experiment    string   `json:"experiment_name"`
	Family        string   `json:"family_name"`
	TransportName string   `json:"transport_name"`
	Provider      string   `json:"provider"`
	BaseURL       string   `json:"base_url"`
	Models        []string `json:"models"`
	Repeats       int      `json:"repeats"`
	TimeoutAbort  int      `json:"request_timeout_abort_threshold"`
	Interrupt     string   `json:"interrupt_policy"`
	CaseCount     int      `json:"case_count"`
	ChainCount    int      `json:"chain_count"`
}

func runGenerateCompositionalCasesCommand(args []string, stdout, stderr io.Writer) error {
	return runGenerateCompositionalCommand("", "generate compositional cases", args, stdout, stderr)
}

func runGenerateBridgeFinalHypothesesCommand(args []string, stdout, stderr io.Writer) error {
	return runGenerateCompositionalCommand(compositional.SurfaceBridgeImportFinalResearch, "generate bridge-final hypotheses", args, stdout, stderr)
}

func runGenerateCompositionalCommand(surfaceOverride, commandName string, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(strings.ReplaceAll(commandName, " ", "-"), flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	runID := fs.String("run-id", timestampRunID(), "Stable compositional generation identifier")
	surface := fs.String("surface", surfaceOverride, "Surface: bridge-import-final-research|bridge-only-authoring|gold-final-composition-only")
	mode := fs.String("mode", "", "Generation mode; supported values depend on surface")
	caseCount := fs.Int("case-count", 0, "Optional case count override; currently supported only for bridge-import-final-research and must preserve full chain groups")
	outputFile := fs.String("output-file", "", "Absolute path, workspace-relative path, or project-relative selector like cases/generated.csv")
	inputFileSelector := fs.String("input-file-selector", "", "Optional benchmark input_file override written into emitted local YAML")
	emitLocalConfig := fs.Bool("emit-local-config", false, "Emit a matching research/config local override")
	localConfigOut := fs.String("local-config-out", "", "Absolute path, workspace-relative path, or research/config-relative local YAML path")
	force := fs.Bool("force", false, "Overwrite existing generated files")
	transportName := fs.String("transport-name", compositional.DefaultTransportName, "Transport key written into emitted local YAML")
	transportProvider := fs.String("transport-provider", compositional.DefaultTransportProvider, "Transport provider written into emitted local YAML")
	baseURL := fs.String("base-url", compositional.DefaultBaseURL, "Transport base_url written into emitted local YAML")
	apiKeyEnv := fs.String("api-key-env", "", "Optional api_key_env written into emitted local YAML")
	familyName := fs.String("family-name", "", "Optional family key override for emitted local YAML")
	experimentName := fs.String("experiment-name", "", "Optional experiment_name override for emitted local YAML")
	promptVersion := fs.String("prompt-version", "", "Optional prompt_version override for emitted local YAML")
	models := fs.String("models", strings.Join([]string{
		"gpt-oss:20b",
		"gpt-oss:120b-cloud",
		"glm-5:cloud",
		"deepseek-v3.1:671b-cloud",
	}, ","), "Comma-separated models for emitted local YAML")
	repeats := fs.Int("repeats", 0, "Override repeats in emitted local YAML; 0 keeps the profile default")
	requestTimeoutAbortThreshold := fs.Int("request-timeout-abort-threshold", 0, "Override request_timeout_abort_threshold in emitted local YAML; 0 keeps the profile default")
	interruptPolicy := fs.String("interrupt-policy", "", "Override interrupt_policy in emitted local YAML")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	loaded, err := researchconfig.Load(workspaceRoot, *configPath, "")
	if err != nil {
		return err
	}
	store, err := state.Open(context.Background(), loaded.WorkspaceRoot, loaded.ResolvePath(loaded.Config.State.DBPath))
	if err != nil {
		return err
	}
	defer store.Close()

	surfaceValue := strings.TrimSpace(*surface)
	if surfaceValue == "" {
		return fmt.Errorf("generate compositional cases requires --surface; supported values: %s", strings.Join(compositional.SupportedSurfaces(), ", "))
	}
	profile, err := compositional.ResolveProfile(surfaceValue, strings.TrimSpace(*mode))
	if err != nil {
		return err
	}

	manifestPath := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ManifestRoot, fmt.Sprintf("generate-compositional-%s-%s_%s.json", profile.Surface, profile.Mode, strings.TrimSpace(*runID)))))
	if err := startRun(context.Background(), store, loaded, *runID, commandName, "generate-compositional", manifestPath); err != nil {
		return err
	}

	outputPath, derivedSelector, err := resolveCompositionalOutputPath(loaded, profile, strings.TrimSpace(*outputFile), *caseCount)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}
	// The CSV can live anywhere, but emitted local YAML must still point back to
	// a benchmark input_file selector that the existing planner understands.
	selector := strings.TrimSpace(*inputFileSelector)
	if selector == "" {
		selector = derivedSelector
	}
	if *emitLocalConfig && strings.TrimSpace(selector) == "" {
		err = fmt.Errorf("cannot derive benchmark input_file from output path %q; pass --input-file-selector explicitly", filepath.ToSlash(outputPath))
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}
	localConfigPath, err := resolveCompositionalLocalConfigPath(loaded, profile, strings.TrimSpace(*localConfigOut), *promptVersion, *emitLocalConfig, *caseCount)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}

	result, err := compositional.Generate(compositional.GenerationOptions{
		Surface:                      profile.Surface,
		Mode:                         profile.Mode,
		CaseCount:                    *caseCount,
		OutputPath:                   outputPath,
		BenchmarkInputFile:           selector,
		LocalConfigPath:              localConfigPath,
		EmitLocalConfig:              *emitLocalConfig || strings.TrimSpace(*localConfigOut) != "",
		Overwrite:                    *force,
		TransportName:                strings.TrimSpace(*transportName),
		TransportProvider:            strings.TrimSpace(*transportProvider),
		BaseURL:                      strings.TrimSpace(*baseURL),
		APIKeyEnv:                    strings.TrimSpace(*apiKeyEnv),
		FamilyName:                   strings.TrimSpace(*familyName),
		ExperimentName:               strings.TrimSpace(*experimentName),
		PromptVersion:                strings.TrimSpace(*promptVersion),
		InterruptPolicy:              strings.TrimSpace(*interruptPolicy),
		Models:                       splitCSV(*models),
		Repeats:                      *repeats,
		RequestTimeoutAbortThreshold: *requestTimeoutAbortThreshold,
	})
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}

	manifest := compositionalGenerationManifest{
		SchemaVersion: "researchctl.generate-compositional/v1",
		RunID:         *runID,
		Command:       commandName,
		Surface:       result.Profile.Surface,
		Mode:          result.Profile.Mode,
		ProjectFolder: result.Profile.ProjectFolder,
		BenchmarkKey:  result.Profile.BenchmarkKey,
		OutputPath:    result.OutputPath,
		InputFile:     result.BenchmarkInputFile,
		LocalConfig:   result.LocalConfigPath,
		PromptVersion: result.PromptVersion,
		Experiment:    result.ExperimentName,
		Family:        result.FamilyName,
		TransportName: result.TransportName,
		Provider:      result.TransportProvider,
		BaseURL:       result.BaseURL,
		Models:        append([]string(nil), result.Models...),
		Repeats:       result.Repeats,
		TimeoutAbort:  result.RequestTimeoutAbortThreshold,
		Interrupt:     result.InterruptPolicy,
		CaseCount:     result.CaseCount,
		ChainCount:    result.ChainCount,
	}
	rawManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return fmt.Errorf("marshal compositional generation manifest: %w", err)
	}
	if err := writeManifestFile(manifestPath, rawManifest); err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}

	if err := store.RecordGeneratedPack(context.Background(), state.GeneratedPack{
		RunID:         *runID,
		PackKey:       strings.TrimSuffix(filepath.Base(result.OutputPath), filepath.Ext(result.OutputPath)),
		ProjectFolder: result.Profile.ProjectFolder,
		SourceKind:    fmt.Sprintf("compositional/%s/%s", result.Profile.Surface, result.Profile.Mode),
		OutputPath:    result.OutputPath,
		ManifestPath:  manifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"input_file":      result.BenchmarkInputFile,
			"local_config":    result.LocalConfigPath,
			"prompt_version":  result.PromptVersion,
			"experiment_name": result.ExperimentName,
		}),
	}); err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
		return err
	}
	for _, artifact := range []state.Artifact{
		{RunID: *runID, Role: "compositional-cases-csv", AbsolutePath: result.OutputPath},
		{RunID: *runID, Role: "compositional-generation-manifest", AbsolutePath: manifestPath},
	} {
		if err := store.RecordArtifact(context.Background(), artifact); err != nil {
			_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
			return err
		}
	}
	if strings.TrimSpace(result.LocalConfigPath) != "" {
		if err := store.RecordArtifact(context.Background(), state.Artifact{
			RunID:        *runID,
			Role:         "compositional-local-config",
			AbsolutePath: result.LocalConfigPath,
		}); err != nil {
			_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, err, nil)
			return err
		}
	}

	if _, err := finishRun(context.Background(), store, *runID, commandName, "generate-compositional", manifestPath, nil, map[string]any{
		"surface":        result.Profile.Surface,
		"mode":           result.Profile.Mode,
		"output_path":    result.OutputPath,
		"input_file":     result.BenchmarkInputFile,
		"local_config":   result.LocalConfigPath,
		"case_count":     result.CaseCount,
		"chain_count":    result.ChainCount,
		"prompt_version": result.PromptVersion,
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "compositional cases generated\nrun_id: %s\nsurface: %s\nmode: %s\noutput_file: %s\ninput_file: %s\nlocal_config: %s\ncase_count: %d\nchain_count: %d\nprompt_version: %s\nexperiment_name: %s\nfamily_name: %s\n",
		*runID,
		result.Profile.Surface,
		result.Profile.Mode,
		result.OutputPath,
		result.BenchmarkInputFile,
		firstNonEmptyGenerate(result.LocalConfigPath, "(not emitted)"),
		result.CaseCount,
		result.ChainCount,
		result.PromptVersion,
		result.ExperimentName,
		result.FamilyName,
	)
	return nil
}

func resolveCompositionalOutputPath(loaded researchconfig.Loaded, profile compositional.SurfaceProfile, raw string, caseCount int) (string, string, error) {
	projectRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, profile.ProjectFolder)))
	raw = strings.TrimSpace(raw)
	if raw == "" {
		defaultOutput := profile.DefaultOutputFile
		if profile.Surface == compositional.SurfaceBridgeImportFinalResearch && caseCount > compositional.BridgeImportFinalPhase1CaseCount {
			defaultOutput = filepath.ToSlash(filepath.Join("cases", fmt.Sprintf("bridge-import-final-research-phase1-expanded-%d.csv", caseCount)))
		}
		return filepath.ToSlash(filepath.Join(projectRoot, filepath.FromSlash(defaultOutput))), filepath.ToSlash(defaultOutput), nil
	}
	if filepath.IsAbs(filepath.FromSlash(raw)) {
		abs := filepath.Clean(filepath.FromSlash(raw))
		return filepath.ToSlash(abs), deriveCompositionalSelector(projectRoot, abs), nil
	}
	if strings.HasPrefix(raw, "research/") || strings.HasPrefix(raw, ".tmp/") {
		abs := loaded.ResolvePath(raw)
		return filepath.ToSlash(abs), deriveCompositionalSelector(projectRoot, abs), nil
	}
	abs := filepath.Join(projectRoot, filepath.FromSlash(raw))
	return filepath.ToSlash(abs), filepath.ToSlash(raw), nil
}

func deriveCompositionalSelector(projectRoot, absolutePath string) string {
	rel, err := filepath.Rel(projectRoot, absolutePath)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") {
		return ""
	}
	return rel
}

func resolveCompositionalLocalConfigPath(loaded researchconfig.Loaded, profile compositional.SurfaceProfile, raw, promptVersion string, emit bool, caseCount int) (string, error) {
	if !emit && strings.TrimSpace(raw) == "" {
		return "", nil
	}
	if strings.TrimSpace(raw) == "" {
		if profile.Surface == compositional.SurfaceBridgeImportFinalResearch && caseCount > compositional.BridgeImportFinalPhase1CaseCount {
			return loaded.ResolvePath(filepath.ToSlash(filepath.Join("research", "config", fmt.Sprintf("local.bridge-import-final-research.phase1-expanded-%d.yaml", caseCount)))), nil
		}
		return loaded.ResolvePath(compositional.DefaultLocalConfigPath(profile, promptVersion)), nil
	}
	if filepath.IsAbs(filepath.FromSlash(raw)) {
		return filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw))), nil
	}
	if strings.HasPrefix(raw, "research/") || strings.HasPrefix(raw, ".tmp/") {
		return loaded.ResolvePath(raw), nil
	}
	return loaded.ResolvePath(filepath.ToSlash(filepath.Join("research", "config", raw))), nil
}

func firstNonEmptyGenerate(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
