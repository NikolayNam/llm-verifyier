package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func benchmarkCasesHaveChainMetadata(cases []benchmarkCase) bool {
	for _, tc := range cases {
		if benchmarkCaseUsesCompositionalChainProtocol(tc) {
			return true
		}
	}
	return false
}

func benchmarkCaseUsesCompositionalChainProtocol(tc benchmarkCase) bool {
	return strings.TrimSpace(tc.ChainID) != "" && strings.TrimSpace(tc.StageID) != ""
}

func benchmarkChainSummaryPath(resultsPath, runID string) string {
	if strings.TrimSpace(resultsPath) == "" || strings.TrimSpace(runID) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(resultsPath), fmt.Sprintf("chain_result_%s.csv", strings.TrimSpace(runID)))
}

type benchmarkChainSummaryRow struct {
	RunID                     string
	ChainID                   string
	ChainProtocol             string
	CaseFamily                string
	ChainDepth                string
	AtomRenamingID            string
	SourceFamily              string
	TargetFamily              string
	TransitionGroup           string
	ImportArity               string
	ReuseShape                string
	BridgeDepth               string
	SymbolOverlap             string
	LemmaSurfaceStyle         string
	NegativeTwinHardness      string
	CaseNotes                 string
	Stage1Pass                string
	Stage2Pass                string
	Stage3GoldPass            string
	Stage3ModelPass           string
	NegativeTwinPass          string
	AllStagesPass             string
	ImportStagePass           string
	FinalCompositionPassGold  string
	FinalCompositionPassModel string
	ConditionalFinalPassGold  string
	ConditionalFinalPassModel string
	StageSequenceJSON         string
	StageRolesJSON            string
	StagePassesJSON           string
	StageFailuresJSON         string
	TrustedReuseStagesJSON    string
	GoldImportStagesJSON      string
	ModelImportStagesJSON     string
	FailureStage              string
	FailureStageRole          string
	FailureType               string
	FailureClass              string
	FailureDetail             string
	Notes                     string
}

type benchmarkChainAggregate struct {
	chainProtocol        string
	caseFamily           string
	caseFamilies         map[string]struct{}
	chainDepth           int
	atomRenamingID       string
	sourceFamily         string
	targetFamily         string
	transitionGroup      string
	importArity          string
	reuseShape           string
	bridgeDepth          string
	symbolOverlap        string
	lemmaSurfaceStyle    string
	negativeTwinHardness string
	caseNotes            string
	stagePass            map[string]bool
	stageFailure         map[string]string
	stagePresent         map[string]bool
	stageOrder           map[string]int
	stageRole            map[string]string
	stageCase            map[string]benchmarkCase
	stageRow             map[string]benchmarkResultRow
	reusableStages       map[string]bool
	goldStages           map[string]bool
	modelStages          map[string]bool
	negativeStages       map[string]bool
}

type benchmarkChainFailureInfo struct {
	StageID       string
	StageRole     string
	FailureType   string
	FailureClass  string
	FailureDetail string
}

func appendBenchmarkChainSummary(path, runID string, cases []benchmarkCase, rows []benchmarkResultRow) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create benchmark chain summary dir: %w", err)
	}
	summaries := buildBenchmarkChainSummaryRows(runID, cases, rows)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open benchmark chain summary: %w", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	header := []string{
		"run_id", "chain_id", "chain_protocol", "case_family", "chain_depth", "atom_renaming_id",
		"source_family", "target_family", "transition_group", "import_arity", "reuse_shape", "bridge_depth", "symbol_overlap", "lemma_surface_style", "negative_twin_hardness", "case_notes",
		"stage1_pass", "stage2_pass", "stage3_gold_pass", "stage3_model_pass", "negative_twin_pass",
		"all_stages_pass", "import_stage_pass", "final_composition_pass_gold", "final_composition_pass_model",
		"conditional_final_pass_gold", "conditional_final_pass_model",
		"stage_sequence_json", "stage_roles_json", "stage_passes_json", "stage_failures_json",
		"trusted_reuse_stages_json", "gold_import_stages_json", "model_import_stages_json",
		"failure_stage", "failure_stage_role", "failure_type", "failure_class", "failure_detail", "notes",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write benchmark chain summary header: %w", err)
	}
	for _, row := range summaries {
		record := []string{
			row.RunID, row.ChainID, row.ChainProtocol, row.CaseFamily, row.ChainDepth, row.AtomRenamingID,
			row.SourceFamily, row.TargetFamily, row.TransitionGroup, row.ImportArity, row.ReuseShape, row.BridgeDepth, row.SymbolOverlap, row.LemmaSurfaceStyle, row.NegativeTwinHardness, row.CaseNotes,
			row.Stage1Pass, row.Stage2Pass, row.Stage3GoldPass, row.Stage3ModelPass, row.NegativeTwinPass,
			row.AllStagesPass, row.ImportStagePass, row.FinalCompositionPassGold, row.FinalCompositionPassModel,
			row.ConditionalFinalPassGold, row.ConditionalFinalPassModel,
			row.StageSequenceJSON, row.StageRolesJSON, row.StagePassesJSON, row.StageFailuresJSON,
			row.TrustedReuseStagesJSON, row.GoldImportStagesJSON, row.ModelImportStagesJSON,
			row.FailureStage, row.FailureStageRole, row.FailureType, row.FailureClass, row.FailureDetail, row.Notes,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write benchmark chain summary row for chain %s: %w", row.ChainID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush benchmark chain summary csv: %w", err)
	}
	return nil
}

