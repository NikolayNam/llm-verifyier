---
title: "Mixed-Family Gold-First Design"
status: active
owner: platform / research
updated_at: "2026-04-03"
source_basis:
  - "C:/Users/nokclock/Downloads/vivaldi_download/codex_compositional_lanes_cleanup_and_gold_first_mixed_family_spec.md"
  - "research/active/documents/compositional-lanes-interpretation.md"
  - "research/active/documents/compositional-stage-semantics-cleanup.md"
---

# Mixed-Family Gold-First Design

## 1. Purpose

This document defines the next canonical mixed-family probe after starter-lane
interpretation and stage-semantics cleanup.

The question under test is intentionally narrow:

> Under the current direct-Hilbert research surface, can the model
> compositionally use verifier-correct imported lemmas originating from a
> different proof family, while preserving negative compositional discipline and
> without relying on self-produced cross-family artifact reuse?

This is the first acceptable mixed-family question because it isolates
cross-family use of correct resources without expanding the runtime trust
boundary and without conflating the probe with self-produced artifact transfer.

## 2. Non-Goals

This design does not attempt to establish:

- self-produced mixed-family reuse;
- cross-chain artifact reuse;
- generalized artifact graph semantics;
- deeper mixed-family ladders;
- any runtime claim beyond research-only evidence.

## 3. Why This Replaces the Current Mixed-Family Starter as Canonical Next Step

The current `compositional-mixed-family-reuse` starter lane is still useful as
exploratory evidence, but it is not the correct next canonical claim surface
because it already mixes:

- semantically dirty source stages;
- `gold` and `model` final stages in the same first probe;
- heterogeneous family labeling with self-produced reuse pressure.

The gold-first probe removes those confounders.

## 4. Required Surface

The preferred implementation shape is a standalone research-only surface:

- `researchctl compositional-mixed-family-gold-first plan|run`
- config key:
  - `benchmarks.compositional_mixed_family_gold_first`
- dedicated local profile:
  - `research/config/compositional/mixed-family-gold-first.yaml`
- dedicated case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase1.csv`
- dedicated output namespaces:
  - `research/artifacts/result_research/compositional-mixed-family-gold-first/`
  - `research/result_research_report/compositional-mixed-family-gold-first/`

## 5. Minimal Stage Protocol

The first canonical probe should stay depth-1 and should not include a model
final stage.

Preferred long-term stage shape:

| Stage ID | Stage Type | Purpose |
| --- | --- | --- |
| `r1` | `trusted_resource_stage` | left resource lemma with source family label |
| `r2` | `trusted_resource_stage` | right resource lemma with source family label |
| `fg` | `gold_import_stage` | final composition using correct imported lemmas from a different family |
| `neg` | `negative_control_stage` | negative twin to test compositional discipline |

Alternative naming such as `res_left` / `res_right` is acceptable if the pack
is clearer that way. The important point is that these are resources, not
claim-bearing local prove stages.

### 5.1 Runnable v1 constraint

The current runnable repository surface intentionally does **not** execute
`r1` / `r2` rows yet.

Reason:

- the current benchmark runtime still treats CSV rows as ordinary executable
  benchmark cases;
- introducing executable `trusted_resource_stage` rows right now would
  recreate the same semantic compromise already documented in
  `compositional-stage-semantics-cleanup.md`.

Therefore the first runnable `compositional-mixed-family-gold-first` pack is
implemented as a narrower but cleaner slice:

| Stage ID | Stage Type | Purpose |
| --- | --- | --- |
| `fg` | `gold_import_stage` | claim-bearing final composition using verifier-correct imported lemmas from a different family |
| `neg` | `negative_control_stage` | negative twin for the same heterogeneous-family pairing |

Source-family provenance is carried in:

- `case_family`
- `comment`
- the imported gold lemmas themselves

This is methodologically cleaner than reintroducing semantically dirty source
rows before the runtime has explicit non-executed `trusted_resource_stage`
support.

## 6. Case-Pack Constraints

The starter gold-first pack should:

- remain depth-1;
- use only verifier-correct imported lemmas;
- keep source-family labels intentionally heterogeneous;
- avoid self-produced transfer;
- avoid theorem-synthesis pressure in the same first probe;
- include one negative twin per chain.
- include enough entailed chains to avoid over-reading a three-example anecdote;
- prefer roughly `15-30` entailed `fg` chains in the first serious pack,
  matched with one `neg` twin each.

Preferred source-family pairings are narrow and interpretable, for example:

- `assumption_import_source + direct_axiom_instance_source`
- `negative_refusal_source + assumption_import_source`
- `direct_axiom_instance_source + mixed_proof_source`

The exact pairings should optimize interpretability, not breadth.

## 7. Reporting Contract

The report for this probe should read more simply than the current
mixed-family starter:

- aggregate section;
- chain summary;
- separated starter-stage vs import-stage vs negative-control breakdowns;
- negative-control status;
- per-family-pair final-gold breakdown.

There should be no `final_model` section in the first canonical probe.

## 8. Success and Kill Read

Good signal:

- `fg` is stable across several runs;
- negative controls stay clean;
- heterogeneous source-family labeling does not itself induce collapse.

Bad signal:

- negative controls degrade;
- `fg` remains unstable even with correct imported resources;
- failures remain dominated by schema/request collapse rather than an
  interpretable frontier.

## 9. Current Status

This design is now materialized as a standalone runnable surface:

- `researchctl compositional-mixed-family-gold-first plan|run`
- `benchmarks.compositional_mixed_family_gold_first`
- `research/config/compositional/mixed-family-gold-first.yaml`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-gold-first-phase1.csv`

