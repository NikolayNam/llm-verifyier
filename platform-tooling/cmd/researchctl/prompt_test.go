package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPromptManifestCommandWritesVersionFileAndHash(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runPromptManifestCommand(nil, &stdout, &stderr); err != nil {
		t.Fatalf("runPromptManifestCommand() error = %v", err)
	}
	text := stdout.String()
	for _, fragment := range []string{
		"family",
		"version",
		"file",
		"sha256",
		"hilbert-ai-verification-benchmark-v1.3",
		"research/active/prompts/files/hilbert-ai-verification-benchmark-v1.3.system.txt",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("manifest output missing %q:\n%s", fragment, text)
		}
	}
}

func TestRunPromptLintCommandPassesForKnownCatalog(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runPromptLintCommand(nil, &stdout, &stderr); err != nil {
		t.Fatalf("runPromptLintCommand() error = %v", err)
	}
	text := stdout.String()
	if !strings.Contains(text, "prompt lint passed") {
		t.Fatalf("lint output missing success marker:\n%s", text)
	}
	if !strings.Contains(text, "hilbert-ai-verification-benchmark-nd-v1.1") {
		t.Fatalf("lint output missing ND prompt entry:\n%s", text)
	}
}

func TestSelectPromptCatalogEntriesRejectsUnknownFamily(t *testing.T) {
	if _, err := selectPromptCatalogEntries("unknown"); err == nil {
		t.Fatal("selectPromptCatalogEntries(unknown) expected error")
	}
}
