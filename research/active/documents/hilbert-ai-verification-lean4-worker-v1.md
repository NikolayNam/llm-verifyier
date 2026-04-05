# Hilbert AI Verification Lean4 Worker v1

- **Title:** **Hilbert AI Verification Lean4 Worker v1**
- **Status:** Draft research pipeline
- **Implementation authority:** Research scaffold authority with limited local confirmation
- **Date:** 2026-03-27
- **Context:** CollabSphere / certificate subsystem / Go orchestration / specialized `.lean` language / Lean4 + mathlib sidecar / formal theorem-backlog generation
- **Cross-track guardrail:** See [Proof-Theory Research Status Matrix](../matrix/proof-theory-research-status-matrix.md)
- **Normative modality:** **MUST / SHOULD / MAY** are used as research-pipeline rules, not as product-runtime guarantees

---

## 1. Purpose

This document defines the first Lean4-backed research pipeline inside the
current Hilbert / ND verification program.

The Lean4 worker is intended to provide:

- a specialized `.lean` language layer for mechanized proof and
  specification work;
- a mechanized proof/specification sidecar;
- generated verification artifacts produced by Go orchestration;
- a formal theorem backlog for theorem packs that can later be proved or
  rejected;
- a bridge from current benchmark-style formal reasoning into a richer proof
  assistant workflow.

---

## 2. Relationship to the current approved result

This document does **not** reopen the already approved finding that the
Hilbert-style certificate kernel functions as a conservative verification
boundary for a formalizable subset of AI-assisted reasoning outputs.

That result remains anchored in:

- [Hilbert AI Verification Benchmark v2 Held-Out](./hilbert-ai-verification-benchmark-v2-held-out.md)
- [Hilbert AI Verification Benchmark ND v1](./hilbert-ai-verification-benchmark-nd-v1.md)

The Lean4 worker is additive. It is a research-side mechanization and
artifact-generation path, not a replacement for the approved boundary.

Interpretation guardrails for this document:

- successful scaffold/bootstrap behavior is not the same as an approved
  empirical benefit claim;
- operator commands in this document are current repo entrypoints, not a
  product-stable interface contract;
- Lean-side artifacts remain outside the current approved trust boundary unless
  another document explicitly upgrades that status.

---

## 3. Working hypothesis

The working hypothesis of this pipeline is:

> Go orchestration plus a Lean4/mathlib worker may improve the quality of
> formal research artifacts, provide a mechanized proof/specification sidecar,
> and generate a formal theorem backlog, while leaving the Hilbert kernel as the
> current approved trust boundary.

This hypothesis is narrower than any claim about open-domain truth
verification. The intended gain is:

- better formal artifact generation;
- richer mechanized specification;
- cleaner candidate-case generation;
- future comparative evidence for or against stronger proof-assistant
  integration.

---

## 4. Explicit non-claims

This pipeline does **not** currently claim that:

- Lean4 replaces the Hilbert kernel;
- Lean4 is now the repository’s approved trust-bearing checker;
- arbitrary noisy inputs become true once routed through proof theory;
- the current worker automatically proves or refutes every generated
  hypothesis;
- the current workstation has already validated broad end-to-end coverage of
  the full mathlib workflow for this research family.

The current value is a research pipeline and artifact seam, not a final
product claim.

---

## 5. MVP v1 pipeline

The MVP v1 workflow is:

1. Go creates or reuses a dedicated Lean4 project scaffold with mathlib.
2. Go accepts a `LeanJob`.
3. Go generates a Lean module from:
   - imports
   - theorem statement
   - helper definitions
   - optional proof script
4. Go invokes:
   - `lake env lean <generated-module>`
5. Go captures:
   - success / failure
   - stdout
   - stderr
   - exit code
6. Go persists the result as a research artifact.

The current canonical scaffold and artifacts live under:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/`

---

## 6. Canonical files

The canonical files for this pipeline are:

- `research/active/documents/hilbert-ai-verification-lean4-worker-v1.md`
- `research/active/documents/proof-research-phase3-runbook.md`
- `docs/technical-specs/platform/lean4-research-worker-v1.md`
- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/`
- `platform-tooling/internal/prooftheory/lean4worker/`

The current worker also has Go entrypoints:

- `go -C platform-tooling run ./cmd/contracts generate-lean4-hypothesis-cases`
- `go -C platform-tooling run ./cmd/contracts export-lean4-hypothesis-cases`
- `go -C platform-tooling run ./cmd/contracts run-lean4-research-job`

