package benchmarkpack

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type benchmarkTemplate struct {
	ID          string
	Category    string
	Label       string
	Difficulty  string
	Assumptions []string
	Goal        string
	Comment     string
}

var identifierPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

var directBaselineTemplates = []benchmarkTemplate{
	{ID: "V2E01", Category: "assumption_import", Label: "entailed", Difficulty: "easy", Assumptions: []string{"R -> S"}, Goal: "R -> S", Comment: "Import a single assumption directly"},
	{ID: "V2E02", Category: "mixed_proof", Label: "entailed", Difficulty: "easy", Assumptions: []string{"R", "R -> S"}, Goal: "S", Comment: "Single modus ponens derivation"},
	{ID: "V2E03", Category: "mixed_proof", Label: "entailed", Difficulty: "medium", Assumptions: []string{"R", "R -> S", "S -> T"}, Goal: "T", Comment: "Two-step modus ponens chain"},
	{ID: "V2E04", Category: "mixed_proof", Label: "entailed", Difficulty: "hard", Assumptions: []string{"R", "R -> S", "S -> T", "T -> U"}, Goal: "U", Comment: "Three-step modus ponens chain"},
	{ID: "V2E05", Category: "direct_axiom_instance", Label: "entailed", Difficulty: "easy", Assumptions: []string{}, Goal: "R -> ((S -> T) -> R)", Comment: "Direct A1 instance"},
	{ID: "V2E06", Category: "direct_axiom_instance", Label: "entailed", Difficulty: "medium", Assumptions: []string{}, Goal: "(R -> (S -> T)) -> ((R -> S) -> (R -> T))", Comment: "Direct A2 instance"},
	{ID: "V2E07", Category: "direct_axiom_instance", Label: "entailed", Difficulty: "medium", Assumptions: []string{}, Goal: "(!S -> !R) -> (R -> S)", Comment: "Direct A3 instance"},
	{ID: "V2E08", Category: "mixed_proof", Label: "entailed", Difficulty: "hard", Assumptions: []string{"!S -> !R", "R"}, Goal: "S", Comment: "A3 premise plus premise import"},
	{ID: "V2E09", Category: "mixed_proof", Label: "entailed", Difficulty: "medium", Assumptions: []string{"R"}, Goal: "S -> R", Comment: "Implication introduction via A1"},
	{ID: "V2E10", Category: "mixed_proof", Label: "entailed", Difficulty: "medium", Assumptions: []string{"R -> (S -> T)", "R", "S"}, Goal: "T", Comment: "Nested implication elimination"},
	{ID: "V2E11", Category: "mixed_proof", Label: "entailed", Difficulty: "hard", Assumptions: []string{"R -> (S -> T)", "R -> S", "R"}, Goal: "T", Comment: "Derived bridge premise then final modus ponens"},
	{ID: "V2E12", Category: "theorem_synthesis", Label: "entailed", Difficulty: "hard", Assumptions: []string{}, Goal: "R -> R", Comment: "Identity theorem without assumptions"},
	{ID: "V2N01", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"R"}, Goal: "S", Comment: "Irrelevant atomic goal"},
	{ID: "V2N02", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"R -> S"}, Goal: "S", Comment: "Missing antecedent for modus ponens"},
	{ID: "V2N03", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"R -> S", "S"}, Goal: "R", Comment: "Consequent does not imply antecedent"},
	{ID: "V2N04", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"R", "S -> T"}, Goal: "T", Comment: "Disconnected implication"},
	{ID: "V2N05", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"R -> S", "T -> U", "R"}, Goal: "U", Comment: "Two chains with a missing bridge"},
	{ID: "V2N06", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"R"}, Goal: "!R", Comment: "Negation not derivable from a single atom"},
	{ID: "V2N07", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"R -> (S -> T)", "R"}, Goal: "T", Comment: "Nested implication missing inner premise"},
	{ID: "V2N08", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"R -> S", "R -> T"}, Goal: "S", Comment: "Parallel consequents without antecedent"},
	{ID: "V2N09", Category: "negative_refusal", Label: "not_entailed", Difficulty: "hard", Assumptions: []string{"R -> (S -> T)", "R -> S"}, Goal: "T", Comment: "Bridge implication lacks the triggering antecedent"},
	{ID: "V2N10", Category: "negative_refusal", Label: "not_entailed", Difficulty: "medium", Assumptions: []string{"!S -> !R"}, Goal: "S", Comment: "Converse of A3 is not available"},
	{ID: "V2N11", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{}, Goal: "R", Comment: "No assumptions for atomic goal"},
	{ID: "V2N12", Category: "negative_refusal", Label: "not_entailed", Difficulty: "easy", Assumptions: []string{"R -> S", "S -> T"}, Goal: "U", Comment: "Conclusion outside the implication chain"},
}

