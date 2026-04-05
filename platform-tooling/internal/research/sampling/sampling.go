package sampling

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

type Params struct {
	Temperature *float64 `json:"temperature,omitempty"`
	Seed        *int64   `json:"seed,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

type Resolution struct {
	Surface          string  `json:"surface,omitempty"`
	Requested        *Params `json:"requested_sampling_family,omitempty"`
	Effective        *Params `json:"effective_sampling_by_surface,omitempty"`
	Unsupported      *Params `json:"unsupported_or_undocumented,omitempty"`
	RequestedProfile string  `json:"requested_sampling_profile,omitempty"`
	EffectiveProfile string  `json:"effective_sampling_profile,omitempty"`
}

func FromConfig(cfg *researchconfig.SamplingConfig) *Params {
	if cfg == nil {
		return nil
	}
	return NewParams(cfg.Temperature, cfg.Seed, cfg.TopP)
}

func NewParams(temperature *float64, seed *int64, topP *float64) *Params {
	params := &Params{
		Temperature: cloneFloat64(temperature),
		Seed:        cloneInt64(seed),
		TopP:        cloneFloat64(topP),
	}
	if params.Empty() {
		return nil
	}
	return params
}

func (p *Params) Empty() bool {
	return p == nil || (p.Temperature == nil && p.Seed == nil && p.TopP == nil)
}

func (p *Params) JSON() string {
	if p.Empty() {
		return ""
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(raw)
}

func Profile(params *Params) string {
	if params.Empty() {
		return ""
	}
	parts := make([]string, 0, 3)
	if params.Temperature != nil {
		parts = append(parts, "temp"+formatFloat(*params.Temperature))
	}
	if params.Seed != nil {
		parts = append(parts, fmt.Sprintf("seed%d", *params.Seed))
	}
	if params.TopP != nil {
		parts = append(parts, "topp"+formatFloat(*params.TopP))
	}
	return strings.Join(parts, "_")
}

func Resolve(provider string, requested *Params) Resolution {
	provider = strings.ToLower(strings.TrimSpace(provider))
	resolution := Resolution{
		Surface:   providerSurface(provider),
		Requested: NewParams(fieldFloat64(requested, func(p *Params) *float64 { return p.Temperature }), fieldInt64(requested, func(p *Params) *int64 { return p.Seed }), fieldFloat64(requested, func(p *Params) *float64 { return p.TopP })),
	}
	switch provider {
	case "", "compatible", "openai":
		resolution.Effective = NewParams(fieldFloat64(requested, func(p *Params) *float64 { return p.Temperature }), fieldInt64(requested, func(p *Params) *int64 { return p.Seed }), fieldFloat64(requested, func(p *Params) *float64 { return p.TopP }))
	case "google", "mistral":
		resolution.Effective = NewParams(fieldFloat64(requested, func(p *Params) *float64 { return p.Temperature }), nil, nil)
		resolution.Unsupported = NewParams(nil, fieldInt64(requested, func(p *Params) *int64 { return p.Seed }), fieldFloat64(requested, func(p *Params) *float64 { return p.TopP }))
	default:
		resolution.Effective = NewParams(fieldFloat64(requested, func(p *Params) *float64 { return p.Temperature }), nil, nil)
		resolution.Unsupported = NewParams(nil, fieldInt64(requested, func(p *Params) *int64 { return p.Seed }), fieldFloat64(requested, func(p *Params) *float64 { return p.TopP }))
	}
	resolution.RequestedProfile = Profile(resolution.Requested)
	resolution.EffectiveProfile = Profile(resolution.Effective)
	return resolution
}

func ResolveExperimentName(rawName string, requestedProfile, effectiveProfile string) string {
	name := sanitizeToken(rawName)
	switch {
	case name != "":
		return name
	case strings.TrimSpace(requestedProfile) != "":
		return sanitizeToken(requestedProfile)
	case strings.TrimSpace(effectiveProfile) != "":
		return sanitizeToken(effectiveProfile)
	default:
		return "default"
	}
}

func BuildExperimentID(experimentName string, sequence int, at time.Time, requestedProfile string) string {
	if sequence <= 0 {
		sequence = 1
	}
	base := sanitizeToken(experimentName)
	profile := sanitizeToken(requestedProfile)
	if base == "" {
		base = "default"
	}
	if profile != "" && profile != base {
		base = base + "-" + profile
	}
	return fmt.Sprintf("%s-r%02d-%s", base, sequence, at.Local().Format("20060102"))
}

func SequenceFromRunID(runID string) int {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return 1
	}
	lower := strings.ToLower(runID)
	for _, token := range []string{"-r", "_r"} {
		idx := strings.LastIndex(lower, token)
		if idx < 0 || idx+len(token) >= len(lower) {
			continue
		}
		number := extractLeadingDigits(lower[idx+len(token):])
		if number == "" {
			continue
		}
		var seq int
		if _, err := fmt.Sscanf(number, "%d", &seq); err == nil && seq > 0 {
			return seq
		}
	}
	return 1
}

func providerSurface(provider string) string {
	switch provider {
	case "", "compatible":
		return "compatible_openai_chat_completions"
	case "openai":
		return "openai_chat_completions"
	case "google":
		return "google_generate_content"
	case "mistral":
		return "mistral_chat_completions"
	default:
		return sanitizeToken(provider)
	}
}

func extractLeadingDigits(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func sanitizeToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func formatFloat(value float64) string {
	formatted := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func fieldFloat64(source *Params, selector func(*Params) *float64) *float64 {
	if source == nil {
		return nil
	}
	return selector(source)
}

func fieldInt64(source *Params, selector func(*Params) *int64) *int64 {
	if source == nil {
		return nil
	}
	return selector(source)
}
