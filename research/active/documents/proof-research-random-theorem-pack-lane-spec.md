# Proof Research Random Theorem-Pack Lane Spec

- **Title:** **Proof Research Random Theorem-Pack Lane Spec**
- **Status:** Planned, documented, not implemented
- **Date:** 2026-03-30
- **Scope:** random theorem-pack lane for `Phase 2` and `Phase 3 compare`
- **Implementation rule:** do not implement yet

---

## 1. Scope Boundary

This specification covers a separate random theorem-pack lane for:

- `Phase 2`
- `Phase 3 compare`

It does **not** redefine canonical fixed-pack runs.

It does **not** define a `Phase 1` random lane. `Phase 1` currently operates on
`v2-held-out/cases.csv`; if a random `Phase 1` lane is ever introduced, it must
be designed separately as a `cases/random` lane rather than reusing this
theorem-pack pipeline.

---

## 2. Design Goal

The random theorem-pack lane exists to support exploratory and adversarial
stress evaluation without contaminating canonical fixed-pack reporting.

The core rule is:

- generate a random pack
- freeze the pack
- run repeated `r01/r02/r03` evaluations on that same frozen pack
- store random-lane outputs separately from canonical outputs
- index the random lane in its own research database

---

## 3. Final Layout

Benchmark-family artifact layout:

```text
research/artifacts/hilbert-ai-verification-benchmark-nd-v1/
  theorems/
    random/
      README.md
      staging/
        <pack_id>/
          pack.csv
          manifest.json
          manifest.md
          raw/
      historical/
        <pack_id>/
          pack.csv
          manifest.json
          manifest.md
          raw/
  result/
    random/
      <pack_id>/
        result_<run_id>.csv
  raw/
    random/
      <pack_id>/
        <run_id>/
          ...
```

Random shared-summary and DB layout:

```text
research/artifacts/result_research_random/
  README.md
  direct/
  nd/
  pair/
  research_db/
    README.md
    research_db.sqlite
```

Random report layout:

```text
research/result_research_report_random/
  README.md
  direct/
  nd/
  pair/
```

Required interpretation:

- `staging/` contains generated but not yet frozen packs
- `historical/` contains frozen packs and is the only legal source for random
  benchmark runs
- canonical `result_research/` and random `result_research_random/` must remain
  separate
- canonical `result_research_report/` and random
  `result_research_report_random/` must remain separate

---

## 4. Target Names

The planned operator targets are:

- `make random-theorem-pack-generate`
- `make random-theorem-pack-freeze`
- `make proof-research-phase2-random`
- `make proof-research-phase3-random-compare`
- `make research-db-sync-random-nd-v1`

Optional convenience target:

- `make random-theorem-pack-generate-and-freeze`

The intended operator order is:

1. generate
2. freeze
3. run phase on frozen pack
4. sync random research DB

---

## 5. Naming Convention

### 5.1 `pack_id`

`pack_id` must follow:

```text
random_<YYYYMMDD>T<HHMMSS>Z_<profile>_<size>_seed<seed>
```

Example:

```text
random_20260330T101530Z_balanced_24_seed104729
```

Rules:

- `profile` is a short quota-profile slug, not a full model name
- `size` is the target theorem count
- `seed` is explicit in the identifier

### 5.2 `run_id`

`run_id` must follow:

```text
<pack_id>_phase2_direct_<safe_model>_r01
<pack_id>_phase2_nd_<safe_model>_r01
<pack_id>_phase3_direct_<safe_model>_r01
<pack_id>_phase3_nd_<safe_model>_r01
```

Examples:

```text
random_20260330T101530Z_balanced_24_seed104729_phase2_direct_gpt-oss-20b_r01
random_20260330T101530Z_balanced_24_seed104729_phase2_nd_gpt-oss-20b_r01
```

### 5.3 Frozen Pack Path

```text
research/artifacts/hilbert-ai-verification-benchmark-nd-v1/theorems/random/historical/<pack_id>/pack.csv
```

### 5.4 Per-Run Result Path

```text
research/artifacts/hilbert-ai-verification-benchmark-nd-v1/result/random/<pack_id>/result_<run_id>.csv
```

### 5.5 Per-Run Raw Path

```text
research/artifacts/hilbert-ai-verification-benchmark-nd-v1/raw/random/<pack_id>/<run_id>/
```

### 5.6 Shared Random Summary Paths

```text
research/artifacts/result_research_random/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_random_phase2_direct_<safe_model>_<pack_id>.csv
research/artifacts/result_research_random/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_random_phase2_nd_<safe_model>_<pack_id>.csv
research/artifacts/result_research_random/direct/hilbert-ai-verification-benchmark-nd-v1_result_summary_random_phase3_direct_<safe_model>_<pack_id>.csv
research/artifacts/result_research_random/nd/hilbert-ai-verification-benchmark-nd-v1_result_summary_random_phase3_nd_<safe_model>_<pack_id>.csv
```

