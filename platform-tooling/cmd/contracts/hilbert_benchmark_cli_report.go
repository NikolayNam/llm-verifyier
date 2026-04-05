package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Report commands share the same summary/model-catalog loading path. Keeping
// them together isolates read-only report assembly from the benchmark run CLI.
func runHilbertBenchmarkReportCommandImpl(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := defaultBenchmarkInputFile(defaultProjectFolder)

	fs := flag.NewFlagSet("hilbert-benchmark-report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input file filter relative to the benchmark project folder")
	casesPath := fs.String("cases", "", "Optional path to the benchmark input CSV used for metadata lookups")
	summaryArg := fs.String("summary", "", "Comma-separated list of benchmark summary CSV files")
	modelCatalogArg := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")
	outPath := fs.String("out", "", "Optional markdown output path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedProjectFolder := strings.TrimSpace(*projectFolder)
	if resolvedProjectFolder == "" {
		resolvedProjectFolder = defaultProjectFolder
	}
	resolvedCasesFile := canonicalBenchmarkInputFile(resolvedProjectFolder, strings.TrimSpace(*casesFile))
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)

	summaryPaths := resolveBenchmarkReportSummaryPaths(workspaceRoot, benchmarkRoot, strings.TrimSpace(*summaryArg))
	rows, err := loadBenchmarkRunSummaryRowsFromPaths(summaryPaths)
	if err != nil {
		return err
	}
	rows = applyBenchmarkReportInputPathOverride(rows, workspaceRoot, resolvedProjectFolder, strings.TrimSpace(*casesPath))
	rows = filterBenchmarkRunSummaryRows(rows, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)

	modelCatalog, modelCatalogPath, err := loadBenchmarkReportModelCatalog(workspaceRoot, benchmarkRoot, strings.TrimSpace(*modelCatalogArg))
	if err != nil {
		return err
	}
	reports, err := buildBenchmarkCasePackReports(rows, resolvedProjectFolder, modelCatalog)
	if err != nil {
		return err
	}

	report := renderBenchmarkMarkdownReport(summaryPaths, resolvedProjectFolder, resolvedCasesFile, modelCatalogPath, reports, time.Now().UTC())
	return writeBenchmarkMarkdownOutput(stdout, *outPath, report, "hilbert benchmark markdown report written")
}

