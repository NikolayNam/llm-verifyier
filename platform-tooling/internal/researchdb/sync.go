package researchdb

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const DefaultDBRelativePath = "research/artifacts/result_research/research_db/research_db.sqlite"

const artifactsRelativeRoot = "research/artifacts"

type SyncOptions struct {
	WorkspaceRoot string
	ProjectFolder string
}

type SyncResult struct {
	DBPath             string
	ResearchSlug       string
	TheoremFiles       int
	TheoremsImported   int
	ResultFiles        int
	ResultRowsImported int
	ChainSummaryFiles  int
	ChainRowsImported  int
}

type theoremFileRow struct {
	TheoremPackID   string
	TheoremID       string
	CaseID          string
	GenerationMode  string
	Category        string
	Label           string
	Derivability    string
	Interesting     bool
	Minimal         bool
	Difficulty      string
	AssumptionsJSON string
	Goal            string
	LogicFragment   string
	LeanStatement   string
	Comment         string
}

type summaryRow struct {
	ProjectFolder            string
	RunID                    string
	TimestampUTC             string
	Provider                 string
	ModelRef                 string
	LLMModel                 string
	ExperimentName           string
	ExperimentID             string
	SamplingSurface          string
	RequestedSamplingProfile string
	EffectiveSamplingProfile string
	RequestedSamplingJSON    string
	EffectiveSamplingJSON    string
	UnsupportedSamplingJSON  string
	PromptVersion            string
	CertificateVersion       string
	TheoremPackID            string
	TheoremsFile             string
	TheoremsPath             string
	CaseSelector             string
	Hypothesis               string
	ResultPath               string
	RawDir                   string
	CasesTotal               int
	PassCount                int
	FalseRefusalCount        int
	FalseAcceptCount         int
	RequestFailureCount      int
	SchemaFailureCount       int
	ParseFailureCount        int
	KernelFailureCount       int
	FormatFailureCount       int
	AvgLatencyMS             float64
	MaxLatencyMS             float64
	RunElapsedSeconds        float64
	CategoryCount            int
}

type resultRow struct {
	ProjectFolder            string
	RunID                    string
	TimestampUTC             string
	Provider                 string
	ModelRef                 string
	LLMModel                 string
	ExperimentName           string
	ExperimentID             string
	SamplingSurface          string
	RequestedSamplingProfile string
	EffectiveSamplingProfile string
	RequestedSamplingJSON    string
	EffectiveSamplingJSON    string
	UnsupportedSamplingJSON  string
	PromptVersion            string
	PromptHash               string
	Surface                  string
	TheoremPackID            string
	TheoremID                string
	CaseID                   string
	Category                 string
	ExpectedLabel            string
	RawOutputKind            string
	SchemaStatus             string
	ParseStatus              string
	KernelStatus             string
	CLIExitCode              string
	ScoreBucket              string
	LatencyMS                float64
	Notes                    string
}

type chainSummaryRow struct {
	RunID                     string
	ChainID                   string
	ChainProtocol             string
	CaseFamily                string
	ChainDepth                int
	AtomRenamingID            string
	Stage1Pass                bool
	Stage2Pass                bool
	Stage3GoldPass            bool
	Stage3ModelPass           bool
	NegativeTwinPass          bool
	AllStagesPass             bool
	ImportStagePass           bool
	FinalCompositionPassGold  bool
	FinalCompositionPassModel bool
	ConditionalFinalPassGold  bool
	ConditionalFinalPassModel bool
	FailureStage              string
	FailureType               string
	Notes                     string
}

func SyncProject(ctx context.Context, dbPath string, opts SyncOptions) (SyncResult, error) {
	workspaceRoot := strings.TrimSpace(opts.WorkspaceRoot)
	if workspaceRoot == "" {
		return SyncResult{}, fmt.Errorf("workspace root is required")
	}
	projectFolder := strings.TrimSpace(opts.ProjectFolder)
	if projectFolder == "" {
		return SyncResult{}, fmt.Errorf("project folder is required")
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		return SyncResult{}, err
	}
	defer db.Close()

	researchID, err := ensureResearch(ctx, db, projectFolder, filepath.ToSlash(filepath.Join("research", "artifacts", projectFolder)))
	if err != nil {
		return SyncResult{}, err
	}

	theoremPaths, summaryPaths, resultPaths, chainSummaryPaths, err := discoverProjectFiles(workspaceRoot, projectFolder)
	if err != nil {
		return SyncResult{}, err
	}

	importedTheoremFiles := 0
	importedTheorems := 0
	for _, theoremPath := range theoremPaths {
		count, err := importTheoremFile(ctx, db, workspaceRoot, researchID, theoremPath)
		if err != nil {
			return SyncResult{}, err
		}
		if count > 0 {
			importedTheoremFiles++
			importedTheorems += count
		}
	}

	importedResultFiles := 0
	importedResultRows := 0
	for _, summaryPath := range summaryPaths {
		count, err := importSummaryFile(ctx, db, workspaceRoot, researchID, projectFolder, summaryPath)
		if err != nil {
			return SyncResult{}, err
		}
		importedResultFiles++
		importedResultRows += count
	}
	for _, resultPath := range resultPaths {
		count, err := importResultFile(ctx, db, workspaceRoot, researchID, projectFolder, resultPath)
		if err != nil {
			return SyncResult{}, err
		}
		importedResultFiles++
		importedResultRows += count
	}
	importedChainSummaryFiles := 0
	importedChainSummaryRows := 0
	for _, chainSummaryPath := range chainSummaryPaths {
		count, err := importChainSummaryFile(ctx, db, workspaceRoot, researchID, chainSummaryPath)
		if err != nil {
			return SyncResult{}, err
		}
		importedResultFiles++
		importedChainSummaryFiles++
		importedChainSummaryRows += count
	}

	return SyncResult{
		DBPath:             dbPath,
		ResearchSlug:       projectFolder,
		TheoremFiles:       importedTheoremFiles,
		TheoremsImported:   importedTheorems,
		ResultFiles:        importedResultFiles,
		ResultRowsImported: importedResultRows,
		ChainSummaryFiles:  importedChainSummaryFiles,
		ChainRowsImported:  importedChainSummaryRows,
	}, nil
}

