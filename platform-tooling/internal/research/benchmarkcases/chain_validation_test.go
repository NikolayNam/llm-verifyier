package benchmarkcases

import (
	"strings"
	"testing"
)

func TestValidateCompositionalDependenciesRejectsMissingImportStage(t *testing.T) {
	cases := []ChainCase{
		{CaseID: "B1", ChainID: "C1", StageID: "b1", StageOrder: 1, ProvenanceMode: "gold"},
		{CaseID: "FM", ChainID: "C1", StageID: "fm", StageOrder: 2, ProvenanceMode: "model", ImportStageIDs: []string{"b2"}},
	}

	err := ValidateCompositionalDependencies(cases)
	if err == nil || !strings.Contains(err.Error(), "missing_import_stage") {
		t.Fatalf("ValidateCompositionalDependencies() error = %v, want missing_import_stage", err)
	}
}

func TestValidateCompositionalDependenciesRejectsFutureImportStage(t *testing.T) {
	cases := []ChainCase{
		{CaseID: "FM", ChainID: "C1", StageID: "fm", StageOrder: 2, ProvenanceMode: "model", ImportStageIDs: []string{"b2"}},
		{CaseID: "B2", ChainID: "C1", StageID: "b2", StageOrder: 3, ProvenanceMode: "gold"},
	}

	err := ValidateCompositionalDependencies(cases)
	if err == nil || !strings.Contains(err.Error(), "future_import_stage") {
		t.Fatalf("ValidateCompositionalDependencies() error = %v, want future_import_stage", err)
	}
}

func TestValidateCompositionalDependenciesRejectsCrossChainImport(t *testing.T) {
	cases := []ChainCase{
		{CaseID: "B1", ChainID: "C2", StageID: "b1", StageOrder: 1, ProvenanceMode: "gold"},
		{CaseID: "FM", ChainID: "C1", StageID: "fm", StageOrder: 2, ProvenanceMode: "model", ImportStageIDs: []string{"b1"}},
	}

	err := ValidateCompositionalDependencies(cases)
	if err == nil || !strings.Contains(err.Error(), "cross_chain_import") {
		t.Fatalf("ValidateCompositionalDependencies() error = %v, want cross_chain_import", err)
	}
}
