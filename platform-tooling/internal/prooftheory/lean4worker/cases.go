package lean4worker

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

const (
	enumeratorLogicFragment    = "implicational-prop-v1"
	generationConfigFileName   = "generation-config.json"
	curatedSeedFileName        = "curated-default.csv"
	proposedSeedFileName       = "model-proposed-seed.csv"
	canonicalTheoremsFileName  = "theorems.csv"
	legacyHypothesesFileName   = "hypotheses.csv"
	defaultPayloadTemplateName = "implicational-default.json"
)

type formulaInfo struct {
	Text      string
	Formula   hilbert.Formula
	Depth     int
	TruthBits string
}

type hypothesisSeed struct {
	CaseID      string
	Category    string
	Difficulty  string
	Assumptions []string
	Goal        string
	Comment     string
}

type valuation map[string]bool

func DefaultHypothesisGenerationConfig() HypothesisGenerationConfig {
	return HypothesisGenerationConfig{
		Mode:            HypothesisGenerationModeEnumerator,
		Atoms:           []string{"P", "Q", "R"},
		MaxFormulaDepth: 1,
		MaxAssumptions:  2,
		Limit:           32,
		Filters:         []string{"non_duplicate", "syntactic_minimal", "interesting"},
	}
}

func DefaultCuratedHypothesisSeeds() []hypothesisSeed {
	return []hypothesisSeed{
		{
			CaseID:      "LEANC01",
			Category:    "assumption_import",
			Difficulty:  "easy",
			Assumptions: []string{"P"},
			Goal:        "P",
			Comment:     "Curated direct premise import",
		},
		{
			CaseID:      "LEANC02",
			Category:    "single_mp",
			Difficulty:  "easy",
			Assumptions: []string{"P", "P -> Q"},
			Goal:        "Q",
			Comment:     "Curated single implication elimination",
		},
		{
			CaseID:      "LEANC03",
			Category:    "mp_chain",
			Difficulty:  "medium",
			Assumptions: []string{"P", "P -> Q", "Q -> R"},
			Goal:        "R",
			Comment:     "Curated two-step implication chain",
		},
		{
			CaseID:      "LEANC04",
			Category:    "nested_imp_intro",
			Difficulty:  "medium",
			Assumptions: []string{"P"},
			Goal:        "Q -> P",
			Comment:     "Curated nested implication introduction target",
		},
		{
			CaseID:      "LEANC05",
			Category:    "theorem_synthesis",
			Difficulty:  "medium",
			Assumptions: []string{},
			Goal:        "P -> P",
			Comment:     "Curated identity theorem candidate",
		},
		{
			CaseID:      "LEANC06",
			Category:    "theorem_synthesis",
			Difficulty:  "hard",
			Assumptions: []string{},
			Goal:        "(P -> (Q -> R)) -> ((P -> Q) -> (P -> R))",
			Comment:     "Curated composition theorem candidate",
		},
		{
			CaseID:      "LEANC07",
			Category:    "negative_refusal",
			Difficulty:  "easy",
			Assumptions: []string{"P -> Q"},
			Goal:        "P",
			Comment:     "Curated refutation target: consequent should not imply antecedent",
		},
		{
			CaseID:      "LEANC08",
			Category:    "negative_refusal",
			Difficulty:  "easy",
			Assumptions: []string{"P"},
			Goal:        "Q",
			Comment:     "Curated refutation target: irrelevant atomic goal",
		},
	}
}

func DefaultModelProposedSeeds() []hypothesisSeed {
	return []hypothesisSeed{
		{
			CaseID:      "LEANP01",
			Assumptions: []string{"P", "Q -> R"},
			Goal:        "P -> R",
			Comment:     "Seed placeholder for future model-proposed normalization; currently not derivable",
		},
		{
			CaseID:      "LEANP02",
			Assumptions: []string{"P -> Q", "Q -> R"},
			Goal:        "P -> R",
			Comment:     "Seed placeholder for future model-proposed theorem candidate",
		},
		{
			CaseID:      "LEANP03",
			Assumptions: []string{},
			Goal:        "P -> (Q -> P)",
			Comment:     "Seed placeholder for future model-proposed tautology candidate",
		},
		{
			CaseID:      "LEANP04",
			Assumptions: []string{"P -> Q"},
			Goal:        "Q -> P",
			Comment:     "Seed placeholder for future model-proposed non-derivable candidate",
		},
	}
}

