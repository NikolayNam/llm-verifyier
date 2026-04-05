# Hilbert AI Verification Benchmark v1

- **Title:** **Hilbert AI Verification Benchmark v1**
- **Status:** Active research benchmark
- **Date:** 2026-03-26
- **Context:** CollabSphere / certificate subsystem / LLM lowering / proof-theoretic verification
- **Normative modality:** **MUST / SHOULD / MAY** are used as benchmark protocol rules, not as product API guarantees

---

## 1. Purpose

This document defines the first benchmark protocol for testing the following
research hypothesis:

> A minimal Hilbert-style certificate kernel can serve as a useful trust
> boundary for a formalizable subset of AI-assisted reasoning outputs.

The benchmark is intentionally narrow.

It does **not** ask whether a Hilbert-style system can verify arbitrary natural
language answers from an AI system.

It asks whether a model can:

1. receive a formally specified task with trusted premises;
2. either produce a valid certificate in the current v1 format;
3. or honestly refuse with `NOT_DERIVABLE`;
4. while the resulting output is checked by the current lower-layer verifier.

---

## 2. Non-goals

This benchmark does **not** test:

- factual truth about the external world;
- natural-language-to-formal-logic mapping quality;
- retrieval quality;
- product workflow integration;
- Phase 2 control-plane behavior;
- sequent operational reasoning;
- proof search quality beyond the current `classical-hilbert-v1` fragment.

---

## 3. Current logical scope

The benchmark is restricted to the current Milestone 1 lower layer:

- certificate envelope:
  - [certificate-format-v1.md](../../../docs/technical-specs/platform/certificate-format-v1.md)
- kernel rule pack:
  - [certificate-kernel-rule-pack-v1.md](../../../docs/technical-specs/platform/certificate-kernel-rule-pack-v1.md)
- local verifier CLI:
  - [hilbertcheck-cli-contract-v1.md](../../../docs/technical-specs/platform/hilbertcheck-cli-contract-v1.md)

The active rule pack is fixed to:

```text
classical-hilbert-v1
```

The active formula fragment is limited to:

- atoms;
- negation `!`;
- implication `->`;
- parentheses.

The active primitive steps are limited to:

- `assumption`
- `axiom`
- `modus_ponens`

---

## 4. Benchmark structure

The benchmark consists of:

- a human-readable protocol document:
  - `research/active/documents/hilbert-ai-verification-benchmark-v1.md`
- a machine-readable case registry:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/cases.csv`
- a local hypothesis cases directory:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/cases/`
- a legacy flat results log:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/results.csv`
- a canonical historical results directory:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/result/`
- a per-run result file pattern:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_<run-id>.csv`
- an append-only run summary file:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/result/result_summary.csv`
- optional raw per-case outputs:
  - `research/artifacts/hilbert-ai-verification-benchmark-v1/raw/<run-id>/<case-id>.txt`

## 4.1 Canonical runner

The canonical local runner is:

```text
go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark
```

The canonical historical summary reader is:

```text
go -C platform-tooling run ./cmd/contracts hilbert-benchmark-summary
```

For repository-local execution, the preferred wrapper is:

```text
make hilbert-benchmark HILBERT_BENCHMARK_BASE_URL=http://localhost:11434 HILBERT_BENCHMARK_MODEL=llama3.2
```

The preferred historical summary wrapper is:

```text
make hilbert-benchmark-summary
```

To aggregate all case packs inside one benchmark project while keeping
separate model groups, use:

```text
make hilbert-benchmark-summary HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v1 HILBERT_BENCHMARK_CASES_FILE=*
```

Useful flags:

- `-provider compatible|openai`
- `-base-url <endpoint-root>`
- `-api-key <token>`
- `-model <model-id>`
- `-project-folder <benchmark-project-folder>`
- `-cases-file <relative-cases-file>`
- `-cases <path-to-cases.csv>`
- `-results <path-to-results.csv>`
- `-summary <path-to-result-summary.csv>`
- `-raw-dir <path-to-raw-dir>`
- `-run-id <stable-run-id>`
- `-case-id <single-case-id>`
- `-limit <n>`

