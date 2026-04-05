# Runbook Phase 2 Proof Research

- **Заголовок:** **Runbook Phase 2 Proof Research**
- **Статус:** активный operator runbook
- **Дата:** 2026-03-30
- **Охват:** парный эксперимент `direct Hilbert` vs
  `ND -> deterministic lowering -> Hilbert`
- **Граница утверждений:** `producer reliability` и `interface advantage`

---

## 1. Назначение

Phase 2 задаёт второй жёсткий вопрос:

> если ND помогает, то выигрыш proof-theoretic, или это только более удобный
> authoring/interface layer?

Trusted component:

- только финальный Hilbert checker

Untrusted components:

- direct Hilbert producer
- ND producer
- ND validation path
- deterministic lowering layer

---

## 2. Канонические входы и выходы

Canonical shared theorem pack:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/pilot_shared_20260327.csv`

Shared direct summaries, сгруппированные по `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv`
- `research/artifacts/result_research/by-date/20260330/google/gemini-2-5-flash/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_direct.csv`

Shared ND summaries, сгруппированные по `date/provider/model/surface`:

- `research/artifacts/result_research/by-date/20260330/openai/gpt-oss-120b-cloud/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv`
- `research/artifacts/result_research/by-date/20260330/mistral/mistral-large-2512/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_phase2_nd.csv`

Per-run result CSVs, сгруппированные по `date/provider/model/surface`:

- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/20260330/openai/gpt-oss-120b-cloud/direct/result_20260330_phase2_direct_openai_gpt-oss-120b-cloud_r01.csv`
- `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/20260330/google/gemini-2-5-flash/nd/result_20260330_phase2_nd_google_gemini-2-5-flash_r01.csv`

Raw outputs, сгруппированные по `date/provider/model/surface`:

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

Примечание по текущей summary-schema:

- direct shared summaries несут `surface=direct`
- ND shared summaries несут `surface=nd`
- non-default значения `PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE`
  кодируются в Google model slug, который использует `make`-wrapper,
  например `google/gemini-2-5-flash_low/...`

---

## 3. ИНСТРУКЦИЯ ДЛЯ ПОЛЬЗОВАТЕЛЯ: ЗАПУСК

Важные технические факты:

- если ты хочешь, чтобы `research-db-sync` автоматически обнаружил shared
  summary-файлы Phase 2, сохраняй benchmark project folder в basename имени
  summary-файла
- shared summaries для Phase 2 теперь per-model, а не один смешанный CSV
- `make proof-research-phase2` пишет только файловые артефакты (`result`,
  `raw`, shared `summary`, Markdown `report`) и **не** синхронизирует
  автоматически
  `research/artifacts/result_research/research_db/research_db.sqlite`
- разделение типа явно видно в обоих местах:
  - filename содержит `direct` или `nd`
  - CSV содержит `surface=direct` или `surface=nd`

### 3.1 Первый проход `r01`

Запускай эти команды из корня репозитория.

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

### 3.2 Повторные проходы `r02` и `r03`

Запусти те же четыре команды ещё раз, меняя только:

- `RUN_ID="${DATE_TAG}_phase2_direct_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase2_direct_<safe-model>_r03"`
- `RUN_ID="${DATE_TAG}_phase2_nd_<safe-model>_r02"`
- `RUN_ID="${DATE_TAG}_phase2_nd_<safe-model>_r03"`

Для повторов одной и той же модели и одной и той же surface оставляй тот же
per-model shared summary path.

### 3.3 Построить direct, ND и pair reports

`hilbert-benchmark-report` для этого theorem-pack family теперь рендерит ту же
observed-result structure, что используется в усилённом Phase 1 reporting:

- aggregate
- entailed-only
- not-entailed-only
- theorem-synthesis focus
- foundational-category audit для `assumption_import` и `single_mp`
- per-category
- ND pipeline trace
- per-case hardest failures

`hilbert-benchmark-research-report` теперь тоже включает те же aggregate и
observed-slice sections, поэтому парное исследование больше не уже, чем
single-surface direct/ND reports.

Текущее примечание по ND-family:

- пока не зафиксирован canonical theorem-specific hard-case audit target set,
  поэтому report сохраняет hard-case audit section, но сейчас он сообщает, что
  фиксированного набора audit targets нет
- если ты прогнал расширенный model slice сверх двух явных примеров ниже,
  допиши соответствующие per-model summary files в `DIRECT_SUMMARY_SOURCES` и
  `ND_SUMMARY_SOURCES` перед генерацией reports
- если существующие ND per-run result CSV были произведены ещё до добавления
  явной pipeline-stage telemetry, секция `ND pipeline trace` классифицируется
  эвристически из `raw_output_kind` и записанных schema/parse/kernel statuses;
  такие строки нужно трактовать как inferred legacy staging, а не как
  first-class stage logs

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

### 3.4 Синхронизация в shared research DB

```bash
make research-db-sync-nd-v1
```

## 4. Convenience Wrapper

Если вместо явных команд выше ты предпочитаешь make-wrapper:

Запускай полный paired slice из корня репозитория:

```bash
make proof-research-phase2
```

Текущее ограничение wrapper:

- `make proof-research-phase2` **не** вызывает
  `make research-db-sync-nd-v1` автоматически, поэтому benchmark rows из
  Phase 2 не индексируются в `research_db.sqlite`, пока ты не выполнишь
  явный sync-шаг из раздела `3.4`

Полезные override-параметры:

```bash
make proof-research-phase2 PROOF_RESEARCH_PHASE2_REPEATS=5
make proof-research-phase2 PROOF_RESEARCH_PHASE2_JOBS=2
make proof-research-phase2 PROOF_RESEARCH_PHASE2_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE2_GOOGLE_MODELS='' PROOF_RESEARCH_PHASE2_MISTRAL_MODELS=''
make proof-research-phase2 PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE=low
make proof-research-phase2 PROOF_RESEARCH_PHASE2_CASE_ID=NDI05
make proof-research-phase2 PROOF_RESEARCH_PHASE2_LIMIT=8
make proof-research-phase2 PROOF_RESEARCH_PHASE2_MODELS='gpt-oss:120b-cloud' PROOF_RESEARCH_PHASE2_DEEPSEEK_MODELS='deepseek-v3.1:671b-cloud' PROOF_RESEARCH_PHASE2_GOOGLE_MODELS='gemini-2.5-flash' PROOF_RESEARCH_PHASE2_MISTRAL_MODELS='mistral-large-2512' PROOF_RESEARCH_PHASE2_REPEATS=10
make proof-research-phase2 PROOF_RESEARCH_PHASE2_REPEATS=10 PROOF_RESEARCH_PHASE2_GOOGLE_API_KEY="$GEMINI_API_KEY" PROOF_RESEARCH_PHASE2_MISTRAL_API_KEY="$MISTRAL_API_KEY"
make proof-research-build-tools
```

Срез по умолчанию:

- локальные срезы на compatible-транспорте наследуют:
  - `PROOF_RESEARCH_PHASE2_BASE_URL=http://localhost:11434`
- внешние display-label:
  - `openai` для `gpt-oss:*`
  - `deepseek` для `PROOF_RESEARCH_PHASE2_DEEPSEEK_MODELS`
- direct prompt: `hilbert-ai-verification-benchmark-v1.3`
- ND prompt: `hilbert-ai-verification-benchmark-nd-v1.1`
- existing models:
  - `gpt-oss:120b-cloud`
- deepseek models:
  - `deepseek-v3.1:671b-cloud`
- google models:
  - `gemini-2.5-flash`
- Google thinking mode:
  - `PROOF_RESEARCH_PHASE2_GOOGLE_THINKING_MODE=default`
- optional Google RPM throttle:
  - `PROOF_RESEARCH_PHASE2_GOOGLE_REQUESTS_PER_MINUTE=5`
- mistral models:
  - `mistral-large-2512`
- repeats: `3`

Важные текущие caveat:

- **не** используй `make hilbert-benchmark-rerun-nd` для canonical Phase 2
  run, потому что этот convenience wrapper всё ещё указывает на старый ND
  prompt profile, а не на `hilbert-ai-verification-benchmark-nd-v1.1`
- если у твоего Google project низкий free-tier RPM cap, задавай
  `PROOF_RESEARCH_PHASE2_GOOGLE_REQUESTS_PER_MINUTE` явно; например, `5`
  навязывает минимум 12 секунд между Google-запросами внутри benchmark
  runner
- текущий wrapper прогоняет по одному theorem pack за раз; если нужен второй
  synthesis-heavy pack, сначала materialize реальный theorem CSV для него, а
  затем запускай отдельный Phase 2 slice против этого explicit pack path
- `proof-research-phase2` теперь включает provider в `run_id` и summary
  filenames, поэтому mixed-provider reruns не конфликтуют между семействами
  `google`, `mistral`, `openai` и `deepseek`
- `proof-research-phase2` теперь пишет per-run results, raw outputs,
  summaries и markdown reports в каталоги формата
  `by-date/<date>/<provider>/<model>/<surface>/` вместо старого flat CSV
  layout
- wrapper теперь заранее собирает и переиспользует бинарники `contracts` и
  `hilbertcheck`; если хочешь прогреть tool cache отдельно, запусти
  `make proof-research-build-tools`
- Google-прогоны теперь пишут необязательный столбец `google_thinking_mode`
  и в per-run result CSV, и в shared summary CSV; `default` сохраняет текущее
  provider-default поведение, а `minimal|low|high|budget0` становятся явной
  метаданной эксперимента
- `make proof-research-phase2` также превращает non-default Google thinking
  mode в отдельные Google result identity, поэтому `budget0`, `low` и `high`
  не попадают в один и тот же каталог `google/<model>/...` или `run_id`
- для Google и Mistral нужны явные API keys; wrapper читает
  `PROOF_RESEARCH_PHASE2_GOOGLE_API_KEY` или экспортированные
  `GEMINI_API_KEY` / `GOOGLE_API_KEY`, и
  `PROOF_RESEARCH_PHASE2_MISTRAL_API_KEY` или экспортированный
  `MISTRAL_API_KEY`
- по умолчанию wrapper также source-ит `research/llm_models/.env` перед
  разрешением provider API keys, поэтому colocated Gemini/Mistral credentials
  подхватываются автоматически при `make proof-research-phase2`
- `PROOF_RESEARCH_PHASE2_JOBS` задаёт ограниченную параллельность по моделям;
  на одном локальном Ollama backend начинай с `1` или `2`

---

## 5. Примечание по агрегации

Phase 2 намеренно хранит outputs в иерархическом operator layout:

- summaries в
  `research/artifacts/result_research/by-date/<date>/<provider>/<model>/<surface>/`
- per-run results в
  `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/by-date/<date>/<provider>/<model>/<surface>/`
- raw outputs в
  `research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/by-date/<date>/<provider>/<model>/<surface>/`
- per-model reports в
  `research/result_research_report/by-date/<date>/<provider>/<model>/<surface>/`
- aggregate reports в
  `research/result_research_report/by-date/<date>/_aggregate/<surface>/`

Дальше comparative markdown interpretation строится из этих per-surface summary
inputs в dated `pair/` report directory.
