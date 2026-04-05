package main

import (
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/toolingpath"
)

var workspaceRootOverride string

func applyWorkspaceRootOverride(args []string) ([]string, func(), error) {
	override, filteredArgs, err := toolingpath.ExtractWorkspaceRootOverrideFlag(args)
	if err != nil {
		return nil, nil, err
	}
	previous := workspaceRootOverride
	workspaceRootOverride = strings.TrimSpace(override)
	return filteredArgs, func() {
		workspaceRootOverride = previous
	}, nil
}

func findWorkspaceRoot() (string, error) {
	return toolingpath.ResolveWorkspaceRoot(workspaceRootOverride)
}
