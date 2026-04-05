package naturaldeduction

import (
	"fmt"
	"slices"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

const LoweringGeneratorV1 = "nd-to-hilbert-v1"

type LoweringOptions struct {
	Domain    string
	Generator string
}

type block struct {
	Items []blockItem
}

type blockItem interface {
	isBlockItem()
}

type plainItem struct {
	Step Step
}

func (plainItem) isBlockItem() {}

type subproofItem struct {
	Assume   Step
	Body     block
	ImpIntro Step
}

func (subproofItem) isBlockItem() {}

type internalLine struct {
	Kind          certificates.StepKind
	Formula       hilbert.Formula
	AssumptionRef int
	Axiom         string
	Premises      []int
}

type internalProof struct {
	assumptions []hilbert.Formula
	lines       []internalLine
}

func LowerToCertificate(proof *Proof, opts LoweringOptions) (*certificates.Certificate, error) {
	if proof == nil {
		return nil, NewError(ErrorClassInternal, fmt.Errorf("natural deduction proof is nil"))
	}
	if err := proof.Validate(); err != nil {
		return nil, NewError(classifyValidationError(err), fmt.Errorf("validate natural deduction proof before lowering: %w", err))
	}

	parsedBlock, err := parseProofBlock(proof.Steps)
	if err != nil {
		return nil, NewError(ErrorClassValidation, fmt.Errorf("parse natural deduction block structure: %w", err))
	}

	topLevelAssumptions := make([]hilbert.Formula, len(proof.Assumptions))
	for idx, assumption := range proof.Assumptions {
		formula, err := parseImplicationalFormula(assumption)
		if err != nil {
			return nil, NewError(ErrorClassParse, fmt.Errorf("parse assumptions[%d] during lowering: %w", idx, err))
		}
		topLevelAssumptions[idx] = formula
	}

	current := &internalProof{
		assumptions: topLevelAssumptions,
		lines:       []internalLine{},
	}
	mapping := make(map[int]int)
	if err := translateBlock(current, mapping, parsedBlock); err != nil {
		return nil, NewError(ErrorClassValidation, fmt.Errorf("translate natural deduction proof: %w", err))
	}

	finalStepID := proof.Steps[len(proof.Steps)-1].ID
	finalLine, ok := mapping[finalStepID]
	if !ok {
		return nil, NewError(ErrorClassInternal, fmt.Errorf("final ND step %d was not lowered", finalStepID))
	}
	if finalLine != len(current.lines) {
		return nil, NewError(ErrorClassInternal, fmt.Errorf("lowering produced final ND line %d at Hilbert line %d, but proof ends at line %d", finalStepID, finalLine, len(current.lines)))
	}

	domain := opts.Domain
	if domain == "" {
		domain = proof.Context.Domain
	}
	generator := opts.Generator
	if generator == "" {
		generator = proof.Context.Generator + "+" + LoweringGeneratorV1
	}
	cert := &certificates.Certificate{
		CertificateVersion: certificates.FormatVersionV1,
		ProofID:            proof.ProofID,
		Goal:               proof.Goal,
		Assumptions:        slices.Clone(proof.Assumptions),
		Context: certificates.Context{
			Domain:    domain,
			RulePack:  hilbert.RulePackID,
			Syntax:    hilbert.SyntaxID,
			Generator: generator,
		},
		Steps: make([]certificates.Step, 0, len(current.lines)),
	}

	for _, line := range current.lines {
		step := certificates.Step{
			Kind:    line.Kind,
			Formula: line.Formula.String(),
		}
		switch line.Kind {
		case certificates.StepKindAssumption:
			step.AssumptionRef = line.AssumptionRef
		case certificates.StepKindAxiom:
			step.Axiom = line.Axiom
		case certificates.StepKindModusPonens:
			step.Premises = slices.Clone(line.Premises)
		default:
			return nil, NewError(ErrorClassInternal, fmt.Errorf("unsupported lowered step kind %q", line.Kind))
		}
		cert.Steps = append(cert.Steps, step)
	}

	if _, err := hilbert.VerifyCertificate(cert); err != nil {
		return nil, NewError(ErrorClassValidation, fmt.Errorf("lowered certificate failed Hilbert verification: %w", err))
	}

	return cert, nil
}

func parseProofBlock(steps []Step) (block, error) {
	parsed, next, err := parseBlockFrom(steps, 0, 0, 0)
	if err != nil {
		return block{}, err
	}
	if next != len(steps) {
		return block{}, fmt.Errorf("unexpected trailing steps after top-level parse at index %d", next)
	}
	return parsed, nil
}

func parseBlockFrom(steps []Step, start, closingScope, closingAssumeID int) (block, int, error) {
	items := make([]blockItem, 0)
	for idx := start; idx < len(steps); {
		step := steps[idx]
		if closingScope != 0 &&
			step.Kind == StepKindImpIntro &&
			step.DischargeScope != nil &&
			*step.DischargeScope == closingScope &&
			len(step.From) == 2 &&
			step.From[0] == closingAssumeID {
			return block{Items: items}, idx, nil
		}

		switch step.Kind {
		case StepKindPremise, StepKindReiterate, StepKindImpElim:
			items = append(items, plainItem{Step: step})
			idx++
		case StepKindAssume:
			if step.Scope == nil {
				return block{}, 0, fmt.Errorf("assume step %d is missing scope", step.ID)
			}
			body, closeIndex, err := parseBlockFrom(steps, idx+1, *step.Scope, step.ID)
			if err != nil {
				return block{}, 0, err
			}
			if closeIndex >= len(steps) {
				return block{}, 0, fmt.Errorf("scope %d opened at step %d has no closing imp_intro", *step.Scope, step.ID)
			}
			closeStep := steps[closeIndex]
			if closeStep.Kind != StepKindImpIntro {
				return block{}, 0, fmt.Errorf("scope %d opened at step %d closed by non-imp_intro step %d", *step.Scope, step.ID, closeStep.ID)
			}
			items = append(items, subproofItem{
				Assume:   step,
				Body:     body,
				ImpIntro: closeStep,
			})
			idx = closeIndex + 1
		case StepKindImpIntro:
			return block{}, 0, fmt.Errorf("unexpected imp_intro step %d outside its matching scope", step.ID)
		default:
			return block{}, 0, fmt.Errorf("unsupported step kind %q", step.Kind)
		}
	}

	if closingScope != 0 {
		return block{}, 0, fmt.Errorf("scope %d opened at step %d has no closing imp_intro", closingScope, closingAssumeID)
	}
	return block{Items: items}, len(steps), nil
}

func translateBlock(current *internalProof, mapping map[int]int, parsed block) error {
	for _, item := range parsed.Items {
		switch entry := item.(type) {
		case plainItem:
			if err := translatePlainItem(current, mapping, entry.Step); err != nil {
				return err
			}
		case subproofItem:
			fragment, localFinalLine, err := translateSubproof(current, mapping, entry)
			if err != nil {
				return err
			}
			offset := len(current.lines)
			current.appendFragment(fragment)
			mapping[entry.ImpIntro.ID] = offset + localFinalLine
		default:
			return fmt.Errorf("unsupported block item type %T", item)
		}
	}
	return nil
}

func translatePlainItem(current *internalProof, mapping map[int]int, step Step) error {
	formula, err := parseImplicationalFormula(step.Formula)
	if err != nil {
		return fmt.Errorf("step %d: %w", step.ID, err)
	}

	switch step.Kind {
	case StepKindPremise:
		line := current.appendAssumption(*step.PremiseRef, formula)
		mapping[step.ID] = line
	case StepKindReiterate:
		sourceLine, ok := mapping[step.From[0]]
		if !ok {
			return fmt.Errorf("step %d: reiterate source step %d was not lowered", step.ID, step.From[0])
		}
		line, err := current.cloneLine(sourceLine)
		if err != nil {
			return fmt.Errorf("step %d: %w", step.ID, err)
		}
		mapping[step.ID] = line
	case StepKindImpElim:
		antecedentLine, ok := mapping[step.From[0]]
		if !ok {
			return fmt.Errorf("step %d: antecedent step %d was not lowered", step.ID, step.From[0])
		}
		implicationLine, ok := mapping[step.From[1]]
		if !ok {
			return fmt.Errorf("step %d: implication step %d was not lowered", step.ID, step.From[1])
		}
		line := current.appendMP(antecedentLine, implicationLine, formula)
		mapping[step.ID] = line
	default:
		return fmt.Errorf("step %d: unsupported plain ND step kind %q", step.ID, step.Kind)
	}

	return nil
}

func translateSubproof(current *internalProof, mapping map[int]int, item subproofItem) (*internalProof, int, error) {
	localFormula, err := parseImplicationalFormula(item.Assume.Formula)
	if err != nil {
		return nil, 0, fmt.Errorf("step %d: %w", item.Assume.ID, err)
	}

	raw := current.clone()
	raw.assumptions = append(raw.assumptions, localFormula)
	dischargedRef := len(raw.assumptions)
	rawMapping := cloneStepMapping(mapping)
	assumeLine := raw.appendAssumption(dischargedRef, localFormula)
	rawMapping[item.Assume.ID] = assumeLine

	if err := translateBlock(raw, rawMapping, item.Body); err != nil {
		return nil, 0, err
	}

	bodyStepID := item.ImpIntro.From[1]
	bodyLine, ok := rawMapping[bodyStepID]
	if !ok {
		return nil, 0, fmt.Errorf("imp_intro step %d: body step %d was not lowered", item.ImpIntro.ID, bodyStepID)
	}

	transformed, lineMap, err := dischargeAssumption(raw, dischargedRef)
	if err != nil {
		return nil, 0, fmt.Errorf("imp_intro step %d: %w", item.ImpIntro.ID, err)
	}

	finalLine, ok := lineMap[bodyLine]
	if !ok {
		return nil, 0, fmt.Errorf("imp_intro step %d: discharged body line %d missing in transformed proof", item.ImpIntro.ID, bodyLine)
	}

	wantFormula, err := parseImplicationalFormula(item.ImpIntro.Formula)
	if err != nil {
		return nil, 0, fmt.Errorf("imp_intro step %d: %w", item.ImpIntro.ID, err)
	}
	gotFormula := transformed.lines[finalLine-1].Formula
	if !hilbert.Equal(gotFormula, wantFormula) {
		return nil, 0, fmt.Errorf("imp_intro step %d: lowered formula %q does not match ND formula %q", item.ImpIntro.ID, gotFormula.String(), wantFormula.String())
	}

	return transformed, finalLine, nil
}

func dischargeAssumption(raw *internalProof, dischargedRef int) (*internalProof, map[int]int, error) {
	if raw == nil {
		return nil, nil, fmt.Errorf("raw proof is nil")
	}
	if dischargedRef <= 0 || dischargedRef > len(raw.assumptions) {
		return nil, nil, fmt.Errorf("discharged assumption ref %d is out of range", dischargedRef)
	}

	discharged := raw.assumptions[dischargedRef-1]
	remainingAssumptions := make([]hilbert.Formula, 0, len(raw.assumptions)-1)
	remainingAssumptions = append(remainingAssumptions, raw.assumptions[:dischargedRef-1]...)
	remainingAssumptions = append(remainingAssumptions, raw.assumptions[dischargedRef:]...)

	out := &internalProof{
		assumptions: remainingAssumptions,
		lines:       make([]internalLine, 0, len(raw.lines)*4),
	}
	lineMap := make(map[int]int, len(raw.lines))

	for idx, line := range raw.lines {
		switch {
		case line.Kind == certificates.StepKindAssumption && line.AssumptionRef == dischargedRef:
			lineMap[idx+1] = emitIdentity(out, discharged)
		case line.Kind == certificates.StepKindAssumption:
			shiftedRef := line.AssumptionRef
			if shiftedRef > dischargedRef {
				shiftedRef--
			}
			base := out.appendAssumption(shiftedRef, line.Formula)
			a1 := emitA1(out, line.Formula, discharged)
			lineMap[idx+1] = out.appendMP(base, a1, imp(discharged, line.Formula))
		case line.Kind == certificates.StepKindAxiom:
			base := out.appendAxiom(line.Axiom, line.Formula)
			a1 := emitA1(out, line.Formula, discharged)
			lineMap[idx+1] = out.appendMP(base, a1, imp(discharged, line.Formula))
		case line.Kind == certificates.StepKindModusPonens:
			antecedentLine := line.Premises[0]
			implicationLine := line.Premises[1]
			if antecedentLine <= 0 || antecedentLine > len(raw.lines) || implicationLine <= 0 || implicationLine > len(raw.lines) {
				return nil, nil, fmt.Errorf("raw MP line %d references out-of-range premises", idx+1)
			}
			antecedent := raw.lines[antecedentLine-1].Formula
			a2 := emitA2(out, discharged, antecedent, line.Formula)
			step1 := out.appendMP(lineMap[implicationLine], a2, imp(imp(discharged, antecedent), imp(discharged, line.Formula)))
			lineMap[idx+1] = out.appendMP(lineMap[antecedentLine], step1, imp(discharged, line.Formula))
		default:
			return nil, nil, fmt.Errorf("unsupported Hilbert line kind %q during deduction transform", line.Kind)
		}
	}

	return out, lineMap, nil
}

func emitIdentity(out *internalProof, formula hilbert.Formula) int {
	line1 := emitA1(out, formula, formula)
	line2 := emitA1(out, formula, imp(formula, formula))
	line3 := emitA2(out, formula, imp(formula, formula), formula)
	line4 := out.appendMP(line2, line3, imp(imp(formula, imp(formula, formula)), imp(formula, formula)))
	return out.appendMP(line1, line4, imp(formula, formula))
}

func emitA1(out *internalProof, phi, psi hilbert.Formula) int {
	return out.appendAxiom("A1", imp(phi, imp(psi, phi)))
}

func emitA2(out *internalProof, phi, psi, chi hilbert.Formula) int {
	return out.appendAxiom("A2", imp(imp(phi, imp(psi, chi)), imp(imp(phi, psi), imp(phi, chi))))
}

func imp(left, right hilbert.Formula) hilbert.Formula {
	return &hilbert.Imp{
		Left:  left,
		Right: right,
	}
}

func cloneStepMapping(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (p *internalProof) clone() *internalProof {
	if p == nil {
		return &internalProof{}
	}
	cloned := &internalProof{
		assumptions: slices.Clone(p.assumptions),
		lines:       make([]internalLine, 0, len(p.lines)),
	}
	for _, line := range p.lines {
		copied := line
		copied.Premises = slices.Clone(line.Premises)
		cloned.lines = append(cloned.lines, copied)
	}
	return cloned
}

func (p *internalProof) appendFragment(fragment *internalProof) int {
	offset := len(p.lines)
	for _, line := range fragment.lines {
		copied := line
		if len(copied.Premises) != 0 {
			copied.Premises = slices.Clone(copied.Premises)
			for idx := range copied.Premises {
				copied.Premises[idx] += offset
			}
		}
		p.lines = append(p.lines, copied)
	}
	return len(p.lines)
}

func (p *internalProof) appendAssumption(ref int, formula hilbert.Formula) int {
	p.lines = append(p.lines, internalLine{
		Kind:          certificates.StepKindAssumption,
		Formula:       formula,
		AssumptionRef: ref,
	})
	return len(p.lines)
}

func (p *internalProof) appendAxiom(name string, formula hilbert.Formula) int {
	p.lines = append(p.lines, internalLine{
		Kind:    certificates.StepKindAxiom,
		Formula: formula,
		Axiom:   name,
	})
	return len(p.lines)
}

func (p *internalProof) appendMP(antecedentLine, implicationLine int, formula hilbert.Formula) int {
	p.lines = append(p.lines, internalLine{
		Kind:    certificates.StepKindModusPonens,
		Formula: formula,
		Premises: []int{
			antecedentLine,
			implicationLine,
		},
	})
	return len(p.lines)
}

func (p *internalProof) cloneLine(sourceLine int) (int, error) {
	if sourceLine <= 0 || sourceLine > len(p.lines) {
		return 0, fmt.Errorf("cannot clone missing line %d", sourceLine)
	}
	source := p.lines[sourceLine-1]
	switch source.Kind {
	case certificates.StepKindAssumption:
		return p.appendAssumption(source.AssumptionRef, source.Formula), nil
	case certificates.StepKindAxiom:
		return p.appendAxiom(source.Axiom, source.Formula), nil
	case certificates.StepKindModusPonens:
		return p.appendMP(source.Premises[0], source.Premises[1], source.Formula), nil
	default:
		return 0, fmt.Errorf("cannot clone unsupported line kind %q", source.Kind)
	}
}