func GenerateHypothesisCases(config HypothesisGenerationConfig) ([]HypothesisCase, HypothesisGenerationConfig, error) {
	normalized, err := normalizeGenerationConfig(config)
	if err != nil {
		return nil, HypothesisGenerationConfig{}, err
	}

	var cases []HypothesisCase
	switch normalized.Mode {
	case HypothesisGenerationModeEnumerator:
		cases, err = generateEnumeratedHypothesisCases(normalized)
	case HypothesisGenerationModeCurated:
		cases, err = generateSeedBackedHypothesisCases(normalized, DefaultCuratedHypothesisSeeds())
	case HypothesisGenerationModeProposed:
		cases, err = generateSeedBackedHypothesisCases(normalized, DefaultModelProposedSeeds())
	default:
		return nil, HypothesisGenerationConfig{}, fmt.Errorf("unsupported lean4 hypothesis generation mode %q", normalized.Mode)
	}
	if err != nil {
		return nil, HypothesisGenerationConfig{}, err
	}
	return filterAndLimitCases(cases, normalized), normalized, nil
}

func GenerateHypothesisArtifacts(root string, config HypothesisGenerationConfig) (string, string, string, error) {
	if root == "" {
		return "", "", "", fmt.Errorf("lean4 artifact root is required")
	}
	cases, normalized, err := GenerateHypothesisCases(config)
	if err != nil {
		return "", "", "", err
	}
	_, cases = FinalizeHypothesisCases(cases, normalized.Mode, time.Now().UTC())
	paths, err := WriteHypothesisArtifacts(root, normalized, cases)
	if err != nil {
		return "", "", "", err
	}
	return paths.TheoremsPath, paths.GenerationConfigPath, paths.DefaultPayloadPath, nil
}

type HypothesisArtifactPaths struct {
	TheoremsPath         string
	HistoricalPath       string
	LegacyCasesPath      string
	GenerationConfigPath string
	DefaultPayloadPath   string
}

func WriteHypothesisArtifacts(root string, config HypothesisGenerationConfig, cases []HypothesisCase) (HypothesisArtifactPaths, error) {
	if root == "" {
		return HypothesisArtifactPaths{}, fmt.Errorf("lean4 artifact root is required")
	}
	if len(cases) == 0 {
		return HypothesisArtifactPaths{}, fmt.Errorf("at least one lean4 hypothesis case is required")
	}
	packID := strings.TrimSpace(cases[0].TheoremPackID)
	if packID == "" {
		return HypothesisArtifactPaths{}, fmt.Errorf("lean4 hypothesis theorem_pack_id is required")
	}
	normalizedForArtifacts := config
	if strings.TrimSpace(normalizedForArtifacts.SourceFile) == "" {
		normalizedForArtifacts.SourceFile = DefaultHypothesisSourcePath(root, normalizedForArtifacts.Mode)
	}
	configPath, payloadPath, err := WriteHypothesisSupportArtifacts(root, normalizedForArtifacts)
	if err != nil {
		return HypothesisArtifactPaths{}, err
	}

	theoremsPath := filepath.Join(root, DefaultTheoremsDir, canonicalTheoremsFileName)
	if err := WriteHypothesisCasesCSV(theoremsPath, cases); err != nil {
		return HypothesisArtifactPaths{}, err
	}
	historicalPath := HistoricalTheoremSnapshotPath(filepath.Join(root, DefaultTheoremsDir), packID)
	if err := writeHistoricalTheoremSnapshot(filepath.Join(root, DefaultTheoremsDir), packID, theoremsPath); err != nil {
		return HypothesisArtifactPaths{}, err
	}
	legacyPath := filepath.Join(root, LegacyCasesDir, legacyHypothesesFileName)
	if err := WriteHypothesisCasesCSV(legacyPath, cases); err != nil {
		return HypothesisArtifactPaths{}, err
	}
	return HypothesisArtifactPaths{
		TheoremsPath:         theoremsPath,
		HistoricalPath:       historicalPath,
		LegacyCasesPath:      legacyPath,
		GenerationConfigPath: configPath,
		DefaultPayloadPath:   payloadPath,
	}, nil
}

