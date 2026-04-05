# Hilbert AI Verification Benchmark ND v1

- **Title:** **Hilbert AI Verification Benchmark ND v1**
- **Status:** Draft research benchmark with approved prerequisite findings
- **Implementation authority:** Documentation/specification baseline for a paired research pilot
- **Date:** 2026-03-26
- **Context:** CollabSphere / certificate subsystem / Natural Deduction front-end / deterministic lowering / Hilbert verification
- **Cross-track guardrail:** See [Proof-Theory Research Status Matrix](../matrix/proof-theory-research-status-matrix.md)
- **Normative modality:** **MUST / SHOULD / MAY** are used as benchmark protocol rules, not as product API guarantees

---

## 1. Purpose

This document defines the first research benchmark family for comparing:

- direct Hilbert-style certificate generation; and
- `Natural Deduction -> deterministic lowering -> Hilbert certificate`.

The question is not whether the Hilbert kernel should be replaced. The kernel
remains the final trust-bearing verifier. The question is whether a richer
front-end authoring discipline can improve end-to-end verified performance,
especially on theorem-synthesis and weaker-model cases.

Interpretation guardrails for this document:

- `Hilbert certificate` here means the `certificate-format-v1` benchmark
  artifact family, not the MEVP / EPCP proof-certificate family.
- `run-nd-hilbert-benchmark` and `make nd-hilbert-benchmark` are the current
  pilot entrypoints in the repo, not a stabilized long-term interface.
- This document defines the paired pilot baseline; it does not upgrade
  `ND -> Hilbert` into an approved empirical result.

---

## 2. Working hypothesis

The working hypothesis of this benchmark family is:

> Direct Hilbert lowering is not necessarily the best LLM-facing proof
> language. A Natural Deduction front-end with deterministic lowering into the
> current Hilbert certificate format may improve theorem synthesis, schema
> discipline, and weaker-model usability, while preserving the Hilbert kernel
> as the final trust boundary.

---

## 3. Explicit non-claims

This benchmark does **not** claim that:

- proof theory can repair false premises;
- the system turns arbitrary noisy inputs into truth;
- `ND -> Hilbert` is already validated;
- the Hilbert kernel should be replaced by ND;
- sequent calculus is part of the phase-1 implementation of this benchmark.

The intended improvement is narrower:

- better refusal discipline;
- better proof-shape discipline;
- better local inference validity;
- better end-to-end verifiability for some models.

---

## 4. Approved prerequisite findings from current Hilbert-direct research

The following findings are treated as approved prerequisites for this ND pilot.
They come from the current direct-Hilbert benchmark family and are **not**
re-opened by this document.

### 4.1 Approved result

The repository already contains strong evidence that a minimal Hilbert-style
certificate kernel functions as a conservative verification boundary for a
formalizable subset of AI-assisted reasoning outputs.

### 4.2 Current approved snapshot

As of `2026-03-26`, the latest valid `v2/v1.3` primary matrix for the
free-model slice shows:

- `compatible:gpt-oss:120b-cloud` — `47/52`
- `compatible:gpt-oss:20b` — `32/52`
- `compatible:deepseek-r1:14b` — `27/52`
- `compatible:deepseek-v3.1:671b-cloud` — `25/52`

Across the selected valid comparative runs in that matrix:

- `false_accept = 0`

### 4.3 Current approved interpretation

The approved interpretation of the current Hilbert-direct evidence is:

- the Hilbert-style kernel is supported as a conservative trust boundary;
- model-agnostic example-free lowering is not yet supported by the evidence;
- a richer front-end is worth testing because the main current pain points are
  theorem synthesis, schema drift, and weaker-model degradation rather than
  unsafe kernel acceptance.

Primary current benchmark references:

- [Hilbert AI Verification Benchmark v1](./hilbert-ai-verification-benchmark-v1.md)
- [Hilbert AI Verification Benchmark v2 Held-Out](./hilbert-ai-verification-benchmark-v2-held-out.md)

