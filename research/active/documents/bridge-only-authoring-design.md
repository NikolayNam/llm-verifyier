---
title: "Bridge-Only Authoring — Experiment Design"
status: active
owner: platform / research
updated_at: "2026-04-04"
source_basis:
  - "research/active/documents/bridge-import-final-research-design.md"
  - "research/docs/bridge-import-final-research-phase1-audit.md"
  - "research/README.md"
---

# Bridge-Only Authoring — Experiment Design

## 1. Purpose

This split lane isolates the bridge-stage authoring question from the older
`bridge-import-final-research` baseline:

> can the model render locally derivable bridge lemmas such as `A -> B` and
> `B -> C` into verifier-acceptable Hilbert certificates when no imported-final
> composition is present?

The lane is not intended to measure final composition. It exists so that
bridge-stage formalization can be read on its own instead of being mixed with
`fg` / `fm`.

## 2. Why this split exists

The preserved baseline `bridge-import-final-research-phase1` turned out to be
unsafe to read as a clean bridge-authoring probe. Its `b1` / `b2` rows use
empty top-level assumptions together with non-tautological bridge goals such as
`A -> B` and `B -> C`. Under the repository Hilbert semantics those rows are
not fair standalone theorem targets.

This split lane corrects that by making each bridge lemma locally derivable via
small local assumptions. Example:

- assumption `B`, goal `A -> B`
- assumption `C`, goal `B -> C`

That still tests certificate authoring, but it removes the baseline wording
defect where the bridge-stage target itself was not derivable from the local
case payload.

## 3. Runnable surface

Repository-backed operator surface:

- `go -C platform-tooling run ./cmd/researchctl bridge-only-authoring plan`
- `go -C platform-tooling run ./cmd/researchctl bridge-only-authoring run`

Canonical files:

- baseline case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-phase1.csv`
- expanded phase2 pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-phase2.csv`
- canonical-variable normalized pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-only-authoring-canonical-phase1.csv`
- local profiles:
  - `research/config/compositional/bridge/only-authoring.yaml`
  - `research/config/compositional/bridge/only-authoring.skeleton.yaml`
  - `research/config/compositional/bridge/only-authoring.canonical.yaml`
  - `research/config/compositional/bridge/only-authoring.phase2.yaml`

## 4. Pack contents

The baseline pack contains only bridge-stage rows:

- `b1`
- `b2`

The expanded `phase2` pack keeps the same stage discipline, but it widens the
surface to `150` cases with balanced difficulty buckets (`50` easy, `50`
medium, `50` hard).

It intentionally omits:

- `fg`
- `fm`
- `neg`

The four logical families are preserved from the baseline interpretation
bundle:

- layered linear
- renamed linear
- distractor
- negated antecedent

There are no imported-lemma semantics in this lane.

## 5. Ablations

The lane has three prompt-surface variants:

- `hilbert-ai-verification-bridge-only-v1.0`
- `hilbert-ai-verification-bridge-only-skeleton-v1.0`
- `hilbert-ai-verification-bridge-only-canonical-vars-v1.0`

The canonical-vars ablation uses the normalized pack rather than hiding the
normalization inside prompt text. That keeps the intervention reproducible and
visible in the case inventory.

## 6. Intended read

This lane is useful only if it answers the narrow bridge-stage question:

- if the baseline bridge-only pack fails, the bottleneck is already visible at
  bridge-stage certificate authoring;
- if the skeleton or canonical-vars ablations lift performance materially, the
  bottleneck is likely a formalization surface problem rather than a pure logic
  wall;
- if the lane becomes strong while `gold-final-composition-only` remains weak,
  then the downstream closure stage remains the live bottleneck.
