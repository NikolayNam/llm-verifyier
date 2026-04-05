package compositional

import (
	"fmt"
	"strings"
)

func generateBridgeImportFinalRows(caseCount int) ([]CaseRow, error) {
	switch {
	case caseCount == 0 || caseCount == BridgeImportFinalPhase1CaseCount:
		return generateBridgeImportFinalPhase1Rows(), nil
	case caseCount < BridgeImportFinalPhase1CaseCount:
		return nil, fmt.Errorf("bridge-import-final-research case_count must be either 0 or >= %d", BridgeImportFinalPhase1CaseCount)
	case caseCount%5 != 0:
		return nil, fmt.Errorf("bridge-import-final-research case_count must be a multiple of 5 because each chain emits b1/b2/fg/fm/neg")
	default:
		return generateBridgeImportFinalExpandedRows(caseCount / 5), nil
	}
}

func generateBridgeImportFinalPhase1Rows() []CaseRow {
	return []CaseRow{
		bridgeFinalRow("BIF01", "b1", 1, "bridge_left", "entailed", "easy", "layered_linear", "none", "r0", true, nil, nil, "A -> B", "prove", "Left bridge lemma"),
		bridgeFinalRow("BIF01", "b2", 2, "bridge_right", "entailed", "easy", "layered_linear", "none", "r0", true, nil, nil, "B -> C", "prove", "Right bridge lemma"),
		bridgeFinalRow("BIF01", "fg", 3, "final_gold", "entailed", "easy", "layered_linear", "gold", "r0", false, []string{"b1", "b2"}, nil, "A -> C", "prove", "Gold final closure from imported bridge lemmas", "A -> B", "B -> C"),
		bridgeFinalRow("BIF01", "fm", 4, "final_model", "entailed", "easy", "layered_linear", "model", "r0", false, []string{"b1", "b2"}, nil, "A -> C", "prove", "Model final closure from imported bridge lemmas", "A -> B", "B -> C"),
		bridgeFinalRow("BIF01", "neg", 5, "negative_control", "not_entailed", "easy", "layered_linear_negative", "gold", "r0", false, []string{"b1", "b2"}, nil, "A -> D", "not_derivable", "Negative control for layered linear chain", "A -> B", "B -> C"),

		bridgeFinalRow("BIF02", "b1", 1, "bridge_left", "entailed", "easy", "layered_linear_renamed", "none", "r1", true, nil, nil, "P -> Q", "prove", "Renamed left bridge lemma"),
		bridgeFinalRow("BIF02", "b2", 2, "bridge_right", "entailed", "easy", "layered_linear_renamed", "none", "r1", true, nil, nil, "Q -> R", "prove", "Renamed right bridge lemma"),
		bridgeFinalRow("BIF02", "fg", 3, "final_gold", "entailed", "easy", "layered_linear_renamed", "gold", "r1", false, []string{"b1", "b2"}, nil, "P -> R", "prove", "Gold final closure under atom renaming", "P -> Q", "Q -> R"),
		bridgeFinalRow("BIF02", "fm", 4, "final_model", "entailed", "easy", "layered_linear_renamed", "model", "r1", false, []string{"b1", "b2"}, nil, "P -> R", "prove", "Model final closure under atom renaming", "P -> Q", "Q -> R"),
		bridgeFinalRow("BIF02", "neg", 5, "negative_control", "not_entailed", "easy", "layered_linear_negative", "gold", "r1", false, []string{"b1", "b2"}, nil, "P -> S", "not_derivable", "Renamed negative control", "P -> Q", "Q -> R"),

		bridgeFinalRow("BIF03", "b1", 1, "bridge_left", "entailed", "medium", "layered_distractor", "none", "r2", true, nil, nil, "M -> N", "prove", "Left bridge lemma with later distractor context"),
		bridgeFinalRow("BIF03", "b2", 2, "bridge_right", "entailed", "medium", "layered_distractor", "none", "r2", true, nil, nil, "N -> O", "prove", "Right bridge lemma with later distractor context"),
		bridgeFinalRow("BIF03", "fg", 3, "final_gold", "entailed", "medium", "layered_distractor", "gold", "r2", false, []string{"b1", "b2"}, []string{"P -> Q"}, "M -> O", "prove", "Gold final closure with irrelevant direct assumption", "M -> N", "N -> O"),
		bridgeFinalRow("BIF03", "fm", 4, "final_model", "entailed", "medium", "layered_distractor", "model", "r2", false, []string{"b1", "b2"}, []string{"P -> Q"}, "M -> O", "prove", "Model final closure with irrelevant direct assumption", "M -> N", "N -> O"),
		bridgeFinalRow("BIF03", "neg", 5, "negative_control", "not_entailed", "medium", "layered_distractor_negative", "gold", "r2", false, []string{"b1", "b2"}, []string{"P -> Q"}, "M -> R", "not_derivable", "Distractor negative control", "M -> N", "Q -> R"),

		bridgeFinalRow("BIF04", "b1", 1, "bridge_left", "entailed", "medium", "layered_negated_bridge", "none", "neg1", true, nil, nil, "!X -> Y", "prove", "Negated antecedent left bridge lemma"),
		bridgeFinalRow("BIF04", "b2", 2, "bridge_right", "entailed", "medium", "layered_negated_bridge", "none", "neg1", true, nil, nil, "Y -> Z", "prove", "Negated antecedent right bridge lemma"),
		bridgeFinalRow("BIF04", "fg", 3, "final_gold", "entailed", "medium", "layered_negated_bridge", "gold", "neg1", false, []string{"b1", "b2"}, nil, "!X -> Z", "prove", "Gold final closure with negated antecedent", "!X -> Y", "Y -> Z"),
		bridgeFinalRow("BIF04", "fm", 4, "final_model", "entailed", "medium", "layered_negated_bridge", "model", "neg1", false, []string{"b1", "b2"}, nil, "!X -> Z", "prove", "Model final closure with negated antecedent", "!X -> Y", "Y -> Z"),
		bridgeFinalRow("BIF04", "neg", 5, "negative_control", "not_entailed", "medium", "layered_negated_bridge_negative", "gold", "neg1", false, []string{"b1", "b2"}, nil, "!X -> W", "not_derivable", "Negated antecedent negative control", "!X -> Y", "Y -> Z"),
	}
}

