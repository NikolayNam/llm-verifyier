package llmprovider

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Catalog struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   CatalogMeta    `yaml:"metadata"`
	Models     []CatalogModel `yaml:"models"`
}

type CatalogMeta struct {
	Name        string `yaml:"name"`
	Source      string `yaml:"source,omitempty"`
	RefreshedAt string `yaml:"refreshedAt,omitempty"`
}

type CatalogModel struct {
	Ref    string `yaml:"ref"`
	Active bool   `yaml:"active"`
}

type ComposeFile struct {
	Name     string                      `yaml:"name,omitempty"`
	Services map[string]ComposeService   `yaml:"services,omitempty"`
	Models   map[string]ComposeModelRef  `yaml:"models,omitempty"`
	Networks map[string]ComposeNetwork   `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolumeRef `yaml:"volumes,omitempty"`
}

type ComposeService struct {
	Profiles    []string          `yaml:"profiles,omitempty"`
	Image       string            `yaml:"image,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Models      []string          `yaml:"models,omitempty"`
}

type ComposeModelRef struct {
	Model string `yaml:"model"`
}

type ComposeNetwork struct {
	Name     string `yaml:"name,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

type ComposeVolumeRef struct {
	Name string `yaml:"name,omitempty"`
}

func PathsFromComposeDir(composeDir string) (catalogPath, templatePath, activatedPath string) {
	composeDir = strings.TrimSpace(composeDir)
	return filepath.Join(composeDir, "llm_provider.yaml"),
		filepath.Join(composeDir, "llm_provider_template.yaml"),
		filepath.Join(composeDir, "llm_provider_activated.yaml")
}

func DefaultPaths() (catalogPath, templatePath, activatedPath string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("get working directory: %w", err)
	}
	candidates := []string{
		strings.TrimSpace(os.Getenv("COLLABSPHERE_LLM_PROVIDER_COMPOSE_DIR")),
		"/app/deploy/compose/llm",
		"/app/deploy/compose",
		cwd,
		filepath.Dir(cwd),
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		composeDirs := []string{
			filepath.Join(candidate, "deploy", "compose", "llm"),
			filepath.Join(candidate, "deploy", "compose"),
			candidate,
		}
		for _, composeDir := range composeDirs {
			info, statErr := os.Stat(composeDir)
			if statErr != nil || !info.IsDir() {
				continue
			}
			if hasProviderManifestFiles(composeDir) {
				catalogPath, templatePath, activatedPath := PathsFromComposeDir(composeDir)
				return catalogPath, templatePath, activatedPath, nil
			}
		}
	}
	return "", "", "", fmt.Errorf("llm provider compose directory not found from %s", cwd)
}

func hasProviderManifestFiles(composeDir string) bool {
	catalogPath, templatePath, _ := PathsFromComposeDir(composeDir)
	if _, err := os.Stat(catalogPath); err != nil {
		return false
	}
	if _, err := os.Stat(templatePath); err != nil {
		return false
	}
	return true
}

func LoadCatalog(path string) (*Catalog, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog %s: %w", path, err)
	}
	var catalog Catalog
	if err := yaml.Unmarshal(payload, &catalog); err != nil {
		return nil, fmt.Errorf("decode catalog %s: %w", path, err)
	}
	if len(catalog.Models) == 0 {
		return nil, fmt.Errorf("catalog %s has no models", path)
	}
	return &catalog, nil
}

func LoadTemplate(path string) (*ComposeFile, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}
	var file ComposeFile
	if err := yaml.Unmarshal(payload, &file); err != nil {
		return nil, fmt.Errorf("decode template %s: %w", path, err)
	}
	if len(file.Services) == 0 {
		return nil, fmt.Errorf("template %s has no services", path)
	}
	return &file, nil
}

func SaveCatalog(path string, catalog *Catalog) error {
	payload, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("encode catalog %s: %w", path, err)
	}
	return os.WriteFile(path, payload, 0o644)
}

func ActivateRefs(catalog *Catalog, refs []string) error {
	if catalog == nil {
		return fmt.Errorf("catalog is nil")
	}
	normalized := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		normalized[ref] = struct{}{}
	}
	if len(normalized) == 0 {
		return fmt.Errorf("no refs were provided")
	}

	found := make(map[string]bool, len(normalized))
	for i := range catalog.Models {
		_, active := normalized[catalog.Models[i].Ref]
		catalog.Models[i].Active = active
		if active {
			found[catalog.Models[i].Ref] = true
		}
	}
	missing := make([]string, 0)
	for ref := range normalized {
		if !found[ref] {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("unknown refs: %s", strings.Join(missing, ", "))
	}
	return nil
}

func ActiveRefs(catalog *Catalog) []string {
	if catalog == nil {
		return nil
	}
	refs := make([]string, 0)
	for _, model := range catalog.Models {
		if model.Active {
			refs = append(refs, model.Ref)
		}
	}
	sort.Strings(refs)
	return refs
}

func RenderActivated(template *ComposeFile, catalog *Catalog) (*ComposeFile, error) {
	if template == nil {
		return nil, fmt.Errorf("template is nil")
	}
	active := make([]CatalogModel, 0)
	for _, model := range catalog.Models {
		if model.Active {
			active = append(active, model)
		}
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("catalog does not contain active models")
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Ref < active[j].Ref })

	rendered := *template
	rendered.Services = make(map[string]ComposeService, len(template.Services))
	for name, service := range template.Services {
		next := service
		next.Models = make([]string, 0, len(active))
		for _, model := range active {
			next.Models = append(next.Models, modelAlias(model.Ref))
		}
		rendered.Services[name] = next
	}
	rendered.Models = make(map[string]ComposeModelRef, len(active))
	for _, model := range active {
		rendered.Models[modelAlias(model.Ref)] = ComposeModelRef{Model: model.Ref}
	}
	return &rendered, nil
}

func SaveActivated(path string, file *ComposeFile) error {
	if file == nil {
		return fmt.Errorf("compose file is nil")
	}
	payload, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode activated compose %s: %w", path, err)
	}
	var out bytes.Buffer
	out.WriteString("# generated by `go -C platform-tooling run ./cmd/provider-manifest render`\n")
	out.Write(payload)
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func modelAlias(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.ToLower(ref)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range ref {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	value := strings.Trim(b.String(), "_")
	if value == "" {
		return "llm"
	}
	return value
}
