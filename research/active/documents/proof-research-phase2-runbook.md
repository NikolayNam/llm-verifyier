# Proof Research Phase 2 Runbook

- **Title:** **Proof Research Phase 2 Runbook**
- **Status:** Active operator runbook
- **Date:** 2026-03-30
- **Scope:** paired `direct Hilbert` vs `ND -> deterministic lowering -> Hilbert`
- **Claim boundary:** `producer reliability` and `interface advantage`

---

## 1. Purpose

Phase 2 asks the second hard question:

> if ND helps, is the gain proof-theoretic, or is it only a more convenient
> authoring/interface layer?

Trusted component:

- final Hilbert checker only

Untrusted components:

- direct Hilbert producer
- ND producer
- ND validation path
- deterministic lowering layer

---

## 2. Canonical Inputs And Outputs

Canonical shared theorem pack:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv`

Shared direct summaries, grouped by `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv`

Shared ND summaries, grouped by `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv`

Per-run result CSVs, grouped by `date/provider/model/surface`:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/20260330/openai/gpt-oss-120b-cloud/direct/result_20260330_phase2_direct_openai_gpt-oss-120b-cloud_r01.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/20260330/google/gemini-2-5-flash/nd/result_20260330_phase2_nd_google_gemini-2-5-flash_r01.csv`

Raw outputs, grouped by `date/provider/model/surface`:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/20260330/openai/gpt-oss-120b-cloud/direct/`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/20260330/mistral/mistral-large-2512/nd/`

Research-facing report outputs:

- per-model:
  - `research/result_research_report/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase2_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/mistral/mistral-large-2512/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase2_pair_20260330.md`
- aggregate:
  - `research/result_research_report/by-date/20260330/_aggregate/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase2_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/nd/hilbert-ai-verification-benchmark-nd-v1_report_phase2_nd_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase2_pair_20260330.md`

Current summary schema note:

- direct shared summaries carry `surface=direct`
- ND shared summaries carry `surface=nd`
- non-default `PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE` values are encoded
  into the Google model slug used by the `make` wrapper, for example
  `google/gemini-2-5-flash_low/...`

---

## 3. INSTRUCTION FOR USER: RUN

Current canonical orchestration shorthand:

```bash
go -C platform-tooling run ./cmd/researchctl phase2 run --config research/config/default.yaml
go -C platform-tooling run ./cmd/researchctl phase2 run --config research/config/default.yaml --jobs 2
```

Repository-level wrapper:

```bash
make proof-research-phase2
```

Important implementation facts:

- if you want `research-db-sync` to discover the shared Phase 2 summaries
  automatically, keep the benchmark project folder in the summary filename
  basenames
- Phase 2 shared summaries are now per-model files
- `make proof-research-phase2` writes file artifacts only (`result`, `raw`,
  shared `summary`, Markdown `report`) and does **not** auto-sync
  `research/artifacts/result_research/research_db/research_db.sqlite`
- the type split is explicit in both places:
  - filename contains `direct` or `nd`
  - CSV contains `surface=direct` or `surface=nd`

### 3.1 First pass `r01`

Run these commands from the repository root.

Direct Hilbert, `gemini-2.5-flash`:

```bash
DATE_TAG="20260330"
PROVIDER="google"
PROVIDER_LABEL="google"
SAFE_PROVIDER="google"
MODEL="gemini-2.5-flash"
SAFE_MODEL="gemini-2-5-flash"
RUN_ID="${DATE_TAG}_phase2_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "https://generativelanguage.googleapis.com/v1beta" \
  -api-key "${GEMINI_API_KEY:?set GEMINI_API_KEY}" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv" \
  -raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
  -run-id "$RUN_ID" \
  -prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

ND, `gemini-2.5-flash`:

```bash
DATE_TAG="20260330"
PROVIDER="google"
PROVIDER_LABEL="google"
SAFE_PROVIDER="google"
MODEL="gemini-2.5-flash"
SAFE_MODEL="gemini-2-5-flash"
RUN_ID="${DATE_TAG}_phase2_nd_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-nd-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "https://generativelanguage.googleapis.com/v1beta" \
  -api-key "${GEMINI_API_KEY:?set GEMINI_API_KEY}" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv" \
  -raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd" \
  -run-id "$RUN_ID" \
  -prompt-version "hilbert-ai-verification-benchmark-nd-v1.1"
```

Direct Hilbert, `gpt-oss:120b-cloud`:

```bash
DATE_TAG="20260330"
PROVIDER="compatible"
PROVIDER_LABEL="openai"
SAFE_PROVIDER="openai"
MODEL="gpt-oss:120b-cloud"
SAFE_MODEL="gpt-oss-120b-cloud"
RUN_ID="${DATE_TAG}_phase2_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "http://localhost:11434" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv" \
  -raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct" \
  -run-id "$RUN_ID" \
  -prompt-version "hilbert-ai-verification-benchmark-v1.3"