func generateBridgeImportFinalExpandedRows(chainCount int) []CaseRow {
	rows := make([]CaseRow, 0, chainCount*5)
	for chain := 1; chain <= chainCount; chain++ {
		rows = append(rows, bridgeImportFinalExpandedChain(chain)...)
	}
	return rows
}

func bridgeImportFinalExpandedChain(chain int) []CaseRow {
	id := fmt.Sprintf("%03d", chain)
	chainID := fmt.Sprintf("BIFX%s", id)
	switch (chain - 1) % 6 {
	case 0:
		left := "A" + id + " -> B" + id
		right := "B" + id + " -> C" + id
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"easy",
			"layered_linear_exact",
			"r0",
			nil,
			left,
			right,
			"A"+id+" -> C"+id,
			"A"+id+" -> D"+id,
			"Exact left bridge lemma",
			"Exact right bridge lemma",
			"Gold final closure over exact bridge lemmas",
			"Model final closure over exact bridge lemmas",
			"Exact negative control outside the imported path",
		)
	case 1:
		left := "P" + id + " -> Q" + id
		right := "Q" + id + " -> R" + id
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"easy",
			"layered_linear_renamed",
			"r1",
			nil,
			left,
			right,
			"P"+id+" -> R"+id,
			"P"+id+" -> S"+id,
			"Renamed left bridge lemma",
			"Renamed right bridge lemma",
			"Gold final closure under renamed atoms",
			"Model final closure under renamed atoms",
			"Renamed negative control outside the imported path",
		)
	case 2:
		left := "M" + id + " -> N" + id
		right := "N" + id + " -> O" + id
		assumptions := []string{"X" + id + " -> Y" + id}
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"medium",
			"layered_distractor",
			"r2",
			assumptions,
			left,
			right,
			"M"+id+" -> O"+id,
			"M"+id+" -> R"+id,
			"Left bridge lemma with irrelevant later distractor context",
			"Right bridge lemma with irrelevant later distractor context",
			"Gold final closure with an irrelevant direct assumption present",
			"Model final closure with an irrelevant direct assumption present",
			"Distractor negative control that should remain underivable",
		)
	case 3:
		left := "!X" + id + " -> Y" + id
		right := "Y" + id + " -> Z" + id
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"medium",
			"layered_negated_bridge",
			"neg1",
			nil,
			left,
			right,
			"!X"+id+" -> Z"+id,
			"!X"+id+" -> W"+id,
			"Negated-antecedent left bridge lemma",
			"Negated-antecedent right bridge lemma",
			"Gold final closure with a negated antecedent",
			"Model final closure with a negated antecedent",
			"Negated-antecedent negative control outside the imported path",
		)
	case 4:
		left := "A" + id + " -> (B" + id + " -> C" + id + ")"
		right := "(B" + id + " -> C" + id + ") -> D" + id
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"hard",
			"layered_nested_middle",
			"h1",
			nil,
			left,
			right,
			"A"+id+" -> D"+id,
			"A"+id+" -> E"+id,
			"Nested left bridge lemma whose consequent is itself implicational",
			"Nested right bridge lemma that consumes the imported middle formula",
			"Gold final closure over a nested middle surface",
			"Model final closure over a nested middle surface",
			"Nested negative control outside the imported path",
		)
	default:
		left := "(A" + id + " -> B" + id + ") -> C" + id
		right := "C" + id + " -> (D" + id + " -> E" + id + ")"
		return bridgeFinalChainRows(
			chainID,
			"bridge-import-final-v1-expanded",
			"hard",
			"layered_compound_consequent",
			"h2",
			nil,
			left,
			right,
			"(A"+id+" -> B"+id+") -> (D"+id+" -> E"+id+")",
			"(A"+id+" -> B"+id+") -> (F"+id+" -> G"+id+")",
			"Compound left bridge lemma that exposes a derived middle certificate",
			"Compound right bridge lemma with an implicational consequent",
			"Gold final closure over compound imported bridge lemmas",
			"Model final closure over compound imported bridge lemmas",
			"Compound negative control outside the imported path",
		)
	}
}

