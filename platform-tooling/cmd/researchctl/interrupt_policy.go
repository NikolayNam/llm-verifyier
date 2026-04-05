package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/runner"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/state"
)

func optionalStringFlag(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func resolveInterruptPolicy(configValue string, override *string) (string, error) {
	policy := researchconfig.NormalizeInterruptPolicy(configValue)
	if override != nil {
		policy = researchconfig.NormalizeInterruptPolicy(*override)
	}
	if err := researchconfig.ValidateInterruptPolicy(policy); err != nil {
		return "", err
	}
	return policy, nil
}

func isInterruptedCommand(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled))
}

func maybeDropInterruptedBenchmarkRun(loaded researchconfig.Loaded, store *state.Store, plan *planner.BenchmarkPlan, result runner.BenchmarkRunResult, progress io.Writer) (bool, error) {
	if plan == nil {
		return false, nil
	}
	policy, err := resolveInterruptPolicy(plan.InterruptPolicy, nil)
	if err != nil {
		return false, err
	}
	meaningful := benchmarkRunHasMeaningfulOutputs(plan, result)
	if !shouldDropInterruptedRun(policy, meaningful) {
		logInterruptCleanup(progress, "benchmark", plan.RunID, policy, meaningful, false)
		return false, nil
	}
	if err := deleteRunArtifacts(loaded.WorkspaceRoot, benchmarkCleanupTargets(plan)); err != nil {
		return false, err
	}
	if err := store.DeleteRun(controlContext(commandContext()), plan.RunID); err != nil {
		return false, err
	}
	logInterruptCleanup(progress, "benchmark", plan.RunID, policy, meaningful, true)
	return true, nil
}

func maybeDropInterruptedWorkflowRun(loaded researchconfig.Loaded, store *state.Store, plan *planner.PairedBenchmarkPlan, directResult, ndResult runner.BenchmarkRunResult, pairReportPath, policy string, progress io.Writer) (bool, error) {
	if plan == nil {
		return false, nil
	}
	policy, err := resolveInterruptPolicy(policy, nil)
	if err != nil {
		return false, err
	}
	meaningful := pairedRunHasMeaningfulOutputs(plan, directResult, ndResult, pairReportPath)
	if !shouldDropInterruptedRun(policy, meaningful) {
		logInterruptCleanup(progress, "workflow", plan.RunID, policy, meaningful, false)
		return false, nil
	}
	targets := []string{plan.ManifestPath, plan.PairModelCatalog, plan.PairReportPath}
	if err := deleteRunArtifacts(loaded.WorkspaceRoot, targets); err != nil {
		return false, err
	}
	if err := store.DeleteRun(controlContext(commandContext()), plan.RunID); err != nil {
		return false, err
	}
	logInterruptCleanup(progress, "workflow", plan.RunID, policy, meaningful, true)
	return true, nil
}

func shouldDropInterruptedRun(policy string, meaningful bool) bool {
	switch researchconfig.NormalizeInterruptPolicy(policy) {
	case researchconfig.InterruptPolicyKeep:
		return false
	case researchconfig.InterruptPolicyDropAlways:
		return true
	case researchconfig.InterruptPolicyDropIfNoResults:
		return !meaningful
	default:
		return false
	}
}

func benchmarkRunHasMeaningfulOutputs(plan *planner.BenchmarkPlan, result runner.BenchmarkRunResult) bool {
	if plan == nil {
		return false
	}
	if result.CompletedJobs > 0 || fileHasContent(result.ReportPath) || csvHasDataRows(plan.SummaryPath) {
		return true
	}
	for _, job := range plan.Jobs {
		if csvHasDataRows(job.ResultsPath) {
			return true
		}
	}
	return false
}

func pairedRunHasMeaningfulOutputs(plan *planner.PairedBenchmarkPlan, directResult, ndResult runner.BenchmarkRunResult, pairReportPath string) bool {
	if plan == nil {
		return false
	}
	return benchmarkRunHasMeaningfulOutputs(plan.Direct, directResult) ||
		benchmarkRunHasMeaningfulOutputs(plan.ND, ndResult) ||
		fileHasContent(firstNonEmptyText(strings.TrimSpace(pairReportPath), plan.PairReportPath))
}

func benchmarkCleanupTargets(plan *planner.BenchmarkPlan) []string {
	if plan == nil {
		return nil
	}
	targets := []string{
		plan.ManifestPath,
		plan.SummaryPath,
		plan.ReportPath,
		plan.ModelCatalogPath,
	}
	for _, job := range plan.Jobs {
		targets = append(targets, job.ResultsPath, filepath.Join(job.RawDir, job.RunID))
	}
	return targets
}

func deleteRunArtifacts(workspaceRoot string, targets []string) error {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		target = filepath.Clean(strings.TrimSpace(target))
		if target == "" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		if err := removePathWithinWorkspace(workspaceRoot, target); err != nil {
			return err
		}
	}
	return nil
}

func removePathWithinWorkspace(workspaceRoot, target string) error {
	workspaceRoot = filepath.Clean(strings.TrimSpace(workspaceRoot))
	target = filepath.Clean(strings.TrimSpace(target))
	if workspaceRoot == "" || target == "" {
		return nil
	}
	rel, err := filepath.Rel(workspaceRoot, target)
	if err != nil {
		return fmt.Errorf("validate cleanup target %q: %w", target, err)
	}
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("refusing to delete path outside workspace root: %s", target)
	}
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat cleanup target %q: %w", target, err)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove cleanup target %q: %w", target, err)
	}
	return nil
}

func csvHasDataRows(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		lineCount++
		if lineCount > 1 {
			return true
		}
	}
	return false
}

func fileHasContent(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() > 0
}

func logInterruptCleanup(w io.Writer, scope, runID, policy string, meaningful, dropped bool) {
	if w == nil {
		return
	}
	action := "kept"
	if dropped {
		action = "dropped"
	}
	_, _ = fmt.Fprintf(w, "interrupt cleanup\nscope: %s\nrun_id: %s\ninterrupt_policy: %s\nmeaningful_outputs: %t\naction: %s\n", scope, runID, researchconfig.NormalizeInterruptPolicy(policy), meaningful, action)
}
