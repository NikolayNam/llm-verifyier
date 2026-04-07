package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/artifactkey"
	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func compactArtifactOutputPath(loaded researchconfig.Loaded, path, prefix, key string) string {
	path = filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if path == "" {
		return ""
	}
	if strings.TrimSpace(prefix) == "" || strings.TrimSpace(key) == "" {
		return filepath.ToSlash(path)
	}
	artifactRoot := loaded.ResolvePath(loaded.Config.Paths.ArtifactRoot)
	if !pathWithinRoot(path, artifactRoot) {
		return filepath.ToSlash(path)
	}
	return artifactkey.CompactFile(path, prefix, key)
}

func pathWithinRoot(path, root string) bool {
	path = filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	root = filepath.Clean(filepath.FromSlash(strings.TrimSpace(root)))
	if path == "" || root == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return false
	}
	return true
}
