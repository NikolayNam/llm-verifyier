# Runbook Phase 3 Proof Research

- **Заголовок:** **Runbook Phase 3 Proof Research**
- **Статус:** активный operator runbook
- **Дата:** 2026-03-30
- **Охват:** Lean4 support-layer runs и необязательное benchmark comparison на
  exported packs
- **Граница утверждений:** `producer reliability` и `specification pressure`

---

## 1. Назначение

Phase 3 оценивает Lean4 только через измеримую полезность:

- theorem backlog generation
- case confirmation
- specification pressure

Он **не** переносит trust boundary с Hilbert checker на Lean4.

---

## 2. Канонические входы и выходы

Lean4 project family:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/`

Canonical generated theorem backlog:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/theorems.csv`

Canonical exported benchmark pack:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/lean4-generated.csv`

Необязательные direct comparison summaries, сгруппированные по
`date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-20b/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`
- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_direct.csv`

Необязательные ND comparison summaries, сгруппированные по
`date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase3_lean4pack_nd.csv`

Необязательные comparison reports:

- per-model:
  - `research/result_research_report/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/mistral/mistral-large-2512/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase3_lean4pack_pair_20260330.md`
- aggregate:
  - `research/result_research_report/by-date/20260330/_aggregate/direct/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_direct_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/nd/hilbert-ai-verification-benchmark-nd-v1_report_phase3_lean4pack_nd_20260330.md`
  - `research/result_research_report/by-date/20260330/_aggregate/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_phase3_lean4pack_pair_20260330.md`

Примечание по текущей summary-schema:

- direct shared summaries несут `surface=direct`
- ND shared summaries несут `surface=nd`
- non-default значения `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE`
  кодируются в Google model slug, который использует `make`-wrapper,
  например `google/gemini-2-5-flash_high/...`

---

## 3. ИНСТРУКЦИЯ ДЛЯ ПОЛЬЗОВАТЕЛЯ: ЗАПУСК

Важные технические факты:

- если ты хочешь, чтобы `research-db-sync` автоматически обнаружил shared
  benchmark summaries Phase 3, сохраняй benchmark project folder в basename
  имени summary-файла
- benchmark summaries для Phase 3 теперь per-model, а не один смешанный CSV
- `make proof-research-phase3` запускает Lean4 DB-first команды, поэтому
  Lean4 hypothesis/export ledger записывается прямо в
  `research/artifacts/result_research/research_db/research_db.sqlite`
- ни `make proof-research-phase3`, ни `make proof-research-phase3-compare`
  **не** запускают автоматически benchmark registry sync, поэтому индексирование
  benchmark-family summaries/results всё ещё требует явных sync-команд из
  раздела `3.5`
- явные comparison-команды ниже сохраняют `-provider compatible` для
  локального Ollama/OpenAI-compatible транспорта, но stamp-ят
  `provider_label` в пути и CSV rows (`openai` для `gpt-oss:*`, `deepseek`
  для `deepseek-*`)
- разделение типа явно видно в обоих местах:
  - filename содержит `direct` или `nd`
  - CSV содержит `surface=direct` или `surface=nd`

### 3.1 Lean4 bootstrap + smoke-check + theorem export

Запускай эти команды из корня репозитория.

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

### 3.2 Необязательное comparison на exported theorem pack, первый проход `r01`

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

### 3.3 Повторные проходы `r02` и `r03`

Запусти те же четыре comparison-команды ещё раз, меняя только:

- `RUN_ID="${DATE_TAG}_phase3_direct_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase3_direct_<provider-safe>_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase3_nd_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase3_nd_<provider-safe>_<safe-model>_r03"`

Для повторов одной и той же модели и одной и той же surface оставляй тот же
per-model shared summary path.

### 3.4 Построить direct, ND и pair reports

Если ты запускаешь необязательное benchmark comparison на exported pack, direct
и ND Markdown reports используют ту же observed-result structure, что и в
усиленном Phase 1 reporting:

- aggregate
- entailed-only
- not-entailed-only
- per-category
- per-case hardest failures

Текущее примечание по ND-family:

- для exported Lean4 comparison pack пока не зафиксирован canonical
  theorem-specific hard-case audit target set, поэтому hard-case audit section
  сохраняется, но сейчас сообщает, что фиксированных audit targets нет

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

### 3.5 Синхронизация в shared research DB

