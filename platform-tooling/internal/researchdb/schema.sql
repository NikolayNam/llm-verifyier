PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS research_types (
    id INTEGER PRIMARY KEY,
    type_code TEXT NOT NULL UNIQUE,
    domain TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS research_researches (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    artifact_root TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_llm_models (
    id INTEGER PRIMARY KEY,
    llm_model TEXT NOT NULL UNIQUE,
    model_ref TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_certificates (
    id INTEGER PRIMARY KEY,
    certificate_version TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS research_theorem_files (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    type_id INTEGER REFERENCES research_types(id),
    theorem_pack_id TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    absolute_path TEXT NOT NULL,
    is_historical INTEGER NOT NULL DEFAULT 0,
    imported_at_utc TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS research_result_files (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    type_id INTEGER REFERENCES research_types(id),
    artifact_group TEXT NOT NULL,
    file_kind TEXT NOT NULL,
    relative_path TEXT NOT NULL UNIQUE,
    absolute_path TEXT NOT NULL,
    imported_at_utc TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS research_theorems (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    theorem_file_id INTEGER NOT NULL REFERENCES research_theorem_files(id) ON DELETE CASCADE,
    theorem_pack_id TEXT NOT NULL,
    theorem_id TEXT NOT NULL,
    row_order INTEGER NOT NULL DEFAULT 0,
    case_id TEXT NOT NULL DEFAULT '',
    generation_mode TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    label TEXT NOT NULL DEFAULT '',
    derivability_status TEXT NOT NULL DEFAULT '',
    interesting INTEGER NOT NULL DEFAULT 0,
    minimal INTEGER NOT NULL DEFAULT 0,
    difficulty TEXT NOT NULL DEFAULT '',
    assumptions_json TEXT NOT NULL DEFAULT '',
    goal TEXT NOT NULL DEFAULT '',
    logic_fragment TEXT NOT NULL DEFAULT '',
    lean_statement TEXT NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    UNIQUE (theorem_file_id, theorem_id)
);

CREATE TABLE IF NOT EXISTS research_result_summaries (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    result_file_id INTEGER NOT NULL REFERENCES research_result_files(id) ON DELETE CASCADE,
    theorem_file_id INTEGER REFERENCES research_theorem_files(id) ON DELETE SET NULL,
    type_id INTEGER REFERENCES research_types(id),
    llm_model_id INTEGER REFERENCES research_llm_models(id),
    certificate_id INTEGER REFERENCES research_certificates(id),
    run_id TEXT NOT NULL DEFAULT '',
    timestamp_utc TEXT NOT NULL DEFAULT '',
    project_folder TEXT NOT NULL DEFAULT '',
    experiment_name TEXT NOT NULL DEFAULT '',
    experiment_id TEXT NOT NULL DEFAULT '',
    sampling_surface TEXT NOT NULL DEFAULT '',
    requested_sampling_profile TEXT NOT NULL DEFAULT '',
    effective_sampling_profile TEXT NOT NULL DEFAULT '',
    requested_sampling_json TEXT NOT NULL DEFAULT '',
    effective_sampling_json TEXT NOT NULL DEFAULT '',
    unsupported_sampling_json TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    theorem_pack_id TEXT NOT NULL DEFAULT '',
    case_selector TEXT NOT NULL DEFAULT '',
    theorems_file TEXT NOT NULL DEFAULT '',
    theorems_path TEXT NOT NULL DEFAULT '',
    result_path TEXT NOT NULL DEFAULT '',
    raw_dir TEXT NOT NULL DEFAULT '',
    hypothesis TEXT NOT NULL DEFAULT '',
    cases_total INTEGER NOT NULL DEFAULT 0,
    pass_count INTEGER NOT NULL DEFAULT 0,
    false_refusal_count INTEGER NOT NULL DEFAULT 0,
    false_accept_count INTEGER NOT NULL DEFAULT 0,
    request_failure_count INTEGER NOT NULL DEFAULT 0,
    schema_failure_count INTEGER NOT NULL DEFAULT 0,
    parse_failure_count INTEGER NOT NULL DEFAULT 0,
    kernel_failure_count INTEGER NOT NULL DEFAULT 0,
    format_failure_count INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms REAL NOT NULL DEFAULT 0,
    max_latency_ms REAL NOT NULL DEFAULT 0,
    run_elapsed_seconds REAL NOT NULL DEFAULT 0,
    category_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS research_result_rows (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    result_file_id INTEGER NOT NULL REFERENCES research_result_files(id) ON DELETE CASCADE,
    theorem_file_id INTEGER REFERENCES research_theorem_files(id) ON DELETE SET NULL,
    type_id INTEGER REFERENCES research_types(id),
    llm_model_id INTEGER REFERENCES research_llm_models(id),
    run_id TEXT NOT NULL DEFAULT '',
    timestamp_utc TEXT NOT NULL DEFAULT '',
    project_folder TEXT NOT NULL DEFAULT '',
    experiment_name TEXT NOT NULL DEFAULT '',
    experiment_id TEXT NOT NULL DEFAULT '',
    sampling_surface TEXT NOT NULL DEFAULT '',
    requested_sampling_profile TEXT NOT NULL DEFAULT '',
    effective_sampling_profile TEXT NOT NULL DEFAULT '',
    requested_sampling_json TEXT NOT NULL DEFAULT '',
    effective_sampling_json TEXT NOT NULL DEFAULT '',
    unsupported_sampling_json TEXT NOT NULL DEFAULT '',
    prompt_version TEXT NOT NULL DEFAULT '',
    theorem_pack_id TEXT NOT NULL DEFAULT '',
    theorem_id TEXT NOT NULL DEFAULT '',
    case_id TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    expected_label TEXT NOT NULL DEFAULT '',
    score_bucket TEXT NOT NULL DEFAULT '',
    raw_output_kind TEXT NOT NULL DEFAULT '',
    schema_status TEXT NOT NULL DEFAULT '',
    parse_status TEXT NOT NULL DEFAULT '',
    kernel_status TEXT NOT NULL DEFAULT '',
    cli_exit_code TEXT NOT NULL DEFAULT '',
    latency_ms REAL NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_chain_summary_rows (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    result_file_id INTEGER NOT NULL REFERENCES research_result_files(id) ON DELETE CASCADE,
    type_id INTEGER REFERENCES research_types(id),
    run_id TEXT NOT NULL DEFAULT '',
    chain_id TEXT NOT NULL DEFAULT '',
    case_family TEXT NOT NULL DEFAULT '',
    chain_depth INTEGER NOT NULL DEFAULT 0,
    atom_renaming_id TEXT NOT NULL DEFAULT '',
    stage1_pass INTEGER NOT NULL DEFAULT 0,
    stage2_pass INTEGER NOT NULL DEFAULT 0,
    stage3_gold_pass INTEGER NOT NULL DEFAULT 0,
    stage3_model_pass INTEGER NOT NULL DEFAULT 0,
    negative_twin_pass INTEGER NOT NULL DEFAULT 0,
    all_stages_pass INTEGER NOT NULL DEFAULT 0,
    import_stage_pass INTEGER NOT NULL DEFAULT 0,
    final_composition_pass_gold INTEGER NOT NULL DEFAULT 0,
    final_composition_pass_model INTEGER NOT NULL DEFAULT 0,
    conditional_final_pass_gold INTEGER NOT NULL DEFAULT 0,
    conditional_final_pass_model INTEGER NOT NULL DEFAULT 0,
    failure_stage TEXT NOT NULL DEFAULT '',
    failure_type TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_generation_jobs (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    generation_mode TEXT NOT NULL DEFAULT '',
    source_file_path TEXT NOT NULL DEFAULT '',
    atoms_json TEXT NOT NULL DEFAULT '[]',
    filters_json TEXT NOT NULL DEFAULT '[]',
    max_formula_depth INTEGER NOT NULL DEFAULT 0,
    max_assumptions INTEGER NOT NULL DEFAULT 0,
    limit_requested INTEGER NOT NULL DEFAULT 0,
    generated_count INTEGER NOT NULL DEFAULT 0,
    artifact_root_relative_path TEXT NOT NULL DEFAULT '',
    generation_config_relative_path TEXT NOT NULL DEFAULT '',
    default_payload_relative_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_text TEXT NOT NULL DEFAULT '',
    created_at_utc TEXT NOT NULL DEFAULT '',
    completed_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_hypothesis_sets (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    generation_job_id INTEGER NOT NULL REFERENCES research_generation_jobs(id) ON DELETE CASCADE,
    theorem_file_id INTEGER REFERENCES research_theorem_files(id) ON DELETE SET NULL,
    theorem_pack_id TEXT NOT NULL,
    generation_mode TEXT NOT NULL DEFAULT '',
    logic_fragment TEXT NOT NULL DEFAULT '',
    hypothesis_count INTEGER NOT NULL DEFAULT 0,
    created_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_hypothesis_export_snapshots (
    id INTEGER PRIMARY KEY,
    research_id INTEGER NOT NULL REFERENCES research_researches(id) ON DELETE CASCADE,
    hypothesis_set_id INTEGER NOT NULL REFERENCES research_hypothesis_sets(id) ON DELETE CASCADE,
    theorem_pack_id TEXT NOT NULL,
    benchmark_project_folder TEXT NOT NULL DEFAULT '',
    mode_filter TEXT NOT NULL DEFAULT '*',
    interesting_only INTEGER NOT NULL DEFAULT 0,
    minimal_only INTEGER NOT NULL DEFAULT 0,
    limit_applied INTEGER NOT NULL DEFAULT 0,
    exported_count INTEGER NOT NULL DEFAULT 0,
    output_relative_path TEXT NOT NULL DEFAULT '',
    historical_relative_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_text TEXT NOT NULL DEFAULT '',
    created_at_utc TEXT NOT NULL DEFAULT '',
    completed_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_runs (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE,
    command TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    manifest_path TEXT NOT NULL DEFAULT '',
    config_fingerprint TEXT NOT NULL DEFAULT '',
    started_at_utc TEXT NOT NULL DEFAULT '',
    finished_at_utc TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS research_run_configs (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE REFERENCES research_runs(run_id) ON DELETE CASCADE,
    config_path TEXT NOT NULL DEFAULT '',
    local_config_path TEXT NOT NULL DEFAULT '',
    resolved_config_json TEXT NOT NULL DEFAULT '',
    source_fingerprint TEXT NOT NULL DEFAULT '',
    created_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_jobs (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES research_runs(run_id) ON DELETE CASCADE,
    job_key TEXT NOT NULL,
    spec_hash TEXT NOT NULL,
    command TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT '',
    family TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    selector TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    result_path TEXT NOT NULL DEFAULT '',
    summary_path TEXT NOT NULL DEFAULT '',
    raw_dir TEXT NOT NULL DEFAULT '',
    started_at_utc TEXT NOT NULL DEFAULT '',
    finished_at_utc TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    UNIQUE (run_id, job_key),
    UNIQUE (run_id, spec_hash)
);

CREATE TABLE IF NOT EXISTS research_run_artifacts (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES research_runs(run_id) ON DELETE CASCADE,
    job_key TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL,
    relative_path TEXT NOT NULL DEFAULT '',
    absolute_path TEXT NOT NULL,
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at_utc TEXT NOT NULL DEFAULT '',
    UNIQUE (run_id, role, absolute_path)
);

CREATE TABLE IF NOT EXISTS research_generated_packs (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES research_runs(run_id) ON DELETE CASCADE,
    pack_key TEXT NOT NULL UNIQUE,
    project_folder TEXT NOT NULL DEFAULT '',
    source_kind TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    output_path TEXT NOT NULL DEFAULT '',
    manifest_path TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS research_exports (
    id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES research_runs(run_id) ON DELETE CASCADE,
    export_key TEXT NOT NULL UNIQUE,
    project_folder TEXT NOT NULL DEFAULT '',
    benchmark_project_folder TEXT NOT NULL DEFAULT '',
    source_pack_id TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    output_path TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at_utc TEXT NOT NULL DEFAULT ''
);

CREATE VIEW IF NOT EXISTS research_results AS
SELECT
    id,
    research_id,
    result_file_id,
    theorem_file_id,
    type_id,
    llm_model_id,
    certificate_id,
    'summary' AS source_kind,
    run_id,
    timestamp_utc,
    project_folder,
    experiment_name,
    experiment_id,
    sampling_surface,
    requested_sampling_profile,
    effective_sampling_profile,
    requested_sampling_json,
    effective_sampling_json,
    unsupported_sampling_json,
    prompt_version,
    theorem_pack_id,
    '' AS theorem_id,
    '' AS case_id,
    case_selector,
    theorems_file,
    theorems_path,
    result_path,
    raw_dir,
    hypothesis,
    '' AS category,
    '' AS expected_label,
    '' AS score_bucket,
    '' AS raw_output_kind,
    '' AS schema_status,
    '' AS parse_status,
    '' AS kernel_status,
    '' AS cli_exit_code,
    0.0 AS latency_ms,
    '' AS notes,
    cases_total,
    pass_count,
    false_refusal_count,
    false_accept_count,
    request_failure_count,
    schema_failure_count,
    parse_failure_count,
    kernel_failure_count,
    format_failure_count,
    avg_latency_ms,
    max_latency_ms,
    run_elapsed_seconds,
    category_count
FROM research_result_summaries
UNION ALL
SELECT
    -id AS id,
    research_id,
    result_file_id,
    theorem_file_id,
    type_id,
    llm_model_id,
    NULL AS certificate_id,
    'result_row' AS source_kind,
    run_id,
    timestamp_utc,
    project_folder,
    experiment_name,
    experiment_id,
    sampling_surface,
    requested_sampling_profile,
    effective_sampling_profile,
    requested_sampling_json,
    effective_sampling_json,
    unsupported_sampling_json,
    prompt_version,
    theorem_pack_id,
    theorem_id,
    case_id,
    '' AS case_selector,
    '' AS theorems_file,
    '' AS theorems_path,
    '' AS result_path,
    '' AS raw_dir,
    '' AS hypothesis,
    category,
    expected_label,
    score_bucket,
    raw_output_kind,
    schema_status,
    parse_status,
    kernel_status,
    cli_exit_code,
    latency_ms,
    notes,
    0 AS cases_total,
    0 AS pass_count,
    0 AS false_refusal_count,
    0 AS false_accept_count,
    0 AS request_failure_count,
    0 AS schema_failure_count,
    0 AS parse_failure_count,
    0 AS kernel_failure_count,
    0 AS format_failure_count,
    0.0 AS avg_latency_ms,
    0.0 AS max_latency_ms,
    0.0 AS run_elapsed_seconds,
    0 AS category_count
FROM research_result_rows;

CREATE INDEX IF NOT EXISTS idx_research_theorem_files_research_id
    ON research_theorem_files (research_id);
CREATE INDEX IF NOT EXISTS idx_research_theorems_pack_id
    ON research_theorems (theorem_pack_id, theorem_id);
CREATE INDEX IF NOT EXISTS idx_research_theorems_file_order
    ON research_theorems (theorem_file_id, row_order, theorem_id);
CREATE INDEX IF NOT EXISTS idx_research_result_files_research_id
    ON research_result_files (research_id);
CREATE INDEX IF NOT EXISTS idx_research_result_summaries_run_id
    ON research_result_summaries (run_id);
CREATE INDEX IF NOT EXISTS idx_research_result_summaries_experiment_id
    ON research_result_summaries (experiment_id);
CREATE INDEX IF NOT EXISTS idx_research_result_summaries_pack_id
    ON research_result_summaries (theorem_pack_id);
CREATE INDEX IF NOT EXISTS idx_research_result_rows_run_id
    ON research_result_rows (run_id);
CREATE INDEX IF NOT EXISTS idx_research_result_rows_experiment_id
    ON research_result_rows (experiment_id);
CREATE INDEX IF NOT EXISTS idx_research_result_rows_pack_id
    ON research_result_rows (theorem_pack_id, theorem_id);
CREATE INDEX IF NOT EXISTS idx_research_chain_summary_rows_run_id
    ON research_chain_summary_rows (run_id);
CREATE INDEX IF NOT EXISTS idx_research_chain_summary_rows_result_file_id
    ON research_chain_summary_rows (result_file_id);
CREATE INDEX IF NOT EXISTS idx_research_chain_summary_rows_chain_id
    ON research_chain_summary_rows (chain_id);
CREATE INDEX IF NOT EXISTS idx_research_generation_jobs_research_status
    ON research_generation_jobs (research_id, status);
CREATE INDEX IF NOT EXISTS idx_research_hypothesis_sets_pack_id
    ON research_hypothesis_sets (theorem_pack_id, created_at_utc);
CREATE INDEX IF NOT EXISTS idx_research_hypothesis_export_snapshots_set_status
    ON research_hypothesis_export_snapshots (hypothesis_set_id, status);
CREATE INDEX IF NOT EXISTS idx_research_runs_phase_status
    ON research_runs (phase, status);
CREATE INDEX IF NOT EXISTS idx_research_jobs_run_status
    ON research_jobs (run_id, status);
CREATE INDEX IF NOT EXISTS idx_research_jobs_phase_model
    ON research_jobs (phase, model);
CREATE INDEX IF NOT EXISTS idx_research_run_artifacts_run_role
    ON research_run_artifacts (run_id, role);
