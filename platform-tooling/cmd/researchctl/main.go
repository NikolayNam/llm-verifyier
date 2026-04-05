package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

var researchctlProcessContext = context.Background()

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	researchctlProcessContext = ctx
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	filteredArgs, restoreWorkspaceRootOverride, err := applyWorkspaceRootOverride(args)
	if err != nil {
		return err
	}
	defer restoreWorkspaceRootOverride()
	args = filteredArgs
	if len(args) == 0 {
		printUsage(stderr)
		return flag.ErrHelp
	}
	switch args[0] {
	case "phase1":
		return runBenchmarkSurface("phase1", args[1:], stdout, stderr)
	case "compositional-assumption-import":
		return runBenchmarkSurface("compositional-assumption-import", args[1:], stdout, stderr)
	case "bridge-import-final-research":
		return runBenchmarkSurface("bridge-import-final-research", args[1:], stdout, stderr)
	case "bridge-only-authoring":
		return runBenchmarkSurface("bridge-only-authoring", args[1:], stdout, stderr)
	case "gold-final-composition-only":
		return runBenchmarkSurface("gold-final-composition-only", args[1:], stdout, stderr)
	case "compositional-depth-ladder":
		return runBenchmarkSurface("compositional-depth-ladder", args[1:], stdout, stderr)
	case "compositional-branching":
		return runBenchmarkSurface("compositional-branching", args[1:], stdout, stderr)
	case "compositional-mixed-family-reuse":
		return runBenchmarkSurface("compositional-mixed-family-reuse", args[1:], stdout, stderr)
	case "compositional-mixed-family-gold-first":
		return runBenchmarkSurface("compositional-mixed-family-gold-first", args[1:], stdout, stderr)
	case "compositional-mixed-family-gold-first-phase2":
		return runBenchmarkSurface("compositional-mixed-family-gold-first-phase2", args[1:], stdout, stderr)
	case "compositional-mixed-family-gold-first-hard":
		return runBenchmarkSurface("compositional-mixed-family-gold-first-hard", args[1:], stdout, stderr)
	case "compositional-mixed-family-semi-gold-stable":
		return runBenchmarkSurface("compositional-mixed-family-semi-gold-stable", args[1:], stdout, stderr)
	case "compositional-mixed-family-semi-gold-frontier":
		return runBenchmarkSurface("compositional-mixed-family-semi-gold-frontier", args[1:], stdout, stderr)
	case "phase2":
		return runPhase2Surface(args[1:], stdout, stderr)
	case "phase3":
		return runPhase3Surface(args[1:], stdout, stderr)
	case "nd":
		return runBenchmarkSurface("nd", args[1:], stdout, stderr)
	case "generate":
		return runGenerateSurface(args[1:], stdout, stderr)
	case "prompt":
		return runPromptSurface(args[1:], stdout, stderr)
	case "export":
		return runExportSurface(args[1:], stdout, stderr)
	case "db":
		return runDBSurface(args[1:], stdout, stderr)
	case "report":
		return runReportSurface(args[1:], stdout, stderr)
	case "analyze":
		return runAnalyzeSurface(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown researchctl command %q", args[0])
	}
}

func writeManifestFile(path string, raw []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("research manifest path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create research manifest dir: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write research manifest %s: %w", path, err)
	}
	return nil
}

func writeBenchmarkManifest(plan *planner.BenchmarkPlan) error {
	if plan == nil {
		return fmt.Errorf("benchmark plan is required")
	}
	raw, err := plan.MarshalIndentedJSON()
	if err != nil {
		return fmt.Errorf("marshal research manifest: %w", err)
	}
	return writeManifestFile(plan.ManifestPath, raw)
}

