package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

func runDBOverviewCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("db-overview", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	projectFolder := fs.String("project-folder", "", "Optional research slug filter")
	dbPath := fs.String("db", "", "SQLite research DB path")

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
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	resolvedDBPath := strings.TrimSpace(*dbPath)
	if resolvedDBPath == "" {
		resolvedDBPath = loaded.ResolvePath(loaded.Config.State.DBPath)
	} else {
		resolvedDBPath = loaded.ResolvePath(resolvedDBPath)
	}

	overview, err := researchdb.LoadOverview(commandContext(), resolvedDBPath, researchdb.OverviewOptions{
		ProjectFolder: strings.TrimSpace(*projectFolder),
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research db overview\ndb_path: %s\nscope: %s\ntotal_researches: %d\ntotal_llm_models: %d\ntotal_certificates: %d\ntotal_generated_packs: %d\ntotal_exports: %d\ntotal_hypothesis_sets: %d\ntotal_chain_summary_rows: %d\nglobal_runs: total=%d running=%d completed=%d partial_failed=%d failed=%d aborted=%d\nglobal_jobs: total=%d running=%d completed=%d skipped=%d failed=%d aborted=%d\n",
		filepath.ToSlash(overview.DBPath),
		overview.Scope,
		overview.TotalResearches,
		overview.TotalLLMModels,
		overview.TotalCertificates,
		overview.TotalGeneratedPacks,
		overview.TotalExports,
		overview.TotalHypothesisSets,
		overview.TotalChainSummaryRows,
		overview.GlobalRuns.Total,
		overview.GlobalRuns.Running,
		overview.GlobalRuns.Completed,
		overview.GlobalRuns.PartialFailed,
		overview.GlobalRuns.Failed,
		overview.GlobalRuns.Aborted,
		overview.GlobalJobs.Total,
		overview.GlobalJobs.Running,
		overview.GlobalJobs.Completed,
		overview.GlobalJobs.Skipped,
		overview.GlobalJobs.Failed,
		overview.GlobalJobs.Aborted,
	)
	if strings.TrimSpace(overview.LatestRun.RunID) != "" {
		_, _ = fmt.Fprintf(stdout, "latest_run: %s phase=%s status=%s started=%s finished=%s\n",
			overview.LatestRun.RunID,
			overview.LatestRun.Phase,
			overview.LatestRun.Status,
			firstNonEmptyText(formatDisplayTime(overview.LatestRun.StartedAtUTC), "-"),
			firstNonEmptyText(formatDisplayTime(overview.LatestRun.FinishedAtUTC), "-"),
		)
	}

	if len(overview.Researches) > 0 {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "researches:")
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "slug\ttheorem_files\thistorical\ttheorems\tresult_files\tsummary_files\tresult_row_files\tchain_summary_files\tsummary_rows\tresult_rows\tchain_summary_rows\thypothesis_sets\tlatest_import")
		for _, row := range overview.Researches {
			_, _ = fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
				row.Slug,
				row.TheoremFiles,
				row.HistoricalTheoremFiles,
				row.Theorems,
				row.ResultFiles,
				row.SummaryFiles,
				row.ResultRowFiles,
				row.ChainSummaryFiles,
				row.SummaryRows,
				row.ResultRows,
				row.ChainSummaryRows,
				row.HypothesisSets,
				firstNonEmptyText(formatDisplayTime(row.LatestArtifactImportUTC), "-"),
			)
		}
		_ = tw.Flush()
	}

	if len(overview.ArtifactGroups) > 0 {
		_, _ = fmt.Fprintln(stdout, "")
		_, _ = fmt.Fprintln(stdout, "artifact_groups:")
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "artifact_group\tresult_files\tchain_summary_files\tsummary_rows\tresult_rows\tchain_summary_rows")
		for _, row := range overview.ArtifactGroups {
			_, _ = fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\n", row.ArtifactGroup, row.ResultFiles, row.ChainSummaryFiles, row.SummaryRows, row.ResultRows, row.ChainSummaryRows)
		}
		_ = tw.Flush()
	}
	return nil
}

func runDBRunsCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("db-runs", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", filepath.ToSlash(filepath.Join("research", "config", "default.yaml")), "Path to canonical research YAML config")
	localConfigPath := fs.String("local-config", defaultWorkflowLocalConfigPath(), "Optional local YAML override")
	dbPath := fs.String("db", "", "SQLite research DB path")
	phase := fs.String("phase", "", "Optional phase filter")
	status := fs.String("status", "", "Optional status filter")
	limit := fs.Int("limit", 20, "Maximum number of runs to show")

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
	loaded, err := loadResearchConfig(fs, workspaceRoot, *configPath, *localConfigPath)
	if err != nil {
		return err
	}
	resolvedDBPath := strings.TrimSpace(*dbPath)
	if resolvedDBPath == "" {
		resolvedDBPath = loaded.ResolvePath(loaded.Config.State.DBPath)
	} else {
		resolvedDBPath = loaded.ResolvePath(resolvedDBPath)
	}

	runs, err := researchdb.ListRuns(commandContext(), resolvedDBPath, researchdb.ListRunsOptions{
		Phase:  strings.TrimSpace(*phase),
		Status: strings.TrimSpace(*status),
		Limit:  *limit,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "research db runs\ndb_path: %s\nphase_filter: %s\nstatus_filter: %s\nlimit: %d\nrun_count: %d\n",
		filepath.ToSlash(resolvedDBPath),
		firstNonEmptyText(strings.TrimSpace(*phase), "-"),
		firstNonEmptyText(strings.TrimSpace(*status), "-"),
		*limit,
		len(runs),
	)
	if len(runs) == 0 {
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "")
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "run_id\tphase\tstatus\tstarted_at\tfinished_at\tjobs[c/s/f/a/r]\tartifacts\tcommand")
	for _, row := range runs {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d/%d/%d/%d/%d\t%d\t%s\n",
			row.RunID,
			firstNonEmptyText(row.Phase, "-"),
			firstNonEmptyText(row.Status, "-"),
			firstNonEmptyText(formatDisplayTime(row.StartedAtUTC), "-"),
			firstNonEmptyText(formatDisplayTime(row.FinishedAtUTC), "-"),
			row.CompletedJobs,
			row.SkippedJobs,
			row.FailedJobs,
			row.AbortedJobs,
			row.RunningJobs,
			row.ArtifactCount,
			firstNonEmptyText(row.Command, "-"),
		)
	}
	_ = tw.Flush()
	return nil
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatDisplayTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return parsed.Local().Format("2006-01-02 15:04:05 -07:00")
}