func discoverProjectFiles(workspaceRoot, projectFolder string) ([]string, []string, []string, []string, error) {
	artifactRoot := filepath.Join(workspaceRoot, filepath.FromSlash(artifactsRelativeRoot), projectFolder)
	theoremPaths := make([]string, 0, 16)
	for _, root := range []string{
		filepath.Join(artifactRoot, "theorems"),
		filepath.Join(artifactRoot, "theorems", "historical"),
	} {
		paths, err := globCSVFiles(root)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		theoremPaths = append(theoremPaths, paths...)
	}
	slices.Sort(theoremPaths)
	theoremPaths = uniqueStrings(theoremPaths)

	localResultPaths, err := globCSVFiles(filepath.Join(artifactRoot, "result"))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	summaryPaths := make([]string, 0, len(localResultPaths))
	resultPaths := make([]string, 0, len(localResultPaths))
	chainSummaryPaths := make([]string, 0, len(localResultPaths))
	for _, path := range localResultPaths {
		base := filepath.Base(path)
		switch {
		case strings.HasPrefix(base, "result_summary") && strings.HasSuffix(base, ".csv"):
			summaryPaths = append(summaryPaths, path)
		case strings.HasPrefix(base, "result_") && strings.HasSuffix(base, ".csv"):
			resultPaths = append(resultPaths, path)
		case strings.HasPrefix(base, "chain_result_") && strings.HasSuffix(base, ".csv"):
			chainSummaryPaths = append(chainSummaryPaths, path)
		}
	}

	sharedRoot := filepath.Join(workspaceRoot, filepath.FromSlash(artifactsRelativeRoot), "result_research")
	sharedPaths, err := globCSVFiles(sharedRoot)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, path := range sharedPaths {
		if sharedSummaryPathMatchesProject(path, projectFolder) {
			summaryPaths = append(summaryPaths, path)
		}
	}

	slices.Sort(summaryPaths)
	summaryPaths = uniqueStrings(summaryPaths)
	slices.Sort(resultPaths)
	resultPaths = uniqueStrings(resultPaths)
	slices.Sort(chainSummaryPaths)
	chainSummaryPaths = uniqueStrings(chainSummaryPaths)
	return theoremPaths, summaryPaths, resultPaths, chainSummaryPaths, nil
}

func globCSVFiles(root string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, nil
	}
	files := make([]string, 0, 16)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".csv") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk csv files under %s: %w", root, err)
	}
	return files, nil
}

func importTheoremFile(ctx context.Context, db *sql.DB, workspaceRoot string, researchID int64, path string) (int, error) {
	rows, theoremPackID, err := loadTheoremFile(path)
	if err != nil {
		return 0, err
	}
	relativePath := relativeSlash(workspaceRoot, path)
	typeCode := "canonical_pack"
	isHistorical := 0
	if strings.Contains(relativePath, "/theorems/historical/") {
		typeCode = "historical_pack"
		isHistorical = 1
		expectedPackID := derivePackIDFromPath(path)
		if isReservedPackID(theoremPackID) {
			if _, err := db.ExecContext(ctx, `DELETE FROM research_theorem_files WHERE relative_path = ?`, relativePath); err != nil {
				return 0, fmt.Errorf("remove reserved historical theorem file %s: %w", relativePath, err)
			}
			return 0, nil
		}
		if expectedPackID != "" && theoremPackID != "" && expectedPackID != theoremPackID {
			if _, err := db.ExecContext(ctx, `DELETE FROM research_theorem_files WHERE relative_path = ?`, relativePath); err != nil {
				return 0, fmt.Errorf("remove stale historical theorem file %s: %w", relativePath, err)
			}
			return 0, nil
		}
	}
	typeID, err := lookupTypeID(ctx, db, typeCode)
	if err != nil {
		return 0, err
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_theorem_files(research_id, type_id, theorem_pack_id, relative_path, absolute_path, is_historical, imported_at_utc)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(relative_path) DO UPDATE SET
    research_id = excluded.research_id,
    type_id = excluded.type_id,
    theorem_pack_id = excluded.theorem_pack_id,
    absolute_path = excluded.absolute_path,
    is_historical = excluded.is_historical,
    imported_at_utc = excluded.imported_at_utc
`, researchID, typeID, theoremPackID, relativePath, filepath.ToSlash(path), isHistorical, importedAt); err != nil {
		return 0, fmt.Errorf("upsert theorem file %s: %w", relativePath, err)
	}
	theoremFileID, err := lookupTheoremFileID(ctx, db, relativePath)
	if err != nil {
		return 0, err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM research_theorems WHERE theorem_file_id = ?`, theoremFileID); err != nil {
		return 0, fmt.Errorf("clear theorem rows for %s: %w", relativePath, err)
	}
	for idx, row := range rows {
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_theorems(
    research_id, theorem_file_id, theorem_pack_id, theorem_id, row_order, case_id, generation_mode, category, label,
    derivability_status, interesting, minimal, difficulty, assumptions_json, goal, logic_fragment, lean_statement, comment
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, theoremFileID, row.TheoremPackID, row.TheoremID, idx+1, row.CaseID, row.GenerationMode, row.Category, row.Label,
			row.Derivability, boolToInt(row.Interesting), boolToInt(row.Minimal), row.Difficulty, row.AssumptionsJSON, row.Goal, row.LogicFragment, row.LeanStatement, row.Comment); err != nil {
			return 0, fmt.Errorf("insert theorem row %s from %s: %w", row.TheoremID, relativePath, err)
		}
	}
	return len(rows), nil
}

