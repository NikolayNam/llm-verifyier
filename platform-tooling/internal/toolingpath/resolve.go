package toolingpath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	repoRootOnce sync.Once
	repoRootDir  string
)

func RepoRoot() string {
	repoRootOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return
		}
		// resolve.go lives in platform-tooling/internal/toolingpath, so three parents up is repo root.
		repoRootDir = filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	})
	return repoRootDir
}

func RepoPath(parts ...string) string {
	return joinUnderRoot(RepoRoot(), parts...)
}

func PlatformPath(parts ...string) string {
	return joinUnderRoot(RepoRoot(), append([]string{"platform"}, parts...)...)
}

func PlatformToolingPath(parts ...string) string {
	return joinUnderRoot(RepoRoot(), append([]string{"platform-tooling"}, parts...)...)
}

func FirstExisting(candidates ...string) string {
	for _, candidate := range candidates {
		if exists(candidate) {
			return candidate
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func joinUnderRoot(root string, parts ...string) string {
	if strings.TrimSpace(root) == "" {
		return filepath.Join(parts...)
	}
	candidate := append([]string{root}, parts...)
	return filepath.Join(candidate...)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
