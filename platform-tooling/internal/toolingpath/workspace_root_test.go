package toolingpath

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExtractWorkspaceRootOverrideFlag(t *testing.T) {
	override, filtered, err := ExtractWorkspaceRootOverrideFlag([]string{
		"report", "meta",
		"--workspace-root", "../workspace",
		"--phase", "phase1",
	})
	if err != nil {
		t.Fatalf("ExtractWorkspaceRootOverrideFlag() error = %v", err)
	}
	if override != "../workspace" {
		t.Fatalf("override = %q, want %q", override, "../workspace")
	}
	want := []string{"report", "meta", "--phase", "phase1"}
	if !slices.Equal(filtered, want) {
		t.Fatalf("filtered args = %#v, want %#v", filtered, want)
	}
}

func TestExtractWorkspaceRootOverrideFlagEqualsSyntax(t *testing.T) {
	override, filtered, err := ExtractWorkspaceRootOverrideFlag([]string{
		"--workspace-root=../workspace",
		"report", "meta",
	})
	if err != nil {
		t.Fatalf("ExtractWorkspaceRootOverrideFlag() error = %v", err)
	}
	if override != "../workspace" {
		t.Fatalf("override = %q, want %q", override, "../workspace")
	}
	want := []string{"report", "meta"}
	if !slices.Equal(filtered, want) {
		t.Fatalf("filtered args = %#v, want %#v", filtered, want)
	}
}

func TestResolveWorkspaceRootFromCurrentLayout(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"platform-tooling", "research", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}
	start := filepath.Join(root, "platform-tooling", "cmd", "researchctl")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("MkdirAll(start) error = %v", err)
	}

	got, err := resolveWorkspaceRootFrom(start, "")
	if err != nil {
		t.Fatalf("resolveWorkspaceRootFrom() error = %v", err)
	}
	if got != root {
		t.Fatalf("resolveWorkspaceRootFrom() = %q, want %q", got, root)
	}
}

func TestResolveWorkspaceRootFromLegacyLayout(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"platform", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}
	start := filepath.Join(root, "platform", "tooling")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("MkdirAll(start) error = %v", err)
	}

	got, err := resolveWorkspaceRootFrom(start, "")
	if err != nil {
		t.Fatalf("resolveWorkspaceRootFrom() error = %v", err)
	}
	if got != root {
		t.Fatalf("resolveWorkspaceRootFrom() = %q, want %q", got, root)
	}
}

func TestResolveWorkspaceRootFromOverride(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"platform-tooling", "research", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}

	start := filepath.Join(root, "platform-tooling")
	got, err := resolveWorkspaceRootFrom(start, "..")
	if err != nil {
		t.Fatalf("resolveWorkspaceRootFrom() error = %v", err)
	}
	if got != root {
		t.Fatalf("resolveWorkspaceRootFrom() = %q, want %q", got, root)
	}
}