func importSummaryFile(ctx context.Context, db *sql.DB, workspaceRoot string, researchID int64, researchSlug, path string) (int, error) {
	rows, err := loadSummaryFile(path)
	if err != nil {
		return 0, err
	}
	relativePath := relativeSlash(workspaceRoot, path)
	artifactGroup := inferArtifactGroup(relativePath)
	typeID, err := lookupTypeID(ctx, db, artifactGroup)
	if err != nil {
		return 0, err
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_result_files(research_id, type_id, artifact_group, file_kind, relative_path, absolute_path, imported_at_utc)
VALUES (?, ?, ?, 'summary', ?, ?, ?)
ON CONFLICT(relative_path) DO UPDATE SET
    research_id = excluded.research_id,
    type_id = excluded.type_id,
    artifact_group = excluded.artifact_group,
    file_kind = excluded.file_kind,
    absolute_path = excluded.absolute_path,
    imported_at_utc = excluded.imported_at_utc
`, researchID, typeID, artifactGroup, relativePath, filepath.ToSlash(path), importedAt); err != nil {
		return 0, fmt.Errorf("upsert summary file %s: %w", relativePath, err)
	}
	resultFileID, err := lookupResultFileID(ctx, db, relativePath)
	if err != nil {
		return 0, err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM research_result_summaries WHERE result_file_id = ?`, resultFileID); err != nil {
		return 0, fmt.Errorf("clear summary rows for %s: %w", relativePath, err)
	}
	summaryTypeID, err := lookupTypeID(ctx, db, "summary")
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		llmModelID, err := ensureLLMModel(ctx, db, row.LLMModel, row.ModelRef, row.Provider)
		if err != nil {
			return 0, err
		}
		certificateID, err := ensureCertificate(ctx, db, row.CertificateVersion)
		if err != nil {
			return 0, err
		}
		theoremFileID, err := lookupTheoremFileByPack(ctx, db, researchID, row.TheoremPackID)
		if err != nil {
			return 0, err
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_result_summaries(
    research_id, result_file_id, theorem_file_id, type_id, llm_model_id, certificate_id, run_id, timestamp_utc,
    project_folder, experiment_name, experiment_id, sampling_surface, requested_sampling_profile, effective_sampling_profile,
    requested_sampling_json, effective_sampling_json, unsupported_sampling_json, prompt_version, theorem_pack_id, case_selector,
    theorems_file, theorems_path, result_path, raw_dir, hypothesis,
    cases_total, pass_count, false_refusal_count, false_accept_count, request_failure_count, schema_failure_count,
    parse_failure_count, kernel_failure_count, format_failure_count, avg_latency_ms, max_latency_ms, run_elapsed_seconds, category_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, resultFileID, theoremFileID, summaryTypeID, llmModelID, certificateID, row.RunID, row.TimestampUTC,
			firstNonEmptyCSV(row.ProjectFolder, researchSlug), row.ExperimentName, row.ExperimentID, row.SamplingSurface,
			row.RequestedSamplingProfile, row.EffectiveSamplingProfile, row.RequestedSamplingJSON, row.EffectiveSamplingJSON,
			row.UnsupportedSamplingJSON, row.PromptVersion, row.TheoremPackID, row.CaseSelector, row.TheoremsFile,
			row.TheoremsPath, row.ResultPath, row.RawDir, row.Hypothesis, row.CasesTotal, row.PassCount, row.FalseRefusalCount,
			row.FalseAcceptCount, row.RequestFailureCount, row.SchemaFailureCount, row.ParseFailureCount, row.KernelFailureCount,
			row.FormatFailureCount, row.AvgLatencyMS, row.MaxLatencyMS, row.RunElapsedSeconds, row.CategoryCount); err != nil {
			return 0, fmt.Errorf("insert summary row from %s: %w", relativePath, err)
		}
	}
	return len(rows), nil
}

