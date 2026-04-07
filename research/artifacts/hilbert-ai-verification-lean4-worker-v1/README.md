# Hilbert AI Verification Lean4 Worker v1 Artifacts

This family stores research artifacts for the Lean4 sidecar pipeline that
supports the current Hilbert / ND verification program.

Current subtrees:

- `lean-project/` — Lean4 + mathlib scaffold used by the Go worker
- `theorems/theorems.csv` — generated formal theorem backlog
- `theorems/generation-config.json` — canonical configuration for the current
  generation run
- `theorems/curated-default.csv` — built-in seed backlog for `Mode B —
  Curated backlog`
- `theorems/model-proposed-seed.csv` — built-in seed backlog for `Mode C —
  Model-proposed hypotheses`
- `theorems/payloads/` — default payload templates for generated Lean jobs
- `cases/` — deprecated compatibility mirror retained during staged migration
- `result/<run-id>/<job>/` — generated modules and result reports

These files are research artifacts, not approved trust-bearing verification
records by themselves.
