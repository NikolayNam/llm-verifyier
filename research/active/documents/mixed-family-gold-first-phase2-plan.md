---
title: "Mixed-Family Gold-First Phase 2 Plan"
status: active
owner: platform / research
updated_at: "2026-04-03"
source_basis:
  - "C:/Users/nokclock/Downloads/vivaldi_download/codex_gold_first_mixed_family_reuse_plan(1).md"
  - "research/active/documents/mixed-family-gold-first-design.md"
---

# Mixed-Family Gold-First Phase 2 Plan

## 1. Purpose

This document defines the first annotated benchmark expansion after the clean
`v1` `gold-first mixed-family reuse` starter lane.

`phase2` is deliberately a separate runnable surface so that:

- the existing `v1` lane remains a clean starter claim surface;
- the wider annotated pack can carry richer metadata without silently changing
  the interpretation of `v1`;
- harder follow-ons such as `hard pack`, ablations, and `semi-gold` remain
  deferred until this annotated lane reads cleanly.

## 2. Runnable Surface

The repository-backed operator surface for this phase is:

- `researchctl compositional-mixed-family-gold-first-phase2 plan|run`
- config key:
  - `benchmarks.compositional_mixed_family_gold_first_phase2`
- local profiles:
  - `research/config/compositional/mixed-family-gold-first-phase2.yaml`
  - `research/config/compositional/mixed-family-gold-first-phase2.gpt-oss-120b-cloud-rerun.yaml`
- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase2.csv`
- output namespaces:
  - `research/artifacts/result_research/compositional-mixed-family-gold-first-phase2/`
  - `research/result_research_report/compositional-mixed-family-gold-first-phase2/`

## 3. Pack Contract

The current annotated pack contains:

- `50` entailed `fg` chains
- `50` paired `neg` twins

The pack is partitioned into five transition groups:

- `GF-MF-A` — one-import linear reuse
- `GF-MF-B` — one-import reuse with local symbol remapping bridge
- `GF-MF-C` — two-import merge
- `GF-MF-D` — bridge depth `>= 1`
- `GF-MF-E` — hardest / adversarial benchmark slice

Methodological guardrails for this batch:

- only `fg` and `neg` rows are claim-bearing;
- there are no executable `r1/r2` benchmark rows;
- there is no `model-final` stage;
- `hard pack`, ablations, and `semi-gold` are out of scope for this batch.

## 4. Annotation Layer

Unlike the starter `v1` lane, the phase2 pack carries explicit optional
annotations that should be interpreted as report/artifact-level metadata:

- `source_family`
- `target_family`
- `transition_group`
- `import_arity`
- `reuse_shape`
- `bridge_depth`
- `symbol_overlap`
- `lemma_surface_style`
- `negative_twin_hardness`
- `notes`

These annotations are propagated into:

- per-row result CSVs
- `chain_result_<run-id>.csv`
- the markdown report

They are intentionally **not** materialized as a new `research_db` schema in
this first batch.

## 5. Reporting Contract

Phase2 reports should keep `chain-level` and `stage-level` evidence primary.

The report now renders grouped sections for:

- transition group
- import arity
- reuse shape
- bridge depth
- symbol overlap
- negative twin hardness
- `fg` failure-type mix
- hardest `fg` failures

The aggregate table remains secondary context.

## 6. Cohort and Execution Defaults

The default cohort for the main phase2 benchmark lane is:

- `gpt-oss:120b-cloud`
- `glm-5:cloud`
- `deepseek-v3.1:671b-cloud`
- `gpt-oss:20b`

Default controls:

- `repeats: 3`
- deterministic sampling: `temperature = 0`, `seed = 42`, `top_p = 1`
- `request_timeout_abort_threshold` and `interrupt_policy` remain active
  operator controls

The targeted rerun profile narrows to:

- `gpt-oss:120b-cloud`
- `repeats: 1`

## 7. Deferred Work

The following are intentionally deferred beyond this phase2 batch:

- separate `hard pack` as its own standalone surface:
  - [Mixed-Family Gold-First Hard Pack Plan](./mixed-family-gold-first-hard-plan.md)
- presentation and arity ablations
- any `semi-gold` bridge
- self-produced mixed-family reuse
- trusted executable resource stages

Those remain valid follow-ons only if this annotated lane stays clean and
interpretable.
