# Research Backlog для Codex: Gold-First Mixed-Family Reuse (экономный план)

## Статус документа

- Формат: рабочий markdown-документ для реализации исследования в Codex
- Язык: русский
- Назначение: превратить текущие выводы по `gold-first mixed-family reuse` в конкретный, экономный и методологически чистый план дальнейших экспериментов
- Приоритет: высокий

---

# 1. Контекст

Текущая исследовательская программа уже показала важное разделение:

1. **Starter-compositional lanes** (`assumption_import`, `branching`, `depth_ladder`) остаются слабыми.
2. **Gold-first mixed-family reuse** показывает сильный и интерпретируемый сигнал у нескольких моделей, если промежуточные леммы заданы как trusted/gold imports.
3. Один из ранних `phase2` прогонов оказался **execution-invalidated** из-за request/rate-limit collapse и не должен смешиваться с содержательными reasoning-результатами.
4. Последующий rerun восстановил сильный сигнал и подтвердил, что линия не является артефактом одного удачного запуска.

Ключевой методологический вывод:

> Следующий этап исследования должен усиливать именно **reuse over trusted intermediates**, а не возвращаться сразу к fully autonomous compositional authoring.

---

# 2. Что уже считается выполненным

## 2.1. Reliability gate для full phase2 — закрыт как уже выполненный

Повторно запускать большой `Reliability gate` в том же масштабе **не нужно**.

Почему:

- уже есть phase2-run, который показал execution collapse;
- уже есть последующий rerun, который показал стабильный содержательный сигнал;
- этого достаточно, чтобы отделить:
  - **execution-invalidated runs**
  - **reasoning-valid runs**

### Решение

Считать задачу `P0.1 full reliability gate` **закрытой**.

Вместо новых дорогих прогонов использовать:

- post-hoc reliability classification;
- subgroup analysis на уже собранных данных;
- targeted micro-packs вместо новых full-scale reruns.

---

# 3. Главная цель следующего этапа

Основная цель:

> Понять, **на каких именно типах cross-family reuse** держится сильный сигнал `gold-first mixed-family phase2`, и как аккуратно перейти от `fully gold` к `semi-gold` без потери интерпретируемости.

Это означает, что дальнейший план должен отвечать на вопросы:

1. Где именно проходит граница между working reuse и collapse?
2. Какие transition classes реально трудны?
3. Насколько strong signal зависит от surface/prompt/protocol?
4. Можно ли заменить часть gold-лемм на model-generated + verifier-approved without catastrophic drop?

---

# 4. Экономный backlog экспериментов

Ниже указан **экономный** вариант backlog-а: каждый следующий шаг должен приносить новое знание без ненужного расхода времени и токенов.

---

## P0.2 — Post-hoc reliability classification

### Цель

Формально классифицировать уже существующие прогоны `gold-first mixed-family phase2` на:

- `reasoning-valid`
- `execution-invalidated`

### Зачем это нужно

Без этого в дальнейшем легко смешать:

- реальный reasoning failure;
- request/rate-limit/infrastructure collapse.

### Что сделать

Для каждого phase2 run вычислять и сохранять:

- `request_failure_rate`
- `schema_failure_rate`
- `kernel_failure_rate`
- chain-level `Gold Final`
- chain-level `Negative Twin`
- run classification

### Предлагаемое правило классификации

#### `execution-invalidated`, если:

- `request_failure >= 20%`, **или**
- collapsed одновременно `Gold Final` и `Negative Twin`, **или**
- в raw outputs/logs есть явные признаки rate-limit / transport failure / provider-side rejection

#### `reasoning-valid`, если:

- `request_failure <= 5%`, **и**
- `Negative Twin` не collapsed, **и**
- dominant failures — это в основном `schema_failure`, `kernel_failure`, `false_refusal`

### Артефакты

Создать:

- `research/artifacts/analysis/gold_first_phase2_run_validity.md`
- `research/artifacts/analysis/gold_first_phase2_run_validity.csv`

