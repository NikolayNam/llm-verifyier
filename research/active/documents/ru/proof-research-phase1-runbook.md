# Runbook Phase 1 Proof Research

- **Заголовок:** **Runbook Phase 1 Proof Research**
- **Статус:** активный operator runbook
- **Дата:** 2026-03-30
- **Охват:** повторные direct Hilbert-прогоны для текущего `v2 held-out` slice
- **Граница утверждений:** только `checker soundness`

Практика:
make proof-research-phase1 PROOF_RESEARCH_PHASE1_GOOGLE_MODELS='gemini-3.1-pro-preview gemini-2.5-flash' \
PROOF_RESEARCH_PHASE1_MISTRAL_MODELS='labs-leanstral-2603 mistral-large-2512 ministral-14b-2512' \
PROOF_RESEARCH_PHASE1_REPEATS=10 \
PROOF_RESEARCH_PHASE1_JOBS=12 \
PROOF_RESEARCH_PHASE1_GOOGLE_REQUESTS_PER_MINUTE=10


---

## 1. Назначение

Phase 1 отвечает на первый жёсткий вопрос:

> сохраняется ли `false_accept = 0` при расширении покрытия, а деградация
> уходит в refusal / request / schema / parse / kernel / format buckets, а не
> в небезопасное принятие?

Trusted component:

- только финальный Hilbert checker

Этот runbook **не** делает утверждений про ND, Lean4 или будущие семейства
certificate/runtime.

---

## 2. Канонические входы и выходы

Текущий канонический input:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/cases.csv`

Общие summary-выходы, сгруппированные по `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/deepseek/deepseek-v3-1-671b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/direct/hilbert-ai-verification-benchmark-v2-held-out_result_summary_phase1_direct.csv`

Research-facing report outputs:

- per-model:
  - `research/result_research_report/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-v2-held-out_report_phase1_direct_20260330.md`
- aggregate:
  - `research/result_research_report/by-date/20260330/_aggregate/direct/hilbert-ai-verification-benchmark-v2-held-out_report_phase1_direct_20260330.md`

Family-local per-run outputs теперь лежат под:

- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/20260330/<provider-label>/<model>/direct/`
- `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/20260330/<provider-label>/<model>/direct/`

Примечание по текущей summary-schema:

- каждая shared summary row теперь несёт `surface=direct`
- non-default значения `PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE`
  кодируются в Google model slug, который использует `make`-wrapper,
  например `google/gemini-2-5-flash_budget0/...`

---

## 3. ИНСТРУКЦИЯ ДЛЯ ПОЛЬЗОВАТЕЛЯ: ЗАПУСК

Важные технические факты:

- если ты хочешь, чтобы `research-db-sync` автоматически обнаружил shared
  summary-файлы Phase 1, сохраняй benchmark project folder в basename имени
  summary-файла
- Phase 1 теперь пишет один shared summary file на модель, а не один смешанный
  catch-all CSV
- `make proof-research-phase1` пишет только файловые артефакты (`result`,
  `raw`, shared `summary`, Markdown `report`) и **не** синхронизирует
  автоматически
  `research/artifacts/result_research/research_db/research_db.sqlite`
- явные команды первого прохода ниже сохраняют `-provider compatible` для
  локального Ollama/OpenAI-compatible транспорта, но stamp-ят
  `provider_label` в пути и CSV rows (`openai` для `gpt-oss:*`, `deepseek`
  для `deepseek-*`)
- разделение типа явно видно в обоих местах:
  - filename содержит `direct`
  - CSV содержит `surface=direct`

### 3.1 Первый проход `r01`

Запускай эти команды из корня репозитория.

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

### 3.2 Повторные проходы до `r08`

Запусти те же четыре команды ещё раз, меняя только:

- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r04"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r05"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r06"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r07"`
- `RUN_ID="${DATE_TAG}_phase1_direct_<provider-safe>_<safe-model>_r08"`

Для всех повторов одной и той же модели используй один и тот же путь к
per-model shared summary file.

### 3.3 Построить aggregate direct report

`hilbert-benchmark-report` теперь принимает comma-separated список summary
files. Итоговый Markdown report теперь включает:

- aggregate
- entailed-only
- not-entailed-only
- per-category
- per-case hardest failures
- hard-case audit targets для `V2E01`, `V2E06`, `V2E08`, `V2E07`, `V2E12`, `V2N07`, `V2N09`

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

### 3.4 Необязательная aggregate terminal summary

Terminal summary теперь печатает:

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

### 3.5 Синхронизация в shared research DB

```bash
make research-db-sync \
  RESEARCH_DB_PROJECT_FOLDER='hilbert-ai-verification-benchmark-v2-held-out' \
  RESEARCH_DB_PATH='research/artifacts/result_research/research_db/research_db.sqlite'