func bridgeFinalChainRows(chainID, protocol, difficulty, caseFamily, atomRenamingID string, assumptions []string, leftGoal, rightGoal, finalGoal, negativeGoal, leftComment, rightComment, goldComment, modelComment, negativeComment string) []CaseRow {
	imported := []string{leftGoal, rightGoal}
	return []CaseRow{
		bridgeFinalRowWithProtocol(chainID, protocol, "b1", 1, "bridge_left", "entailed", difficulty, caseFamily, "none", atomRenamingID, true, nil, nil, leftGoal, "prove", leftComment),
		bridgeFinalRowWithProtocol(chainID, protocol, "b2", 2, "bridge_right", "entailed", difficulty, caseFamily, "none", atomRenamingID, true, nil, nil, rightGoal, "prove", rightComment),
		bridgeFinalRowWithProtocol(chainID, protocol, "fg", 3, "final_gold", "entailed", difficulty, caseFamily, "gold", atomRenamingID, false, []string{"b1", "b2"}, assumptions, finalGoal, "prove", goldComment, imported...),
		bridgeFinalRowWithProtocol(chainID, protocol, "fm", 4, "final_model", "entailed", difficulty, caseFamily, "model", atomRenamingID, false, []string{"b1", "b2"}, assumptions, finalGoal, "prove", modelComment, imported...),
		bridgeFinalRowWithProtocol(chainID, protocol, "neg", 5, "negative_control", "not_entailed", difficulty, caseFamily+"_negative", "gold", atomRenamingID, false, []string{"b1", "b2"}, assumptions, negativeGoal, "not_derivable", negativeComment, imported...),
	}
}

func bridgeFinalRow(chainID, stageID string, stageOrder int, stageRole, label, difficulty, caseFamily, provenanceMode, atomRenamingID string, trustedReuse bool, importStageIDs, assumptions []string, goal, expectedBehavior, comment string, importedLemmas ...string) CaseRow {
	return bridgeFinalRowWithProtocol(chainID, "bridge-import-final-v1", stageID, stageOrder, stageRole, label, difficulty, caseFamily, provenanceMode, atomRenamingID, trustedReuse, importStageIDs, assumptions, goal, expectedBehavior, comment, importedLemmas...)
}

