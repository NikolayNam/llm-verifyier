# Research DB

This directory stores the SQLite research registry for theorem packs, benchmark
summaries, and per-theorem result rows.

Canonical file:

- `research_db.sqlite`

Important implementation note:

- SQLite does not support PostgreSQL-style named schemas like
  `research_schema`.
- The repository therefore models the requested namespace by using a dedicated
  database file and `research_*` table names.

Current core tables:

- `research_researches`
- `research_types`
- `research_theorem_files`
- `research_result_files`
- `research_result_summaries`
- `research_result_rows`
- `research_theorems`
- `research_llm_models`
- `research_certificates`
- `research_generation_jobs`
- `research_hypothesis_sets`
- `research_hypothesis_export_snapshots`
- `research_runs`
- `research_run_configs`
- `research_jobs`
- `research_run_artifacts`
- `research_generated_packs`
- `research_exports`

Compatibility view:

- `research_results`

Current Lean4 lineage rule:

- row-level generated theorem/hypothesis state is canonical in
  `research_theorems`;
- `research_hypothesis_sets` and export snapshot tables keep provenance;
- active rebuilt databases no longer create `research_hypotheses`.

Operator-facing maintenance entrypoints:

- `make research-db-init`
- `make research-db-reset`
- `make research-db-sync`
- `make research-db-prune`
- `make research-db-overview`
- `make research-db-runs`

The database is an index over generated research artifacts. It does not replace
the canonical CSV/Markdown files.