func WriteHypothesisCasesCSV(path string, cases []HypothesisCase) error {
	if path == "" {
		return fmt.Errorf("lean4 hypotheses csv path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create lean4 hypotheses dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create lean4 hypotheses csv: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"theorem_pack_id",
		"theorem_id",
		"case_id",
		"generation_mode",
		"category",
		"derivability_status",
		"interesting",
		"minimal",
		"difficulty",
		"assumptions_json",
		"goal",
		"logic_fragment",
		"lean_statement",
		"comment",
	}); err != nil {
		return fmt.Errorf("write lean4 hypotheses header: %w", err)
	}
	for _, hc := range cases {
		assumptionsJSON, err := marshalJSONNoEscape(hc.Assumptions)
		if err != nil {
			return fmt.Errorf("marshal lean4 case assumptions for %s: %w", hc.CaseID, err)
		}
		if err := writer.Write([]string{
			hc.TheoremPackID,
			firstNonEmpty(hc.TheoremID, hc.CaseID),
			hc.CaseID,
			hc.GenerationMode,
			hc.Category,
			hc.DerivabilityStatus,
			strconv.FormatBool(hc.Interesting),
			strconv.FormatBool(hc.Minimal),
			hc.Difficulty,
			string(assumptionsJSON),
			hc.Goal,
			hc.LogicFragment,
			hc.LeanStatement,
			hc.Comment,
		}); err != nil {
			return fmt.Errorf("write lean4 hypothesis row %s: %w", hc.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush lean4 hypotheses csv: %w", err)
	}
	return nil
}

func generateEnumeratedHypothesisCases(config HypothesisGenerationConfig) ([]HypothesisCase, error) {
	valuations := enumerateValuations(config.Atoms)
	formulas, err := enumerateImplicationalFormulas(config.Atoms, config.MaxFormulaDepth, valuations)
	if err != nil {
		return nil, err
	}
	assumptionSets := enumerateAssumptionSets(formulas, config.MaxAssumptions)

	cases := make([]HypothesisCase, 0, len(assumptionSets)*len(formulas))
	for _, assumptions := range assumptionSets {
		for _, goal := range formulas {
			status := classifyDerivability(assumptions, goal, valuations)
			minimal := isMinimalAssumptionSet(assumptions, valuations)
			interesting := isInterestingCase(assumptions, goal, status)
			cases = append(cases, HypothesisCase{
				GenerationMode:     config.Mode,
				Category:           classifyCategory(assumptions, goal, status),
				DerivabilityStatus: status,
				Interesting:        interesting,
				Minimal:            minimal,
				Difficulty:         classifyDifficulty(assumptions, goal),
				Assumptions:        assumptionTexts(assumptions),
				Goal:               goal.Text,
				LogicFragment:      enumeratorLogicFragment,
				LeanStatement:      curryLeanStatement(assumptions, goal),
				Comment:            buildEnumeratorComment(assumptions, goal, status, minimal, interesting),
			})
		}
	}
	return cases, nil
}

func generateSeedBackedHypothesisCases(config HypothesisGenerationConfig, fallback []hypothesisSeed) ([]HypothesisCase, error) {
	seeds := fallback
	if strings.TrimSpace(config.SourceFile) != "" {
		loadedSeeds, err := loadHypothesisSeedCSV(config.SourceFile)
		if err != nil {
			return nil, err
		}
		seeds = loadedSeeds
	}

	parsedSeeds, atoms, err := parseSeedHypotheses(seeds, config.Atoms)
	if err != nil {
		return nil, err
	}
	valuations := enumerateValuations(atoms)

	cases := make([]HypothesisCase, 0, len(parsedSeeds))
	for _, parsed := range parsedSeeds {
		assumptions := materializeFormulaInfos(parsed.Assumptions, valuations)
		goal := materializeFormulaInfo(parsed.Goal, valuations)
		status := classifyDerivability(assumptions, goal, valuations)
		minimal := isMinimalAssumptionSet(assumptions, valuations)
		interesting := isInterestingCase(assumptions, goal, status)

		category := parsed.Seed.Category
		if strings.TrimSpace(category) == "" {
			category = classifyCategory(assumptions, goal, status)
		}
		difficulty := parsed.Seed.Difficulty
		if strings.TrimSpace(difficulty) == "" {
			difficulty = classifyDifficulty(assumptions, goal)
		}
		comment := strings.TrimSpace(parsed.Seed.Comment)
		if comment == "" {
			comment = buildEnumeratorComment(assumptions, goal, status, minimal, interesting)
		}

		cases = append(cases, HypothesisCase{
			CaseID:             strings.TrimSpace(parsed.Seed.CaseID),
			GenerationMode:     config.Mode,
			Category:           category,
			DerivabilityStatus: status,
			Interesting:        interesting,
			Minimal:            minimal,
			Difficulty:         difficulty,
			Assumptions:        assumptionTexts(assumptions),
			Goal:               goal.Text,
			LogicFragment:      enumeratorLogicFragment,
			LeanStatement:      curryLeanStatement(assumptions, goal),
			Comment:            comment,
		})
	}
	return cases, nil
}

func filterAndLimitCases(cases []HypothesisCase, config HypothesisGenerationConfig) []HypothesisCase {
	seen := make(map[string]struct{})
	filtered := make([]HypothesisCase, 0, len(cases))
	for _, hypothesis := range cases {
		if hasGenerationFilter(config.Filters, "interesting") && !hypothesis.Interesting {
			continue
		}
		if hasGenerationFilter(config.Filters, "syntactic_minimal") && !hypothesis.Minimal {
			continue
		}
		if hasGenerationFilter(config.Filters, "non_duplicate") {
			key := buildSyntacticCaseKey(hypothesis.Assumptions, hypothesis.Goal)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}
		if strings.TrimSpace(hypothesis.CaseID) == "" {
			hypothesis.CaseID = generatedCaseID(config.Mode, len(filtered)+1)
		}
		filtered = append(filtered, hypothesis)
		if config.Limit > 0 && len(filtered) >= config.Limit {
			break
		}
	}
	return filtered
}

func normalizeGenerationConfig(config HypothesisGenerationConfig) (HypothesisGenerationConfig, error) {
	normalized := config
	if strings.TrimSpace(normalized.Mode) == "" {
		normalized.Mode = HypothesisGenerationModeEnumerator
	}
	switch strings.TrimSpace(normalized.Mode) {
	case HypothesisGenerationModeEnumerator, HypothesisGenerationModeCurated, HypothesisGenerationModeProposed:
	default:
		return HypothesisGenerationConfig{}, fmt.Errorf("unsupported lean4 hypothesis generation mode %q", normalized.Mode)
	}

	defaults := DefaultHypothesisGenerationConfig()
	if len(normalized.Atoms) == 0 {
		normalized.Atoms = slices.Clone(defaults.Atoms)
	}
	normalized.Atoms = normalizeAtoms(normalized.Atoms)
	if len(normalized.Atoms) == 0 {
		return HypothesisGenerationConfig{}, fmt.Errorf("atoms must not be empty")
	}
	if normalized.MaxFormulaDepth < 0 {
		return HypothesisGenerationConfig{}, fmt.Errorf("max_formula_depth must be non-negative")
	}
	if normalized.MaxAssumptions < 0 {
		return HypothesisGenerationConfig{}, fmt.Errorf("max_assumptions must be non-negative")
	}
	if normalized.MaxFormulaDepth == 0 {
		normalized.MaxFormulaDepth = defaults.MaxFormulaDepth
	}
	if normalized.MaxAssumptions == 0 {
		normalized.MaxAssumptions = defaults.MaxAssumptions
	}
	if normalized.Limit == 0 {
		normalized.Limit = defaults.Limit
	}
	if len(normalized.Filters) == 0 {
		normalized.Filters = slices.Clone(defaults.Filters)
	}
	filterSeen := make(map[string]struct{}, len(normalized.Filters))
	cleanFilters := make([]string, 0, len(normalized.Filters))
	for _, filter := range normalized.Filters {
		filter = strings.TrimSpace(strings.ToLower(filter))
		if filter == "" {
			continue
		}
		switch filter {
		case "non_duplicate", "syntactic_minimal", "interesting":
		default:
			return HypothesisGenerationConfig{}, fmt.Errorf("unsupported hypothesis-generation filter %q", filter)
		}
		if _, exists := filterSeen[filter]; exists {
			continue
		}
		filterSeen[filter] = struct{}{}
		cleanFilters = append(cleanFilters, filter)
	}
	normalized.Filters = cleanFilters
	return normalized, nil
}

func normalizeAtoms(atoms []string) []string {
	seen := make(map[string]struct{}, len(atoms))
	result := make([]string, 0, len(atoms))
	for _, atom := range atoms {
		atom = strings.TrimSpace(atom)
		if atom == "" {
			continue
		}
		if _, exists := seen[atom]; exists {
			continue
		}
		seen[atom] = struct{}{}
		result = append(result, atom)
	}
	slices.Sort(result)
	return result
}

func enumerateImplicationalFormulas(atoms []string, maxDepth int, valuations []valuation) ([]formulaInfo, error) {
	buckets := make([][]hilbert.Formula, maxDepth+1)
	for _, atom := range atoms {
		buckets[0] = append(buckets[0], &hilbert.Var{Name: atom})
	}
	for depth := 1; depth <= maxDepth; depth++ {
		seen := make(map[string]hilbert.Formula)
		for leftDepth := 0; leftDepth < depth; leftDepth++ {
			for rightDepth := 0; rightDepth < depth; rightDepth++ {
				if max(leftDepth, rightDepth) != depth-1 {
					continue
				}
				for _, left := range buckets[leftDepth] {
					for _, right := range buckets[rightDepth] {
						formula := &hilbert.Imp{Left: left, Right: right}
						seen[formula.String()] = formula
					}
				}
			}
		}
		texts := make([]string, 0, len(seen))
		for text := range seen {
			texts = append(texts, text)
		}
		slices.Sort(texts)
		bucket := make([]hilbert.Formula, 0, len(texts))
		for _, text := range texts {
			bucket = append(bucket, seen[text])
		}
		buckets[depth] = bucket
	}

	infos := make([]formulaInfo, 0)
	for depth, bucket := range buckets {
		for _, formula := range bucket {
			infos = append(infos, formulaInfo{
				Text:      formula.String(),
				Formula:   formula,
				Depth:     depth,
				TruthBits: truthBits(formula, valuations),
			})
		}
	}
	slices.SortFunc(infos, func(a, b formulaInfo) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		return strings.Compare(a.Text, b.Text)
	})
	return infos, nil
}

func enumerateAssumptionSets(formulas []formulaInfo, maxAssumptions int) [][]formulaInfo {
	results := make([][]formulaInfo, 0)
	for size := 0; size <= maxAssumptions; size++ {
		var walk func(start int, current []formulaInfo)
		walk = func(start int, current []formulaInfo) {
			if len(current) == size {
				results = append(results, append([]formulaInfo(nil), current...))
				return
			}
			for idx := start; idx < len(formulas); idx++ {
				walk(idx+1, append(current, formulas[idx]))
			}
		}
		walk(0, nil)
	}
	return results
}

func enumerateValuations(atoms []string) []valuation {
	total := 1 << len(atoms)
	result := make([]valuation, 0, total)
	for mask := 0; mask < total; mask++ {
		assign := make(valuation, len(atoms))
		for idx, atom := range atoms {
			assign[atom] = (mask & (1 << idx)) != 0
		}
		result = append(result, assign)
	}
	return result
}

func truthBits(formula hilbert.Formula, valuations []valuation) string {
	var b strings.Builder
	b.Grow(len(valuations))
	for _, assign := range valuations {
		if evaluateFormula(formula, assign) {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func evaluateFormula(formula hilbert.Formula, values valuation) bool {
	switch f := formula.(type) {
	case *hilbert.Var:
		return values[f.Name]
	case *hilbert.Imp:
		return !evaluateFormula(f.Left, values) || evaluateFormula(f.Right, values)
	case *hilbert.Not:
		return !evaluateFormula(f.Value, values)
	default:
		return false
	}
}

func classifyDerivability(assumptions []formulaInfo, goal formulaInfo, valuations []valuation) string {
	if entailsAssumptions(assumptions, goal, valuations) {
		return DerivabilityStatusDerivable
	}
	return DerivabilityStatusNotDerivable
}

func entailsAssumptions(assumptions []formulaInfo, goal formulaInfo, valuations []valuation) bool {
	for _, assign := range valuations {
		allTrue := true
		for _, assumption := range assumptions {
			if !evaluateFormula(assumption.Formula, assign) {
				allTrue = false
				break
			}
		}
		if allTrue && !evaluateFormula(goal.Formula, assign) {
			return false
		}
	}
	return true
}

func isMinimalAssumptionSet(assumptions []formulaInfo, valuations []valuation) bool {
	if len(assumptions) <= 1 {
		return true
	}
	for idx, assumption := range assumptions {
		others := make([]formulaInfo, 0, len(assumptions)-1)
		others = append(others, assumptions[:idx]...)
		others = append(others, assumptions[idx+1:]...)
		if entailsAssumptions(others, assumption, valuations) {
			return false
		}
	}
	return true
}

func isInterestingCase(assumptions []formulaInfo, goal formulaInfo, status string) bool {
	if len(assumptions) == 0 {
		return goal.Depth > 0
	}
	if len(assumptions) == 1 && assumptions[0].Text == goal.Text {
		return false
	}
	if status == DerivabilityStatusNotDerivable {
		return !(len(assumptions) == 1 && assumptions[0].Depth == 0 && goal.Depth == 0)
	}
	if isSingleMPCase(assumptions, goal) {
		return true
	}
	return len(assumptions) > 1 || goal.Depth > 0
}

func classifyDifficulty(assumptions []formulaInfo, goal formulaInfo) string {
	score := goal.Depth
	for _, assumption := range assumptions {
		score += assumption.Depth
	}
	score += len(assumptions)
	switch {
	case score <= 1:
		return "easy"
	case score <= 3:
		return "medium"
	default:
		return "hard"
	}
}

func classifyCategory(assumptions []formulaInfo, goal formulaInfo, status string) string {
	for _, assumption := range assumptions {
		if assumption.Text == goal.Text {
			return "assumption_import"
		}
	}
	if isSingleMPCase(assumptions, goal) {
		return "single_mp"
	}
	if len(assumptions) == 0 && status == DerivabilityStatusDerivable {
		return "theorem_synthesis"
	}
	if status == DerivabilityStatusNotDerivable {
		return "negative_refusal"
	}
	return "enumerated_candidate"
}

func isSingleMPCase(assumptions []formulaInfo, goal formulaInfo) bool {
	if len(assumptions) != 2 {
		return false
	}
	for idx, assumption := range assumptions {
		imp, ok := assumption.Formula.(*hilbert.Imp)
		if !ok {
			continue
		}
		other := assumptions[1-idx]
		if hilbert.Equal(other.Formula, imp.Left) && hilbert.Equal(goal.Formula, imp.Right) {
			return true
		}
	}
	return false
}

func buildEnumeratorComment(assumptions []formulaInfo, goal formulaInfo, status string, minimal, interesting bool) string {
	return fmt.Sprintf("Mode A enumerator case: status=%s assumptions=%d goal_depth=%d interesting=%t minimal=%t", status, len(assumptions), goal.Depth, interesting, minimal)
}

func curryLeanStatement(assumptions []formulaInfo, goal formulaInfo) string {
	current := goal.Formula
	for idx := len(assumptions) - 1; idx >= 0; idx-- {
		current = &hilbert.Imp{
			Left:  assumptions[idx].Formula,
			Right: current,
		}
	}
	return current.String()
}

func assumptionTexts(assumptions []formulaInfo) []string {
	result := make([]string, 0, len(assumptions))
	for _, assumption := range assumptions {
		result = append(result, assumption.Text)
	}
	return result
}

func buildSyntacticCaseKey(assumptions []string, goal string) string {
	return strings.Join(assumptions, " ; ") + " |- " + goal
}

func hasGenerationFilter(filters []string, want string) bool {
	for _, filter := range filters {
		if filter == want {
			return true
		}
	}
	return false
}

func generatedCaseID(mode string, index int) string {
	switch mode {
	case HypothesisGenerationModeCurated:
		return fmt.Sprintf("LEANC%02d", index)
	case HypothesisGenerationModeProposed:
		return fmt.Sprintf("LEANP%02d", index)
	default:
		return fmt.Sprintf("LEANE%02d", index)
	}
}

func buildGeneratedTheoremPackID(mode string, ts time.Time) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = HypothesisGenerationModeEnumerator
	}
	return sanitizeTheoremPackID(fmt.Sprintf("%s_%s", mode, ts.UTC().Format("20060102T150405Z")))
}

func sanitizeTheoremPackID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown_pack"
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
	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "unknown_pack"
	}
	return result
}

