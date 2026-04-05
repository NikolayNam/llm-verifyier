package benchmarkpack

import "fmt"

var ndPositiveTemplates = map[string][]benchmarkTemplate{
	"easy": {
		{ID: "NDI01", Category: "assumption_import", Label: "entailed", Difficulty: "easy", Assumptions: []string{"P"}, Goal: "P", Comment: "Direct premise import"},
		{ID: "NDI02", Category: "single_mp", Label: "entailed", Difficulty: "easy", Assumptions: []string{"P", "P -> Q"}, Goal: "Q", Comment: "Single implication elimination"},
	},
	"medium": {
		{ID: "NDI03", Category: "mp_chain", Label: "entailed", Difficulty: "medium", Assumptions: []string{"P", "P -> Q", "Q -> R"}, Goal: "R", Comment: "Two-step implication chain"},
		{ID: "NDI04", Category: "nested_imp_intro", Label: "entailed", Difficulty: "medium", Assumptions: []string{"P"}, Goal: "Q -> P", Comment: "Introduce implication under nested assumption"},
	},
	"hard": {
		{ID: "NDI05", Category: "theorem_synthesis", Label: "entailed", Difficulty: "hard", Assumptions: []string{}, Goal: "P -> P", Comment: "Identity theorem without premises"},
		{ID: "NDI06", Category: "theorem_synthesis", Label: "entailed", Difficulty: "hard", Assumptions: []string{}, Goal: "(P -> (Q -> R)) -> ((P -> Q) -> (P -> R))", Comment: "Composition theorem in implicational fragment"},
	},
}

var ndNegativeTemplates = map[string][]benchmarkTemplate{
	"easy": {
		{ID: "NDI07", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"P -> Q"}, Goal: "P", Comment: "Consequent does not imply antecedent"},
		{ID: "NDI08", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"P"}, Goal: "Q", Comment: "Irrelevant atomic goal should be refused"},
	},
	"medium": {
		{ID: "NDN09", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"P -> Q", "Q"}, Goal: "P", Comment: "Reverse modus ponens is not available"},
		{ID: "NDN10", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"P -> Q", "P -> R"}, Goal: "Q", Comment: "Shared antecedent without premise import"},
	},
	"hard": {
		{ID: "NDN11", Category: "negative_refusal", Label: "not_entailed", Difficulty: "hard", Assumptions: []string{"P -> (Q -> R)", "P -> Q"}, Goal: "R", Comment: "Bridge implication lacks the triggering premise"},
		{ID: "NDN12", Category: "negative_refusal", Label: "not_entailed", Difficulty: "hard", Assumptions: []string{}, Goal: "P", Comment: "Atomic theorem is not derivable from the empty context"},
	},
}

func generateNDRows(caseCount int) ([]Row, error) {
	if caseCount == 0 || caseCount == NDBaselineCaseCount {
		return ndBaselineRows(), nil
	}
	if caseCount < NDBaselineCaseCount {
		return nil, fmt.Errorf("nd benchmark --case-count must be 0 or >= %d", NDBaselineCaseCount)
	}
	if caseCount%2 != 0 {
		return nil, fmt.Errorf("nd benchmark --case-count must be even to preserve entailed/not_entailed balance")
	}

	positiveNeeded := caseCount / 2
	negativeNeeded := caseCount / 2
	positiveRows := generateNDBalancedRows("NDE", positiveNeeded, ndPositiveTemplates)
	negativeRows := generateNDBalancedRows("NDN", negativeNeeded, ndNegativeTemplates)

	rows := make([]Row, 0, caseCount)
	// Keep the exported theorem pack explicitly label-balanced rather than
	// inheriting the skewed 6/2 ratio of the tiny pilot baseline.
	for i := 0; i < positiveNeeded; i++ {
		rows = append(rows, positiveRows[i], negativeRows[i])
	}
	return rows, nil
}

func ndBaselineRows() []Row {
	rows := make([]Row, 0, NDBaselineCaseCount)
	for _, tmpl := range []benchmarkTemplate{
		ndPositiveTemplates["easy"][0],
		ndPositiveTemplates["easy"][1],
		ndPositiveTemplates["medium"][0],
		ndPositiveTemplates["medium"][1],
		ndPositiveTemplates["hard"][0],
		ndPositiveTemplates["hard"][1],
	} {
		rows = append(rows, ndRowFromTemplate(tmpl, tmpl.ID, "pilot_shared_20260327", 0))
	}
	for _, tmpl := range []benchmarkTemplate{
		ndNegativeTemplates["easy"][0],
		ndNegativeTemplates["easy"][1],
	} {
		rows = append(rows, ndRowFromTemplate(tmpl, tmpl.ID, "pilot_shared_20260327", 0))
	}
	return rows
}

func generateNDBalancedRows(idPrefix string, count int, buckets map[string][]benchmarkTemplate) []Row {
	rows := make([]Row, 0, count)
	difficulties := []string{"easy", "medium", "hard"}
	indices := map[string]int{"easy": 0, "medium": 0, "hard": 0}
	for i := 0; i < count; i++ {
		difficulty := difficulties[i%len(difficulties)]
		pool := buckets[difficulty]
		template := pool[indices[difficulty]%len(pool)]
		indices[difficulty]++
		caseOrdinal := i + 1
		caseID := fmt.Sprintf("%s%03d", idPrefix, caseOrdinal)
		rows = append(rows, ndRowFromTemplate(template, caseID, fmt.Sprintf("pilot_shared_20260327_expanded_%d", count*2), caseOrdinal))
	}
	return rows
}

func ndRowFromTemplate(tmpl benchmarkTemplate, caseID, packID string, ordinal int) Row {
	replacements := map[string]string{}
	if ordinal > 0 {
		for idx, symbol := range orderedTemplateSymbols(tmpl) {
			replacements[symbol] = fmt.Sprintf("%s%03d", directAtomBase(idx), ordinal)
		}
	}
	assumptions := make([]string, 0, len(tmpl.Assumptions))
	for _, assumption := range tmpl.Assumptions {
		assumptions = append(assumptions, rewriteFormula(assumption, replacements))
	}
	comment := tmpl.Comment
	if ordinal > 0 {
		comment = fmt.Sprintf("Expanded from %s with canonical atom renaming #%03d. %s", tmpl.ID, ordinal, tmpl.Comment)
	}
	return Row{
		TheoremPackID: packID,
		TheoremID:     caseID,
		CaseID:        caseID,
		Category:      tmpl.Category,
		Label:         tmpl.Label,
		Difficulty:    tmpl.Difficulty,
		Assumptions:   assumptions,
		Goal:          rewriteFormula(tmpl.Goal, replacements),
		LogicFragment: "implicational-prop-v1",
		Comment:       comment,
	}
}
