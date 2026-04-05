---
title: "Compositional Lanes Interpretation"
status: active
owner: platform / research
updated_at: "2026-04-03"
source_basis:
  - "research/result_research_report/compositional/compositional-assumption-import_20260403T082656+0300.md"
  - "research/result_research_report/compositional-depth-ladder/compositional-depth-ladder_depth-ladder-20260403--best-effort.md"
  - "research/result_research_report/compositional-branching/compositional-branching_20260403T133842+0300.md"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-assumption-import-phase1.csv"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-depth-ladder-phase1.csv"
  - "research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-branching-phase1.csv"
---

# Compositional Lanes Interpretation

## 1. Purpose

This document is the interpretation layer that was missing between the starter
compositional reports and any stronger follow-on claim such as mixed-family
reuse.

Its purpose is narrower than protocol design:

- summarize what the current starter compositional lanes actually show;
- distinguish aggregate reading from authoritative stage-level reading;
- identify which apparent failures are genuine compositional failures and which
  are stage-semantics artifacts;
- define the current claim boundary honestly before any next-step lane is used
  for canonical research claims.

This document does not change formal checker semantics and does not retroactively
delete historical runs. It only constrains interpretation.

## 2. Lane Inventory

| Lane | Current Status | Question Under Test | Current Claim Boundary |
| --- | --- | --- | --- |
| `compositional_assumption_import_phase1` | Starter diagnostic lane with usable negative control and gold-final signal, but semantically dirty seed stages | Can the model reuse already available lemmas compositionally for a simple final entailed target? | Claim-bearing only for negative discipline and bounded gold-final signal; not clean evidence for local lemma proving or self-produced reuse |
| `compositional_depth_ladder_phase1` | Starter diagnostic lane with deeper chain pressure and strong authoritative stage-level asymmetry | Does compositional reliability survive controlled depth-2/3 reuse pressure? | Claim-bearing only through the authoritative stage protocol breakdown; legacy chain-summary compatibility columns must not be over-read |
| `compositional_branching_phase1` | Starter diagnostic lane with the highest stage-semantic ambiguity | Can fan-in composition survive multiple prerequisite branches under direct-Hilbert authoring? | Claim-bearing only for negative discipline and bounded gold-final read; branch-building stages are not clean local prove evidence |

## 3. Stage Contract Per Lane

### 3.1 Compositional Assumption Import

Current stage ids and intended roles:

| Stage ID | Intended Role | Current Semantics | Authoritative Summary Layer |
| --- | --- | --- | --- |
| `s1` | local lemma 1 | currently not a clean local prove task when goal is theorem-like only by label, with empty assumptions/imports | chain summary plus failure breakdown |
| `s2` | local lemma 2 | same issue as `s1` | chain summary plus failure breakdown |
| `s3g` | gold import stage | coherent as a gold import diagnostic | chain summary |
| `s3m` | model import stage | coherent as a model import diagnostic, but currently blocked by earlier-stage collapse | chain summary |
| `neg` | negative control | coherent | chain summary |

For this lane the `Chain Summary` table is materially useful, but it still must
be read together with the failure breakdown. The repeated `s1 false_refusal`
pattern is the dominant blocker.

### 3.2 Compositional Depth Ladder

Current stage ids and intended roles:

| Stage ID | Stage Role | Current Semantics | Authoritative Summary Layer |
| --- | --- | --- | --- |
| `s1` | `seed_lemma` | semantically dubious as a local prove task in the current starter pack | stage protocol breakdown |
| `s2` | `bridge_lemma` | same concern as `s1` | stage protocol breakdown |
| `s3` | `bridge_lemma` | same concern as `s1`/`s2` in depth-3 chains | stage protocol breakdown |
| `d2g` / `d3g` | `final_gold` | coherent as gold-supported composition | stage protocol breakdown |
| `d2m` / `d3m` | `final_model` | coherent in intent, but current evidence is only meaningful through the authoritative stage protocol table | stage protocol breakdown |
| `neg` | `negative_control` | coherent | stage protocol breakdown |

For this lane the legacy chain-summary columns are compatibility projections
only. The authoritative stage-level read is the `Stage protocol breakdown`
section in the markdown report.

### 3.3 Compositional Branching

Current stage ids and intended roles:

| Stage ID | Stage Role | Current Semantics | Authoritative Summary Layer |
| --- | --- | --- | --- |
| `s1` | `branch_left` | semantically dubious as a local prove task in the current starter pack | stage protocol breakdown |
| `s2` | `branch_right` | semantically dubious as a local prove task in the current starter pack | stage protocol breakdown |
| `s3` | `merge_rule` | semantically dubious as a local prove task in the current starter pack | stage protocol breakdown |
| `bg` | `final_gold` | coherent as gold-supported fan-in composition | stage protocol breakdown |
| `bm` | `final_model` | coherent in intent, but current evidence is only meaningful through the authoritative stage protocol table | stage protocol breakdown |
| `neg` | `negative_control` | coherent | stage protocol breakdown |

