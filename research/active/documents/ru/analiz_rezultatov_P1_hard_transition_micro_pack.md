# Анализ результатов P1 hard-transition micro-pack

## Статус документа

- Формат: рабочий `.md` документ для Codex / исследовательского документа
- Язык: русский
- Назначение: зафиксировать и интерпретировать результаты `P1 = hard-transition micro-pack`
- Контекст: продолжение линии `gold-first mixed-family reuse` после `P0.1 run validity` и `P0.2 subgroup analysis`

---

# 1. Краткий вывод

`P1 hard-transition micro-pack` дал **сильный положительный результат**.

Главный смысл результатов такой:

1. `gold-first mixed-family reuse` сохраняет сильный сигнал **не только на broad phase2**, но и на специально ужесточённом hard-slice.
2. Все три модели (`deepseek-v3.1:671b-cloud`, `glm-5:cloud`, `gpt-oss:120b-cloud`) сохраняют **идеальную negative-control discipline**.
3. Значит, bottleneck сидит не в общей дисциплине отказа, а именно в **entailed gold-import composition**.
4. При этом граница сложности у моделей различается:
   - `gpt-oss:120b-cloud` — почти полностью устойчив даже на hard micro;
   - `deepseek-v3.1:671b-cloud` — сильный, но ломается на узком kernel frontier;
   - `glm-5:cloud` — рабочий, но заметно более schema-heavy и менее устойчивый.

Итог:

> `P1` подтверждает, что линия `gold-first mixed-family reuse` не является артефактом easy-cases и уже выдерживает targeted hard-case pressure.

---

# 2. Что именно проверял P1

`P1` — это не новый большой phase2 rerun, а **узкий hard-transition micro-pack**.

## Параметры пакета

- case pack: `cases/compositional-mixed-family-gold-first-hard-micro.csv`
- theorem pack id: `compositional_mixed_family_gold_first_hard_micro`
- всего прогонов: `9`
- модели:
  - `deepseek-v3.1:671b-cloud`
  - `glm-5:cloud`
  - `gpt-oss:120b-cloud`
- на прогон:
  - `20` кейсов
  - `10 entailed`
  - `10 not_entailed`

## Семантическая роль пакета

Это уже не broad benchmark, а **targeted pressure test** на hardest slices, найденные после `P0.2 subgroup analysis`.

Его смысл:

- не проверить “есть ли вообще сигнал”;
- а проверить, сохраняется ли `gold-first` сигнал на специально ужесточённой подвыборке.

---

# 3. Главный структурный факт

## Negative-control discipline осталась идеальной у всех трёх моделей

Во всех трёх моделях и во всех трёх прогонах:

- `Negative Twin = 10/10`

Это критически важно, потому что означает:

- нет общего collapse по протоколу;
- нет ложного принятия отрицательных кейсов;
- нет распада refusal boundary.

Следовательно, observed failures в `P1` нужно читать как **composition-specific failures**, а не как общую деградацию системы.

Иными словами:

> `P1` проверяет именно hardest entailed gold-import composition, а не случайный шум всего пайплайна.

---

# 4. Общая картина по моделям

## 4.1. `gpt-oss:120b-cloud`

### Aggregate
- `20/20 = 100%`
- `19/20 = 95%`
- `19/20 = 95%`

### Chain-level
- `Gold Final = 10/10`, `9/10`, `9/10`
- `Negative Twin = 10/10` во всех прогонах

### Failure profile
- только единичные `schema_failure`
- request-level collapse отсутствует
- kernel-level collapse отсутствует

### Интерпретация
`gpt-oss:120b-cloud` на текущем этапе выглядит как:

- **лучший hard-case baseline**
- **самая ровная модель линии**
- **наиболее удобный инженерный reference point**

Это важный результат, потому что он показывает:

> даже на специально ужесточённом hard micro-pack модель сохраняет почти полный coverage.

---

## 4.2. `deepseek-v3.1:671b-cloud`

### Aggregate
- `17/20 = 85%`
- `17/20 = 85%`
- `18/20 = 90%`

### Chain-level
- `Gold Final = 7/10`, `7/10`, `8/10`
- `Negative Twin = 10/10` во всех прогонах

### Failure profile
- доминирующий failure type: `kernel_failure`
- не `schema_failure`
- не `false_refusal`
- не `request_failure`