---

## 5. Relationship to current ADRs

This benchmark family is anchored to the following architecture decisions:

- [Keep the Certificate Kernel Minimal and Separate from the Operational Layer](../../../docs/architecture/adr/global/adr-certificate-kernel-operational-boundary.md)
- [Freeze the Certificate Kernel Rule Pack v1](../../../docs/architecture/adr/global/adr-freeze-certificate-kernel-rule-pack-v1.md)
- [Adopt Natural-Deduction Authoring as the First Proof-Theoretic Operational Front-End to the Hilbert Kernel](../../../docs/architecture/adr/global/adr-proof-theoretic-operational-layer-nd-to-hilbert.md)

This document exists because the current architecture already permits richer
authoring outside the kernel, but the repository did not yet define a concrete
ND pilot, a front-end contract, or a shared benchmark pack for honest direct
comparison.

---

## 6. Canonical files

The canonical ND v1 benchmark files are:

- `research/active/documents/hilbert-ai-verification-benchmark-nd-v1.md`
- `research/active/documents/proof-research-phase2-runbook.md`
- `docs/technical-specs/platform/natural-deduction-v1.md`
- `docs/contracts/schemas/natural-deduction.schema.json`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/result_summary_v2.csv`
- `platform-tooling/internal/prooftheory/naturaldeduction/`
- `scripts/run-nd-hilbert-benchmark.sh`

The repository currently contains the following pilot artifacts and
provisional repo entrypoints:

- the canonical `natural-deduction-v1` front-end contract;
- a structural validator for ND proof objects;
- a deterministic `ND -> Hilbert` lowering layer for the phase-1 implicational
  fragment;
- a runnable `run-nd-hilbert-benchmark` contracts command and `make
  nd-hilbert-benchmark` wrapper as the current provisional paired-pilot
  entrypoint.

The family is still a pilot. The existence of a runner and a lowerer does not
mean that the ND front-end is already empirically validated.

---

## 7. Exact pilot scope

Phase 1 is deliberately restricted to:

- the implicational propositional fragment;
- the `natural-deduction-v1` front-end contract;
- deterministic lowering into `certificate-format-v1`;
- end-to-end final verification by the current Hilbert kernel.

Phase 1 does **not** include:

- negation;
- classical-only ND rules;
- contradiction introduction or elimination;
- sequent proof objects;
- LLM-based lowering or repair loops.

---

## 8. Shared pilot pack

The canonical pilot theorem pack is:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/historical/theorems_pilot_shared_20260327.csv`

Staged-migration note:

- this ND family now treats the theorem pack above as the canonical input;
- theorem-oriented packs now carry a canonical `theorem_pack_id` and
  `theorem_id`;
- `theorems/historical/` preserves immutable historical theorem-pack snapshots
  under the naming rule `theorems_<theorem_pack_id>.csv`;
- legacy `cases.csv` is retained as a compatibility alias for existing
  summaries and older local commands;
- new theorem-oriented summary writes now target
  `result/result_summary_v2.csv`;
- `result/result_summary.csv` is retained as a legacy compatibility summary
  source for older ND rows;
- repository-wide `v1/v2` Hilbert families are intentionally not renamed by
  this document.

This pack is deliberately shared across both pipelines:

- `Hilbert-direct`
- `ND -> Hilbert`

The theorems are front-end agnostic. The final outcome must be judged by the same
Hilbert checker in both pipelines.

### 8.3 Paired execution rule

The same pilot pack is intended to be executed through both:

- `make hilbert-benchmark`
- `make nd-hilbert-benchmark`

with the same:

- model;
- provider;
- endpoint;
- timeout;
- selected theorem pack.

This is the canonical paired-execution basis for comparing `Hilbert-direct`
against `ND -> Hilbert`.

### 8.1 Theorem-pack schema

```csv
theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment
```

Schema note:

- `theorem_id` is the canonical theorem-row identifier for theorem-oriented
  research families;
- `case_id` remains as a compatibility alias for existing benchmark internals
  and historical tooling;
- `theorem_pack_id` identifies the theorem-pack version used in the run and
  enables historical replay and comparison across theorem-pack revisions.

### 8.2 Initial pilot cases

```csv
theorem_pack_id,theorem_id,case_id,category,label,difficulty,assumptions_json,goal,logic_fragment,comment
pilot_shared_20260327,NDI01,NDI01,assumption_import,entailed,easy,"[""P""]",P,implicational-prop-v1,Direct premise import
pilot_shared_20260327,NDI02,NDI02,single_mp,entailed,easy,"[""P"",""P -> Q""]",Q,implicational-prop-v1,Single implication elimination
pilot_shared_20260327,NDI03,NDI03,mp_chain,entailed,medium,"[""P"",""P -> Q"",""Q -> R""]",R,implicational-prop-v1,Two-step implication chain
pilot_shared_20260327,NDI04,NDI04,nested_imp_intro,entailed,medium,"[""P""]","Q -> P",implicational-prop-v1,Introduce implication under nested assumption
pilot_shared_20260327,NDI05,NDI05,theorem_synthesis,entailed,medium,[],"P -> P",implicational-prop-v1,Identity theorem without premises
pilot_shared_20260327,NDI06,NDI06,theorem_synthesis,entailed,hard,[],"(P -> (Q -> R)) -> ((P -> Q) -> (P -> R))",implicational-prop-v1,Composition theorem in implicational fragment
pilot_shared_20260327,NDI07,NDI07,negative_refusal,not_entailed,easy,"[""P -> Q""]",P,implicational-prop-v1,Consequent does not imply antecedent
pilot_shared_20260327,NDI08,NDI08,negative_refusal,not_entailed,easy,"[""P""]",Q,implicational-prop-v1,Irrelevant atomic goal should be refused
```

---

## 9. Exact comparison protocol

### Pipeline A

- `LLM -> certificate-format-v1 -> Hilbert checker`

### Pipeline B

- `LLM -> natural-deduction-v1 -> deterministic lowering -> certificate-format-v1 -> Hilbert checker`

### Fairness rules

The comparison must keep all of the following equal:

- model;
- theorem pack;
- timeout;
- endpoint/provider;
- final verdict source;
- no repair loop for either pipeline.

### 9.1 Current provisional paired-pilot entrypoints

Direct Hilbert:

```bash
make hilbert-benchmark \
  HILBERT_BENCHMARK_LLM_PROVIDER=compatible \
  HILBERT_BENCHMARK_BASE_URL=http://localhost:11434 \
  HILBERT_BENCHMARK_MODEL='gpt-oss:20b' \
  HILBERT_BENCHMARK_PROJECT_FOLDER='hilbert-ai-verification-benchmark-nd-v1' \
  HILBERT_BENCHMARK_THEOREMS_FILE='theorems/pilot_shared_20260327.csv' \
  HILBERT_BENCHMARK_PROMPT_VERSION='hilbert-ai-verification-benchmark-v1.3'
```

ND front-end with deterministic lowering:

```bash
make nd-hilbert-benchmark \
  HILBERT_BENCHMARK_LLM_PROVIDER=compatible \
  HILBERT_BENCHMARK_BASE_URL=http://localhost:11434 \
  HILBERT_BENCHMARK_MODEL='gpt-oss:20b' \
  HILBERT_BENCHMARK_PROJECT_FOLDER='hilbert-ai-verification-benchmark-nd-v1' \
  HILBERT_BENCHMARK_THEOREMS_FILE='theorems/pilot_shared_20260327.csv' \
  HILBERT_BENCHMARK_PROMPT_VERSION='hilbert-ai-verification-benchmark-nd-v1.1'
```

These commands are the current repo entrypoints for the paired pilot. They are
not yet the stabilized long-term interface for ND benchmarking.