func bridgeFinalRowWithProtocol(chainID, protocol, stageID string, stageOrder int, stageRole, label, difficulty, caseFamily, provenanceMode, atomRenamingID string, trustedReuse bool, importStageIDs, assumptions []string, goal, expectedBehavior, comment string, importedLemmas ...string) CaseRow {
	return CaseRow{
		CaseID:           chainID + "-" + strings.ToUpper(stageID),
		ChainID:          chainID,
		ChainProtocol:    protocol,
		StageID:          stageID,
		StageOrder:       stageOrder,
		StageRole:        stageRole,
		Category:         "bridge_import_final_research",
		Label:            label,
		Difficulty:       difficulty,
		CaseFamily:       caseFamily,
		ChainDepth:       1,
		ProvenanceMode:   provenanceMode,
		AtomRenamingID:   atomRenamingID,
		TrustedReuse:     boolPtr(trustedReuse),
		ImportStageIDs:   append([]string(nil), importStageIDs...),
		Assumptions:      append([]string(nil), assumptions...),
		ImportedLemmas:   append([]string(nil), importedLemmas...),
		Goal:             goal,
		ExpectedBehavior: expectedBehavior,
		Comment:          comment,
	}
}

func generateBridgeOnlyPhase1Rows() []CaseRow {
	return []CaseRow{
		bridgeOnlyRow("BOA01", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "easy", "layered_linear", "r0", "BOA-LAYERED", "exact", "local_assumption_lift", "a1_lift_bridge_left", []string{"B"}, "A -> B", "Derive left bridge lemma from local consequent assumption via A1"),
		bridgeOnlyRow("BOA01", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "easy", "layered_linear", "r0", "BOA-LAYERED", "exact", "local_assumption_lift", "a1_lift_bridge_right", []string{"C"}, "B -> C", "Derive right bridge lemma from local consequent assumption via A1"),
		bridgeOnlyRow("BOA02", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "easy", "layered_linear_renamed", "r1", "BOA-RENAMED", "renamed", "local_assumption_lift", "a1_lift_bridge_left", []string{"Q"}, "P -> Q", "Renamed left bridge lemma from local consequent assumption"),
		bridgeOnlyRow("BOA02", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "easy", "layered_linear_renamed", "r1", "BOA-RENAMED", "renamed", "local_assumption_lift", "a1_lift_bridge_right", []string{"R"}, "Q -> R", "Renamed right bridge lemma from local consequent assumption"),
		bridgeOnlyRow("BOA03", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "medium", "layered_distractor", "r2", "BOA-DISTRACTOR", "distractor", "local_assumption_lift", "distractor_present", []string{"N", "P -> Q"}, "M -> N", "Bridge lemma with irrelevant extra assumption that should be ignored"),
		bridgeOnlyRow("BOA03", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "medium", "layered_distractor", "r2", "BOA-DISTRACTOR", "distractor", "local_assumption_lift", "distractor_present", []string{"O", "P -> Q"}, "N -> O", "Right bridge lemma with irrelevant extra assumption that should be ignored"),
		bridgeOnlyRow("BOA04", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "medium", "layered_negated_bridge", "neg1", "BOA-NEGATED", "negated", "local_assumption_lift", "negated_antecedent", []string{"Y"}, "!X -> Y", "Negated-antecedent bridge lemma from local consequent assumption"),
		bridgeOnlyRow("BOA04", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "medium", "layered_negated_bridge", "neg1", "BOA-NEGATED", "negated", "local_assumption_lift", "negated_antecedent", []string{"Z"}, "Y -> Z", "Right bridge lemma for negated-antecedent family"),
	}
}