The current runnable pack is intentionally broader than the first three-chain
starter slice:

- `15` entailed `fg` chains
- `15` paired `neg` twins
- several heterogeneous family transitions rather than a single narrow pairing

This is enough to support a more serious gold-first read while still staying
inside the cleaned `fg/neg` claim boundary.

## 10. Phase 2 Annotated Expansion

The clean starter `v1` lane is intentionally kept intact.

The next annotated benchmark expansion now lives in a separate document and
surface:

- [Mixed-Family Gold-First Phase 2 Plan](./mixed-family-gold-first-phase2-plan.md)
- `researchctl compositional-mixed-family-gold-first-phase2`
- `research/config/compositional/mixed-family-gold-first-phase2.yaml`

Important guardrail:

- `v1` remains the clean starter claim surface;
- `phase2` is the annotated benchmark lane;
- `hard pack` and ablations remain separate follow-on layers.
- `semi-gold` is now materialized separately via:
  - `researchctl compositional-mixed-family-semi-gold-stable`
  - `researchctl compositional-mixed-family-semi-gold-frontier`
  - `research/active/documents/mixed-family-semi-gold-plan.md`

## 11. Hard-Pack Follow-On

The next adversarial sibling surface now exists separately:

- [Mixed-Family Gold-First Hard Pack Plan](./mixed-family-gold-first-hard-plan.md)
- `researchctl compositional-mixed-family-gold-first-hard`
- `research/config/compositional/mixed-family-gold-first-hard.yaml`

This hard pack should be interpreted as:

- a stricter follow-on to the clean `v1` and annotated `phase2` lanes;
- not a replacement of either of them;
- not evidence that exploratory compositional starter lanes are rehabilitated.

Important honesty constraint:

- this runnable v1 is intentionally narrower than the preferred long-term
  `r1/r2/fg/neg` protocol;
- it should be interpreted as the first clean gold-first probe, not as proof
  that the repository already supports first-class trusted resource stages.
- even if this lane reads well, it MUST NOT be treated as evidence that the
  exploratory `compositional-depth-ladder` or `compositional-branching`
  surfaces are already rehabilitated; those remain separate, weaker signals.
