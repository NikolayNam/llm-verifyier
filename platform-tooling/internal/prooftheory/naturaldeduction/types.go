package naturaldeduction

import (
	"fmt"
	"slices"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

const (
	ProofVersionV1   = "1.0.0"
	LogicFragmentV1  = "implicational-prop-v1"
	FrontEndIDV1     = "natural-deduction-v1"
	LoweringTargetV1 = "certificate-format-v1"
)

type StepKind string

const (
	StepKindPremise   StepKind = "premise"
	StepKindAssume    StepKind = "assume"
	StepKindReiterate StepKind = "reiterate"
	StepKindImpElim   StepKind = "imp_elim"
	StepKindImpIntro  StepKind = "imp_intro"
)

type Proof struct {
	ProofVersion string   `json:"proof_version"`
	ProofID      string   `json:"proof_id"`
	Goal         string   `json:"goal"`
	Assumptions  []string `json:"assumptions,omitempty"`
	Context      Context  `json:"context"`
	Steps        []Step   `json:"steps"`
}

type Context struct {
	Domain         string `json:"domain"`
	LogicFragment  string `json:"logic_fragment"`
	FrontEnd       string `json:"front_end"`
	LoweringTarget string `json:"lowering_target"`
	Generator      string `json:"generator"`
}

type Step struct {
	ID             int      `json:"id"`
	Kind           StepKind `json:"kind"`
	PremiseRef     *int     `json:"premise_ref,omitempty"`
	From           []int    `json:"from,omitempty"`
	Scope          *int     `json:"scope,omitempty"`
	DischargeScope *int     `json:"discharge_scope,omitempty"`
	Formula        string   `json:"formula"`
}

func (p *Proof) Normalize() {
	if p.Assumptions == nil {
		p.Assumptions = []string{}
	}
}

func (p *Proof) Validate() error {
	p.Normalize()

	if strings.TrimSpace(p.ProofVersion) != ProofVersionV1 {
		return fmt.Errorf("proof_version must be %q", ProofVersionV1)
	}
	if strings.TrimSpace(p.ProofID) == "" {
		return fmt.Errorf("proof_id is required")
	}
	goal, err := parseImplicationalFormula(p.Goal)
	if err != nil {
		return fmt.Errorf("goal: %w", err)
	}
	if strings.TrimSpace(p.Context.Domain) == "" {
		return fmt.Errorf("context.domain is required")
	}
	if strings.TrimSpace(p.Context.LogicFragment) != LogicFragmentV1 {
		return fmt.Errorf("context.logic_fragment must be %q", LogicFragmentV1)
	}
	if strings.TrimSpace(p.Context.FrontEnd) != FrontEndIDV1 {
		return fmt.Errorf("context.front_end must be %q", FrontEndIDV1)
	}
	if strings.TrimSpace(p.Context.LoweringTarget) != LoweringTargetV1 {
		return fmt.Errorf("context.lowering_target must be %q", LoweringTargetV1)
	}
	if strings.TrimSpace(p.Context.Generator) == "" {
		return fmt.Errorf("context.generator is required")
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("steps must not be empty")
	}

	seenAssumptions := make(map[string]struct{}, len(p.Assumptions))
	assumptions := make([]hilbert.Formula, len(p.Assumptions))
	for idx, assumption := range p.Assumptions {
		if strings.TrimSpace(assumption) == "" {
			return fmt.Errorf("assumptions[%d] must not be blank", idx)
		}
		if _, exists := seenAssumptions[assumption]; exists {
			return fmt.Errorf("assumptions[%d] duplicates an earlier assumption", idx)
		}
		seenAssumptions[assumption] = struct{}{}
		formula, err := parseImplicationalFormula(assumption)
		if err != nil {
			return fmt.Errorf("assumptions[%d]: %w", idx, err)
		}
		assumptions[idx] = formula
	}

	stepFormulas := make(map[int]hilbert.Formula, len(p.Steps))
	stepScopes := make(map[int][]int, len(p.Steps))
	scopeOpeners := make(map[int]int)
	openScopes := make([]int, 0, len(p.Steps))

	prevID := 0
	for idx := range p.Steps {
		step := p.Steps[idx]
		if err := step.validateShape(); err != nil {
			return err
		}
		if step.ID <= prevID {
			return fmt.Errorf("step ids must be strictly increasing; got %d after %d", step.ID, prevID)
		}
		prevID = step.ID

		formula, err := parseImplicationalFormula(step.Formula)
		if err != nil {
			return fmt.Errorf("step %d: %w", step.ID, err)
		}

		var stepContext []int

		switch step.Kind {
		case StepKindPremise:
			if len(openScopes) != 0 {
				return fmt.Errorf("step %d: premise steps must appear at top level", step.ID)
			}
			ref := *step.PremiseRef
			if ref > len(assumptions) {
				return fmt.Errorf("step %d: premise_ref %d exceeds assumptions length %d", step.ID, ref, len(assumptions))
			}
			if !hilbert.Equal(formula, assumptions[ref-1]) {
				return fmt.Errorf("step %d: premise formula %q does not match assumptions[%d]", step.ID, formula.String(), ref-1)
			}
			stepContext = nil
		case StepKindAssume:
			scope := *step.Scope
			if _, exists := scopeOpeners[scope]; exists {
				return fmt.Errorf("step %d: scope %d is reused", step.ID, scope)
			}
			stepContext = append(copyScopeChain(openScopes), scope)
			openScopes = append(openScopes, scope)
			scopeOpeners[scope] = step.ID
		case StepKindReiterate:
			stepContext, err = validateScopedStep(step, openScopes)
			if err != nil {
				return err
			}

			sourceID := step.From[0]
			sourceFormula, ok := stepFormulas[sourceID]
			if !ok {
				return fmt.Errorf("step %d: from step %d is missing or not prior", step.ID, sourceID)
			}
			sourceContext := stepScopes[sourceID]
			if !isAccessibleFrom(stepContext, sourceContext) {
				return fmt.Errorf("step %d: step %d is not accessible from the current scope chain", step.ID, sourceID)
			}
			if !hilbert.Equal(formula, sourceFormula) {
				return fmt.Errorf("step %d: reiterated formula %q does not match step %d", step.ID, formula.String(), sourceID)
			}
		case StepKindImpElim:
			stepContext, err = validateScopedStep(step, openScopes)
			if err != nil {
				return err
			}

			antecedentID, implicationID := step.From[0], step.From[1]
			antecedent, ok := stepFormulas[antecedentID]
			if !ok {
				return fmt.Errorf("step %d: antecedent step %d is missing or not prior", step.ID, antecedentID)
			}
			implication, ok := stepFormulas[implicationID]
			if !ok {
				return fmt.Errorf("step %d: implication step %d is missing or not prior", step.ID, implicationID)
			}
			if !isAccessibleFrom(stepContext, stepScopes[antecedentID]) {
				return fmt.Errorf("step %d: antecedent step %d is not accessible from the current scope chain", step.ID, antecedentID)
			}
			if !isAccessibleFrom(stepContext, stepScopes[implicationID]) {
				return fmt.Errorf("step %d: implication step %d is not accessible from the current scope chain", step.ID, implicationID)
			}
			imp, ok := implication.(*hilbert.Imp)
			if !ok {
				return fmt.Errorf("step %d: implication step %d is not an implication", step.ID, implicationID)
			}
			if !hilbert.Equal(antecedent, imp.Left) {
				return fmt.Errorf("step %d: antecedent step %d does not match the left side of implication step %d", step.ID, antecedentID, implicationID)
			}
			if !hilbert.Equal(formula, imp.Right) {
				return fmt.Errorf("step %d: formula %q does not match the right side of implication step %d", step.ID, formula.String(), implicationID)
			}
		case StepKindImpIntro:
			if len(openScopes) == 0 {
				return fmt.Errorf("step %d: imp_intro requires an open scope to discharge", step.ID)
			}

			dischargeScope := *step.DischargeScope
			currentChain := copyScopeChain(openScopes)
			if currentChain[len(currentChain)-1] != dischargeScope {
				return fmt.Errorf("step %d: discharge_scope %d is not the current innermost open scope", step.ID, dischargeScope)
			}

			assumeID, bodyID := step.From[0], step.From[1]
			openerID, ok := scopeOpeners[dischargeScope]
			if !ok {
				return fmt.Errorf("step %d: discharge_scope %d was never opened", step.ID, dischargeScope)
			}
			if assumeID != openerID {
				return fmt.Errorf("step %d: first imp_intro reference must be the assume step that opened scope %d", step.ID, dischargeScope)
			}

			assumeFormula, ok := stepFormulas[assumeID]
			if !ok {
				return fmt.Errorf("step %d: assume step %d is missing or not prior", step.ID, assumeID)
			}
			bodyFormula, ok := stepFormulas[bodyID]
			if !ok {
				return fmt.Errorf("step %d: body step %d is missing or not prior", step.ID, bodyID)
			}
			if !slices.Equal(stepScopes[assumeID], currentChain) {
				return fmt.Errorf("step %d: assume step %d does not belong to the discharged scope chain", step.ID, assumeID)
			}
			if !slices.Equal(stepScopes[bodyID], currentChain) {
				return fmt.Errorf("step %d: body step %d does not belong to the discharged scope chain", step.ID, bodyID)
			}

			expected := &hilbert.Imp{
				Left:  assumeFormula,
				Right: bodyFormula,
			}
			if !hilbert.Equal(formula, expected) {
				return fmt.Errorf("step %d: imp_intro formula %q does not match the discharged implication", step.ID, formula.String())
			}

			openScopes = openScopes[:len(openScopes)-1]
			stepContext = copyScopeChain(openScopes)
		default:
			return fmt.Errorf("step %d: unsupported step kind %q", step.ID, step.Kind)
		}

		stepFormulas[step.ID] = formula
		stepScopes[step.ID] = stepContext
	}

	lastStep := p.Steps[len(p.Steps)-1]
	lastFormula := stepFormulas[lastStep.ID]
	if !hilbert.Equal(goal, lastFormula) {
		return fmt.Errorf("goal %q does not match the final step formula %q", goal.String(), lastFormula.String())
	}

	return nil
}

func (s Step) validateShape() error {
	if s.ID <= 0 {
		return fmt.Errorf("step id must be positive")
	}
	if strings.TrimSpace(s.Formula) == "" {
		return fmt.Errorf("step %d: formula is required", s.ID)
	}

	switch s.Kind {
	case StepKindPremise:
		if s.PremiseRef == nil || *s.PremiseRef <= 0 {
			return fmt.Errorf("step %d: premise requires a positive premise_ref", s.ID)
		}
		if len(s.From) != 0 {
			return fmt.Errorf("step %d: premise must not define from", s.ID)
		}
		if s.Scope != nil {
			return fmt.Errorf("step %d: premise must not define scope", s.ID)
		}
		if s.DischargeScope != nil {
			return fmt.Errorf("step %d: premise must not define discharge_scope", s.ID)
		}
	case StepKindAssume:
		if s.Scope == nil || *s.Scope <= 0 {
			return fmt.Errorf("step %d: assume requires a positive scope", s.ID)
		}
		if s.PremiseRef != nil {
			return fmt.Errorf("step %d: assume must not define premise_ref", s.ID)
		}
		if len(s.From) != 0 {
			return fmt.Errorf("step %d: assume must not define from", s.ID)
		}
		if s.DischargeScope != nil {
			return fmt.Errorf("step %d: assume must not define discharge_scope", s.ID)
		}
	case StepKindReiterate:
		if s.PremiseRef != nil {
			return fmt.Errorf("step %d: reiterate must not define premise_ref", s.ID)
		}
		if s.DischargeScope != nil {
			return fmt.Errorf("step %d: reiterate must not define discharge_scope", s.ID)
		}
		if err := validateFromIDs(s.ID, s.From, 1, false); err != nil {
			return err
		}
		if s.Scope != nil && *s.Scope <= 0 {
			return fmt.Errorf("step %d: scope must be positive when present", s.ID)
		}
	case StepKindImpElim:
		if s.PremiseRef != nil {
			return fmt.Errorf("step %d: imp_elim must not define premise_ref", s.ID)
		}
		if s.DischargeScope != nil {
			return fmt.Errorf("step %d: imp_elim must not define discharge_scope", s.ID)
		}
		if err := validateFromIDs(s.ID, s.From, 2, true); err != nil {
			return err
		}
		if s.Scope != nil && *s.Scope <= 0 {
			return fmt.Errorf("step %d: scope must be positive when present", s.ID)
		}
	case StepKindImpIntro:
		if s.DischargeScope == nil || *s.DischargeScope <= 0 {
			return fmt.Errorf("step %d: imp_intro requires a positive discharge_scope", s.ID)
		}
		if s.PremiseRef != nil {
			return fmt.Errorf("step %d: imp_intro must not define premise_ref", s.ID)
		}
		if s.Scope != nil {
			return fmt.Errorf("step %d: imp_intro must not define scope", s.ID)
		}
		if err := validateFromIDs(s.ID, s.From, 2, true); err != nil {
			return err
		}
	default:
		return fmt.Errorf("step %d: unsupported step kind %q", s.ID, s.Kind)
	}

	return nil
}

func validateFromIDs(stepID int, refs []int, want int, mustBeDistinct bool) error {
	if len(refs) != want {
		return fmt.Errorf("step %d: expected %d from references, got %d", stepID, want, len(refs))
	}
	for _, ref := range refs {
		if ref <= 0 {
			return fmt.Errorf("step %d: from references must be positive", stepID)
		}
	}
	if mustBeDistinct && refs[0] == refs[1] {
		return fmt.Errorf("step %d: from references must be distinct", stepID)
	}
	return nil
}

func validateScopedStep(step Step, openScopes []int) ([]int, error) {
	if len(openScopes) == 0 {
		if step.Scope != nil {
			return nil, fmt.Errorf("step %d: top-level %s step must omit scope", step.ID, step.Kind)
		}
		return nil, nil
	}
	if step.Scope == nil {
		return nil, fmt.Errorf("step %d: %s inside an open scope must declare the current scope", step.ID, step.Kind)
	}
	current := openScopes[len(openScopes)-1]
	if *step.Scope != current {
		return nil, fmt.Errorf("step %d: scope %d is not the current innermost open scope %d", step.ID, *step.Scope, current)
	}
	return copyScopeChain(openScopes), nil
}

func copyScopeChain(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	return out
}

func isAccessibleFrom(current, referenced []int) bool {
	if len(referenced) == 0 {
		return true
	}
	if len(referenced) > len(current) {
		return false
	}
	return slices.Equal(current[:len(referenced)], referenced)
}

func parseImplicationalFormula(input string) (hilbert.Formula, error) {
	formula, err := hilbert.ParseFormula(input)
	if err != nil {
		return nil, fmt.Errorf("parse formula: %w", err)
	}
	if containsNegation(formula) {
		return nil, fmt.Errorf("parse formula: negation is not allowed in %s", LogicFragmentV1)
	}
	return formula, nil
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
