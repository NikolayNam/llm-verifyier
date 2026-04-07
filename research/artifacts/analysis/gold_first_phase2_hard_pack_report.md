# Gold-First Phase2 Hard Pack Report

Status: generated artifact  
Scope: hard-transition micro-pack derived from the heaviest `gold-first phase2` transition groups

- cases_file: `cases/compositional-mixed-family-gold-first-hard-phase2.csv`
- summary_dir: `C:/Users/nokclock/Documents/GitHub/llm-verifyier/research/artifacts/result_research/compositional-mixed-family-gold-first-hard`
- matched summary runs: `1`
- entailed chains: `10`
- hard negative twins: `10`
- transition groups: `GF-MF-D`, `GF-MF-E`

## Matched Runs

| Summary run | Classification | Request failure | Schema failure | Kernel failure | Gold Final | Negative Twin | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `20260404T021451+0300` | `reasoning-valid` | `0.025` | `0.158` | `0.025` | `0.500` | `0.983` | low request failure; negative twin stable; dominant failures stay in reasoning-visible buckets |

## Model Summary

| Model | Jobs | Cases | Gold Final | Negative Twin | Request failure | Schema failure | Dominant summary failure | Dominant chain failure |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `gpt-oss:120b-cloud` | `3` | `60` | `0.767` | `1.000` | `0.000` | `0.083` | `schema_failure` | `schema_failure` |
| `deepseek-v3.1:671b-cloud` | `3` | `60` | `0.567` | `1.000` | `0.000` | `0.000` | `false_refusal` | `false_refusal` |
| `glm-5:cloud` | `3` | `60` | `0.367` | `0.933` | `0.100` | `0.250` | `schema_failure` | `schema_failure` |
| `gpt-oss:20b-cloud` | `3` | `60` | `0.300` | `1.000` | `0.000` | `0.300` | `schema_failure` | `schema_failure` |

## Success Criteria

- Best model: `gpt-oss:120b-cloud` -> Gold Final `0.767`, Negative Twin `1.000` => `pass`
- Second model: `deepseek-v3.1:671b-cloud` -> Gold Final `0.567` => `pass`
- Weakest strong model among `gpt-oss:120b-cloud`, `glm-5:cloud`, `deepseek-v3.1:671b-cloud`: `glm-5:cloud` with Gold Final `0.367`.
- The `above random/trivial level` criterion still requires manual interpretation; this report surfaces the exact rate rather than inventing a new threshold.

## Kill Criteria

- Best-model collapse below `0.40`: `not triggered` (`gpt-oss:120b-cloud` at `0.767`).
- Hard negative twins breaking the refusal boundary (`Negative Twin < 0.95` on any model): `triggered`.
- `Easy-case phenomenon` is only partially testable here; this pack is already restricted to the heaviest `GF-MF-D/E` slice, so the remaining question is whether the strongest models still sustain signal under that narrowed distribution.

## Notes

- This report is filtered by `cases_file`, so it does not mix the new hard-phase2 slice with the older full hard pack or the earlier hard-micro pack.
- The pack intentionally combines the hardest `GF-MF-D` and `GF-MF-E` phase2 groups: `GF-MF-D` contributes deeper bridge stress, while `GF-MF-E` contributes lower symbol overlap and `import_arity >= 2` adversarial reuse.
