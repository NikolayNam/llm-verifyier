# Research Artifacts

This directory stores derived research artifacts that belong to active
benchmark and experiment families under `docs/research/`.

Current families:

- `hilbert-ai-verification-lean4-worker-v1/`
- `hilbert-ai-verification-benchmark-v1/`
- `hilbert-ai-verification-benchmark-v2-held-out/`
- `hilbert-ai-verification-benchmark-nd-v1/`
- `analysis/`
- `result_research/`

Each family may contain:

- `cases.csv` as the default machine-readable case registry
- `cases/` for local hypothesis-specific case packs
- `theorems/` for theorem-oriented registries and theorem-pack exports in newer
  research families
- `theorems/historical/` for immutable theorem-pack snapshots such as
  `theorems_<theorem_pack_id>.csv`
- `result/result_<run-id>.csv` for historical per-run outputs
- `result/result_summary.csv` for append-only run metadata
- `result/result_summary_v2.csv` for theorem-oriented benchmark families that
  explicitly record `theorems_file` / `theorems_path`
- `raw/<run-id>/...` for preserved raw model outputs

Shared research-level artifact space:

- `analysis/` stores post-hoc derived analysis artifacts that sit above a
  single benchmark family or summarize multiple benchmark runs under one
  research slice.
- `result_research/` stores research-slice summaries or other derived
  experiment-level result files that are intentionally kept outside a single
  family-local `result/` directory.
- `result_research/direct/`, `result_research/nd/`, `result_research/pair/`,
  and `result_research/waves/` split those shared CSV artifacts by research
  surface or wave.
- `result_research/research_db/research_db.sqlite` stores the SQLite research
  registry that indexes theorem packs, summary files, and per-theorem result
  rows.

Current naming risk note:

- legacy Hilbert benchmark families still use `cases.csv` / `cases/` as the
  canonical benchmark-input terminology;
- newer theorem-oriented or Lean-backed research families MAY introduce
  theorem-centric registries or packs in parallel, and MAY also introduce
  `result_summary_v2.csv` with `theorems_file` / `theorems_path` as the
  canonical summary fields;
- those theorem-oriented families MAY also introduce `theorem_pack_id` and
  `theorem_id` so that summary/result layers can be traced back to a specific
  historical theorem-pack snapshot;
- when a theorem-oriented family adopts `result_summary_v2.csv`,
  `result_summary.csv` SHOULD be treated as a legacy compatibility summary
  source rather than the append target for new runs;
- historical families SHOULD NOT be renamed destructively unless the runner,
  summaries, raw artifacts, and cross-links are migrated together, because
  those artifacts are already part of the research audit trail.

These files are research artifacts, not normative technical-spec source files.
