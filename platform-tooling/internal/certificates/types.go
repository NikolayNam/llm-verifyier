package certificates

import (
	"fmt"
	"strings"
)

const (
	FormatVersionV1 = "1.0.0"
	DefaultSyntaxV1 = "hilbert-prop-ascii-v1"
)

type StepKind string

const (
	StepKindAssumption  StepKind = "assumption"
	StepKindAxiom       StepKind = "axiom"
	StepKindModusPonens StepKind = "modus_ponens"
)

type Certificate struct {
	CertificateVersion string      `json:"certificate_version"`
	ProofID            string      `json:"proof_id"`
	Goal               string      `json:"goal"`
	Assumptions        []string    `json:"assumptions,omitempty"`
	Context            Context     `json:"context"`
	Steps              []Step      `json:"steps"`
	Sources            []SourceRef `json:"sources,omitempty"`
	Meta               *Meta       `json:"meta,omitempty"`
	Hashes             *Hashes     `json:"hashes,omitempty"`
}

type Context struct {
	Domain    string `json:"domain"`
	RulePack  string `json:"rule_pack"`
	Syntax    string `json:"syntax,omitempty"`
	Generator string `json:"generator,omitempty"`
}

type Step struct {
	Kind          StepKind    `json:"kind"`
	AssumptionRef int         `json:"assumption_ref,omitempty"`
	Axiom         string      `json:"axiom,omitempty"`
	Premises      []int       `json:"premises,omitempty"`
	Formula       string      `json:"formula"`
	SourceRefs    []SourceRef `json:"source_refs,omitempty"`
	Explain       string      `json:"explain,omitempty"`
}

type SourceRef struct {
	ID   string `json:"id"`
	Kind string `json:"kind,omitempty"`
	URI  string `json:"uri,omitempty"`
	Hash string `json:"hash,omitempty"`
}

type Meta struct {
	CreatedAt        string `json:"created_at,omitempty"`
	Generator        string `json:"generator,omitempty"`
	ExplainAvailable bool   `json:"explain_available,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type Hashes struct {
	CertificateSHA256 string `json:"certificate_sha256,omitempty"`
	SourcesSHA256     string `json:"sources_sha256,omitempty"`
	ReportSHA256      string `json:"report_sha256,omitempty"`
}

func (c *Certificate) Normalize() {
	if c.Context.Syntax == "" {
		c.Context.Syntax = DefaultSyntaxV1
	}
	if c.Assumptions == nil {
		c.Assumptions = []string{}
	}
	if c.Sources == nil {
		c.Sources = []SourceRef{}
	}
}

func (c *Certificate) Validate() error {
	c.Normalize()

	if strings.TrimSpace(c.CertificateVersion) != FormatVersionV1 {
		return fmt.Errorf("certificate_version must be %q", FormatVersionV1)
	}
	if strings.TrimSpace(c.ProofID) == "" {
		return fmt.Errorf("proof_id is required")
	}
	if strings.TrimSpace(c.Goal) == "" {
		return fmt.Errorf("goal is required")
	}
	if strings.TrimSpace(c.Context.Domain) == "" {
		return fmt.Errorf("context.domain is required")
	}
	if strings.TrimSpace(c.Context.RulePack) == "" {
		return fmt.Errorf("context.rule_pack is required")
	}
	if strings.TrimSpace(c.Context.Syntax) == "" {
		return fmt.Errorf("context.syntax is required")
	}
	if len(c.Steps) == 0 {
		return fmt.Errorf("steps must not be empty")
	}

	seenAssumptions := make(map[string]struct{}, len(c.Assumptions))
	for idx, assumption := range c.Assumptions {
		if strings.TrimSpace(assumption) == "" {
			return fmt.Errorf("assumptions[%d] must not be blank", idx)
		}
		if _, exists := seenAssumptions[assumption]; exists {
			return fmt.Errorf("assumptions[%d] duplicates an earlier assumption", idx)
		}
		seenAssumptions[assumption] = struct{}{}
	}

	for idx := range c.Steps {
		if err := c.Steps[idx].Validate(idx); err != nil {
			return err
		}
	}

	return nil
}

func (s Step) Validate(index int) error {
	line := index + 1

	if strings.TrimSpace(s.Formula) == "" {
		return fmt.Errorf("line %d: formula is required", line)
	}

	switch s.Kind {
	case StepKindAssumption:
		if s.AssumptionRef <= 0 {
			return fmt.Errorf("line %d: assumption_ref must be positive", line)
		}
		if strings.TrimSpace(s.Axiom) != "" {
			return fmt.Errorf("line %d: assumption step must not define axiom", line)
		}
		if len(s.Premises) != 0 {
			return fmt.Errorf("line %d: assumption step must not define premises", line)
		}
	case StepKindAxiom:
		if strings.TrimSpace(s.Axiom) == "" {
			return fmt.Errorf("line %d: axiom step requires axiom", line)
		}
		if s.AssumptionRef != 0 {
			return fmt.Errorf("line %d: axiom step must not define assumption_ref", line)
		}
		if len(s.Premises) != 0 {
			return fmt.Errorf("line %d: axiom step must not define premises", line)
		}
	case StepKindModusPonens:
		if len(s.Premises) != 2 {
			return fmt.Errorf("line %d: modus_ponens requires exactly 2 premises", line)
		}
		if s.AssumptionRef != 0 {
			return fmt.Errorf("line %d: modus_ponens must not define assumption_ref", line)
		}
		if strings.TrimSpace(s.Axiom) != "" {
			return fmt.Errorf("line %d: modus_ponens must not define axiom", line)
		}
	default:
		return fmt.Errorf("line %d: unsupported step kind %q", line, s.Kind)
	}

	for srcIdx, src := range s.SourceRefs {
		if strings.TrimSpace(src.ID) == "" {
			return fmt.Errorf("line %d: source_refs[%d].id is required", line, srcIdx)
		}
	}

	return nil
}
