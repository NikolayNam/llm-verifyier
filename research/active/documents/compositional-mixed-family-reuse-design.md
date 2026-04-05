# Compositional Mixed-Family Reuse — Experiment Design

Status: provisional exploratory  
Scope: standalone direct-Hilbert compositional experiment for bounded reuse
across heterogeneous case families within one trusted chain  
Audience: researchers, platform engineers, benchmark operators  
Last reviewed: 2026-04-03

## 1. Purpose

Current status note:

- this lane remains repository-backed and runnable;
- however, after the starter-lane interpretation and stage-semantics cleanup,
  it should be treated as exploratory evidence, not as the next canonical
  mixed-family claim surface;
- the canonical next-step target is now
  `research/active/documents/mixed-family-gold-first-design.md`;
- that canonical gold-first target is now materialized separately as:
  - `researchctl compositional-mixed-family-gold-first`
  - `research/config/compositional/mixed-family-gold-first.yaml`.

This experiment is the follow-on after:

- `compositional-assumption-import`
- `compositional-depth-ladder`
- `compositional-branching`

Its purpose is narrower than a full artifact graph:

- test whether trusted imported certificate objects remain reusable when the
  contributing stages are intentionally labeled as different source families;
- separate "same-family compositional reuse works" from "heterogeneous
  family reuse still works";
- do this without yet claiming cross-chain or graph-generalized reuse.

It is still a direct-Hilbert authoring experiment. It does **not** settle:

- cross-chain artifact reuse;
- generalized artifact graph semantics;
- `ND -> Hilbert`.

## 2. Operator Surface

This experiment still has its own standalone `researchctl` surface:

```powershell
go -C platform-tooling run ./cmd/researchctl compositional-mixed-family-reuse plan --local-config research/config/compositional/mixed-family-reuse.yaml --jobs 2
go -C platform-tooling run ./cmd/researchctl compositional-mixed-family-reuse run --local-config research/config/compositional/mixed-family-reuse.yaml --jobs 2 --best-effort
```

Canonical config key:

- `benchmarks.compositional_mixed_family_reuse`

Dedicated local profile:

- `research/config/compositional/mixed-family-reuse.yaml`

## 3. Current Artifact Surface

Starter inputs:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-reuse-phase1.csv`

Shared summary/report outputs:

- `research/artifacts/result_research/compositional-mixed-family-reuse/compositional-mixed-family-reuse_<run-id>.csv`
- `research/result_research_report/compositional-mixed-family-reuse/compositional-mixed-family-reuse_<run-id>.md`

Per-run sidecars:

- `research/artifacts/<project>/result/chain_result_<run-id>.csv`
- `research/artifacts/<project>/result/imported_certificates/<run-id>/...`
- `research/artifacts/<project>/result/imported_certificate_catalog_<run-id>.csv`

`db sync` classifies the shared summary and chain sidecar for this lane under
artifact group:

- `compositional_mixed_family_reuse`

## 4. Current Bounded Protocol

The lane reuses the generic chain metadata surface:

- `chain_protocol`
- `stage_order`
- `stage_role`
- `trusted_reuse`
- `import_stage_ids_json`

Current active protocol id:

- `mixed-family-v1`

The key experimental difference from the earlier compositional lanes is:

- source stages inside the same chain intentionally use different
  `case_family` labels;
- final stages then reuse those heterogeneous source artifacts through the
  same trusted local import layer.

Current honesty boundary:

- imports still resolve within one bounded `chain_id`;
- there is no new cross-chain graph resolver here.
- source stages `s1` and `s2` should not be over-read as clean claim-bearing
  local prove stages in canonical conclusions.

## 5. Current Starter Pack

The repository-backed starter pack is intentionally small:

- `3` chains
- `5` stages per chain
- heterogeneous source-family labels
- one `final_gold`
- one `final_model`
- one `negative_control`

Current source-family combinations:

- `assumption_import_source + direct_axiom_instance_source`
- `mixed_proof_source + theorem_synthesis_source`
- `theorem_synthesis_source + assumption_import_source`

This is enough to test exploratory behavior around:

- whether trusted reuse survives heterogeneous source labeling;
- whether `final_model` stays close to `final_gold`;
- whether negative controls remain conservative under cross-family imports.

It is not the preferred first canonical mixed-family probe anymore, because it
already mixes `gold` and `model` final stages before the gold-first question
has been isolated cleanly.

## 6. Reporting Contract

This lane uses the same generic chain report surface as the depth-ladder and
branching experiments:

- standard chain summary table
- generic `Stage protocol breakdown`
- chain failure breakdown

Because chain stages may now have heterogeneous `case_family` labels, the
chain-level `case_family` column should be read as a compatibility projection:

- a single family if all stages agree;
- `mixed_family_reuse` when multiple source families participate.

Stage-level result rows remain the authoritative record for per-stage family
labels.

## 7. Success / Kill Read

Good signal:

- trusted source stages pass reliably;
- `final_model` remains near `final_gold`;
- heterogeneous source-family labeling does not itself induce collapse;
- negative controls remain clean.

Bad signal:

- source stages pass but heterogeneous-family merge fails sharply;
- `gold` reuse works while `model` reuse collapses;
- failures remain dominated by schema/request instability instead of
  interpretable composition limits.

## 8. Deferred Work

This lane still does **not** provide:

- reuse across unrelated chains;
- generalized artifact graph semantics;
- mixed-family fan-out/fan-in across a non-linear global reuse graph;
- an alternative authoring surface such as `ND -> Hilbert`.