func generateBridgeOnlyCanonicalRows() []CaseRow {
	return []CaseRow{
		bridgeOnlyRow("BOAC01", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "easy", "layered_linear_canonical", "canon0", "BOA-CANON-LAYERED", "canonical", "local_assumption_lift", "canonical_vars", []string{"B"}, "A -> B", "Canonical-variable left bridge lemma"),
		bridgeOnlyRow("BOAC01", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "easy", "layered_linear_canonical", "canon0", "BOA-CANON-LAYERED", "canonical", "local_assumption_lift", "canonical_vars", []string{"C"}, "B -> C", "Canonical-variable right bridge lemma"),
		bridgeOnlyRow("BOAC02", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "easy", "layered_linear_renamed_canonical", "canon1", "BOA-CANON-RENAMED", "canonical", "local_assumption_lift", "canonical_vars", []string{"B"}, "A -> B", "Normalized renamed-family left bridge lemma"),
		bridgeOnlyRow("BOAC02", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "easy", "layered_linear_renamed_canonical", "canon1", "BOA-CANON-RENAMED", "canonical", "local_assumption_lift", "canonical_vars", []string{"C"}, "B -> C", "Normalized renamed-family right bridge lemma"),
		bridgeOnlyRow("BOAC03", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "medium", "layered_distractor_canonical", "canon2", "BOA-CANON-DISTRACTOR", "canonical_distractor", "local_assumption_lift", "canonical_vars", []string{"B", "D -> E"}, "A -> B", "Canonical-variable bridge lemma with standardized distractor assumption"),
		bridgeOnlyRow("BOAC03", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "medium", "layered_distractor_canonical", "canon2", "BOA-CANON-DISTRACTOR", "canonical_distractor", "local_assumption_lift", "canonical_vars", []string{"C", "D -> E"}, "B -> C", "Canonical-variable right bridge lemma with standardized distractor assumption"),
		bridgeOnlyRow("BOAC04", "bridge-only-authoring-v1", "b1", 1, "bridge_left", "medium", "layered_negated_bridge_canonical", "canon3", "BOA-CANON-NEGATED", "canonical_negated", "local_assumption_lift", "canonical_vars", []string{"B"}, "!A -> B", "Canonical-variable negated-antecedent bridge lemma"),
		bridgeOnlyRow("BOAC04", "bridge-only-authoring-v1", "b2", 2, "bridge_right", "medium", "layered_negated_bridge_canonical", "canon3", "BOA-CANON-NEGATED", "canonical_negated", "local_assumption_lift", "canonical_vars", []string{"C"}, "B -> C", "Canonical-variable right bridge lemma for negated family"),
	}
}

func bridgeOnlyRow(chainID, protocol, stageID string, stageOrder int, stageRole, difficulty, caseFamily, atomRenamingID, transitionGroup, symbolOverlap, lemmaSurfaceStyle, notes string, assumptions []string, goal, comment string) CaseRow {
	return CaseRow{
		CaseID:            chainID + "-" + strings.ToUpper(stageID),
		ChainID:           chainID,
		ChainProtocol:     protocol,
		StageID:           stageID,
		StageOrder:        stageOrder,
		StageRole:         stageRole,
		Category:          "bridge_only_authoring",
		Label:             "entailed",
		Difficulty:        difficulty,
		CaseFamily:        caseFamily,
		ChainDepth:        1,
		ProvenanceMode:    "none",
		AtomRenamingID:    atomRenamingID,
		TrustedReuse:      boolPtr(true),
		TransitionGroup:   transitionGroup,
		ImportArity:       "0",
		ReuseShape:        "bridge_only",
		BridgeDepth:       "1",
		SymbolOverlap:     symbolOverlap,
		LemmaSurfaceStyle: lemmaSurfaceStyle,
		Notes:             notes,
		ImportStageIDs:    []string{},
		Assumptions:       append([]string(nil), assumptions...),
		ImportedLemmas:    []string{},
		Goal:              goal,
		ExpectedBehavior:  "prove",
		Comment:           comment,
	}
}