### Репозиторный operator surface

Использовать:

- `go -C platform-tooling run ./cmd/researchctl analyze gold-first-phase2 --local-config research/config/compositional/mixed-family-gold-first-phase2.yaml`

### Success criteria

- все уже существующие phase2 runs размечены по единому правилу;
- в дальнейших анализах invalidated runs больше не используются как reasoning evidence.

### Kill criteria

- rule-set не позволяет уверенно отделить invalidated runs от валидных;
- слишком много ambiguous runs, требующих ручной интерпретации case-by-case.

---

## P0.3 — Subgroup analysis на уже собранном phase2

### Цель

Получить новое знание **без новых массовых прогонов**.

### Вопрос

На каких именно transition classes сильный сигнал уже держится, а на каких начинает ломаться?

### Что сделать

По уже собранному валидному phase2 rerun разрезать результаты по следующим осям:

- `transition_group`
- `import_arity`
- `reuse_shape`
- `bridge_depth`
- `symbol_overlap`
- `negative_twin_hardness`

### Выходные таблицы

Для каждой модели строить:

1. `Gold Final pass rate by transition_group`
2. `Gold Final pass rate by import_arity`
3. `Gold Final pass rate by bridge_depth`
4. `Negative Twin pass rate by hardness`
5. `dominant failure type by subgroup`

### Артефакты

Создать:

- `research/artifacts/analysis/gold_first_phase2_subgroup_analysis.md`
- `research/artifacts/analysis/gold_first_phase2_subgroup_analysis.csv`
- при необходимости: `research/artifacts/analysis/gold_first_phase2_subgroup_analysis.ipynb` или скрипт

### Репозиторный operator surface

Тот же analysis command должен порождать и subgroup artifacts:

- `go -C platform-tooling run ./cmd/researchctl analyze gold-first-phase2 --local-config research/config/compositional/mixed-family-gold-first-phase2.yaml`

### Success criteria

- найдено минимум 2–3 subgroup axes, которые реально объясняют падение качества;
- можно формулировать не только “модель хороша/плоха”, а “модель слаба на конкретном типе reuse”.

### Kill criteria

- subgroup breakdown не даёт полезного сигнала;
- observed variance выглядит почти случайной и не объясняется аннотациями.

---

## P1. Эксперимент 1 — Hard-transition micro-pack

### Цель

Сделать маленький, но очень информативный пакет только из самых тяжёлых переходов.

### Почему именно так

Новый full-scale phase2 уже есть. Следующий шаг должен быть **точечным** и дешёвым.

### Дизайн

Собрать micro-pack:

- `8–12` entailed chains
- `8–12` hard negative twins
- только самые тяжёлые transition classes из subgroup analysis

### Какие cases включать

Предпочтительно брать chains с сочетанием:

- высокий `bridge_depth`
- низкий `symbol_overlap`
- `import_arity >= 2`
- hardest `negative_twin_hardness`

### Модели

Гонять только:

- `deepseek-v3.1:671b-cloud`
- `gpt-oss:120b-cloud`
- `glm-5:cloud`

Опционально:
- `gpt-oss:20b-cloud` как weaker contrast baseline

### Expected signal

Лучшие модели сохранят рабочий режим, но quality просядет относительно общего phase2.

### Success criteria

- лучшая модель: `Gold Final >= 60%`, `Negative Twin >= 95%`
- вторая модель: `Gold Final >= 45%`
- weakest strong model всё ещё показывает сигнал выше случайного/тривиального уровня

### Kill criteria

- даже лучшая модель падает ниже `40%` на hard slice;
- hard negative twins ломают refusal boundary;
- весь успех phase2 оказывается easy-case phenomenon.

### Артефакты

- `cases/compositional-mixed-family-gold-first-hard-phase2.csv`
- `research/artifacts/result_research/compositional-mixed-family-gold-first-hard/`
- `research/artifacts/analysis/gold_first_phase2_hard_pack_report.md`

---

