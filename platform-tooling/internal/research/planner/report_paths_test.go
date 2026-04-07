package planner

import (
	"path/filepath"
	"testing"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func TestBenchmarkReportFilenameUsesRunIDInDedicatedDir(t *testing.T) {
	loaded := researchconfig.Loaded{
		WorkspaceRoot: filepath.Clean(`C:\repo`),
		Config: researchconfig.Config{
			Reports: researchconfig.ReportsConfig{
				BenchmarkDir: "research/result_research_report_v1/direct",
				NDDir:        "research/result_research_report_v1/nd",
			},
		},
	}
	reportDir := filepath.Join(loaded.WorkspaceRoot, "research", "result_research_report_v1", "bridge-import-final-research")
	got := BenchmarkReportFilename(loaded, "bridge-import-final-research", reportDir, "bridge-import-final-research-phase1-20260404")
	if got != "bridge-import-final-research-phase1-20260404.md" {
		t.Fatalf("BenchmarkReportFilename(dedicated) = %q", got)
	}
}

func TestBenchmarkReportFilenameDropsDuplicatePhasePrefixInGenericDir(t *testing.T) {
	loaded := researchconfig.Loaded{
		WorkspaceRoot: filepath.Clean(`C:\repo`),
		Config: researchconfig.Config{
			Reports: researchconfig.ReportsConfig{
				BenchmarkDir: "research/result_research_report_v1/direct",
				NDDir:        "research/result_research_report_v1/nd",
			},
		},
	}
	reportDir := filepath.Join(loaded.WorkspaceRoot, "research", "result_research_report", "direct")
	got := BenchmarkReportFilename(loaded, "phase1", reportDir, "phase1-expanded-100-20260404")
	if got != "phase1-expanded-100-20260404.md" {
		t.Fatalf("BenchmarkReportFilename(generic direct) = %q", got)
	}
}

func TestBenchmarkReportFilenameDropsWorkflowPhaseRedundancy(t *testing.T) {
	loaded := researchconfig.Loaded{
		WorkspaceRoot: filepath.Clean(`C:\repo`),
		Config: researchconfig.Config{
			Reports: researchconfig.ReportsConfig{
				BenchmarkDir: "research/result_research_report_v1/direct",
				NDDir:        "research/result_research_report_v1/nd",
			},
		},
	}
	reportDir := filepath.Join(loaded.WorkspaceRoot, "research", "result_research_report", "direct")
	got := BenchmarkReportFilename(loaded, "phase2-direct", reportDir, "phase2-20260404_direct")
	if got != "phase2-20260404_direct.md" {
		t.Fatalf("BenchmarkReportFilename(phase2-direct) = %q", got)
	}
}

func TestPairReportFilenameDropsDuplicatePhasePrefix(t *testing.T) {
	loaded := researchconfig.Loaded{
		WorkspaceRoot: filepath.Clean(`C:\repo`),
		Config: researchconfig.Config{
			Reports: researchconfig.ReportsConfig{
				ResearchDir: "research/result_research_report_v1/pair",
			},
		},
	}
	reportDir := filepath.Join(loaded.WorkspaceRoot, "research", "result_research_report", "pair")
	got := PairReportFilename(loaded, "phase2", reportDir, "phase2-20260404")
	if got != "phase2-20260404.md" {
		t.Fatalf("PairReportFilename() = %q", got)
	}
}
