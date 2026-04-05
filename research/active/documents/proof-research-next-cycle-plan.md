# Proof Research Next-Cycle Plan

- **Title:** **Proof Research Next-Cycle Plan**
- **Status:** Active research execution plan
- **Date:** 2026-04-02
- **Scope:** immediate next-cycle ordering across direct Hilbert, paired ND, and Lean4-adjacent follow-on work
- **Claim boundary:** research prioritization only; this document does not create a new empirical approval claim

---

## 1. Purpose

This document defines the immediate execution order for the next proof-theory
research cycle.

It exists because the current repository now has:

- a bounded positive result for the Hilbert checker as a conservative boundary;
- evidence that `v1.3` is methodologically cleaner but operationally harder;
- clear signs that some runs mix real model limitations with operational
  invalidity;
- a remaining ambiguity about whether the main frontier problem is:
  - proof construction itself;
  - contract / lowering discipline;
  - or the direct-Hilbert authoring language.

The goal of this plan is to stop broad unstructured reruns and instead execute
the next cycle in an order that maximizes interpretability.

This plan does **not** rename or replace the existing workflow phases:

- `phase1` remains the direct-Hilbert rerun slice;
- `phase2` remains the paired `direct` vs `ND -> Hilbert` slice;
- `phase3` remains the Lean4 support-layer generation/export slice;
- `phase3 compare` remains the benchmark comparison lane on exported Lean packs.

---

## 2. Governing Constraints

The next cycle MUST continue to respect the currently active benchmark-policy
constraints from:

- [Hilbert AI Verification Benchmark v2 Held-Out](./hilbert-ai-verification-benchmark-v2-held-out.md)
- [Proof Research Phase 1 Runbook](./proof-research-phase1-runbook.md)
- [Proof Research Phase 2 Runbook](./proof-research-phase2-runbook.md)
- [Proof Research Phase 3 Runbook](./proof-research-phase3-runbook.md)

In particular:

- `false_accept = 0` remains the central conservative-boundary condition;
- `v1.3` remains the preferred clean evaluation baseline;
- purely operationally invalid runs MUST remain historical evidence but MUST
  NOT drive main comparative claims;
- the current direct-Hilbert evidence still does **not** settle whether direct
  Hilbert is the best LLM-facing authoring language.

Important terminology rule for this plan:

- when this document uses numeric thresholds such as `>= 90%` or
  `<= 15-20 percentage points`, treat them as **operator heuristics** for the
  next cycle, not as already-approved product gates unless another canonical
  benchmark document explicitly says so.

---

## 3. Immediate Execution Order

The next cycle SHOULD be executed in this order:

1. valid-run cleanup
2. `assumption_import` canary on `v1.3`
3. `entailed` vs `not_entailed` gap audit plus a `negative_refusal` sanity lane
4. `direct_axiom_instance -> mixed_proof` difficulty ladder
5. `theorem_synthesis-deep-stress`
6. paired `v1.2` vs `v1.3` study
7. `ND -> Hilbert`

Rationale for the ordering:

- items `1-4` decide whether the current frontier problem is mostly
  operational / contract-facing or already fundamentally compositional;
- item `5` is the main scientific stress test, but it is too expensive to
  interpret cleanly before simpler failure classes are isolated;
- item `6` is the methodological delta-study for prompt-shape support;
- item `7` is the front-end-language follow-on experiment and should not be
  used as an escape hatch before the direct-Hilbert failure modes are
  characterized honestly.

Broad new waves SHOULD pause until this sequence is either:

- executed in order; or
- explicitly replaced by a newer active plan.

Related focused follow-on lane:

- [Compositional Assumption Import — Experiment Design](./compositional-assumption-import-experiment-design.md)
- [Compositional Depth Ladder — Experiment Design](./compositional-depth-ladder-design.md)
- [Compositional Branching — Experiment Design](./compositional-branching-design.md)
- [Compositional Mixed-Family Reuse — Experiment Design](./compositional-mixed-family-reuse-design.md)

