---
title: "Compositional Assumption Import — Experiment Design"
status: active
owner: platform / research
updated_at: "2026-04-02"
source_basis:
  - "research/active/documents/hilbert-ai-verification-benchmark-v2-held-out.md"
  - "research/active/documents/proof-research-phase1-runbook.md"
  - "research/active/documents/proof-research-next-cycle-plan.md"
  - "docs/architecture/adr/global/adr-proof-theoretic-operational-layer-nd-to-hilbert.md"
  - "research/artifacts/result_research/waves/phase1_20260402T153348+0300.csv"
  - "research/artifacts/result_research/waves/phase1_phase1-entailed-gap-audit-20260402.csv"
  - "research/artifacts/result_research/waves/phase1_phase1-direct-axiom-mixed-proof-ladder-20260402.csv"
---

# Compositional Assumption Import — Experiment Design

- **Title:** **Compositional Assumption Import — Experiment Design**
- **Status:** Active staged diagnostic lane
- **Date:** 2026-04-02
- **Scope:** standalone direct-Hilbert experiment for compositional lemma reuse
- **Claim boundary:** research-only evidence; no runtime trust-boundary expansion

---

## 1. Purpose

This document defines the current repository-backed experiment for the next
question after the `assumption_import` canary:

> can a model that proves local entailed lemmas also reuse those lemmas
> compositionally in a later proof context without losing formal reliability?

This is narrower than general theorem synthesis and narrower than the
`ND -> Hilbert` question. The experiment isolates the compositional reuse
boundary inside the existing direct-Hilbert benchmark family.

The claim under test is intentionally limited:

> local proof success does not automatically imply stable lemma reuse across
> proof stages.

---

## 2. Current Implementation Boundary

The experiment is implemented on top of the existing direct-Hilbert
benchmark engine, but it now has its own standalone `researchctl`
surface. The current runner now does have a bounded first-class
imported-certificate object and a minimal trusted reuse layer for the
starter compositional slice.

The current implementation therefore works as follows:

- the Hilbert checker remains the only trust-conferring verifier;
- successful chain stages now emit first-class imported certificate objects
  under `result/imported_certificates/<run-id>/`;
- each compositional run also emits an artifact catalog
  `imported_certificate_catalog_<run-id>.csv`;
- imported lemmas are still represented as formulas made available to the
  model during certificate construction, but the `model` path now resolves
  those formulas through verifier-confirmed imported certificate objects rather
  than directly from prior case goals;
- gold import is backed by the static `imported_lemmas_json` field in the case
  pack;
- model import is backed by formulas from **prior stages in the same chain**
  whose imported certificate objects are marked `trusted_for_reuse=true`;
- trusted reuse is currently limited to prerequisite local lemma stages
  `s1` and `s2`;
- chain-level aggregation is written to a sidecar
  `chain_result_<run-id>.csv`;
- the shared compositional summary CSV **is** now auto-discovered by
  `db sync` and indexed under artifact group `compositional`.
- `chain_result_<run-id>.csv` is now imported by `db sync` into the
  dedicated table `research_chain_summary_rows`; the file itself is indexed in
  `research_result_files` with `file_kind = 'chain_summary'` and artifact
  group `compositional`.

This document is therefore honest about the current semantics:

- the experiment already measures compositional reuse pressure inside the
  direct-Hilbert pipeline;
- it now exercises a bounded artifact-level import protocol for the starter
  slice;
- it does **not** yet validate deeper branching/depth-2+ artifact reuse.

---

## 3. Why This Experiment Exists Now

Current evidence already suggests:

- negative controls are comparatively strong for the best models;
- the main weakness has shifted toward entailed/formalization behavior;
- `assumption_import` is a visible bottleneck;
- category-level results still do not tell us whether models can *reuse*
  previously established lemmas compositionally.

This experiment separates three capabilities that otherwise remain conflated:

1. local proof construction;
2. reuse of already validated lemmas inside a fresh proof context;
3. composition of those lemmas into a new goal.

---

## 4. Main Hypotheses

### H1 — Local success does not imply compositional success

A model may prove `A -> B` and `B -> C` locally while still failing to reuse
those lemmas compositionally for `A -> C`.