func historicalTheoremSnapshotPath(theoremsDir, packID string) string {
	return filepath.Join(theoremsDir, HistoricalTheoremsDir, "theorems_"+sanitizeTheoremPackID(packID)+".csv")
}

func HistoricalTheoremSnapshotPath(theoremsDir, packID string) string {
	return historicalTheoremSnapshotPath(theoremsDir, packID)
}

func writeHistoricalTheoremSnapshot(theoremsDir, packID, sourcePath string) error {
	bytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read theorem snapshot source: %w", err)
	}
	targetPath := historicalTheoremSnapshotPath(theoremsDir, packID)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create theorem historical snapshot dir: %w", err)
	}
	if err := os.WriteFile(targetPath, bytes, 0o644); err != nil {
		return fmt.Errorf("write theorem historical snapshot: %w", err)
	}
	return nil
}

func writeHypothesisSeedCSV(path string, seeds []hypothesisSeed) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create lean4 seed dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create lean4 seed csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"case_id", "category", "difficulty", "assumptions_json", "goal", "comment"}); err != nil {
		return fmt.Errorf("write lean4 seed header: %w", err)
	}
	for _, seed := range seeds {
		assumptionsJSON, err := marshalJSONNoEscape(seed.Assumptions)
		if err != nil {
			return fmt.Errorf("marshal lean4 seed assumptions for %s: %w", seed.CaseID, err)
		}
		if err := writer.Write([]string{
			seed.CaseID,
			seed.Category,
			seed.Difficulty,
			string(assumptionsJSON),
			seed.Goal,
			seed.Comment,
		}); err != nil {
			return fmt.Errorf("write lean4 seed row %s: %w", seed.CaseID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush lean4 seed csv: %w", err)
	}
	return nil
}

func loadHypothesisSeedCSV(path string) ([]hypothesisSeed, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open hypothesis seed csv: %w", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read hypothesis seed csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("hypothesis seed csv is empty")
	}
	columns := make(map[string]int, len(records[0]))
	for idx, header := range records[0] {
		columns[strings.TrimSpace(header)] = idx
	}
	required := []string{"assumptions_json", "goal"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			return nil, fmt.Errorf("hypothesis seed csv is missing required column %q", column)
		}
	}

	seeds := make([]hypothesisSeed, 0, len(records)-1)
	for rowIndex, record := range records[1:] {
		seed := hypothesisSeed{
			CaseID:     valueAt(record, columns, "case_id"),
			Category:   valueAt(record, columns, "category"),
			Difficulty: valueAt(record, columns, "difficulty"),
			Goal:       valueAt(record, columns, "goal"),
			Comment:    valueAt(record, columns, "comment"),
		}
		if strings.TrimSpace(seed.Goal) == "" {
			return nil, fmt.Errorf("hypothesis seed row %d has blank goal", rowIndex+2)
		}
		if err := json.Unmarshal([]byte(valueAt(record, columns, "assumptions_json")), &seed.Assumptions); err != nil {
			return nil, fmt.Errorf("decode assumptions_json at row %d: %w", rowIndex+2, err)
		}
		seeds = append(seeds, seed)
	}
	return seeds, nil
}