Prompt-version note:

- `hilbert-ai-verification-benchmark-nd-v1` remains the baseline prompt used
  in the first paired pilot result;
- `hilbert-ai-verification-benchmark-nd-v1.1` is the next engineering-tightened
  ND prompt and explicitly strengthens the required handling of `id`, `from`,
  `scope`, and `discharge_scope`;
- the next paired rerun should keep direct Hilbert on
  `hilbert-ai-verification-benchmark-v1.3` and move the ND side to
  `hilbert-ai-verification-benchmark-nd-v1.1`.

Convenience report targets for the current dated rerun slice:

- `make hilbert-benchmark-rerun-direct HILBERT_BENCHMARK_LLM_PROVIDER=... HILBERT_BENCHMARK_BASE_URL=... HILBERT_BENCHMARK_MODEL=...`
- `make hilbert-benchmark-rerun-nd HILBERT_BENCHMARK_LLM_PROVIDER=... HILBERT_BENCHMARK_BASE_URL=... HILBERT_BENCHMARK_MODEL=...`
- `make hilbert-benchmark-report-direct`
- `make hilbert-benchmark-report-nd`
- `make hilbert-benchmark-research-report-pair`

Those wrappers write to the current dated theorem-oriented report outputs under:

- `research/result_research_report/direct/hilbert-ai-verification-benchmark-nd-v1_report_direct_20260327.md`
- `research/result_research_report/nd/hilbert-ai-verification-benchmark-nd-v1_report_nd_20260327.md`
- `research/result_research_report/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_pair_20260327.md`

The research-report target reads the summary layer rather than raw proof files.
It is intended to produce a narrative research interpretation from one or more
summary sources without replacing the lower-level audit artifacts.

The theorem-pack Markdown report layer now also renders observed-result slices
from the deduplicated per-run result files:

- aggregate
- entailed-only
- not-entailed-only
- theorem-synthesis focus
- foundational-category audit for `assumption_import` and `single_mp`
- per-category
- ND pipeline trace
- per-case hardest failures

The paired research report now surfaces the same aggregate and observed-slice
structure instead of stopping at pair findings alone.

Current audit-target note:

- unlike the `v2 held-out` direct benchmark family, this ND theorem-pack family
  does not yet define a canonical fixed hard-case audit target list;
- the report therefore keeps the hard-case audit section for format parity, but
  it will currently report that no fixed audit targets are available.

Legacy ND per-run result CSVs that predate explicit pipeline-stage telemetry
are still loadable. Their `ND pipeline trace` rows are classified heuristically
from `raw_output_kind` and the recorded schema/parse/kernel statuses, so those
counts should be read as inferred legacy staging rather than first-class stage
logs.

The paired rerun summaries themselves are written under the shared artifacts
directory:

- `research/artifacts/result_research/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_v2_rerun_direct_20260327.csv`
- `research/artifacts/result_research/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_v2_rerun_nd_20260327.csv`

The shared research registry for theorem/result imports is:

- `research/artifacts/result_research/research_db/research_db.sqlite`

Operational note:

- the two `report-*` targets expect the corresponding rerun summary file to
  exist already;
- if the rerun summary is missing, run the matching `rerun-*` target first.

### 9.3 Tightened-prompt rerun snapshot for `gpt-oss:120b-cloud`

The next engineering-tightened ND rerun was executed on the shared theorem pack
`theorems/pilot_shared_20260327.csv` with:

- direct Hilbert on `hilbert-ai-verification-benchmark-v1.3`
- ND on `hilbert-ai-verification-benchmark-nd-v1.1`

Recorded outputs:

- direct summary:
  `research/artifacts/result_research/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_v2_rerun_direct_gpt-oss-120b-cloud_20260327.csv`
- ND summary:
  `research/artifacts/result_research/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_v2_rerun_nd_gpt-oss-120b-cloud_v11_20260327.csv`
