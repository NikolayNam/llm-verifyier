package lean4worker

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureProjectScaffold(projectDir string) error {
	if projectDir == "" {
		return fmt.Errorf("lean4 project dir is required")
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("create lean4 project dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, DefaultGeneratedDir), 0o755); err != nil {
		return fmt.Errorf("create lean4 generated dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, DefaultProjectName), 0o755); err != nil {
		return fmt.Errorf("create lean4 library dir: %w", err)
	}

	files := map[string]string{
		filepath.Join(projectDir, "lean-toolchain"):                        defaultLeanToolchainFile(),
		filepath.Join(projectDir, "lakefile.lean"):                         defaultLakefile(),
		filepath.Join(projectDir, DefaultProjectName+".lean"):              defaultLibraryRoot(),
		filepath.Join(projectDir, DefaultProjectName, "Basic.lean"):        defaultLibraryBasic(),
		filepath.Join(projectDir, DefaultProjectName, "KernelParity.lean"): defaultKernelParityModule(),
	}
	for path, content := range files {
		if err := writeFileIfMissing(path, content); err != nil {
			return err
		}
	}
	return nil
}

func writeFileIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %q: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write scaffold file %q: %w", path, err)
	}
	return nil
}

func defaultLeanToolchainFile() string {
	return DefaultToolchain + "\n"
}

func defaultLakefile() string {
	return `import Lake
open Lake DSL

package "CollabSphereLean" where

require mathlib from git
  "https://github.com/leanprover-community/mathlib4.git" @ "` + DefaultMathlibRev + `"

@[default_target]
lean_lib CollabSphereLean
`
}

func defaultLibraryRoot() string {
	return `import CollabSphereLean.Basic
`
}

func defaultLibraryBasic() string {
	return `namespace CollabSphereLean

/- Base namespace for local Lean4 research helpers and generated checks. -/

end CollabSphereLean
`
}