Environment fallbacks:

- `HILBERT_BENCHMARK_LLM_PROVIDER`
- `HILBERT_BENCHMARK_BASE_URL`
- `HILBERT_BENCHMARK_API_KEY`
- `HILBERT_BENCHMARK_MODEL`
- `HILBERT_BENCHMARK_PROJECT_FOLDER`
- `HILBERT_BENCHMARK_CASES_FILE`
- `HILBERT_BENCHMARK_RESULTS`
- `HILBERT_BENCHMARK_SUMMARY`
- `HILBERT_BENCHMARK_RUN_ID`
- `HILBERT_BENCHMARK_PROMPT_VERSION`
- `OPENAI_BASE_URL`
- `OPENAI_API_KEY`
- `OPENAI_MODEL`
- `LLM_PROVIDER`
- `LLM_BASE_URL`
- `LLM_API_KEY`
- `LLM_MODEL`

The runner:

1. reads `cases.csv`;
2. builds the fixed benchmark prompt;
3. saves raw model output per case;
4. runs candidate JSON through the canonical `hilbertcheck` binary;
5. appends scored rows to `result/result_<run-id>.csv`;
6. appends one run-level metadata row to `result/result_summary.csv`.

### 4.2 Historical result layout

The canonical storage layout for new runs is:

- `result/result_<run-id>.csv`
- `result/result_summary.csv`
- `raw/<run-id>/...`

`run-id` SHOULD default to a generated UTC timestamp when
`HILBERT_BENCHMARK_RUN_ID` is not provided explicitly.

The legacy flat file:

- `research/artifacts/hilbert-ai-verification-benchmark-v1/results.csv`

remains valid historical evidence and MUST NOT be rewritten retroactively.

### 4.3 Exact `result_summary.csv` schema

The canonical `result_summary.csv` schema is:

```csv
run_id,timestamp_utc,model,llm_model,prompt_version,certificate_version,project_folder,cases_file,case_selector,hypothesis,cases_path,result_file,raw_dir,cases_total,pass_count,false_refusal_count,false_accept_count,request_failure_count,schema_failure_count,parse_failure_count,kernel_failure_count,format_failure_count,avg_latency_ms,max_latency_ms,run_elapsed_seconds,category_count
```

`run_id` is mandatory and identifies the corresponding per-run result file:

- `result/result_<run-id>.csv`

`llm_model`, `project_folder`, `cases_file`, and `case_selector` describe:

- the raw model id such as `gpt-oss:20b`;
- which benchmark family was selected;
- which concrete cases file was run;
- whether the run targeted the full set or a filtered selector such as one case.

`avg_latency_ms`, `max_latency_ms`, and `run_elapsed_seconds` describe:

- the average per-case model latency for that run;
- the worst per-case model latency for that run;
- the total wall-clock duration of that benchmark run.

Older historical `result_summary.csv` rows without these extra columns remain
valid legacy audit trail.

`cases_file=*` is a special summary-reader mode. It means:

- filter only by `project_folder`;
- include all case packs recorded in that project's `result_summary.csv`;
- still aggregate separately by model and prompt profile.

For a Markdown comparison report by case-pack, use:

```text
make hilbert-benchmark-report HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v1 HILBERT_BENCHMARK_CASES_FILE=*
```

To write the Markdown report to a file instead of stdout:

```text
make hilbert-benchmark-report HILBERT_BENCHMARK_PROJECT_FOLDER=hilbert-ai-verification-benchmark-v1 HILBERT_BENCHMARK_CASES_FILE=* HILBERT_BENCHMARK_REPORT_OUT=research/artifacts/hilbert-ai-verification-benchmark-v1/result/model-comparison.md
```

### 4.4 Hypothesis case packs

Additional local case packs MAY be stored under:

- `research/artifacts/hilbert-ai-verification-benchmark-v1/cases/`

This directory is intended for hypothesis-specific CSV files such as:

- `cases/hypothesis_contract_following.csv`
- `cases/hypothesis_negative_refusal.csv`
- `cases/hypothesis_theorem_synthesis.csv`

