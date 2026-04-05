package main

import (
	"sync"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/promptcatalog"
)

var (
	benchmarkPromptProfiles     map[string]benchmarkPromptProfile
	benchmarkPromptProfilesErr  error
	benchmarkPromptProfilesOnce sync.Once

	ndBenchmarkPromptProfiles     map[string]ndBenchmarkPromptProfile
	ndBenchmarkPromptProfilesErr  error
	ndBenchmarkPromptProfilesOnce sync.Once
)

func loadBenchmarkPromptProfiles() (map[string]benchmarkPromptProfile, error) {
	benchmarkPromptProfilesOnce.Do(func() {
		benchmarkPromptProfiles, benchmarkPromptProfilesErr = loadPromptProfiles([]string{
			benchmarkPromptVersionV1,
			benchmarkPromptVersionV11,
			benchmarkPromptVersionV12,
			benchmarkPromptVersionV13,
			benchmarkPromptVersionBridgeOnlyV1,
			benchmarkPromptVersionBridgeOnlySkeletonV1,
			benchmarkPromptVersionBridgeOnlyCanonicalVarsV1,
			benchmarkPromptVersionGoldFinalOnlyV1,
			benchmarkPromptVersionGoldFinalOnlyExplicitImportRefsV1,
		}, func(version, prompt string) benchmarkPromptProfile {
			return benchmarkPromptProfile{
				Version:      version,
				SystemPrompt: prompt,
			}
		})
	})
	return benchmarkPromptProfiles, benchmarkPromptProfilesErr
}

func loadNDBenchmarkPromptProfiles() (map[string]ndBenchmarkPromptProfile, error) {
	ndBenchmarkPromptProfilesOnce.Do(func() {
		ndBenchmarkPromptProfiles, ndBenchmarkPromptProfilesErr = loadPromptProfiles([]string{
			ndBenchmarkPromptVersionV1,
			ndBenchmarkPromptVersionV11,
		}, func(version, prompt string) ndBenchmarkPromptProfile {
			return ndBenchmarkPromptProfile{
				Version:      version,
				SystemPrompt: prompt,
			}
		})
	})
	return ndBenchmarkPromptProfiles, ndBenchmarkPromptProfilesErr
}

func loadPromptProfiles[T any](versions []string, build func(version, prompt string) T) (map[string]T, error) {
	out := make(map[string]T, len(versions))
	for _, version := range versions {
		entry, err := promptcatalog.Lookup(version)
		if err != nil {
			return nil, err
		}
		out[version] = build(version, entry.SystemPrompt)
	}
	return out, nil
}
