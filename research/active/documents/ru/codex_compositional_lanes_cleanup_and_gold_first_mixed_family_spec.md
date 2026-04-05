# Codex Spec: cleanup starter compositional lanes и gold-first mixed-family probe

## Статус
Draft for implementation

## Контекст
Текущие starter compositional lanes уже дали полезный исследовательский сигнал, но ещё не дают чистого и интерпретируемого основания для следующего шага уровня `mixed-family reuse`.

Основная проблема не в checker boundary и не в negative controls, а в том, что compositional lanes пока смешивают несколько разных failure modes:

- локальное построение леммы;
- import resolution;
- self-produced reuse;
- gold-supported final composition;
- местами — плохую stage semantics.

Перед любым `mixed-family reuse` необходимо:

1. добить интерпретацию starter compositional lanes;
2. почистить stage semantics;
3. зафиксировать, что именно измеряется в следующем mixed-family эксперименте.

Этот документ задаёт реализацию этих трёх шагов.

---

# 1. Цель

Подготовить репозиторий и исследовательскую поверхность к корректному следующему шагу в compositional line без расширения runtime trust boundary.

Нужный результат:

- есть единая интерпретация для `linear import`, `depth ladder`, `branching`;
- stage semantics очищены от случаев, где локальные `prove`-стадии семантически невалидны;
- сформулирован и реализован узкий `gold-first mixed-family probe`, который отвечает только на вопрос о reuse между proof families при корректно заданных леммах.

---

# 2. Не-цели

Этот документ **не** ставит цель:

- доказать, что self-produced compositional reuse уже работает;
- расширить runtime trust boundary;
- запускать deeper mixed-family ladder;
- смешивать `gold` и `model` режимы в одном первом mixed-family claim;
- менять formal checker semantics или proof verification rules.

---

# 3. Главный принцип

Все изменения выполняются при сохранении следующей границы:

- Hilbert checker остаётся единственным trust-conferring verifier;
- новые lanes дают только research-only evidence;
- любые новые mixed-family claims допускаются только после clean interpretation starter lanes.

---

# 4. Deliverables

По окончании работы в репозитории должны появиться или быть обновлены следующие артефакты.

## 4.1 Документы

1. `research/active/documents/compositional-lanes-interpretation.md`
2. `research/active/documents/compositional-stage-semantics-cleanup.md`
3. `research/active/documents/mixed-family-gold-first-design.md`
4. обновление `research/README.md`
5. при необходимости обновление `research-db-v1.md`

## 4.2 Конфигурация и case packs

1. исправленные case packs для starter lanes, если stage semantics некорректна;
2. новый case pack для gold-first mixed-family probe;
3. новый local config для gold-first mixed-family probe.

## 4.3 Код

1. если нужен новый standalone surface — добавить его как отдельный research-only lane;
2. если нового surface не требуется, реализовать probe поверх существующего generic chain protocol;
3. db sync и report generation должны корректно видеть новый artifact group.

---

# 5. Workstream A — добить интерпретацию starter compositional lanes

## 5.1 Задача

Нужно не просто хранить три отдельных отчёта, а получить **единый интерпретационный слой** по:

- `compositional_assumption_import_phase1`
- `compositional_depth_ladder_phase1`
- `compositional_branching_phase1`

## 5.2 Вопросы, на которые документ обязан ответить

Для каждого lane документ обязан явно ответить:

1. где рушится `s1`;
2. где рушится `import resolution`;
3. где живёт `gold-final`;
4. где умирает `model-final`;
5. сохраняется ли negative discipline;
6. есть ли operationally usable chain;
7. где failure является stage-semantic artifact, а где реальным compositional failure.

## 5.3 Требуемая структура интерпретационного документа

Документ `compositional-lanes-interpretation.md` должен содержать разделы:

### A. Lane inventory
- lane name
- current status
- question under test
- current claim boundary

### B. Stage contract per lane
- stage ids
- stage roles
- provenance mode semantics
- authoritative summary layer

### C. Result summary per lane
Для каждого lane:
- aggregate pass band
- negative control status
- `s1` status
- import-stage status
- gold-final status
- model-final status
- all-stages status

### D. Cross-lane synthesis
Нужно сформулировать три итоговых вывода:

1. что ломается одинаково во всех starter lanes;
2. что специфично только для отдельного lane;
3. что из этого уже можно трактовать как общий compositional bottleneck.

## 5.4 Acceptance criteria для Workstream A

Работа считается завершённой, если:

- для каждого starter lane есть единая краткая карточка интерпретации;
- различие между aggregate и authoritative stage-level reading явно зафиксировано там, где это нужно;
- по всем трём lanes есть единый summary section `shared failure pattern`;
- документ можно читать отдельно от сырых markdown-отчётов.

---

# 6. Workstream B — cleanup stage semantics

## 6.1 Задача

Нужно удалить или исправить stage-cases, где текущая формулировка задачи делает lane плохо интерпретируемым.

Главный целевой случай сейчас — starter branching.

## 6.2 Правило cleanup

Если stage имеет:

- `expected_behavior=prove`
- пустой `assumptions_json`
- пустой `imported_lemmas_json`
- и goal не является допустимой theorem-level целью для данного surface,

то такой stage нельзя оставлять как обычный локальный `prove`-шаг без явной оговорки.

## 6.3 Требуемая классификация stage semantics

Документ `compositional-stage-semantics-cleanup.md` должен вводить обязательную классификацию stage types:

1. `local_prove_stage`
   - stage реально должен быть доказуем из заданного локального контекста;

2. `trusted_resource_stage`
   - stage не является локальным prove-task, а представляет заранее доступный корректный ресурс;

3. `gold_import_stage`
   - stage проверяет использование корректно заданных imported lemmas;

4. `model_import_stage`
   - stage проверяет reuse verifier-confirmed model-produced artifacts;

5. `negative_control_stage`
   - stage проверяет compositional discipline и не должен случайно поощрять тупое склеивание.

## 6.4 Что нужно сделать по branching

Для `compositional_branching_phase1` Codex должен:

1. пройти case pack построчно;
2. отметить стадии, которые сейчас выглядят как semantically invalid local prove tasks;
3. предложить для каждой такой стадии один из вариантов:
   - перевести в `trusted_resource_stage`;
   - перевести в `gold_import_stage`;
   - удалить из starter slice;
   - переписать assumptions/imports так, чтобы stage стал честным local prove task.

## 6.5 Предпочтительное решение для branching starter slice

Если нет сильной причины сохранять full local branch construction, то phase1 branching следует перевести в более чистую форму:

- убрать семантически сомнительные local prove stages;
- оставить branching как `gold-final / model-final / negative` diagnostic lane;
- использовать корректно заданные branch lemmas как trusted/gold resources;
- не смешивать в одной phase1 постановке theorem synthesis и branching reuse.

## 6.6 Acceptance criteria для Workstream B

Работа считается завершённой, если:

- для каждого starter lane нет stage-cases с сомнительной семантикой без явной классификации;
- branching pack либо очищен, либо документирован как intentionally diagnostic with non-local resource semantics;
- в документации явно указано, какой stage type имеет каждая starter stage role.

---

# 7. Workstream C — определить, что именно измеряет mixed-family reuse

## 7.1 Ключевая развилка

Перед реализацией mixed-family reuse нужно зафиксировать, какой именно вопрос ставится.

Есть два разных вопроса.

### Вопрос 1 — gold-first mixed-family reuse

> Может ли модель использовать корректно заданные леммы из другой proof family при построении новой композиции?

Это хороший и безопасный первый probe.

### Вопрос 2 — self-produced mixed-family reuse

> Может ли модель сама вывести леммы в family A, затем перенести их в family B и использовать дальше?

Этот вопрос сейчас преждевременен.

## 7.2 Решение для текущего этапа

В текущем цикле Codex должен реализовывать **только вариант 1**:

- `gold-first mixed-family probe`
- research-only
- без runtime claims
- без self-produced imported-family transfer

## 7.3 Формальный question under test

Нужно зафиксировать в `mixed-family-gold-first-design.md` следующий вопрос:

> Under the current direct-Hilbert research surface, can the model compositionally use verifier-correct imported lemmas originating from a different proof family, while preserving negative compositional discipline and without relying on self-produced cross-family artifact reuse?

Русская формулировка:

> Может ли модель в текущем direct-Hilbert research surface композиционно использовать корректно заданные imported lemmas, происходящие из другой proof family, при сохранении отрицательной дисциплины и без опоры на self-produced cross-family reuse?

---

# 8. Design для gold-first mixed-family probe

## 8.1 Scope

Первая версия mixed-family probe должна быть очень узкой.

### Обязательно
- только `gold` path;
- только depth-1;
- только 1 final composition stage на chain;
- только 1 negative twin на chain;
- только 2–3 proof families максимум.

### Нельзя включать в phase1
- self-produced model import across families;
- depth-2+;
- branching + mixed-family одновременно;
- theoremized composition;
- mixed-family with dynamic artifact reuse.

## 8.2 Recommended starter families

