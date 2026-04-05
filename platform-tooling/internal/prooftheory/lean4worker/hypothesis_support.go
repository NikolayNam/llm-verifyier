package lean4worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HypothesisSupportArtifacts struct {
	GenerationConfigPath string
	DefaultPayloadPath   string
	CuratedSeedPath      string
	ProposedSeedPath     string
}

func HypothesisSupportArtifactPaths(root string) HypothesisSupportArtifacts {
	theoremsDir := filepath.Join(root, DefaultTheoremsDir)
	payloadDir := filepath.Join(theoremsDir, "payloads")
	return HypothesisSupportArtifacts{
		GenerationConfigPath: filepath.Join(theoremsDir, generationConfigFileName),
		DefaultPayloadPath:   filepath.Join(payloadDir, defaultPayloadTemplateName),
		CuratedSeedPath:      filepath.Join(theoremsDir, curatedSeedFileName),
		ProposedSeedPath:     filepath.Join(theoremsDir, proposedSeedFileName),
	}
}

func DefaultHypothesisSourcePath(root, mode string) string {
	paths := HypothesisSupportArtifactPaths(root)
	switch strings.TrimSpace(mode) {
	case HypothesisGenerationModeCurated:
		return paths.CuratedSeedPath
	case HypothesisGenerationModeProposed:
		return paths.ProposedSeedPath
	default:
		return ""
	}
}

func FinalizeHypothesisCases(cases []HypothesisCase, mode string, ts time.Time) (string, []HypothesisCase) {
	packID := buildGeneratedTheoremPackID(mode, ts)
	finalized := make([]HypothesisCase, len(cases))
	for idx, hypothesis := range cases {
		finalized[idx] = hypothesis
		finalized[idx].TheoremPackID = packID
		if strings.TrimSpace(finalized[idx].TheoremID) == "" {
			finalized[idx].TheoremID = strings.TrimSpace(finalized[idx].CaseID)
		}
	}
	return packID, finalized
}

func WriteHypothesisSupportArtifacts(root string, config HypothesisGenerationConfig) (string, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", fmt.Errorf("lean4 artifact root is required")
	}
	paths := HypothesisSupportArtifactPaths(root)
	legacyCasesDir := filepath.Join(root, LegacyCasesDir)
	legacyPayloadDir := filepath.Join(legacyCasesDir, "payloads")

	if err := os.MkdirAll(filepath.Dir(paths.DefaultPayloadPath), 0o755); err != nil {
		return "", "", fmt.Errorf("create lean4 payload dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, DefaultTheoremsDir, HistoricalTheoremsDir), 0o755); err != nil {
		return "", "", fmt.Errorf("create lean4 theorem history dir: %w", err)
	}
	if err := os.MkdirAll(legacyPayloadDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create lean4 legacy payload dir: %w", err)
	}

	if err := writeHypothesisSeedCSV(paths.CuratedSeedPath, DefaultCuratedHypothesisSeeds()); err != nil {
		return "", "", err
	}
	if err := writeHypothesisSeedCSV(filepath.Join(legacyCasesDir, curatedSeedFileName), DefaultCuratedHypothesisSeeds()); err != nil {
		return "", "", err
	}
	if err := writeHypothesisSeedCSV(paths.ProposedSeedPath, DefaultModelProposedSeeds()); err != nil {
		return "", "", err
	}
	if err := writeHypothesisSeedCSV(filepath.Join(legacyCasesDir, proposedSeedFileName), DefaultModelProposedSeeds()); err != nil {
		return "", "", err
	}

	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal generation config: %w", err)
	}
	if err := os.WriteFile(paths.GenerationConfigPath, configBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write generation config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(legacyCasesDir, generationConfigFileName), configBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write legacy generation config: %w", err)
	}

	payloadBytes, err := json.MarshalIndent(ModulePayload{
		Imports: []string{},
		Helpers: []string{},
		Proof:   "",
	}, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal default lean payload: %w", err)
	}
	if err := os.WriteFile(paths.DefaultPayloadPath, payloadBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write default lean payload: %w", err)
	}
	if err := os.WriteFile(filepath.Join(legacyPayloadDir, defaultPayloadTemplateName), payloadBytes, 0o644); err != nil {
		return "", "", fmt.Errorf("write legacy default lean payload: %w", err)
	}
	return paths.GenerationConfigPath, paths.DefaultPayloadPath, nil
}
