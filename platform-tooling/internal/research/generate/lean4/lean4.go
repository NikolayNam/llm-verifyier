package lean4

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	lean4worker "github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

type BootstrapResult struct {
	WorkspaceDir string
}

func WorkspaceDir(loaded researchconfig.Loaded, runID string) string {
	return loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Generate.Lean4.WorkspaceRoot, runID, loaded.Config.Generate.Lean4.ProjectFolder, lean4worker.DefaultProjectDir)))
}

func Bootstrap(ctx context.Context, loaded researchconfig.Loaded, runID string) (BootstrapResult, error) {
	projectDir := WorkspaceDir(loaded, runID)
	if err := lean4worker.EnsureProjectScaffold(projectDir); err != nil {
		return BootstrapResult{}, err
	}
	if err := runLakeCommand(ctx, loaded, projectDir, "update"); err != nil {
		return BootstrapResult{}, err
	}
	if err := runLakeCommand(ctx, loaded, projectDir, "exe", "cache", "get"); err != nil {
		return BootstrapResult{}, err
	}
	if err := runLakeCommand(ctx, loaded, projectDir, "build"); err != nil {
		return BootstrapResult{}, err
	}
	return BootstrapResult{WorkspaceDir: filepath.ToSlash(projectDir)}, nil
}

func RunJob(ctx context.Context, loaded researchconfig.Loaded, runID, name, statement string, payload []byte) (lean4worker.Result, string, error) {
	projectDir := WorkspaceDir(loaded, runID)
	if err := lean4worker.EnsureProjectScaffold(projectDir); err != nil {
		return lean4worker.Result{}, "", err
	}
	artifactRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder)))
	resultRoot := filepath.Join(artifactRoot, lean4worker.DefaultResultDir)
	timeout, err := parseLeanTimeout(loaded)
	if err != nil {
		return lean4worker.Result{}, "", err
	}
	result, err := lean4worker.RunJob(ctx, lean4worker.Job{
		Name:      name,
		Statement: statement,
		Payload:   payload,
	}, lean4worker.Options{
		ProjectDir: projectDir,
		ResultRoot: resultRoot,
		RunID:      runID,
		LakeBinary: loaded.Config.Generate.Lean4.LakeBinary,
		Timeout:    timeout,
	})
	if err != nil {
		return lean4worker.Result{}, "", err
	}
	return result, filepath.ToSlash(projectDir), nil
}

func GenerateCases(loaded researchconfig.Loaded, config lean4worker.HypothesisGenerationConfig) (string, string, string, error) {
	artifactRoot := loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder)))
	return lean4worker.GenerateHypothesisArtifacts(artifactRoot, config)
}

func ExportToBenchmark(loaded researchconfig.Loaded, sourceFile, outputFile string, exportConfig lean4worker.BenchmarkCaseExportConfig) (int, error) {
	if strings.TrimSpace(sourceFile) == "" {
		sourceFile = filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.ProjectFolder))), lean4worker.DefaultTheoremsDir, "theorems.csv")
	} else {
		sourceFile = loaded.ResolvePath(sourceFile)
	}
	if strings.TrimSpace(outputFile) == "" {
		outputFile = filepath.Join(loaded.ResolvePath(filepath.ToSlash(filepath.Join(loaded.Config.Paths.ArtifactRoot, loaded.Config.Generate.Lean4.BenchmarkProjectFolder))), filepath.FromSlash(loaded.Config.Generate.Lean4.ExportOutputFile))
	} else {
		outputFile = loaded.ResolvePath(outputFile)
	}
	return lean4worker.ExportBenchmarkCasesCSV(sourceFile, outputFile, exportConfig)
}

func parseLeanTimeout(loaded researchconfig.Loaded) (duration time.Duration, err error) {
	return time.ParseDuration(loaded.Config.Generate.Lean4.Timeout)
}

func runLakeCommand(ctx context.Context, loaded researchconfig.Loaded, projectDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, loaded.Config.Generate.Lean4.LakeBinary, args...)
	cmd.Dir = projectDir
	cmd.Env = lakeCommandEnv(projectDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %s %s: %w\n%s", loaded.Config.Generate.Lean4.LakeBinary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func lakeCommandEnv(projectDir string) []string {
	base := make([]string, 0, len(os.Environ())+8)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GIT_CONFIG_COUNT=") ||
			strings.HasPrefix(entry, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(entry, "GIT_CONFIG_VALUE_") {
			continue
		}
		base = append(base, entry)
	}
	dirs := gitSafeDirectories(projectDir)
	if len(dirs) == 0 {
		return base
	}
	base = append(base, fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(dirs)))
	for idx, dir := range dirs {
		base = append(base,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=safe.directory", idx),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", idx, dir),
		)
	}
	return base
}

func gitSafeDirectories(projectDir string) []string {
	seen := map[string]struct{}{}
	dirs := make([]string, 0, 8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
	}
	add(projectDir)
	packagesRoot := filepath.Join(projectDir, ".lake", "packages")
	add(packagesRoot)
	entries, err := os.ReadDir(packagesRoot)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				add(filepath.Join(packagesRoot, entry.Name()))
			}
		}
	}
	sort.Strings(dirs)
	return dirs
}
