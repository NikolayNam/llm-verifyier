# Gold-First Phase2 Subgroup Analysis

Status: generated artifact  
Scope: subgroup analysis over `reasoning-valid` `gold-first phase2` runs only

## Reasoning-Valid Runs Used

- `20260403T204508+0300`

## Gold Final pass rate by transition group

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `GF-MF-A` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `GF-MF-B` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `GF-MF-C` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `GF-MF-D` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `GF-MF-E` | `50` | `21` | `0.420` | `false_refusal` |
| `deepseek-v3.2:cloud` | `GF-MF-A` | `50` | `28` | `0.560` | `schema_failure` |
| `deepseek-v3.2:cloud` | `GF-MF-B` | `50` | `22` | `0.440` | `schema_failure` |
| `deepseek-v3.2:cloud` | `GF-MF-C` | `50` | `13` | `0.260` | `schema_failure` |
| `deepseek-v3.2:cloud` | `GF-MF-D` | `50` | `23` | `0.460` | `schema_failure` |
| `deepseek-v3.2:cloud` | `GF-MF-E` | `50` | `22` | `0.440` | `request_failure` |
| `glm-5:cloud` | `GF-MF-A` | `50` | `47` | `0.940` | `schema_failure` |
| `glm-5:cloud` | `GF-MF-B` | `50` | `32` | `0.640` | `schema_failure` |
| `glm-5:cloud` | `GF-MF-C` | `50` | `41` | `0.820` | `schema_failure` |
| `glm-5:cloud` | `GF-MF-D` | `50` | `16` | `0.320` | `schema_failure` |
| `glm-5:cloud` | `GF-MF-E` | `50` | `13` | `0.260` | `schema_failure` |
| `gpt-oss:120b-cloud` | `GF-MF-A` | `50` | `45` | `0.900` | `schema_failure` |
| `gpt-oss:120b-cloud` | `GF-MF-B` | `50` | `35` | `0.700` | `schema_failure` |
| `gpt-oss:120b-cloud` | `GF-MF-C` | `50` | `42` | `0.840` | `schema_failure` |
| `gpt-oss:120b-cloud` | `GF-MF-D` | `50` | `43` | `0.860` | `schema_failure` |
| `gpt-oss:120b-cloud` | `GF-MF-E` | `50` | `38` | `0.760` | `schema_failure` |
| `gpt-oss:20b-cloud` | `GF-MF-A` | `50` | `22` | `0.440` | `schema_failure` |
| `gpt-oss:20b-cloud` | `GF-MF-B` | `50` | `19` | `0.380` | `schema_failure` |
| `gpt-oss:20b-cloud` | `GF-MF-C` | `50` | `16` | `0.320` | `schema_failure` |
| `gpt-oss:20b-cloud` | `GF-MF-D` | `50` | `16` | `0.320` | `schema_failure` |
| `gpt-oss:20b-cloud` | `GF-MF-E` | `50` | `18` | `0.360` | `schema_failure` |

## Gold Final pass rate by import arity

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `1` | `150` | `150` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `2` | `100` | `71` | `0.710` | `false_refusal` |
| `deepseek-v3.2:cloud` | `1` | `150` | `73` | `0.487` | `schema_failure` |
| `deepseek-v3.2:cloud` | `2` | `100` | `35` | `0.350` | `schema_failure` |
| `glm-5:cloud` | `1` | `150` | `95` | `0.633` | `schema_failure` |
| `glm-5:cloud` | `2` | `100` | `54` | `0.540` | `schema_failure` |
| `gpt-oss:120b-cloud` | `1` | `150` | `123` | `0.820` | `schema_failure` |
| `gpt-oss:120b-cloud` | `2` | `100` | `80` | `0.800` | `schema_failure` |
| `gpt-oss:20b-cloud` | `1` | `150` | `57` | `0.380` | `schema_failure` |
| `gpt-oss:20b-cloud` | `2` | `100` | `34` | `0.340` | `schema_failure` |

