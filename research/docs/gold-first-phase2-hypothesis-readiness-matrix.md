# Gold-First Phase2 Hypothesis Readiness Matrix

- **Title:** **Gold-First Phase2 Hypothesis Readiness Matrix**
- **Status:** Active synthesized research note
- **Date:** 2026-04-04
- **Scope:** `codex_research_backlog_gold_first_phase2_efficient.md`
- **Evidence base:** current `P0.2` / `P0.3` artifacts plus active hypothesis-validity guardrails

---

## 1. Purpose

This document gives a short launch-read matrix for the hypotheses and
experiment lines named in:

- [codex_research_backlog_gold_first_phase2_efficient.md](../active/documents/ru/codex_research_backlog_gold_first_phase2_efficient.md)

The goal is not to restate the backlog.

The goal is to answer a narrower operator question:

> Which backlog items are methodologically fit **right now**, which should be
> deferred, which are not theory hypotheses at all, and which should stay in
> exploratory territory?

This matrix is derived from:

- [gold_first_phase2_run_validity.md](../artifacts/analysis/gold_first_phase2_run_validity.md)
- [gold_first_phase2_subgroup_analysis.md](../artifacts/analysis/gold_first_phase2_subgroup_analysis.md)
- [proof-theory-hypothesis-validity-matrix.md](../active/matrix/proof-theory-hypothesis-validity-matrix.md)

For an operator-facing launch filter that can be used immediately before new
runs, see:

- [gold-first-phase2-pre-run-checklist.md](./gold-first-phase2-pre-run-checklist.md)

---

## 2. Column meanings

- `годится сейчас` — hypothesis is suitable for immediate claim-bearing or
  claim-adjacent research execution in the current cycle.
- `годится позже` — hypothesis is methodologically meaningful, but should be
  deferred until earlier cleaner steps stabilize.
- `не theory hypothesis` — this item is important, but it is an analysis,
  workflow, or robustness-control task rather than a proof-theory claim.
- `exploratory only` — current execution of this idea should not be read as
  clean claim-bearing evidence.

Only one primary column should be read as the classification for each row.

---

## 3. Short Matrix

| Backlog item | Core question | Годится сейчас | Годится позже | Не theory hypothesis | Exploratory only | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `P0.1` full reliability gate | Do we need another full-scale reliability rerun before proceeding? |  |  | yes |  | Closed as an operational decision, not a new theory claim. Current evidence already separates one `execution-invalidated` run from one `reasoning-valid` run. |
| `P0.2` post-hoc reliability classification | Can existing phase2 runs be separated into `reasoning-valid` and `execution-invalidated` under a stable rule-set? |  |  | yes |  | Implemented and useful, but this is an analysis/control layer. It is not a distinct proof-theory hypothesis. |
| `P0.3` subgroup analysis | Does existing valid phase2 data already show structured reuse-dependent variation? |  |  | yes |  | Implemented and highly useful. The underlying claim about structured dependence is supported, but the backlog item itself is still an analysis surface rather than a standalone theory hypothesis. |
| `P1.1` hard-transition micro-pack | Does the strong `gold-first` line survive on the hardest transition groups? | yes |  |  |  | Best immediate next experiment. `P0.3` already localizes hard zones (`GF-MF-D/E`, low overlap, higher bridge depth), so this is not a blind stress pack. |
| `P1.2` semi-gold single-generated bridge | Can one trusted gold bridge be replaced by one verifier-approved model-generated bridge without catastrophic collapse? |  | yes |  |  | Valid next-layer hypothesis, but it requires a new protocol surface (`model propose -> verifier accept -> trusted reuse`). Not the next immediate batch. |
| `P1.3` semi-gold multi-import merge | Can semi-gold reuse survive when multiple imports must be merged? |  | yes |  |  | Methodologically meaningful, but later than single-bridge. It compounds both protocol novelty and `import_arity >= 2` difficulty. |
| `P1.4` transformational reuse | Does the current signal survive beyond literal or near-literal reuse into more transformed transitions? |  | yes |  |  | Good later hypothesis. It is useful once semi-gold or harder reuse structure is stabilized, not before. |
| `P1.5` hard negative twins / adversarial near-miss controls | Does the conservative refusal boundary remain stable under harder negatives? |  | yes |  |  | Worth doing, but after the micro hard slice. Current phase2 negatives already look strong, so this is a robustness extension rather than the highest-priority next step. |
| `P2.6` prompt / protocol ablations | Is the observed strength robust to prompt/protocol variation? |  |  | yes |  | Important robustness work, but not a theory hypothesis. Use only after the main line is stabilized. |
| `P2.7` ND front-end variant | Does `ND -> Hilbert` improve the same `gold-first` line without violating the conservative boundary? |  | yes |  |  | Valid comparative hypothesis, but second-order. Direct `gold-first` should be stabilized on hard slices first. |
| `P2.8` back-transfer to starter lanes | Can `gold-first` lessons be transferred back into semantically weaker starter compositional lanes? |  |  |  | yes | Do not promote this into the current claim-bearing core. Starter lanes remain semantically compromised and are still frozen as exploratory-only surfaces. |

---

## 4. Working conclusion

Current execution priority should be:

1. `P1.1` hard-transition micro-pack
2. `P1.2` semi-gold single-generated bridge
3. `P1.3` semi-gold multi-import merge
4. `P2.7` ND front-end variant

Current support/control layers should continue to exist, but not be reported
as theory claims:

- `P0.1`
- `P0.2`
- `P0.3`
- `P2.6`

Current items that should stay out of the claim-bearing core:

- `P2.8` back-transfer to starter lanes

---

## 5. Evidence-linked notes

The current recommendation is not arbitrary.

It follows from the actual phase2-derived artifacts:

- [gold_first_phase2_run_validity.md](../artifacts/analysis/gold_first_phase2_run_validity.md)
  shows one run is cleanly `execution-invalidated` and one is cleanly
  `reasoning-valid`, so new expensive full reliability reruns are not the best
  next use of budget.
- [gold_first_phase2_subgroup_analysis.md](../artifacts/analysis/gold_first_phase2_subgroup_analysis.md)
  shows the strongest current immediate knowledge gain is in the hard
  transition zones, not in repeating broad easy slices.
- [proof-theory-hypothesis-validity-matrix.md](../active/matrix/proof-theory-hypothesis-validity-matrix.md)
  already constrains starter compositional lanes to `exploratory only`, which
  is why `P2.8` is intentionally not promoted into the current theory core.