That lane is now repository-backed as its own standalone
`researchctl compositional-assumption-import` surface, but it SHOULD still be
interpreted through the ordering discipline in this plan rather than as a
replacement for the current next-cycle sequence.

The deeper follow-on now also exists as a separate standalone surface:

- `researchctl compositional-depth-ladder`
- `researchctl compositional-branching`
- `researchctl compositional-mixed-family-reuse`

It SHOULD be read as a protocol-expansion experiment rather than as a
replacement for the main next-cycle ordering.

Current honesty boundary for that lane:

- it now includes a bounded artifact-level trusted reuse layer for the starter
  slice;
- the new depth-ladder lane extends this to controlled depth-2/3 chains;
- the new branching lane extends this to bounded fan-in composition;
- the new mixed-family lane extends this to bounded heterogeneous-family reuse
  within one chain;
- none of these lanes yet settles cross-chain reuse or a fully general
  imported-certificate graph.

Additional 2026-04-03 correction:

- the canonical compositional positive read should now come primarily from the
  cleaned `compositional-mixed-family-gold-first` surface;
- that surface now uses a larger `fg/neg` pack and should be interpreted as a
  cleaner gold-first reuse probe;
- the next annotated expansion now exists separately as
  `compositional-mixed-family-gold-first-phase2`, rather than silently
  replacing the clean starter `v1` lane;
- the next adversarial follow-on now also exists separately as
  `compositional-mixed-family-gold-first-hard`;
- it MUST NOT be treated as automatic rehabilitation of the exploratory
  `compositional-depth-ladder` or `compositional-branching` surfaces.

---

## 4. Stage 1 — Valid-Run Cleanup

### 4.1 Why first

This step is first because the benchmark family already defines a comparative
validity rule:

- runs with `request_failure_count == cases_total`;
- or pure `model not found`;
- or pure endpoint / transport failure before meaningful inference

MUST remain historical evidence but MUST be excluded from the main comparison.

### 4.2 What this stage tests

This stage separates:

- real model limitations;
- from operational invalidity.

### 4.3 Expected signal

The main cohort should become interpretable as a model comparison rather than a
mixture of proof-quality evidence and serving-layer failures.

### 4.4 Success criteria

For all models kept in the main cohort:

- `request_failure_count < cases_total`
- no pure `model not found` runs
- no pure pre-inference endpoint failures
- final comparison tables use only comparatively valid runs

### 4.5 Kill criteria

If a model remains mostly operationally invalid after cleanup and one rerun
attempt under a fixed serving configuration, it SHOULD be removed from the
main comparative slice and SHOULD stop consuming benchmark budget in that
cycle.

---

## 5. Stage 2 — `assumption_import` Canary on `v1.3`

### 5.1 Why second

This is the cheapest clean test of lowering discipline rather than deep proof
construction.

It asks whether a model can avoid damaging a trivial certificate when reasoning
burden is near zero.

### 5.2 What this stage tests

- front-end output discipline
- contract following under the stricter `v1.3` prompt
- whether the main bottleneck is authoring/lowering rather than proof theory

### 5.3 Expected signal

If narrow contract-oriented prompt fixes produce a sharp improvement here, the
main bottleneck is likely contract/lowering discipline rather than deep formal
reasoning.

### 5.4 Success criteria

For the strongest `v1.3` models, the operator target for the next cycle is:

- `assumption_import >= 90%`
- rare `schema_failure`
- rare `request_failure`
- `false_accept = 0`

These numeric targets are operator heuristics, not a previously approved
product gate.

### 5.5 Kill criteria

If even trivial entailed cases remain unstable after targeted contract/output
fixes, that is strong evidence that the current direct-Hilbert front-end is
too fragile for stronger claims at this stage.

---

## 6. Stage 3 — `entailed` vs `not_entailed` Gap Audit

### 6.1 Why third

The core research question is whether the system is becoming:

- a proof constructor;
- or only a strong rejector.

