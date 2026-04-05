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
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/generate/benchmarkpack"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

type benchmarkGenerationManifest struct {
	SchemaVersion string   `json:"schema_version"`
	RunID         string   `json:"run_id"`
	Command       string   `json:"command"`
	Surface       string   `json:"surface"`
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
}

func runGenerateBenchmarkCasesCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate-benchmark-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	runID := fs.String("run-id", timestampRunID(), "Stable benchmark-pack generation identifier")
	surface := fs.String("surface", "", "Surface: direct|nd")
	caseCount := fs.Int("case-count", 0, "Optional expanded pack size; must preserve generator guardrails for the selected surface")
	outputFile := fs.String("output-file", "", "Absolute path, workspace-relative path, or project-relative selector like cases/generated.csv")
	inputFileSelector := fs.String("input-file-selector", "", "Optional benchmark input_file override written into emitted local YAML")
	emitLocalConfig := fs.Bool("emit-local-config", false, "Emit a matching research/config local override")
	localConfigOut := fs.String("local-config-out", "", "Absolute path, workspace-relative path, or research/config-relative local YAML path")
	force := fs.Bool("force", false, "Overwrite existing generated files")
	transportName := fs.String("transport-name", benchmarkpack.DefaultTransportName, "Transport key written into emitted local YAML")
	transportProvider := fs.String("transport-provider", benchmarkpack.DefaultTransportProvider, "Transport provider written into emitted local YAML")
	baseURL := fs.String("base-url", benchmarkpack.DefaultBaseURL, "Transport base_url written into emitted local YAML")
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

	profile, err := benchmarkpack.ResolveProfile(strings.TrimSpace(*surface))
	if err != nil {
		return err
	}

	commandName := "generate benchmark cases"
	manifestPath := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ManifestRoot, fmt.Sprintf("generate-benchmark-%s_%s.json", profile.Surface, strings.TrimSpace(*runID)))))
	if err := startRun(context.Background(), store, loaded, *runID, commandName, "generate-benchmark", manifestPath); err != nil {
		return err
	}

	outputPath, derivedSelector, err := resolveBenchmarkOutputPath(loaded, profile, strings.TrimSpace(*outputFile), *caseCount)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}
	selector := strings.TrimSpace(*inputFileSelector)
	if selector == "" {
		selector = derivedSelector
	}
	if *emitLocalConfig && strings.TrimSpace(selector) == "" {
		err = fmt.Errorf("cannot derive benchmark input_file from output path %q; pass --input-file-selector explicitly", filepath.ToSlash(outputPath))
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}
	localConfigPath, err := resolveBenchmarkLocalConfigPath(loaded, profile, strings.TrimSpace(*localConfigOut), *emitLocalConfig, *caseCount)
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}

	result, err := benchmarkpack.Generate(benchmarkpack.GenerationOptions{
		Surface:                      profile.Surface,
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
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}

	manifest := benchmarkGenerationManifest{
		SchemaVersion: "researchctl.generate-benchmark/v1",
		RunID:         *runID,
		Command:       commandName,
		Surface:       result.Profile.Surface,
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
	}
	rawManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return fmt.Errorf("marshal benchmark generation manifest: %w", err)
	}
	if err := writeManifestFile(manifestPath, rawManifest); err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}

	if err := store.RecordGeneratedPack(context.Background(), state.GeneratedPack{
		RunID:         *runID,
		PackKey:       strings.TrimSuffix(filepath.Base(result.OutputPath), filepath.Ext(result.OutputPath)),
		ProjectFolder: result.Profile.ProjectFolder,
		SourceKind:    fmt.Sprintf("benchmark/%s", result.Profile.Surface),
		OutputPath:    result.OutputPath,
		ManifestPath:  manifestPath,
		MetadataJSON: state.MetadataJSON(map[string]any{
			"input_file":      result.BenchmarkInputFile,
			"local_config":    result.LocalConfigPath,
			"prompt_version":  result.PromptVersion,
			"experiment_name": result.ExperimentName,
			"case_count":      result.CaseCount,
		}),
	}); err != nil {
		_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
		return err
	}
	for _, artifact := range []state.Artifact{
		{RunID: *runID, Role: "benchmark-cases-csv", AbsolutePath: result.OutputPath},
		{RunID: *runID, Role: "benchmark-generation-manifest", AbsolutePath: manifestPath},
	} {
		if err := store.RecordArtifact(context.Background(), artifact); err != nil {
			_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
			return err
		}
	}
	if strings.TrimSpace(result.LocalConfigPath) != "" {
		if err := store.RecordArtifact(context.Background(), state.Artifact{
			RunID:        *runID,
			Role:         "benchmark-local-config",
			AbsolutePath: result.LocalConfigPath,
		}); err != nil {
			_, _ = finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, err, nil)
			return err
		}
	}

	if _, err := finishRun(context.Background(), store, *runID, commandName, "generate-benchmark", manifestPath, nil, map[string]any{
		"surface":        result.Profile.Surface,
		"output_path":    result.OutputPath,
		"input_file":     result.BenchmarkInputFile,
		"local_config":   result.LocalConfigPath,
		"case_count":     result.CaseCount,
		"prompt_version": result.PromptVersion,
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "benchmark cases generated\nrun_id: %s\nsurface: %s\noutput_file: %s\ninput_file: %s\nlocal_config: %s\ncase_count: %d\nprompt_version: %s\nexperiment_name: %s\nfamily_name: %s\n",
		*runID,
		result.Profile.Surface,
		result.OutputPath,
		result.BenchmarkInputFile,
		firstNonEmptyGenerate(result.LocalConfigPath, "(not emitted)"),
		result.CaseCount,
		result.PromptVersion,
		result.ExperimentName,
		result.FamilyName,
	)
	return nil
}

func resolveBenchmarkOutputPath(loaded researchconfig.Loaded, profile benchmarkpack.SurfaceProfile, raw string, caseCount int) (string, string, error) {
	projectRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, profile.ProjectFolder)))
	raw = strings.TrimSpace(raw)
	if raw == "" {
		defaultOutput := profile.DefaultOutputFile
		// Expanded packs intentionally switch to new paths so operators do not
		// silently overwrite the checked-in baseline benchmark files.
		if caseCount > 0 && caseCount != benchmarkpack.BaselineCaseCount(profile.Surface) {
			if expanded := benchmarkpack.DefaultOutputFileSelector(profile.Surface, caseCount); expanded != "" {
				defaultOutput = expanded
			}
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

func resolveBenchmarkLocalConfigPath(loaded researchconfig.Loaded, profile benchmarkpack.SurfaceProfile, raw string, emit bool, caseCount int) (string, error) {
	if !emit && strings.TrimSpace(raw) == "" {
		return "", nil
	}
	if strings.TrimSpace(raw) == "" {
		if caseCount > 0 && caseCount != benchmarkpack.BaselineCaseCount(profile.Surface) {
			return loaded.ResolvePath(benchmarkpack.DefaultExpandedLocalConfigPath(profile.Surface, caseCount)), nil
		}
		return loaded.ResolvePath(profile.DefaultLocalConfigPath), nil
	}
	if filepath.IsAbs(filepath.FromSlash(raw)) {
		return filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw))), nil
	}
	if strings.HasPrefix(raw, "research/") || strings.HasPrefix(raw, ".tmp/") {
		return loaded.ResolvePath(raw), nil
	}
	return loaded.ResolvePath(filepath.ToSlash(filepath.Join("research", "config", raw))), nil
}
