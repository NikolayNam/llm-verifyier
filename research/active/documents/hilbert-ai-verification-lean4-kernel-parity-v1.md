# Hilbert AI Verification Lean4 Kernel Parity v1

- **Title:** **Hilbert AI Verification Lean4 Kernel Parity v1**
- **Status:** Active research surface with local corpus-level parity confirmation
- **Date:** 2026-04-04
- **Context:** CollabSphere / minimal Hilbert certificate checker / Lean4 replicated kernel / fixed corpus comparison
- **Cross-track guardrail:** See [Proof-Theory Research Status Matrix](../matrix/proof-theory-research-status-matrix.md)

---

## 1. Purpose

This document defines the separate research surface for checking whether a
Lean4 replication of the repository’s minimal Hilbert checker produces the same
acceptance behavior as the current Go kernel on a fixed certificate corpus.

The intended question is narrow:

> If the same repository-tracked certificates are evaluated by the current Go
> kernel and by a replicated Lean4 checker, do the verdicts coincide case by
> case?

---

## 2. Why this surface is useful

This surface is useful because it isolates checker equivalence from broader
research questions.

It does **not** ask whether Lean4 is a better trusted boundary.

It asks whether the currently implemented Lean4 kernel replica behaves like the
existing mathematically constrained Go checker on the tested corpus.

If the verdicts differ, the interpretation is strict:

- there is either a bug in the Lean replica;
- or a bug in the Go checker;
- or a real semantic non-equivalence in one of the layers being mirrored.

---

## 3. Canonical method

The canonical method is:

1. Fix a repository-local corpus of certificates.
2. Evaluate every case with the current Go checker.
3. Evaluate every case with the Lean4 replicated checker.
4. Compare verdicts case-by-case.

Primary comparison key:

- `accepted`
- `class`
- `reason_code`

Secondary comparison key:

- exact `detail`

Current operator entrypoint:

```text
go -C platform-tooling run ./cmd/researchctl analyze lean-kernel-parity
```

For a larger deterministic sample from a bigger frozen raw corpus, the surface
also supports:

```text
go -C platform-tooling run ./cmd/researchctl analyze lean-kernel-parity --corpus-root <path> --corpus-limit 500
```

For larger corpora, Lean-side execution may be split into multiple generated
modules with:

```text
go -C platform-tooling run ./cmd/researchctl analyze lean-kernel-parity --corpus-root <path> --corpus-limit 500 --lean-batch-size 100
```

---

## 4. Current scope and limits

This `v1` surface currently covers:

- the fixed corpus under
  `platform-tooling/internal/certificates/testdata`
- the minimal Hilbert checker logic mirrored into Lean4
- a narrow raw-admission parity shim for the currently observed decoder drift
  patterns in the frozen larger corpus
- case-by-case comparison artifacts
- a compact mismatch artifact containing only non-matching cases

This `v1` surface does **not** yet claim parity for:

- malformed JSON decoding behavior;
- arbitrary unknown-field rejection behavior outside the currently enumerated
  corpus drift patterns;
- arbitrary future corpora not yet run through the surface;
- promotion of Lean4 into the operational trust boundary.

So the current result is a corpus-level kernel-equivalence result, not a trust
upgrade.

---

## 5. Current local result

The repository now contains two recorded local parity runs:

- `project_folder`
  - `hilbert-ai-verification-lean4-kernel-parity-v1`
- `run_id`
  - `lean-kernel-parity-smoke`
- `total_cases`
  - `19`
- `behavior_mismatches`
  - `0`
- `detail_mismatches`
  - `0`
- `run_id`
  - `lean-kernel-parity-500`
- `total_cases`
  - `500`
- `behavior_mismatches`
  - `0`
- `detail_mismatches`
  - `0`

Recorded artifact paths:

- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/comparison.csv`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/mismatch_manifest.csv`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/mismatch_cases/`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/mismatch_summary.md`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/report.md`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-500/kernel_parity/comparison.csv`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-500/kernel_parity/mismatch_manifest.csv`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-500/kernel_parity/mismatch_cases/`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-500/kernel_parity/mismatch_summary.md`
- `research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-500/kernel_parity/report.md`

Interpretation:

- on the currently fixed corpus, the replicated Lean4 checker matches the Go
  checker exactly at both behavior and detail level;
- on the current frozen `500`-case corpus slice, the same is also true after
  adding a narrow raw-admission shim for the observed decoder drift patterns;
- this is therefore stronger than kernel-only smoke evidence, but still weaker
  than a claim of general JSON decoder equivalence.

---

## 6. Correct interpretation of this result

This result is stronger than a scaffold-only claim.

It shows that, for the tested corpus, the Lean4 replicated checker is not just
syntactically runnable but operationally aligned with the current Go kernel.

But the result is still narrower than a trust-boundary change.

What the result supports:

- the Lean4 replica is a meaningful equivalence sidecar for the current corpus;
- future mismatches can be interpreted as concrete checker-drift evidence;
- the parity surface can now be used as a regression guard for kernel changes.

What the result does **not** support by itself:

- replacing Go with Lean4 in repository runtime paths;
- claiming broad parity over malformed input handling not represented in the
  fixed corpus;
- claiming that Lean4 has already become the approved verification authority.

---

## 7. Next useful follow-ons

The most useful next follow-ons are:

1. extend parity beyond the current fixed corpus to a larger frozen benchmark
   snapshot;
2. replace the current narrow raw-admission shim with a general decoder-parity
   surface for malformed JSON and unknown fields;
3. use this parity surface as a regression check whenever the Go kernel or the
   Lean replica changes.
