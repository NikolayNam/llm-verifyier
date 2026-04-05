package promptcatalog

import "testing"

func TestCatalogLoadsUniquePromptEntries(t *testing.T) {
	entries, err := Catalog()
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("len(entries) = %d, want at least 2", len(entries))
	}
	seenVersions := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Version == "" {
			t.Fatal("entry.Version is empty")
		}
		if entry.RelativePath == "" {
			t.Fatalf("entry.RelativePath for %q is empty", entry.Version)
		}
		if len(entry.SHA256) != 64 {
			t.Fatalf("entry.SHA256 for %q = %q, want 64-char hex", entry.Version, entry.SHA256)
		}
		if entry.ByteCount == 0 {
			t.Fatalf("entry.ByteCount for %q = 0", entry.Version)
		}
		if _, ok := seenVersions[entry.Version]; ok {
			t.Fatalf("duplicate version %q", entry.Version)
		}
		seenVersions[entry.Version] = struct{}{}
	}
}

func TestLookupResolvesKnownPromptVersion(t *testing.T) {
	entry, err := Lookup("hilbert-ai-verification-benchmark-v1.3")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if entry.FileName != "hilbert-ai-verification-benchmark-v1.3.system.txt" {
		t.Fatalf("entry.FileName = %q", entry.FileName)
	}
}
