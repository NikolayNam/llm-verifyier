# Proof Research Phase 1 Runbook

- **Title:** **Proof Research Phase 1 Runbook**
- **Status:** Active operator runbook
- **Date:** 2026-03-30
- **Scope:** direct Hilbert stress reruns for the current `v2 held-out` slice
- **Claim boundary:** `checker soundness` only

---

## 1. Purpose

Phase 1 is the direct-Hilbert rerun slice for the first hard question:

> does `false_accept = 0` remain preserved as coverage expands, while
> degradation moves into refusal / request / schema / parse / kernel / format
> failure buckets rather than unsafe acceptance?

Trusted component:

- final Hilbert checker only

This runbook does **not** make a claim about ND, Lean4, or future runtime
certificate families.

---

## 2. Canonical Inputs And Outputs

Current canonical input:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv`

Shared summary outputs, grouped by `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/deepseek/deepseek-v3-1-671b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`

Research-facing report outputs:

- per-model:
  - `research/result_research_report/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_report_phase1_direct_20260330.md`
- aggregate:
  - `research/result_research_report/by-date/20260330/_aggregate/direct/hilbert-ai-verification-benchmark-v2-held-out_report_phase1_direct_20260330.md`

Family-local per-run outputs now live under:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/20260330/<provider-label>/<model>/direct/`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/20260330/<provider-label>/<model>/direct/`

Current summary schema note:

- each shared summary row now carries `surface=direct`
- non-default `PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE` values are encoded
  into the Google model slug used by the `make` wrapper, for example
  `google/gemini-2-5-flash_budget0/...`

---

## 3. INSTRUCTION FOR USER: RUN

Important implementation facts:

- if you want `research-db-sync` to discover the Phase 1 shared summaries
automatically, keep the benchmark project folder in the summary filename
basename
- Phase 1 now writes one shared summary file per model, not one mixed
catch-all CSV
- `make proof-research-phase1` writes file artifacts only (`result`, `raw`,
  shared `summary`, Markdown `report`) and does **not** auto-sync
  `research/artifacts/result_research/research_db/research_db.sqlite`
- `run-hilbert-benchmark` and `researchctl phase1 run` support
  `request-timeout-abort-threshold`; when set to a positive integer, the run
  stops new LLM requests after that many consecutive request timeouts and
  fills remaining cases as `request_failure`
- the benchmark-level default can now live in research YAML config via
  `benchmarks.phase1.request_timeout_abort_threshold`; CLI flags override the
  config value when explicitly passed
- `researchctl phase1 run` now also supports `--jobs N` for bounded
  parallelism across benchmark jobs inside one invocation; `--jobs 1`
  preserves the historical sequential behavior
- interrupted benchmark handling now also lives in research YAML config via
  `benchmarks.phase1.interrupt_policy`; CLI flags override the config value
  when explicitly passed
- allowed interrupt policies are:
  - `keep`
  - `drop_if_no_results`
  - `drop_always`
- the canonical default is `drop_if_no_results`: interrupted runs are kept as
  `aborted` only once they have meaningful outputs; otherwise the control-plane
  row and run-scoped artifacts are dropped
- family-level sampling and experimental provenance now also live in research
  YAML config; the canonical fields are:
  - `families.<name>.experiment_name`
  - `families.<name>.sampling.temperature`
  - `families.<name>.sampling.seed`
  - `families.<name>.sampling.top_p`
- `experiment_name` is required whenever a family declares sampling overrides,
  because the runner now materializes a distinct `experiment_id` and writes it
  into the manifest, per-run CSV artifacts, and research DB import layer
- the benchmark artifacts now separate:
  - `requested_sampling_json`
  - `effective_sampling_json`
  - `unsupported_sampling_json`
  so that OpenAI-compatible and other provider surfaces are not misreported as
  having applied undocumented parameters
- the explicit first-pass commands below keep `-provider compatible` for the
  local Ollama/OpenAI-compatible transport, but stamp `provider_label` into
  paths and CSV rows (`openai` for `gpt-oss:*`, `deepseek` for
  `deepseek-*`)
- the type split is explicit now in both places:
- filename contains `direct`
- CSV contains `surface=direct`

Minimal family example:

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

Local overrides MAY change only the experimental setup while keeping the same
transport and model family, for example:

```yaml
families:
  local-compatible:
    experiment_name: balanced-mode
    sampling:
      top_p: 0.9
```

When this happens, the run MUST produce a new `experiment_id` rather than
reusing the old experiment lineage.

Minimal benchmark policy example:

```yaml
benchmarks:
  phase1:
    request_timeout_abort_threshold: 10
    interrupt_policy: drop_if_no_results
