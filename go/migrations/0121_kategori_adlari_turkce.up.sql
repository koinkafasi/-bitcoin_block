-- Kripto bölümlerinin adlarını Türkçeleştirir.
--
-- NEDEN: bölüm adı hem bölüm bloğu başlığında, hem makale kicker'ında
-- (büyük harfe çevrilerek: "POLICY"), hem mega menüde, hem alt bilgide
-- görünüyor. Adlar İngilizce kaldığı için Türkçe sitede "Markets · Policy ·
-- Tech · Finance" yazıyordu; CoinDesk /tr ise "Piyasalar · Politika ·
-- Teknoloji · Finans" kullanıyor (ölçüldü, 7 Kasım 2026).
--
-- SLUG'LAR DEĞİŞMİYOR: adresler kanonik ve indekslenmiş (/markets,
-- /policy, …). Yalnız görünen ad değişiyor.
--
-- 1-15 arası eski (theblock) kategorileri zaten Türkçe ve yayında içerikleri
-- yok; dokunulmuyor.

UPDATE categories SET name = 'Piyasalar'         WHERE slug = 'markets';
UPDATE categories SET name = 'Finans'            WHERE slug = 'business';
UPDATE categories SET name = 'Teknoloji'         WHERE slug = 'tech';
UPDATE categories SET name = 'Politika'          WHERE slug = 'policy';
UPDATE categories SET name = 'Odak'              WHERE slug = 'focus';
UPDATE categories SET name = 'Görüş'             WHERE slug = 'opinion';
UPDATE categories SET name = 'Haber Analizi'     WHERE slug = 'news-analysis';
UPDATE categories SET name = 'Basın Bültenleri'  WHERE slug = 'press-release';
UPDATE categories SET name = 'Sponsorlu'         WHERE slug = 'sponsored-content';
UPDATE categories SET name = 'Videolar'          WHERE slug = 'videos';
UPDATE categories SET name = 'Podcastlar'        WHERE slug = 'podcasts';
UPDATE categories SET name = 'Araştırma'         WHERE slug = 'research';
UPDATE categories SET name = 'Fiyatlar'          WHERE slug = 'price';
UPDATE categories SET name = 'Endeksler'         WHERE slug = 'index';
UPDATE categories SET name = 'Son Haberler'      WHERE slug = 'latest-crypto-news';
UPDATE categories SET name = 'Kripto'            WHERE slug = 'kripto';