### Интерпретация
Это очень важное уточнение.

После `P0.2` можно было думать, что узкое место у `deepseek-v3.1` — это скорее:
- adversarial hardness,
- refusal frontier,
- или общее давление harder reuse.

Но `P1` показал более точную картину:

> у `deepseek-v3.1` не общий hard-case collapse, а **узкий kernel frontier** на конкретном типе hardest gold-import composition.

То есть модель остаётся сильной, но ломается не “везде понемногу”, а на очень определённом типе кейсов.

---

## 4.3. `glm-5:cloud`

### Aggregate
- `13/20 = 65%`
- `16/20 = 80%`
- `18/20 = 90%`

### Chain-level
- `Gold Final = 3/10`, `6/10`, `8/10`
- `Negative Twin = 10/10` во всех прогонах

### Failure profile
- доминирующий failure type: `schema_failure`
- один прогон содержит единичный `request_failure`
- kernel-level failures не доминируют

### Интерпретация
`glm-5` не выглядит “плохой моделью”, но выглядит **заметно менее устойчивой**, чем `gpt-oss:120b-cloud`, и менее содержательно чистой, чем `deepseek-v3.1`.

Его слабость здесь выглядит так:

> не узкий reasoning frontier, а более широкая **структурная / схемная хрупкость** на hard entailed composition.

Именно поэтому его профиль лучше описывать как:

- promising,
- но нестабильный,
- schema-heavy under hard transition pressure.

---

# 5. Где именно сидит hard frontier

## 5.1. Для `deepseek-v3.1`

`P1` показал, что реальная hard-граница у этой модели намного уже, чем можно было думать раньше.

### Что проходит идеально
- `GF-MFH-D` → `5/5, 5/5, 5/5`
- `adversarial_gold_reuse` → `5/5, 5/5, 5/5`
- `bridge_depth = 2` → `5/5, 5/5, 5/5`
- `symbol_overlap = minimal` → `5/5, 5/5, 5/5`

### Где сидит провал
- `GF-MFH-C` → `2/5, 2/5, 3/5`
- `two_import_merge` → `2/5, 2/5, 3/5`
- `bridge_depth = 1` → `2/5, 2/5, 3/5`
- `symbol_overlap = low` → `2/5, 2/5, 3/5`

### Смысл
Это очень сильное уточнение. Теперь более точная формулировка такая:

> `deepseek-v3.1` проваливается не на adversarial reuse вообще, а на узком классе `low-overlap + two-import-merge + bridge-depth-1`, причём failure проявляется как `kernel_failure`.

Это уже полноценный исследовательский вывод.

---

## 5.2. Для `glm-5`

У `glm-5` картина не такая узкая и не такая чистая.

### Паттерн
- `GF-MFH-C` → стабильно `3/5`
- `GF-MFH-D` → `0/5`, затем `3/5`, затем `5/5`
- `adversarial_gold_reuse` → `0/5`, затем `3/5`, затем `5/5`
- `two_import_merge` → стабильно `3/5`

### Смысл
Здесь нет ощущения одной “чистой” reasoning-границы.  
Скорее видно:

> модель способна справляться с частью hard cases, но ломается более хаотично и в основном как structured artifact producer.

То есть её слабость в `P1` лучше читать не как “она не понимает hardest cases”, а как:

- нестабильная serializability,
- schema fragility,
- неполная устойчивость на hard composition.

---

## 5.3. Для `gpt-oss:120b-cloud`

У этой модели hard frontier почти не обнаруживается.

### Что видно
- `GF-MFH-D` → `5/5, 5/5, 5/5`
- `adversarial_gold_reuse` → `5/5, 5/5, 5/5`
- `two_import_merge` → `5/5, 4/5, 4/5`
- `bridge_depth = 1` → `5/5, 4/5, 4/5`
- `symbol_overlap = low` → `5/5, 4/5, 4/5`

### Смысл
Здесь нет явного hard-case collapse.  
Есть только единичные schema-level потери.

Это означает:

> на текущем `P1` пакете `gpt-oss:120b-cloud` уже очень близок к режиму “почти решено”.

---

# 6. Самые тяжёлые кейсы пакета

По `Per-case hardest failures` наиболее трудным кейсом оказался:

- `GMFH13-FG` → `3/9 = 33.33%`