and matching `make` targets:

- `make lean4-generate-cases`
- `make lean4-export-cases`
- `make lean4-research-job`
- `make lean4-bootstrap`
- `make lean4-run`

---

## 7. Theorem backlog generation

The Lean4 family is allowed to generate a theorem backlog rather than only
consume one.

The canonical theorem backlog files are:

- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/theorems.csv`
- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/historical/`
- `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/generation-config.json`

Staged-migration note:

- the Lean4 family now treats `theorems/` as the canonical artifact root for
  generated theorem registries and payload templates;
- generated theorem registries now record both `theorem_pack_id` and
  `theorem_id`, and each generated pack is snapshotted into
  `theorems/historical/theorems_<theorem_pack_id>.csv`;
- exported theorem packs and downstream benchmark summaries can now be indexed
  into the shared SQLite registry at
  `research/artifacts/result_research/research_db/research_db.sqlite`
  through `go -C platform-tooling run ./cmd/contracts research-db-sync -project-folder ...`;
- a deprecated `cases/` mirror is still emitted for backward-compatible local
  commands and older manual references during the migration window.

It is intentionally different from the current Hilbert-direct benchmark CSV.
This research family now separates three generation modes:

- `Mode A — Enumerator`
- `Mode B — Curated backlog`
- `Mode C — Model-proposed hypotheses`

`Mode A` remains the canonical default. In that mode, Go enumerates all
formulas up to a bounded implicational depth, all bounded `(assumptions, goal)`
pairs, and then filters the stream through:

- `derivable`
- `not_derivable`
- `interesting`
- `minimal`
- `non_duplicate`

`Mode B` and `Mode C` are now also switchable:

- `curated_backlog`
  - normalizes a curated seed backlog and reclassifies it through the same
    bounded Go-side formal filters.
- `model_proposed`
  - normalizes a source-backed proposal backlog and reclassifies it through the
    same bounded Go-side formal filters.

Its purpose is to define:

- statements that are derivable in the bounded fragment;
- statements that are not derivable in the bounded fragment;
- a formal queue of theorems that may later be translated into benchmark packs or
  paired experiments.

For the phase-1 pilot, the default generated theorems stay within the
implicational propositional fragment so that they remain comparable to the
current ND pilot. The current default enumerator configuration is deliberately
small:

- atoms `{P, Q, R}`
- formula depth `<= 1`
- assumption count `<= 2`
- output limit `32`

Lean-generated theorems can also be exported into benchmark-compatible theorem
packs through:

- `make lean4-export-cases`
- `go -C platform-tooling run ./cmd/contracts export-lean4-hypothesis-cases`

That export path is intentionally one-way: it produces benchmark input
artifacts, but it does not upgrade Lean-side generation into an approved trust
verdict by itself.

---

## 8. Current implementation status

The repository now contains:

- a Lean4 worker API in Go (`LeanJob`, `LeanResult`);
- deterministic module rendering from job payload;
- a project scaffold writer for a mathlib-based Lean project;
- a job runner that persists:
  - generated modules
  - payload copies
  - job metadata
  - result reports
- an enumerator-backed generator for default formal theorem backlogs.

The repository does **not** currently contain approved evidence that the
Lean4 worker improves end-to-end verified benchmark performance. That claim
remains open.

Important current limitation:

- the current `derivable` / `not_derivable` split for the enumerator is
  computed by bounded formal filtering in Go and is not yet a confirmed
  local Lean execution result for every generated theorem in the backlog on
  this workstation.

The local environment is no longer purely theoretical:

- the official `elan` installer was used to install the pinned Lean `v4.22.0`
  toolchain;
- the local scaffold completed:
  - `lake update`
  - `lake exe cache get`
  - `lake build`
  - `lake env lean CollabSphereLean/Basic.lean`
- the Go worker completed a successful end-to-end proof run for `P -> P`
  using:
  - `research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/payloads/identity-proof.json`
  - result artifact:
    `research/artifacts/hilbert-ai-verification-lean4-worker-v1/result/20260326T225357Z/identity_proved/report.json`

That means the scaffold and one positive smoke-check path are now locally
confirmed, while broad theorem coverage and research-level benefit remain open.

The status split in this section is deliberate:

- confirmed locally:
  - scaffold/bootstrap behavior;
  - one positive `P -> P` smoke-check path;
