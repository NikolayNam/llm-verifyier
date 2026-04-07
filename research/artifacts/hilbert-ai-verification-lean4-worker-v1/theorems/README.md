# Lean4 Worker Theorem Backlog

This directory is the canonical theorem-oriented artifact root for
`hilbert-ai-verification-lean4-worker-v1`.

Canonical files emitted by the worker:

- `theorems.csv`
- `historical/`
- `generation-config.json`
- `curated-default.csv`
- `model-proposed-seed.csv`
- `payloads/`

Staged migration note:

- `../cases/` remains as a deprecated compatibility mirror during the current
  migration window;
- new docs and commands SHOULD point to `theorems/` first.
- generated theorem registries SHOULD preserve immutable snapshots under
  `historical/` with names of the form `theorems_<theorem_pack_id>.csv`.
