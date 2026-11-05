-- 0115: toplanan içerikte kaynak görseli + gerçek etiketler
--
-- NEDEN: CoinDesk şablonunda her kart bir kapak görseli taşır. Toplanan
-- içerikte görsel alanı yoktu; kartlar boş gri kutu olarak basılıyordu
-- (tasarım "bozuk" görünüyordu).
--
-- GÖRSEL: kaynağın kendi CDN adresi saklanır (mutlak http/https URL).
-- Neden indirmiyoruz: telif — görseli kendi sunucumuza kopyalamak yeniden
-- yayın sayılabilir; kaynağın CDN'inden `referrerpolicy=no-referrer` ile
-- göstermek toplayıcı sitelerin standart uygulamasıdır. İleride lisanslı
-- kaynaklar için indirme yolu (image-svc) ayrı bir bayrakla açılır.
--
-- ETİKET: RSS `<category>` alanı kaynağın GERÇEK etiketleridir
-- (CoinDesk: "Markets", "Bitcoin"; Decrypt: "Bitcoin", "DeFi"…).
-- Bunlar `tags` + `content_tags` tablolarına yazılır; /tag/{slug} sayfaları
-- ve makale altındaki etiket listesi buradan beslenir.

ALTER TABLE contents ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS ix_contents_image ON contents (id) WHERE image_url <> '';

-- Etiket adı → slug eşlemesi için hızlı arama.
CREATE INDEX IF NOT EXISTS ix_tags_slug ON tags (slug);
