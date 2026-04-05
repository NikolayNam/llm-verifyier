package toolingpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	WorkspaceRootFlagName = "workspace-root"
	WorkspaceRootEnvKey   = "COLLABSPHERE_WORKSPACE_ROOT"
)

var workspaceRootEnvKeys = []string{
	WorkspaceRootEnvKey,
	"RESEARCH_WORKSPACE_ROOT",
}

var workspaceRootLayouts = [][]string{
	{"platform", "docs"},
	{"platform-tooling", "research", "docs"},
}

func WorkspaceRootEnvHelp() string {
	return strings.Join(workspaceRootEnvKeys, ", ")
}

func WorkspaceRootOverrideFromEnv() (string, string) {
	for _, key := range workspaceRootEnvKeys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return key, value
		}
	}
	return "", ""
}

func ExtractWorkspaceRootOverrideFlag(args []string) (string, []string, error) {
	filtered := make([]string, 0, len(args))
	override := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--"+WorkspaceRootFlagName:
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return "", nil, fmt.Errorf("--%s requires a non-empty path value", WorkspaceRootFlagName)
			}
			value := strings.TrimSpace(args[i+1])
			if override != "" && override != value {
				return "", nil, fmt.Errorf("multiple --%s values provided: %q and %q", WorkspaceRootFlagName, override, value)
			}
			override = value
			i++
		case strings.HasPrefix(arg, "--"+WorkspaceRootFlagName+"="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--"+WorkspaceRootFlagName+"="))
			if value == "" {
				return "", nil, fmt.Errorf("--%s requires a non-empty path value", WorkspaceRootFlagName)
			}
			if override != "" && override != value {
				return "", nil, fmt.Errorf("multiple --%s values provided: %q and %q", WorkspaceRootFlagName, override, value)
			}
			override = value
		default:
			filtered = append(filtered, arg)
		}
	}
	return override, filtered, nil
}

func ResolveWorkspaceRoot(override string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("detect workspace root: %w", err)
	}
	return resolveWorkspaceRootFrom(wd, override)
}

func resolveWorkspaceRootFrom(startDir, override string) (string, error) {
	startDir, err := normalizeRootCandidate("", startDir)
	if err != nil {
		return "", fmt.Errorf("detect workspace root: %w", err)
	}
	if root, err := resolveWorkspaceRootOverride(startDir, strings.TrimSpace(override), "--"+WorkspaceRootFlagName); root != "" || err != nil {
		return root, err
	}
	if envKey, envValue := WorkspaceRootOverrideFromEnv(); envValue != "" {
		return resolveWorkspaceRootOverride(startDir, envValue, envKey)
	}
	for _, candidate := range workspaceRootCandidates(startDir) {
		if isWorkspaceRoot(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not locate workspace root from %q (expected %s; override with --%s or %s)",
		startDir,
		workspaceRootLayoutHelp(),
		WorkspaceRootFlagName,
		WorkspaceRootEnvHelp(),
	)
}

func resolveWorkspaceRootOverride(baseDir, override, source string) (string, error) {
	if strings.TrimSpace(override) == "" {
		return "", nil
	}
	candidate, err := normalizeRootCandidate(baseDir, override)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root from %s: %w", source, err)
	}
	if !isWorkspaceRoot(candidate) {
		return "", fmt.Errorf("%s=%q is not a supported workspace root (expected %s)", source, override, workspaceRootLayoutHelp())
	}
	return candidate, nil
}

func workspaceRootCandidates(startDir string) []string {
	candidates := make([]string, 0, 8)
	current := filepath.Clean(startDir)
	for range 8 {
		candidates = append(candidates, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return candidates
}

func normalizeRootCandidate(baseDir, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	trimmed = filepath.FromSlash(trimmed)
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}
	if strings.TrimSpace(baseDir) == "" {
		return filepath.Abs(trimmed)
	}
	return filepath.Clean(filepath.Join(baseDir, trimmed)), nil
}

func isWorkspaceRoot(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	for _, layout := range workspaceRootLayouts {
		matched := true
		for _, entry := range layout {
			if !dirExists(filepath.Join(path, entry)) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func workspaceRootLayoutHelp() string {
	descriptions := make([]string, 0, len(workspaceRootLayouts))
	for _, layout := range workspaceRootLayouts {
		descriptions = append(descriptions, strings.Join(layout, "+"))
	}
	return strings.Join(descriptions, " or ")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
