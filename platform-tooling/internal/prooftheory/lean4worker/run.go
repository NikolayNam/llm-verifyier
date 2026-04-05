package lean4worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func RunJob(ctx context.Context, job Job, opts Options) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.ProjectDir == "" {
		return Result{}, fmt.Errorf("lean4 project dir is required")
	}
	if opts.ResultRoot == "" {
		return Result{}, fmt.Errorf("lean4 result root is required")
	}
	if strings.TrimSpace(opts.RunID) == "" {
		return Result{}, fmt.Errorf("lean4 run id is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultRunTimeout
	}
	if strings.TrimSpace(opts.LakeBinary) == "" {
		opts.LakeBinary = DefaultLakeBinary
	}
	if err := EnsureProjectScaffold(opts.ProjectDir); err != nil {
		return Result{}, err
	}

	moduleText, err := RenderModule(job)
	if err != nil {
		return Result{}, err
	}

	jobDirName := sanitizeLeanIdentifier(job.Name)
	generatedDir := filepath.Join(opts.ProjectDir, DefaultGeneratedDir, opts.RunID, jobDirName)
	artifactDir := filepath.Join(opts.ResultRoot, opts.RunID, jobDirName)
	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create generated dir: %w", err)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create artifact dir: %w", err)
	}

	modulePath := filepath.Join(generatedDir, "Generated.lean")
	if err := os.WriteFile(modulePath, []byte(moduleText), 0o644); err != nil {
		return Result{}, fmt.Errorf("write generated lean module: %w", err)
	}

	jobSummaryPath := filepath.Join(artifactDir, "job.json")
	jobSummaryBytes, err := json.MarshalIndent(map[string]any{
		"name":      job.Name,
		"statement": job.Statement,
	}, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("marshal lean job summary: %w", err)
	}
	if err := os.WriteFile(jobSummaryPath, jobSummaryBytes, 0o644); err != nil {
		return Result{}, fmt.Errorf("write lean job summary: %w", err)
	}
	if len(job.Payload) > 0 {
		if err := os.WriteFile(filepath.Join(artifactDir, "payload.json"), job.Payload, 0o644); err != nil {
			return Result{}, fmt.Errorf("write lean payload: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "Generated.lean"), []byte(moduleText), 0o644); err != nil {
		return Result{}, fmt.Errorf("copy lean module to artifacts: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, opts.LakeBinary, "env", "lean", modulePath)
	cmd.Dir = opts.ProjectDir
	cmd.Env = leanCommandEnv(opts.ProjectDir)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	result := Result{
		ArtifactDir: filepath.ToSlash(artifactDir),
		ModulePath:  filepath.ToSlash(modulePath),
		ExitCode:    -1,
	}

	runErr := cmd.Run()
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()
	if runErr == nil {
		result.OK = true
		result.ExitCode = 0
	} else {
		var exitErr *exec.ExitError
		switch {
		case errors.As(runErr, &exitErr):
			result.OK = false
			result.ExitCode = exitErr.ExitCode()
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			result.OK = false
			result.ExitCode = -1
			result.Stderr = appendLeanFailureMessage(result.Stderr, "lean job timed out")
		default:
			result.OK = false
			result.ExitCode = -1
			result.Stderr = appendLeanFailureMessage(result.Stderr, runErr.Error())
		}
	}

	reportPath := filepath.Join(artifactDir, "report.json")
	result.ReportPath = filepath.ToSlash(reportPath)

	report := Report{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		ProjectDir:     filepath.ToSlash(opts.ProjectDir),
		LakeBinary:     opts.LakeBinary,
		JobName:        job.Name,
		Statement:      job.Statement,
		PayloadPresent: len(job.Payload) > 0,
		Result:         result,
	}
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("marshal lean report: %w", err)
	}
	if err := os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		return Result{}, fmt.Errorf("write lean report: %w", err)
	}
	return result, nil
}

func appendLeanFailureMessage(current, message string) string {
	current = strings.TrimSpace(current)
	message = strings.TrimSpace(message)
	switch {
	case current == "":
		return message
	case message == "":
		return current
	default:
		return current + "\n" + message
	}
}

func leanCommandEnv(projectDir string) []string {
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
	if err != nil {
		sort.Strings(dirs)
		return dirs
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		add(filepath.Join(packagesRoot, entry.Name()))
	}
	sort.Strings(dirs)
	return dirs
}