func startRun(ctx context.Context, store *state.Store, loaded researchconfig.Loaded, runID, command, phase, manifestPath string) error {
	if err := store.UpsertRun(controlContext(ctx), state.Run{
		RunID:             runID,
		Command:           command,
		Phase:             phase,
		Status:            "running",
		ManifestPath:      manifestPath,
		ConfigFingerprint: loaded.Fingerprint,
		StartedAtUTC:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	return store.SaveRunConfig(controlContext(ctx), runID, loaded.ConfigPath, loaded.LocalConfigPath, loaded.ResolvedJSON, loaded.Fingerprint)
}

func finishRun(ctx context.Context, store *state.Store, runID, command, phase, manifestPath string, runErr error, metadata map[string]any) (string, error) {
	status := "completed"
	if isInterruptedCommand(ctx, runErr) {
		status = "aborted"
	} else if runErr != nil {
		status = "failed"
	}
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	if err := store.UpsertRun(controlContext(ctx), state.Run{
		RunID:         runID,
		Command:       command,
		Phase:         phase,
		Status:        status,
		ManifestPath:  manifestPath,
		FinishedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Error:         errText,
		MetadataJSON:  state.MetadataJSON(metadata),
	}); err != nil {
		if runErr != nil {
			return status, fmt.Errorf("%w; finalize run: %v", runErr, err)
		}
		return status, err
	}
	return status, runErr
}

func normalizePhaseName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "phase1", "phase-1", "direct":
		return "phase1"
	case "nd", "phase2", "phase-2":
		return "nd"
	case "compositional-assumption-import", "compositional_assumption_import":
		return "compositional-assumption-import"
	case "bridge-import-final-research", "bridge_import_final_research":
		return "bridge-import-final-research"
	case "bridge-only-authoring", "bridge_only_authoring":
		return "bridge-only-authoring"
	case "gold-final-composition-only", "gold_final_composition_only":
		return "gold-final-composition-only"
	case "compositional-depth-ladder", "compositional_depth_ladder":
		return "compositional-depth-ladder"
	case "compositional-branching", "compositional_branching":
		return "compositional-branching"
	case "compositional-mixed-family-reuse", "compositional_mixed_family_reuse":
		return "compositional-mixed-family-reuse"
	case "compositional-mixed-family-gold-first", "compositional_mixed_family_gold_first":
		return "compositional-mixed-family-gold-first"
	case "compositional-mixed-family-gold-first-phase2", "compositional_mixed_family_gold_first_phase2":
		return "compositional-mixed-family-gold-first-phase2"
	case "compositional-mixed-family-gold-first-hard", "compositional_mixed_family_gold_first_hard":
		return "compositional-mixed-family-gold-first-hard"
	case "compositional-mixed-family-semi-gold-stable", "compositional_mixed_family_semi_gold_stable":
		return "compositional-mixed-family-semi-gold-stable"
	case "compositional-mixed-family-semi-gold-frontier", "compositional_mixed_family_semi_gold_frontier":
		return "compositional-mixed-family-semi-gold-frontier"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func timestampRunID() string {
	return time.Now().Local().Format("20060102T150405-0700")
}

func commandContext() context.Context {
	if researchctlProcessContext == nil {
		return context.Background()
	}
	return researchctlProcessContext
}

func controlContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, strings.TrimSpace(`
usage: go -C platform-tooling run ./cmd/researchctl [--workspace-root PATH] <command>

global options:
  --workspace-root <path>   override detected workspace root (env: COLLABSPHERE_WORKSPACE_ROOT, RESEARCH_WORKSPACE_ROOT)

commands:
  phase1 plan|run
  compositional-assumption-import plan|run
  bridge-import-final-research plan|run
  bridge-only-authoring plan|run
  gold-final-composition-only plan|run
  compositional-depth-ladder plan|run
  compositional-branching plan|run
  compositional-mixed-family-reuse plan|run
  compositional-mixed-family-gold-first plan|run
  compositional-mixed-family-gold-first-phase2 plan|run
  compositional-mixed-family-gold-first-hard plan|run
  compositional-mixed-family-semi-gold-stable plan|run
  compositional-mixed-family-semi-gold-frontier plan|run
  phase2 plan|run
  phase3 plan|run
  phase3 compare plan|run
  nd plan|run
  generate lean bootstrap|job|cases
  generate benchmark cases
  generate compositional cases
  generate bridge-final hypotheses
  prompt manifest|lint
  export lean-to-benchmark
  db sync|prune|overview|runs
  report benchmark
  report research
  report meta
  report cases
  analyze gold-first-phase2
  analyze gold-first-phase2-hard-pack
  analyze lean-kernel-parity
`)+"\n")
}