func generateBridgeOnlyPhase2Rows() []CaseRow {
	rows := make([]CaseRow, 0, 150)
	for chain := 201; chain <= 210; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "easy", "layered_linear_exact", "r0", "BOA2-E-EXACT", "exact", "local_assumption_lift", "a1_lift_atomic", []string{"B" + id}, "A"+id+" -> B"+id, "Exact atomic left bridge lemma from local consequent assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "easy", "layered_linear_exact", "r0", "BOA2-E-EXACT", "exact", "local_assumption_lift", "a1_lift_atomic", []string{"C" + id}, "B"+id+" -> C"+id, "Exact atomic right bridge lemma from local consequent assumption"),
		)
	}
	for chain := 211; chain <= 220; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "easy", "layered_linear_renamed", "r1", "BOA2-E-RENAMED", "renamed", "local_assumption_lift", "a1_lift_atomic", []string{"Q" + id}, "P"+id+" -> Q"+id, "Renamed atomic left bridge lemma from local consequent assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "easy", "layered_linear_renamed", "r1", "BOA2-E-RENAMED", "renamed", "local_assumption_lift", "a1_lift_atomic", []string{"R" + id}, "Q"+id+" -> R"+id, "Renamed atomic right bridge lemma from local consequent assumption"),
		)
	}
	for chain := 221; chain <= 225; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "easy", "layered_negated_bridge_easy", "neg1", "BOA2-E-NEGATED", "negated", "local_assumption_lift", "negated_antecedent", []string{"B" + id}, "!A"+id+" -> B"+id, "Negated-antecedent left bridge lemma from local consequent assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "easy", "layered_negated_bridge_easy", "neg1", "BOA2-E-NEGATED", "negated", "local_assumption_lift", "negated_antecedent", []string{"C" + id}, "B"+id+" -> C"+id, "Right bridge lemma paired with a negated-antecedent left stage"),
		)
	}
	for chain := 226; chain <= 230; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "medium", "layered_distractor_exact", "r2", "BOA2-M-DISTRACTOR-EXACT", "distractor", "local_assumption_lift", "single_irrelevant_assumption", []string{"B" + id, "X" + id + " -> Y" + id}, "A"+id+" -> B"+id, "Exact bridge lemma with one irrelevant extra assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "medium", "layered_distractor_exact", "r2", "BOA2-M-DISTRACTOR-EXACT", "distractor", "local_assumption_lift", "single_irrelevant_assumption", []string{"C" + id, "X" + id + " -> Y" + id}, "B"+id+" -> C"+id, "Right bridge lemma with one irrelevant extra assumption"),
		)
	}
	for chain := 231; chain <= 235; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "medium", "layered_distractor_renamed", "r3", "BOA2-M-DISTRACTOR-RENAMED", "distractor_renamed", "local_assumption_lift", "single_irrelevant_assumption", []string{"Q" + id, "U" + id + " -> V" + id}, "P"+id+" -> Q"+id, "Renamed bridge lemma with one irrelevant extra assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "medium", "layered_distractor_renamed", "r3", "BOA2-M-DISTRACTOR-RENAMED", "distractor_renamed", "local_assumption_lift", "single_irrelevant_assumption", []string{"R" + id, "U" + id + " -> V" + id}, "Q"+id+" -> R"+id, "Renamed right bridge lemma with one irrelevant extra assumption"),
		)
	}
	for chain := 236; chain <= 240; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "medium", "compound_left_antecedent", "c1", "BOA2-M-COMPOUND-LEFT", "compound", "compound_assumption_lift", "compound_left_antecedent", []string{"C" + id}, "(A"+id+" -> B"+id+") -> C"+id, "Compound-antecedent left bridge lemma from local consequent assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "medium", "compound_left_antecedent", "c1", "BOA2-M-COMPOUND-LEFT", "compound", "compound_assumption_lift", "compound_left_antecedent", []string{"D" + id}, "C"+id+" -> D"+id, "Atomic right bridge lemma paired with a compound left stage"),
		)
	}
	for chain := 241; chain <= 245; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "medium", "compound_final_consequent", "c2", "BOA2-M-COMPOUND-FINAL", "compound", "compound_assumption_lift", "compound_consequent", []string{"C" + id}, "A"+id+" -> C"+id, "Atomic left bridge lemma paired with a compound right stage"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "medium", "compound_final_consequent", "c2", "BOA2-M-COMPOUND-FINAL", "compound", "compound_assumption_lift", "compound_consequent", []string{"D" + id + " -> E" + id}, "C"+id+" -> (D"+id+" -> E"+id+")", "Compound-consequent right bridge lemma from local assumption"),
		)
	}
	for chain := 246; chain <= 250; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "medium", "negated_final_consequent", "neg2", "BOA2-M-NEGATED-FINAL", "negated", "local_assumption_lift", "negated_final_consequent", []string{"B" + id}, "A"+id+" -> B"+id, "Atomic left bridge lemma paired with a negated final consequent"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "medium", "negated_final_consequent", "neg2", "BOA2-M-NEGATED-FINAL", "negated", "local_assumption_lift", "negated_final_consequent", []string{"!C" + id}, "B"+id+" -> !C"+id, "Negated-consequent right bridge lemma from local assumption"),
		)
	}
	for chain := 251; chain <= 255; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "hard", "nested_middle_surface", "h1", "BOA2-H-NESTED-MIDDLE", "nested", "nested_assumption_lift", "nested_middle_surface", []string{"B" + id + " -> C" + id}, "A"+id+" -> (B"+id+" -> C"+id+")", "Nested-middle left bridge lemma from local higher-order assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "hard", "nested_middle_surface", "h1", "BOA2-H-NESTED-MIDDLE", "nested", "nested_assumption_lift", "nested_middle_surface", []string{"D" + id}, "(B"+id+" -> C"+id+") -> D"+id, "Bridge lemma that targets the nested middle formula as antecedent"),
		)
	}
	for chain := 256; chain <= 260; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "hard", "compound_left_and_middle", "h2", "BOA2-H-COMPOUND-CHAIN", "compound", "nested_assumption_lift", "compound_left_and_middle", []string{"C" + id + " -> D" + id}, "(A"+id+" -> B"+id+") -> (C"+id+" -> D"+id+")", "Compound left bridge lemma from a higher-order consequent assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "hard", "compound_left_and_middle", "h2", "BOA2-H-COMPOUND-CHAIN", "compound", "nested_assumption_lift", "compound_left_and_middle", []string{"E" + id}, "(C"+id+" -> D"+id+") -> E"+id, "Right bridge lemma whose antecedent matches the compound middle formula"),
		)
	}
	for chain := 261; chain <= 265; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "hard", "compound_middle_and_final", "h3", "BOA2-H-COMPOUND-FINAL", "compound", "nested_assumption_lift", "compound_middle_and_final", []string{"B" + id + " -> C" + id}, "A"+id+" -> (B"+id+" -> C"+id+")", "Left bridge lemma whose consequent is already a compound imported shape"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "hard", "compound_middle_and_final", "h3", "BOA2-H-COMPOUND-FINAL", "compound", "nested_assumption_lift", "compound_middle_and_final", []string{"D" + id + " -> E" + id}, "(B"+id+" -> C"+id+") -> (D"+id+" -> E"+id+")", "Right bridge lemma from a compound consequent assumption"),
		)
	}
	for chain := 266; chain <= 270; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "hard", "compound_with_two_distractors", "h4", "BOA2-H-DISTRACTOR-COMPOUND", "distractor", "compound_assumption_lift", "two_irrelevant_assumptions", []string{"B" + id, "X" + id + " -> Y" + id, "U" + id + " -> V" + id}, "A"+id+" -> B"+id, "Atomic bridge lemma with two irrelevant extra assumptions"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "hard", "compound_with_two_distractors", "h4", "BOA2-H-DISTRACTOR-COMPOUND", "distractor", "compound_assumption_lift", "two_irrelevant_assumptions", []string{"C" + id + " -> D" + id, "X" + id + " -> Y" + id, "U" + id + " -> V" + id}, "B"+id+" -> (C"+id+" -> D"+id+")", "Compound right bridge lemma with two irrelevant extra assumptions"),
		)
	}
	for chain := 271; chain <= 275; chain++ {
		id := phase2AtomID(chain)
		rows = append(rows,
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b1", 1, "bridge_left", "hard", "negated_nested_bridge", "neg3", "BOA2-H-NEGATED-NESTED", "negated_nested", "nested_assumption_lift", "negated_nested_surface", []string{"B" + id + " -> C" + id}, "!A"+id+" -> (B"+id+" -> C"+id+")", "Negated nested left bridge lemma from local higher-order assumption"),
			bridgeOnlyRow(phase2ChainID("BOA", chain), "bridge-only-authoring-v2", "b2", 2, "bridge_right", "hard", "negated_nested_bridge", "neg3", "BOA2-H-NEGATED-NESTED", "negated_nested", "nested_assumption_lift", "negated_nested_surface", []string{"!D" + id}, "(B"+id+" -> C"+id+") -> !D"+id, "Right bridge lemma that closes into a negated consequent"),
		)
	}
	return rows
}