func runHilbertBenchmarkResearchReportCommandImpl(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := defaultBenchmarkInputFile(defaultProjectFolder)

	fs := flag.NewFlagSet("hilbert-benchmark-research-report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input file filter relative to the benchmark project folder")
	casesPath := fs.String("cases", "", "Optional path to the benchmark input CSV used for metadata lookups")
	summaryArg := fs.String("summary", "", "Comma-separated list of benchmark summary CSV files")
	modelCatalogArg := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")
	outPath := fs.String("out", "", "Optional markdown output path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedProjectFolder := strings.TrimSpace(*projectFolder)
	if resolvedProjectFolder == "" {
		resolvedProjectFolder = defaultProjectFolder
	}
	resolvedCasesFile := canonicalBenchmarkInputFile(resolvedProjectFolder, strings.TrimSpace(*casesFile))
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)

	summaryPaths := resolveBenchmarkReportSummaryPaths(workspaceRoot, benchmarkRoot, strings.TrimSpace(*summaryArg))
	rows, err := loadBenchmarkRunSummaryRowsFromPaths(summaryPaths)
	if err != nil {
		return err
	}
	rows = applyBenchmarkReportInputPathOverride(rows, workspaceRoot, resolvedProjectFolder, strings.TrimSpace(*casesPath))
	rows = filterBenchmarkRunSummaryRows(rows, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)

	modelCatalog, modelCatalogPath, err := loadBenchmarkReportModelCatalog(workspaceRoot, benchmarkRoot, strings.TrimSpace(*modelCatalogArg))
	if err != nil {
		return err
	}
	reports, err := buildBenchmarkCasePackReports(rows, resolvedProjectFolder, modelCatalog)
	if err != nil {
		return err
	}

	report, err := renderBenchmarkResearchMarkdownReport(summaryPaths, resolvedProjectFolder, resolvedCasesFile, modelCatalogPath, reports, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeBenchmarkMarkdownOutput(stdout, *outPath, report, "hilbert benchmark research report written")
}

func runHilbertBenchmarkMetaReportCommandImpl(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := defaultBenchmarkInputFile(defaultProjectFolder)

	fs := flag.NewFlagSet("hilbert-benchmark-meta-report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input file filter relative to the benchmark project folder")
	casesPath := fs.String("cases", "", "Optional path to the benchmark input CSV used for metadata lookups")
	summaryArg := fs.String("summary", "", "Comma-separated list of benchmark summary CSV files")
	modelCatalogArg := fs.String("model-catalog", "", "Optional benchmark model catalog CSV")
	outPath := fs.String("out", "", "Optional markdown output path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedProjectFolder := strings.TrimSpace(*projectFolder)
	if resolvedProjectFolder == "" {
		resolvedProjectFolder = defaultProjectFolder
	}
	resolvedCasesFile := canonicalBenchmarkInputFile(resolvedProjectFolder, strings.TrimSpace(*casesFile))
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)

	summaryPaths := resolveBenchmarkReportSummaryPaths(workspaceRoot, benchmarkRoot, strings.TrimSpace(*summaryArg))
	rows, err := loadBenchmarkRunSummaryRowsFromPaths(summaryPaths)
	if err != nil {
		return err
	}
	rows = applyBenchmarkReportInputPathOverride(rows, workspaceRoot, resolvedProjectFolder, strings.TrimSpace(*casesPath))
	rows = filterBenchmarkRunSummaryRows(rows, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)

	modelCatalog, modelCatalogPath, err := loadBenchmarkReportModelCatalog(workspaceRoot, benchmarkRoot, strings.TrimSpace(*modelCatalogArg))
	if err != nil {
		return err
	}
	report, err := renderBenchmarkMetaMarkdownReport(summaryPaths, resolvedProjectFolder, resolvedCasesFile, modelCatalogPath, rows, modelCatalog, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeBenchmarkMarkdownOutput(stdout, *outPath, report, "hilbert benchmark meta report written")
}

func runHilbertBenchmarkCaseReportCommandImpl(args []string, stdout, stderr io.Writer) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	defaultProjectFolder := firstNonEmptyEnv("HILBERT_BENCHMARK_PROJECT_FOLDER")
	if defaultProjectFolder == "" {
		defaultProjectFolder = "hilbert-ai-verification-benchmark-v1"
	}
	defaultCasesFile := defaultBenchmarkInputFile(defaultProjectFolder)

	fs := flag.NewFlagSet("hilbert-benchmark-case-report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectFolder := fs.String("project-folder", defaultProjectFolder, "Benchmark project folder under research/artifacts")
	casesFile := fs.String("cases-file", defaultCasesFile, "Benchmark input file filter relative to the benchmark project folder")
	casesPath := fs.String("cases", "", "Optional path to the benchmark input CSV used for metadata lookups")
	summaryArg := fs.String("summary", "", "Comma-separated list of benchmark summary CSV files")
	outPath := fs.String("out", "", "Optional markdown output path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	resolvedProjectFolder := strings.TrimSpace(*projectFolder)
	if resolvedProjectFolder == "" {
		resolvedProjectFolder = defaultProjectFolder
	}
	resolvedCasesFile := canonicalBenchmarkInputFile(resolvedProjectFolder, strings.TrimSpace(*casesFile))
	benchmarkRoot := benchmarkProjectRoot(workspaceRoot, resolvedProjectFolder)

	summaryPaths := resolveBenchmarkReportSummaryPaths(workspaceRoot, benchmarkRoot, strings.TrimSpace(*summaryArg))
	rows, err := loadBenchmarkRunSummaryRowsFromPaths(summaryPaths)
	if err != nil {
		return err
	}
	rows = applyBenchmarkReportInputPathOverride(rows, workspaceRoot, resolvedProjectFolder, strings.TrimSpace(*casesPath))
	rows = filterBenchmarkRunSummaryRows(rows, workspaceRoot, resolvedProjectFolder, resolvedCasesFile)

	report, err := renderBenchmarkCaseMarkdownReport(summaryPaths, resolvedProjectFolder, resolvedCasesFile, rows, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeBenchmarkMarkdownOutput(stdout, *outPath, report, "hilbert benchmark case report written")
}

func resolveBenchmarkReportSummaryPaths(workspaceRoot, benchmarkRoot, raw string) []string {
	if resolved := parseBenchmarkSummaryArgPaths(workspaceRoot, raw); len(resolved) > 0 {
		return resolved
	}
	return defaultBenchmarkSummaryPaths(benchmarkRoot)
}

func loadBenchmarkReportModelCatalog(workspaceRoot, benchmarkRoot, raw string) ([]benchmarkModelCatalogRow, string, error) {
	modelCatalogPath := resolveBenchmarkArgPath(workspaceRoot, raw)
	if strings.TrimSpace(modelCatalogPath) == "" {
		modelCatalogPath = defaultBenchmarkModelCatalogPath(benchmarkRoot)
		if _, err := os.Stat(modelCatalogPath); err != nil {
			if os.IsNotExist(err) {
				return nil, "", nil
			}
			return nil, "", fmt.Errorf("stat benchmark model catalog: %w", err)
		}
	} else {
		if _, err := os.Stat(modelCatalogPath); err != nil {
			return nil, "", fmt.Errorf("stat benchmark model catalog: %w", err)
		}
	}

	rows, err := loadBenchmarkModelCatalog(modelCatalogPath)
	if err != nil {
		return nil, "", err
	}
	return rows, modelCatalogPath, nil
}

func applyBenchmarkReportInputPathOverride(rows []benchmarkRunSummaryRow, workspaceRoot, projectFolder, raw string) []benchmarkRunSummaryRow {
	resolvedPath := resolveBenchmarkArgPath(workspaceRoot, raw)
	if strings.TrimSpace(resolvedPath) == "" {
		return rows
	}
	overridden := make([]benchmarkRunSummaryRow, len(rows))
	copy(overridden, rows)
	for idx := range overridden {
		overridden[idx].CasesPath = resolvedPath
		if benchmarkProjectUsesTheoremSummaryV2(projectFolder) {
			overridden[idx].TheoremsPath = resolvedPath
		}
	}
	return overridden
}

func writeBenchmarkMarkdownOutput(stdout io.Writer, outPath, markdown, confirmation string) error {
	if strings.TrimSpace(outPath) == "" {
		_, err := io.WriteString(stdout, markdown)
		return err
	}
	resolvedOut := filepath.Clean(outPath)
	if err := os.MkdirAll(filepath.Dir(resolvedOut), 0o755); err != nil {
		return fmt.Errorf("create benchmark report dir: %w", err)
	}
	if err := os.WriteFile(resolvedOut, []byte(markdown), 0o644); err != nil {
		return fmt.Errorf("write benchmark report: %w", err)
	}
	_, err := fmt.Fprintf(stdout, "%s\npath: %s\n", confirmation, normalizeBenchmarkPath(resolvedOut))
	return err
}
