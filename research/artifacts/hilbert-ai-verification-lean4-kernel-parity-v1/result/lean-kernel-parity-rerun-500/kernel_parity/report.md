# Lean4 Kernel Parity Report

Status: generated artifact  
Scope: compare the repo-tracked certificate corpus across the current Go Hilbert checker and a replicated Lean4 minimal checker

- project_folder: `hilbert-ai-verification-lean4-kernel-parity-v1`
- run_id: `lean-kernel-parity-rerun-500`
- corpus_root: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw`
- corpus_selection: first `500` sorted JSON files under corpus root
- total_cases: `500`
- lean_eligible_cases: `499`
- behavior_matches: `496`
- detail_matches: `496`
- lean_exit_code: `0`

## Artifact Paths

- corpus_manifest: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/corpus_manifest.csv`
- go_verdicts: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/go_verdicts.csv`
- lean_verdicts: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/lean_verdicts.csv`
- comparison_csv: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/comparison.csv`
- generated_modules_dir: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/batches`
- batch_count: `5`
- lean_stdout: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/lean_stdout.txt`
- lean_stderr: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-rerun-500/kernel_parity/lean_stderr.txt`

## Behavior Mismatches

| Case | Go | Lean | Go reason | Lean reason | Note |
| --- | --- | --- | --- | --- | --- |
| `20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r02/V2E03.json` | `schema_error` | `schema_error` | `json_decode_error` | `missing_step_formula` | behavior_mismatch |
| `20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r03/V2E03.json` | `schema_error` | `schema_error` | `json_decode_error` | `missing_step_formula` | behavior_mismatch |
| `20260402T093635+0300_phase1-matrix-glm-4-7-flash-latest-r04/V2E03.json` | `schema_error` | `schema_error` | `json_decode_error` | `missing_step_formula` | behavior_mismatch |
| `20260402T093635+0300_phase1-matrix-gpt-oss-120b-cloud-r07/V2E02.json` | `schema_error` | `` | `json_decode_error` | `` | lean_ineligible_raw_json |

## Notes

- Primary parity criterion is canonicalized `accepted + class + reason_code`, not human-readable wording.
- `detail_match` is tracked separately because exact message text is weaker evidence than verdict behavior.
- This v1 surface targets minimal checker parity on the fixed corpus; malformed-JSON / unknown-field CLI parity remains a separate extension step.
