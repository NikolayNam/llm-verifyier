---
title: "Mixed-Family Gold-First Hard Pack Plan"
status: active
owner: platform / research
updated_at: "2026-04-03"
source_basis:
  - "research/active/documents/mixed-family-gold-first-phase2-plan.md"
  - "research/active/documents/mixed-family-gold-first-design.md"
---

# Mixed-Family Gold-First Hard Pack Plan

## 1. Purpose

This document defines the next adversarial follow-on after the annotated
`compositional-mixed-family-gold-first-phase2` lane.

The hard pack is intentionally a separate runnable surface so that:

- the clean `v1` lane remains the canonical starter claim surface;
- the annotated `phase2` lane remains the main richer benchmark lane;
- adversarial stress can expand without silently redefining either earlier
  surface.

## 2. Runnable Surface

The repository-backed operator surface for this hard pack is:

- `researchctl compositional-mixed-family-gold-first-hard plan|run`
- config key:
  - `benchmarks.compositional_mixed_family_gold_first_hard`
- local profile:
  - `research/config/compositional/mixed-family-gold-first-hard.yaml`
- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard.csv`
- output namespaces:
  - `research/artifacts/result_research/compositional-mixed-family-gold-first-hard/`
  - `research/result_research_report/compositional-mixed-family-gold-first-hard/`

## 3. Pack Contract

The current hard pack contains:

- `20` entailed `fg` chains
- `20` paired `neg` twins

The pack is partitioned into four hard transition groups:

- `GF-MFH-A` — one-import long bridge
- `GF-MFH-B` — one-import negation bridge
- `GF-MFH-C` — two-import merge
- `GF-MFH-D` — adversarial gold reuse

There is also a smaller operator-focused micro slice on the same surface:

- `research/config/compositional/mixed-family-gold-first-hard.micro.yaml`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-micro.csv`

There is now also a phase2-derived hard-transition micro slice on the same
surface:

- `research/config/compositional/mixed-family-gold-first-hard-phase2.yaml`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-hard-phase2.csv`
- `research/artifacts/analysis/gold_first_phase2_hard_pack_report.md`

The micro slice is intentionally built only from the hardest transition groups:

- `GF-MFH-C`
- `GF-MFH-D`

and narrows the cohort to the strongest currently interesting models instead of
repeating a broad confirmation run.

The newer `hard-phase2` slice is intentionally different from the older
`hard.micro` slice:

- it is derived directly from the `reasoning-valid` `gold-first phase2`
  subgroup analysis rather than from the earlier hard-pack taxonomy;
- it keeps only `GF-MF-D` and `GF-MF-E`, the two heaviest transition groups
  inside the phase2 analysis;
- it reintroduces `gpt-oss:20b-cloud` as a weaker contrast baseline while
  keeping the three stronger models in the same operator profile.

Methodological guardrails remain unchanged:

- only `fg` and `neg` rows are claim-bearing;
- there are no executable resource rows;
- there is no `model-final` stage;
- this lane is an adversarial stress follow-on, not evidence for generalized
  self-produced reuse.

## 4. Annotation Layer

The hard pack reuses the phase2 optional annotation layer:

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

These annotations remain report/artifact-first metadata:

- row-level result provenance
- `chain_result_<run-id>.csv`
- grouped markdown report sections

They do not introduce a new `research_db` schema in this batch.

## 5. Interpretation

This hard pack should be read as:

- an adversarial stress test for clean gold-first mixed-family reuse;
- a probe for whether the `fg` signal survives longer bridges, lower symbol
  overlap, and harder negative twins.

It should **not** be read as:

- rehabilitation of exploratory starter compositional lanes;
- proof that self-produced mixed-family reuse is solved;
- proof that deeper branching or artifact-graph reuse is solved.

## 6. Deferred Work

The following still remain deferred beyond this hard-pack surface:

- dedicated arity ablations
- presentation / surface-style ablations
- any `semi-gold` bridge
- executable trusted resource stages
- self-produced mixed-family reuse
