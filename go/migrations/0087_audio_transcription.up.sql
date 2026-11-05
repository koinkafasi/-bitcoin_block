-- 0087: röportaj ses kaydı (private) + transkripsiyon işleri + transkriptler.
-- Ses kayıtları public /media/ altında servis edilmez; media_files.visibility
-- ile işaretlenir ve ayrı bir private depolama kökünde tutulur.
ALTER TABLE media_files ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public';
ALTER TABLE media_files DROP CONSTRAINT IF EXISTS media_files_visibility_check;
ALTER TABLE media_files ADD CONSTRAINT media_files_visibility_check
    CHECK (visibility IN ('public', 'private'));
CREATE INDEX IF NOT EXISTS idx_media_files_visibility
    ON media_files(site_id, visibility, created_at DESC);

CREATE TABLE IF NOT EXISTS transcription_jobs (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id           BIGINT NOT NULL DEFAULT 1,
    media_id          BIGINT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    content_id        BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    user_id           BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status            TEXT NOT NULL DEFAULT 'queued'
                      CHECK (status IN ('queued','running','completed','failed','cancelled')),
    provider          TEXT NOT NULL DEFAULT 'assemblyai',
    model             TEXT NOT NULL DEFAULT '',
    language          TEXT NOT NULL DEFAULT 'tr',
    provider_job_id   TEXT NOT NULL DEFAULT '',
    progress          INT NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    duration_ms       BIGINT NOT NULL DEFAULT 0,
    speakers_expected INT NOT NULL DEFAULT 0 CHECK (speakers_expected >= 0 AND speakers_expected <= 10),
    attempts          INT NOT NULL DEFAULT 0,
    error             TEXT NOT NULL DEFAULT '',
    consent_at        TIMESTAMPTZ,
    locked_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_transcription_jobs_status
    ON transcription_jobs(status, id);
CREATE INDEX IF NOT EXISTS idx_transcription_jobs_content
    ON transcription_jobs(content_id, id DESC);
-- Aynı ses dosyası için aynı anda tek aktif iş çalışır.
CREATE UNIQUE INDEX IF NOT EXISTS uq_transcription_jobs_active_media
    ON transcription_jobs(media_id) WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS audio_transcripts (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id           BIGINT NOT NULL DEFAULT 1,
    media_id          BIGINT NOT NULL UNIQUE REFERENCES media_files(id) ON DELETE CASCADE,
    content_id        BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    job_id            BIGINT REFERENCES transcription_jobs(id) ON DELETE SET NULL,
    language          TEXT NOT NULL DEFAULT 'tr',
    provider          TEXT NOT NULL DEFAULT '',
    model             TEXT NOT NULL DEFAULT '',
    duration_ms       BIGINT NOT NULL DEFAULT 0,
    text              TEXT NOT NULL DEFAULT '',
    segments          JSONB NOT NULL DEFAULT '[]'::jsonb,
    speakers          JSONB NOT NULL DEFAULT '[]'::jsonb,
    search            TSVECTOR,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audio_transcripts_content
    ON audio_transcripts(content_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audio_transcripts_search
    ON audio_transcripts USING GIN(search);

-- Arama vektörü transkript metninden otomatik güncellenir. 'simple' sözlüğü
-- Türkçe ekler için agresif gövdeleme yapmaz; ham metin araması korunur.
CREATE OR REPLACE FUNCTION audio_transcripts_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search := to_tsvector('simple', COALESCE(NEW.text, ''));
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audio_transcripts_search ON audio_transcripts;
CREATE TRIGGER trg_audio_transcripts_search
    BEFORE INSERT OR UPDATE OF text ON audio_transcripts
    FOR EACH ROW EXECUTE FUNCTION audio_transcripts_search_update();