### 5.7 Random Report Paths

```text
research/result_research_report_random/direct/hilbert-ai-verification-benchmark-nd-v1_report_random_phase2_direct_<pack_id>.md
research/result_research_report_random/nd/hilbert-ai-verification-benchmark-nd-v1_report_random_phase2_nd_<pack_id>.md
research/result_research_report_random/pair/hilbert-ai-verification-benchmark-nd-v1_research_report_random_phase2_pair_<pack_id>.md
```

---

## 6. DB Behavior

Canonical DB remains:

```text
research/artifacts/result_research/research_db/research_db.sqlite
```

Random lane gets its own DB:

```text
research/artifacts/result_research_random/research_db/research_db.sqlite
```

The random DB must index:

- frozen theorem packs from `theorems/random/historical/<pack_id>/pack.csv`
- `manifest.json` for frozen packs
- random shared summaries from `result_research_random/direct`
- random shared summaries from `result_research_random/nd`
- random shared summaries from `result_research_random/pair` if such artifacts
  exist
- per-run result CSV files from `result/random/<pack_id>/`

The random DB must not index:

- `theorems/random/staging/**`
- Markdown reports as the primary record

The DB-level separation rule is:

- canonical and random lanes must not share the same SQLite file

The summary/result ingest rule is:

- `surface` must be preserved in DB with values such as `direct` and `nd`

The theorem-pack identity rule is:

- `theorem_pack_id` in `pack.csv` must equal the directory `pack_id`

The manifest metadata rule is:

- `manifest.json` must preserve at least:
  - `pack_id`
  - `created_at_utc`
  - `seed`
  - `profile`
  - `target_size`
  - `actual_size`
  - `category_quotas`
  - `generator_kind`
  - `generator_provider`
  - `generator_model`
  - `generator_prompt_version`
  - `freeze_status`
  - `pack_csv`
  - `raw_dir`

Recommended schema direction:

- keep the existing theorem/result tables
- add a dedicated table for pack manifests, for example
  `research_pack_manifests`
- link pack-manifest records and result records through `theorem_pack_id`

---

## 7. Required Changes In `sync.go`

The implementation file affected later will be:

- `platform-tooling/internal/researchdb/sync.go`

Required changes, when implementation begins:

### 7.1 Make shared-summary root configurable

Current behavior is hardcoded to:

```text
research/artifacts/result_research
```

Planned behavior:

- add a sync option such as `SharedResearchRoot`
- canonical sync passes `research/artifacts/result_research`
- random sync passes `research/artifacts/result_research_random`

### 7.2 Exclude random staging from theorem-file ingest

Current theorem discovery walks the full `theorems/` tree recursively, which
would also discover `theorems/random/staging/**`.

Planned behavior:

- skip `theorems/random/staging/**`
- include `theorems/random/historical/**`
- continue supporting existing canonical theorem directories

### 7.3 Import frozen-pack manifests

Current sync imports theorem CSV, summary CSV, and result CSV only.

Planned behavior:

- detect `manifest.json` under `theorems/random/historical/<pack_id>/`
- parse manifest metadata
- write manifest metadata into the research DB

### 7.4 Preserve `surface`

Current summary and result ingest structures do not preserve the CSV `surface`
column.

Planned behavior:

- add `Surface string` to summary-ingest and result-ingest structures
- read the `surface` column from summary CSV
- read the `surface` column from per-run result CSV
- persist `surface` into the research DB

### 7.5 Recognize random shared-summary buckets

Current path classification logic recognizes only:

- `/result_research/direct/`
- `/result_research/nd/`
- `/result_research/pair/`
- `/result_research/waves/`

Planned behavior:

- add recognition for `/result_research_random/direct/`
- add recognition for `/result_research_random/nd/`
- add recognition for `/result_research_random/pair/`

### 7.6 Keep basename-based project matching

The existing rule that shared summaries are discovered when the basename
contains `<project_folder>_` should remain in place.

This implies that random-lane shared summary filenames must keep the benchmark
project folder prefix in their basenames.

### 7.7 Keep recursive discovery for local result files

Current recursive discovery under `result/` is compatible with:

```text
result/random/<pack_id>/result_<run_id>.csv
```

This behavior should remain.

### 7.8 Add a random default DB path

Current default DB path points at canonical research DB only.

Planned behavior:

- add a random default DB path such as
  `research/artifacts/result_research_random/research_db/research_db.sqlite`

---

## 8. Non-Implementation Rule

This document records the agreed direction only.

Until explicitly requested otherwise:

- do not implement these targets
- do not modify `sync.go`
- do not modify research DB schema
- do not add random-lane filesystem writers

