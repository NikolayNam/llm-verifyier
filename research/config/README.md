# Config Layout

`research/config/` now uses grouped canonical profiles.

Canonical files:

- `default.yaml`
- `workflow/`
- `direct/`
- `nd/`
- `compositional/`

Directory intent:

- `workflow/`: shared local overrides and workflow-wide profiles such as the
  default local matrix, `phase2`, `phase3`, and `test`
- `direct/`: direct Hilbert / phase1 benchmark packs and focused diagnostic
  lanes
- `nd/`: natural-deduction benchmark packs
- `compositional/`: compositional, bridge, gold-final, and mixed-family lanes

Recommended usage:

- Use grouped paths in commands and documentation.
- Keep `default.yaml` at the root so the repository-default config path remains
  obvious.

Model identity:

- `families.*.models` are canonical research IDs. They are the `llm_model`
  values used in summary CSVs, reports, and model-catalog joins.
- `transports.*.model_aliases` is an optional runtime-only mapping from a
  canonical ID to the concrete model name exposed by the endpoint.
- Use `model_aliases` only for honest name-normalization of the same model, for
  example `gpt-oss:20b-cloud -> gpt-oss:20b`.
- Do not use `model_aliases` to substitute one model for another. If a strong
  canonical model is unavailable on the endpoint, preflight should still fail.
