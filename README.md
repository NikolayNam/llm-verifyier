# Research Workspace

Короткий README для запуска `researchctl` из этого репозитория.

Исходные артефакты исследования можно скачать тут: https://drive.google.com/file/d/1jA5GSNd9fRWD7Yhk4qKxCGrow4E-rVnX/view?usp=drive_link
Весит он 1.5 Gb



## Что запускать

Канонический entrypoint из корня репозитория:

```powershell
go -C platform-tooling run ./cmd/researchctl ...
```

Самые частые команды:

### 1. Посмотреть план

```powershell
go -C platform-tooling run ./cmd/researchctl phase1 plan --local-config research/config/direct/phase1-expanded-100.yaml
```

Это ничего не исполняет у модели. Команда показывает:

- какой `run_id` будет использован;
- куда запишется manifest;
- куда пойдут summary и report;
- сколько всего benchmark-job будет создано;
- какой `parallel_jobs` реально выбран.

### 2. Запустить phase1

```powershell
go -C platform-tooling run ./cmd/researchctl phase1 run --local-config research/config/direct/phase1-expanded-100.yaml
```

Если нужно явно переопределить параллелизм:

```powershell
go -C platform-tooling run ./cmd/researchctl phase1 run --local-config research/config/direct/phase1-expanded-100.yaml --jobs 2
```

Если `--jobs` не передан, берётся `benchmarks.phase1.jobs` из YAML.

### 3. Построить markdown-отчёт по одному summary CSV

```powershell
go -C platform-tooling run ./cmd/researchctl report benchmark \
  --phase phase1 \
  --local-config research/config/direct/phase1-expanded-100.yaml \
  --run-id <run-id>
```

### 4. Построить meta-отчёт по одному или нескольким summary CSV

```powershell
go -C platform-tooling run ./cmd/researchctl report meta \
  --phase phase1 \
  --local-config research/config/direct/phase1-expanded-100.yaml \
  --summary research/artifacts/result_research/waves/phase1_<run-id>.csv
```

Или автоматически взять последние summary:

```powershell
go -C platform-tooling run ./cmd/researchctl report meta \
  --phase phase1 \
  --local-config research/config/direct/phase1-expanded-100.yaml \
  --latest 1
```

## Что требует запуск

Минимально нужно:

- Go;
- рабочий workspace root репозитория;
- корректный config-файл;
- доступный LLM endpoint, указанный в `transports.*.base_url`;
- prompt-файлы, если запускается именно `run`, а не только `report`.

Для `phase1-expanded-100` основной профиль сейчас:

- [research/config/direct/phase1-expanded-100.yaml](/C:/Users/nokclock/Documents/GitHub/llm-verifyier/research/config/direct/phase1-expanded-100.yaml)

Файл `--local-config` должен реально существовать. Если путь неверный, команда теперь падает с явной ошибкой и не откатывается молча к `default.yaml`.

Пример неверного пути:

```powershell
--local-config research/config/local.phase1-expanded-100.yaml
```

Такого файла больше нет. Правильный путь:

```powershell
--local-config research/config/direct/phase1-expanded-100.yaml
```

## Где лежат конфиги

Канонический layout:

- `research/config/default.yaml`
- `research/config/workflow/`
- `research/config/direct/`
- `research/config/nd/`
- `research/config/compositional/`

Чаще всего:

- direct / `phase1`: `research/config/direct/...`
- natural deduction / `nd`: `research/config/nd/...`
- парные workflow (`phase2`, `phase3 compare`): `research/config/workflow/...`
- compositional surfaces: `research/config/compositional/...`

## Где что появляется

### После `plan`

Обычно создаётся manifest:

- `research/artifacts/result_research/manifests/<phase>_<run-id>.json`

В stdout также печатаются будущие пути для:

- summary CSV;
- markdown report;
- model catalog CSV.

### После `run`

Появляются:

- manifest:
  `research/artifacts/result_research/manifests/...`
- summary CSV:
  `research/artifacts/result_research/...`
