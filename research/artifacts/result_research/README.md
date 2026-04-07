# Result Research

This directory stores shared research-level result artifacts that are kept
outside any single family-local `result/` directory.

Current intended use:

- clean rerun summaries for paired or staged research slices
- summary files that intentionally combine or compare multiple benchmark
  surfaces
- experiment-specific result ledgers that should remain separate from a
  family's canonical historical summary

Subdirectories:

- `direct/`
  - shared direct-Hilbert summary CSVs
- `nd/`
  - shared `ND -> Hilbert` summary CSVs
- `pair/`
  - reserved pair-level CSV artifacts
- `waves/`
  - cross-run or aggregated CSV artifacts that do not belong only to `direct/`,
    `nd/`, or `pair/`
- `research_db/`
  - SQLite registry for imported theorem/result metadata

These files are derived artifacts and should be treated as research-supporting
outputs rather than normative technical-spec source files.