```

Focused next-cycle phase-1 lane profiles currently available in-repo:

- `research/config/direct/phase1-cross-model-sanity.yaml`
- `research/config/direct/phase1-assumption-import-canary.yaml`
- `research/config/direct/phase1-entailed-gap-audit.yaml`
- `research/config/direct/phase1-negative-refusal-sanity.yaml`
- `research/config/direct/phase1-direct-axiom-mixed-proof-ladder.yaml`
- `research/config/direct/phase1-theorem-synthesis-deep-stress.yaml`

They are intended for `researchctl phase1 plan|run --local-config ...` rather
than for the broader `make proof-research-phase1` wrapper.

The compositional lemma-reuse experiment is no longer documented as a phase-1
lane. It now lives on a dedicated standalone surface:

- `researchctl compositional-assumption-import plan|run`
- local profile:
  - `research/config/compositional/assumption-import.yaml`
- follow-on surfaces:
  - `researchctl compositional-depth-ladder plan|run`
  - `research/config/compositional/depth-ladder.yaml`
  - `researchctl compositional-branching plan|run`
  - `research/config/compositional/branching.yaml`
  - `researchctl compositional-mixed-family-reuse plan|run`
  - `research/config/compositional/mixed-family-reuse.yaml`
  - `researchctl compositional-mixed-family-gold-first plan|run`
  - `research/config/compositional/mixed-family-gold-first.yaml`

### 3.1 First pass `r01`

Run these commands from the repository root.

```bash
DATE_TAG="20260330"
PROVIDER="google"
PROVIDER_LABEL="google"
SAFE_PROVIDER="google"
MODEL="gemini-2.5-flash"
SAFE_MODEL="gemini-2-5-flash"
RUN_ID="${DATE_TAG}_phase1_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
-provider "$PROVIDER" \
-provider-label "$PROVIDER_LABEL" \
-base-url "https://generativelanguage.googleapis.com/v1beta" \
-api-key "${GEMINI_API_KEY:?set GEMINI_API_KEY}" \
-model "$MODEL" \
-timeout "60s" \
-project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
-cases-file "cases.csv" \
-cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv" \
-results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
-summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv" \
-raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
-run-id "$RUN_ID" \
-prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

```bash
DATE_TAG="20260330"
PROVIDER="compatible"
PROVIDER_LABEL="openai"
SAFE_PROVIDER="openai"
MODEL="gpt-oss:120b-cloud"
SAFE_MODEL="gpt-oss-120b-cloud"
RUN_ID="${DATE_TAG}_phase1_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
-provider "$PROVIDER" \
-provider-label "$PROVIDER_LABEL" \
-base-url "http://localhost:11434" \
-model "$MODEL" \
-timeout "60s" \
-project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
-cases-file "cases.csv" \
-cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv" \
-results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
-summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv" \
-raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
-run-id "$RUN_ID" \
-prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

```bash
DATE_TAG="20260330"
PROVIDER="mistral"
PROVIDER_LABEL="mistral"
SAFE_PROVIDER="mistral"
MODEL="mistral-large-2512"
SAFE_MODEL="mistral-large-2512"
RUN_ID="${DATE_TAG}_phase1_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
-provider "$PROVIDER" \
-provider-label "$PROVIDER_LABEL" \
-base-url "https://api.mistral.ai" \
-api-key "${MISTRAL_API_KEY:?set MISTRAL_API_KEY}" \
-model "$MODEL" \
-timeout "60s" \
-project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
-cases-file "cases.csv" \
-cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv" \
-results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
-summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv" \
-raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
-run-id "$RUN_ID" \
-prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

```bash
DATE_TAG="20260330"
PROVIDER="compatible"
PROVIDER_LABEL="deepseek"
SAFE_PROVIDER="deepseek"
MODEL="deepseek-v3.1:671b-cloud"
SAFE_MODEL="deepseek-v3-1-671b-cloud"
RUN_ID="${DATE_TAG}_phase1_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
-provider "$PROVIDER" \
-provider-label "$PROVIDER_LABEL" \
-base-url "http://localhost:11434" \
-model "$MODEL" \
-timeout "60s" \
-project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
-cases-file "cases.csv" \
-cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv" \
-results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
-summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv" \
-raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
-run-id "$RUN_ID" \
-prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

### 3.2 Repeat passes through `r08`

Run the same four commands again, changing only:

- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r04"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r05"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r06"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r07"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r08"`

Keep the same per-model shared summary file path for all repeats of that
model.

### 3.3 Build the aggregate direct report

`hilbert-benchmark-report` now accepts a comma-separated list of summary files.
The resulting Markdown report now includes:

- aggregate
- entailed-only
- not-entailed-only
- per-category
- per-case hardest failures
- hard-case audit targets for `V2E01`, `V2E06`, `V2E08`, `V2E07`, `V2E12`, `V2N07`, `V2N09`

```bash
DATE_TAG="20260330"
SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/deepseek/deepseek-v3-1-671b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/mistral/mistral-large-2512/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-report \
  -project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
  -cases-file "cases.csv" \
  -summary "$SUMMARY_SOURCES" \
  -model-catalog "$PWD/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/model-catalog.csv" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/direct/hilbert-ai-verification-benchmark-v2-held-out_report_phase1_direct_${DATE_TAG}.md"
```

### 3.4 Optional aggregate terminal summary

The terminal summary now prints:

- aggregate
- entailed-only
- not-entailed-only
- per-category slices