- markdown report:
  `research/result_research_report_v1/...`
- per-job result CSV:
  `research/artifacts/<project>/result/...`
- raw provider outputs:
  `research/artifacts/<project>/raw/...`

Для direct `phase1` типичные директории такие:

- summary: `research/artifacts/result_research/waves/`
- report: `research/result_research_report_v1/direct/`

### После `report meta`

Появляется markdown-файл:

- `research/result_research_report_v1/summary/meta/...`

### После `report cases`

Появляется markdown-файл:

- `research/result_research_report_v1/summary/cases/...`

Пояснение к столбцам таблицы:
Pack — идентификатор набора кейсов. Обычно это key пака, практически чаще всего путь/имя входного cases-файла.
Cases — число различных кейсов в этом паке (DistinctCases), а не число запусков.
Observations — число всех наблюдений по этому паку, то есть всех result-rows по всем выбранным run/model/repeat. Это главный знаменатель для rate-метрик.
Pass — сколько наблюдений попало в bucket pass.
Pass Rate — доля Pass / Observations, см. hilbert_benchmark_case_report.go:109.
False Refusal — модель отказалась (not_derivable) там, где кейс на самом деле entailed, см. hilbert_benchmark_run.go:161 и ND-вариант nd_hilbert_benchmark.go:394.
False Accept — модель выдала сертификат, verifier его принял, но сам кейс был not_entailed, см. hilbert_benchmark_run.go:1122.
Schema — число наблюдений с schema_failure: JSON/структура proof/certificate не проходит schema-level проверку, см. hilbert_benchmark_run.go:1131 и nd_hilbert_benchmark.go:639.
Parse — число наблюдений с parse_failure: структура в целом допустима, но формулы/шаги не распарсились, см. hilbert_benchmark_run.go:1137 и nd_hilbert_benchmark.go:634.
Kernel — число наблюдений с kernel_failure: объект дошёл до верификации, но ядро его отвергло; в ND сюда также попадает lowering validation failure, см. hilbert_benchmark_run.go:1144 и nd_hilbert_benchmark.go:659.
Contract — число наблюдений с contract_failure: сертификат может быть формально принят, но нарушает benchmark contract по метаданным (certificate_version, context.domain, context.generator, rule_pack, syntax), см. hilbert_benchmark_run.go:1081.
Format — число наблюдений с format_failure: модель вернула мусор/обёртку/не тот тип output, например invalid_json или other, см. hilbert_benchmark_run.go:167 и nd_hilbert_benchmark.go:399.


## Как читать вывод `plan`

В `plan` есть два разных числа:

- `jobs` = сколько benchmark-job вообще будет создано;
- `parallel_jobs` = сколько из них можно исполнять одновременно.

Пример:

- `jobs: 8`
- `parallel_jobs: 3`

Это означает не "8 потоков", а "всего 8 задач, одновременно можно 3".

## Самый простой рабочий сценарий

### Посмотреть план

```powershell
go -C platform-tooling run ./cmd/researchctl phase1 plan --local-config research/config/direct/phase1-expanded-100.yaml
```

### Запустить benchmark

```powershell
go -C platform-tooling run ./cmd/researchctl phase1 run --local-config research/config/direct/phase1-expanded-100.yaml
```

### Построить meta-отчёт по последнему summary

```powershell
go -C platform-tooling run ./cmd/researchctl report meta --phase phase1 --local-config research/config/direct/phase1-expanded-100.yaml --latest 1
```

## Если команда ведёт себя не так, как ожидается

Проверьте по порядку:

1. Точно ли существует `--local-config`.
2. Тот ли это grouped path, а не старый legacy path.
3. Что показывает `phase1 plan` для `local_config`, `jobs` и `parallel_jobs`.
4. Что endpoint из `transports.*.base_url` реально доступен.
5. Что модели из `families.*.models` действительно доступны на этом endpoint.

Если `plan` показывает не тот `input_file`, не тот `parallel_jobs` или не тот набор моделей, почти всегда проблема в том, что был передан неправильный config-путь.
