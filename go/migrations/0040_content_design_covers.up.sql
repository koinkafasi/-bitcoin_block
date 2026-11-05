-- 0040_content_design_covers: blok başına kapak şablonu eşlemesi (Faz 3).
-- Bir içerik için (design_id) ile otomatik üretilen kapak görselinin medya kaydı.
-- İçerik veya tasarım güncellenince updated_at karşılaştırmasıyla yeniden üretilir.
CREATE TABLE IF NOT EXISTS content_design_covers (
  content_id        BIGINT      NOT NULL REFERENCES contents(id)         ON DELETE CASCADE,
  design_id         BIGINT      NOT NULL REFERENCES headline_designs(id) ON DELETE CASCADE,
  media_id          BIGINT      NOT NULL REFERENCES media_files(id)      ON DELETE CASCADE,
  content_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (content_id, design_id)
);

CREATE INDEX IF NOT EXISTS idx_content_design_covers_design ON content_design_covers (design_id);
