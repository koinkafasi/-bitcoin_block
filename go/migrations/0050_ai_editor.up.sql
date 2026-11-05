-- 0050_ai_editor: AI editör işleri, araştırma kaynakları ve içerik sürümleri.
-- AI sonucu doğrudan contents'e yazılmaz; önce ai_jobs.result içinde tutulur.

CREATE TABLE IF NOT EXISTS ai_jobs (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id           BIGINT NOT NULL DEFAULT 1,
    content_id        BIGINT REFERENCES contents(id) ON DELETE CASCADE,
    user_id           BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action            TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'queued'
                      CHECK (status IN ('queued','running','completed','failed','cancelled')),
    selected_text     TEXT NOT NULL DEFAULT '',
    instruction       TEXT NOT NULL DEFAULT '',
    article_version   INT NOT NULL DEFAULT 0,
    context           JSONB NOT NULL DEFAULT '{}',
    result            JSONB,
    error             TEXT NOT NULL DEFAULT '',
    provider          TEXT NOT NULL DEFAULT '',
    model             TEXT NOT NULL DEFAULT '',
    prompt_version    TEXT NOT NULL DEFAULT '',
    prompt_tokens     INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    cost_est          NUMERIC(12,6) NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS ix_ai_jobs_content ON ai_jobs (content_id, id DESC);
CREATE INDEX IF NOT EXISTS ix_ai_jobs_user ON ai_jobs (user_id, id DESC);
CREATE INDEX IF NOT EXISTS ix_ai_jobs_status ON ai_jobs (status, id);

CREATE TABLE IF NOT EXISTS ai_job_sources (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_id       BIGINT NOT NULL REFERENCES ai_jobs(id) ON DELETE CASCADE,
    url          TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    publisher    TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    snippet      TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_ai_job_sources_job ON ai_job_sources (job_id, id);

CREATE TABLE IF NOT EXISTS content_revisions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    version    INT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    spot       TEXT NOT NULL DEFAULT '',
    detail     JSONB NOT NULL DEFAULT '[]',
    snapshot   JSONB NOT NULL DEFAULT '{}',
    user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    source     TEXT NOT NULL DEFAULT 'manuel',
    ai_job_id  BIGINT REFERENCES ai_jobs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (content_id, version)
);
CREATE INDEX IF NOT EXISTS ix_content_revisions_content ON content_revisions (content_id, version DESC);

ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS job_id BIGINT REFERENCES ai_jobs(id) ON DELETE SET NULL;
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS prompt_version TEXT;
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS latency_ms INT NOT NULL DEFAULT 0;
