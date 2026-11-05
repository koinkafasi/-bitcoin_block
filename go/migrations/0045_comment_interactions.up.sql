ALTER TABLE comments ADD COLUMN IF NOT EXISTS parent_id bigint REFERENCES comments(id) ON DELETE CASCADE;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS likes integer NOT NULL DEFAULT 0;

-- Yorum beğenileri: aynı (yorum, ip) çifti en fazla 1 kez (PK korur).
CREATE TABLE IF NOT EXISTS comment_likes (
  comment_id bigint NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  ip inet NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (comment_id, ip)
);
