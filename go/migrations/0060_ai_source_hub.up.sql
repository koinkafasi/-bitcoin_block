-- 0060: AI haber kaynak merkezi (staging + benzer haber kümeleri)
CREATE TABLE IF NOT EXISTS ai_source_items (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id      BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    feed_id      BIGINT REFERENCES feeds(id) ON DELETE SET NULL,
    source_name  TEXT NOT NULL,
    category_id  BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL,
    url_hash     TEXT NOT NULL UNIQUE,
    summary      TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ai_source_items_cat ON ai_source_items(site_id, category_id, fetched_at DESC);

CREATE TABLE IF NOT EXISTS ai_story_clusters (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id      BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    category_id  BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    title        TEXT NOT NULL,
    summary      TEXT NOT NULL DEFAULT '',
    keywords     TEXT NOT NULL DEFAULT '',
    item_count   INT NOT NULL DEFAULT 0,
    source_count INT NOT NULL DEFAULT 0,
    trend_term   TEXT NOT NULL DEFAULT '',
    trend_score  INT NOT NULL DEFAULT 0,
    score        INT NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'yeni',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ai_story_clusters_cat ON ai_story_clusters(site_id, category_id, score DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_story_cluster_items (
    cluster_id BIGINT NOT NULL REFERENCES ai_story_clusters(id) ON DELETE CASCADE,
    item_id    BIGINT NOT NULL REFERENCES ai_source_items(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, item_id)
);