```

ND, `gpt-oss:120b-cloud`:

```bash
DATE_TAG="20260330"
PROVIDER="compatible"
PROVIDER_LABEL="openai"
SAFE_PROVIDER="openai"
MODEL="gpt-oss:120b-cloud"
SAFE_MODEL="gpt-oss-120b-cloud"
RUN_ID="${DATE_TAG}_phase2_nd_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-nd-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "http://localhost:11434" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv" \
  -raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd" \
  -run-id "$RUN_ID" \
  -prompt-version "hilbert-ai-verification-benchmark-nd-v1.1"
```

### 3.2 Repeat passes `r02` and `r03`

Run the same four commands again, changing only:

- `RUN_ID="${DATE_TAG}_phase2_direct_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase2_direct_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase2_nd_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase2_nd_<safe-model>_r03"`

Keep the same per-model shared summary path for all repeats of that same model
and surface.

### 3.3 Build direct, ND, and pair reports

`hilbert-benchmark-report` for this theorem-pack family now renders the same
observed-result structure used in the strengthened Phase 1 reporting:

- aggregate
- entailed-only
- not-entailed-only
- theorem-synthesis focus
- foundational-category audit for `assumption_import` and `single_mp`
- per-category
- ND pipeline trace
- per-case hardest failures

`hilbert-benchmark-research-report` now also includes the same aggregate and
observed-slice structure, so the paired study is no longer narrower than the
direct/ND single-surface reports.

Current ND-family note:

- no canonical theorem-specific hard-case audit target set is fixed yet, so the
  report keeps the hard-case audit section but it will currently state that no
  fixed audit targets are available

- if you ran an expanded model slice beyond the two explicit examples below,
  append the matching per-model summary files to `DIRECT_SUMMARY_SOURCES` and
  `ND_SUMMARY_SOURCES` before generating the reports

- if your existing ND per-run result CSVs were produced before explicit
  pipeline-stage telemetry was added, the `ND pipeline trace` section is
  classified heuristically from `raw_output_kind` and the recorded
  schema/parse/kernel statuses; treat those rows as inferred legacy staging,
  not as first-class stage logs

Direct aggregate report:

```bash
DATE_TAG="20260330"
DIRECT_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -summary "$DIRECT_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase2_direct_${DATE_TAG}.md"
```

ND aggregate report:

```bash
DATE_TAG="20260330"
ND_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -summary "$ND_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/nd/hilbert-ai-verification-benchmark-nd-v1_report_phase2_nd_${DATE_TAG}.md"
```

Paired research report:

```bash
DATE_TAG="20260330"
DIRECT_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv"
ND_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-research-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/pilot_shared_20260327.csv" \
  -summary "$DIRECT_SUMMARY_SOURCES,$ND_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase2_pair_${DATE_TAG}.md"
```

### 3.4 Sync into the shared research DB

```bash
make research-db-sync-nd-v1
```

## 4. Convenience Wrapper

If you prefer the make wrapper instead of the explicit commands above:

Run the full paired slice from the repository root:

```bash
make proof-research-phase2
```

Current wrapper caveat:

- `make proof-research-phase2` does **not** call `make research-db-sync-nd-v1`
  automatically, so Phase 2 benchmark rows are not indexed into
  `research_db.sqlite` until you run the explicit sync step from section `3.4`

Useful overrides:

```bash
make proof-research-phase2 PROOF_RESEARCH_PHASE2_REPEATS=5
make proof-research-phase2 PROOF_RESEARCH_PHASE2_JOBS=2
make proof-research-phase2 PROOF_RESEARCH_PHASE2_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE2_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE2_MISTRAL_MODELS=''
make proof-research-phase2 PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE=low
make proof-research-phase2 PROOF_RESEARCH_PHASE2_MISTRAL_REQUESTS_PER_SECOND=1
make proof-research-phase2 PROOF_RESEARCH_PHASE2_CASE_ID=NDI05
make proof-research-phase2 PROOF_RESEARCH_PHASE2_LIMIT=8
make proof-research-phase2 PROOF_RESEARCH_PHASE2_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE2_DEEPSEEK_MODELS='deepseek-v3.1:671b-cloud' PROOF_RESEARCH_PHASE2_GOOGLE_MODELS='gemini-2.5-flash' PROOF_RESEARCH_PHASE2_MISTRAL_MODELS='mistral-large-2512' PROOF_RESEARCH_PHASE2_REPEATS=10
make proof-research-phase2 PROOF_RESEARCH_PHASE2_REPEATS=10 PROOF_RESEARCH_PHASE2_GOOGLE_API_KEY="$GEMINI_API_KEY" PROOF_RESEARCH_PHASE2_MISTRAL_API_KEY="$MISTRAL_API_KEY"
make proof-research-build-tools
```

Default operator slice:

- local compatible transport slices inherit:
  - `PROOF_RESEARCH_PHASE2_BASE_URL=http://localhost:11434`