Рекомендуемый минимальный набор:

1. `linear_chain`
2. `renamed_linear`
3. `distractor_linear`

Идея:
- леммы берутся из одной family;
- финальная цель находится в другой structurally equivalent or lightly shifted family;
- negative twin проверяет, что модель не склеивает всё подряд.

## 8.3 Stage protocol

Минимальный stage protocol для mixed-family gold-first phase1:

1. `res`
   - resource declaration row or implicit trusted resource set;
2. `fg`
   - final gold composition using imported lemmas from another family;
3. `neg`
   - negative twin.

Если нужен полный chain protocol, допускается альтернативный вариант:

- `prep`
- `fg`
- `neg`

Но без `model` stage в phase1.

## 8.4 Required metrics

Нужно считать минимум:

- `pass(fg)`
- `pass(neg)`
- `conditional_fg_pass`
- `composition_gap_gold`
- failure localization
- negative discipline status

## 8.5 Success criteria

Для phase1 mixed-family gold-first probe хороший сигнал:

- `fg >= 0.80` для лучших моделей;
- `neg` остаётся высоким;
- `false_accept = 0`;
- failure mix не доминируется бессмысленной schema/request нестабильностью.

Сильный сигнал:

- `fg >= 0.90`
- negative twin remains high
- no stage-semantic ambiguity

## 8.6 Kill criteria

Остановить expansion, если:

- even gold-first mixed-family probe collapses mostly at stage semantics;
- negative twin деградирует;
- failure mix выглядит как noise from poor pack design, а не как interpretable compositional signal;
- отличить межсемейный transfer failure от обычного local failure нельзя.

---

# 9. Требуемые изменения по файлам

## 9.1 Документы

### Создать
- `research/active/documents/compositional-lanes-interpretation.md`
- `research/active/documents/compositional-stage-semantics-cleanup.md`
- `research/active/documents/mixed-family-gold-first-design.md`

### Обновить
- `research/README.md`
- при необходимости `research-db-v1.md`

## 9.2 Case packs

### Проверить и при необходимости исправить
- `cases/compositional-assumption-import-phase1.csv`
- `cases/compositional-depth-ladder-phase1.csv`
- `cases/compositional-branching-phase1.csv`

### Создать
- `cases/compositional-mixed-family-gold-first-phase1.csv`

## 9.3 Config

### Создать
- `research/config/compositional/mixed-family-gold-first.yaml`

## 9.4 Code

Если нужен новый lane:
- добавить standalone surface `compositional-mixed-family`
- добавить config key
- добавить planner entrypoint
- добавить artifact group
- подключить db sync
- подключить summary/report directories

Если новый lane можно выразить через existing generic chain protocol без нового surface, это предпочтительнее.

---

# 10. Требования к Codex implementation style

Codex должен действовать в следующем порядке:

1. сначала документы интерпретации;
2. потом cleanup stage semantics;
3. только потом mixed-family gold-first design;
4. только после согласованного дизайна — код и case pack.

Codex не должен:

- сразу делать deeper mixed-family ladder;
- сразу добавлять self-produced cross-family model import;
- менять checker semantics;
- смешивать cleanup и claim expansion в одном PR.

---

# 11. Предпочтительная разбивка на PR / patch sets

## PR 1 — Interpretation layer

Содержимое:
- `compositional-lanes-interpretation.md`
- минимальные doc updates

Цель:
- зафиксировать общий вывод по starter lanes

## PR 2 — Stage semantics cleanup

Содержимое:
- `compositional-stage-semantics-cleanup.md`
- исправления case packs при необходимости
- doc updates

Цель:
- убрать неинтерпретируемые стадии

## PR 3 — Mixed-family gold-first design

Содержимое:
- `mixed-family-gold-first-design.md`
- новый case pack
- local config
- при необходимости новый lane surface

Цель:
- подготовить чистый следующий probe

---

# 12. Итоговый expected outcome

После реализации этого документа проект должен получить:

1. единый, честный и воспроизводимый вывод по starter compositional lanes;
2. очищенную stage semantics;
3. узкий и интерпретируемый `gold-first mixed-family reuse` probe;
4. отсутствие ложного расширения trust boundary;
5. основу для решения, стоит ли вообще когда-либо переходить к self-produced mixed-family reuse.

---

# 13. Короткая формулировка для commit / PR description

Prepare compositional research lanes for a clean gold-first mixed-family reuse probe by finalizing starter-lane interpretation, cleaning stage semantics, and deferring any self-produced cross-family reuse claims until the starter slices become semantically valid and methodologically interpretable.

