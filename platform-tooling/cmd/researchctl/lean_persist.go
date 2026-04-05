package main

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	lean4worker "github.com/NikolayNam/collabsphere/platform-tooling/internal/prooftheory/lean4worker"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/researchdb"
)

type persistedLeanGeneration struct {
	Result       researchdb.PersistHypothesisGenerationResult
	Cases        []lean4worker.HypothesisCase
	TheoremPackID string
	HistoricalPath string
}

func persistLeanGenerationFromArtifacts(ctx context.Context, dbPath, workspaceRoot, projectFolder, artifactRoot string, genConfig lean4worker.HypothesisGenerationConfig, theoremsPath, configOutPath, payloadPath string, createdAt time.Time) (persistedLeanGeneration, error) {
	cases, err := lean4worker.LoadHypothesisCasesCSV(theoremsPath)
	if err != nil {
		return persistedLeanGeneration{}, err
	}
	packID := ""
	for _, hypothesis := range cases {
		if value := strings.TrimSpace(hypothesis.TheoremPackID); value != "" {
			packID = value
			break
		}
	}
	if packID == "" {
		return persistedLeanGeneration{}, errNoTheoremPackID(theoremsPath)
	}
	historicalPath := lean4worker.HistoricalTheoremSnapshotPath(filepath.Join(artifactRoot, lean4worker.DefaultTheoremsDir), packID)

	db, err := researchdb.Open(ctx, dbPath)
	if err != nil {
		return persistedLeanGeneration{}, err
	}
	defer db.Close()

	persisted, err := researchdb.PersistHypothesisGeneration(ctx, db, researchdb.PersistHypothesisGenerationInput{
		WorkspaceRoot:         workspaceRoot,
		ProjectFolder:         projectFolder,
		ArtifactRoot:          artifactRoot,
		SourceFilePath:        strings.TrimSpace(genConfig.SourceFile),
		HistoricalTheoremPath: historicalPath,
		Config:                genConfig,
		Cases:                 cases,
		GenerationConfigPath:  configOutPath,
		DefaultPayloadPath:    payloadPath,
		CreatedAt:             createdAt,
	})
	if err != nil {
		return persistedLeanGeneration{}, err
	}
	if err := researchdb.CompleteGenerationJob(ctx, db, persisted.GenerationJobID, time.Now().UTC()); err != nil {
		return persistedLeanGeneration{}, err
	}
	return persistedLeanGeneration{
		Result:         persisted,
		Cases:          cases,
		TheoremPackID:  packID,
		HistoricalPath: historicalPath,
	}, nil
}

func errNoTheoremPackID(path string) error {
	return &noTheoremPackIDError{path: path}
}

type noTheoremPackIDError struct {
	path string
}

func (e *noTheoremPackIDError) Error() string {
	return "no theorem_pack_id found in " + e.path
}