- local display labels:
  - `openai` for `gpt-oss:*`
  - `deepseek` for `PROOF_RESEARCH_PHASE2_DEEPSEEK_MODELS`
- direct prompt: `hilbert-ai-verification-benchmark-v1.3`
- ND prompt: `hilbert-ai-verification-benchmark-nd-v1.1`
- existing models:
  - `gpt-oss:120b-cloud`
- deepseek models:
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-5.1-pro-preview`
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE2_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `labs-leanstral-2603`
  - `mistral-large-2512`
- optional Mistral RPS throttle:
  - `PROOF_RESEARCH_PHASE2_MISTRAL_REQUESTS_PER_SECOND=1`
- repeats: `3`

Important current caveat:

- do **not** use `make hilbert-benchmark-rerun-nd` for the canonical Phase 2
  run, because that convenience wrapper still points at the older ND prompt
  profile rather than `hilbert-ai-verification-benchmark-nd-v1.1`
- if your Google project is on a low free-tier RPM cap, set
  `PROOF_RESEARCH_PHASE2_GOOGLE_REQUESTS_PER_MINUTE` explicitly; for example,
  `5` enforces a 12-second minimum gap between Google requests inside the
  benchmark runner
- if your Mistral account is capped at `1 req/sec`, leave
  `PROOF_RESEARCH_PHASE2_MISTRAL_REQUESTS_PER_SECOND=1`; the wrapper also
  serializes Mistral models within the family when this limit is enabled
- the current wrapper runs one theorem pack at a time; if you want a second
  synthesis-heavy pack, you must first materialize a real theorem CSV for it
  and then run a separate Phase 2 slice against that explicit pack path
- `proof-research-phase2` now stamps provider into `run_id` and summary
  filenames, so mixed-provider reruns do not collide on `google`, `mistral`,
  `openai`, and `deepseek` family labels
- `proof-research-phase2` now writes per-run results, raw outputs, summaries,
  and markdown reports into `by-date/<date>/<provider>/<model>/<surface>/`
  style directories instead of the older flat CSV layout
- the wrapper now prebuilds and reuses the `contracts` and `hilbertcheck`
  binaries; if you want to warm the tool cache explicitly, run
  `make proof-research-build-tools`
- Google and Mistral slices require explicit API keys; the wrapper reads
  `PROOF_RESEARCH_PHASE2_GOOGLE_API_KEY` or exported `GEMINI_API_KEY` /
  `GOOGLE_API_KEY`, and `PROOF_RESEARCH_PHASE2_MISTRAL_API_KEY` or exported
  `MISTRAL_API_KEY`
- by default the wrapper also sources `research/llm_models/.env` before
  resolving provider API keys, so colocated Gemini/Mistral credentials are
  picked up automatically during `make proof-research-phase2`
- `PROOF_RESEARCH_PHASE2_JOBS` controls bounded parallelism across models; on
  a single local Ollama backend, start with `1` or `2`
- `researchctl phase2 run --jobs N` applies the same bounded-parallelism
  control inside each child benchmark lane (`phase2-direct` and `phase2-nd`)
- Google runs now stamp an optional `google_thinking_mode` column into both
  per-run result CSVs and shared summary CSVs; `default` preserves current
  provider-default behavior, while `minimal|low|high|budget0` become explicit
  experiment metadata
- the `make proof-research-phase2` wrapper also turns non-default Google
  thinking modes into distinct Google result identities, so `budget0`, `low`,
  and `high` do not land in the same `google/<model>/...` directory or
  `run_id`

---

## 5. Aggregation Note

Phase 2 intentionally stores outputs under a hierarchical operator layout:

- summaries into
  `research/artifacts/result_research/by-date/<date>/<provider>/<model>/<surface>/`
- per-run results into
  `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/<date>/<provider>/<model>/<surface>/`
- raw outputs into
  `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/<date>/<provider>/<model>/<surface>/`
- per-model reports into
  `research/result_research_report/by-date/<date>/<provider>/<model>/<surface>/`
- aggregate reports into
  `research/result_research_report/by-date/<date>/_aggregate/<surface>/`

The comparative markdown interpretation is then built from those per-surface
summary inputs into the dated `pair/` report directory.
