# Lean4 Kernel Parity Report

Status: generated artifact  
Scope: compare the repo-tracked certificate corpus across the current Go Hilbert checker and a replicated Lean4 minimal checker

- project_folder: `hilbert-ai-verification-lean4-kernel-parity-v1`
- run_id: `lean-kernel-parity-smoke`
- corpus_root: `C:/Users/nokclock/Documents/GitHub/collabsphere/platform-tooling/internal/certificates/testdata`
- total_cases: `19`
- lean_eligible_cases: `19`
- behavior_matches: `19`
- detail_matches: `19`
- lean_exit_code: `0`

## Artifact Paths

- corpus_manifest: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/corpus_manifest.csv`
- go_verdicts: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/go_verdicts.csv`
- lean_verdicts: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/lean_verdicts.csv`
- comparison_csv: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/comparison.csv`
- generated_module: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/Generated.lean`
- lean_stdout: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/lean_stdout.txt`
- lean_stderr: `C:/Users/nokclock/Documents/GitHub/collabsphere/research/artifacts/hilbert-ai-verification-lean4-kernel-parity-v1/result/lean-kernel-parity-smoke/kernel_parity/lean_stderr.txt`

## Verdict

No behavior mismatches were detected on the current fixed corpus.

## Notes

- Primary parity criterion is canonicalized `accepted + class + reason_code`, not human-readable wording.
- `detail_match` is tracked separately because exact message text is weaker evidence than verdict behavior.
- This v1 surface targets minimal checker parity on the fixed corpus; malformed-JSON / unknown-field CLI parity remains a separate extension step.
