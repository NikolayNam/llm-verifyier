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
