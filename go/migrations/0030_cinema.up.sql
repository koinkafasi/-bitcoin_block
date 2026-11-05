-- 0030_cinema: Sinemalar — film listesi (ad, süre, tür, fragman) (spec 02 §217)
CREATE TABLE films (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id      BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    cinema       TEXT NOT NULL DEFAULT 'Genel',
    title        TEXT NOT NULL,
    genre        TEXT,
    duration_min INT,
    poster_url   TEXT,
    trailer_url  TEXT,
    synopsis     TEXT,
    week_start   DATE NOT NULL DEFAULT date_trunc('week', CURRENT_DATE)::date,
    sort         INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_films_week ON films (site_id, week_start DESC);
CREATE INDEX ix_films_cinema ON films (site_id, cinema);

CREATE TRIGGER trg_films_updated BEFORE UPDATE ON films FOR EACH ROW EXECUTE FUNCTION set_updated_at();