Далее идут:
- `GMFH14-FG` → `6/9 = 66.67%`
- `GMFH15-FG` → `6/9 = 66.67%`

Это важно, потому что показывает:

- hardest slice действительно не декоративный;
- у него есть конкретные collapse-points;
- эти collapse-points сосредоточены в entailed final-gold composition, а не в negative-control части.

---

# 7. Что P1 говорит о всей линии исследования

`P1` делает всю программу заметно сильнее.

До этого уже было:

- `P0.1` — разделение `execution-invalidated` и `reasoning-valid` прогонов;
- `P0.2` — карта групп сложности по transition classes;
- `gold-first phase2` как сильная и воспроизводимая линия.

Теперь `P1` добавляет третью вещь:

> strongest models сохраняют signal даже на targeted hard micro-slice.

Это очень важно методологически.  
Потому что иначе всегда оставалось бы сомнение:

> “может быть, весь успех сидит в более лёгких transition groups”.

Теперь это сомнение заметно слабее.

---

# 8. Главный исследовательский вывод из P1

Ниже формулировка, которую уже можно использовать почти как claim.

> `P1 hard-transition micro-pack` подтверждает, что `gold-first mixed-family reuse` остаётся сильной и интерпретируемой линией даже при целевом давлении hardest slices. У лучших моделей сохраняется идеальная negative-control discipline, а деградация сосредотачивается исключительно в `gold_import_stage`. При этом граница сложности оказывается модельно-специфической: `gpt-oss:120b-cloud` остаётся почти полностью устойчивым, `deepseek-v3.1:671b-cloud` ломается на узком kernel frontier (`low-overlap + two-import-merge + bridge-depth-1`), а `glm-5:cloud` демонстрирует более широкий schema-heavy instability pattern. Тем самым `P1` переводит линию из режима “есть ли сигнал” в режим “какова форма hard-case frontier у разных моделей”.

---

# 9. Что P1 меняет в приоритетах следующего этапа

После `P1` уже не выглядит разумным:

- снова делать большой broad rerun;
- или просто “добавлять ещё harder cases вообще”.

Теперь следующий шаг должен быть более точным.

## Правильный следующий ход
### `P2 = semi-gold bridge`

Почему именно он:

1. `gold-first` уже подтверждён достаточно сильно;
2. hardest slices локализованы;
3. теперь нужно проверить следующий мост:

> что произойдёт, если часть trusted intermediates перестанет быть fully gold и станет model-generated, но verifier-approved?

---

# 10. Как уточнить P2 после P1

После результатов `P1` `P2` лучше разбить на два режима.

## P2A — stable-axis semi-gold
Проверять semi-gold на тех типах transition, где сигнал уже стабилен:

- cases, похожие на сильные зоны `gpt-oss:120b-cloud`
- cases, похожие на сильные зоны `deepseek-v3.1`

### Цель
Понять, не рушится ли линия сразу после замены одного gold intermediate на verified model-generated.

## P2B — frontier-axis semi-gold
Проверять semi-gold именно там, где `P1` нашёл hard frontier:

- `low symbol overlap`
- `two_import_merge`
- `bridge-depth-1`
- hardest `GF-MFH-C` family

### Цель
Понять, это всё ещё reuse frontier,
или уже fragile dependency on fully-gold imported layer.

---

# 11. Практические выводы для Codex / следующей реализации

## Уже считается установленным
1. `gold-first mixed-family reuse` выдерживает targeted hard-case pressure.
2. Negative-control часть на `P1` полностью стабильна.
3. Broad failure mode split по моделям стал достаточно ясным:
   - `gpt-oss:120b-cloud` → almost solved hard micro
   - `deepseek-v3.1:671b-cloud` → narrow kernel frontier
   - `glm-5:cloud` → wider schema-heavy instability

## Что реализовывать дальше
1. `P2A stable-axis semi-gold`
2. `P2B frontier-axis semi-gold`
3. при необходимости — отдельный mini-pack только вокруг `GMFH13-FG`, `GMFH14-FG`, `GMFH15-FG`

---

# 12. Одной фразой

> `P1` показал, что strongest models уже умеют не только broad gold-first reuse, но и hard-case gold-first reuse; теперь центральный вопрос смещается с “работает ли линия вообще” на “что произойдёт, когда trusted layer станет не полностью gold, а partially verifier-approved model-generated”.
