-- theblock — Trend Merkezi: site içi arama logu + Search Console sorguları (Faz 104)

CREATE TABLE search_queries (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id       BIGINT NOT NULL DEFAULT 1,
    query         TEXT NOT NULL,
    query_norm    TEXT NOT NULL,
    count         BIGINT NOT NULL DEFAULT 1,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, query_norm)
);
CREATE INDEX ix_search_queries_count ON search_queries (site_id, count DESC, last_seen_at DESC);

CREATE TABLE gsc_queries (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL DEFAULT 1,
    property    TEXT NOT NULL,
    day         DATE NOT NULL,
    query       TEXT NOT NULL DEFAULT '',
    page        TEXT NOT NULL DEFAULT '',
    clicks      INT NOT NULL DEFAULT 0,
    impressions INT NOT NULL DEFAULT 0,
    ctr         DOUBLE PRECISION NOT NULL DEFAULT 0,
    position    DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, property, day, query, page)
);
CREATE INDEX ix_gsc_queries_day ON gsc_queries (site_id, day DESC, impressions DESC);
CREATE INDEX ix_gsc_queries_query ON gsc_queries (query);
