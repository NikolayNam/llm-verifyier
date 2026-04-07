# Gold-First Phase2 Run Validity

Status: generated artifact  
Scope: post-hoc reliability classification for existing `gold-first phase2` benchmark runs  
Gold Final collapse threshold: `< 0.60`  
Negative Twin collapse threshold: `< 0.90`

## Run Classification

| Summary run | Classification | Request failure | Gold Final | Negative Twin | Dominant summary failure | Dominant chain failure | Notes |
| --- | --- | ---: | ---: | ---: | --- | --- | --- |
| `20260403T193255+0300` | `execution-invalidated` | `0.609` | `0.202` | `0.390` | `request_failure` | `request_failure` | request_failure_rate >= 0.20 |
| `20260403T204508+0300` | `reasoning-valid` | `0.023` | `0.618` | `0.990` | `schema_failure` | `schema_failure` | low request failure; negative twin stable; dominant failures stay in reasoning-visible buckets |

## Notes

- This batch is artifact-first and does not automatically inspect provider logs or transport traces.
- `ambiguous-needs-review` means the current artifact-only rule-set is insufficient for a clean label.
