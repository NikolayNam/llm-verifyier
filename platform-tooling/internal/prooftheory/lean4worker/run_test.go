package lean4worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureProjectScaffold(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	if err := EnsureProjectScaffold(projectDir); err != nil {
		t.Fatalf("EnsureProjectScaffold() error = %v", err)
	}
	required := []string{
		filepath.Join(projectDir, "lean-toolchain"),
		filepath.Join(projectDir, "lakefile.lean"),
		filepath.Join(projectDir, "CollabSphereLean.lean"),
		filepath.Join(projectDir, "CollabSphereLean", "Basic.lean"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
	}
}

func TestRunJob_MissingLakeBinaryProducesOperationalFailureReport(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	resultRoot := filepath.Join(root, "result")

	result, err := RunJob(context.Background(), Job{
		Name:      "identity",
		Statement: "P -> P",
	}, Options{
		ProjectDir: projectDir,
		ResultRoot: resultRoot,
		RunID:      "run-1",
		LakeBinary: "definitely-not-a-real-lake-binary",
	})
	if err != nil {
		t.Fatalf("RunJob() error = %v", err)
	}
	if result.OK {
		t.Fatal("RunJob() OK = true, want false")
	}
	if result.ExitCode != -1 {
		t.Fatalf("RunJob() ExitCode = %d, want -1", result.ExitCode)
	}
	if result.ReportPath == "" {
		t.Fatal("RunJob() ReportPath is empty")
	}
	if _, err := os.Stat(filepath.FromSlash(result.ReportPath)); err != nil {
		t.Fatalf("Stat(report) error = %v", err)
	}
	if _, err := os.Stat(filepath.FromSlash(result.ModulePath)); err != nil {
		t.Fatalf("Stat(module) error = %v", err)
	}

	reportBytes, err := os.ReadFile(filepath.FromSlash(result.ReportPath))
	if err != nil {
		t.Fatalf("ReadFile(report) error = %v", err)
	}
	var report Report
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatalf("Unmarshal(report) error = %v", err)
	}
	if report.Result.ReportPath == "" {
		t.Fatal("report.Result.ReportPath is empty")
	}
}

func TestLeanCommandEnv_IncludesSafeDirectories(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "lean-project")
	packagesDir := filepath.Join(projectDir, ".lake", "packages")
	if err := os.MkdirAll(filepath.Join(packagesDir, "mathlib"), 0o755); err != nil {
		t.Fatalf("MkdirAll(mathlib) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(packagesDir, "plausible"), 0o755); err != nil {
		t.Fatalf("MkdirAll(plausible) error = %v", err)
	}

	env := leanCommandEnv(projectDir)
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "GIT_CONFIG_COUNT=4") {
		t.Fatalf("leanCommandEnv() missing GIT_CONFIG_COUNT=4:\n%s", joined)
	}
	for _, path := range []string{
		filepath.Clean(projectDir),
		filepath.Clean(packagesDir),
		filepath.Clean(filepath.Join(packagesDir, "mathlib")),
		filepath.Clean(filepath.Join(packagesDir, "plausible")),
	} {
		if !strings.Contains(joined, path) {
			t.Fatalf("leanCommandEnv() missing safe directory %q:\n%s", path, joined)
		}
	}
}
