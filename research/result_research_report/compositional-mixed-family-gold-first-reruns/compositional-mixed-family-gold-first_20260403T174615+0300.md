# Hilbert Benchmark Model Comparison Report

- Generated at (UTC): `2026-04-03T14:50:39Z`
- Project folder: `hilbert-ai-verification-benchmark-v2-held-out`
- Cases file filter: `cases/compositional-mixed-family-gold-first-phase1.csv`
- Summary source: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/result_research/compositional-mixed-family-gold-first-reruns/compositional-mixed-family-gold-first_20260403T174615+0300.csv`
- Model catalog: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/result_research/manifests/compositional-mixed-family-gold-first_20260403T174615+0300_model_catalog.csv`
- Case packs: `1`

## Case Pack: `cases/compositional-mixed-family-gold-first-phase1.csv`

- Theorem pack id: `compositional_mixed_family_gold_first_phase1`
- Runs: `1`
- Hypothesis: A minimal Hilbert-style certificate kernel can serve as a useful trust boundary for a formalizable subset of AI-assisted reasoning outputs.

### Aggregate

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Certificate | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Avg Run Elapsed (s) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `1.0.0` | 1 | 30 | 28 | 93.33% | 0 | 0 | 0 | 1 | 0 | 1 | 0 | 8775.23 | 36647 | 263.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Chain Summary

Chain-level aggregation across `chain_result_<run-id>.csv` sidecars. This lane records gold-final composition and negative-control discipline only. Trusted resource stages are intentionally omitted from the runnable gold-first slice, so legacy import/model compatibility columns are hidden here.

| Model | LLM Model | Display | Prompt | Surface | Runs | Chains | Gold Final | Negative Twin | All Stages | Latest Run |
| --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | 1 | 15 | 13/15 (86.67%) | 15/15 (100.00%) | 13/15 (86.67%) | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

#### Starter-stage breakdown

Local prerequisite or seed-like stages. If this section stays clean while import stages collapse, the bottleneck is composition over imported lemmas rather than starter-stage authoring.

No starter-stage rows are present in this pack.

#### Import-stage breakdown

Stages that consume imported lemmas or perform final imported composition. This is the primary place to look for gold-first collapse.

| Model | LLM Model | Display | Prompt | Surface | Stage ID | Stage Role | Chains | Pass | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `fg` | `gold_import_stage` | 15 | 13/15 (86.67%) | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

#### Negative-control breakdown

Negative twins / refusal-discipline stages. When this section stays clean while import stages fail, the failure mode is composition-specific rather than general negative-control collapse.

| Model | LLM Model | Display | Prompt | Surface | Stage ID | Stage Role | Chains | Pass | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `neg` | `negative_control_stage` | 15 | 15/15 (100.00%) | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

#### Stage protocol breakdown

Per-stage pass rates derived from `stage_sequence_json`, `stage_roles_json`, and `stage_passes_json`. For non-starter protocols, this section is the authoritative stage-level view.

| Model | LLM Model | Display | Prompt | Surface | Stage ID | Stage Role | Chains | Pass | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `fg` | `gold_import_stage` | 15 | 13/15 (86.67%) | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `neg` | `negative_control_stage` | 15 | 15/15 (100.00%) | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

#### Chain failure breakdown

| Model | LLM Model | Display | Prompt | Surface | Failure Stage | Failure Type | Chains | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `fg` | `kernel_failure` | 1 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | `fg` | `schema_failure` | 1 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Observed-data warnings

- hard-case audit target `V2E01` is missing from cases.csv
- hard-case audit target `V2E06` is missing from cases.csv
- hard-case audit target `V2E08` is missing from cases.csv
- hard-case audit target `V2E07` is missing from cases.csv
- hard-case audit target `V2E12` is missing from cases.csv
- hard-case audit target `V2N07` is missing from cases.csv
- hard-case audit target `V2N09` is missing from cases.csv

### Entailed-only

Deduplicated result-row slice filtered to `expected_label = entailed`.

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Surface | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | 1 | 15 | 13 | 86.67% | 0 | 0 | 0 | 1 | 0 | 1 | 0 | 15041.93 | 36647.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Not-entailed-only

Deduplicated result-row slice filtered to `expected_label = not_entailed`.

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Surface | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | 1 | 15 | 15 | 100.00% | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 2508.53 | 4608.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Per-category

### Category: `compositional_mixed_family_gold_first`

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Surface | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `direct` | 1 | 30 | 28 | 93.33% | 0 | 0 | 0 | 1 | 0 | 1 | 0 | 8775.23 | 36647.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Per-case hardest failures

Ranked by lowest pass rate, then unsafe acceptance burden, then total failures.

| Case ID | Category | Label | Difficulty | Runs | Observations | Pass | Pass Rate | Failures | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Latest Run | Latest Result | Latest Raw |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |
| `MFG04-FG` | `compositional_mixed_family_gold_first` | `entailed` | `hard` | 1 | 1 | 0 | 0.00% | 1 | 0 | 0 | 0 | 0 | 0 | 1 | 0 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` | `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01.csv` | `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |
| `MFG10-FG` | `compositional_mixed_family_gold_first` | `entailed` | `easy` | 1 | 1 | 0 | 0.00% | 1 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` | `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01.csv` | `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Hard-case audit targets

#### `V2E01` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2E06` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2E08` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2E07` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2E12` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2N07` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

#### `V2N09` — `n/a` / `n/a` / `n/a`

- Goal: `n/a`

No recorded observations for this hard-case target.

### Engineering-best-performance (v1.2)

Valid engineering runs recorded under `hilbert-ai-verification-benchmark-v1.2`.

No matching runs.

### Clean baseline (v1.3)

Valid comparative runs recorded under `hilbert-ai-verification-benchmark-v1.3`.

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Certificate | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Avg Run Elapsed (s) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `1.0.0` | 1 | 30 | 28 | 93.33% | 0 | 0 | 0 | 1 | 0 | 1 | 0 | 8775.23 | 36647 | 263.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### <=20b cohort

Primary `v1.3` comparative slice for models classified as `<=20b`.

No matching runs.

### >20b cohort

Primary `v1.3` comparative slice for models classified as `>20b`.

No matching runs.

### Unknown-size appendix

Valid `v1.3` runs kept outside the size-based primary conclusion because the catalog marks them as `unknown` or `appendix`.

| Model | LLM Model | Display | Size Bucket | Size Label | Prompt | Certificate | Runs | Cases | Pass | Pass Rate | False Refusal | False Accept | Request Failure | Schema Failure | Parse Failure | Kernel Failure | Format Failure | Avg Latency (ms) | Max Latency (ms) | Avg Run Elapsed (s) | Latest Run |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `compatible:gpt-oss:120b-cloud` | `gpt-oss:120b-cloud` | `gpt-oss:120b-cloud [exp: compositional-mixed-family-gold-first-gpt-oss-120b-cloud-r03-rerun-v1-temp0-seed42-topp1-r01-20260403]` | `unknown` | `n/a` | `hilbert-ai-verification-benchmark-v1.3` | `1.0.0` | 1 | 30 | 28 | 93.33% | 0 | 0 | 0 | 1 | 0 | 1 | 0 | 8775.23 | 36647 | 263.00 | `20260403T174615+0300_compositional-mixed-family-gold-first-gpt-oss-120b-cloud-rerun-gpt-oss-120b-cloud-r01` |

### Insufficient evidence / invalid runs

Catalog entries without a valid `v2/v1.3` run and operationally invalid runs remain here as appendix evidence only.

No appendix entries.

