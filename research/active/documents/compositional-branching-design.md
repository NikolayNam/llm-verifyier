# Compositional Branching — Experiment Design

Status: active  
Scope: standalone direct-Hilbert compositional experiment for explicit
branching fan-in reuse chains  
Audience: researchers, platform engineers, benchmark operators  
Last reviewed: 2026-04-03

## 1. Purpose

This experiment is the explicit follow-on to
`compositional-depth-ladder`.

Its job is different from the linear depth ladder:

- move from linear reuse chains to explicit multi-parent fan-in;
- test whether trusted local artifacts stay reusable when the final stage
  depends on more than one prerequisite branch;
- separate linear-depth degradation from true branching-composition failure.

It still does **not** answer the broader `ND -> Hilbert` question. It remains
a direct-Hilbert authoring experiment.

## 2. Operator Surface

This experiment has its own standalone `researchctl` surface:

```powershell
go -C platform-tooling run ./cmd/researchctl compositional-branching plan --local-config research/config/compositional/branching.yaml --jobs 2
go -C platform-tooling run ./cmd/researchctl compositional-branching run --local-config research/config/compositional/branching.yaml --jobs 2 --best-effort
```

Canonical config key:

- `benchmarks.compositional_branching`

Dedicated local profile:

- `research/config/compositional/branching.yaml`

## 3. Current Artifact Surface

Starter inputs:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-branching-phase1.csv`

Shared summary/report outputs:

- `research/artifacts/result_research/compositional-branching/compositional-branching_<run-id>.csv`
- `research/result_research_report/compositional-branching/compositional-branching_<run-id>.md`

Per-run sidecars:

- `research/artifacts/<project>/result/chain_result_<run-id>.csv`
- `research/artifacts/<project>/result/imported_certificates/<run-id>/...`
- `research/artifacts/<project>/result/imported_certificate_catalog_<run-id>.csv`

`db sync` classifies the shared summary and chain sidecar for this lane under
artifact group:

- `compositional_branching`

## 4. Generic Branching Protocol

The branching lane reuses the generic chain metadata fields already present in
the benchmark engine:

- `chain_protocol`
- `stage_order`
- `stage_role`
- `trusted_reuse`
- `import_stage_ids_json`

Current active protocol id:

- `branching-v1`

The important semantic difference from the depth ladder is:

- `import_stage_ids_json` may name more than one independent prerequisite
  branch for a final composition stage;
- the final stage is interpreted as a fan-in merge, not only as the next step
  in a linear ladder.

## 5. Current Starter Branching Pack

The repository-backed starter pack intentionally stays narrow:

- `3` chains
- `6` stages per chain
- explicit `branch_left` and `branch_right` seed lemmas
- one explicit `merge_rule`
- one `final_gold`
- one `final_model`
- one `negative_control`

Current starter case families:

- `fan_in_merge`
- `fan_in_merge_renamed`
- `fan_in_with_distractor`

This is enough to test:

- whether multiple trusted prerequisites remain reusable in one final stage;
- whether `model` fan-in is weaker than `gold` fan-in under the same stage
  dependencies;
- whether negative controls remain conservative under explicit branching.

## 6. Reporting Contract

The standalone markdown report for this lane uses the same generic chain
summary/reporting surface as the depth ladder:

- standard chain summary table
- generic `Stage protocol breakdown`
- chain failure breakdown

For this lane the generic stage breakdown is the authoritative read, because
legacy starter fields like `stage1_pass` and `stage3_model_pass` are only
compatibility projections.

## 7. Current Honesty Boundary

What is implemented now:

- explicit multi-parent stage dependencies through `import_stage_ids_json`
- first-class imported certificate objects
- trusted local artifact reuse across branching final stages
- standalone summary/report namespaces

What is still deferred:

- mixed-family reuse across unrelated chains
- cross-chain artifact reuse
- generalized artifact graph semantics beyond the current bounded branching
  protocol
- a new authoring surface such as `ND -> Hilbert`

## 8. Success / Kill Read

Good signal:

- both prerequisite branches pass reliably;
- `final_model` remains close to `final_gold`;
- negative controls stay clean;
- failures move away from request/schema collapse toward interpretable frontier
  limits.

Bad signal:

- branch prerequisites pass locally but fail when merged;
- `gold` fan-in works while `model` fan-in collapses sharply;
- negative controls degrade while final composition still appears superficially
  strong.
