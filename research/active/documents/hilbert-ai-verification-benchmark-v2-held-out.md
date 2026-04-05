# Hilbert AI Verification Benchmark v2 Held-Out

- **Title:** **Hilbert AI Verification Benchmark v2 Held-Out**
- **Status:** Draft research benchmark
- **Implementation authority:** Approved empirical claim with bounded scope
- **Date:** 2026-03-26
- **Context:** CollabSphere / certificate subsystem / LLM lowering / held-out generalization
- **Cross-track guardrail:** See [Proof-Theory Research Status Matrix](../matrix/proof-theory-research-status-matrix.md)
- **Normative modality:** **MUST / SHOULD / MAY** are used as benchmark protocol rules, not as product API guarantees

---

## 1. Purpose

This document defines the held-out benchmark family for testing a stronger
research hypothesis than benchmark v1:

> A model that succeeds on the current Hilbert-style certificate benchmark can
> generalize to unseen formal cases, rather than only following prompt-aligned
> examples.

This benchmark family is explicitly designed to distinguish:

- contract following;
- prompt-specific imitation;
- genuine generalization to unseen formal cases.

This document confirms only the direct-Hilbert benchmark boundary within its
current bounded formal fragment. It does not approve `ND -> Hilbert`, the
Lean4 sidecar, or any MEVP / EPCP runtime decision.

---

## 2. Canonical data files

The canonical v2 held-out files are:

- `research/active/documents/hilbert-ai-verification-benchmark-v2-held-out.md`
- `research/active/documents/proof-research-phase1-runbook.md`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/model-catalog.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/results.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_<run-id>.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_summary.csv`
- optional raw outputs under:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/<run-id>/`

New runs SHOULD use the historical layout under `result/`, while the flat
`results.csv` file MAY continue to exist as a compatibility log.

The same summary reader MAY be used for v2 held-out by pointing it at the v2
summary file:

```text
make hilbert-benchmark-summary HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v2-held-out HILBERT_BENCHMARK_CASES_FILE=cases.csv HILBERT_BENCHMARK_SUMMARY=research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_summary.csv
```

To aggregate all held-out case packs in the project while keeping separate
model groups:

```text
make hilbert-benchmark-summary HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v2-held-out HILBERT_BENCHMARK_CASES_FILE=*
```

---

## 3. Exact `cases.csv` schema

The canonical v2 held-out `cases.csv` schema is:

```csv
case_id,category,label,difficulty,assumptions_json,goal,comment
```

### Column meanings

- `case_id`
  - stable benchmark case identifier
- `category`
  - one of:
    - `assumption_import`
    - `direct_axiom_instance`
    - `mixed_proof`
    - `theorem_synthesis`
    - `negative_refusal`
- `label`
  - one of:
    - `entailed`
    - `not_entailed`
- `difficulty`
  - free benchmark difficulty marker such as `easy`, `medium`, `hard`
- `assumptions_json`
  - JSON array of top-level assumptions
- `goal`
  - target formula string
- `comment`
  - short human-readable case note

`category` is mandatory for v2 held-out.

---

## 4. Exact `results.csv` schema

The canonical v2 held-out `results.csv` schema is:

```csv
run_id,timestamp_utc,model,llm_model,experiment_name,experiment_id,sampling_surface,requested_sampling_profile,effective_sampling_profile,requested_sampling_json,effective_sampling_json,unsupported_sampling_json,prompt_version,surface,case_id,category,expected_label,raw_output_kind,schema_status,parse_status,kernel_status,cli_exit_code,score_bucket,latency_ms,notes
```

### Column meanings

- `run_id`
  - stable identifier for one benchmark run
- `timestamp_utc`
  - RFC3339 timestamp for the row
- `model`
  - provider-prefixed model label
- `llm_model`
  - raw model id such as `gpt-oss:20b`
- `experiment_name`
  - human-readable experimental setup label configured at the family layer
