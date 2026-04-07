package planner

import (
	"fmt"
	"path/filepath"
	"strings"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func BenchmarkReportFilename(loaded researchconfig.Loaded, phase, reportDir, runID string) string {
	return compactRunScopedReportFilename(
		phase,
		runID,
		reportDir,
		loaded.ResolvePath(benchmarkGenericReportDir(loaded.Config.Reports, phase)),
	)
}

func PairReportFilename(loaded researchconfig.Loaded, phase, reportDir, runID string) string {
	return compactRunScopedReportFilename(
		phase,
		runID,
		reportDir,
		loaded.ResolvePath(loaded.Config.Reports.ResearchDir),
	)
}

func compactRunScopedReportFilename(phase, runID, reportDir, genericReportDir string) string {
	phase = strings.TrimSpace(phase)
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = defaultTimestampRunID()
	}
	if reportDirIsDedicated(reportDir, genericReportDir) {
		return runID + ".md"
	}
	if runIDEncodesPhase(phase, runID) {
		return runID + ".md"
	}
	return fmt.Sprintf("%s_%s.md", phase, runID)
}

func benchmarkGenericReportDir(reports researchconfig.ReportsConfig, phase string) string {
	phase = strings.TrimSpace(phase)
	switch {
	case phase == "nd", strings.HasSuffix(phase, "-nd"):
		return reports.NDDir
	default:
		return reports.BenchmarkDir
	}
}

func reportDirIsDedicated(reportDir, genericReportDir string) bool {
	return !reportDirMatchesGeneric(reportDir, genericReportDir)
}

func reportDirMatchesGeneric(reportDir, genericReportDir string) bool {
	reportDir = normalizeReportDir(reportDir)
	genericReportDir = normalizeReportDir(genericReportDir)
	if reportDir == "" || genericReportDir == "" {
		return false
	}
	if reportDir == genericReportDir {
		return true
	}
	return strings.EqualFold(filepath.Base(filepath.FromSlash(reportDir)), filepath.Base(filepath.FromSlash(genericReportDir)))
}

func normalizeReportDir(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
}

func runIDEncodesPhase(phase, runID string) bool {
	runID = strings.TrimSpace(runID)
	for _, prefix := range phasePrefixes(phase) {
		if runID == prefix ||
			strings.HasPrefix(runID, prefix+"-") ||
			strings.HasPrefix(runID, prefix+"_") ||
			strings.HasPrefix(runID, prefix+".") {
			return true
		}
	}
	return false
}

func phasePrefixes(phase string) []string {
	phase = strings.TrimSpace(phase)
	if phase == "" {
		return nil
	}
	prefixes := []string{phase}
	for _, suffix := range []string{"-direct", "-nd", "-pair"} {
		if strings.HasSuffix(phase, suffix) {
			prefixes = append(prefixes, strings.TrimSuffix(phase, suffix))
		}
	}
	seen := make(map[string]struct{}, len(prefixes))
	compact := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		compact = append(compact, prefix)
	}
	return compact
}
