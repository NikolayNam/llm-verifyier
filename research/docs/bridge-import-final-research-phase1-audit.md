# Bridge Import Final Research Phase1 Audit

Status: derived audit note  
Scope: baseline run `bridge-import-final-research-phase1-20260404`  
Updated: 2026-04-04

## Purpose

This note audits the preserved baseline run for the bridge-stage rows only:

- `BIF01-B1 ... BIF04-B2`
- `4` chains
- `2` bridge stages per chain
- `4` models
- `5` repeats
- `160` bridge-stage observations total

Source-of-truth inputs:

- summary report:
  - `research/result_research_report/bridge-import-final-research/bridge-import-final-research_bridge-import-final-research-phase1-20260404.md`
- result rows:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/result/result_bridge-import-final-research-phase1-20260404_<...>.csv`
- raw outputs:
  - `research/artifacts/hilbert-ai-verification-benchmark-v2-held-out/raw/bridge-import-final-research-phase1-20260404_<...>/<case_id>.txt`

## Executive read

The crude interpretation

> everything breaks only at final imported composition

is not supported by this baseline.

The preserved baseline already shows that the bridge-stage slice itself is not
a clean authoring probe. Every `b1` / `b2` target uses empty top-level
assumptions together with a non-tautological implication goal such as:

- `A -> B`
- `B -> C`
- `P -> Q`
- `!X -> Y`

Under the repository Hilbert semantics, those are not derivable as standalone
theorems from the local case payload. On that reading, the repeated
`NOT_DERIVABLE` responses are consistent with the task contract rather than
being clean evidence of a bridge-authoring capability wall.

Current evidence therefore leans strongly toward:

- `case_wording_defect_suspect`

and away from the stronger claim:

- `clear_capability_wall`

## Cross-model pattern

- `159/160` bridge-stage observations produced `score_bucket = false_refusal`
  with raw output exactly `NOT_DERIVABLE`.
- `1/160` bridge-stage observation produced `score_bucket = request_failure`
  because the request timed out before any substantive model output was
  returned:
  - model: `glm-5:cloud`
  - case: `BIF01-B2`
  - repeat: `r01`
- `schema_status`, `parse_status`, and `kernel_status` are `not_run` for every
  audited bridge-stage row because no bridge-stage certificate was returned.
- The same baseline still records nontrivial `fg` success for stronger models
  while `fm = 0/4` everywhere, so the baseline is evidence against a
  final-only story and in favor of a mixed upstream/downstream failure picture.

## Manual classification legend

Cell codes below expand as follows:

- `FR`
  - `score_bucket = false_refusal`
  - `schema_status = not_run`
  - `parse_status = not_run`
  - `kernel_status = not_run`
  - `manual_classification = case_wording_defect_suspect`
  - raw output observed: exact token `NOT_DERIVABLE`
- `RF`
  - `score_bucket = request_failure`
  - `schema_status = not_run`
  - `parse_status = not_run`
  - `kernel_status = not_run`
  - `manual_classification = inconclusive`
  - raw output observed: timeout string beginning with `LLM_REQUEST_ERROR:`

The bridge-stage explanation attached to every `FR` cell is the same:

- the row is not a fair standalone bridge-authoring target under the local
  case payload because the case provides empty assumptions and a
  non-theorem implication goal

## Per-model audit matrix

### Model: `gpt-oss:20b`

Prompt version: `hilbert-ai-verification-benchmark-v1.3`

| case_id | r01 | r02 | r03 | r04 | r05 |
| --- | --- | --- | --- | --- | --- |
| `BIF01-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF01-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |

### Model: `gpt-oss:120b-cloud`

Prompt version: `hilbert-ai-verification-benchmark-v1.3`

| case_id | r01 | r02 | r03 | r04 | r05 |
| --- | --- | --- | --- | --- | --- |
| `BIF01-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF01-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |

### Model: `glm-5:cloud`

Prompt version: `hilbert-ai-verification-benchmark-v1.3`

| case_id | r01 | r02 | r03 | r04 | r05 |
| --- | --- | --- | --- | --- | --- |
| `BIF01-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF01-B2` | `RF` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |

### Model: `deepseek-v3.1:671b-cloud`

Prompt version: `hilbert-ai-verification-benchmark-v1.3`

| case_id | r01 | r02 | r03 | r04 | r05 |
| --- | --- | --- | --- | --- | --- |
| `BIF01-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF01-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF02-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF03-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B1` | `FR` | `FR` | `FR` | `FR` | `FR` |
| `BIF04-B2` | `FR` | `FR` | `FR` | `FR` | `FR` |

## What this baseline now supports

- bridge-stage failure can already dominate a mixed compositional pipeline
- `fg` and `fm` cannot be interpreted in isolation
- the repository needed split follow-on lanes

## What this baseline does not honestly support

- a clean claim that models already failed a fair standalone bridge-authoring
  task

That is why the next runnable split is:

- `bridge-only-authoring`
- `gold-final-composition-only`