Current evidence already suggests that `not_entailed` is often easier than
`entailed`.

### 6.2 What this stage tests

- whether the system is only conservatively refusing unsafe outputs;
- whether strong `negative_refusal` coexists with usable entailed-case
  construction;
- whether the current success is mostly barrier-like rather than generative.

### 6.3 Expected signal

This stage should show whether the strongest `v1.3` models can reduce the
`entailed` / `not_entailed` gap while preserving:

- `false_accept = 0`
- high `negative_refusal`

### 6.4 Success criteria

Operator target for the strongest `v1.3` models:

- preserve `false_accept = 0`
- preserve high `negative_refusal`
- reduce the `entailed` vs `not_entailed` pass-rate gap to roughly
  `<= 15-20 percentage points`

Again, the numeric gap target is an operator heuristic rather than an existing
benchmark-family hard gate.

### 6.5 Kill criteria

If `not_entailed` remains excellent while `entailed` remains badly degraded,
the honest current interpretation SHOULD be:

> the system is acting as a safe rejector more than as a reliable proof
> generator.

---

## 7. Stage 4 — Difficulty Ladder

Stage 4 uses the category ladder:

- `direct_axiom_instance`
- `mixed_proof`

### 7.1 Why fourth

This isolates whether the pipeline breaks:

- already at basic axiom application;
- or only when multiple valid steps must be composed.

### 7.2 Expected signal

The ladder should distinguish:

- formatting/contract weakness at the basic step level;
- from compositional planning weakness in multi-step proof assembly.

### 7.3 Success criteria

Operator targets for the strongest models:

- `direct_axiom_instance >= 85%`
- `mixed_proof >= 70%`
- failure mix shifts away from schema/request collapse and toward more
  interpretable logical or frontier-limit failures

The phrase `acceptable mixed_proof` is already aligned with the current gate
language; the explicit percentages here remain operator heuristics.

### 7.4 Kill criteria

If `direct_axiom_instance` improves materially while `mixed_proof` does not,
the next-cycle interpretation SHOULD be:

- the main remaining problem is compositional reasoning/planning;
- not merely output formatting.

---

## 8. Stage 5 — `theorem_synthesis-deep-stress`

### 8.1 Why fifth

This is the main scientific stress test, but it is more expensive and more
ambiguous than the earlier stages.

The current evidence already flags `theorem_synthesis` as the weakest major
area.

### 8.2 Expected signal

Either:

- the strongest `v1.3` models begin to show moderate stable synthesis ability;

or:

- the current frontier is still below reliable theorem invention and closer to
  mixed-proof composition only.

### 8.3 Success criteria

Operator target:

- roughly `60-70%+` on `theorem-synthesis-deep-stress`
- across multiple reruns
- while preserving `false_accept = 0`

This is intended as a working interpretation of the existing benchmark-family
requirement that the strongest models show at least moderate
`theorem_synthesis`.

### 8.4 Kill criteria

If theorem-synthesis continues to fail mainly through:

- `schema_failure`
- `request_failure`

rather than through interpretable logical limits, the current direct-Hilbert
front-end SHOULD be treated as not yet ready for strong generalization claims.

---

## 9. Stage 6 — Paired `v1.2` vs `v1.3`

### 9.1 Why sixth

The repository already distinguishes:

- `v1.2` as engineering-best-performance;
- `v1.3` as the cleaner generalization baseline with inline examples removed.

The next cycle still needs a cleaner paired delta-study.

### 9.2 What this stage tests

This stage estimates how much apparent success depends on:

- prompt-shape support and benchmark-aligned examples;
- versus actual transferable reasoning under the stricter prompt.

### 9.3 Success criteria

Desired outcome:

- `v1.3` is weaker than `v1.2`, but not catastrophically so;
- qualitative ranking of the strongest models does not invert chaotically;
- strongest models preserve `false_accept = 0` on both prompt versions

### 9.4 Kill criteria

