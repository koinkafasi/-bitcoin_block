-- 0028_events: Etkinlikler modülü (spec 01 §B2, 02 §207) — etkinlik + hatırlatma şeridi
CREATE TABLE events (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id     BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT,
    location    TEXT,
    url         TEXT,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    start_at    TIMESTAMPTZ NOT NULL,
    end_at      TIMESTAMPTZ,
    remind_days INT[] NOT NULL DEFAULT '{1,3,7}',
    status      TEXT NOT NULL DEFAULT 'aktif' CHECK (status IN ('aktif','iptal','taslak')),
    created_by  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_events_site_start ON events (site_id, start_at);

CREATE TRIGGER trg_events_updated BEFORE UPDATE ON events FOR EACH ROW EXECUTE FUNCTION set_updated_at();