## Gold Final pass rate by reuse shape

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `adversarial_gold_reuse` | `50` | `21` | `0.420` | `false_refusal` |
| `deepseek-v3.1:671b-cloud` | `bridge_depth_ge_1` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `one_import_linear` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `one_import_symbol_remap` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `two_import_merge` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.2:cloud` | `adversarial_gold_reuse` | `50` | `22` | `0.440` | `request_failure` |
| `deepseek-v3.2:cloud` | `bridge_depth_ge_1` | `50` | `23` | `0.460` | `schema_failure` |
| `deepseek-v3.2:cloud` | `one_import_linear` | `50` | `28` | `0.560` | `schema_failure` |
| `deepseek-v3.2:cloud` | `one_import_symbol_remap` | `50` | `22` | `0.440` | `schema_failure` |
| `deepseek-v3.2:cloud` | `two_import_merge` | `50` | `13` | `0.260` | `schema_failure` |
| `glm-5:cloud` | `adversarial_gold_reuse` | `50` | `13` | `0.260` | `schema_failure` |
| `glm-5:cloud` | `bridge_depth_ge_1` | `50` | `16` | `0.320` | `schema_failure` |
| `glm-5:cloud` | `one_import_linear` | `50` | `47` | `0.940` | `schema_failure` |
| `glm-5:cloud` | `one_import_symbol_remap` | `50` | `32` | `0.640` | `schema_failure` |
| `glm-5:cloud` | `two_import_merge` | `50` | `41` | `0.820` | `schema_failure` |
| `gpt-oss:120b-cloud` | `adversarial_gold_reuse` | `50` | `38` | `0.760` | `schema_failure` |
| `gpt-oss:120b-cloud` | `bridge_depth_ge_1` | `50` | `43` | `0.860` | `schema_failure` |
| `gpt-oss:120b-cloud` | `one_import_linear` | `50` | `45` | `0.900` | `schema_failure` |
| `gpt-oss:120b-cloud` | `one_import_symbol_remap` | `50` | `35` | `0.700` | `schema_failure` |
| `gpt-oss:120b-cloud` | `two_import_merge` | `50` | `42` | `0.840` | `schema_failure` |
| `gpt-oss:20b-cloud` | `adversarial_gold_reuse` | `50` | `18` | `0.360` | `schema_failure` |
| `gpt-oss:20b-cloud` | `bridge_depth_ge_1` | `50` | `16` | `0.320` | `schema_failure` |
| `gpt-oss:20b-cloud` | `one_import_linear` | `50` | `22` | `0.440` | `schema_failure` |
| `gpt-oss:20b-cloud` | `one_import_symbol_remap` | `50` | `19` | `0.380` | `schema_failure` |
| `gpt-oss:20b-cloud` | `two_import_merge` | `50` | `16` | `0.320` | `schema_failure` |

## Gold Final pass rate by bridge depth

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `0` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `1` | `100` | `100` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `2` | `100` | `71` | `0.710` | `false_refusal` |
| `deepseek-v3.2:cloud` | `0` | `50` | `28` | `0.560` | `schema_failure` |
| `deepseek-v3.2:cloud` | `1` | `100` | `35` | `0.350` | `schema_failure` |
| `deepseek-v3.2:cloud` | `2` | `100` | `45` | `0.450` | `schema_failure` |
| `glm-5:cloud` | `0` | `50` | `47` | `0.940` | `schema_failure` |
| `glm-5:cloud` | `1` | `100` | `73` | `0.730` | `schema_failure` |
| `glm-5:cloud` | `2` | `100` | `29` | `0.290` | `schema_failure` |
| `gpt-oss:120b-cloud` | `0` | `50` | `45` | `0.900` | `schema_failure` |
| `gpt-oss:120b-cloud` | `1` | `100` | `77` | `0.770` | `schema_failure` |
| `gpt-oss:120b-cloud` | `2` | `100` | `81` | `0.810` | `schema_failure` |
| `gpt-oss:20b-cloud` | `0` | `50` | `22` | `0.440` | `schema_failure` |
| `gpt-oss:20b-cloud` | `1` | `100` | `35` | `0.350` | `schema_failure` |
| `gpt-oss:20b-cloud` | `2` | `100` | `34` | `0.340` | `schema_failure` |

## Gold Final pass rate by symbol overlap

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `high` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `low` | `100` | `71` | `0.710` | `false_refusal` |
| `deepseek-v3.1:671b-cloud` | `medium` | `100` | `100` | `1.000` | `pass_only` |
| `deepseek-v3.2:cloud` | `high` | `50` | `28` | `0.560` | `schema_failure` |
| `deepseek-v3.2:cloud` | `low` | `100` | `44` | `0.440` | `schema_failure` |
| `deepseek-v3.2:cloud` | `medium` | `100` | `36` | `0.360` | `schema_failure` |
| `glm-5:cloud` | `high` | `50` | `47` | `0.940` | `schema_failure` |
| `glm-5:cloud` | `low` | `100` | `45` | `0.450` | `schema_failure` |
| `glm-5:cloud` | `medium` | `100` | `57` | `0.570` | `schema_failure` |
| `gpt-oss:120b-cloud` | `high` | `50` | `45` | `0.900` | `schema_failure` |
| `gpt-oss:120b-cloud` | `low` | `100` | `73` | `0.730` | `schema_failure` |
| `gpt-oss:120b-cloud` | `medium` | `100` | `85` | `0.850` | `schema_failure` |
| `gpt-oss:20b-cloud` | `high` | `50` | `22` | `0.440` | `schema_failure` |
| `gpt-oss:20b-cloud` | `low` | `100` | `37` | `0.370` | `schema_failure` |
| `gpt-oss:20b-cloud` | `medium` | `100` | `32` | `0.320` | `schema_failure` |

## Negative Twin pass rate by hardness

| Model | Value | Total | Pass | Pass rate | Dominant failure |
| --- | --- | ---: | ---: | ---: | --- |
| `deepseek-v3.1:671b-cloud` | `easy` | `50` | `50` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `hard` | `100` | `100` | `1.000` | `pass_only` |
| `deepseek-v3.1:671b-cloud` | `medium` | `100` | `100` | `1.000` | `pass_only` |
| `deepseek-v3.2:cloud` | `easy` | `50` | `46` | `0.920` | `request_failure` |
| `deepseek-v3.2:cloud` | `hard` | `100` | `97` | `0.970` | `request_failure` |
| `deepseek-v3.2:cloud` | `medium` | `100` | `97` | `0.970` | `request_failure` |
| `glm-5:cloud` | `easy` | `50` | `49` | `0.980` | `format_failure` |
| `glm-5:cloud` | `hard` | `100` | `100` | `1.000` | `pass_only` |
| `glm-5:cloud` | `medium` | `100` | `100` | `1.000` | `pass_only` |
| `gpt-oss:120b-cloud` | `easy` | `50` | `50` | `1.000` | `pass_only` |
| `gpt-oss:120b-cloud` | `hard` | `100` | `99` | `0.990` | `format_failure` |
| `gpt-oss:120b-cloud` | `medium` | `100` | `100` | `1.000` | `pass_only` |
| `gpt-oss:20b-cloud` | `easy` | `50` | `50` | `1.000` | `pass_only` |
| `gpt-oss:20b-cloud` | `hard` | `100` | `100` | `1.000` | `pass_only` |
| `gpt-oss:20b-cloud` | `medium` | `100` | `100` | `1.000` | `pass_only` |

## Hardest FG failures

| Model | Axis | Value | Pass rate | Dominant failure |
| --- | --- | --- | ---: | --- |
| `deepseek-v3.2:cloud` | `reuse_shape` | `two_import_merge` | `0.260` | `schema_failure` |
| `deepseek-v3.2:cloud` | `transition_group` | `GF-MF-C` | `0.260` | `schema_failure` |
| `glm-5:cloud` | `reuse_shape` | `adversarial_gold_reuse` | `0.260` | `schema_failure` |
| `glm-5:cloud` | `transition_group` | `GF-MF-E` | `0.260` | `schema_failure` |
| `glm-5:cloud` | `bridge_depth` | `2` | `0.290` | `schema_failure` |
| `glm-5:cloud` | `reuse_shape` | `bridge_depth_ge_1` | `0.320` | `schema_failure` |
| `glm-5:cloud` | `transition_group` | `GF-MF-D` | `0.320` | `schema_failure` |
| `gpt-oss:20b-cloud` | `reuse_shape` | `bridge_depth_ge_1` | `0.320` | `schema_failure` |

## Notes

- `FG failure-type mix` is represented through `dominant_failure_type` inside each subgroup row.
- This batch aggregates only over `reasoning-valid` runs and excludes `execution-invalidated` runs from reasoning evidence.
