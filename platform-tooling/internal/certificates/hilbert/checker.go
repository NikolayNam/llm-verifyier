package hilbert

import (
	"fmt"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
)

type RulePack struct {
	ID      string
	Syntax  string
	Schemas map[string]Schema
}

func ResolveRulePack(id string) (RulePack, error) {
	switch id {
	case RulePackID:
		schemas := ClassicalSchemas()
		byName := make(map[string]Schema, len(schemas))
		for _, schema := range schemas {
			byName[schema.Name] = schema
		}
		return RulePack{
			ID:      RulePackID,
			Syntax:  SyntaxID,
			Schemas: byName,
		}, nil
	default:
		return RulePack{}, fmt.Errorf("unsupported rule pack %q", id)
	}
}

type VerifiedStep struct {
	Line    int
	Kind    certificates.StepKind
	Formula Formula
	Reason  string
}

type Report struct {
	ProofID  string
	RulePack string
	Goal     Formula
	Steps    []VerifiedStep
}

func (r *Report) String() string {
	var b strings.Builder
	for i, step := range r.Steps {
		if i > 0 {
			b.WriteByte('\n')
		}
		_, _ = fmt.Fprintf(&b, "%d. %s [%s]", step.Line, step.Formula.String(), step.Reason)
	}
	return b.String()
}

func VerifyCertificate(cert *certificates.Certificate) (*Report, error) {
	if cert == nil {
		return nil, certificates.NewError(certificates.ErrorClassInternal, fmt.Errorf("certificate is nil"))
	}
	if err := cert.Validate(); err != nil {
		return nil, certificates.NewError(certificates.ErrorClassSchema, err)
	}

	rulePack, err := ResolveRulePack(cert.Context.RulePack)
	if err != nil {
		return nil, certificates.NewError(certificates.ErrorClassKernel, err)
	}
	if cert.Context.Syntax != rulePack.Syntax {
		return nil, certificates.NewError(
			certificates.ErrorClassParse,
			fmt.Errorf("unsupported syntax %q for rule pack %q", cert.Context.Syntax, cert.Context.RulePack),
		)
	}

	goal, err := ParseFormula(cert.Goal)
	if err != nil {
		return nil, certificates.NewError(certificates.ErrorClassParse, fmt.Errorf("parse goal: %w", err))
	}

	assumptions := make([]Formula, len(cert.Assumptions))
	for idx, assumption := range cert.Assumptions {
		formula, err := ParseFormula(assumption)
		if err != nil {
			return nil, certificates.NewError(certificates.ErrorClassParse, fmt.Errorf("parse assumptions[%d]: %w", idx, err))
		}
		assumptions[idx] = formula
	}

	verified := make([]Formula, len(cert.Steps))
	report := &Report{
		ProofID:  cert.ProofID,
		RulePack: rulePack.ID,
		Goal:     goal,
		Steps:    make([]VerifiedStep, 0, len(cert.Steps)),
	}

	for idx, step := range cert.Steps {
		lineNo := idx + 1
		formula, err := ParseFormula(step.Formula)
		if err != nil {
			return nil, certificates.NewError(certificates.ErrorClassParse, fmt.Errorf("line %d: parse formula: %w", lineNo, err))
		}

		var reason string
		switch step.Kind {
		case certificates.StepKindAssumption:
			if step.AssumptionRef > len(assumptions) {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: assumption_ref %d exceeds assumptions length %d",
					lineNo,
					step.AssumptionRef,
					len(assumptions),
				))
			}
			expected := assumptions[step.AssumptionRef-1]
			if !Equal(formula, expected) {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: assumption mismatch, expected %q but got %q",
					lineNo,
					expected.String(),
					formula.String(),
				))
			}
			reason = fmt.Sprintf("assumption %d", step.AssumptionRef)
		case certificates.StepKindAxiom:
			schema, ok := rulePack.Schemas[step.Axiom]
			if !ok {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf("line %d: unknown axiom %q", lineNo, step.Axiom))
			}
			env := make(map[string]Formula)
			if !matchSchema(schema.Pattern, formula, env) {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: formula %q does not match axiom %s",
					lineNo,
					formula.String(),
					step.Axiom,
				))
			}
			reason = "axiom " + step.Axiom
		case certificates.StepKindModusPonens:
			antecedentLine := step.Premises[0]
			implicationLine := step.Premises[1]
			if antecedentLine <= 0 || implicationLine <= 0 {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf("line %d: MP premises must be positive", lineNo))
			}
			if antecedentLine >= lineNo || implicationLine >= lineNo {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf("line %d: MP premises must reference previous lines only", lineNo))
			}

			antecedent := verified[antecedentLine-1]
			implicationFormula := verified[implicationLine-1]
			imp, ok := implicationFormula.(*Imp)
			if !ok {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: line %d is not an implication, cannot use MP",
					lineNo,
					implicationLine,
				))
			}
			if !Equal(antecedent, imp.Left) {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: MP mismatch, line %d is %q but implication antecedent is %q",
					lineNo,
					antecedentLine,
					antecedent.String(),
					imp.Left.String(),
				))
			}
			if !Equal(formula, imp.Right) {
				return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf(
					"line %d: MP mismatch, expected consequent %q but got %q",
					lineNo,
					imp.Right.String(),
					formula.String(),
				))
			}
			reason = fmt.Sprintf("MP %d,%d", antecedentLine, implicationLine)
		default:
			return nil, certificates.NewError(certificates.ErrorClassKernel, fmt.Errorf("line %d: unsupported step kind %q", lineNo, step.Kind))
		}

		verified[idx] = formula
		report.Steps = append(report.Steps, VerifiedStep{
			Line:    lineNo,
			Kind:    step.Kind,
			Formula: formula,
			Reason:  reason,
		})
	}

	last := verified[len(verified)-1]
	if !Equal(last, goal) {
		return nil, certificates.NewError(
			certificates.ErrorClassKernel,
			fmt.Errorf("final goal mismatch, expected %q but got %q", goal.String(), last.String()),
		)
	}

	return report, nil
}
