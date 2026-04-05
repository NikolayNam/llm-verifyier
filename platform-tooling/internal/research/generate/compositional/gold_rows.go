package compositional

import "strings"

func generateGoldFinalPhase1Rows() []CaseRow {
	return []CaseRow{
		goldFinalRow("GFC01", "gold-final-composition-only-v1", "fg", 1, "final_gold", "entailed", "easy", "layered_linear", "r0", "GFO-LAYERED", "2", "1", "exact", "paired_negative", "static_gold_resources", nil, nil, []string{"A -> B", "B -> C"}, "A -> C", "prove", "Final closure over fixed gold-backed bridge lemmas"),
		goldFinalRow("GFC01", "gold-final-composition-only-v1", "neg", 2, "negative_control", "not_entailed", "easy", "layered_linear_negative", "r0", "GFO-LAYERED", "2", "1", "exact", "paired_negative", "static_gold_resources", nil, nil, []string{"A -> B", "B -> C"}, "A -> D", "not_derivable", "Negative control over the same gold-backed resources"),
		goldFinalRow("GFC02", "gold-final-composition-only-v1", "fg", 1, "final_gold", "entailed", "easy", "layered_linear_renamed", "r1", "GFO-RENAMED", "2", "1", "renamed", "paired_negative", "static_gold_resources", nil, nil, []string{"P -> Q", "Q -> R"}, "P -> R", "prove", "Final closure under atom renaming with static gold imports"),
		goldFinalRow("GFC02", "gold-final-composition-only-v1", "neg", 2, "negative_control", "not_entailed", "easy", "layered_linear_negative", "r1", "GFO-RENAMED", "2", "1", "renamed", "paired_negative", "static_gold_resources", nil, nil, []string{"P -> Q", "Q -> R"}, "P -> S", "not_derivable", "Renamed negative control for gold-final-only lane"),
		goldFinalRow("GFC03", "gold-final-composition-only-v1", "fg", 1, "final_gold", "entailed", "medium", "layered_distractor", "r2", "GFO-DISTRACTOR", "2", "1", "distractor", "paired_negative", "static_gold_resources", nil, []string{"P -> Q"}, []string{"M -> N", "N -> O"}, "M -> O", "prove", "Final closure with irrelevant direct assumption and static gold imports"),
		goldFinalRow("GFC03", "gold-final-composition-only-v1", "neg", 2, "negative_control", "not_entailed", "medium", "layered_distractor_negative", "r2", "GFO-DISTRACTOR", "2", "1", "distractor", "paired_negative", "static_gold_resources", nil, []string{"P -> Q"}, []string{"M -> N", "Q -> R"}, "M -> R", "not_derivable", "Distractor negative control with mismatched imported resources"),
		goldFinalRow("GFC04", "gold-final-composition-only-v1", "fg", 1, "final_gold", "entailed", "medium", "layered_negated_bridge", "neg1", "GFO-NEGATED", "2", "1", "negated", "paired_negative", "static_gold_resources", nil, nil, []string{"!X -> Y", "Y -> Z"}, "!X -> Z", "prove", "Final closure with negated antecedent over static gold imports"),
		goldFinalRow("GFC04", "gold-final-composition-only-v1", "neg", 2, "negative_control", "not_entailed", "medium", "layered_negated_bridge_negative", "neg1", "GFO-NEGATED", "2", "1", "negated", "paired_negative", "static_gold_resources", nil, nil, []string{"!X -> Y", "Y -> Z"}, "!X -> W", "not_derivable", "Negated-antecedent negative control for gold-final-only lane"),
	}
}

func goldFinalRow(chainID, protocol, stageID string, stageOrder int, stageRole, label, difficulty, caseFamily, atomRenamingID, transitionGroup, importArity, bridgeDepth, symbolOverlap, negativeTwinHardness, notes string, importStageIDs, assumptions, importedLemmas []string, goal, expectedBehavior, comment string) CaseRow {
	return CaseRow{
		CaseID:               chainID + "-" + strings.ToUpper(stageID),
		ChainID:              chainID,
		ChainProtocol:        protocol,
		StageID:              stageID,
		StageOrder:           stageOrder,
		StageRole:            stageRole,
		Category:             "gold_final_composition_only",
		Label:                label,
		Difficulty:           difficulty,
		CaseFamily:           caseFamily,
		ChainDepth:           1,
		ProvenanceMode:       "gold",
		AtomRenamingID:       atomRenamingID,
		TrustedReuse:         boolPtr(false),
		TransitionGroup:      transitionGroup,
		ImportArity:          importArity,
		ReuseShape:           "gold_final_only",
		BridgeDepth:          bridgeDepth,
		SymbolOverlap:        symbolOverlap,
		LemmaSurfaceStyle:    "static_gold_imports",
		NegativeTwinHardness: negativeTwinHardness,
		Notes:                notes,
		ImportStageIDs:       append([]string(nil), importStageIDs...),
		Assumptions:          append([]string(nil), assumptions...),
		ImportedLemmas:       append([]string(nil), importedLemmas...),
		Goal:                 goal,
		ExpectedBehavior:     expectedBehavior,
		Comment:              comment,
	}
}

