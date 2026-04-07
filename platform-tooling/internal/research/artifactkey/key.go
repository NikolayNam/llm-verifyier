package artifactkey

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func BenchmarkJob(phase, runID, jobKey string) string {
	return short("bj", phase, runID, jobKey)
}

func GeneratedPack(runID, packKey string) string {
	return short("gp", runID, packKey)
}

func Export(runID, exportKey string) string {
	return short("ex", runID, exportKey)
}

func CompactFile(path, prefix, key string) string {
	path = strings.TrimSpace(path)
	prefix = strings.TrimSpace(prefix)
	key = strings.TrimSpace(key)
	if path == "" || prefix == "" || key == "" {
		return filepath.ToSlash(path)
	}
	ext := filepath.Ext(path)
	base := prefix + "_" + key + ext
	return filepath.ToSlash(filepath.Join(filepath.Dir(path), base))
}

func short(prefix string, parts ...string) string {
	prefix = strings.TrimSpace(prefix)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	token := hex.EncodeToString(sum[:])[:12]
	if prefix == "" {
		return token
	}
	return prefix + "_" + strings.ToLower(token)
}
