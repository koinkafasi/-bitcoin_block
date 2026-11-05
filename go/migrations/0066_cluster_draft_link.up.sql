-- 0066: kümeden üretilen taslağı kümeye bağla (tekrar üretimi önler)
ALTER TABLE ai_story_clusters
    ADD COLUMN IF NOT EXISTS content_id BIGINT REFERENCES contents(id) ON DELETE SET NULL;
