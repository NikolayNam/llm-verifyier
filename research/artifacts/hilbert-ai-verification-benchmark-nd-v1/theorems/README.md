# ND v1 Theorem Packs

This directory is the canonical theorem-pack home for the
`hilbert-ai-verification-benchmark-nd-v1` research family.

Current canonical pack:

- `pilot_shared_20260327.csv`
- `historical/theorems_pilot_shared_20260327.csv`

The theorem-pack schema is theorem-oriented:

- `theorem_pack_id`
- `theorem_id`
- `case_id` as a compatibility alias for older benchmark internals

Staged migration note:

- `../cases.csv` remains as a deprecated compatibility alias for older runs and
  summary rows;
- new paired `Hilbert-direct` and `ND -> Hilbert` runs SHOULD target theorem
  packs under this directory.
- immutable historical pack snapshots SHOULD be preserved under `historical/`
  using the naming rule `theorems_<theorem_pack_id>.csv`.
- generic placeholder pack ids such as `theorems` or `cases` SHOULD NOT be
  treated as valid historical theorem-pack identifiers; the SQLite importer
  skips those snapshots as stale artifacts.
