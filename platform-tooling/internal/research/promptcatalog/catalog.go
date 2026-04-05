package promptcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

type Entry struct {
	Family       string
	Version      string
	FileName     string
	RelativePath string
	AbsolutePath string
	SystemPrompt string
	SHA256       string
	ByteCount    int
}

type descriptor struct {
	Family   string
	Version  string
	FileName string
}

var descriptors = []descriptor{
	{Family: "hilbert", Version: "hilbert-ai-verification-benchmark-v1", FileName: "hilbert-ai-verification-benchmark-v1.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-benchmark-v1.1", FileName: "hilbert-ai-verification-benchmark-v1.1.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-benchmark-v1.2", FileName: "hilbert-ai-verification-benchmark-v1.2.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-benchmark-v1.3", FileName: "hilbert-ai-verification-benchmark-v1.3.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-bridge-only-v1.0", FileName: "hilbert-ai-verification-bridge-only-v1.0.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-bridge-only-skeleton-v1.0", FileName: "hilbert-ai-verification-bridge-only-skeleton-v1.0.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-bridge-only-canonical-vars-v1.0", FileName: "hilbert-ai-verification-bridge-only-canonical-vars-v1.0.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-gold-final-only-v1.0", FileName: "hilbert-ai-verification-gold-final-only-v1.0.system.txt"},
	{Family: "hilbert", Version: "hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0", FileName: "hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0.system.txt"},
	{Family: "nd", Version: "hilbert-ai-verification-benchmark-nd-v1", FileName: "hilbert-ai-verification-benchmark-nd-v1.system.txt"},
	{Family: "nd", Version: "hilbert-ai-verification-benchmark-nd-v1.1", FileName: "hilbert-ai-verification-benchmark-nd-v1.1.system.txt"},
}

func Catalog() ([]Entry, error) {
	seenVersions := make(map[string]string, len(descriptors))
	seenFiles := make(map[string]string, len(descriptors))
	entries := make([]Entry, 0, len(descriptors))
	for _, desc := range descriptors {
		version := strings.TrimSpace(desc.Version)
		family := strings.TrimSpace(desc.Family)
		fileName := strings.TrimSpace(desc.FileName)
		if family == "" {
			return nil, fmt.Errorf("prompt catalog contains empty family for version %q", version)
		}
		if version == "" {
			return nil, fmt.Errorf("prompt catalog contains empty version for file %q", fileName)
		}
		if fileName == "" {
			return nil, fmt.Errorf("prompt catalog contains empty file name for version %q", version)
		}
		if prior, ok := seenVersions[version]; ok {
			return nil, fmt.Errorf("duplicate prompt version %q (%s, %s)", version, prior, fileName)
		}
		if prior, ok := seenFiles[fileName]; ok {
			return nil, fmt.Errorf("duplicate prompt file %q (%s, %s)", fileName, prior, version)
		}
		seenVersions[version] = fileName
		seenFiles[fileName] = version
		entry, err := loadEntry(desc)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(entry.FileName, ".system.txt") {
			return nil, fmt.Errorf("prompt file %q for version %q must end with .system.txt", entry.FileName, entry.Version)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Family != entries[j].Family {
			return entries[i].Family < entries[j].Family
		}
		return entries[i].Version < entries[j].Version
	})
	return entries, nil
}

func Lookup(version string) (Entry, error) {
	version = strings.TrimSpace(version)
	entries, err := Catalog()
	if err != nil {
		return Entry{}, err
	}
	for _, entry := range entries {
		if entry.Version == version {
			return entry, nil
		}
	}
	return Entry{}, fmt.Errorf("unsupported prompt version %q", version)
}

func loadEntry(desc descriptor) (Entry, error) {
	absolutePath := toolingpath.RepoPath("research", "active", "prompts", "files", desc.FileName)
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return Entry{}, fmt.Errorf("read prompt file %q: %w", absolutePath, err)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		return Entry{}, fmt.Errorf("prompt file %q is empty", absolutePath)
	}
	sum := sha256.Sum256([]byte(text))
	return Entry{
		Family:       desc.Family,
		Version:      desc.Version,
		FileName:     desc.FileName,
		RelativePath: filepath.ToSlash(filepath.Join("research", "active", "prompts", "files", desc.FileName)),
		AbsolutePath: filepath.ToSlash(absolutePath),
		SystemPrompt: text,
		SHA256:       hex.EncodeToString(sum[:]),
		ByteCount:    len([]byte(text)),
	}, nil
}