## P1. Эксперимент 2 — Semi-gold single-generated bridge

### Цель

Проверить промежуточный режим между `fully gold` и `fully model-authored`.

### Идея

Одна промежуточная лемма остаётся gold.
Вторая лемма:

1. генерируется моделью;
2. проходит verifier/kernel filtering;
3. если валидна — используется как trusted import в финальной композиции.

### Почему это важно

Это первый шаг к реалистичной архитектуре:

> model propose -> verifier accept -> trusted reuse

### Дизайн

Для каждого chain:

- `gold lemma A`
- `candidate lemma B` от модели
- verifier check для `B`
- final composition only if `B` accepted

### Expected signal

Semi-gold будет слабее fully-gold, но существенно сильнее starter-compositional lanes.

### Success criteria

- лучшая модель: `Gold Final >= 65%`, `Negative Twin >= 95%`
- результат semi-gold явно выше, чем current `assumption_import` / `branching` baseline

### Kill criteria

- semi-gold почти не отличается от starter baseline;
- verifier-approved bridge не даёт заметного прироста;
- quality разваливается на раннем локальном шаге до начала reuse.

### Артефакты

- `cases/compositional-mixed-family-semi-gold-1.csv`
- `research/artifacts/analysis/semi_gold_single_bridge_report.md`

---

## P1. Эксперимент 3 — Semi-gold multi-import merge

### Цель

Понять, выдерживает ли линия merge нескольких imported resources, а не только single-bridge reuse.

### Дизайн

В каждом entailed chain использовать:

- 1 gold lemma
- 1 model-generated + verifier-approved lemma
- final merge/composition over both

### Expected signal

Качество будет ниже, чем в single-generated bridge, но лучшие модели сохранят usable режим.

### Success criteria

- лучшая модель: `Gold Final >= 50%`, `Negative Twin >= 95%`
- падение относительно single-bridge объяснимо, но не катастрофично

### Kill criteria

- `import_arity=2` обнуляет практическую ценность линии;
- merge pressure полностью убивает signal;
- refusal boundary начинает течь.

### Артефакты

- `cases/compositional-mixed-family-semi-gold-merge.csv`
- `research/artifacts/analysis/semi_gold_merge_report.md`

---

## P1. Эксперимент 4 — Transformational reuse

### Цель

Проверить, умеет ли модель reuse-ить trusted lemma под контролируемым преобразованием, а не только в почти буквальной форме.

### Варианты одного и того же chain

1. `literal reuse`
2. `symbol remapping`
3. `weak normalization`
4. `same structure, altered surface`

### Expected signal

Лучшие модели просядут относительно literal reuse, но останутся выше слабого baseline.

### Success criteria

- лучшая модель:
  - literal `>= 85%`
  - transformed `>= 60%`
- gap между literal и transformed не больше `25 pp`

### Kill criteria

- transformed reuse почти у всех падает в ноль;
- prior success оказывается в основном surface-level literal compatibility.

### Артефакты

- `cases/compositional-mixed-family-transformational.csv`
- `research/artifacts/analysis/transformational_reuse_report.md`

---

## P1. Эксперимент 5 — Hard negative twins / adversarial near-miss controls

### Цель

Проверить, насколько чиста отрицательная граница при более “соблазнительных” неверных цепочках.

### Дизайн

Ввести уровни negative twins:

- `easy negative`
- `structural near-miss`
- `semantically tempting near-miss`

### Expected signal

Лучшие модели сохранят почти чистый refusal boundary, но часть weaker models начнёт течь на hardest negatives.

### Success criteria

- у лучших моделей `Negative Twin >= 95%` даже на hard negatives
- `false_accept` остаётся 0 или около 0

### Kill criteria

- hard negative twins массово порождают false accept;
- текущая чистота negative-control оказывается артефактом слишком лёгких twins.

### Артефакты

- `cases/compositional-mixed-family-hard-negatives.csv`
- `research/artifacts/analysis/hard_negative_twin_report.md`

