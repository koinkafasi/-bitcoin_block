-- 0067: kategori bazlı AI içerik üretim planı
CREATE TABLE IF NOT EXISTS ai_category_plans (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id       BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    category_id   BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    daily_target  INT NOT NULL DEFAULT 2,
    min_sources   INT NOT NULL DEFAULT 1,
    trend_weight  INT NOT NULL DEFAULT 50,
    max_age_hours INT NOT NULL DEFAULT 48,
    aktif         BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, category_id)
);

INSERT INTO ai_category_plans (site_id, category_id, daily_target, min_sources, trend_weight, max_age_hours)
SELECT 1, c.id, v.target, v.min_sources, v.trend_weight, v.max_age
FROM (VALUES
    ('gundem',       8, 2, 70, 24),
    ('ekonomi',      4, 2, 60, 36),
    ('spor',         4, 2, 40, 24),
    ('dunya',        3, 2, 50, 36),
    ('teknoloji',    2, 1, 50, 48),
    ('yasam',        2, 1, 30, 72),
    ('egitim-kultur',2, 1, 30, 72),
    ('sinema',       1, 1, 20, 72),
    ('tarih',        1, 1, 20, 168),
    ('turizm',       1, 1, 20, 168),
    ('oyun',         1, 1, 20, 72),
    ('mimari',       1, 1, 20, 168)
) AS v(slug, target, min_sources, trend_weight, max_age)
JOIN categories c ON c.slug = v.slug
ON CONFLICT (site_id, category_id) DO NOTHING;
