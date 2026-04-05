# Proof Research Phase 3 Runbook

- **Title:** **Proof Research Phase 3 Runbook**
- **Status:** Active operator runbook
- **Date:** 2026-03-30
- **Scope:** Lean4 support-layer runs and optional benchmark comparison on exported packs
- **Claim boundary:** `producer reliability` and `specification pressure`

---

## 1. Purpose

Phase 3 evaluates Lean4 only through measurable utility:

- theorem backlog generation
- case confirmation
- specification pressure

It does **not** move the trust boundary away from the Hilbert checker.

---

## 2. Canonical Inputs And Outputs

Lean4 project family:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/`

Canonical generated theorem backlog:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/theorems.csv`

Canonical exported benchmark pack:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv`

Optional direct comparison summaries, grouped by `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`
- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`

Optional ND comparison summaries, grouped by `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv`

Optional comparison reports:

- per-model:
  - `research/result_research_report/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/mistral/mistral-large-2512/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase3_lean4pack_pair_20260330.md`
- aggregate:
  - `research/result_research_report/by-date/20260330/_aggregate/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/nd/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_nd_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase3_lean4pack_pair_20260330.md`

Current summary schema note:

- direct shared summaries carry `surface=direct`
- ND shared summaries carry `surface=nd`
- non-default `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE` values are
  encoded into the Google model slug used by the `make` wrapper, for example
  `google/gemini-2-5-flash_high/...`

---

## 3. INSTRUCTION FOR USER: RUN

Current canonical orchestration shorthands:

```bash
go -C platform-tooling run ./cmd/researchctl phase3 run --config research/config/default.yaml
go -C platform-tooling run ./cmd/researchctl phase3 compare run --config research/config/default.yaml
go -C platform-tooling run ./cmd/researchctl phase3 compare run --config research/config/default.yaml --jobs 2
```

Repository-level wrappers:

```bash
make proof-research-phase3
make proof-research-phase3-compare
```

Important implementation facts:

- if you want `research-db-sync` to discover the shared Phase 3 benchmark
  summaries automatically, keep the benchmark project folder in the summary
  filename basenames
- Phase 3 benchmark summaries are now per-model files
- `make proof-research-phase3` runs the Lean4 DB-first commands, so the
  Lean4 hypothesis/export ledger is written into
  `research/artifacts/result_research/research_db/research_db.sqlite`
  directly
- neither `make proof-research-phase3` nor
  `make proof-research-phase3-compare` auto-run the benchmark registry sync
  steps, so benchmark-family summary/result indexing still requires the
  explicit sync commands from section `3.5`
- the explicit comparison commands below keep `-provider compatible` for the
  local Ollama/OpenAI-compatible transport, but stamp `provider_label` into
  paths and CSV rows (`openai` for `gpt-oss:*`, `deepseek` for
  `deepseek-*`)
- the type split is explicit in both places:
  - filename contains `direct` or `nd`
  - CSV contains `surface=direct` or `surface=nd`

### 3.1 Lean4 bootstrap + smoke-check + theorem export

Run these commands from the repository root.

```bash
make lean4-bootstrap \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1'
```

```bash
make lean4-research-job \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_JOB_NAME='identity_proved' \
  LEAN4_JOB_STATEMENT='P -> P' \
  LEAN4_JOB_PAYLOAD_FILE='research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/payloads/identity-proof.json'
```

```bash
make lean4-generate-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1'
```

```bash
make lean4-export-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  HILBERT_BENCHMARK_PROJECT_FOLDER='hilbert-ai-verification-benchmark-nd-v1' \
  LEAN4_EXPORT_OUTPUT_FILE='theorems/lean4-generated.csv'
```

### 3.2 Optional comparison on the exported theorem pack, first pass `r01`

Direct Hilbert, `gemini-2.5-flash`:

```bash
DATE_TAG="20260330"
PROVIDER="google"
PROVIDER_LABEL="google"
SAFE_PROVIDER="google"
MODEL="gemini-2.5-flash"
SAFE_MODEL="gemini-2-5-flash"
RUN_ID="${DATE_TAG}_phase3_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "https://generativelanguage.googleapis.com/v1beta" \
  -api-key "${GEMINI_API_KEY:?set GEMINI_API_KEY}" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv" \
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
RUN_ID="${DATE_TAG}_phase3_nd_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-nd-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "https://generativelanguage.googleapis.com/v1beta" \
  -api-key "${GEMINI_API_KEY:?set GEMINI_API_KEY}" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv" \
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
RUN_ID="${DATE_TAG}_phase3_direct_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "http://localhost:11434" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv" \
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
RUN_ID="${DATE_TAG}_phase3_nd_${SAFE_PROVIDER}_${SAFE_MODEL}_r01"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts run-nd-hilbert-benchmark \
  -provider "$PROVIDER" \
  -provider-label "$PROVIDER_LABEL" \
  -base-url "http://localhost:11434" \
  -model "$MODEL" \
  -timeout "60s" \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -cases "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv" \
  -results "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/result_${RUN_ID}.csv" \
  -summary "$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv" \
  -raw-dir "$PWD/research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/${DATE_TAG}/${SAFE_PROVIDER}/${SAFE_MODEL}/nd" \
  -run-id "$RUN_ID" \
  -prompt-version "hilbert-ai-verification-benchmark-nd-v1.1"
```

### 3.3 Repeat passes `r02` and `r03`

Run the same four comparison commands again, changing only:

- `RUN_ID="${DATE_TAG}_phase3_direct_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase3_direct_<provider-safe>_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase3_nd_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase3_nd_<provider-safe>_<safe-model>_r03"`

Keep the same per-model shared summary path for all repeats of that same model
and surface.

### 3.4 Build direct, ND, and pair reports

If you run the optional exported-pack benchmark comparison, the direct and ND
Markdown reports now use the same observed-result structure as the strengthened
Phase 1 reporting:

- aggregate
- entailed-only
- not-entailed-only
- per-category
- per-case hardest failures

Current ND-family note:

- no canonical theorem-specific hard-case audit target set is fixed yet for the
  exported Lean4 comparison pack, so the hard-case audit section remains
  present but currently reports that no fixed audit targets are available

Direct aggregate report:

```bash
DATE_TAG="20260330"
DIRECT_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -summary "$DIRECT_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_direct_${DATE_TAG}.md"
```

ND aggregate report:

```bash
DATE_TAG="20260330"
ND_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -summary "$ND_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/nd/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_nd_${DATE_TAG}.md"
```

Paired research report:

```bash
DATE_TAG="20260330"
DIRECT_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv"
ND_SUMMARY_SOURCES="$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-20b/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv,$PWD/research/artifacts/result_research/by-date/${DATE_TAG}/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv"

/usr/local/go/bin/go -C platform-tooling run ./cmd/contracts hilbert-benchmark-research-report \
  -project-folder "hilbert-ai-verification-benchmark-nd-v1" \
  -cases-file "theorems/lean4-generated.csv" \
  -summary "$DIRECT_SUMMARY_SOURCES,$ND_SUMMARY_SOURCES" \
  -out "$PWD/research/result_research_report/by-date/${DATE_TAG}/_aggregate/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase3_lean4pack_pair_${DATE_TAG}.md"
```

### 3.5 Sync into the shared research DB

```bash
make research-db-sync \
  RESEARCH_DB_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  RESEARCH_DB_PATH='research/artifacts/result_research/research_db/research_db.sqlite'
```

```bash
make research-db-sync-nd-v1
```

## 4. Convenience Wrappers

If you prefer the make wrappers instead of the explicit commands above:

Bootstrap + smoke-check + backlog generation + export:

```bash
make proof-research-phase3
```

Current wrapper caveat:

- `make proof-research-phase3` updates the Lean4 DB-first ledger directly, but
  it does **not** call either `make research-db-sync` or
  `make research-db-sync-nd-v1` automatically
- `make proof-research-phase3-compare` writes file-based benchmark artifacts
  only and also does **not** auto-sync them into `research_db.sqlite`

Optional direct-vs-ND comparison on the exported theorem pack:

```bash
make proof-research-phase3-compare
```

Useful overrides:

```bash
make proof-research-phase3 PROOF_RESEARCH_PHASE3_JOB_STATEMENT='P -> P'
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_MODELS='gpt-oss:120b-cloud'
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_MODELS=''
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE=high
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_REQUESTS_PER_SECOND=1
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_REPEATS=5
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_JOBS=2
make proof-research-build-tools
```

Default comparison slice:

- local compatible transport slices inherit:
  - `PROOF_RESEARCH_PHASE3_COMPARE_BASE_URL=http://localhost:11434`
- local display labels:
  - `openai` for `gpt-oss:*`
  - `deepseek` for `PROOF_RESEARCH_PHASE3_COMPARE_DEEPSEEK_MODELS`
- existing models:
  - `gpt-oss:120b-cloud`
- deepseek models:
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-3.1-pro-preview`
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `labs-leanstral-2603`
  - `mistral-large-2512`
- optional Mistral RPS throttle:
  - `PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_REQUESTS_PER_SECOND=1`
- repeats: `3`

Important current caveat:

- Google and Mistral slices require explicit API keys; the wrapper reads
  `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_API_KEY` or exported
  `GEMINI_API_KEY` / `GOOGLE_API_KEY`, and
  `PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_API_KEY` or exported
  `MISTRAL_API_KEY`
- if your Google project is on a low free-tier RPM cap, set
  `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_REQUESTS_PER_MINUTE` explicitly; for
  example, `5` enforces a 12-second minimum gap between Google requests
  inside the benchmark runner
- if your Mistral account is capped at `1 req/sec`, leave
  `PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_REQUESTS_PER_SECOND=1`; the wrapper
  also serializes Mistral models within the family when this limit is enabled
- by default the comparison wrapper also sources `research/llm_models/.env`
  before resolving provider API keys, so colocated Gemini/Mistral credentials
  are picked up automatically during `make proof-research-phase3-compare`
- the benchmark comparison wrapper now prebuilds and reuses the `contracts`
  and `hilbertcheck` binaries; if you want to warm the tool cache explicitly,
  run `make proof-research-build-tools`
- `PROOF_RESEARCH_PHASE3_COMPARE_JOBS` controls bounded parallelism across
  models; on a single local Ollama backend, start with `1` or `2`
- `researchctl phase3 compare run --jobs N` is the canonical non-make
  equivalent of that same bounded-parallelism control
- Google runs now stamp an optional `google_thinking_mode` column into both
  per-run result CSVs and shared summary CSVs; `default` preserves current
  provider-default behavior, while `minimal|low|high|budget0` become explicit
  experiment metadata
- the `make proof-research-phase3-compare` wrapper also turns non-default
  Google thinking modes into distinct Google result identities, so `budget0`,
  `low`, and `high` do not land in the same `google/<model>/...` directory or
  `run_id`

## 5. Aggregation Note

Lean4-generated theorem packs remain support-layer artifacts. Once exported
into the benchmark family, their downstream comparison outputs now use:

- summaries: `research/artifacts/result_research/by-date/<date>/<provider>/<model>/<surface>/`
- per-run results: `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/<date>/<provider>/<model>/<surface>/`
- raw outputs: `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/<date>/<provider>/<model>/<surface>/`
- per-model reports:
  `research/result_research_report/by-date/<date>/<provider>/<model>/<surface>/`
- aggregate reports:
  `research/result_research_report/by-date/<date>/_aggregate/<surface>/`

The SQLite registry under `research/artifacts/result_research/research_db/`
indexes those artifacts; it does not replace the canonical CSV and Markdown
outputs.
