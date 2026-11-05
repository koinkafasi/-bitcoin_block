-- theblock — P5-B Konu Kümesi SEO (0013)
CREATE TABLE keyword_clusters (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL DEFAULT 1,
    name        TEXT NOT NULL,
    intent      TEXT NOT NULL DEFAULT 'bilgi',
    pillar_id   BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_clusters_site ON keyword_clusters (site_id);

CREATE TABLE content_clusters (
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    cluster_id BIGINT NOT NULL REFERENCES keyword_clusters(id) ON DELETE CASCADE,
    PRIMARY KEY (content_id, cluster_id)
);
CREATE INDEX ix_cc_cluster ON content_clusters (cluster_id);