func buildBenchmarkChainSummaryRows(runID string, cases []benchmarkCase, rows []benchmarkResultRow) []benchmarkChainSummaryRow {
	caseByID := make(map[string]benchmarkCase, len(cases))
	chains := make(map[string]*benchmarkChainAggregate)
	for idx, tc := range cases {
		caseByID[tc.CaseID] = tc
		if strings.TrimSpace(tc.ChainID) == "" || strings.TrimSpace(tc.StageID) == "" {
			continue
		}
		group, ok := chains[tc.ChainID]
		if !ok {
			group = &benchmarkChainAggregate{
				chainProtocol:        firstNonEmptyBenchmarkCell(tc.ChainProtocol, "starter-v1"),
				caseFamily:           tc.CaseFamily,
				caseFamilies:         make(map[string]struct{}, 4),
				chainDepth:           tc.ChainDepth,
				atomRenamingID:       tc.AtomRenamingID,
				sourceFamily:         tc.SourceFamily,
				targetFamily:         tc.TargetFamily,
				transitionGroup:      tc.TransitionGroup,
				importArity:          tc.ImportArity,
				reuseShape:           tc.ReuseShape,
				bridgeDepth:          tc.BridgeDepth,
				symbolOverlap:        tc.SymbolOverlap,
				lemmaSurfaceStyle:    tc.LemmaSurfaceStyle,
				negativeTwinHardness: tc.NegativeTwinHardness,
				caseNotes:            tc.Notes,
				stagePass:            make(map[string]bool, 8),
				stageFailure:         make(map[string]string, 8),
				stagePresent:         make(map[string]bool, 8),
				stageOrder:           make(map[string]int, 8),
				stageRole:            make(map[string]string, 8),
				stageCase:            make(map[string]benchmarkCase, 8),
				stageRow:             make(map[string]benchmarkResultRow, 8),
				reusableStages:       make(map[string]bool, 8),
				goldStages:           make(map[string]bool, 8),
				modelStages:          make(map[string]bool, 8),
				negativeStages:       make(map[string]bool, 8),
			}
			chains[tc.ChainID] = group
		}
		stageID := strings.TrimSpace(tc.StageID)
		if stageID == "" {
			continue
		}
		group.stageCase[stageID] = tc
		stageOrder := tc.StageOrder
		if stageOrder <= 0 {
			stageOrder = idx + 1
		}
		group.stageOrder[stageID] = stageOrder
		group.stageRole[stageID] = strings.TrimSpace(tc.StageRole)
		if caseFamily := strings.TrimSpace(tc.CaseFamily); caseFamily != "" {
			group.caseFamilies[caseFamily] = struct{}{}
			if group.caseFamily == "" {
				group.caseFamily = caseFamily
			}
		}
		group.sourceFamily = firstNonEmptyBenchmarkCell(group.sourceFamily, tc.SourceFamily)
		group.targetFamily = firstNonEmptyBenchmarkCell(group.targetFamily, tc.TargetFamily)
		group.transitionGroup = firstNonEmptyBenchmarkCell(group.transitionGroup, tc.TransitionGroup)
		group.importArity = firstNonEmptyBenchmarkCell(group.importArity, tc.ImportArity)
		group.reuseShape = firstNonEmptyBenchmarkCell(group.reuseShape, tc.ReuseShape)
		group.bridgeDepth = firstNonEmptyBenchmarkCell(group.bridgeDepth, tc.BridgeDepth)
		group.symbolOverlap = firstNonEmptyBenchmarkCell(group.symbolOverlap, tc.SymbolOverlap)
		group.lemmaSurfaceStyle = firstNonEmptyBenchmarkCell(group.lemmaSurfaceStyle, tc.LemmaSurfaceStyle)
		group.negativeTwinHardness = firstNonEmptyBenchmarkCell(group.negativeTwinHardness, tc.NegativeTwinHardness)
		group.caseNotes = firstNonEmptyBenchmarkCell(group.caseNotes, tc.Notes)
		group.reusableStages[stageID] = benchmarkCaseTrustedReuseEligible(tc)
		group.negativeStages[stageID] = strings.EqualFold(strings.TrimSpace(tc.Label), "not_entailed") || strings.EqualFold(strings.TrimSpace(tc.ExpectedBehavior), "not_derivable") || strings.EqualFold(strings.TrimSpace(tc.StageRole), "negative_control")
		stageGroup := benchmarkChainStageGroup(stageID, tc.StageRole)
		provenanceMode := strings.ToLower(strings.TrimSpace(tc.ProvenanceMode))
		group.goldStages[stageID] = stageGroup == "import" && !group.negativeStages[stageID] && provenanceMode == "gold"
		group.modelStages[stageID] = stageGroup == "import" && !group.negativeStages[stageID] && (provenanceMode == "model" || provenanceMode == "semi_gold")
	}
	for _, row := range rows {
		tc, ok := caseByID[row.CaseID]
		if !ok || strings.TrimSpace(tc.ChainID) == "" {
			continue
		}
		group, ok := chains[tc.ChainID]
		if !ok {
			continue
		}
		stageID := strings.TrimSpace(firstNonEmptyBenchmarkCell(tc.StageID, row.StageID))
		if stageID == "" {
			continue
		}
		group.stagePresent[stageID] = true
		group.stagePass[stageID] = strings.TrimSpace(row.ScoreBucket) == "pass"
		group.stageFailure[stageID] = strings.TrimSpace(row.ScoreBucket)
		group.stageRow[stageID] = row
	}
	chainIDs := make([]string, 0, len(chains))
	for chainID := range chains {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	result := make([]benchmarkChainSummaryRow, 0, len(chainIDs))
	for _, chainID := range chainIDs {
		group := chains[chainID]
		stageSequence := benchmarkOrderedChainStageIDs(group)
		stagePassesJSON := make(map[string]bool, len(stageSequence))
		stageFailuresJSON := make(map[string]string, len(stageSequence))
		stageRolesJSON := make(map[string]string, len(stageSequence))
		reusableStageIDs := make([]string, 0, len(stageSequence))
		goldStageIDs := make([]string, 0, len(stageSequence))
		modelStageIDs := make([]string, 0, len(stageSequence))
		negativeStageIDs := make([]string, 0, len(stageSequence))
		for _, stageID := range stageSequence {
			stagePassesJSON[stageID] = group.stagePresent[stageID] && group.stagePass[stageID]
			stageFailuresJSON[stageID] = benchmarkChainStageFailureValue(group, stageID)
			if role := strings.TrimSpace(group.stageRole[stageID]); role != "" {
				stageRolesJSON[stageID] = role
			}
			if group.reusableStages[stageID] {
				reusableStageIDs = append(reusableStageIDs, stageID)
			}
			if group.goldStages[stageID] {
				goldStageIDs = append(goldStageIDs, stageID)
			}
			if group.modelStages[stageID] {
				modelStageIDs = append(modelStageIDs, stageID)
			}
			if group.negativeStages[stageID] {
				negativeStageIDs = append(negativeStageIDs, stageID)
			}
		}
		stage1Pass := benchmarkChainLegacyStagePass(group, "s1", stageSequence, 0)
		stage2Pass := benchmarkChainLegacyStagePass(group, "s2", stageSequence, 1)
		finalCompositionPassGold := benchmarkLastChainStagePass(group, goldStageIDs)
		finalCompositionPassModel := benchmarkLastChainStagePass(group, modelStageIDs)
		stage3GoldPass := benchmarkChainNamedOrFallbackStagePass(group, "s3g", finalCompositionPassGold)
		stage3ModelPass := benchmarkChainNamedOrFallbackStagePass(group, "s3m", finalCompositionPassModel)
		negativeTwinPass := benchmarkChainNamedOrFallbackStagePass(group, "neg", benchmarkAllChainStagesPass(group, negativeStageIDs))
		importStagePass := benchmarkAllChainStagesPass(group, reusableStageIDs)
		allStagesPass := benchmarkAllChainStagesPass(group, stageSequence)
		failure := benchmarkFirstChainFailure(group)
		result = append(result, benchmarkChainSummaryRow{
			RunID:                     runID,
			ChainID:                   chainID,
			ChainProtocol:             group.chainProtocol,
			CaseFamily:                benchmarkResolvedChainCaseFamily(group),
			ChainDepth:                strconv.Itoa(group.chainDepth),
			AtomRenamingID:            group.atomRenamingID,
			SourceFamily:              group.sourceFamily,
			TargetFamily:              group.targetFamily,
			TransitionGroup:           group.transitionGroup,
			ImportArity:               group.importArity,
			ReuseShape:                group.reuseShape,
			BridgeDepth:               group.bridgeDepth,
			SymbolOverlap:             group.symbolOverlap,
			LemmaSurfaceStyle:         group.lemmaSurfaceStyle,
			NegativeTwinHardness:      group.negativeTwinHardness,
			CaseNotes:                 group.caseNotes,
			Stage1Pass:                strconv.FormatBool(stage1Pass),
			Stage2Pass:                strconv.FormatBool(stage2Pass),
			Stage3GoldPass:            strconv.FormatBool(stage3GoldPass),
			Stage3ModelPass:           strconv.FormatBool(stage3ModelPass),
			NegativeTwinPass:          strconv.FormatBool(negativeTwinPass),
			AllStagesPass:             strconv.FormatBool(allStagesPass),
			ImportStagePass:           strconv.FormatBool(importStagePass),
			FinalCompositionPassGold:  strconv.FormatBool(finalCompositionPassGold),
			FinalCompositionPassModel: strconv.FormatBool(finalCompositionPassModel),
			ConditionalFinalPassGold:  strconv.FormatBool(importStagePass && finalCompositionPassGold),
			ConditionalFinalPassModel: strconv.FormatBool(importStagePass && finalCompositionPassModel),
			StageSequenceJSON:         benchmarkJSONString(stageSequence),
			StageRolesJSON:            benchmarkJSONString(stageRolesJSON),
			StagePassesJSON:           benchmarkJSONString(stagePassesJSON),
			StageFailuresJSON:         benchmarkJSONString(stageFailuresJSON),
			TrustedReuseStagesJSON:    benchmarkJSONString(reusableStageIDs),
			GoldImportStagesJSON:      benchmarkJSONString(goldStageIDs),
			ModelImportStagesJSON:     benchmarkJSONString(modelStageIDs),
			FailureStage:              failure.StageID,
			FailureStageRole:          failure.StageRole,
			FailureType:               failure.FailureType,
			FailureClass:              failure.FailureClass,
			FailureDetail:             failure.FailureDetail,
			Notes:                     benchmarkChainSummaryNote(importStagePass, finalCompositionPassGold, finalCompositionPassModel, negativeTwinPass, allStagesPass, len(reusableStageIDs), len(goldStageIDs), len(modelStageIDs), len(negativeStageIDs)),
		})
	}
	return result
}

func benchmarkResolvedChainCaseFamily(group *benchmarkChainAggregate) string {
	if group == nil {
		return ""
	}
	if len(group.caseFamilies) > 1 {
		return "mixed_family_reuse"
	}
	return group.caseFamily
}

func benchmarkChainStagePassed(group *benchmarkChainAggregate, stageID string) bool {
	stageID = strings.TrimSpace(stageID)
	return group != nil && stageID != "" && group.stagePresent[stageID] && group.stagePass[stageID]
}

func benchmarkAllChainStagesPass(group *benchmarkChainAggregate, stageIDs []string) bool {
	if group == nil || len(stageIDs) == 0 {
		return false
	}
	for _, stageID := range stageIDs {
		if !benchmarkChainStagePassed(group, stageID) {
			return false
		}
	}
	return true
}

func benchmarkLastChainStagePass(group *benchmarkChainAggregate, stageIDs []string) bool {
	if len(stageIDs) == 0 {
		return false
	}
	return benchmarkChainStagePassed(group, stageIDs[len(stageIDs)-1])
}

func benchmarkChainLegacyStagePass(group *benchmarkChainAggregate, namedStageID string, stageSequence []string, fallbackIndex int) bool {
	if benchmarkChainStagePassed(group, namedStageID) {
		return true
	}
	if group == nil || fallbackIndex < 0 || fallbackIndex >= len(stageSequence) {
		return false
	}
	return benchmarkChainStagePassed(group, stageSequence[fallbackIndex])
}

func benchmarkChainNamedOrFallbackStagePass(group *benchmarkChainAggregate, namedStageID string, fallback bool) bool {
	if benchmarkChainStagePassed(group, namedStageID) {
		return true
	}
	return fallback
}

func benchmarkOrderedChainStageIDs(group *benchmarkChainAggregate) []string {
	if group == nil {
		return nil
	}
	stageIDs := make([]string, 0, len(group.stageOrder))
	for stageID := range group.stageOrder {
		stageIDs = append(stageIDs, stageID)
	}
	slices.SortFunc(stageIDs, func(left, right string) int {
		leftOrder := group.stageOrder[left]
		rightOrder := group.stageOrder[right]
		switch {
		case leftOrder < rightOrder:
			return -1
		case leftOrder > rightOrder:
			return 1
		default:
			return strings.Compare(left, right)
		}
	})
	if len(stageIDs) == 0 {
		return []string{"s1", "s2", "s3g", "s3m", "neg"}
	}
	return stageIDs
}

func benchmarkChainStageFailureValue(group *benchmarkChainAggregate, stageID string) string {
	switch {
	case group == nil:
		return "missing_stage"
	case !group.stagePresent[stageID]:
		return "missing_stage"
	case strings.TrimSpace(group.stageFailure[stageID]) == "":
		return "unknown_failure"
	default:
		return strings.TrimSpace(group.stageFailure[stageID])
	}
}

func benchmarkFirstChainFailure(group *benchmarkChainAggregate) benchmarkChainFailureInfo {
	for _, stageID := range benchmarkOrderedChainStageIDs(group) {
		tc := group.stageCase[stageID]
		stageRole := strings.TrimSpace(firstNonEmptyBenchmarkCell(group.stageRole[stageID], tc.StageRole))
		if !group.stagePresent[stageID] {
			failureClass, failureDetail := benchmarkClassifyChainStageFailure(tc, benchmarkResultRow{}, false)
			return benchmarkChainFailureInfo{
				StageID:       stageID,
				StageRole:     stageRole,
				FailureType:   "missing_stage",
				FailureClass:  failureClass,
				FailureDetail: failureDetail,
			}
		}
		if !group.stagePass[stageID] {
			row := group.stageRow[stageID]
			failureType := strings.TrimSpace(firstNonEmptyBenchmarkCell(row.ScoreBucket, group.stageFailure[stageID], "unknown_failure"))
			failureClass, failureDetail := benchmarkClassifyChainStageFailure(tc, row, true)
			return benchmarkChainFailureInfo{
				StageID:       stageID,
				StageRole:     stageRole,
				FailureType:   failureType,
				FailureClass:  failureClass,
				FailureDetail: failureDetail,
			}
		}
	}
	return benchmarkChainFailureInfo{}
}

func benchmarkClassifyChainStageFailure(tc benchmarkCase, row benchmarkResultRow, stagePresent bool) (string, string) {
	stageID := strings.TrimSpace(firstNonEmptyBenchmarkCell(tc.StageID, row.StageID))
	stageRole := strings.TrimSpace(tc.StageRole)
	stageGroup := benchmarkChainStageGroup(stageID, stageRole)
	failureType := strings.TrimSpace(row.ScoreBucket)
	if !stagePresent {
		failureType = "missing_stage"
	}
	switch {
	case stageGroup == "negative":
		return "negative_control_failure", fmt.Sprintf("negative-control stage `%s` failed with `%s`", stageID, failureType)
	case stageGroup == "starter":
		return "semantic_transport_failure", fmt.Sprintf("prerequisite transport stage `%s` (`%s`) failed with `%s`", stageID, valueOrUnknownChainField(stageRole), failureType)
	}
	if ok, detail := benchmarkImportBindingFailureDetail(tc, row, stagePresent); ok {
		return "import_binding_failure", detail
	}
	switch {
	case benchmarkScoreBucketIsAssemblyFailure(failureType):
		return "certificate_assembly_failure", fmt.Sprintf("stage `%s` failed before a verifier-acceptable certificate could be assembled (`%s`)", stageID, failureType)
	case benchmarkScoreBucketIsClosureFailure(failureType):
		return "final_proof_closure_failure", fmt.Sprintf("stage `%s` reached proof-closure failure after import materialization (`%s`)", stageID, failureType)
	case !stagePresent:
		return "other_failure", fmt.Sprintf("stage `%s` is missing from the result set", stageID)
	default:
		return "other_failure", fmt.Sprintf("stage `%s` failed with `%s`", stageID, failureType)
	}
}

func benchmarkImportBindingFailureDetail(tc benchmarkCase, row benchmarkResultRow, stagePresent bool) (bool, string) {
	mode := strings.ToLower(strings.TrimSpace(tc.ProvenanceMode))
	if mode != "model" && mode != "semi_gold" {
		return false, ""
	}
	requestedStages := benchmarkTrimmedStageIDs(tc.ImportStageIDs)
	resolvedRefs := benchmarkParseImportedArtifactRefsJSON(row.ResolvedImportRefsJSON)
	resolvedStages := make(map[string]struct{}, len(resolvedRefs))
	resolvedStageIDs := make([]string, 0, len(resolvedRefs))
	for _, ref := range resolvedRefs {
		stageID := strings.TrimSpace(ref.StageID)
		if stageID == "" {
			continue
		}
		key := strings.ToLower(stageID)
		if _, ok := resolvedStages[key]; ok {
			continue
		}
		resolvedStages[key] = struct{}{}
		resolvedStageIDs = append(resolvedStageIDs, stageID)
	}
	if len(requestedStages) > 0 {
		missingStages := make([]string, 0, len(requestedStages))
		for _, stageID := range requestedStages {
			if _, ok := resolvedStages[strings.ToLower(stageID)]; !ok {
				missingStages = append(missingStages, stageID)
			}
		}
		if len(missingStages) > 0 {
			if !stagePresent {
				return true, fmt.Sprintf("import/final stage is missing and never bound requested prerequisite stage(s): `%s`", strings.Join(missingStages, "`, `"))
			}
			return true, fmt.Sprintf("resolved import stages `%s`; missing requested prerequisite stage(s) `%s`", strings.Join(resolvedStageIDs, "`, `"), strings.Join(missingStages, "`, `"))
		}
	}
	if !stagePresent {
		return false, ""
	}
	effectiveImports := parseBenchmarkJSONStringSlice(row.EffectiveImportsJSON)
	requestedImports := parseBenchmarkJSONStringSlice(row.RequestedImportsJSON)
	if len(requestedStages) == 0 && len(requestedImports) > 0 && len(effectiveImports) < len(requestedImports) {
		return true, fmt.Sprintf("effective imported lemmas %d < requested imported lemmas %d", len(effectiveImports), len(requestedImports))
	}
	return false, ""
}

func benchmarkTrimmedStageIDs(stageIDs []string) []string {
	if len(stageIDs) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(stageIDs))
	seen := make(map[string]struct{}, len(stageIDs))
	for _, stageID := range stageIDs {
		stageID = strings.TrimSpace(stageID)
		if stageID == "" {
			continue
		}
		key := strings.ToLower(stageID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		trimmed = append(trimmed, stageID)
	}
	return trimmed
}

func benchmarkParseImportedArtifactRefsJSON(raw string) []benchmarkImportedArtifactRef {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var refs []benchmarkImportedArtifactRef
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil
	}
	return refs
}