---

## P2. Эксперимент 6 — Prompt / protocol ablations

### Цель

Понять, насколько сильный сигнал зависит от конкретного framing.

### Варианты

1. текущий prompt
2. более короткий prompt
3. явная инструкция “treat imported lemmas as trusted”
4. явное stage separation
5. строгий schema reminder

### Почему это уже позже

Сначала нужно закрепить сам reasoning signal. Только потом имеет смысл проверять его prompt-устойчивость.

### Expected signal

Лучшие модели сохраняют рабочий режим при нескольких protocol variants, хотя абсолютные цифры плавают.

### Success criteria

- у лучшей модели минимум в 3 из 5 variants:
  - `Gold Final >= 80%`
  - `Negative Twin >= 95%`
- ни один variant не вызывает полный collapse сильной модели

### Kill criteria

- небольшой prompt drift рушит весь сигнал;
- observed strength почти полностью зависит от одной хрупкой формулировки.

### Артефакты

- `research/artifacts/analysis/prompt_protocol_ablation_report.md`

---

## P2. Эксперимент 7 — ND front-end variant для уже стабилизированной gold-first line

### Цель

Понять, помогает ли `ND -> Hilbert` именно на `gold-first mixed-family reuse`, а не только на отдельном pilot pack.

### Почему не раньше

Текущие direct-vs-ND результаты смешанные:

- для части моделей ND иногда помогает;
- для части моделей ND часто нейтрален или регрессивен;
- значит, ND пока не надо делать главным направлением.

### Дизайн

После стабилизации direct gold-first line:

- сделать ND-front-end версию того же самого gold-first micro-pack или semi-gold pack;
- сравнивать direct vs ND pairwise.

### Expected signal

ND может помочь weaker models на schema/kernel boundary, но не обязан улучшать лидеров.

### Success criteria

- хотя бы у одной модели ND даёт `+10 pp` на `Gold Final` без нарушения conservative boundary

### Kill criteria

- ND системно ухудшает gold-first line;
- выигрыш редок, шумен и не воспроизводим;
- latency cost слишком велик для observed gain.

### Артефакты

- `research/artifacts/analysis/gold_first_nd_pair_report.md`

---

## P2. Эксперимент 8 — Back-transfer в starter lanes

### Цель

Проверить, можно ли lessons from gold-first transfer-ить обратно в слабые starter-compositional lanes.

### Важно

Не делать “просто ещё более длинный assumption_import”.
Нужно переносить именно:

- trusted-stage separation
- improved import object format
- protocol lessons
- refusal/control handling

### Целевые lanes

- `assumption_import`
- `branching`
- опционально: `depth_ladder`

### Expected signal

Если lessons transfer, starter lanes должны вырасти хотя бы умеренно.

### Success criteria

- `assumption_import` и/или `branching` дают прирост минимум `+10 pp` к текущему baseline
- failure mix смещается из раннего collapse в более интерпретируемый schema/kernel regime

### Kill criteria

- transfer effect почти отсутствует;
- значит, gold-first reuse и starter authoring — это реально две разные capability zones.

### Артефакты

- `research/artifacts/analysis/gold_first_to_starter_transfer_report.md`

---

# 5. Рекомендуемый порядок запуска

## Сначала

1. `P0.2` — post-hoc reliability classification
2. `P0.3` — subgroup analysis on existing phase2 data
3. `P1.1` — hard-transition micro-pack

## Затем

4. `P1.2` — semi-gold single-generated bridge
5. `P1.3` — semi-gold multi-import merge
6. `P1.4` — transformational reuse
7. `P1.5` — hard negative twins

## Потом

8. `P2.6` — prompt/protocol ablations
9. `P2.7` — ND front-end variant
10. `P2.8` — back-transfer to starter lanes

---

# 6. Что считать главным успехом всей программы

Программа считается сильной, если подтвердится следующее:

