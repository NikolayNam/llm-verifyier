package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/promptcatalog"
)

func runPromptSurface(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("prompt requires: manifest or lint")
	}
	switch args[0] {
	case "manifest":
		return runPromptManifestCommand(args[1:], stdout, stderr)
	case "lint":
		return runPromptLintCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown prompt subcommand %q", args[0])
	}
}

func runPromptManifestCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("prompt-manifest", flag.ContinueOnError)
	fs.SetOutput(stderr)

	familiesFlag := fs.String("family", "", "Optional comma-separated prompt families: hilbert,nd")
	format := fs.String("format", "table", "Output format: table|json|csv")
	output := fs.String("output", "", "Optional output file path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	entries, err := selectPromptCatalogEntries(*familiesFlag)
	if err != nil {
		return err
	}
	rendered, err := renderPromptManifest(entries, strings.TrimSpace(*format))
	if err != nil {
		return err
	}
	if strings.TrimSpace(*output) != "" {
		return writePromptOutput(*output, rendered)
	}
	_, _ = io.WriteString(stdout, rendered)
	return nil
}

func runPromptLintCommand(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("prompt-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)

	familiesFlag := fs.String("family", "", "Optional comma-separated prompt families: hilbert,nd")
	format := fs.String("format", "table", "Output format for the catalog listing: table|json|csv")
	output := fs.String("output", "", "Optional output file path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if extra := fs.Args(); len(extra) != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(extra, ", "))
	}

	entries, err := selectPromptCatalogEntries(*familiesFlag)
	if err != nil {
		return err
	}
	if err := lintPromptEntries(entries); err != nil {
		return err
	}
	rendered, err := renderPromptManifest(entries, strings.TrimSpace(*format))
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("prompt lint passed\nentries: %d\n\n", len(entries)))
	b.WriteString(rendered)
	if strings.TrimSpace(*output) != "" {
		return writePromptOutput(*output, b.String())
	}
	_, _ = io.WriteString(stdout, b.String())
	return nil
}

func selectPromptCatalogEntries(rawFamilies string) ([]promptcatalog.Entry, error) {
	entries, err := promptcatalog.Catalog()
	if err != nil {
		return nil, err
	}
	families := splitCSV(rawFamilies)
	if len(families) == 0 {
		return entries, nil
	}
	allowed := []string{"hilbert", "nd"}
	for _, family := range families {
		if !slices.Contains(allowed, family) {
			return nil, fmt.Errorf("unsupported prompt family %q", family)
		}
	}
	filtered := make([]promptcatalog.Entry, 0, len(entries))
	for _, entry := range entries {
		if slices.Contains(families, entry.Family) {
			filtered = append(filtered, entry)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("prompt selection resolved no entries")
	}
	return filtered, nil
}

func lintPromptEntries(entries []promptcatalog.Entry) error {
	if len(entries) == 0 {
		return fmt.Errorf("prompt lint requires at least one entry")
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.FileName, ".system.txt") {
			return fmt.Errorf("prompt %q uses non-system prompt file %q", entry.Version, entry.FileName)
		}
		if strings.TrimSuffix(entry.FileName, ".system.txt") != entry.Version {
			return fmt.Errorf("prompt %q file %q does not match version name", entry.Version, entry.FileName)
		}
		if len(entry.SHA256) != 64 {
			return fmt.Errorf("prompt %q hash has invalid length", entry.Version)
		}
		if strings.TrimSpace(entry.SystemPrompt) == "" {
			return fmt.Errorf("prompt %q is empty", entry.Version)
		}
	}
	return nil
}

func renderPromptManifest(entries []promptcatalog.Entry, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "table":
		return renderPromptManifestTable(entries), nil
	case "json":
		raw, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal prompt manifest: %w", err)
		}
		return string(raw) + "\n", nil
	case "csv":
		return renderPromptManifestCSV(entries)
	default:
		return "", fmt.Errorf("unsupported prompt manifest format %q", format)
	}
}

func renderPromptManifestTable(entries []promptcatalog.Entry) string {
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "family\tversion\tfile\tsha256")
	for _, entry := range entries {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", entry.Family, entry.Version, entry.RelativePath, entry.SHA256)
	}
	_ = tw.Flush()
	return b.String()
}

func renderPromptManifestCSV(entries []promptcatalog.Entry) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"family", "version", "file", "sha256"}); err != nil {
		return "", fmt.Errorf("write prompt manifest header: %w", err)
	}
	for _, entry := range entries {
		if err := w.Write([]string{entry.Family, entry.Version, entry.RelativePath, entry.SHA256}); err != nil {
			return "", fmt.Errorf("write prompt manifest row %q: %w", entry.Version, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("flush prompt manifest csv: %w", err)
	}
	return b.String(), nil
}

func writePromptOutput(rawPath, content string) error {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	path := rawPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(workspaceRoot, filepath.FromSlash(rawPath))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create prompt output dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write prompt output %q: %w", path, err)
	}
	return nil
}
