package main

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestValidateReportSummarySelection(t *testing.T) {
	tests := []struct {
		name        string
		summary     string
		summaryGlob string
		latest      int
		wantErr     bool
	}{
		{name: "explicit summary", summary: "a.csv", wantErr: false},
		{name: "summary glob", summaryGlob: "dir/*.csv", wantErr: false},
		{name: "latest", latest: 2, wantErr: false},
		{name: "missing selector", wantErr: true},
		{name: "multiple selectors", summary: "a.csv", latest: 1, wantErr: true},
		{name: "multiple selector kinds", summaryGlob: "dir/*.csv", latest: 1, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateReportSummarySelection(tc.summary, tc.summaryGlob, tc.latest)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateReportSummarySelection() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestResolveReportSummaryGlobs(t *testing.T) {
	root := t.TempDir()
	loaded := researchconfig.Loaded{WorkspaceRoot: root}

	first := filepath.Join(root, "research", "artifacts", "result_research", "waves", "phase1_one.csv")
	second := filepath.Join(root, "research", "artifacts", "result_research", "waves", "phase1_two.csv")
	for _, path := range []string{first, second} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte("run_id\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	got, err := resolveReportSummaryGlobs(loaded, []string{
		filepath.ToSlash(filepath.Join("research", "artifacts", "result_research", "waves", "phase1_*.csv")),
	})
	if err != nil {
		t.Fatalf("resolveReportSummaryGlobs() error = %v", err)
	}
	want := []string{filepath.ToSlash(first), filepath.ToSlash(second)}
	if !slices.Equal(got, want) {
		t.Fatalf("resolveReportSummaryGlobs() = %#v, want %#v", got, want)
	}
}

func TestResolveReportLatestSummaries(t *testing.T) {
	root := t.TempDir()
	loaded := researchconfig.Loaded{WorkspaceRoot: root}
	benchmarkConfig := researchconfig.Benchmark{
		SummaryDir: filepath.ToSlash(filepath.Join("research", "artifacts", "result_research", "waves")),
	}

	older := filepath.Join(root, "research", "artifacts", "result_research", "waves", "phase1_old.csv")
	middle := filepath.Join(root, "research", "artifacts", "result_research", "waves", "phase1_middle.csv")
	newest := filepath.Join(root, "research", "artifacts", "result_research", "waves", "phase1_new.csv")
	otherPhase := filepath.Join(root, "research", "artifacts", "result_research", "waves", "nd_other.csv")
	for _, path := range []string{older, middle, newest, otherPhase} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(path, []byte("run_id\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	now := time.Now()
	if err := os.Chtimes(older, now.Add(-3*time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatalf("Chtimes(older) error = %v", err)
	}
	if err := os.Chtimes(middle, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("Chtimes(middle) error = %v", err)
	}
	if err := os.Chtimes(newest, now.Add(-1*time.Hour), now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("Chtimes(newest) error = %v", err)
	}
	if err := os.Chtimes(otherPhase, now, now); err != nil {
		t.Fatalf("Chtimes(otherPhase) error = %v", err)
	}

	got, err := resolveReportLatestSummaries(loaded, benchmarkConfig, "phase1", 2)
	if err != nil {
		t.Fatalf("resolveReportLatestSummaries() error = %v", err)
	}
	want := []string{filepath.ToSlash(newest), filepath.ToSlash(middle)}
	if !slices.Equal(got, want) {
		t.Fatalf("resolveReportLatestSummaries() = %#v, want %#v", got, want)
	}
}

func TestDefaultReportFilename(t *testing.T) {
	if got := defaultReportFilename("benchmark", "phase1", "phase1-expanded-100-20260404"); got != "benchmark-phase1-expanded-100-20260404.md" {
		t.Fatalf("defaultReportFilename(benchmark) = %q", got)
	}
	if got := defaultReportFilename("meta", "phase1", ""); !strings.HasPrefix(got, "meta-phase1-") || !strings.HasSuffix(got, ".md") {
		t.Fatalf("defaultReportFilename(meta) = %q", got)
	}
	if got := defaultReportFilename("cases", "bridge-import-final-research", ""); !strings.HasPrefix(got, "cases-bridge-import-final-research-") || !strings.HasSuffix(got, ".md") {
		t.Fatalf("defaultReportFilename(cases) = %q", got)
	}
}

func TestResolveBenchmarkParallelJobs(t *testing.T) {
	tests := []struct {
		name      string
		config    researchconfig.Benchmark
		jobsFlag  int
		want      int
		wantError bool
	}{
		{
			name:     "uses config when flag omitted",
			config:   researchconfig.Benchmark{ParallelJobs: 3},
			jobsFlag: -1,
			want:     3,
		},
		{
			name:     "cli overrides config",
			config:   researchconfig.Benchmark{ParallelJobs: 3},
			jobsFlag: 5,
			want:     5,
		},
		{
			name:      "rejects missing config value when flag omitted",
			config:    researchconfig.Benchmark{},
			jobsFlag:  -1,
			wantError: true,
		},
		{
			name:      "rejects invalid cli value",
			config:    researchconfig.Benchmark{ParallelJobs: 3},
			jobsFlag:  0,
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveBenchmarkParallelJobs(tc.config, tc.jobsFlag)
			if tc.wantError {
				if err == nil {
					t.Fatalf("resolveBenchmarkParallelJobs() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveBenchmarkParallelJobs() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveBenchmarkParallelJobs() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestLoadResearchConfigFailsWhenExplicitLocalConfigMissing(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "research", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) error = %v", err)
	}

	defaultConfig := `schema_version: researchctl.config/v1
paths:
  artifact_root: research/artifacts
state:
  db_path: research/artifacts/result_research/research_db/research_db.sqlite
transports:
  local-compatible:
    provider: compatible
    base_url: http://localhost:11434
families:
  local-compatible:
    transport: local-compatible
    models: [gpt-oss:20b]
benchmarks:
  phase1:
    project_folder: hilbert-ai-verification-benchmark-v2-held-out
    input_kind: cases
    input_file: cases.csv
    prompt_version: hilbert-ai-verification-benchmark-v1.3
    families: [local-compatible]
    repeats: 1
    summary_dir: research/artifacts/result_research/waves
    report_dir: research/result_research_report_v1/direct
reports:
  benchmark_dir: research/result_research_report_v1/direct
  nd_dir: research/result_research_report_v1/nd
  research_dir: research/result_research_report_v1/pair
  meta_dir: research/result_research_report_v1/summary/meta
  cases_dir: research/result_research_report_v1/summary/cases
db:
  project_folder: hilbert-ai-verification-benchmark-nd-v1
`
	if err := os.WriteFile(filepath.Join(configDir, "default.yaml"), []byte(defaultConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(default.yaml) error = %v", err)
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = fs.String("local-config", "", "")
	if err := fs.Parse([]string{"--local-config", "research/config/direct/missing.yaml"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err := loadResearchConfig(fs, root, filepath.Join("research", "config", "default.yaml"), "research/config/direct/missing.yaml")
	if err == nil {
		t.Fatalf("loadResearchConfig() error = nil, want missing explicit local-config error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("loadResearchConfig() error = %v, want missing file message", err)
	}
}