The runner does not require a special naming convention. Any CSV file with the
benchmark schema may be passed via:

- `-cases <path>`
- `HILBERT_BENCHMARK_CASES=<path>`

The `cases/` directory is a working research area for local experiments. It is
not the canonical source of truth for the benchmark family by itself.

---

## 5. Prompt versions

The benchmark family remains the same across prompt revisions.

Existing rows in:

- `research/artifacts/hilbert-ai-verification-benchmark-v1/results.csv`

must remain untouched as historical evidence, even if later prompt revisions
improve contract adherence.

### 5.1 Prompt version `hilbert-ai-verification-benchmark-v1`

This is the initial strict zero-shot contract.

It contains no canonical example and serves as the baseline for early runs,
including model availability failures and the first real schema failure on
`H01`.

### 5.2 Prompt version `hilbert-ai-verification-benchmark-v1.1`

This is a contract-tightening revision of the same benchmark protocol.

It adds:

- explicit required top-level fields;
- explicit required `context` fields;
- explicit prohibition of `line` and `goal_line`;
- explicit guidance for `assumption_ref`;
- one minimal valid shape example for `H01`.

`v1.1` is still a single-shot benchmark prompt.

It does **not** introduce:

- repair loops;
- second-chance prompting;
- manual correction;
- benchmark case changes.

The preferred local default after 2026-03-26 is `v1.1`.

### 5.3 Prompt version `hilbert-ai-verification-benchmark-v1.2`

This is a targeted axiom-shape refinement of the same benchmark protocol.

It preserves the `v1.1` contract tightening and additionally adds:

- an explicit rule that every `axiom` step must contain the `axiom` field;
- an explicit closed value set for the current pack:
  - `A1`
  - `A2`
  - `A3`
- one minimal valid direct-axiom example aligned with `H08`.

`v1.2` does **not** change:

- benchmark cases;
- result storage layout;
- scoring buckets;
- historical `v1` or `v1.1` rows.

`v1.2` is intended for targeted reruns of axiom-bearing cases such as `H06`
and `H08`, without changing the benchmark family or rewriting the audit trail.

### 5.4 Prompt version `hilbert-ai-verification-benchmark-v1.3`

This is an example-free contract-hardening revision of the same benchmark
protocol.

It preserves the `v1.2` contract tightening and additionally:

- removes inline benchmark examples entirely;
- keeps explicit `axiom`-step requirements;
- adds explicit anti-drift rules against alternative schema names such as:
  - `theorem`
  - `theorems`
  - `axioms`
  - `rules`
  - `goal_line`
- adds an explicit rule that `name` must not be used instead of `axiom`;
- makes it explicit that the `context` object must contain only:
  - `domain`
  - `rule_pack`
  - `syntax`
  - `generator`

`v1.3` is intended for cleaner held-out evaluation where the benchmark should
depend less on sample imitation and more on direct contract following.

`v1.3` does **not** change:

- benchmark cases;
- result storage layout;
- scoring buckets;
- historical `v1`, `v1.1`, or `v1.2` rows.

---

## 6. Case selection rules

Benchmark v1 includes only formal cases.

Each case must define:

- a stable `case_id`;
- a binary expected label:
  - `entailed`
  - `not_entailed`
- an ordered list of trusted assumptions;
- a single goal formula;
- a short comment explaining the intent of the case.

The benchmark must include:

- positive cases that are derivable in the current fragment;
- negative cases that are not derivable in the current fragment;
- at least one direct axiom-instance case;
- at least one assumption-import case;
- at least one chained `modus_ponens` case;
- at least one contrapositive/A3-style case;
- at least one failure case that would tempt a model into invalid reasoning.

---

## 7. Prompt protocol

### 7.1 System prompt

Use the following system prompt for benchmark version
`hilbert-ai-verification-benchmark-v1`:

