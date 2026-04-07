# ND Benchmark Result Summaries

This result directory currently uses a split summary policy:

- `result_summary_v2.csv` is the canonical append target for new theorem-oriented
  ND benchmark runs.
- `result_summary.csv` is a legacy compatibility summary source that preserves
  older ND rows recorded before the theorem-oriented summary schema was
  introduced.

For this family:

- new runs SHOULD append to `result_summary_v2.csv`;
- summary/report readers MAY load both files by default in order to preserve
  the historical audit trail;
- legacy repository benchmark families outside this ND track are intentionally
  unaffected by this split.