```bash
make research-db-sync \
  RESEARCH_DB_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  RESEARCH_DB_PATH='research/artifacts/result_research/research_db/research_db.sqlite'
```

```bash
make research-db-sync-nd-v1
```

## 4. Convenience Wrappers

Если вместо явных команд выше ты предпочитаешь make-wrapper:

Bootstrap + smoke-check + backlog generation + export:

```bash
make proof-research-phase3
```

Текущее ограничение wrapper:

- `make proof-research-phase3` обновляет Lean4 DB-first ledger напрямую, но
  **не** вызывает автоматически ни `make research-db-sync`, ни
  `make research-db-sync-nd-v1`
- `make proof-research-phase3-compare` пишет только file-based benchmark
  artifacts и тоже **не** синхронизирует их автоматически в
  `research_db.sqlite`

Необязательный direct-vs-ND comparison на exported theorem pack:

```bash
make proof-research-phase3-compare
```

Полезные override-параметры:

```bash
make proof-research-phase3 PROOF_RESEARCH_PHASE3_JOB_STATEMENT='P -> P'
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_MODELS='gpt-oss:120b-cloud'
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_MODELS=''
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE=high
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_REPEATS=5
make proof-research-phase3-compare PROOF_RESEARCH_PHASE3_COMPARE_JOBS=2
make proof-research-build-tools
```

Default comparison slice:

- локальные срезы на compatible-транспорте наследуют:
  - `PROOF_RESEARCH_PHASE3_COMPARE_BASE_URL=http://localhost:11434`
- внешние display-label:
  - `openai` для `gpt-oss:*`
  - `deepseek` для `PROOF_RESEARCH_PHASE3_COMPARE_DEEPSEEK_MODELS`
- existing models:
  - `gpt-oss:120b-cloud`
- deepseek models:
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `mistral-large-2512`
- repeats: `3`

Важные текущие caveat:

- для Google и Mistral нужны явные API keys; wrapper читает
  `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_API_KEY` или экспортированные
  `GEMINI_API_KEY` / `GOOGLE_API_KEY`, и
  `PROOF_RESEARCH_PHASE3_COMPARE_MISTRAL_API_KEY` или экспортированный
  `MISTRAL_API_KEY`
- если у твоего Google project низкий free-tier RPM cap, задавай
  `PROOF_RESEARCH_PHASE3_COMPARE_GOOGLE_REQUESTS_PER_MINUTE` явно; например,
  `5` навязывает минимум 12 секунд между Google-запросами внутри benchmark
  runner
- по умолчанию comparison-wrapper также source-ит
  `research/llm_models/.env` перед разрешением provider API keys, поэтому
  colocated Gemini/Mistral credentials подхватываются автоматически при
  `make proof-research-phase3-compare`
- benchmark comparison wrapper теперь заранее собирает и переиспользует
  бинарники `contracts` и `hilbertcheck`; если хочешь прогреть tool cache
  отдельно, запусти `make proof-research-build-tools`
- Google-прогоны теперь пишут необязательный столбец `google_thinking_mode`
  и в per-run result CSV, и в shared summary CSV; `default` сохраняет текущее
  provider-default поведение, а `minimal|low|high|budget0` становятся явной
  метаданной эксперимента
- `make proof-research-phase3-compare` также превращает non-default Google
  thinking mode в отдельные Google result identity, поэтому `budget0`, `low`
  и `high` не попадают в один и тот же каталог `google/<model>/...` или
  `run_id`
- `PROOF_RESEARCH_PHASE3_COMPARE_JOBS` задаёт ограниченную параллельность по
  моделям; на одном локальном Ollama backend начинай с `1` или `2`

## 5. Примечание по агрегации

Lean4-generated theorem packs остаются support-layer артефактами. После
экспорта в benchmark family их downstream comparison outputs используют:

- summaries: `research/artifacts/result_research/by-date/<date>/<provider>/<model>/<surface>/`
- per-run results: `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/<date>/<provider>/<model>/<surface>/`
- raw outputs: `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/<date>/<provider>/<model>/<surface>/`
- per-model reports:
  `research/result_research_report/by-date/<date>/<provider>/<model>/<surface>/`
- aggregate reports:
  `research/result_research_report/by-date/<date>/_aggregate/<surface>/`

SQLite registry под `research/artifacts/result_research/research_db/`
индексирует эти артефакты; он не заменяет канонические CSV и Markdown outputs.
