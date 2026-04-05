# Compositional Depth Ladder — Experiment Design

Status: active  
Scope: standalone direct-Hilbert compositional experiment for controlled
depth-2/3 reuse chains  
Audience: researchers, platform engineers, benchmark operators  
Last reviewed: 2026-04-03

## 1. Purpose

This experiment is the explicit follow-on to the starter
`compositional-assumption-import` slice.

Its job is narrower and more demanding:

- move from depth-1 reuse to controlled depth-2/3 reuse chains;
- test whether trusted intermediate artifacts can be reused compositionally;
- separate protocol failure from genuinely deeper compositional reasoning
  failure.

It does **not** answer the broader `ND -> Hilbert` question. It remains a
direct-Hilbert authoring experiment.

## 2. Operator Surface

This experiment now has its own standalone `researchctl` surface:

```powershell
go -C platform-tooling run ./cmd/researchctl compositional-depth-ladder plan --local-config research/config/compositional/depth-ladder.yaml --jobs 2
go -C platform-tooling run ./cmd/researchctl compositional-depth-ladder run --local-config research/config/compositional/depth-ladder.yaml --jobs 2 --best-effort
```

Canonical config key:

- `benchmarks.compositional_depth_ladder`

Dedicated local profile:

- `research/config/compositional/depth-ladder.yaml`

## 3. Current Artifact Surface

Starter inputs:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-depth-ladder-phase1.csv`

Shared summary/report outputs:

- `research/artifacts/result_research/compositional-depth-ladder/compositional-depth-ladder_<run-id>.csv`
- `research/result_research_report/compositional-depth-ladder/compositional-depth-ladder_<run-id>.md`

Per-run sidecars:

- `research/artifacts/<project>/result/chain_result_<run-id>.csv`
- `research/artifacts/<project>/result/imported_certificates/<run-id>/...`
- `research/artifacts/<project>/result/imported_certificate_catalog_<run-id>.csv`

`db sync` now classifies the shared summary and chain sidecar for this lane
under artifact group:

- `compositional_depth_ladder`

## 4. Generic Chain Stage Protocol

The depth-ladder lane uses the generic chain metadata fields that were added to
the benchmark engine:

- `chain_protocol`
- `stage_order`
- `stage_role`
- `trusted_reuse`
- `import_stage_ids_json`

Current active protocol id:

- `depth-ladder-v1`

These fields are interpreted as follows:

- `stage_order` defines the deterministic chain order for summary and failure
  reporting.
- `stage_role` documents the semantic role of the stage, for example
  `seed_lemma`, `intermediate_model`, `final_model`, `negative_control`.
- `trusted_reuse=true` means a passing stage may be materialized as a trusted
  imported certificate object for later `provenance_mode=model` stages.
- `import_stage_ids_json` is the authoritative dependency list for model-side
  artifact reuse in non-starter protocols.

## 5. Current Starter Ladder

The current repository-backed pack intentionally stays narrow:

- one depth-2 chain;
- two depth-3 chains;
- explicit intermediate model reuse via `d2m`;
- explicit gold/model final comparison;
- negative controls at each chain depth.

This is enough to test:

- whether local lemma success survives one more compositional hop;
- whether trusted intermediate model artifacts remain reusable;
- whether negative controls stay conservative after deeper reuse.

## 6. Reporting Contract

The standalone markdown report now renders:

- the standard chain summary table;
- a generic `Stage protocol breakdown` section derived from
  `stage_sequence_json`, `stage_roles_json`, and `stage_passes_json`;
- the chain failure breakdown table.

For non-starter protocols, the generic stage breakdown is the authoritative
stage-level interpretation layer. Legacy fields such as `stage1_pass` or
`stage3_model_pass` remain compatibility projections only.

## 7. Current Honesty Boundary

This implementation is stronger than the starter slice, but it is still not a
fully general proof-artifact import system.

What is implemented:

- first-class imported certificate objects;
- trusted intermediate model reuse;
- explicit depth-2/3 chain dependencies;
- generic chain stage ordering and stage-role reporting.

What is still deferred:

- broader branching composition beyond the current bounded follow-on lane;
- mixed-family reuse across unrelated chains;
- generalized artifact graph semantics beyond the current linear ladder;
- a new authoring surface such as `ND -> Hilbert`.

## 8. Success / Kill Read

The depth-ladder lane should be interpreted conservatively.

Good signal:

- trusted prerequisite stages pass reliably;
- intermediate model stages remain reusable;
- final model stages do not collapse relative to gold controls;
- negative controls stay clean.

Bad signal:

- intermediate model artifacts pass locally but fail when reused one level
  deeper;
- failure mix stays dominated by schema/request collapse instead of honest
  frontier limits;
- negative controls degrade while final model stages still look superficially
  strong.