- not yet confirmed:
  - general Lean confirmation for the generated backlog;
  - empirical improvement over the current benchmark families;
  - any promotion of Lean into a trust-bearing verdict layer.

### 8.1 How mathlib is intended to be used

The Lean4 project scaffold is intentionally mathlib-backed. The intended local
bootstrap sequence is:

1. `lake update`
2. `lake exe cache get`
3. `lake build`
4. `lake env lean <module>`

The generated modules import `Mathlib.Tactic` by default and MAY request
broader or narrower imports through the payload contract. This narrower
default proved more stable for phase-1 proof jobs than importing the full
`Mathlib` umbrella module by default. Go renders the module, Lake resolves the
Lean/mathlib environment, and the result is persisted as a research artifact.

### 8.2 Current provisional operator entrypoints

These commands are current repo/operator entrypoints for this research family.
They are not, by themselves, evidence of empirical benefit or a product-stable
interface.

Default enumerator generation:

```bash
make lean4-generate-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1'
```

Curated backlog generation:

```bash
make lean4-generate-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_HYPOTHESIS_MODE='curated_backlog'
```

Model-proposed normalization:

```bash
make lean4-generate-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_HYPOTHESIS_MODE='model_proposed'
```

External source-backed normalization may use:

```bash
make lean4-generate-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_HYPOTHESIS_MODE='model_proposed' \
  LEAN4_THEOREMS_FILE='/abs/path/to/proposals.csv'
```

Export Lean-generated theorems into a benchmark-compatible theorem pack:

```bash
make lean4-export-cases \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  HILBERT_BENCHMARK_PROJECT_FOLDER='hilbert-ai-verification-benchmark-nd-v1'
```

Bootstrap the local Lean4 + mathlib project:

```bash
make lean4-bootstrap \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1'
```

Direct Lean run inside the research project:

```bash
make lean4-run \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_MODULE='CollabSphereLean/Basic.lean'
```

Successful proof run with an explicit payload:

```bash
make lean4-research-job \
  LEAN4_RESEARCH_PROJECT_FOLDER='hilbert-ai-verification-lean4-worker-v1' \
  LEAN4_JOB_NAME='identity_proved' \
  LEAN4_JOB_STATEMENT='P -> P' \
  LEAN4_JOB_PAYLOAD_FILE='research/artifacts/hilbert-ai-verification-lean4-worker-v1/theorems/payloads/identity-proof.json'
```

Expected behavior:

- `lean4-generate-cases` writes `theorems.csv`, `generation-config.json`,
  the default payload template, and seed files for `curated_backlog` and
  `model_proposed`.
- `lean4-export-cases` writes a benchmark-compatible theorem pack derived from
  the Lean theorem backlog.
- `lean4-bootstrap` performs the expected Lake-side dependency and build steps:
  `lake update`, `lake exe cache get`, `lake build`.
- `run-lean4-research-job` writes `Generated.lean`, `job.json`, optional
  `payload.json`, and `report.json`.
- the current canonical positive smoke check is:
  - theorem statement `P -> P`
  - payload `theorems/payloads/identity-proof.json`
  - expected outcome `OK = true`, `exit_code = 0`
- if `lake` is missing, the worker must fail honestly with `OK = false`
  instead of pretending verification succeeded.
- `make` uses the shell-local `lake` binary. If operators run `make` from WSL,
  Lean must also be installed in that WSL environment; a Windows-only install
  does not satisfy the WSL runtime automatically.

---

## 9. Why this matters to the current research program

The current Hilbert-direct evidence already supports a narrow trust-boundary
claim. The ND pilot explores whether a richer front-end improves LLM-facing
proof construction. The Lean4 worker adds a third, different capability:

- mechanized proof/specification artifacts;
- formal theorem-backlog generation;
- a research path toward stronger mechanized checking without moving the
  currently approved boundary.

The intended architecture is therefore:

- Go = orchestration, data handling, artifact management;
- Hilbert kernel = approved trust boundary;
- ND = experimental operational authoring front-end;
- Lean4 = mechanized proof/specification sidecar and hypothesis generator.

---

## 10. Next empirical question

The next empirical question for this pipeline is not whether Lean4 is useful
in the abstract. It is narrower:

> Does a Lean4-backed sidecar materially improve case generation, proof
> specification quality, or future paired verification workflows without
> weakening the current approved Hilbert boundary?

That question remains open and should be answered only after real runs with a
working local or remote Lean4 environment.

