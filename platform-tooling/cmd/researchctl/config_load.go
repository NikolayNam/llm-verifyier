package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	researchconfig "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/config"
)

func loadResearchConfig(fs *flag.FlagSet, workspaceRoot, configPath, localConfigPath string) (researchconfig.Loaded, error) {
	loaded, err := researchconfig.Load(workspaceRoot, configPath, localConfigPath)
	if err != nil {
		return researchconfig.Loaded{}, err
	}
	if isFlagExplicitlySet(fs, "local-config") && strings.TrimSpace(localConfigPath) != "" && !loaded.LocalConfigFound {
		return researchconfig.Loaded{}, fmt.Errorf("local config %q was explicitly requested but does not exist", filepath.ToSlash(loaded.LocalConfigPath))
	}
	return loaded, nil
}

func isFlagExplicitlySet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