### H2 — Import-stage instability is a distinct failure class

The dominant failures are expected to remain in:

- `request_failure`
- `schema_failure`
- `parse_failure`
- `kernel_failure`
- missing or degraded imported lemmas in `model` mode

rather than only in clean logical refusal.

### H3 — Reliability should degrade with chain complexity

Even if depth-1 chains are usable, deeper chains or richer mixed proof shapes
are expected to degrade quickly.

### H4 — Gold import and model import answer different questions

- `gold` measures whether the model can use correct formal resources.
- `model` measures whether an end-to-end self-produced pipeline is stable.

---

## 5. Current Runnable Slice

The repository now materializes a bounded **starter slice** rather than the
full future ladder.

Current in-repo starter slice:

- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-assumption-import-phase1.csv`
- local config:
  - `research/config/compositional/assumption-import.yaml`

Current starter coverage:

- `4` chains
- `5` stages per chain
- total `20` stage-cases
- depth `1` only
- positive chain families:
  - atomic linear chain
  - structure-preserving renaming
  - distractor/irrelevant-assumption composition
  - negated-antecedent composition
- each chain also includes a negative twin

Deferred for later expansion:

- theoremized composition
- depth-2 and depth-3 ladders
- branching composition
- richer mixed-proof composition

Those tiers remain part of the design space, but they are **not** yet the
current runnable baseline.

---

## 6. Canonical Stage Protocol

Each chain is executed as a fixed stage sequence.

### Stage `s1`

- local lemma 1
- example: `A -> B`

### Stage `s2`

- local lemma 2
- example: `B -> C`

### Stage `s3g`

- gold import stage
- uses the static `imported_lemmas_json` list from the case pack
- example goal: `A -> C`

### Stage `s3m`

- model import stage
- imported lemmas are resolved dynamically from earlier stages in the **same**
  chain whose result rows already scored `pass`
- in the current implementation, the imported formulas are the earlier stage
  goals, not a separate imported certificate object

### Stage `neg`

- negative twin
- near-miss composition control
- expected behavior: `not_derivable`

---

## 7. Gold vs Model Import in the Current Runner

### 7.1 Gold import (`provenance_mode=gold`)

For `gold` rows, the runner uses:

- `assumptions_json`
- `imported_lemmas_json`

exactly as stored in the case pack.

### 7.2 Model import (`provenance_mode=model`)

For `model` rows, the runner resolves imported lemmas from prior chain stages
through imported certificate objects:

- same `chain_id`
- earlier file order in the same run
- only stages `s1` and `s2`
- only if those earlier stages produced imported certificate objects with:
  - `score_bucket = pass`
  - `schema_status = pass`
  - `parse_status = pass`
  - `kernel_status = accept`
  - `trusted_for_reuse = true`

The runner then places the resolved formulas into the prompt as imported
lemmas and also records:

- requested import count
- resolved import count
- resolved imported artifact refs
- per-row `certificate_object_file` for stages that emitted a verified reusable
  artifact

inside the stage-level `notes` field.

This is intentionally narrower than a future generalized artifact-reuse
protocol, but it is already methodologically useful:

- if `s3g` is strong and `s3m` is weak, the main issue is not composition in
  the abstract but end-to-end self-produced pipeline stability;
- if both are weak, the problem is composition itself or front-end authoring
  discipline.

---

## 8. Files and Operator Commands

Canonical files for the current lane:

- design document:
  - `research/active/documents/compositional-assumption-import-experiment-design.md`
- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/compositional-assumption-import-phase1.csv`
- local config:
  - `research/config/compositional/assumption-import.yaml`
