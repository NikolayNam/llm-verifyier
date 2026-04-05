# Hilbert AI Verification Prompt Registry

- **Status:** Active research prompt registry
- **Scope:** direct Hilbert benchmark prompts, ND benchmark prompts, adjacent Lean4 note
- **Date:** 2026-03-27

---

## 1. Purpose

This document separates prompt-family tracking from the main research
documents. The canonical research writeups live under
`research/active/documents/`, while this file tracks which prompt
profiles exist and what role they play in the current experiments.

---

## 2. Direct Hilbert prompt family

Research family:

- [Hilbert AI Verification Benchmark v1](../documents/hilbert-ai-verification-benchmark-v1.md)
- [Hilbert AI Verification Benchmark v2 Held-Out](../documents/hilbert-ai-verification-benchmark-v2-held-out.md)

Implementation location:

- `platform-tooling/cmd/contracts/hilbert_benchmark.go`

Current known prompt profiles:

- `hilbert-ai-verification-benchmark-v1.1`
  - early benchmark prompt family baseline
- `hilbert-ai-verification-benchmark-v1.2`
  - current engineering-best-performance snapshot
- `hilbert-ai-verification-benchmark-v1.3`
  - current methodologically-clean evaluation baseline without inline examples

Current interpretation:

- use `v1.2` when measuring best achieved engineering behavior;
- use `v1.3` when making cleaner generalization claims.

---

## 3. ND -> Hilbert prompt family

Research family:

- [Hilbert AI Verification Benchmark ND v1](../documents/hilbert-ai-verification-benchmark-nd-v1.md)

Implementation location:

- `platform-tooling/cmd/contracts/nd_hilbert_benchmark.go`

Current known prompt profiles:

- `hilbert-ai-verification-benchmark-nd-v1`
  - first paired-pilot ND prompt baseline
- `hilbert-ai-verification-benchmark-nd-v1.1`
  - tightened ND prompt that explicitly strengthens `id`, `from`, `scope`,
    and `discharge_scope`

Current interpretation:

- `nd-v1` is the first paired-pilot baseline;
- `nd-v1.1` is the next engineering-tightened rerun profile.

---

## 4. Lean4 note

The Lean4 worker currently does not define a benchmark prompt family in the
same sense as the direct Hilbert and ND benchmark runners.

Related research family:

- [Hilbert AI Verification Lean4 Worker v1](../documents/hilbert-ai-verification-lean4-worker-v1.md)

Current mode families there are:

- `enumerator`
- `curated_backlog`
- `model_proposed`

These are theorem-generation modes, not LLM benchmark prompt versions.