Branching is the lane where the gap between intended semantics and current
starter-stage semantics is largest.

## 4. Result Summary Per Lane

### 4.1 Compositional Assumption Import

- Aggregate pass band:
  - roughly `20%` to `40%` across the current strongest and weakest recorded
    runs.
- Negative control status:
  - clean; `Negative Twin` is consistently `4/4`.
- `s1` status:
  - dominant collapse point; the failure breakdown repeatedly reports
    `s1 false_refusal`.
- Import-stage status:
  - `0/4` across the displayed chain summaries.
- Gold-final status:
  - non-zero and often substantial for stronger models; for example
    `gpt-oss:120b-cloud` repeatedly reaches `3/4` or `4/4`.
- Model-final status:
  - currently `0/4` across the displayed chain summaries.
- All-stages status:
  - currently `0/4`.

Interpretation:

- the lane does show that negative compositional discipline survives;
- it also shows that gold-supported final composition can work;
- but it does not currently support a clean claim that the same model can first
  prove local lemmas and then stably reuse them, because the early seed stages
  collapse before that claim can be isolated.

### 4.2 Compositional Depth Ladder

- Aggregate pass band:
  - roughly `14%` to `38%`.
- Negative control status:
  - clean in the authoritative stage read; `neg` is `3/3`.
- `s1` status:
  - collapses in the authoritative stage read; `s1` is repeatedly `0/3`.
- Import-stage status:
  - compatibility projection shows `0/3`.
- Gold-final status:
  - the authoritative stage read shows non-zero and sometimes strong `d2g` and
    `d3g`, up to `100%` for stronger runs.
- Model-final status:
  - the authoritative stage read shows `d2m` and `d3m` at `0/3` and `0/2` in
    the displayed rows.
- All-stages status:
  - compatibility projection shows `0/3`.

Interpretation:

- this lane contains a critical read-side nuance:
  - the legacy chain-summary compatibility projection can appear to show strong
    `Model Final`,
  - while the authoritative `Stage protocol breakdown` shows the opposite.
- therefore all claims for this lane must be anchored on the stage protocol
  table, not on the compatibility projection.

### 4.3 Compositional Branching

- Aggregate pass band:
  - roughly `16%` to `33%`.
- Negative control status:
  - clean in the authoritative stage read; `neg` is `3/3`.
- `s1` status:
  - collapses in the authoritative stage read; `s1` is repeatedly `0/3`.
- Import-stage status:
  - compatibility projection shows `0/3`.
- Gold-final status:
  - the authoritative stage read shows `bg` ranging from `0/3` to `3/3`
    depending on model and run.
- Model-final status:
  - the authoritative stage read shows `bm` at `0/3` in the displayed rows.
- All-stages status:
  - compatibility projection shows `0/3`.

Interpretation:

- branching preserves the same negative-discipline signal as the simpler lanes;
- it also preserves a bounded gold-final signal;
- but it is the lane where the branch-building stages themselves are least
  interpretable as local prove tasks.

## 5. Cross-Lane Synthesis

### 5.1 Shared Failure Pattern

Across all three starter lanes:

- negative controls remain strong;
- the earliest local seed or bridge stages collapse first;
- gold-supported final composition can still succeed after that;
- self-produced model-side reuse is not yet established cleanly;
- the current starter packs therefore mix front-end stage-semantics artifacts
  with genuine compositional pressure.

### 5.2 Lane-Specific Pattern

- `compositional_assumption_import_phase1`
  - simplest lane;
  - easiest place to see that `gold-final` can be alive while `import-stage`
    remains `0/4`.
- `compositional_depth_ladder_phase1`
  - strongest example of aggregate/compatibility-projection ambiguity;
  - must be read through the stage protocol breakdown.
- `compositional_branching_phase1`
  - strongest stage-semantics problem;
  - branch-building and merge-rule stages are not cleanly interpretable as
    local proof-construction tasks.

### 5.3 Current Shared Bottleneck

The current evidence leans toward the following conclusion:

- the Hilbert checker boundary itself is not the main issue here;
- negative compositional discipline is already comparatively strong;
- the shared bottleneck is front-end authoring plus stage semantics, especially
  around supposedly local proof stages that are not honestly well-posed local
  prove tasks in the current starter packs.

## 6. Current Claim Boundary

Until stage-semantics cleanup is complete, the current starter compositional
lanes should be read as follows:

- valid as exploratory research evidence;
- valid for measuring negative discipline;
- valid for observing that gold-supported final composition sometimes survives;
- not valid as clean evidence that local proof construction plus self-produced
  compositional reuse has already been demonstrated.

The next canonical mixed-family step therefore must not begin with self-produced
reuse. It should begin with a narrower gold-first probe after cleanup.