func importResultFile(ctx context.Context, db *sql.DB, workspaceRoot string, researchID int64, researchSlug, path string) (int, error) {
	rows, err := loadResultFile(path)
	if err != nil {
		return 0, err
	}
	relativePath := relativeSlash(workspaceRoot, path)
	artifactGroup := inferArtifactGroup(relativePath)
	typeID, err := lookupTypeID(ctx, db, artifactGroup)
	if err != nil {
		return 0, err
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_result_files(research_id, type_id, artifact_group, file_kind, relative_path, absolute_path, imported_at_utc)
VALUES (?, ?, ?, 'result_rows', ?, ?, ?)
ON CONFLICT(relative_path) DO UPDATE SET
    research_id = excluded.research_id,
    type_id = excluded.type_id,
    artifact_group = excluded.artifact_group,
    file_kind = excluded.file_kind,
    absolute_path = excluded.absolute_path,
    imported_at_utc = excluded.imported_at_utc
`, researchID, typeID, artifactGroup, relativePath, filepath.ToSlash(path), importedAt); err != nil {
		return 0, fmt.Errorf("upsert result file %s: %w", relativePath, err)
	}
	resultFileID, err := lookupResultFileID(ctx, db, relativePath)
	if err != nil {
		return 0, err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM research_result_rows WHERE result_file_id = ?`, resultFileID); err != nil {
		return 0, fmt.Errorf("clear result rows for %s: %w", relativePath, err)
	}
	resultTypeID, err := lookupTypeID(ctx, db, "result_row")
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		llmModelID, err := ensureLLMModel(ctx, db, row.LLMModel, row.ModelRef, row.Provider)
		if err != nil {
			return 0, err
		}
		theoremFileID, err := lookupTheoremFileByPack(ctx, db, researchID, row.TheoremPackID)
		if err != nil {
			return 0, err
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_result_rows(
    research_id, result_file_id, theorem_file_id, type_id, llm_model_id, run_id, timestamp_utc,
    project_folder, experiment_name, experiment_id, sampling_surface, requested_sampling_profile, effective_sampling_profile,
    requested_sampling_json, effective_sampling_json, unsupported_sampling_json, prompt_version, theorem_pack_id, theorem_id, case_id, category, expected_label, score_bucket,
    raw_output_kind, schema_status, parse_status, kernel_status, cli_exit_code, latency_ms, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, resultFileID, theoremFileID, resultTypeID, llmModelID, row.RunID, row.TimestampUTC, firstNonEmptyCSV(row.ProjectFolder, researchSlug),
			row.ExperimentName, row.ExperimentID, row.SamplingSurface, row.RequestedSamplingProfile, row.EffectiveSamplingProfile,
			row.RequestedSamplingJSON, row.EffectiveSamplingJSON, row.UnsupportedSamplingJSON, row.PromptVersion,
			row.TheoremPackID, row.TheoremID, row.CaseID, row.Category, row.ExpectedLabel, row.ScoreBucket,
			row.RawOutputKind, row.SchemaStatus, row.ParseStatus, row.KernelStatus, row.CLIExitCode, row.LatencyMS, row.Notes); err != nil {
			return 0, fmt.Errorf("insert result row from %s: %w", relativePath, err)
		}
	}
	return len(rows), nil
}

func importChainSummaryFile(ctx context.Context, db *sql.DB, workspaceRoot string, researchID int64, path string) (int, error) {
	rows, err := loadChainSummaryFile(path)
	if err != nil {
		return 0, err
	}
	relativePath := relativeSlash(workspaceRoot, path)
	artifactGroup := inferChainSummaryArtifactGroup(relativePath, rows)
	typeID, err := lookupTypeID(ctx, db, artifactGroup)
	if err != nil {
		return 0, err
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_result_files(research_id, type_id, artifact_group, file_kind, relative_path, absolute_path, imported_at_utc)
VALUES (?, ?, ?, 'chain_summary', ?, ?, ?)
ON CONFLICT(relative_path) DO UPDATE SET
    research_id = excluded.research_id,
    type_id = excluded.type_id,
    artifact_group = excluded.artifact_group,
    file_kind = excluded.file_kind,
    absolute_path = excluded.absolute_path,
    imported_at_utc = excluded.imported_at_utc
`, researchID, typeID, artifactGroup, relativePath, filepath.ToSlash(path), importedAt); err != nil {
		return 0, fmt.Errorf("upsert chain summary file %s: %w", relativePath, err)
	}
	resultFileID, err := lookupResultFileID(ctx, db, relativePath)
	if err != nil {
		return 0, err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM research_chain_summary_rows WHERE result_file_id = ?`, resultFileID); err != nil {
		return 0, fmt.Errorf("clear chain summary rows for %s: %w", relativePath, err)
	}
	chainSummaryTypeID, err := lookupTypeID(ctx, db, "chain_summary_row")
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_chain_summary_rows(
    research_id, result_file_id, type_id, run_id, chain_id, case_family, chain_depth, atom_renaming_id,
    stage1_pass, stage2_pass, stage3_gold_pass, stage3_model_pass, negative_twin_pass, all_stages_pass,
    import_stage_pass, final_composition_pass_gold, final_composition_pass_model,
    conditional_final_pass_gold, conditional_final_pass_model, failure_stage, failure_type, notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, researchID, resultFileID, chainSummaryTypeID, row.RunID, row.ChainID, row.CaseFamily, row.ChainDepth, row.AtomRenamingID,
			boolToInt(row.Stage1Pass), boolToInt(row.Stage2Pass), boolToInt(row.Stage3GoldPass), boolToInt(row.Stage3ModelPass),
			boolToInt(row.NegativeTwinPass), boolToInt(row.AllStagesPass), boolToInt(row.ImportStagePass),
			boolToInt(row.FinalCompositionPassGold), boolToInt(row.FinalCompositionPassModel),
			boolToInt(row.ConditionalFinalPassGold), boolToInt(row.ConditionalFinalPassModel),
			row.FailureStage, row.FailureType, row.Notes); err != nil {
			return 0, fmt.Errorf("insert chain summary row %s from %s: %w", row.ChainID, relativePath, err)
		}
	}
	return len(rows), nil
}

