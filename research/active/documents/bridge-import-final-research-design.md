---
title: "Bridge Import Final Research — Experiment Design"
status: active
owner: platform / research
updated_at: "2026-04-04"
source_basis:
  - "research/active/documents/compositional-assumption-import-experiment-design.md"
  - "research/active/documents/proof-research-next-cycle-plan.md"
  - "research/README.md"
---

# Bridge Import Final Research — Experiment Design

## 1. Purpose

This lane isolates one narrower question than the older starter compositional
surface:

> if the model separately proves `A -> B` and `B -> C`, where does the
> transition break when it later tries to close `A -> C` through imported
> bridge lemmas?

The goal is not to prove general compositional reuse. The goal is to make the
`bridge/import -> final` transition legible enough that failures stop being a
single undifferentiated `score_bucket`.

## 2. Runnable surface

Repository-backed operator surface:

- `go -C platform-tooling run ./cmd/researchctl bridge-import-final-research plan`
- `go -C platform-tooling run ./cmd/researchctl bridge-import-final-research run`

Canonical files:

- case pack:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/bridge-import-final-research-phase1.csv`
- local profile:
  - `research/config/compositional/bridge/import-final-research.yaml`

The pack is intentionally small and layered:

- `b1`: prove the left bridge lemma, for example `A -> B`
- `b2`: prove the right bridge lemma, for example `B -> C`
- `fg`: final gold-import closure for `A -> C`
- `fm`: final model-import closure for `A -> C`
- `neg`: negative control

The lane uses the generic stage protocol rather than the older starter-only
`s1/s2/s3g/s3m/neg` naming.

## 2.1 Baseline `v1` interpretation guardrail

The current runnable `phase1` baseline remains preserved as historical
evidence. It should not be retroactively rewritten.

At the same time, its interpretation must now be narrower than the original
rough reading "everything breaks only at final imported composition".

The more precise evidence formula is:

> in compositional pipelines, failure may arise already at the bridge stage;
> final-model collapse can be a downstream consequence of earlier semantic
> transport failure rather than the only bottleneck.

The follow-on split lanes `bridge-only-authoring` and
`gold-final-composition-only` exist precisely because the preserved baseline
mixes bridge-stage authoring and final closure inside one compact pack.

## 3. Failure taxonomy

The chain-summary layer now classifies the first bridge/import/final break into
one of the following classes:

- `semantic_transport_failure`
  - a prerequisite bridge stage failed, so the transport chain never became
    semantically usable for final closure
- `import_binding_failure`
  - the final `model` or `semi_gold` stage did not bind all requested imported
    prerequisite stages
- `certificate_assembly_failure`
  - the final stage failed before a verifier-acceptable certificate could be
    assembled
- `final_proof_closure_failure`
  - the final stage assembled a candidate proof object but still failed at the
    proof-closure boundary (`kernel_failure`, `false_refusal`, or
    `false_accept`)

Negative-control failures remain explicit and are not silently folded into the
bridge/import/final taxonomy.

## 4. Chain-summary contract

For chains in this lane, `chain_result_<run-id>.csv` now carries:

- `failure_stage`
- `failure_stage_role`
- `failure_type`
- `failure_class`
- `failure_detail`

`failure_type` stays close to the existing `score_bucket` / missing-stage view.
`failure_class` is the new explanatory layer for the bridge/import/final
transition.

## 5. Intended read

This lane is useful only if the best models can now be separated into cases
such as:

- bridge stages are weak, so the failure is still upstream semantic transport
- bridge stages pass, but imported prerequisites are not bound cleanly
- imports bind, but certificate assembly is unstable
- imports bind and certificate assembly succeeds, but proof closure still fails

That decomposition is the whole point of the lane.

## 6. What this lane no longer licenses

After the preserved `phase1` baseline, repository-facing reporting should no
longer say only:

> the problem is final imported composition

That wording is too coarse for the current evidence.

The lane now supports only the more careful reading above: early bridge-stage
failure can already dominate the observed collapse, and `fg` / `fm` must be
read together with the bridge-stage rows rather than as an isolated final-only
story.
