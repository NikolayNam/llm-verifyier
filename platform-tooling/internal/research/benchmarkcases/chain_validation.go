package benchmarkcases

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ChainCase struct {
	CaseID         string
	ChainID        string
	ChainProtocol  string
	StageID        string
	StageOrder     int
	StageRole      string
	ProvenanceMode string
	ImportStageIDs []string
}

func LoadChainCasesCSV(path string) ([]ChainCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open benchmark cases: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read benchmark cases csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("benchmark cases csv must contain a header and at least one row")
	}

	header := make(map[string]int, len(records[0]))
	for idx, name := range records[0] {
		header[strings.TrimSpace(name)] = idx
	}
	if _, ok := header["case_id"]; !ok {
		if _, ok := header["theorem_id"]; !ok {
			return nil, fmt.Errorf("benchmark cases csv must contain either %q or %q", "case_id", "theorem_id")
		}
	}

	cases := make([]ChainCase, 0, len(records)-1)
	for rowIdx, record := range records[1:] {
		importStageIDs := make([]string, 0)
		if _, ok := header["import_stage_ids_json"]; ok {
			if raw := strings.TrimSpace(csvCell(record, header, "import_stage_ids_json")); raw != "" {
				if err := json.Unmarshal([]byte(raw), &importStageIDs); err != nil {
					return nil, fmt.Errorf("benchmark case row %d import_stage_ids_json: %w", rowIdx+2, err)
				}
			}
		}
		caseID := firstNonEmpty(csvCell(record, header, "case_id"), csvCell(record, header, "theorem_id"))
		cases = append(cases, ChainCase{
			CaseID:         caseID,
			ChainID:        csvCell(record, header, "chain_id"),
			ChainProtocol:  csvCell(record, header, "chain_protocol"),
			StageID:        csvCell(record, header, "stage_id"),
			StageOrder:     parseIntLoose(csvCell(record, header, "stage_order"), rowIdx+1),
			StageRole:      csvCell(record, header, "stage_role"),
			ProvenanceMode: csvCell(record, header, "provenance_mode"),
			ImportStageIDs: importStageIDs,
		})
	}
	return cases, nil
}

func ValidateCompositionalDependencies(cases []ChainCase) error {
	type chainStage struct {
		CaseID     string
		StageOrder int
	}

	byChain := make(map[string]map[string]chainStage, len(cases))
	stageOwners := make(map[string]map[string]struct{}, len(cases))

	for idx, c := range cases {
		if !UsesChainProtocol(c) {
			continue
		}
		chainKey := strings.ToLower(strings.TrimSpace(c.ChainID))
		stageKey := strings.ToLower(strings.TrimSpace(c.StageID))
		if chainKey == "" || stageKey == "" {
			continue
		}
		if _, ok := byChain[chainKey]; !ok {
			byChain[chainKey] = make(map[string]chainStage, 8)
		}
		byChain[chainKey][stageKey] = chainStage{
			CaseID:     strings.TrimSpace(c.CaseID),
			StageOrder: effectiveStageOrder(c.StageOrder, idx+1),
		}
		if _, ok := stageOwners[stageKey]; !ok {
			stageOwners[stageKey] = make(map[string]struct{}, 4)
		}
		stageOwners[stageKey][chainKey] = struct{}{}
	}

	for idx, c := range cases {
		if !UsesChainProtocol(c) {
			continue
		}
		mode := strings.ToLower(strings.TrimSpace(c.ProvenanceMode))
		if mode != "model" && mode != "semi_gold" {
			continue
		}
		requested := trimStageIDs(c.ImportStageIDs)
		if len(requested) == 0 {
			return fmt.Errorf("missing_import_stage: chain_id=%s case_id=%s provenance_mode=%s requires explicit import_stage_ids_json", strings.TrimSpace(c.ChainID), strings.TrimSpace(c.CaseID), mode)
		}
		chainKey := strings.ToLower(strings.TrimSpace(c.ChainID))
		currentOrder := effectiveStageOrder(c.StageOrder, idx+1)
		for _, requestedStageID := range requested {
			stageKey := strings.ToLower(requestedStageID)
			stage, ok := byChain[chainKey][stageKey]
			if !ok {
				if owners := stageOwners[stageKey]; len(owners) > 0 {
					return fmt.Errorf("cross_chain_import: chain_id=%s case_id=%s stage_id=%s referenced stage is absent from the local chain slice and only appears in another chain", strings.TrimSpace(c.ChainID), strings.TrimSpace(c.CaseID), requestedStageID)
				}
				return fmt.Errorf("missing_import_stage: chain_id=%s case_id=%s stage_id=%s referenced stage is absent from the selected slice", strings.TrimSpace(c.ChainID), strings.TrimSpace(c.CaseID), requestedStageID)
			}
			if stage.StageOrder >= currentOrder {
				return fmt.Errorf("future_import_stage: chain_id=%s case_id=%s stage_id=%s referenced stage order %d must be earlier than current stage order %d", strings.TrimSpace(c.ChainID), strings.TrimSpace(c.CaseID), requestedStageID, stage.StageOrder, currentOrder)
			}
		}
	}

	return nil
}

func UsesChainProtocol(c ChainCase) bool {
	return strings.TrimSpace(c.ChainID) != "" && strings.TrimSpace(c.StageID) != ""
}

func trimStageIDs(stageIDs []string) []string {
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

func csvCell(record []string, header map[string]int, name string) string {
	idx, ok := header[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func effectiveStageOrder(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parseIntLoose(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value <= 0 {
		return fallback
	}
	return value
}
