---
title: "Gold-Final Composition Only — Experiment Design"
status: active
owner: platform / research
updated_at: "2026-04-04"
source_basis:
  - "research/active/documents/bridge-import-final-research-design.md"
  - "research/README.md"
---

# Gold-Final Composition Only — Experiment Design

## 1. Purpose

This split lane isolates final composition over fixed trusted imported lemmas:

> if the imported bridge lemmas are already correct and static, can the model
> close the final goal cleanly?

The lane intentionally removes bridge-stage authoring from the causal graph.

## 2. Runnable surface

Repository-backed operator surface:

- `go -C platform-tooling run ./cmd/researchctl gold-final-composition-only plan`
- `go -C platform-tooling run ./cmd/researchctl gold-final-composition-only run`

Canonical files:

- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/gold-final-composition-only-phase1.csv`
- local profiles:
  - `research/config/compositional/gold/final-composition-only.yaml`
  - `research/config/compositional/gold/final-composition-only.explicit-import-refs.yaml`
  - `research/config/compositional/gold/final-composition-only.phase2.yaml`
- expanded phase2 pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/gold-final-composition-only-phase2.csv`

## 3. Pack contents

The pack contains only:

- `fg`
- `neg`

The expanded `phase2` pack preserves that same split, but it widens the lane
to `150` cases with balanced difficulty buckets and paired positive/negative
rows (`75` `fg`, `75` `neg`).

It intentionally omits:

- `b1`
- `b2`
- `fm`

Imported lemmas remain static gold-backed resources through
`imported_lemmas_json`. The pack therefore measures final closure under fixed
trusted resources rather than self-produced import reuse.

## 4. Ablation

The baseline prompt surface is:

- `hilbert-ai-verification-gold-final-only-v1.0`

The explicit import-handling ablation is:

- `hilbert-ai-verification-gold-final-only-explicit-import-refs-v1.0`

That ablation exists to make the imported-lemma contract explicit:

- which lemmas are available
- in which order they should be copied
- that they must be treated as trusted reusable resources
- that they must not be re-proved

## 5. Intended read

This lane is useful only if it answers the narrow final-closure question:

- if this lane stays strong, then the older baseline `fg` signal was likely
  real and bridge-stage failure is the better explanation for mixed-lane
  collapse;
- if this lane is weak even with static gold imports, then the repository
  still has a genuine final-composition problem independent of bridge-stage
  authoring;
- if the explicit-import prompt lifts performance, the bottleneck is at least
  partially about imported-resource formalization rather than only pure proof
  search.