func generateDirectRows(caseCount int) ([]Row, error) {
	if caseCount == 0 || caseCount == DirectBaselineCaseCount {
		return directBaselineRows(), nil
	}
	if caseCount < DirectBaselineCaseCount {
		return nil, fmt.Errorf("direct benchmark --case-count must be 0 or >= %d", DirectBaselineCaseCount)
	}
	if caseCount%2 != 0 {
		return nil, fmt.Errorf("direct benchmark --case-count must be even to preserve entailed/not_entailed balance")
	}

	entailedByDifficulty := bucketTemplates(directBaselineTemplates, "entailed")
	negativeByDifficulty := bucketTemplates(directBaselineTemplates, "not_entailed")
	positiveNeeded := caseCount / 2
	negativeNeeded := caseCount / 2
	positiveRows := generateBalancedRows("V2XE", positiveNeeded, entailedByDifficulty)
	negativeRows := generateBalancedRows("V2XN", negativeNeeded, negativeByDifficulty)

	rows := make([]Row, 0, caseCount)
	// Interleave entailed and negative cases so prefixes of the generated pack do
	// not collapse into a one-sided label slice during partial inspection.
	for i := 0; i < positiveNeeded; i++ {
		rows = append(rows, positiveRows[i], negativeRows[i])
	}
	return rows, nil
}

func directBaselineRows() []Row {
	rows := make([]Row, 0, len(directBaselineTemplates))
	for _, tmpl := range directBaselineTemplates {
		rows = append(rows, Row{
			CaseID:       tmpl.ID,
			SourceCaseID: tmpl.ID,
			Category:     tmpl.Category,
			Label:        tmpl.Label,
			Difficulty:   tmpl.Difficulty,
			Assumptions:  append([]string(nil), tmpl.Assumptions...),
			Goal:         tmpl.Goal,
			Comment:      tmpl.Comment,
		})
	}
	return rows
}

func bucketTemplates(templates []benchmarkTemplate, label string) map[string][]benchmarkTemplate {
	result := map[string][]benchmarkTemplate{
		"easy":   {},
		"medium": {},
		"hard":   {},
	}
	for _, tmpl := range templates {
		if tmpl.Label != label {
			continue
		}
		result[tmpl.Difficulty] = append(result[tmpl.Difficulty], tmpl)
	}
	return result
}

func generateBalancedRows(idPrefix string, count int, buckets map[string][]benchmarkTemplate) []Row {
	rows := make([]Row, 0, count)
	difficulties := []string{"easy", "medium", "hard"}
	indices := map[string]int{
		"easy":   0,
		"medium": 0,
		"hard":   0,
	}
	for i := 0; i < count; i++ {
		difficulty := difficulties[i%len(difficulties)]
		pool := buckets[difficulty]
		if len(pool) == 0 {
			pool = fallbackTemplatePool(buckets)
		}
		templateIndex := indices[difficulty] % len(pool)
		indices[difficulty]++
		template := pool[templateIndex]
		rows = append(rows, instantiateTemplate(idPrefix, i+1, template))
	}
	return rows
}

func fallbackTemplatePool(buckets map[string][]benchmarkTemplate) []benchmarkTemplate {
	for _, difficulty := range []string{"easy", "medium", "hard"} {
		if len(buckets[difficulty]) > 0 {
			return buckets[difficulty]
		}
	}
	return nil
}

func instantiateTemplate(idPrefix string, ordinal int, tmpl benchmarkTemplate) Row {
	symbols := orderedTemplateSymbols(tmpl)
	replacements := make(map[string]string, len(symbols))
	for idx, symbol := range symbols {
		replacements[symbol] = fmt.Sprintf("%s%03d", directAtomBase(idx), ordinal)
	}
	assumptions := make([]string, 0, len(tmpl.Assumptions))
	for _, assumption := range tmpl.Assumptions {
		assumptions = append(assumptions, rewriteFormula(assumption, replacements))
	}
	return Row{
		CaseID:       fmt.Sprintf("%s%03d", idPrefix, ordinal),
		SourceCaseID: tmpl.ID,
		Category:     tmpl.Category,
		Label:        tmpl.Label,
		Difficulty:   tmpl.Difficulty,
		Assumptions:  assumptions,
		Goal:         rewriteFormula(tmpl.Goal, replacements),
		Comment:      fmt.Sprintf("Expanded from %s with canonical atom renaming #%03d. %s", tmpl.ID, ordinal, tmpl.Comment),
	}
}

func orderedTemplateSymbols(tmpl benchmarkTemplate) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, 6)
	appendSymbols := func(text string) {
		for _, token := range identifierPattern.FindAllString(text, -1) {
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			result = append(result, token)
		}
	}
	for _, assumption := range tmpl.Assumptions {
		appendSymbols(assumption)
	}
	appendSymbols(tmpl.Goal)
	return result
}

func rewriteFormula(text string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return text
	}
	return identifierPattern.ReplaceAllStringFunc(text, func(token string) string {
		if replacement, ok := replacements[token]; ok {
			return replacement
		}
		return token
	})
}

func directAtomBase(index int) string {
	bases := []string{
		"P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O",
	}
	if index < len(bases) {
		return bases[index]
	}
	return fmt.Sprintf("X%d", index)
}

func directDifficultyCounts(rows []Row) map[string]int {
	counts := map[string]int{"easy": 0, "medium": 0, "hard": 0}
	for _, row := range rows {
		counts[strings.TrimSpace(row.Difficulty)]++
	}
	return counts
}

func directLabels(rows []Row) []string {
	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, row.Label)
	}
	slices.Sort(labels)
	return labels
}
