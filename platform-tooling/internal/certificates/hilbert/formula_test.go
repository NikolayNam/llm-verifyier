package hilbert

import "testing"

func TestParseFormula_RightAssociativeImplication(t *testing.T) {
	formula, err := ParseFormula("A -> B -> C")
	if err != nil {
		t.Fatalf("ParseFormula() error = %v", err)
	}

	want := &Imp{
		Left: &Var{Name: "A"},
		Right: &Imp{
			Left:  &Var{Name: "B"},
			Right: &Var{Name: "C"},
		},
	}
	if !Equal(formula, want) {
		t.Fatalf("ParseFormula() = %q, want %q", formula.String(), want.String())
	}
}

func TestParseFormula_WithNegationAndParentheses(t *testing.T) {
	formula, err := ParseFormula("!(A -> B) -> C")
	if err != nil {
		t.Fatalf("ParseFormula() error = %v", err)
	}

	if formula.String() != "!(A -> B) -> C" {
		t.Fatalf("ParseFormula().String() = %q", formula.String())
	}
}