func loadTheoremFile(path string) ([]theoremFileRow, string, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, "", err
	}
	if _, ok := header["goal"]; !ok {
		return nil, "", fmt.Errorf("theorem csv %s is missing required column %q", path, "goal")
	}
	if _, ok := header["case_id"]; !ok {
		if _, ok := header["theorem_id"]; !ok {
			return nil, "", fmt.Errorf("theorem csv %s must contain either %q or %q", path, "case_id", "theorem_id")
		}
	}
	defaultPackID := derivePackIDFromPath(path)
	rows := make([]theoremFileRow, 0, len(records))
	packID := ""
	for idx, record := range records {
		theoremID := firstNonEmptyCSV(csvCell(record, header, "theorem_id"), csvCell(record, header, "case_id"))
		if theoremID == "" {
			return nil, "", fmt.Errorf("theorem csv %s row %d has blank theorem_id/case_id", path, idx+2)
		}
		rowPackID := firstNonEmptyCSV(csvCell(record, header, "theorem_pack_id"), defaultPackID)
		if packID == "" {
			packID = rowPackID
		}
		rows = append(rows, theoremFileRow{
			TheoremPackID:   rowPackID,
			TheoremID:       theoremID,
			CaseID:          firstNonEmptyCSV(csvCell(record, header, "case_id"), theoremID),
			GenerationMode:  csvCell(record, header, "generation_mode"),
			Category:        csvCell(record, header, "category"),
			Label:           csvCell(record, header, "label"),
			Derivability:    csvCell(record, header, "derivability_status"),
			Interesting:     parseBoolLoose(csvCell(record, header, "interesting")),
			Minimal:         parseBoolLoose(csvCell(record, header, "minimal")),
			Difficulty:      csvCell(record, header, "difficulty"),
			AssumptionsJSON: csvCell(record, header, "assumptions_json"),
			Goal:            csvCell(record, header, "goal"),
			LogicFragment:   csvCell(record, header, "logic_fragment"),
			LeanStatement:   csvCell(record, header, "lean_statement"),
			Comment:         csvCell(record, header, "comment"),
		})
	}
	if packID == "" {
		packID = defaultPackID
	}
	return rows, packID, nil
}

func loadSummaryFile(path string) ([]summaryRow, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	rows := make([]summaryRow, 0, len(records))
	for _, record := range records {
		theoremsFile := firstNonEmptyCSV(csvCell(record, header, "theorems_file"), csvCell(record, header, "cases_file"))
		theoremsPath := firstNonEmptyCSV(csvCell(record, header, "theorems_path"), csvCell(record, header, "cases_path"))
		rows = append(rows, summaryRow{
			ProjectFolder:            csvCell(record, header, "project_folder"),
			RunID:                    csvCell(record, header, "run_id"),
			TimestampUTC:             csvCell(record, header, "timestamp_utc"),
			Provider:                 firstNonEmptyCSV(csvCell(record, header, "provider"), deriveProviderFromModelRef(csvCell(record, header, "model"))),
			ModelRef:                 csvCell(record, header, "model"),
			LLMModel:                 firstNonEmptyCSV(csvCell(record, header, "llm_model"), csvCell(record, header, "model")),
			ExperimentName:           csvCell(record, header, "experiment_name"),
			ExperimentID:             csvCell(record, header, "experiment_id"),
			SamplingSurface:          csvCell(record, header, "sampling_surface"),
			RequestedSamplingProfile: csvCell(record, header, "requested_sampling_profile"),
			EffectiveSamplingProfile: csvCell(record, header, "effective_sampling_profile"),
			RequestedSamplingJSON:    csvCell(record, header, "requested_sampling_json"),
			EffectiveSamplingJSON:    csvCell(record, header, "effective_sampling_json"),
			UnsupportedSamplingJSON:  csvCell(record, header, "unsupported_sampling_json"),
			PromptVersion:            csvCell(record, header, "prompt_version"),
			CertificateVersion:       csvCell(record, header, "certificate_version"),
			TheoremPackID:            firstNonEmptyCSV(csvCell(record, header, "theorem_pack_id"), derivePackIDFromSummaryCells(theoremsPath, theoremsFile)),
			TheoremsFile:             theoremsFile,
			TheoremsPath:             theoremsPath,
			CaseSelector:             csvCell(record, header, "case_selector"),
			Hypothesis:               csvCell(record, header, "hypothesis"),
			ResultPath:               csvCell(record, header, "result_file"),
			RawDir:                   csvCell(record, header, "raw_dir"),
			CasesTotal:               parseIntLoose(csvCell(record, header, "cases_total")),
			PassCount:                parseIntLoose(csvCell(record, header, "pass_count")),
			FalseRefusalCount:        parseIntLoose(csvCell(record, header, "false_refusal_count")),
			FalseAcceptCount:         parseIntLoose(csvCell(record, header, "false_accept_count")),
			RequestFailureCount:      parseIntLoose(csvCell(record, header, "request_failure_count")),
			SchemaFailureCount:       parseIntLoose(csvCell(record, header, "schema_failure_count")),
			ParseFailureCount:        parseIntLoose(csvCell(record, header, "parse_failure_count")),
			KernelFailureCount:       parseIntLoose(csvCell(record, header, "kernel_failure_count")),
			FormatFailureCount:       parseIntLoose(csvCell(record, header, "format_failure_count")),
			AvgLatencyMS:             parseFloatLoose(csvCell(record, header, "avg_latency_ms")),
			MaxLatencyMS:             parseFloatLoose(csvCell(record, header, "max_latency_ms")),
			RunElapsedSeconds:        parseFloatLoose(csvCell(record, header, "run_elapsed_seconds")),
			CategoryCount:            parseIntLoose(csvCell(record, header, "category_count")),
		})
	}
	return dedupeSummaryRows(rows), nil
}

