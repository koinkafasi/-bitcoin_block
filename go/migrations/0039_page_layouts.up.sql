-- 0039_page_layouts: Modüler sayfa/tasarım oluşturucu (spec 08) — blok tabanlı ana sayfa düzeni
CREATE TABLE page_layouts (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id    BIGINT NOT NULL DEFAULT 1 REFERENCES sites(id) ON DELETE CASCADE,
    page_key   TEXT NOT NULL DEFAULT 'home',
    blocks     JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, page_key)
);

INSERT INTO page_layouts (site_id, page_key, blocks) VALUES (1, 'home',
'[
 {"id":"b-slider","type":"slider","visible":true,"opts":{"count":5,"autoplay":true,"interval":6,"dots":true,"arrows":true}},
 {"id":"b-sondakika","type":"son_dakika","visible":true,"opts":{"count":6}},
 {"id":"b-manset","type":"manset","visible":true,"opts":{"count":5}},
 {"id":"b-cikanlar","type":"one_cikanlar","visible":true,"opts":{"count":8}},
 {"id":"b-kategori","type":"kategori_izgara","visible":true,"opts":{"count":8}},
 {"id":"b-trend","type":"trend","visible":true,"opts":{"count":8}}
]'::jsonb)
ON CONFLICT (site_id, page_key) DO NOTHING;