func valueAt(record []string, columns map[string]int, name string) string {
	idx, ok := columns[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func marshalJSONNoEscape(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return []byte(strings.TrimSpace(buf.String())), nil
}

type parsedFormulaSeed struct {
	Text    string
	Formula hilbert.Formula
	Depth   int
}

type parsedHypothesisSeed struct {
	Seed        hypothesisSeed
	Assumptions []parsedFormulaSeed
	Goal        parsedFormulaSeed
}

func parseSeedHypotheses(seeds []hypothesisSeed, configAtoms []string) ([]parsedHypothesisSeed, []string, error) {
	atomSet := make(map[string]struct{}, len(configAtoms))
	for _, atom := range configAtoms {
		atomSet[strings.TrimSpace(atom)] = struct{}{}
	}

	parsed := make([]parsedHypothesisSeed, 0, len(seeds))
	for _, seed := range seeds {
		parsedAssumptions := make([]parsedFormulaSeed, 0, len(seed.Assumptions))
		for _, assumption := range seed.Assumptions {
			formula, depth, err := parseImplicationalFormula(assumption)
			if err != nil {
				return nil, nil, fmt.Errorf("parse seed assumption %q: %w", assumption, err)
			}
			for atom := range collectAtoms(formula) {
				atomSet[atom] = struct{}{}
			}
			parsedAssumptions = append(parsedAssumptions, parsedFormulaSeed{
				Text:    formula.String(),
				Formula: formula,
				Depth:   depth,
			})
		}
		goalFormula, goalDepth, err := parseImplicationalFormula(seed.Goal)
		if err != nil {
			return nil, nil, fmt.Errorf("parse seed goal %q: %w", seed.Goal, err)
		}
		for atom := range collectAtoms(goalFormula) {
			atomSet[atom] = struct{}{}
		}
		parsed = append(parsed, parsedHypothesisSeed{
			Seed:        seed,
			Assumptions: parsedAssumptions,
			Goal: parsedFormulaSeed{
				Text:    goalFormula.String(),
				Formula: goalFormula,
				Depth:   goalDepth,
			},
		})
	}

	atoms := make([]string, 0, len(atomSet))
	for atom := range atomSet {
		if strings.TrimSpace(atom) == "" {
			continue
		}
		atoms = append(atoms, atom)
	}
	slices.Sort(atoms)
	return parsed, atoms, nil
}

func materializeFormulaInfos(parsed []parsedFormulaSeed, valuations []valuation) []formulaInfo {
	result := make([]formulaInfo, 0, len(parsed))
	for _, item := range parsed {
		result = append(result, materializeFormulaInfo(item, valuations))
	}
	return result
}

func materializeFormulaInfo(parsed parsedFormulaSeed, valuations []valuation) formulaInfo {
	return formulaInfo{
		Text:      parsed.Text,
		Formula:   parsed.Formula,
		Depth:     parsed.Depth,
		TruthBits: truthBits(parsed.Formula, valuations),
	}
}

func parseImplicationalFormula(input string) (hilbert.Formula, int, error) {
	formula, err := hilbert.ParseFormula(input)
	if err != nil {
		return nil, 0, err
	}
	if containsNegation(formula) {
		return nil, 0, fmt.Errorf("negation is not allowed in %s", enumeratorLogicFragment)
	}
	return formula, formulaDepth(formula), nil
}

func containsNegation(formula hilbert.Formula) bool {
	switch f := formula.(type) {
	case *hilbert.Not:
		return true
	case *hilbert.Imp:
		return containsNegation(f.Left) || containsNegation(f.Right)
	default:
		return false
	}
}

func formulaDepth(formula hilbert.Formula) int {
	switch f := formula.(type) {
	case *hilbert.Var:
		return 0
	case *hilbert.Imp:
		return 1 + max(formulaDepth(f.Left), formulaDepth(f.Right))
	case *hilbert.Not:
		return 1 + formulaDepth(f.Value)
	default:
		return 0
	}
}

func collectAtoms(formula hilbert.Formula) map[string]struct{} {
	result := make(map[string]struct{})
	switch f := formula.(type) {
	case *hilbert.Var:
		result[f.Name] = struct{}{}
	case *hilbert.Imp:
		for atom := range collectAtoms(f.Left) {
			result[atom] = struct{}{}
		}
		for atom := range collectAtoms(f.Right) {
			result[atom] = struct{}{}
		}
	case *hilbert.Not:
		for atom := range collectAtoms(f.Value) {
			result[atom] = struct{}{}
		}
	}
	return result
}