func benchmarkScoreBucketIsAssemblyFailure(score string) bool {
	switch strings.TrimSpace(score) {
	case "request_failure", "format_failure", "schema_failure", "parse_failure", "contract_failure":
		return true
	default:
		return false
	}
}

func benchmarkScoreBucketIsClosureFailure(score string) bool {
	switch strings.TrimSpace(score) {
	case "kernel_failure", "false_refusal", "false_accept":
		return true
	default:
		return false
	}
}

func valueOrUnknownChainField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown_stage_role"
	}
	return value
}

func benchmarkChainSummaryNote(importStagePass, finalGoldPass, finalModelPass, negativeTwinPass, allStagesPass bool, reusableStageCount, goldStageCount, modelStageCount, negativeStageCount int) string {
	switch {
	case reusableStageCount > 0 && !importStagePass:
		return "Trusted reuse prerequisite stages did not all pass."
	case goldStageCount > 0 && !finalGoldPass:
		return "Gold-import composition failed at the final target stage."
	case modelStageCount > 0 && !finalModelPass:
		return "Model-import composition failed at the final target stage."
	case negativeStageCount > 0 && !negativeTwinPass:
		return "Negative control failed; conservative compositional discrimination is not intact."
	case !allStagesPass:
		return "At least one chain stage failed outside the primary final target stages."
	default:
		return "All observed compositional chain stages passed."
	}
}
