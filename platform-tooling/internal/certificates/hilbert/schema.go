package hilbert

type Pattern interface {
	isPattern()
}

type PVar struct {
	Name string
}

func (*PVar) isPattern() {}

type PNot struct {
	Value Pattern
}

func (*PNot) isPattern() {}

type PImp struct {
	Left  Pattern
	Right Pattern
}

func (*PImp) isPattern() {}

type Schema struct {
	Name    string
	Pattern Pattern
}

func matchSchema(pattern Pattern, formula Formula, env map[string]Formula) bool {
	switch p := pattern.(type) {
	case *PVar:
		if bound, ok := env[p.Name]; ok {
			return Equal(bound, formula)
		}
		env[p.Name] = formula
		return true
	case *PNot:
		f, ok := formula.(*Not)
		if !ok {
			return false
		}
		return matchSchema(p.Value, f.Value, env)
	case *PImp:
		f, ok := formula.(*Imp)
		if !ok {
			return false
		}
		return matchSchema(p.Left, f.Left, env) && matchSchema(p.Right, f.Right, env)
	default:
		return false
	}
}

func ClassicalSchemas() []Schema {
	phi := &PVar{Name: "phi"}
	psi := &PVar{Name: "psi"}
	chi := &PVar{Name: "chi"}

	return []Schema{
		{
			Name: "A1",
			Pattern: &PImp{
				Left:  phi,
				Right: &PImp{Left: psi, Right: phi},
			},
		},
		{
			Name: "A2",
			Pattern: &PImp{
				Left: &PImp{
					Left:  phi,
					Right: &PImp{Left: psi, Right: chi},
				},
				Right: &PImp{
					Left:  &PImp{Left: phi, Right: psi},
					Right: &PImp{Left: phi, Right: chi},
				},
			},
		},
		{
			Name: "A3",
			Pattern: &PImp{
				Left: &PImp{
					Left:  &PNot{Value: psi},
					Right: &PNot{Value: phi},
				},
				Right: &PImp{
					Left:  phi,
					Right: psi,
				},
			},
		},
	}
}
