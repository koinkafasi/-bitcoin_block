-- theblock P0 ÇEKİRDEK ŞEMA (spec 03 §3) — 0001
-- Kapsam: contents/categories/tags/mahrets, users(RBAC 5 rol), media_files,
--         galleries/videos, pages, settings, redirects, headline_designs,
--         counters_hit/share, comments, contents_extra.
-- Migration sahibi: Go (cmd/migrate).

CREATE EXTENSION IF NOT EXISTS citext;

-- ---- Kullanıcılar (RBAC: 5 rol; rol sütunu, yetki matrisi sonraki migration'da) ----
CREATE TABLE users (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                TEXT NOT NULL,
    email               CITEXT NOT NULL UNIQUE,
    password_hash       TEXT NOT NULL,
    role                TEXT NOT NULL DEFAULT 'Muhabir'
                        CHECK (role IN ('Yönetici','Editör','Muhabir','Yazar','Üye')),
    status              TEXT NOT NULL DEFAULT 'aktif'
                        CHECK (status IN ('aktif','pasif','askida')),
    two_factor_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- İçerik türleri (12) ----
CREATE TABLE content_types (
    key        TEXT PRIMARY KEY,
    label      TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

-- ---- Kategoriler (düz yapı; alt kategori YOK — 02 §17) ----
CREATE TABLE categories (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    custom_url  TEXT,
    open_target TEXT,
    in_header   BOOLEAN NOT NULL DEFAULT TRUE,
    in_menu     BOOLEAN NOT NULL DEFAULT TRUE,
    in_footer   BOOLEAN NOT NULL DEFAULT FALSE,
    status      TEXT NOT NULL DEFAULT 'yayinda' CHECK (status IN ('yayinda','taslak','arsiv')),
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE
);

CREATE TABLE mahrets (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE
);

-- ---- Medya (metadata + konum; dosyanın kendisi depolama katmanında — 03 §3) ----
CREATE TABLE media_files (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    kind        TEXT NOT NULL DEFAULT 'image' CHECK (kind IN ('image','video','document','file')),
    filename    TEXT NOT NULL,
    path        TEXT NOT NULL,
    mime        TEXT,
    size_bytes  BIGINT,
    width       INT,
    height      INT,
    alt         TEXT,
    caption     TEXT,
    is_starred  BOOLEAN NOT NULL DEFAULT FALSE,
    source      TEXT NOT NULL DEFAULT 'upload' CHECK (source IN ('upload','library','internet','render')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Haber/içerik ana kaydı (polimorfik — 02 §1) ----
CREATE TABLE contents (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    type_key          TEXT NOT NULL REFERENCES content_types(key),
    status            TEXT NOT NULL DEFAULT 'taslak'
                      CHECK (status IN ('taslak','planlanmis','yayinda','arsiv','cop')),
    title             TEXT NOT NULL,
    spot              TEXT,
    detail            JSONB NOT NULL DEFAULT '[]',
    slug              TEXT,
    category_id       BIGINT REFERENCES categories(id),
    author_id         BIGINT NOT NULL REFERENCES users(id),
    mahrep_id         BIGINT REFERENCES mahrets(id),
    cover_media_id    BIGINT REFERENCES media_files(id),
    manset_media_id   BIGINT REFERENCES media_files(id),
    social_media_id   BIGINT REFERENCES media_files(id),
    -- Rozet bitleri AM/ÜM/DM/SD/FH/HK (02 §2, §40)
    is_am             BOOLEAN NOT NULL DEFAULT FALSE,
    is_um             BOOLEAN NOT NULL DEFAULT FALSE,
    is_dm             BOOLEAN NOT NULL DEFAULT FALSE,
    is_sd             BOOLEAN NOT NULL DEFAULT FALSE,
    is_fh             BOOLEAN NOT NULL DEFAULT FALSE,
    is_hk             BOOLEAN NOT NULL DEFAULT FALSE,
    source_url        TEXT,
    source_name       TEXT,
    comment_status    TEXT NOT NULL DEFAULT 'acik' CHECK (comment_status IN ('acik','kapali')),
    is_exclusive      BOOLEAN NOT NULL DEFAULT FALSE,
    -- SEO (02 §? SERP/meta)
    seo_title         TEXT,
    seo_description   TEXT,
    canonical_url     TEXT,
    publish_at        TIMESTAMPTZ,
    published_at      TIMESTAMPTZ,
    version           INT NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_contents_slug ON contents (slug) WHERE slug IS NOT NULL AND slug <> '';
CREATE INDEX ix_contents_status   ON contents (status, publish_at DESC);
CREATE INDEX ix_contents_category ON contents (category_id) WHERE category_id IS NOT NULL;
CREATE INDEX ix_contents_author   ON contents (author_id);
CREATE INDEX ix_contents_type     ON contents (type_key);
CREATE INDEX ix_contents_published ON contents (published_at DESC) WHERE published_at IS NOT NULL;

CREATE TABLE content_tags (
    content_id BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (content_id, tag_id)
);

-- ---- Tür-özel alanlar (anki/ilan/BİK/e-gazete vb.) ----
CREATE TABLE contents_extra (
    content_id BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    extra      JSONB NOT NULL DEFAULT '{}'
);

-- ---- Galeriler ----
CREATE TABLE galleries (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title          TEXT NOT NULL,
    content_id     BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    layout         TEXT NOT NULL DEFAULT 'vertical' CHECK (layout IN ('vertical','horizontal')),
    cover_media_id BIGINT REFERENCES media_files(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE gallery_items (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    gallery_id  BIGINT NOT NULL REFERENCES galleries(id) ON DELETE CASCADE,
    media_id    BIGINT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    caption     TEXT,
    description TEXT,
    sort_order  INT NOT NULL DEFAULT 0
);

-- ---- Videolar (3 yol: dosya/YouTube/platform — 02 §6) ----
CREATE TABLE videos (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id     BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    type           TEXT NOT NULL CHECK (type IN ('file','youtube','platform')),
    title          TEXT,
    embed_url      TEXT,
    platform       TEXT,
    cover_media_id BIGINT REFERENCES media_files(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Sayfalar (genel/kurumsal; kurumsal kilitli — 02 §17) ----
CREATE TABLE pages (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind        TEXT NOT NULL DEFAULT 'genel' CHECK (kind IN ('genel','kurumsal')),
    title       TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    custom_url  TEXT,
    spot        TEXT,
    detail      JSONB NOT NULL DEFAULT '[]',
    status      TEXT NOT NULL DEFAULT 'taslak' CHECK (status IN ('taslak','yayinda','arsiv')),
    in_menu     BOOLEAN NOT NULL DEFAULT FALSE,
    in_footer   BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Ayarlar (02 §18: genel/görünüm/modüller/izinler/... — section+key) ----
CREATE TABLE settings (
    section    TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (section, key)
);

-- ---- 301 yönlendirme ----
CREATE TABLE redirects (
    source_path TEXT NOT NULL PRIMARY KEY,
    target_url  TEXT NOT NULL,
    status_code INT NOT NULL DEFAULT 301,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Görsel manşet tasarım belgeleri (02 §8 — katmanlı canvas JSON) ----
CREATE TABLE headline_designs (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT NOT NULL,
    design      JSONB NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    category_id BIGINT REFERENCES categories(id),
    content_id  BIGINT REFERENCES contents(id) ON DELETE SET NULL,
    created_by  BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Okunma + paylaşım sayaçları (03 §3 counters_hit/share) ----
CREATE TABLE counters_hit (
    content_id BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    total      BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE counters_share (
    content_id BIGINT PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    total      BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Yorumlar (manuel onay; Onay Bekliyor/Onaylı; reddet yok, cevap yok — 02 §13) ----
CREATE TABLE comments (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    content_id   BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    user_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    author_name  TEXT NOT NULL,
    author_email TEXT,
    ip           INET,
    body         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'onay_bekliyor' CHECK (status IN ('onay_bekliyor','onayli')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_comments_content ON comments (content_id, status);

-- ---- updated_at otomasyonu (tablo başına trigger) ----
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated     BEFORE UPDATE ON users            FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_categories_updated BEFORE UPDATE ON categories      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_contents_updated  BEFORE UPDATE ON contents         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_pages_updated     BEFORE UPDATE ON pages            FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_designs_updated   BEFORE UPDATE ON headline_designs FOR EACH ROW EXECUTE FUNCTION set_updated_at();
