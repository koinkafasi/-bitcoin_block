-- 0050_ai_editor rollback.
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS latency_ms;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS prompt_version;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS job_id;
DROP TABLE IF EXISTS content_revisions;
DROP TABLE IF EXISTS ai_job_sources;
DROP TABLE IF EXISTS ai_jobs;
