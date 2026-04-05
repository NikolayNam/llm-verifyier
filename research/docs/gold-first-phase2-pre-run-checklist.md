# Gold-First Phase2 Pre-Run Checklist

- **Title:** **Gold-First Phase2 Pre-Run Checklist**
- **Status:** Active operator checklist
- **Date:** 2026-04-04
- **Scope:** `gold-first` / `phase2` / `hard` research launches

---

## 1. Purpose

This checklist is the short operator-facing companion to:

- [Gold-First Phase2 Hypothesis Readiness Matrix](./gold-first-phase2-hypothesis-readiness-matrix.md)

Use it immediately before starting a new benchmark run.

The question it answers is narrow:

> Is the next run claim-bearing, or is it only exploratory/support?

---

## 2. Claim-Bearing Runs

You MAY treat the next run as claim-bearing when all of the following are
true:

- the run is on one of these surfaces:
  - `compositional-mixed-family-gold-first`
  - `compositional-mixed-family-gold-first-phase2`
  - `compositional-mixed-family-gold-first-hard`
- the target question matches one of these claim-bearing hypotheses:
  - trusted gold mixed-family reuse
  - structured subgroup dependence
  - hard-transition stress on the same gold-first line
- the run does not reintroduce executable `trusted_resource_stage` rows or
  other semantically dirty starter-stage patterns
- the report language will stay narrow:
  - no claim about generalized self-produced reuse
  - no claim about full artifact-graph reasoning
  - no claim that starter compositional lanes are now repaired

If any of these conditions fail, do not label the run as claim-bearing.

---

## 3. Exploratory / Support Runs

Treat the next run as `exploratory/support only` when it belongs to one of
these categories:

- post-hoc reliability classification
- subgroup analysis
- prompt/protocol ablations
- reruns used only to confirm execution health
- starter compositional back-transfer
- any run on older starter compositional lanes:
  - `compositional-assumption-import`
  - `compositional-depth-ladder`
  - `compositional-branching`
  - `compositional-mixed-family-reuse`

These runs may still be useful, but they should not be reported as clean
theory evidence.

---

## 4. Deferred-but-Valid Next-Layer Runs

These are valid research hypotheses, but they should be treated as `годится
позже`, not as the immediate next core launch:

- semi-gold single-generated bridge
- semi-gold multi-import merge
- transformational reuse
- harder negative-control expansions
- `ND -> Hilbert` variant for the stabilized gold-first line

If one of these is launched early, report it as a next-layer experimental
extension, not as the current core line.

---

## 5. Five-Step Operator Check

Before running, ask:

1. Is this run on `gold-first`, `phase2`, or `hard`, rather than on an older
   starter lane?
2. Is the question still about trusted-gold reuse rather than semi-gold or
   fully autonomous compositional authoring?
3. Is the target of the run new knowledge, not just another confirmation of an
   already established easy-case result?
4. Will the result still be interpretable if it fails?
5. Can the resulting report be written without over-claiming beyond the narrow
   trust-boundary hypothesis?

If the answer is `no` to any of these, downgrade the launch to
`exploratory/support`.

---

## 6. Current Recommended Launch Order

As of `2026-04-04`, the preferred order is:

1. hard-transition micro-pack
2. semi-gold single-generated bridge
3. semi-gold multi-import merge
4. `ND -> Hilbert` variant on the stabilized gold-first line

Current support layers that should continue, but not as theory claims:

- `P0.2` post-hoc reliability classification
- `P0.3` subgroup analysis
- prompt/protocol robustness checks

Current line that should remain outside the claim-bearing core:

- back-transfer into starter lanes