```bash
DATE_TAG="20260330"
SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/deepseek/deepseek-v3-1-671b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/mistral/mistral-large-2512/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-summary \
  -project-folder "hilbert-ai-verification-benchmark-v2-held-out" \
  -cases-file "cases.csv" \
  -summary "$SUMMARY_SOURCES"
```

### 3.5 Sync into the shared research DB

```bash
make research-db-sync \
  RESEARCH_DB_PROJECT_FOLDER='hilbert-ai-verification-benchmark-v2-held-out' \
  RESEARCH_DB_PATH='research/artifacts/result_research/research_db/research_db.sqlite'
```

## 4. Convenience Wrapper

If you prefer the make wrapper instead of the explicit commands above:

Run from the repository root:

```bash
make proof-research-phase1
```

Current wrapper caveat:

- `make proof-research-phase1` does **not** call `make research-db-sync`
  automatically, so Phase 1 benchmark rows are not indexed into
  `research_db.sqlite` until you run the explicit sync step from section `3.5`

Useful overrides:

```bash
make proof-research-phase1 PROOF_RESEARCH_PHASE1_REPEATS=8
make proof-research-phase1 PROOF_RESEARCH_PHASE1_JOBS=2
go -C platform-tooling run ./cmd/researchctl phase1 run --config research/config/default.yaml --jobs 2
make proof-research-phase1 PROOF_RESEARCH_PHASE1_OPENAI_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE1_DEEPSEEK_MODELS='deepseek-v3.1:671b-cloud'
make proof-research-phase1 PROOF_RESEARCH_PHASE1_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE1_MISTRAL_MODELS=''
make proof-research-phase1 PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE=budget0
make proof-research-phase1 PROOF_RESEARCH_PHASE1_MISTRAL_REQUESTS_PER_SECOND=1
make proof-research-phase1 PROOF_RESEARCH_PHASE1_CASE_ID=V2E12
make proof-research-phase1 PROOF_RESEARCH_PHASE1_LIMIT=8
make proof-research-build-tools
```

Default operator slice:

- local compatible transport slices inherit:
  - `PROOF_RESEARCH_PHASE1_BASE_URL=http://localhost:11434`
- local display labels:
  - `openai` for `gpt-oss:*`
  - `deepseek` for `deepseek-*`
- prompt: `hilbert-ai-verification-benchmark-v1.3`
- existing models:
  - `gpt-oss:120b-cloud`
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-3.1-pro-preview`
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE1_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `labs-leanstral-2603`
  - `mistral-large-2512`
- optional Mistral RPS throttle:
  - `PROOF_RESEARCH_PHASE1_MISTRAL_REQUESTS_PER_SECOND=1`
- repeats: `8`

Important current caveat:

- Google and Mistral slices require explicit API keys; the wrapper reads
  `PROOF_RESEARCH_PHASE1_GOOGLE_API_KEY` or exported `GEMINI_API_KEY` /
  `GOOGLE_API_KEY`, and `PROOF_RESEARCH_PHASE1_MISTRAL_API_KEY` or exported
  `MISTRAL_API_KEY`
- if your Google project is on a low free-tier RPM cap, set
  `PROOF_RESEARCH_PHASE1_GOOGLE_REQUESTS_PER_MINUTE` explicitly; for example,
  `5` enforces a 12-second minimum gap between Google requests inside the
  benchmark runner
- if your Mistral account is capped at `1 req/sec`, leave
  `PROOF_RESEARCH_PHASE1_MISTRAL_REQUESTS_PER_SECOND=1`; the wrapper also
  serializes Mistral models within the family when this limit is enabled
- by default the wrapper also sources `research/llm_models/.env` before
  resolving provider API keys, so colocated Gemini/Mistral credentials are
  picked up automatically during `make proof-research-phase1`
- the wrapper now prebuilds and reuses the `contracts` and `hilbertcheck`
  binaries; if you want to warm the tool cache explicitly, run
  `make proof-research-build-tools`
- `PROOF_RESEARCH_PHASE1_JOBS` controls bounded parallelism across models; for
  a single local Ollama backend, start with `1` or `2` rather than a larger
- `researchctl phase1 run --jobs N` is the canonical non-make equivalent of
  that same bounded-parallelism control
- Google runs now stamp an optional `google_thinking_mode` column into both
  per-run result CSVs and shared summary CSVs; `default` preserves current
  provider-default behavior, while `minimal|low|high|budget0` become explicit
  experiment metadata
- the `make proof-research-phase1` wrapper also turns non-default Google
  thinking modes into distinct Google result identities, so `budget0` and
  `high` do not land in the same `google/<model>/...` directory or `run_id`
  value

---

## 5. Aggregation Note

This runbook intentionally writes fresh outputs into:

- summaries: `research/artifacts/result_research/by-date/<date>/<provider>/<model>/direct/`
- per-run results: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/<date>/<provider>/<model>/direct/`
- raw outputs: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/<date>/<provider>/<model>/direct/`
- per-model reports:
  `research/result_research_report/by-date/<date>/<provider>/<model>/direct/`
- aggregate reports:
  `research/result_research_report/by-date/<date>/_aggregate/direct/`

`research-db-sync` now indexes both the older flat summary layout and the new
dated layout.
