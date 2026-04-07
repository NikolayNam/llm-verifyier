# Research Benchmarks

`researchctl` always works with two model identities:

- canonical `llm_model`: the research-facing model ID stored in summary CSVs,
  reports, and model-catalog joins
- runtime model: the concrete model name that the configured endpoint actually
  serves

For local compatible endpoints, keep canonical IDs in benchmark families and
declare honest name-normalization under `transports.*.model_aliases`.

Example:

```yaml
transports:
  local-compatible:
    provider: compatible
    base_url: http://localhost:11434
    model_aliases:
      gpt-oss:20b-cloud: gpt-oss:20b
```

This means:

- manifests and state metadata record both canonical and runtime model IDs
- benchmark CSVs and reports still use canonical `llm_model`
- preflight checks the runtime model against `/v1/models`

Strict rule:

- use aliases only when the runtime name is the same model under a different
  endpoint label
- do not alias unavailable strong models like `gpt-oss:120b-cloud` or
  `deepseek-v3.1:671b-cloud` to different local models

If a canonical model has no honest runtime alias and is not present on the
endpoint, benchmark `run` should fail at preflight.
