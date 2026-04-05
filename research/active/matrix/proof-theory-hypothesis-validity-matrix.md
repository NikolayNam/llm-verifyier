# Proof-Theory Hypothesis Validity Matrix

- **Title:** **Proof-Theory Hypothesis Validity Matrix**
- **Status:** Active launch-read guardrail
- **Date:** 2026-04-04
- **Context:** CollabSphere / Direct Hilbert / ND-to-Hilbert / Lean4 sidecar / compositional reuse lanes
- **Implementation authority:** Classification guardrail only

---

## 1. Purpose

This document classifies the current active proof-theory research hypotheses
into four launch-read columns:

- `valid`
- `invalid`
- `exploratory only`
- `not theory hypothesis`

The matrix exists to prevent two recurring errors:

- reading an exploratory lane as clean claim-bearing evidence;
- reading an operational control, gate, or tooling question as if it were a
  theory hypothesis.

This document does not upgrade any result. It only constrains how current
research surfaces should be interpreted before further runs.

---

## 2. Column meanings

### `valid`

Use `valid` only when the hypothesis is methodologically suitable for
claim-bearing research within its explicitly bounded scope.

`valid` does **not** mean empirically confirmed. It only means:

- the hypothesis is falsifiable enough to test;
- the current runnable surface does not carry a known semantic compromise that
  invalidates clean reading; and
- the claim boundary is narrow enough to defend honestly.

### `invalid`

Use `invalid` only when the current formulation itself should not be used as a
claim-bearing research hypothesis.

Current snapshot note:

- no active row below is currently classified as `invalid`;
- the main problem in the compromised compositional starter lanes is not that
  the broad question is nonsensical, but that the current empirical surface is
  not clean enough to support strong reading.

### `exploratory only`

Use `exploratory only` when:

- the lane may still produce useful debugging or diagnostic signal;
- but the current runnable surface is semantically too compromised, too dirty,
  or too provisional to support a clean theory claim.

### `not theory hypothesis`

Use `not theory hypothesis` when the item is active and important, but is
actually one of the following:

- an operator workflow;
- a tooling or infrastructure hypothesis;
- a benchmark expansion without a distinct theory claim;
- a gate, heuristic, threshold, or reporting/control layer.

---

## 3. Working Matrix