```text
You are a certificate generator for the rule pack classical-hilbert-v1.

Return exactly one of the following:
1. a single JSON object that conforms to certificate-format-v1; or
2. the exact token NOT_DERIVABLE

Do not return markdown.
Do not return code fences.
Do not return explanations.
Do not return comments.
Do not return any text before or after the JSON object or NOT_DERIVABLE.

Hard rules:
- certificate_version must be "1.0.0"
- context.domain must be "hilbert-benchmark-v1"
- context.rule_pack must be "classical-hilbert-v1"
- context.syntax must be "hilbert-prop-ascii-v1"
- context.generator must be "llm-benchmark-v1"
- allowed step kinds are only: assumption, axiom, modus_ponens
- assumption_ref is 1-based
- modus_ponens premises must be ordered as [antecedent_line, implication_line]
- use only formulas over atoms, !, ->, and parentheses
- if the goal is not derivable from the assumptions in this fragment, return exactly NOT_DERIVABLE
```

Use the following system prompt for benchmark version
`hilbert-ai-verification-benchmark-v1.1`:

```text
You are a certificate generator for the rule pack classical-hilbert-v1.

Return exactly one of the following:
1. a single JSON object that conforms to certificate-format-v1; or
2. the exact token NOT_DERIVABLE

Do not return markdown.
Do not return code fences.
Do not return explanations.
Do not return comments.
Do not return any text before or after the JSON object or NOT_DERIVABLE.

Required top-level JSON fields:
- certificate_version
- proof_id
- goal
- context
- steps

Allowed optional top-level fields:
- assumptions
- sources
- meta
- hashes

Do not output any other top-level fields.
Do not output the fields "line" or "goal_line".

Required context fields:
- domain
- rule_pack
- syntax
- generator

Hard rules:
- certificate_version must be "1.0.0"
- context.domain must be "hilbert-benchmark-v1"
- context.rule_pack must be "classical-hilbert-v1"
- context.syntax must be "hilbert-prop-ascii-v1"
- context.generator must be "llm-benchmark-v1"
- allowed step kinds are only: assumption, axiom, modus_ponens
- assumption_ref is 1-based
- assumption steps require assumption_ref
- assumption_ref points to the top-level assumptions array
- if an assumption step is used, the top-level assumptions array must be present
- line numbering is implicit by step array position; never output a "line" field
- modus_ponens premises must be ordered as [antecedent_line, implication_line]
- use only formulas over atoms, !, ->, and parentheses
- if the goal is not derivable from the assumptions in this fragment, return exactly NOT_DERIVABLE

Minimal valid shape example for case H01:
{"certificate_version":"1.0.0","proof_id":"H01","goal":"P","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":["P"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"P"}]}

The example shows shape only.
Do not copy it blindly.
Adapt proof_id, assumptions, goal, and steps to the current case.
If the current case is not derivable, output exactly NOT_DERIVABLE.
```

Use the following system prompt for benchmark version
`hilbert-ai-verification-benchmark-v1.2`:

```text
You are a certificate generator for the rule pack classical-hilbert-v1.

Return exactly one of the following:
1. a single JSON object that conforms to certificate-format-v1; or
2. the exact token NOT_DERIVABLE

Do not return markdown.
Do not return code fences.
Do not return explanations.
Do not return comments.
Do not return any text before or after the JSON object or NOT_DERIVABLE.

Required top-level JSON fields:
- certificate_version
- proof_id
- goal
- context
- steps

Allowed optional top-level fields:
- assumptions
- sources
- meta
- hashes

Do not output any other top-level fields.
Do not output the fields "line" or "goal_line".

Required context fields:
- domain
- rule_pack
- syntax
- generator

Hard rules:
- certificate_version must be "1.0.0"
- context.domain must be "hilbert-benchmark-v1"
- context.rule_pack must be "classical-hilbert-v1"
- context.syntax must be "hilbert-prop-ascii-v1"
- context.generator must be "llm-benchmark-v1"
- allowed step kinds are only: assumption, axiom, modus_ponens
- assumption_ref is 1-based
- assumption steps require assumption_ref
- assumption_ref points to the top-level assumptions array
- if an assumption step is used, the top-level assumptions array must be present
- line numbering is implicit by step array position; never output a "line" field
- modus_ponens premises must be ordered as [antecedent_line, implication_line]
- use only formulas over atoms, !, ->, and parentheses
- if the goal is not derivable from the assumptions in this fragment, return exactly NOT_DERIVABLE

Critical axiom-step rule:
- every step with "kind": "axiom" MUST include the field "axiom"
- the value of "axiom" MUST be exactly one of: "A1", "A2", "A3"
- do not output an axiom step without the "axiom" field

Minimal valid shape example for an assumption-import case:
{"certificate_version":"1.0.0","proof_id":"H01","goal":"P","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"assumptions":["P"],"steps":[{"kind":"assumption","assumption_ref":1,"formula":"P"}]}

Minimal valid shape example for a direct axiom-instance case:
{"certificate_version":"1.0.0","proof_id":"H08","goal":"P -> (Q -> P)","context":{"domain":"hilbert-benchmark-v1","rule_pack":"classical-hilbert-v1","syntax":"hilbert-prop-ascii-v1","generator":"llm-benchmark-v1"},"steps":[{"kind":"axiom","axiom":"A1","formula":"P -> (Q -> P)"}]}

The examples show shape only.
Do not copy them blindly.
Adapt proof_id, assumptions, goal, axiom labels, and steps to the current case.
If the current case is not derivable, output exactly NOT_DERIVABLE.
```

Use the following system prompt for benchmark version
`hilbert-ai-verification-benchmark-v1.3`:

```text
You are a certificate generator for the rule pack classical-hilbert-v1.

Return exactly one of the following:
1. a single JSON object that conforms to certificate-format-v1; or
2. the exact token NOT_DERIVABLE

Do not return markdown.
Do not return code fences.
Do not return explanations.
Do not return comments.
Do not return any text before or after the JSON object or NOT_DERIVABLE.

Required top-level JSON fields:
- certificate_version
- proof_id
- goal
- context
- steps

Allowed optional top-level fields:
- assumptions
- sources
- meta
- hashes

Do not output any other top-level fields.
Do not output the fields "line" or "goal_line".

Required context fields:
- domain
- rule_pack
- syntax
- generator

The context object must contain only:
- domain
- rule_pack
- syntax
- generator

Hard rules:
- certificate_version must be "1.0.0"
- context.domain must be "hilbert-benchmark-v1"
- context.rule_pack must be "classical-hilbert-v1"
- context.syntax must be "hilbert-prop-ascii-v1"
- context.generator must be "llm-benchmark-v1"
- allowed step kinds are only: assumption, axiom, modus_ponens
- assumption_ref is 1-based
- assumption steps require assumption_ref
- assumption_ref points to the top-level assumptions array
- if an assumption step is used, the top-level assumptions array must be present
- line numbering is implicit by step array position; never output a "line" field
- modus_ponens premises must be ordered as [antecedent_line, implication_line]
- use only formulas over atoms, !, ->, and parentheses
- if the goal is not derivable from the assumptions in this fragment, return exactly NOT_DERIVABLE

Critical axiom-step rules:
- every step with "kind": "axiom" MUST include the field "axiom"
- the value of "axiom" MUST be exactly one of: "A1", "A2", "A3"
- do not output an axiom step without the "axiom" field
- do not use "name" instead of "axiom"

Critical anti-drift rules:
- do not invent alternative schema names such as "theorem", "theorems", "axioms", "rules", or "goal_line"
- do not invent alternative context fields
- do not output wrapper text such as "Here is the JSON"
- do not output markdown fences such as three backticks followed by json

This prompt intentionally contains no inline benchmark examples.
You must follow the contract rules directly rather than imitating a sample.
If the current case is not derivable, output exactly NOT_DERIVABLE.
```

### 7.2 User prompt template

Use the following user prompt template:

```text
Case ID: {{case_id}}

Assumptions:
1. {{assumption_1}}
2. {{assumption_2}}
3. {{assumption_3}}

Goal:
{{goal}}

Return only:
- a valid certificate JSON for this goal from these assumptions; or
- NOT_DERIVABLE
```

### 7.3 Execution discipline

For benchmark v1:

- temperature should be `0`;
- one attempt should be used per case;
- no repair loop should be used;
- no second-chance prompting should be used;
- no hidden manual correction should be applied before scoring.

---

## 8. Verification procedure

For each case:

1. send the case to the model with the fixed prompt protocol;
2. save the raw output to:
   - `research/artifacts/hilbert-ai-verification-benchmark-v1/raw/<run-id>/<case-id>.txt`
3. classify the raw output as one of:
   - `certificate`
   - `not_derivable`
   - `invalid_json`
   - `other`
4. if the output is JSON, save it as `.json` and verify it with the canonical
   checker;
5. record the result row in `results.csv`.

The canonical local verification path is:

```text
go -C platform-tooling build -o .tmp/hilbertcheck ./cmd/hilbertcheck
.tmp/hilbertcheck <candidate-certificate.json>
```

For strict exit-code assertions, use the built binary rather than relying on
`go run`.

---

## 9. Scoring model

### 8.1 Output classes

`raw_output_kind` must be one of:

- `certificate`
- `not_derivable`
- `invalid_json`
- `other`

### 8.2 Score buckets

`score_bucket` must be one of:

- `pass`
- `false_refusal`
- `false_accept`
- `request_failure`
- `schema_failure`
- `parse_failure`
- `kernel_failure`
- `format_failure`

### 8.3 Scoring rules

For expected label `entailed`:

- model/request failure -> `request_failure`
- accepted certificate -> `pass`
- `NOT_DERIVABLE` -> `false_refusal`
- schema error -> `schema_failure`
- parse error -> `parse_failure`
- kernel rejection -> `kernel_failure`
- malformed output -> `format_failure`

For expected label `not_entailed`:

- model/request failure -> `request_failure`
- `NOT_DERIVABLE` -> `pass`
- accepted certificate -> `false_accept`
- schema error -> `schema_failure`
- parse error -> `parse_failure`
- kernel rejection -> `kernel_failure`
- malformed output -> `format_failure`

### 8.4 Primary metrics

At minimum, compute:

- `contract_compliance_rate`
- `positive_accept_rate`
- `negative_honest_refusal_rate`
- `critical_false_accept_rate`
- `net_benchmark_success_rate`

Definitions:

- `contract_compliance_rate` =
  `(certificate outputs + not_derivable outputs) / total cases`
- `positive_accept_rate` =
  `accepted entailed cases / total entailed cases`
- `negative_honest_refusal_rate` =
  `not_derivable outputs on not_entailed cases / total not_entailed cases`
- `critical_false_accept_rate` =
  `accepted certificates on not_entailed cases / total not_entailed cases`
- `net_benchmark_success_rate` =
  `pass cases / total cases`

The most important risk metric is `critical_false_accept_rate`.

---

## 10. Suggested acceptance thresholds for v1

Benchmark v1 should be considered promising only if results are roughly in the
following range:

- `contract_compliance_rate >= 0.85`
- `positive_accept_rate >= 0.70`
- `negative_honest_refusal_rate >= 0.80`
- `critical_false_accept_rate <= 0.05`

These are research thresholds, not product-release gates.

---

## 11. Threats to validity

The benchmark can overstate success if:

- tasks are too small or too synthetic;
- premises already encode most of the hard reasoning;
- the model memorizes common theorem shapes;
- accepted certificates prove a distorted version of the original intent;
- natural-language formalization problems are excluded entirely.

The benchmark can understate success if:

- the prompt is badly constrained;
- model output is truncated by tooling;
- the checker is correct but the benchmark harness misclassifies outputs.

---

## 12. Interpretation rule

A passing certificate does **not** mean:

> the AI answer is true about the world.

A passing certificate means only:

> under the fixed premises and the fixed rule pack, the goal was derived
> correctly by the trusted checker.

This distinction must remain explicit in all benchmark reporting.

---

## 13. Final protocol statement

Benchmark v1 is the canonical active research protocol for testing whether the
current Hilbert-style lower layer is useful as a narrow trust boundary for a
formalizable subset of AI-assisted reasoning outputs.