func loadResultFile(path string) ([]resultRow, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	rows := make([]resultRow, 0, len(records))
	for _, record := range records {
		theoremID := firstNonEmptyCSV(csvCell(record, header, "theorem_id"), csvCell(record, header, "case_id"))
		rows = append(rows, resultRow{
			ProjectFolder:            csvCell(record, header, "project_folder"),
			RunID:                    csvCell(record, header, "run_id"),
			TimestampUTC:             csvCell(record, header, "timestamp_utc"),
			Provider:                 firstNonEmptyCSV(csvCell(record, header, "provider"), deriveProviderFromModelRef(csvCell(record, header, "model"))),
			ModelRef:                 csvCell(record, header, "model"),
			LLMModel:                 firstNonEmptyCSV(csvCell(record, header, "llm_model"), csvCell(record, header, "model")),
			ExperimentName:           csvCell(record, header, "experiment_name"),
			ExperimentID:             csvCell(record, header, "experiment_id"),
			SamplingSurface:          csvCell(record, header, "sampling_surface"),
			RequestedSamplingProfile: csvCell(record, header, "requested_sampling_profile"),
			EffectiveSamplingProfile: csvCell(record, header, "effective_sampling_profile"),
			RequestedSamplingJSON:    csvCell(record, header, "requested_sampling_json"),
			EffectiveSamplingJSON:    csvCell(record, header, "effective_sampling_json"),
			UnsupportedSamplingJSON:  csvCell(record, header, "unsupported_sampling_json"),
			PromptVersion:            csvCell(record, header, "prompt_version"),
			PromptHash:               csvCell(record, header, "prompt_hash"),
			Surface:                  csvCell(record, header, "surface"),
			TheoremPackID:            csvCell(record, header, "theorem_pack_id"),
			TheoremID:                theoremID,
			CaseID:                   firstNonEmptyCSV(csvCell(record, header, "case_id"), theoremID),
			Category:                 csvCell(record, header, "category"),
			ExpectedLabel:            firstNonEmptyCSV(csvCell(record, header, "expected_label"), csvCell(record, header, "label")),
			RawOutputKind:            csvCell(record, header, "raw_output_kind"),
			SchemaStatus:             csvCell(record, header, "schema_status"),
			ParseStatus:              csvCell(record, header, "parse_status"),
			KernelStatus:             csvCell(record, header, "kernel_status"),
			CLIExitCode:              csvCell(record, header, "cli_exit_code"),
			ScoreBucket:              csvCell(record, header, "score_bucket"),
			LatencyMS:                parseFloatLoose(csvCell(record, header, "latency_ms")),
			Notes:                    csvCell(record, header, "notes"),
		})
	}
	return dedupeResultRows(rows), nil
}

func loadChainSummaryFile(path string) ([]chainSummaryRow, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	rows := make([]chainSummaryRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, chainSummaryRow{
			RunID:                     csvCell(record, header, "run_id"),
			ChainID:                   csvCell(record, header, "chain_id"),
			ChainProtocol:             csvCell(record, header, "chain_protocol"),
			CaseFamily:                csvCell(record, header, "case_family"),
			ChainDepth:                parseIntLoose(csvCell(record, header, "chain_depth")),
			AtomRenamingID:            csvCell(record, header, "atom_renaming_id"),
			Stage1Pass:                parseBoolLoose(csvCell(record, header, "stage1_pass")),
			Stage2Pass:                parseBoolLoose(csvCell(record, header, "stage2_pass")),
			Stage3GoldPass:            parseBoolLoose(csvCell(record, header, "stage3_gold_pass")),
			Stage3ModelPass:           parseBoolLoose(csvCell(record, header, "stage3_model_pass")),
			NegativeTwinPass:          parseBoolLoose(csvCell(record, header, "negative_twin_pass")),
			AllStagesPass:             parseBoolLoose(csvCell(record, header, "all_stages_pass")),
			ImportStagePass:           parseBoolLoose(csvCell(record, header, "import_stage_pass")),
			FinalCompositionPassGold:  parseBoolLoose(csvCell(record, header, "final_composition_pass_gold")),
			FinalCompositionPassModel: parseBoolLoose(csvCell(record, header, "final_composition_pass_model")),
			ConditionalFinalPassGold:  parseBoolLoose(csvCell(record, header, "conditional_final_pass_gold")),
			ConditionalFinalPassModel: parseBoolLoose(csvCell(record, header, "conditional_final_pass_model")),
			FailureStage:              csvCell(record, header, "failure_stage"),
			FailureType:               csvCell(record, header, "failure_type"),
			Notes:                     csvCell(record, header, "notes"),
		})
	}
	return dedupeChainSummaryRows(rows), nil
}

func dedupeSummaryRows(rows []summaryRow) []summaryRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]summaryRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.Join([]string{
			strings.TrimSpace(row.ProjectFolder),
			strings.TrimSpace(row.RunID),
			strings.TrimSpace(row.Provider),
			strings.TrimSpace(row.ModelRef),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentName),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.RequestedSamplingProfile),
			strings.TrimSpace(row.EffectiveSamplingProfile),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.ResultPath),
		}, "\x00")
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func dedupeResultRows(rows []resultRow) []resultRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]resultRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.Join([]string{
			strings.TrimSpace(row.ProjectFolder),
			strings.TrimSpace(row.RunID),
			strings.TrimSpace(row.Provider),
			strings.TrimSpace(row.ModelRef),
			strings.TrimSpace(row.LLMModel),
			strings.TrimSpace(row.ExperimentName),
			strings.TrimSpace(row.ExperimentID),
			strings.TrimSpace(row.RequestedSamplingProfile),
			strings.TrimSpace(row.EffectiveSamplingProfile),
			strings.TrimSpace(row.PromptVersion),
			strings.TrimSpace(row.PromptHash),
			strings.TrimSpace(row.Surface),
			strings.TrimSpace(row.TheoremPackID),
			strings.TrimSpace(row.TheoremID),
			strings.TrimSpace(row.CaseID),
			strings.TrimSpace(row.Category),
			strings.TrimSpace(row.ExpectedLabel),
		}, "\x00")
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func dedupeChainSummaryRows(rows []chainSummaryRow) []chainSummaryRow {
	if len(rows) <= 1 {
		return rows
	}
	deduped := make([]chainSummaryRow, 0, len(rows))
	positions := make(map[string]int, len(rows))
	for _, row := range rows {
		key := strings.Join([]string{
			strings.TrimSpace(row.RunID),
			strings.TrimSpace(row.ChainID),
			strings.TrimSpace(row.CaseFamily),
			strings.TrimSpace(row.AtomRenamingID),
		}, "\x00")
		if idx, ok := positions[key]; ok {
			deduped[idx] = row
			continue
		}
		positions[key] = len(deduped)
		deduped = append(deduped, row)
	}
	return deduped
}