| Surface | Core question | Valid | Invalid | Exploratory only | Not theory hypothesis | Notes / current claim boundary |
| --- | --- | --- | --- | --- | --- | --- |
| Direct Hilbert benchmark `v1` | Can a minimal Hilbert-style certificate kernel serve as a useful trust boundary for a bounded formalizable subset of AI-assisted reasoning outputs? | yes |  |  |  | Canonical bounded trust-boundary hypothesis. Do not inflate to open-domain truth verification. Evidence anchor: [`hilbert-ai-verification-benchmark-v1.md`](../documents/hilbert-ai-verification-benchmark-v1.md). |
| Direct Hilbert held-out benchmark `v2` | Does the same Hilbert trust boundary retain value under held-out `v1.3` prompt discipline with reduced scaffolding and benchmark-aligned anti-shape-imitation controls? | yes |  |  |  | Valid as a bounded held-out benchmark hypothesis. Gates such as `false_accept = 0` and pass-rate thresholds are evaluation criteria, not the hypothesis itself. Evidence anchor: [`hilbert-ai-verification-benchmark-v2-held-out.md`](../documents/hilbert-ai-verification-benchmark-v2-held-out.md). |
| `ND -> Hilbert` pilot | Can a Natural Deduction front-end with deterministic lowering improve end-to-end verified performance while preserving the Hilbert kernel as the final trust boundary? | yes |  |  |  | Valid comparative front-end hypothesis. The runtime is still a pilot; do not read current infrastructure as already proving benefit. Evidence anchor: [`hilbert-ai-verification-benchmark-nd-v1.md`](../documents/hilbert-ai-verification-benchmark-nd-v1.md). |
| Lean4 worker / sidecar | Can Go orchestration plus a Lean4 worker provide mechanized support, backlog generation, and formal sidecar artifacts without replacing the Hilbert kernel as the approved trust boundary? |  |  |  | yes | This is a method and infrastructure hypothesis, not a direct theory hypothesis about model proof behavior. Keep it out of theory-result headlines. Evidence anchor: [`hilbert-ai-verification-lean4-worker-v1.md`](../documents/hilbert-ai-verification-lean4-worker-v1.md). |
| Compositional assumption-import starter lane | Can a starter chain show clean local imported-lemma reuse signal? |  |  | yes |  | Current runnable surface is semantically compromised by dirty source-stage interpretation; useful only as exploratory signal. Evidence anchors: [`compositional-assumption-import-experiment-design.md`](../documents/compositional-assumption-import-experiment-design.md), [`compositional-lanes-interpretation.md`](../documents/compositional-lanes-interpretation.md), [`compositional-stage-semantics-cleanup.md`](../documents/compositional-stage-semantics-cleanup.md). |
| Compositional depth-ladder starter lane | Can deeper linear chain composition already be read as clean theory-bearing evidence? |  |  | yes |  | Current starter depth ladder remains exploratory only for the same stage-semantics reasons. Do not treat it as clean evidence for generalized compositional authoring. Evidence anchors: [`compositional-depth-ladder-design.md`](../documents/compositional-depth-ladder-design.md), [`compositional-lanes-interpretation.md`](../documents/compositional-lanes-interpretation.md). |
| Compositional branching starter lane | Can multi-parent branching composition already be read as clean theory-bearing evidence? |  |  | yes |  | The branching starter surface is still exploratory only. It may provide runtime/debugging signal, but not a defended theory claim. Evidence anchors: [`compositional-branching-design.md`](../documents/compositional-branching-design.md), [`compositional-stage-semantics-cleanup.md`](../documents/compositional-stage-semantics-cleanup.md). |
| Compositional mixed-family reuse starter lane | Can the provisional mixed-family starter surface support a clean compositional reuse claim? |  |  | yes |  | No. Treat it as provisional exploratory evidence only. The canonical next mixed-family step was intentionally moved away from this lane. Evidence anchors: [`compositional-mixed-family-reuse-design.md`](../documents/compositional-mixed-family-reuse-design.md), [`compositional-lanes-interpretation.md`](../documents/compositional-lanes-interpretation.md). |
| Mixed-family gold-first `v1` | Can a model use verifier-correct imported gold lemmas from another proof family without self-produced reuse pressure? | yes |  |  |  | Canonical current compositional theory hypothesis. Narrow claim only: heterogeneous-family gold reuse plus preserved negative discipline. Evidence anchor: [`mixed-family-gold-first-design.md`](../documents/mixed-family-gold-first-design.md). |
| Mixed-family gold-first `phase2` annotated lane | Does the same narrow gold-first hypothesis still hold across richer transition groups, import arities, bridge depths, and other annotated slices? | yes |  |  |  | Valid as an annotated benchmark expansion of the same narrow hypothesis. Do not reinterpret it as self-produced reuse or a general artifact-graph claim. Evidence anchors: [`mixed-family-gold-first-phase2-plan.md`](../documents/mixed-family-gold-first-phase2-plan.md), [`mixed-family-gold-first-design.md`](../documents/mixed-family-gold-first-design.md). |
| Mixed-family gold-first `hard` lane | Does the same narrow gold-first hypothesis survive under explicitly harder and adversarial transition slices? | yes |  |  |  | Valid as an adversarial stress extension of the gold-first hypothesis. A failing hard result does not by itself refute the core trust-boundary hypothesis. Evidence anchor: [`mixed-family-gold-first-hard-plan.md`](../documents/mixed-family-gold-first-hard-plan.md). |
| Mixed-family gold-first `hard micro` slice | Should the smaller micro-pack be read as a separate theory hypothesis? |  |  |  | yes | No. It is an operator-focused read on the hardest transition groups for the strongest models, not a distinct theory hypothesis. Treat it as a narrower execution slice of the hard lane. Evidence anchor: [`mixed-family-gold-first-hard-plan.md`](../documents/mixed-family-gold-first-hard-plan.md). |
| Focused direct diagnostic lanes | Are `cross-model-sanity`, `assumption-import-canary`, `entailed-gap-audit`, `negative-refusal-sanity`, `direct-axiom-to-mixed-proof-ladder`, and `theorem-synthesis-deep-stress` separate theory hypotheses? |  |  |  | yes | No. These are diagnostic or stress surfaces that help measure the direct-Hilbert hypotheses; they are not distinct theory claims by themselves. Evidence anchor: [`proof-research-next-cycle-plan.md`](../documents/proof-research-next-cycle-plan.md). |
| Product gates and thresholds | Are `false_accept = 0`, `high negative_refusal`, `>= 90%`, `<= 15–20 pp`, and similar thresholds theory hypotheses? |  |  |  | yes | No. These are operator gates or heuristics. Some are document-level gates, others are working thresholds; none of them are theory hypotheses. Evidence anchors: [`hilbert-ai-verification-benchmark-v2-held-out.md`](../documents/hilbert-ai-verification-benchmark-v2-held-out.md), [`proof-research-next-cycle-plan.md`](../documents/proof-research-next-cycle-plan.md). |
| Phase runbooks and launch workflows | Are `phase1`, `phase2`, `phase3`, rerun profiles, and launch workflows theory hypotheses? |  |  |  | yes | No. These are workflow and operator-control artifacts. They matter for execution discipline, not for proof-theory truth claims. Evidence anchors: [`proof-research-phase1-runbook.md`](../documents/proof-research-phase1-runbook.md), [`proof-research-phase2-runbook.md`](../documents/proof-research-phase2-runbook.md), [`proof-research-phase3-runbook.md`](../documents/proof-research-phase3-runbook.md). |

---

## 4. Launch-read rules

Before running a new benchmark wave:

- use `valid` rows for claim-bearing benchmark design and report language;
- use `exploratory only` rows only for local debugging, runtime learning, or
  bounded exploratory notes;
- use `not theory hypothesis` rows only as workflow controls, gates, or
  methodological support artifacts;
- do not silently upgrade an `exploratory only` lane into a claim-bearing
  result because its latest run happened to look good.

Current practical launch rule:

- claim-bearing compositional launches should currently center on
  `mixed-family gold-first`, `phase2`, and `hard`;
- the older starter compositional lanes should remain frozen as exploratory
  only;
- Lean4 should be read as sidecar support, not as a substitute trust boundary.

---

## 5. Snapshot conclusion

As of `2026-04-04`:

- the direct Hilbert trust-boundary hypothesis remains valid;
- the `ND -> Hilbert` front-end comparison remains valid;
- the canonical compositional theory line is now the `gold-first` family, not
  the older starter compositional lanes;
- no current active hypothesis needs to be marked `invalid`, but several older
  compositional surfaces remain `exploratory only`;
- several active and important items in the current package are not theory
  hypotheses at all and should not be reported as such.
