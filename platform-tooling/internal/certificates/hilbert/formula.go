package hilbert

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	RulePackID = "classical-hilbert-v1"
	SyntaxID   = "hilbert-prop-ascii-v1"
)

type Formula interface {
	isFormula()
	String() string
}

type Var struct {
	Name string
}

func (*Var) isFormula() {}

func (v *Var) String() string {
	return v.Name
}

type Not struct {
	Value Formula
}

func (*Not) isFormula() {}

func (n *Not) String() string {
	switch n.Value.(type) {
	case *Var, *Not:
		return "!" + n.Value.String()
	default:
		return "!(" + n.Value.String() + ")"
	}
}

type Imp struct {
	Left  Formula
	Right Formula
}

func (*Imp) isFormula() {}

func (i *Imp) String() string {
	left := i.Left.String()
	right := i.Right.String()

	if _, ok := i.Left.(*Imp); ok {
		left = "(" + left + ")"
	}
	if _, ok := i.Right.(*Imp); ok {
		right = "(" + right + ")"
	}

	return left + " -> " + right
}

func Equal(a, b Formula) bool {
	switch av := a.(type) {
	case *Var:
		bv, ok := b.(*Var)
		return ok && av.Name == bv.Name
	case *Not:
		bv, ok := b.(*Not)
		return ok && Equal(av.Value, bv.Value)
	case *Imp:
		bv, ok := b.(*Imp)
		return ok && Equal(av.Left, bv.Left) && Equal(av.Right, bv.Right)
	default:
		return false
	}
}

func ParseFormula(input string) (Formula, error) {
	p := &parser{input: strings.TrimSpace(input)}
	if p.input == "" {
		return nil, fmt.Errorf("formula must not be blank")
	}

	formula, err := p.parseImplication()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if !p.eof() {
		return nil, fmt.Errorf("unexpected token at position %d", p.pos+1)
	}

	return formula, nil
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseImplication() (Formula, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()
	if p.match("->") {
		right, err := p.parseImplication()
		if err != nil {
			return nil, err
		}
		return &Imp{Left: left, Right: right}, nil
	}

	return left, nil
}

func (p *parser) parseUnary() (Formula, error) {
	p.skipWhitespace()
	if p.match("!") {
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Not{Value: value}, nil
	}

	return p.parseAtom()
}

func (p *parser) parseAtom() (Formula, error) {
	p.skipWhitespace()
	if p.eof() {
		return nil, fmt.Errorf("unexpected end of formula")
	}

	if p.match("(") {
		value, err := p.parseImplication()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.match(")") {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos+1)
		}
		return value, nil
	}

	identifier, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	return &Var{Name: identifier}, nil
}

func (p *parser) parseIdentifier() (string, error) {
	p.skipWhitespace()
	if p.eof() {
		return "", fmt.Errorf("expected identifier at position %d", p.pos+1)
	}

	start := p.pos
	r, size := p.peekRune()
	if !(unicode.IsLetter(r) || r == '_') {
		return "", fmt.Errorf("expected identifier at position %d", p.pos+1)
	}
	p.pos += size

	for !p.eof() {
		r, size = p.peekRune()
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			break
		}
		p.pos += size
	}

	return p.input[start:p.pos], nil
}

func (p *parser) skipWhitespace() {
	for !p.eof() {
		r, size := p.peekRune()
		if !unicode.IsSpace(r) {
			return
		}
		p.pos += size
	}
}

func (p *parser) match(token string) bool {
	if strings.HasPrefix(p.input[p.pos:], token) {
		p.pos += len(token)
		return true
	}
	return false
}

func (p *parser) peekRune() (rune, int) {
	return rune(p.input[p.pos]), 1
}

func (p *parser) eof() bool {
	return p.pos >= len(p.input)
}
