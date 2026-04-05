---
title: "Compositional Stage Semantics Cleanup"
status: active
owner: platform / research
updated_at: "2026-04-03"
source_basis:
  - "research/active/documents/compositional-lanes-interpretation.md"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-assumption-import-phase1.csv"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-depth-ladder-phase1.csv"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-branching-phase1.csv"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-mixed-family-reuse-phase1.csv"
---

# Compositional Stage Semantics Cleanup

## 1. Purpose

This document defines the cleanup layer required before stronger compositional
claims are made from the starter lanes and before the next canonical
mixed-family probe is implemented.

The problem is not the checker itself. The problem is that several starter
stage rows currently look like local prove tasks on paper while not being
honestly well-posed local prove tasks under the current direct-Hilbert surface.

## 2. Cleanup Rule

If a stage has all of the following:

- `expected_behavior=prove`
- empty `assumptions_json`
- empty `imported_lemmas_json`
- a goal that is not honestly justified as a theorem-level target for the
  intended local stage

then that stage must not be interpreted as an ordinary local prove task without
an explicit semantic reclassification.

Historical runs are not deleted because of this rule, but their interpretation
must be constrained accordingly.

## 3. Stage Type Classification

The canonical stage-type classification for compositional lanes is now:

### 3.1 `local_prove_stage`

The stage is a real local proof-construction task. The local assumptions and
imports given to the model are intended to be sufficient context for the proof
task as posed.

### 3.2 `trusted_resource_stage`

The stage is not a claim-bearing local prove task. It represents a verifier-
accepted or operator-provided resource that is available to later stages.

### 3.3 `gold_import_stage`

The stage measures whether the model can use correctly specified imported
lemmas.

### 3.4 `model_import_stage`

The stage measures whether the model can reuse verifier-confirmed model-
produced artifacts.

### 3.5 `negative_control_stage`

The stage measures compositional discipline and should not accidentally reward
blind concatenation or spurious derivability.

## 4. Current Starter-Lane Classification

### 4.1 Compositional Assumption Import

| Stage | Current Role | Current Interpretation | Cleanup Status |
| --- | --- | --- | --- |
| `s1` | local lemma 1 | semantically closer to `trusted_resource_stage` than a clean `local_prove_stage` in the current starter pack | not claim-bearing as local prove |
| `s2` | local lemma 2 | same issue as `s1` | not claim-bearing as local prove |
| `s3g` | gold final composition | `gold_import_stage` | acceptable |
| `s3m` | model final composition | `model_import_stage` | acceptable in intent, but blocked by earlier-stage collapse |
| `neg` | negative twin | `negative_control_stage` | acceptable |

### 4.2 Compositional Depth Ladder

| Stage | Current Role | Current Interpretation | Cleanup Status |
| --- | --- | --- | --- |
| `s1` | `seed_lemma` | semantically closer to `trusted_resource_stage` in the current starter pack | not claim-bearing as local prove |
| `s2` | `bridge_lemma` | semantically closer to `trusted_resource_stage` in the current starter pack | not claim-bearing as local prove |
| `s3` | `bridge_lemma` | semantically closer to `trusted_resource_stage` in the current starter pack | not claim-bearing as local prove |
| `d2g` / `d3g` | final gold composition | `gold_import_stage` | acceptable |
| `d2m` / `d3m` | final model composition | `model_import_stage` | acceptable in intent |
| `neg` | negative control | `negative_control_stage` | acceptable |

### 4.3 Compositional Branching

| Stage | Current Role | Current Interpretation | Cleanup Status |
| --- | --- | --- | --- |
| `s1` | `branch_left` | semantically invalid as a clean local prove task in the current starter pack | must be reclassified or rewritten |
| `s2` | `branch_right` | semantically invalid as a clean local prove task in the current starter pack | must be reclassified or rewritten |
| `s3` | `merge_rule` | semantically invalid as a clean local prove task in the current starter pack | must be reclassified or rewritten |
| `bg` | final gold composition | `gold_import_stage` | acceptable |
| `bm` | final model composition | `model_import_stage` | acceptable in intent |
| `neg` | negative control | `negative_control_stage` | acceptable |

### 4.4 Current Mixed-Family Reuse Starter Pack

| Stage | Current Role | Current Interpretation | Cleanup Status |
| --- | --- | --- | --- |
| `s1` | source family left | semantically closer to `trusted_resource_stage` than a clean local prove task | not canonical for local prove claims |
| `s2` | source family right | same issue as `s1` | not canonical for local prove claims |
| `fg` | final gold composition | `gold_import_stage` | acceptable in isolation |
| `fm` | final model composition | `model_import_stage` | methodologically premature for the next canonical probe |
| `neg` | negative control | `negative_control_stage` | acceptable |

## 5. Branching Cleanup Decision

The preferred cleanup for the current starter branching slice is:

1. stop reading `s1`, `s2`, and `s3` as claim-bearing local prove stages;
2. treat the current historical starter runs as exploratory diagnostic evidence
   with non-local resource semantics for those stages;
3. for the next canonical branching slice, either:
   - rewrite those stages into honest local prove tasks with proper local
     context, or
   - remove them from the claim-bearing slice and keep branching as a narrower
     `gold-final / model-final / negative` lane built on explicit trusted
     resources.

At the current evidence level, the second option is preferred. It is cleaner
and easier to interpret.

## 6. Immediate Policy

Effective immediately for repository interpretation:

- starter compositional lanes remain repository-backed and runnable;
- historical runs remain in history;
- but the seed and bridge stages identified above are no longer to be treated
  as clean local proof-construction evidence in canonical conclusions;
- `compositional-mixed-family-reuse` is provisional exploratory evidence, not
  the next canonical mixed-family claim surface.

## 7. Consequence for the Next Mixed-Family Step

Because of this cleanup, the next canonical mixed-family experiment must be:

- gold-first;
- depth-1;
- without `model` final stage;
- without self-produced cross-family transfer;
- explicitly framed as a question about using correct imported lemmas across
  proof-family labels, not as a question about model-produced artifact reuse.

That next step is defined in:

- `research/active/documents/mixed-family-gold-first-design.md`