- paired research report:
  `research/result_research_report/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_pair_gpt-oss-120b-cloud_20260327.md`

Observed result:

- direct Hilbert: `5/8` verified pass (`62.50%`)
- `ND -> Hilbert` with the tightened `v1.1` prompt: `7/8` verified pass
  (`87.50%`)
- `false_accept = 0` in both runs

Interpretation:

- this is a meaningful positive signal that prompt-tightened ND authoring may
  improve end-to-end verified performance for a stronger model on the current
  pilot theorem pack;
- this does **not** yet establish a broad model-agnostic conclusion, because
  the evidence is still limited to one model and one theorem pack.

Compatibility note:

- the runtime still reuses generic benchmark internals that originated in
  `cases_*` naming;
- the canonical ND summary schema is now theorem-oriented and writes
  `theorems_file` / `theorems_path` into
  `result/result_summary_v2.csv`;
- shared summary artifacts that are intentionally kept outside the family-local
  `result/` tree are now split under:
  - `research/artifacts/result_research/direct/`
  - `research/artifacts/result_research/nd/`
  - `research/artifacts/result_research/pair/`
  - `research/artifacts/result_research/waves/`;
- those generated theorem/result artifacts can be imported into the SQLite
  registry at
  `research/artifacts/result_research/research_db/research_db.sqlite`
  through `go -C platform-tooling run ./cmd/contracts research-db-sync -project-folder hilbert-ai-verification-benchmark-nd-v1`;
- legacy `result/result_summary.csv` is treated as a compatibility source and
  is not the append target for new ND runs.

### 9.2 Exact theorem-oriented summary schema

The canonical ND theorem-oriented summary schema is:

```csv
run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,theorem_pack_id,theorems_file,case_selector,hypothesis,theorems_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
```

Notes:

- `cases_total` remains the current wire name for the count column to avoid a
  wider historical schema fork;
- `result_summary.csv` remains loadable as a legacy compatibility source for
  ND historical rows whose input metadata was recorded under `cases_file` /
  `cases_path`;
- legacy repository benchmark families `v1` and `v2 held-out` remain on their
  original `result_summary.csv` schema and are not modified by this document.

---

## 10. Primary metrics

The pilot should record at minimum:

- `end_to_end_verified_pass_rate`
- `false_accept_count`
- `false_refusal_count`
- `front_end_schema_failure`
- `lowering_failure`
- `hilbert_kernel_failure`
- `avg_latency_ms`
- `proof_length_blowup_ratio`

`proof_length_blowup_ratio` means:

```text
hilbert_steps / nd_steps
```

for successfully lowered proofs.

---

## 11. Success criteria for the pilot

The ND pilot should be treated as promising only if:

- `false_accept` remains `0`;
- `ND -> Hilbert` improves verified pass rate or refusal discipline relative to
  direct Hilbert generation;
- improvements are visible especially on:
  - theorem synthesis;
  - weaker models;
  - schema / format drift cases.

If pass rate rises but unsafe acceptances also rise, the pilot must be treated
as a failure for trust-boundary purposes.

---

## 12. Relationship to future work

This benchmark family does not reject richer proof-theoretic front-ends. It
only establishes the first one.

Future-compatible directions may include:

- sequent-style operational proof objects;
- broader logical fragments;
- richer deterministic lowering layers.
- a Lean4/mathlib proof-specification sidecar that generates formal case
  hypotheses and persists mechanized verification artifacts.

Those directions are out of scope for `ND v1` unless separately specified and
benchmarked.

### 12.1 Related research

The next adjacent research layer is:

- [Hilbert AI Verification Lean4 Worker v1](./hilbert-ai-verification-lean4-worker-v1.md)

That worker does not replace either `Hilbert-direct` or `ND -> Hilbert`. It
adds a specialized `.lean` mechanized sidecar for proof/specification
experiments, formal case-hypothesis generation, and future Lean-to-benchmark
artifact export.


