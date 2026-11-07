-- 0122: YouTube video arşivi (video · short · canlı yayın).
--
-- NEDEN TABLO: ana sayfa "Bugünün Videoları", "Bugünün Klipleri" ve /videos
-- sayfaları gerçek YouTube içeriğiyle dolmalı. Veriyi her istekte YouTube'dan
-- çekmek olmaz (gecikme + dış servise bağımlılık + rate limit); scheduler
-- periyodik olarak çeker, sayfalar tablodan okur. Böylece YouTube'a yeni
-- video düştüğünde sayfa kendiliğinden güncellenir.
--
-- KAYNAK: kanal RSS'i (video/short ayrımı bağlantı yolundan: /shorts/<id>)
-- + /streams sayfası (geçmiş canlı yayınlar) + /live sayfası (şu an canlı mı).
--
-- YÖNETİM ALANLARI: `one_cikan`, `canli`, `gizli`, `sira` panelden değişir ve
-- çekme işlemi bunları EZMEZ (upsert yalnız YouTube'dan gelen alanları yazar).
-- Bu yüzden editörün "öne çıkar"/"gizle" kararı her senkronizasyonda korunur.

CREATE TABLE IF NOT EXISTS youtube_icerik (
    video_id    text PRIMARY KEY,
    tur         text NOT NULL CHECK (tur IN ('video', 'short', 'stream')),
    baslik      text NOT NULL,
    aciklama    text NOT NULL DEFAULT '',
    kanal       text NOT NULL DEFAULT '',
    gorsel      text NOT NULL DEFAULT '',
    url         text NOT NULL DEFAULT '',
    sure_sn     integer NOT NULL DEFAULT 0,
    izlenme     bigint NOT NULL DEFAULT 0,
    -- yayin_at: RSS'ten gelen yayın zamanı. /streams sayfası tarih vermez,
    -- bu yüzden NULL olabilir; sıralama `sira` ile korunur (YouTube yeni→eski).
    yayin_at    timestamptz,
    canli       boolean NOT NULL DEFAULT false,
    one_cikan   boolean NOT NULL DEFAULT false,
    gizli       boolean NOT NULL DEFAULT false,
    sira        integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS youtube_icerik_tur_idx
    ON youtube_icerik (tur, gizli, yayin_at DESC NULLS LAST, sira);

INSERT INTO settings (section, key, value) VALUES
    ('youtube', 'kanal_id',        '"UCPtM806ijpKatyIHH5w4kmg"'::jsonb),
    ('youtube', 'kanal_handle',    '"bitcosar"'::jsonb),
    ('youtube', 'dakika',          '15'::jsonb),
    ('youtube', 'aktif',           'true'::jsonb),
    ('youtube', 'video_goster',    'true'::jsonb),
    ('youtube', 'short_goster',    'true'::jsonb),
    ('youtube', 'canli_goster',    'true'::jsonb),
    ('youtube', 'canli_video_id',  '""'::jsonb),
    ('youtube', 'son_cekme',       '""'::jsonb),
    ('youtube', 'son_hata',        '""'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