func readCSV(path string) ([][]string, map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open csv %s: %w", path, err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("read csv %s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("csv %s is empty", path)
	}
	header := make(map[string]int, len(records[0]))
	for idx, value := range records[0] {
		header[strings.TrimSpace(value)] = idx
	}
	return records[1:], header, nil
}

func csvCell(record []string, header map[string]int, name string) string {
	idx, ok := header[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func ensureResearch(ctx context.Context, db *sql.DB, slug, artifactRoot string) (int64, error) {
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_researches(slug, title, artifact_root)
VALUES (?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    title = excluded.title,
    artifact_root = excluded.artifact_root
`, slug, prettifySlug(slug), filepath.ToSlash(artifactRoot)); err != nil {
		return 0, fmt.Errorf("upsert research %q: %w", slug, err)
	}
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_researches WHERE slug = ?`, slug).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup research id %q: %w", slug, err)
	}
	return id, nil
}

func ensureLLMModel(ctx context.Context, db *sql.DB, llmModel, modelRef, provider string) (sql.NullInt64, error) {
	llmModel = strings.TrimSpace(llmModel)
	if llmModel == "" {
		return sql.NullInt64{}, nil
	}
	modelRef = strings.TrimSpace(modelRef)
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = deriveProviderFromModelRef(modelRef)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_llm_models(llm_model, model_ref, provider, display_name)
VALUES (?, ?, ?, ?)
ON CONFLICT(llm_model) DO UPDATE SET
    model_ref = excluded.model_ref,
    provider = excluded.provider,
    display_name = excluded.display_name
`, llmModel, modelRef, provider, llmModel); err != nil {
		return sql.NullInt64{}, fmt.Errorf("upsert llm model %q: %w", llmModel, err)
	}
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_llm_models WHERE llm_model = ?`, llmModel).Scan(&id); err != nil {
		return sql.NullInt64{}, fmt.Errorf("lookup llm model %q: %w", llmModel, err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func deriveProviderFromModelRef(modelRef string) string {
	modelRef = strings.TrimSpace(modelRef)
	if modelRef == "" {
		return ""
	}
	parts := strings.SplitN(modelRef, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func ensureCertificate(ctx context.Context, db *sql.DB, certificateVersion string) (sql.NullInt64, error) {
	certificateVersion = strings.TrimSpace(certificateVersion)
	if certificateVersion == "" {
		return sql.NullInt64{}, nil
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_certificates(certificate_version)
VALUES (?)
ON CONFLICT(certificate_version) DO NOTHING
`, certificateVersion); err != nil {
		return sql.NullInt64{}, fmt.Errorf("upsert certificate version %q: %w", certificateVersion, err)
	}
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_certificates WHERE certificate_version = ?`, certificateVersion).Scan(&id); err != nil {
		return sql.NullInt64{}, fmt.Errorf("lookup certificate version %q: %w", certificateVersion, err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func lookupTypeID(ctx context.Context, db *sql.DB, typeCode string) (sql.NullInt64, error) {
	typeCode = strings.TrimSpace(typeCode)
	if typeCode == "" {
		return sql.NullInt64{}, nil
	}
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_types WHERE type_code = ?`, typeCode).Scan(&id); err != nil {
		return sql.NullInt64{}, fmt.Errorf("lookup research type %q: %w", typeCode, err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func lookupTheoremFileID(ctx context.Context, db *sql.DB, relativePath string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_theorem_files WHERE relative_path = ?`, relativePath).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup theorem file %s: %w", relativePath, err)
	}
	return id, nil
}

func lookupTheoremFileByPack(ctx context.Context, db *sql.DB, researchID int64, theoremPackID string) (sql.NullInt64, error) {
	theoremPackID = strings.TrimSpace(theoremPackID)
	if theoremPackID == "" {
		return sql.NullInt64{}, nil
	}
	var id int64
	err := db.QueryRowContext(ctx, `
SELECT id
FROM research_theorem_files
WHERE research_id = ? AND theorem_pack_id = ?
ORDER BY is_historical ASC, id ASC
LIMIT 1
`, researchID, theoremPackID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.NullInt64{}, nil
		}
		return sql.NullInt64{}, fmt.Errorf("lookup theorem file by pack %q: %w", theoremPackID, err)
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func lookupResultFileID(ctx context.Context, db *sql.DB, relativePath string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM research_result_files WHERE relative_path = ?`, relativePath).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup result file %s: %w", relativePath, err)
	}
	return id, nil
}

func inferArtifactGroup(relativePath string) string {
	lower := strings.ToLower(filepath.ToSlash(relativePath))
	parts := strings.Split(strings.Trim(lower, "/"), "/")
	for idx, part := range parts {
		if part != "result_research" {
			continue
		}
		for _, nested := range parts[idx+1:] {
			switch nested {
			case "direct":
				return "direct"
			case "nd":
				return "nd"
			case "pair":
				return "pair"
			case "compositional":
				return "compositional"
			case "compositional-depth-ladder", "compositional_depth_ladder":
				return "compositional_depth_ladder"
			case "compositional-branching", "compositional_branching":
				return "compositional_branching"
			case "compositional-mixed-family-reuse", "compositional_mixed_family_reuse":
				return "compositional_mixed_family_reuse"
			case "compositional-mixed-family-gold-first", "compositional_mixed_family_gold_first":
				return "compositional_mixed_family_gold_first"
			case "compositional-mixed-family-gold-first-hard", "compositional_mixed_family_gold_first_hard":
				return "compositional_mixed_family_gold_first_hard"
			case "compositional-mixed-family-semi-gold-stable", "compositional_mixed_family_semi_gold_stable":
				return "compositional_mixed_family_semi_gold_stable"
			case "compositional-mixed-family-semi-gold-frontier", "compositional_mixed_family_semi_gold_frontier":
				return "compositional_mixed_family_semi_gold_frontier"
			case "waves", "wave":
				return "wave"
			}
		}
	}
	switch {
	case strings.Contains(lower, "/result_research/direct/"):
		return "direct"
	case strings.Contains(lower, "/result_research/nd/"):
		return "nd"
	case strings.Contains(lower, "/result_research/pair/"):
		return "pair"
	case strings.Contains(lower, "/result_research/compositional/"):
		return "compositional"
	case strings.Contains(lower, "/result_research/compositional-depth-ladder/"), strings.Contains(lower, "/result_research/compositional_depth_ladder/"):
		return "compositional_depth_ladder"
	case strings.Contains(lower, "/result_research/compositional-branching/"), strings.Contains(lower, "/result_research/compositional_branching/"):
		return "compositional_branching"
	case strings.Contains(lower, "/result_research/compositional-mixed-family-reuse/"), strings.Contains(lower, "/result_research/compositional_mixed_family_reuse/"):
		return "compositional_mixed_family_reuse"
	case strings.Contains(lower, "/result_research/compositional-mixed-family-gold-first/"), strings.Contains(lower, "/result_research/compositional_mixed_family_gold_first/"):
		return "compositional_mixed_family_gold_first"
	case strings.Contains(lower, "/result_research/compositional-mixed-family-gold-first-hard/"), strings.Contains(lower, "/result_research/compositional_mixed_family_gold_first_hard/"):
		return "compositional_mixed_family_gold_first_hard"
	case strings.Contains(lower, "/result_research/compositional-mixed-family-semi-gold-stable/"), strings.Contains(lower, "/result_research/compositional_mixed_family_semi_gold_stable/"):
		return "compositional_mixed_family_semi_gold_stable"
	case strings.Contains(lower, "/result_research/compositional-mixed-family-semi-gold-frontier/"), strings.Contains(lower, "/result_research/compositional_mixed_family_semi_gold_frontier/"):
		return "compositional_mixed_family_semi_gold_frontier"
	case strings.Contains(lower, "/result_research/waves/"):
		return "wave"
	default:
		return "family"
	}
}

func inferChainSummaryArtifactGroup(relativePath string, rows []chainSummaryRow) string {
	if inferred := inferArtifactGroup(relativePath); inferred != "family" {
		return inferred
	}
	for _, row := range rows {
		protocol := strings.ToLower(strings.TrimSpace(row.ChainProtocol))
		switch protocol {
		case "depth-ladder-v1", "depth_ladder_v1":
			return "compositional_depth_ladder"
		case "branching-v1", "branching_v1":
			return "compositional_branching"
		case "mixed-family-v1", "mixed_family_v1":
			return "compositional_mixed_family_reuse"
		case "mixed-family-gold-first-v1", "mixed_family_gold_first_v1":
			return "compositional_mixed_family_gold_first"
		case "mixed-family-gold-first-hard-v1", "mixed_family_gold_first_hard_v1":
			return "compositional_mixed_family_gold_first_hard"
		case "mixed-family-semi-gold-stable-v1", "mixed_family_semi_gold_stable_v1":
			return "compositional_mixed_family_semi_gold_stable"
		case "mixed-family-semi-gold-frontier-v1", "mixed_family_semi_gold_frontier_v1":
			return "compositional_mixed_family_semi_gold_frontier"
		case "starter-v1", "starter_v1":
			return "compositional"
		}
	}
	return "compositional"
}

func sharedSummaryPathMatchesProject(path, projectFolder string) bool {
	projectFolder = strings.TrimSpace(projectFolder)
	if projectFolder == "" {
		return false
	}
	base := filepath.Base(path)
	if !strings.EqualFold(filepath.Ext(base), ".csv") {
		return false
	}
	lowerBase := strings.ToLower(strings.TrimSpace(base))
	switch {
	case strings.HasPrefix(lowerBase, "chain_result_"):
		return false
	case strings.Contains(base, projectFolder+"_"):
		return true
	}

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headerRow, err := reader.Read()
	if err != nil {
		return false
	}
	header := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		header[strings.TrimSpace(value)] = idx
	}
	if _, ok := header["project_folder"]; !ok {
		return false
	}
	if _, ok := header["run_id"]; !ok {
		return false
	}

	for {
		record, err := reader.Read()
		if err != nil {
			return false
		}
		if isCSVRecordBlank(record) {
			continue
		}
		return strings.TrimSpace(csvCell(record, header, "project_folder")) == projectFolder
	}
}

func isCSVRecordBlank(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func relativeSlash(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(relative)
}

func derivePackIDFromSummaryCells(theoremsPath, theoremsFile string) string {
	return derivePackIDFromPath(firstNonEmptyCSV(theoremsPath, theoremsFile))
}

func derivePackIDFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	lower := strings.ToLower(base)
	switch {
	case strings.HasPrefix(lower, "theorems_"):
		base = base[len("theorems_"):]
	case strings.HasPrefix(lower, "cases_"):
		base = base[len("cases_"):]
	}
	return sanitizePackID(base)
}

func sanitizePackID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func isReservedPackID(value string) bool {
	switch sanitizePackID(value) {
	case "", "theorems", "cases", "historical", "result":
		return true
	default:
		return false
	}
}

func firstNonEmptyCSV(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseBoolLoose(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseIntLoose(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseFloatLoose(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func prettifySlug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unnamed Research"
	}
	parts := strings.Split(value, "-")
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