- per-run stage results:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_<run-id>.csv`
- shared compositional summary:
- `research/artifacts/result_research/compositional/compositional-assumption-import_<run-id>.csv`
- per-run chain summary sidecar:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/chain_result_<run-id>.csv`
- per-run imported certificate objects:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/imported_certificates/<run-id>/`
- per-run imported certificate catalog:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/imported_certificate_catalog_<run-id>.csv`
- per-run raw outputs:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/<run-id>/`

Canonical operator commands:

```powershell
Set-Location 'C:\Users\nokclock\Documents\GitHub\collabsphere'
go -C platform-tooling run ./cmd/researchctl compositional-assumption-import plan --local-config research/config/compositional/assumption-import.yaml --jobs 2
go -C platform-tooling run ./cmd/researchctl compositional-assumption-import run --local-config research/config/compositional/assumption-import.yaml --jobs 2 --best-effort
```

Use `--jobs 1` if the local serving surface cannot sustain concurrent model
requests.

---

## 9. Current Case-Pack Schema

The implemented case-pack schema is:

```csv
case_id,chain_id,stage_id,category,label,difficulty,case_family,chain_depth,provenance_mode,atom_renaming_id,assumptions_json,imported_lemmas_json,goal,expected_behavior,comment
```

Column meanings:

- `case_id`
  - stable stage-case identifier such as `CI01-S1`
- `chain_id`
  - stable chain identifier such as `CI01`
- `stage_id`
  - one of `s1`, `s2`, `s3g`, `s3m`, `neg`
- `category`
  - current canonical value:
    - `compositional_assumption_import`
- `label`
  - `entailed` or `not_entailed`
- `difficulty`
  - bounded difficulty marker
- `case_family`
  - current starter values:
    - `linear_chain`
    - `distractor`
    - `near_miss_negative`
- `chain_depth`
  - currently `1` for the starter slice
- `provenance_mode`
  - `none`, `gold`, or `model`
- `atom_renaming_id`
  - structural-equivalence marker such as `r0`, `r1`, `neg1`
- `assumptions_json`
  - top-level assumptions directly present in the stage
- `imported_lemmas_json`
  - target imported lemmas for `gold`;
  - reference import list for `model`
- `goal`
  - target formula
- `expected_behavior`
  - `prove` or `not_derivable`
- `comment`
  - short human-readable note

Important implementation note:

- for `provenance_mode=model`, the runner does **not** blindly trust
  `imported_lemmas_json` as the effective imported set;
- instead it resolves effective imported lemmas from prior successful stages
  and records the discrepancy in the result row `notes`.

---

## 10. Stage-Level Result Contract

The existing `result_<run-id>.csv` surface has been extended for this lane.

Additional columns now carried through from the case pack or the richer import
resolution layer:

```csv
chain_id,stage_id,case_family,chain_depth,provenance_mode,atom_renaming_id,output_length,proof_steps_count,requested_imports_json,effective_imports_json,resolved_import_refs_json,certificate_object_file
```

The stage-level rows therefore now preserve:

- chain identity
- stage identity
- family/depth provenance
- requested vs effective import provenance
- resolved imported artifact references
- emitted imported certificate object path
- raw output length
- number of proof steps when a certificate JSON was produced

This is sufficient to support:

- per-stage score analysis
- chain reconstruction
- simple prompt-discipline diagnostics
- future aggregation without introducing a second incompatible result format

---

## 11. Chain Summary Contract

The current implementation writes a dedicated sidecar:

```text
chain_result_<run-id>.csv
```

Schema:

```csv
run_id,chain_id,case_family,chain_depth,atom_renaming_id,stage1_pass,stage2_pass,stage3_gold_pass,stage3_model_pass,negative_twin_pass,all_stages_pass,import_stage_pass,final_composition_pass_gold,final_composition_pass_model,conditional_final_pass_gold,conditional_final_pass_model,failure_stage,failure_type,notes
```

Current derived formulas are:

- `stage1_pass`
  - `true` iff `s1` scored `pass`
- `stage2_pass`
  - `true` iff `s2` scored `pass`
- `stage3_gold_pass`
  - `true` iff `s3g` scored `pass`
- `stage3_model_pass`
  - `true` iff `s3m` scored `pass`
- `negative_twin_pass`
  - `true` iff `neg` scored `pass`
- `import_stage_pass`
  - `stage1_pass && stage2_pass`
- `final_composition_pass_gold`
  - `stage3_gold_pass`
- `final_composition_pass_model`
  - `stage3_model_pass`
- `conditional_final_pass_gold`
  - `import_stage_pass && stage3_gold_pass`
- `conditional_final_pass_model`
  - `import_stage_pass && stage3_model_pass`
- `all_stages_pass`
  - conjunction of all five stage booleans
- `failure_stage`
  - first missing or failing stage in order:
    - `s1`
    - `s2`
    - `s3g`
    - `s3m`
    - `neg`
- `failure_type`
  - `missing_stage` when a stage row is absent;
  - otherwise the first failing stage's `score_bucket`

This explicit chain layer is required because stage pass rates alone do not
answer whether the compositional pipeline is operationally usable.

Current DB/read-side visibility for this layer:

- top-level `researchctl db overview` now reports `total_chain_summary_rows`
- per-research overview rows now report:
  - `chain_summary_files`
  - `chain_summary_rows`
- per-artifact-group overview rows now report:
  - `chain_summary_files`
  - `chain_summary_rows`
- standalone compositional markdown reports now render a dedicated
  `Chain Summary` section from the same sidecar layer when chain rows exist

---

## 12. Core Metrics

The current starter slice SHOULD be read through these metrics:

### 12.1 Stage pass rates

- `pass(s1)`
- `pass(s2)`
- `pass(s3g)`
- `pass(s3m)`
- `pass(neg)`

### 12.2 End-to-end chain pass rate

- fraction of chains where all mandatory stages pass

### 12.3 Conditional composition pass

- `P(s3g pass | s1 and s2 pass)`
- `P(s3m pass | s1 and s2 pass)`

The second metric is especially important because it isolates the extra damage
introduced by self-produced intermediate artifacts.

### 12.4 Composition gap

Operational definition for the starter slice:

```text
composition_gap_gold = min(pass(s1), pass(s2)) - pass(s3g)
composition_gap_model = min(pass(s1), pass(s2)) - pass(s3m)
```

### 12.5 Failure localization

Failure mix MUST still be read across:

- `request_failure`
- `schema_failure`
- `parse_failure`
- `kernel_failure`
- `false_refusal`
- `false_accept`

### 12.6 Negative compositional discipline

The negative twin MUST remain explicit so the lane does not accidentally reward
models that glue together any visible implication pair.

---

## 13. Success Criteria for the Starter Slice

The current continuation thresholds are intentionally modest and bound to the
implemented depth-1 starter slice.

For the strongest `v1.3` models, a good signal is:

- `s3g >= 0.90`
- `s3m >= 0.70`
- `negative_twin pass` remains high
- `false_accept = 0`
- `schema_failure` and `request_failure` at import stages are occasional
  rather than dominant

The strongest positive signal would be:

- `s3g >= 0.95`
- `s3m >= 0.80`
- `composition_gap_gold <= 10` percentage points
- `composition_gap_model <= 20` percentage points
- no collapse in the negative twin

These are operator heuristics for the current research cycle, not product
promotion thresholds.

---

## 14. Kill Criteria

The current lane SHOULD be treated as a stop signal for deeper expansion if any
of the following holds:

- depth-1 `gold` composition is still materially weak after prompt/contract
  cleanup;
- `model` mode collapses mostly through request/schema instability rather than
  interpretable logical limits;
- negative twins start drifting toward unsafe acceptance;
- imported-lemma resolution stays too brittle to support even this starter
  slice.

If that happens, the honest conclusion is:

> the current direct-Hilbert front-end is still too fragile for stronger
> compositional claims, even though the checker boundary itself remains useful.

---

## 15. What This Lane Does Not Yet Establish

Even a strong result here does **not** yet establish:

- that deeper composition or branching proof graphs are stable;
- that direct Hilbert is the best long-run authoring surface;
- that `ND -> Hilbert` is unnecessary;
- that this should be promoted directly into runtime product claims.

What it *would* establish is narrower and still valuable:

> under the current direct-Hilbert research surface, the best models can or
> cannot reuse previously established lemmas compositionally at depth 1 with a
> measurable gap between gold-import and model-import behavior.

---

## 16. Next Expansion Only After a Clean Starter Read

Only after the current starter slice is stable and interpretable SHOULD the
experiment expand to:

1. theoremized composition
2. depth-2 linear chains
3. branching composition
4. mixed compositional proof families
5. richer artifact-level import beyond the current starter trusted-reuse layer

Until then, the canonical question is the narrow one:

> does depth-1 compositional lemma reuse survive the current direct-Hilbert
> surface well enough to justify deeper work?