```

## 4. Convenience Wrapper

Если вместо явных команд выше ты предпочитаешь make-wrapper:

Запускай из корня репозитория:

```bash
make proof-research-phase1
```

Текущее ограничение wrapper:

- `make proof-research-phase1` **не** вызывает `make research-db-sync`
  автоматически, поэтому benchmark rows из Phase 1 не индексируются в
  `research_db.sqlite`, пока ты не выполнишь явный sync-шаг из раздела `3.5`

Полезные override-параметры:

```bash
make proof-research-phase1 PROOF_RESEARCH_PHASE1_REPEATS=8
make proof-research-phase1 PROOF_RESEARCH_PHASE1_JOBS=2
make proof-research-phase1 PROOF_RESEARCH_PHASE1_OPENAI_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE1_DEEPSEEK_MODELS='deepseek-v3.1:671b-cloud'
make proof-research-phase1 PROOF_RESEARCH_PHASE1_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE1_MISTRAL_MODELS=''
make proof-research-phase1 PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE=budget0
make proof-research-phase1 PROOF_RESEARCH_PHASE1_CASE_ID=V2E12
make proof-research-phase1 PROOF_RESEARCH_PHASE1_LIMIT=8
make proof-research-build-tools
```

Срез по умолчанию:

- локальные срезы на compatible-транспорте наследуют:
  - `PROOF_RESEARCH_PHASE1_BASE_URL=http://localhost:11434`
- внешние display-label:
  - `openai` для `gpt-oss:*`
  - `deepseek` для `deepseek-*`
- prompt: `hilbert-ai-verification-benchmark-v1.3`
- existing models:
  - `gpt-oss:120b-cloud`
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE1_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE1_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `mistral-large-2512`
- repeats: `8`

Важные текущие caveat:

- для Google и Mistral нужны явные API keys; wrapper читает
  `PROOF_RESEARCH_PHASE1_GOOGLE_API_KEY` или экспортированные
  `GEMINI_API_KEY` / `GOOGLE_API_KEY`, и
  `PROOF_RESEARCH_PHASE1_MISTRAL_API_KEY` или экспортированный
  `MISTRAL_API_KEY`
- если у твоего Google project низкий free-tier RPM cap, задавай
  `PROOF_RESEARCH_PHASE1_GOOGLE_REQUESTS_PER_MINUTE` явно; например, `5`
  навязывает минимум 12 секунд между Google-запросами внутри benchmark
  runner
- по умолчанию wrapper также source-ит `research/llm_models/.env` перед
  разрешением provider API keys, поэтому colocated Gemini/Mistral credentials
  подхватываются автоматически при `make proof-research-phase1`
- wrapper теперь заранее собирает и переиспользует бинарники `contracts` и
  `hilbertcheck`; если хочешь прогреть tool cache отдельно, запусти
  `make proof-research-build-tools`
- Google-прогоны теперь пишут необязательный столбец `google_thinking_mode`
  и в per-run result CSV, и в shared summary CSV; `default` сохраняет текущее
  provider-default поведение, а `minimal|low|high|budget0` становятся явной
  метаданной эксперимента
- `make proof-research-phase1` также превращает non-default Google thinking
  mode в отдельные Google result identity, поэтому `budget0` и `high` не
  попадают в один и тот же каталог `google/<model>/...` или `run_id`
- `PROOF_RESEARCH_PHASE1_JOBS` задаёт ограниченную параллельность по моделям;
  для одного локального Ollama backend начинай с `1` или `2`

---

## 5. Примечание по агрегации

Этот runbook намеренно пишет fresh outputs в:

- summaries: `research/artifacts/result_research/by-date/<date>/<provider>/<model>/direct/`
- per-run results: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/by-date/<date>/<provider>/<model>/direct/`
- raw outputs: `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/by-date/<date>/<provider>/<model>/direct/`
- per-model reports:
  `research/result_research_report/by-date/<date>/<provider>/<model>/direct/`
- aggregate reports:
  `research/result_research_report/by-date/<date>/_aggregate/direct/`

`research-db-sync` теперь индексирует и старый flat summary layout, и новый
dated layout.