1. минимум 2 модели стабильно держат `Gold Final >= 80–85%` на расширенном gold-first phase2;
2. strongest model остаётся рабочей на hard-transition slice;
3. semi-gold даёт заметный и воспроизводимый сигнал выше starter baseline;
4. hard negative twins не ломают conservative refusal boundary;
5. можно объяснить observed quality через структуру reuse, а не только через top-line aggregate.

Если это подтверждается, тогда можно формулировать сильный вывод:

> LLM не обязана надёжно строить всю compositional chain с нуля, чтобы быть полезной. Достаточно, чтобы она умела устойчиво reuse-ить доверенные промежуточные леммы и работать внутри формально верифицируемого контура.

---

# 7. Что считать сигналом на сужение гипотезы

Программа должна быть сужена, если выяснится, что:

- signal держится только на very easy transition classes;
- hard-transition slice почти обнуляет даже лучшие модели;
- semi-gold почти не лучше starter baseline;
- transformed reuse collapses;
- hard negative twins ломают refusal boundary.

В этом случае гипотеза должна быть ослаблена до:

> observed success — это узкий protocol-specific success, а не общий reusable reasoning capability.

---

# 8. Задачи для Codex

## 8.1. Файлы и артефакты

Codex должен поддержать создание и обновление:

- case CSV packs
- scripts for subgroup analysis
- scripts for run validity classification
- summary markdown reports
- helper notebooks or reproducible analysis scripts

## 8.2. Минимальный checklist реализации

### Task A — reliability classification
- [ ] собрать список phase2 runs
- [ ] вычислить run-level validity labels
- [ ] сохранить CSV и markdown summary

### Task B — subgroup analysis
- [ ] загрузить valid phase2 runs
- [ ] присоединить chain metadata
- [ ] посчитать pass/failure by subgroup
- [ ] сохранить markdown report

### Task C — hard-transition micro-pack
- [ ] выбрать hardest transition classes
- [ ] собрать compact CSV pack
- [ ] прогнать 2–3 модели
- [ ] сохранить summary

### Task D — semi-gold single-bridge
- [ ] определить format для model-generated candidate lemma
- [ ] встроить verifier gating
- [ ] собрать cases
- [ ] прогнать benchmark

### Task E — semi-gold merge
- [ ] определить cases with two imported resources
- [ ] проверить merge difficulty
- [ ] сохранить comparative report

### Task F — transformational reuse
- [ ] подготовить literal vs transformed case variants
- [ ] построить side-by-side comparison

### Task G — hard negatives
- [ ] ввести hardness levels
- [ ] проверить refusal boundary

### Task H — protocol ablations
- [ ] подготовить prompt variants
- [ ] прогнать compact slice
- [ ] сравнить stability

### Task I — ND variant
- [ ] клонировать stabilized direct gold-first slice в ND form
- [ ] сравнить paired results

### Task J — transfer to starter lanes
- [ ] выделить reusable protocol lessons
- [ ] применить их к starter packs
- [ ] измерить uplift

---

# 9. Рекомендация по ресурсам

Чтобы не перегружать инфраструктуру и лимиты:

- не запускать повторно full phase2 без новой причины;
- сначала извлекать максимум из уже существующих runs;
- новые дорогостоящие прогоны делать только для:
  - hard-transition micro-pack;
  - semi-gold;
  - compact ablation slices.

Практическое правило:

> если следующий шаг не даёт нового типа знания, а только ещё раз подтверждает уже установленный факт, его нужно отложить.

---

# 10. Краткий итог

Следующий этап исследования должен быть:

- **дешевле**, чем ещё один giant rerun;
- **точнее**, чем просто “дать модели ещё более сложные гипотезы”;
- **сильнее методологически**, потому что он будет отделять reusable reasoning signal от infrastructure noise;
- **ближе к реалистичной архитектуре**, где LLM работает поверх trusted or verifier-approved intermediates.

Главный следующий шаг:

> не повторять full phase2, а извлечь максимум знания из уже собранного rerun и перейти к targeted hard slices + semi-gold bridge.
