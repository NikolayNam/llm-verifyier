# Proof-Theory Research Status Matrix

- **Title:** **Proof-Theory Research Status Matrix**
- **Status:** Active coordination note
- **Date:** 2026-03-27
- **Context:** CollabSphere / Hilbert benchmark / ND pilot / Lean4 sidecar / MEVP-EPCP research boundaries
- **Implementation authority:** Scope-and-status coordination only

---

## 1. Purpose

This document is a coordination guardrail for reading the current proof-theory
documents together.

It exists to reduce semantic overreach when multiple tracks share words such
as `kernel`, `certificate`, `proof object`, `runtime`, or `checker`.

It does not expand the claims of any underlying document. It only narrows how
those documents should be interpreted together.

---

## 2. Hard interpretation rules

When reading the current proof-theory package:

- do not upgrade `draft`, `pilot`, `research`, `open`, or `not yet confirmed`
  into a stronger claim;
- do not treat an implemented scaffold or repo entrypoint as an approved
  empirical result;
- do not treat MEVP / EPCP research language as an adopted repository runtime
  decision;
- do not merge the Hilbert benchmark certificate family with the MEVP / EPCP
  proof-certificate family unless a separate bridge document makes that
  relation explicit.

---

## 3. Status Matrix

Status snapshot as of `2026-03-27`:

| Track | Implementation authority | Confirmed now | Implemented now | Still unproven / not approved | Explicitly out of scope |
| --- | --- | --- | --- | --- | --- |
| Direct Hilbert benchmark (`v2 held-out`) | Approved empirical claim with bounded scope | The Hilbert-style checker is supported as a conservative verification boundary for the current bounded formal fragment. | Benchmark cases, runners, result summaries, and current `v1.3` comparison workflow exist in the repo. | Open-domain truth verification, verification of natural-language answers without a trusted formalization step, and stable theorem synthesis across weaker models. | Silent promotion from bounded benchmark evidence to general product/runtime law. |
| `ND -> Hilbert` pilot (`ND v1`) | Documentation/specification baseline for a paired research pilot | Only the prerequisite Hilbert-boundary findings are approved; the ND benefit claim is not. | `natural-deduction-v1`, schema validation, deterministic lowering, and `run-nd-hilbert-benchmark` / `make nd-hilbert-benchmark` as current provisional repo entrypoints. | Any claim that ND already improves performance, should replace the Hilbert kernel, or already defines the future runtime calculus. | Treating the current ND runner as a stabilized long-term interface or as evidence that sequent/runtime adoption is already approved. |
| Lean4 worker (`Lean4 Worker v1`) | Research scaffold authority with limited local confirmation | The local scaffold/bootstrap path and one positive `P -> P` smoke-check are confirmed; the approved trust boundary still comes from the Hilbert track, not from Lean. | Go worker, Lean project scaffold, case generation/export flows, and research-job entrypoints exist in the repo. | Broad theorem coverage, research benefit, general backlog confirmation via Lean, or Lean as a trust-bearing verifier. | Smoothing away local-environment caveats or promoting Lean artifacts into repository verdicts by themselves. |
| MEVP / EPCP `v0.2` | Architecture-review foundation draft | Terminology, layer separation, and proof-theoretic framing for review only. | Draft foundation, draft proof-object structures, and checker concepts exist only as research documentation. | Production rulebook, adopted runtime, checker implementation, and repository-level protocol authority. | Treating the draft as an already adopted runtime law or as equivalent to the Hilbert benchmark stack. |
| MEVP / EPCP `v0.3` | Research addendum / architecture direction only | The three-layer direction is a research recommendation, not an implemented or approved repository runtime. | Research roadmap, proposed checker boundary, and proposed deliverables are documented. | Sequent runtime adoption, checker implementation, cross-system certificate exchange, and any fixed bridge to `certificate-format-v1`. | Silent repo-wide refactor from current Hilbert/ND/Lean tracks into MEVP / EPCP runtime reality. |

---

## 4. Certificate Namespace Note

The current package uses the word `certificate` for more than one artifact
family.

Use the following distinction:

- **Hilbert benchmark certificate**
  - the `certificate-format-v1` artifact family used by the direct-Hilbert and
    `ND -> Hilbert` benchmark flows;
- **MEVP / EPCP proof certificate**
  - the protocol-level justification artifact family described in the
    `MEVP / EPCP v0.2` and `v0.3` research drafts.

Current repository rule:

- the relation between these two certificate families is **not yet fixed**;
- no current document grants automatic schema equivalence;
- no current document grants automatic checker equivalence;
- no current document grants automatic runtime unification.

Any future bridge between these families should be introduced by a separate
document that states:

- the mapping direction;
- the semantic preservation claim;
- the checker boundary; and
- the approval scope.

---

## 5. Usage Note

When a document in this area sounds stronger than the current package status,
interpret it through this matrix before making repository changes.
