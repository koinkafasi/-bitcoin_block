-- theblock — Topluluk: bookmark/follow/bildirim + gamification (0023)
-- spec 06 K-G (01 §L7, §L8)
CREATE TABLE bookmarks (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id  BIGINT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, content_id)
);
CREATE INDEX ix_bookmarks_user ON bookmarks (user_id, created_at DESC);

CREATE TABLE follows (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type  TEXT NOT NULL CHECK (target_type IN ('kategori','etiket','yazar')),
    target_id    BIGINT,
    target_slug  TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, target_type, target_slug)
);
CREATE INDEX ix_follows_user ON follows (user_id, created_at DESC);

CREATE TABLE notifications (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL DEFAULT 'bilgi',
    title       TEXT NOT NULL,
    body        TEXT,
    url         TEXT,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_notifications_user ON notifications (user_id, is_read, created_at DESC);

CREATE TABLE user_points (
    user_id     BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    points      INT NOT NULL DEFAULT 0,
    streak      INT NOT NULL DEFAULT 0,
    last_active DATE,
    achievements TEXT[] NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);