If `v1.2` remains far stronger while `v1.3` degrades severely, the current
direct-Hilbert result SHOULD be interpreted more as prompt-assisted contract
retention than as strong held-out generalization.

---

## 10. Stage 7 — `ND -> Hilbert`

### 10.1 Why seventh

This remains the explicit follow-on front-end experiment, but it should not be
used to avoid characterizing direct-Hilbert honestly first.

### 10.2 What this stage tests

Whether the main frontier problem is:

- the Hilbert kernel itself;
- or the direct Hilbert authoring language.

### 10.3 Success criteria

The ND track should be treated as promising only if it:

- reduces `schema_failure` and `request_failure`
- improves entailed, `mixed_proof`, and `theorem_synthesis`
- preserves `false_accept = 0`

### 10.4 Kill criteria

If the ND front-end does not materially reduce front-end failures or weakens
the conservative boundary, the branch SHOULD be frozen rather than expanded.

---

## 11. Main Branch Point For The Next Cycle

The most important branch point is this:

If stages `1-4` show that:

- `false_accept` remains `0`;
- `not_entailed` remains strong;
- but trivial and mid-level entailed cases remain unstable;

then the most honest current conclusion is:

> the kernel boundary is already useful, but direct-Hilbert proof authoring is
> not yet a reliably strong front-end.

That branch point SHOULD determine whether the next major investment goes into:

- more direct-Hilbert prompt/contract work;
- or the `ND -> Hilbert` front-end path.

---

## 12. Relationship To Existing Workflow Phases

This plan spans multiple existing workflow surfaces:

- stages `1-6` are mainly `phase1` work, with selective reuse of auxiliary case
  packs and prompt-version comparisons;
- stage `7` is mainly `phase2`;
- Lean4 remains relevant as an additive support layer through `phase3`, but it
  is not the primary diagnostic lane for the immediate next cycle.

Therefore this document defines:

- execution order across phases;

not:

- a replacement for the current `phase1` / `phase2` / `phase3` command model.

---

## 13. Concrete Repository-Local Lanes For Stages 2-5

The repository now materializes stages `2-5` as concrete `phase1` auxiliary
case-pack lanes.

Stage-to-lane mapping:

- Stage `2`:
  - case pack: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/assumption-import-canary.csv`
  - config: `research/config/direct/phase1-assumption-import-canary.yaml`
- Stage `3`:
  - gap-audit pack: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/entailed-gap-audit.csv`
  - sanity pack: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/negative-refusal-sanity.csv`
  - configs:
    - `research/config/direct/phase1-entailed-gap-audit.yaml`
    - `research/config/direct/phase1-negative-refusal-sanity.yaml`
- Stage `4`:
  - case pack: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/direct-axiom-to-mixed-proof-ladder.csv`
  - config: `research/config/direct/phase1-direct-axiom-mixed-proof-ladder.yaml`
- Stage `5`:
  - case pack: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/theorem-synthesis-deep-stress.csv`
  - config: `research/config/direct/phase1-theorem-synthesis-deep-stress.yaml`

All of these repository-local profiles currently use:

- the current main-cohort compatible family only:
  - `gpt-oss:20b`
  - `gpt-oss:120b-cloud`
  - `deepseek-r1:14b`
  - `deepseek-v3.1:671b-cloud`
- deterministic family-level sampling:
  - `temperature=0`
  - `seed=42`
  - `top_p=1`
- `repeats=3`
- positive `request_timeout_abort_threshold`
- `interrupt_policy=drop_if_no_results`

These defaults are intentionally narrower than a broad local matrix so that
the stages can be run as focused diagnostics rather than as another mixed
appendix wave.

The adjacent broad-but-cheap appendix/main-cohort screen now also has an
explicit repository-local lane:

- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/cross-model-sanity.csv`
- config:
  - `research/config/direct/phase1-cross-model-sanity.yaml`

Unlike the stage `2-5` focused lanes, this cross-model sanity profile keeps
`glm-5:cloud` in the family so that cheap breadth-first screening can still be
run without switching back to the broader default local matrix.