- `experiment_id`
  - stable experiment identity derived from `experiment_name`, repeat
    sequence, local run date, and the requested sampling profile
- `sampling_surface`
  - the concrete request surface actually used by the runner, such as
    `compatible_openai_chat_completions`
- `requested_sampling_profile`
  - compact identifier for the requested family-level sampling tuple
- `effective_sampling_profile`
  - compact identifier for the sampling tuple actually applied on that surface
- `requested_sampling_json`
  - JSON object describing the family-level sampling request
- `effective_sampling_json`
  - JSON object describing the sampling fields actually sent on the concrete
    provider surface
- `unsupported_sampling_json`
  - JSON object for requested fields that were not supported or not
    documented on the concrete request surface
- `prompt_version`
  - benchmark prompt profile identifier
- `surface`
  - benchmark surface identifier such as `direct`
- `case_id`
  - stable benchmark case identifier
- `category`
  - same category value from `cases.csv`
- `expected_label`
  - copied from the benchmark case
- `raw_output_kind`
  - one of:
    - `certificate`
    - `not_derivable`
    - `invalid_json`
    - `other`
- `schema_status`
  - one of:
    - `pass`
    - `fail`
    - `not_run`
- `parse_status`
  - one of:
    - `pass`
    - `fail`
    - `not_run`
- `kernel_status`
  - one of:
    - `accept`
    - `reject`
    - `not_run`
- `cli_exit_code`
  - canonical `hilbertcheck` exit code when applicable
- `score_bucket`
  - one of:
    - `pass`
    - `false_refusal`
    - `false_accept`
    - `request_failure`
    - `schema_failure`
    - `parse_failure`
    - `kernel_failure`
    - `format_failure`
- `latency_ms`
  - request latency in milliseconds
- `notes`
  - sanitized diagnostic text

## 4.1 Exact `result_summary.csv` schema

The canonical v2 held-out `result_summary.csv` schema is:

```csv
run_id,timestamp_utc,model,llm_model,experiment_name,experiment_id,sampling_surface,requested_sampling_profile,effective_sampling_profile,requested_sampling_json,effective_sampling_json,unsupported_sampling_json,prompt_version,certificate_version,project_folder,surface,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
```

`run_id` is mandatory and identifies:

- the per-run result file `result/result_<run-id>.csv`
- the raw directory `raw/<run-id>/...`

`avg_latency_ms`, `max_latency_ms`, and `run_elapsed_seconds` capture the
time profile of that run:

- average per-case LLM latency
- max per-case LLM latency
- total wall-clock run duration

### 4.1.1 Sampling provenance and experiment identity

Benchmark provenance MUST distinguish:

- what the operator requested at the family level;
- what the concrete provider surface actually received.

The canonical family-level config now lives in research YAML:

```yaml
families:
  local-compatible:
    transport: local-compatible
    experiment_name: formal-mode
    sampling:
      temperature: 0
      seed: 42
      top_p: 1
```

Interpret the result columns as follows:

- `requested_sampling_json`
  - the exact family-level request, for example
    `{"temperature":0,"seed":42,"top_p":1}`
- `effective_sampling_json`
  - the request tuple actually passed through the concrete client surface
- `unsupported_sampling_json`
  - requested fields that the concrete surface did not apply

This split is required for honest reproducibility. The repository MUST NOT
pretend that a field such as `seed` was part of the effective wire contract
when it was only requested but not actually passed through the client.

Changing the requested sampling tuple MUST create a distinct experiment
identity. The canonical derived `experiment_id` format is:

```text
<experiment-name>-<requested-sampling-profile>-rNN-YYYYMMDD
```

Example:

```text
formal-mode-temp0-seed42-topp1-r01-20260402
```

This means that two runs with the same model and prompt version but different
requested sampling parameters MUST be treated as different experiments rather
than silently aggregated together.

`cases_file=*` is a special summary-reader mode. It means:

- filter only by `project_folder`;
- include all case packs recorded in that project's `result_summary.csv`;
- still aggregate separately by model and prompt profile.

For a Markdown comparison report by case-pack, use:

```text
make hilbert-benchmark-report HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v2-held-out HILBERT_BENCHMARK_CASES_FILE=*
```

To write the Markdown report to a file instead of stdout:

```text
make hilbert-benchmark-report HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v2-held-out HILBERT_BENCHMARK_CASES_FILE=* HILBERT_BENCHMARK_REPORT_OUT=research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/model-comparison.md
```

For the canonical `Phase 1` rerun lane, the operator target currently uses:

- 8 reruns per model by default
- per-model shared summary CSV files under `research/artifacts/result_research/direct/`
- Markdown reporting with explicit sections for:
  - aggregate
  - entailed-only
  - not-entailed-only
  - per-category
  - per-case hardest failures
  - hard-case audit targets

## 4.2 Operational timeout short-circuit

The direct Hilbert benchmark runner MAY be configured with
`request-timeout-abort-threshold = N`.

When this threshold is enabled and a single run records `N` consecutive
request-layer timeouts:

- the runner MUST stop issuing new LLM requests for that run;
- the remaining selected cases MUST still be written into the result CSV as
  `request_failure`;
- the remaining raw files MUST be materialized with a short-circuit timeout
  note so the run remains auditable;
- the run MUST still remain historical evidence and MUST still be treated
  under the invalid-run policy from section `7.3` when
  `request_failure_count == cases_total`.

`0` disables this safeguard and preserves the fully eager per-case request
loop.

The benchmark-level default for this safeguard SHOULD live in research YAML
config:

- `benchmarks.phase1.request_timeout_abort_threshold`
- `benchmarks.nd.request_timeout_abort_threshold`

Interrupted-run handling SHOULD also live in research YAML config:

- `benchmarks.phase1.interrupt_policy`
- `benchmarks.nd.interrupt_policy`

Allowed values:

- `keep`
- `drop_if_no_results`
- `drop_always`

The canonical default is `drop_if_no_results`.

`researchctl` CLI flags MAY override the config value per run. When the flag
is omitted, the resolved benchmark default MUST be written into the manifest
and used for all child jobs. Paired workflows such as `phase2` and
`phase3 compare` MUST resolve the threshold per child lane:

- direct lanes inherit the `phase1` default unless explicitly overridden;
- ND lanes inherit the `nd` default unless explicitly overridden.

Operational interpretation:

- interrupted runs with no completed benchmark outputs SHOULD be dropped from
  the control-plane together with run-scoped artifacts;
- interrupted runs with meaningful outputs SHOULD remain visible as
  `status=aborted` and SHOULD NOT be silently rewritten as `failed` or left in
  `running`.

## 4.3 Canonical model catalog

The canonical model catalog for this benchmark family is:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/model-catalog.csv`

The catalog schema is:

```csv
llm_model,display_name,size_bucket,size_label,tier,comparison_status,notes
```

The catalog is used only for comparative interpretation and cohort slicing. It
MUST NOT rewrite historical benchmark rows in `result_summary.csv`.

## 4.2 Held-out hypothesis case packs

Additional local held-out case packs MAY be stored under:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases/`

This directory is intended for hypothesis-specific held-out CSV files such as:

- `cases/generalization-strict.csv`
- `cases/cross-model-sanity.csv`
- `cases/theorem-synthesis-stress.csv`
- `cases/negative-refusal-sanity.csv`
- `cases/mixed-proof-deep-stress.csv`
- `cases/theorem-synthesis-deep-stress.csv`
- `cases/assumption-import-canary.csv`
- `cases/entailed-gap-audit.csv`
- `cases/direct-axiom-to-mixed-proof-ladder.csv`

Repository-local next-cycle diagnostic packs currently materialized in this
workspace are:

- `cases/cross-model-sanity.csv`
- `cases/assumption-import-canary.csv`
- `cases/entailed-gap-audit.csv`
- `cases/negative-refusal-sanity.csv`
- `cases/direct-axiom-to-mixed-proof-ladder.csv`
- `cases/theorem-synthesis-deep-stress.csv`

These packs are operator-defined auxiliary lanes for the active next-cycle
research plan. They are not a replacement for the canonical full held-out
`cases.csv`.

Any CSV file with the v2 held-out schema may be passed via:

- `-project-folder hilbert-ai-verification-benchmark-v2-held-out`
- `-cases-file cases/<file>.csv`
- `-cases <path>`
- `HILBERT_BENCHMARK_CASES=<path>`

---

## 5. Category metrics

The canonical runner for v2 held-out MUST summarize results by category.

At minimum, each category summary should include:

- `cases_total`
- `pass`
- `false_refusal`
- `false_accept`
- `request_failure`
- `schema_failure`
- `parse_failure`
- `kernel_failure`
- `format_failure`

This is required because a model may perform well on:

- `assumption_import`
- `direct_axiom_instance`
- `negative_refusal`

while still failing on:

- `theorem_synthesis`
- `mixed_proof`

Category-level measurement is therefore part of the benchmark contract, not an
optional reporting convenience.

---

## 6. Compatibility rule

The runner must remain backward compatible with benchmark v1 results.

This means:

- existing v1 `results.csv` files without `category` MUST remain valid;
- existing v1 historical rows MUST NOT be rewritten;
- v2 held-out may use the extended results schema with `category`.

---

## 7. Final protocol statement

Benchmark v2 held-out is the canonical next-step protocol for testing
generalization beyond benchmark-aligned prompting. It extends the benchmark
data contract with mandatory `category` classification and category-level
summary metrics.

### 7.1 Research policy for `v1.2` vs `v1.3`

The current benchmark family MUST distinguish two different research roles:

- `hilbert-ai-verification-benchmark-v1.2`
  - treat as **engineering-best-performance**
  - use when the goal is to measure the best currently achieved contract
    retention and pass rate under the known prompt stack
- `hilbert-ai-verification-benchmark-v1.3`
  - treat as **methodologically-clean evaluation baseline**
  - use when the goal is to reduce benchmark-aligned shape imitation and make
    held-out generalization claims more defensible

Main comparative conclusions about model quality in v2 held-out SHOULD be based
on `v2/v1.3`, not on legacy `v1` runs.

### 7.2 Model cohort policy

For the current free-model research slice, the canonical size buckets are:

- `<=20b`
- `>20b`
- `unknown`

For this research round:

- `gpt-oss:20b` MUST be treated as `<=20b`
- `glm-5:cloud` MUST remain `unknown` unless its size is independently
  confirmed

The current canonical catalog also distinguishes:

- `primary`
  - eligible for main cohort comparison once a valid `v2/v1.3` run exists
- `insufficient-evidence`
  - kept in the catalog, but excluded from the main quantitative conclusion
    until a valid `v2/v1.3` run exists
- `appendix`
  - kept outside the size-based conclusion, even if operationally useful

### 7.3 Valid comparative run policy

For research comparison tables, a run is **comparatively valid** only if:

- `request_failure_count < cases_total`
- the run is not a pure `model not found` outcome
- the run is not a pure transport or endpoint failure before meaningful
  inference

Runs with:

- `request_failure_count == cases_total`
- or diagnostics indicating `model not found`
- or diagnostics indicating endpoint failure before meaningful inference

MUST remain in `result_summary.csv` and historical reports, but MUST be
excluded from the main cohort comparison and final research claims.

### 7.4 Canonical `v1.3` matrix for the free-model slice

The primary `v1.3` comparison matrix for this research round is:

- `deepseek-r1:14b`
- `gpt-oss:20b`
- `gpt-oss:120b-cloud`
- `deepseek-v3.1:671b-cloud`

The minimum case packs for this matrix are:

- `cases.csv`
- `cases/cross-model-sanity.csv`
- `cases/mixed-proof-deep-stress.csv`
- `cases/theorem-synthesis-deep-stress.csv`

Secondary appendix entries currently include:

- `qwen3:12b`
- `glm-5:cloud`

### 7.5 Premium-tier separation

Current free/current-cloud models MUST be analyzed separately from future
premium-tier experiments. Future runs against `ChatGPT-5 API` or other paid
models SHOULD be treated as a separate research slice and MUST NOT be silently
mixed into the current free-model conclusion.

### 7.6 Related next-step research

The current direct-Hilbert benchmark family does not settle whether direct
Hilbert generation is the best LLM-facing proof language. It only establishes
that the Hilbert kernel itself is already useful as a conservative verification
boundary.

The next related research step is:

- [Hilbert AI Verification Benchmark ND v1](./hilbert-ai-verification-benchmark-nd-v1.md)

That family treats `Natural Deduction -> deterministic lowering -> Hilbert
certificate` as the next front-end experiment. It is a follow-on research
direction, not a replacement for the already approved Hilbert boundary result
recorded in this document.

---

## 8. Conclusion

The current v2 held-out evidence supports a narrow positive research
conclusion:

> A minimal Hilbert-style certificate kernel can function as a practical trust
> boundary for a formalizable subset of AI-assisted reasoning outputs.