func generateGoldFinalPhase2Rows() []CaseRow {
	rows := make([]CaseRow, 0, 150)
	for chain := 201; chain <= 210; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> B" + id, "B" + id + " -> C" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "easy", "layered_linear_exact", "r0", "GFC2-E-ARITY2-EXACT", "2", "1", "exact", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> C"+id, "prove", "Exact final closure over two static gold-backed imports"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "easy", "layered_linear_exact", "r0", "GFC2-E-ARITY2-EXACT", "2", "1", "exact", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> D"+id, "not_derivable", "Negative control over the same exact gold-backed imports"),
		)
	}
	for chain := 211; chain <= 220; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"P" + id + " -> Q" + id, "Q" + id + " -> R" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "easy", "layered_linear_renamed", "r1", "GFC2-E-ARITY2-RENAMED", "2", "1", "renamed", "paired_negative", "static_gold_resources", nil, nil, imports, "P"+id+" -> R"+id, "prove", "Renamed final closure over two static gold-backed imports"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "easy", "layered_linear_renamed", "r1", "GFC2-E-ARITY2-RENAMED", "2", "1", "renamed", "paired_negative", "static_gold_resources", nil, nil, imports, "P"+id+" -> S"+id, "not_derivable", "Renamed negative control over the same static imports"),
		)
	}
	for chain := 221; chain <= 225; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"!A" + id + " -> B" + id, "B" + id + " -> C" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "easy", "layered_negated_bridge", "neg1", "GFC2-E-ARITY2-NEGATED", "2", "1", "negated", "paired_negative", "static_gold_resources", nil, nil, imports, "!A"+id+" -> C"+id, "prove", "Negated final closure over two static gold-backed imports"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "easy", "layered_negated_bridge", "neg1", "GFC2-E-ARITY2-NEGATED", "2", "1", "negated", "paired_negative", "static_gold_resources", nil, nil, imports, "!A"+id+" -> D"+id, "not_derivable", "Negated negative control over the same static imports"),
		)
	}
	for chain := 226; chain <= 235; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> B" + id, "B" + id + " -> C" + id, "C" + id + " -> D" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "medium", "layered_linear_exact_arity3", "r2", "GFC2-M-ARITY3-EXACT", "3", "2", "exact", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> D"+id, "prove", "Three-import final closure over exact gold-backed resources"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "medium", "layered_linear_exact_arity3", "r2", "GFC2-M-ARITY3-EXACT", "3", "2", "exact", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> E"+id, "not_derivable", "Negative control over three exact gold-backed resources"),
		)
	}
	for chain := 236; chain <= 240; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"M" + id + " -> N" + id, "N" + id + " -> O" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "medium", "layered_distractor_arity2", "r3", "GFC2-M-DISTRACTOR-ARITY2", "2", "1", "distractor", "paired_negative", "static_gold_resources_with_distractor_assumption", nil, []string{"P" + id + " -> Q" + id}, imports, "M"+id+" -> O"+id, "prove", "Final closure with one irrelevant direct assumption and two gold imports"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "medium", "layered_distractor_arity2", "r3", "GFC2-M-DISTRACTOR-ARITY2", "2", "1", "distractor", "paired_negative", "static_gold_resources_with_distractor_assumption", nil, []string{"P" + id + " -> Q" + id}, imports, "M"+id+" -> R"+id, "not_derivable", "Negative control with one irrelevant direct assumption and two gold imports"),
		)
	}
	for chain := 241; chain <= 245; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> (B" + id + " -> C" + id + ")", "(B" + id + " -> C" + id + ") -> D" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "medium", "compound_middle_arity2", "c1", "GFC2-M-COMPOUND-MIDDLE", "2", "1", "compound", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> D"+id, "prove", "Final closure through a compound middle imported formula"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "medium", "compound_middle_arity2", "c1", "GFC2-M-COMPOUND-MIDDLE", "2", "1", "compound", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> E"+id, "not_derivable", "Negative control paired with a compound middle imported formula"),
		)
	}
	for chain := 246; chain <= 250; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> B" + id, "B" + id + " -> (C" + id + " -> D" + id + ")"}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "medium", "compound_final_arity2", "c2", "GFC2-M-COMPOUND-FINAL", "2", "1", "compound", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> (C"+id+" -> D"+id+")", "prove", "Final closure where the target consequent is itself compound"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "medium", "compound_final_arity2", "c2", "GFC2-M-COMPOUND-FINAL", "2", "1", "compound", "paired_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> (E"+id+" -> F"+id+")", "not_derivable", "Negative control against a different compound consequent"),
		)
	}
	for chain := 251; chain <= 260; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> B" + id, "B" + id + " -> C" + id, "C" + id + " -> D" + id, "D" + id + " -> E" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "hard", "layered_linear_exact_arity4", "h1", "GFC2-H-ARITY4-EXACT", "4", "3", "exact", "hard_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> E"+id, "prove", "Four-import final closure over exact gold-backed resources"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "hard", "layered_linear_exact_arity4", "h1", "GFC2-H-ARITY4-EXACT", "4", "3", "exact", "hard_negative", "static_gold_resources", nil, nil, imports, "A"+id+" -> F"+id, "not_derivable", "Hard negative control over four exact gold-backed resources"),
		)
	}
	for chain := 261; chain <= 265; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"!A" + id + " -> B" + id, "B" + id + " -> (C" + id + " -> D" + id + ")", "(C" + id + " -> D" + id + ") -> E" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "hard", "negated_compound_arity3", "neg2", "GFC2-H-NEGATED-COMPOUND", "3", "2", "negated_compound", "hard_negative", "static_gold_resources", nil, nil, imports, "!A"+id+" -> E"+id, "prove", "Negated hard final closure over three gold-backed imports"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "hard", "negated_compound_arity3", "neg2", "GFC2-H-NEGATED-COMPOUND", "3", "2", "negated_compound", "hard_negative", "static_gold_resources", nil, nil, imports, "!A"+id+" -> F"+id, "not_derivable", "Hard negative control paired with a negated compound import chain"),
		)
	}
	for chain := 266; chain <= 270; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"A" + id + " -> B" + id, "B" + id + " -> C" + id, "C" + id + " -> D" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "hard", "layered_distractor_arity3", "h2", "GFC2-H-DISTRACTOR-ARITY3", "3", "2", "distractor", "hard_negative", "static_gold_resources_with_two_distractor_assumptions", nil, []string{"X" + id + " -> Y" + id, "U" + id + " -> V" + id}, imports, "A"+id+" -> D"+id, "prove", "Three-import final closure with two irrelevant direct assumptions"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "hard", "layered_distractor_arity3", "h2", "GFC2-H-DISTRACTOR-ARITY3", "3", "2", "distractor", "hard_negative", "static_gold_resources_with_two_distractor_assumptions", nil, []string{"X" + id + " -> Y" + id, "U" + id + " -> V" + id}, imports, "A"+id+" -> E"+id, "not_derivable", "Hard negative control with two irrelevant direct assumptions"),
		)
	}
	for chain := 271; chain <= 275; chain++ {
		id := phase2AtomID(chain)
		imports := []string{"(A" + id + " -> B" + id + ") -> C" + id, "C" + id + " -> D" + id}
		rows = append(rows,
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "fg", 1, "final_gold", "entailed", "hard", "nested_antecedent_arity2", "h3", "GFC2-H-NESTED-ANTECEDENT", "2", "1", "nested", "hard_negative", "static_gold_resources", nil, nil, imports, "(A"+id+" -> B"+id+") -> D"+id, "prove", "Final closure where the antecedent of the target is itself nested"),
			goldFinalRow(phase2ChainID("GFC", chain), "gold-final-composition-only-v2", "neg", 2, "negative_control", "not_entailed", "hard", "nested_antecedent_arity2", "h3", "GFC2-H-NESTED-ANTECEDENT", "2", "1", "nested", "hard_negative", "static_gold_resources", nil, nil, imports, "(A"+id+" -> B"+id+") -> E"+id, "not_derivable", "Hard negative control against a different nested-target consequent"),
		)
	}
	return rows
}