As of `2026-03-26`, the strongest valid runs recorded in:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_summary.csv`

include:

- `compatible:gpt-oss:20b` on full held-out `cases.csv`: `23/24` pass
  (`95.83%`)
- `compatible:gpt-oss:120b-cloud` on full held-out `cases.csv`: `24/24` pass
  (`100%`)
- `compatible:glm-5:cloud` on `cases/cross-model-sanity.csv`: `8/8` pass
  (`100%`)
- `compatible:deepseek-r1:8b` on `cases/cross-model-sanity.csv`: `7/8` pass
  (`87.50%`)
- `compatible:devstral-small-2:latest` on `cases/cross-model-sanity.csv`:
  `6/8` pass (`75.00%`)

Across these valid runs, the most important qualitative signal is that the
kernel rejects bad proofs rather than silently accepting them. In other words,
the system is already useful as a verification boundary even when lowering
quality varies across models.

This is strong evidence for the narrow trust-boundary claim. It is not yet
evidence for broad open-domain truth verification.

## 9. Scope of Confirmation

The current evidence confirms only the following scoped claim:

- given a fixed formal fragment;
- given a fixed certificate format;
- given a model that attempts lowering into that format;
- the Hilbert-style checker can accept valid certificates and reject invalid
  or non-derivable attempts with useful precision.

More concretely, the evidence currently supports:

- formal held-out generalization on unseen propositional benchmark cases;
- cross-model portability of the protocol beyond a single LLM;
- practical usefulness of category-aware evaluation, especially for:
  - `assumption_import`
  - `direct_axiom_instance`
  - `mixed_proof`
  - `negative_refusal`
- the claim that kernel validation is doing real work, rather than merely
  mirroring prompt compliance.

The evidence does **not** justify a stronger statement such as:

> "The system verifies arbitrary AI answers."

That statement is outside the demonstrated scope of this benchmark family.

When interpreting summary tables, request-level failures MUST be separated from
true reasoning failures. For example, runs that return only
`request_failure` because the model id is not available are operational
failures, not evidence about proof quality.

## 10. What Remains Unproven

The following claims remain unproven or only weakly supported:

- robust theorem synthesis without contract drift;
- generalization to richer calculi beyond the current Hilbert fragment;
- verification of natural-language answers without a trusted formalization
  step;
- factual correctness about the external world;
- model-independence in the strong sense that weak or medium models perform
  comparably to stronger ones.

The weakest area in the current evidence remains `theorem_synthesis`.
`compatible:gpt-oss:20b` performed well overall, but its theorem-synthesis
stress run still recorded `4/5` pass with one `request_failure`, and an
earlier full held-out run failed one theorem-style case by contract drift
rather than kernel rejection. This means theorem synthesis is promising, but
not yet stable enough to claim as solved.

The benchmark also still depends on prompt design quality. Prompt version
`hilbert-ai-verification-benchmark-v1.2` materially improved contract
retention, especially for axiom-shaped outputs. That is useful engineering
evidence, but it also means the benchmark should not be misrepresented as a
proof that prompting no longer matters.

For future held-out evaluation, prompt version
`hilbert-ai-verification-benchmark-v1.3` SHOULD be preferred when the goal is
to reduce benchmark-specific sample imitation. `v1.3` keeps the explicit
contract and anti-drift rules from earlier prompt revisions but removes inline
shape examples, making the evaluation methodologically cleaner even if raw pass
rates later decrease.

### 10.1 Snapshot: `v1.3` result wave on `2026-03-26`

- `v1.3` removed inline benchmark examples and immediately reduced pass rates
  relative to `v1.2`, which is evidence that earlier success partly reflected
  contract-following help from examples rather than reasoning alone.
- On full held-out `cases.csv`, the strongest recorded `v1.3` runs are:
  `compatible:gpt-oss:120b-cloud` `22/24`, `compatible:gpt-oss:20b` `16/24`,
  and `compatible:deepseek-r1:14b` `11/24`.
- On `cases/theorem-synthesis-deep-stress.csv`, the strongest recorded
  `v1.3` runs are: `compatible:gpt-oss:120b-cloud` `7/10`,
  `compatible:deepseek-v3.1:671b-cloud` `6/10`, `compatible:gpt-oss:20b`
  `5/10`, `compatible:deepseek-r1:14b` `5/10`, `compatible:qwen3-coder:480b-cloud`
  `4/10`, and `compatible:minimax-m2:cloud` `2/10`.
- The dominant `v1.3` failure modes are `schema_failure`,
  `request_failure`, and occasional `parse_failure` or `kernel_failure`;
  the evidence does not show a wave of false positive acceptance.
- Across the currently recorded `v1.3` summary rows in
  `result_summary.csv`, `false_accept_count` remains `0`, which means the
  checker boundary still looks conservative even when model outputs degrade.
- The practical interpretation is narrow but important: the trust-boundary
  hypothesis survives the stricter prompt, while example-free lowering remains
  materially harder and strongly model-dependent.

### 10.2 Product decision gate

The research stage for bounded internal verification SHOULD be treated as
complete only if the strongest `v1.3` models maintain all of the following:

- `false_accept = 0`
- high `negative_refusal`
- acceptable `mixed_proof`
- at least moderate `theorem_synthesis`
- and consistency not only on `v1.2`, but also on `v1.3`

If these conditions are met, the repository may honestly claim that the
bounded internal verification research stage is complete. Even then, any claim
about public or open-domain verification remains outside the confirmed scope.

Therefore the current research position should be stated conservatively:

> The repository now contains strong evidence that Hilbert-style certificate
> checking is viable as a bounded verification layer for formalizable LLM
> outputs. It does not yet contain sufficient evidence that arbitrary AI
> answers can be verified in the same way.

## 11. Related research next-step

The current direct-Hilbert results motivate two distinct next-step research
tracks:

- richer front-end authoring (`ND -> Hilbert`);
- a mechanized Lean4/mathlib sidecar for proof/specification artifacts and
  formal case-hypothesis generation.

The immediate next-cycle execution order for those follow-on steps is currently
tracked separately in:

- [Proof Research Next-Cycle Plan](./proof-research-next-cycle-plan.md)

The Lean4 track is specified in:

- [Hilbert AI Verification Lean4 Worker v1](./hilbert-ai-verification-lean4-worker-v1.md)

That research direction is additive. It does not weaken or replace the current
approved Hilbert-style boundary result.

