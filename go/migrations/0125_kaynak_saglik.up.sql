-- 0125_kaynak_saglik — RSS kaynaklarında sağlık ve kalite takibi.
--
-- DURUM (12 Kasım 2026): `feeds` tablosu son taramanın sonucunu
-- (last_status, last_error, last_item_count) tutuyordu ama GEÇMİŞİ
-- tutmuyordu. Sonuçları ölçüldü:
--   * 34 kaynağın 22'si kapalıydı; bir kısmı yıllardır çalışıyordu,
--     bir kısmı bozulmuştu — ikisi birbirinden ayırt edilemiyordu.
--   * Bozuk kaynaklar (Coinbase Blog 403, Tether XML değil) kapalı
--     listede sessizce duruyordu; kimse "neden kapalı?" sorusunu
--     yanıtlayamıyordu.
--
-- Bu göç üç şey ekliyor:
--   1) ardışık hata sayacı → kendiliğinden devre dışı bırakma
--   2) son BAŞARILI çekim zamanı → "ne zamandır ölü?" sorusu
--   3) kalite metrikleri → görsel oranı ve dil
ALTER TABLE feeds
  ADD COLUMN IF NOT EXISTS ardisik_hata      integer     NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_ok_at        timestamptz,
  ADD COLUMN IF NOT EXISTS gorsel_oran       smallint,
  ADD COLUMN IF NOT EXISTS dil               text        NOT NULL DEFAULT 'en',
  ADD COLUMN IF NOT EXISTS otomatik_kapandi  boolean     NOT NULL DEFAULT false;

COMMENT ON COLUMN feeds.ardisik_hata IS
  'Üst üste başarısız çekim sayısı; başarılı çekimde sıfırlanır';
COMMENT ON COLUMN feeds.last_ok_at IS
  'Son BAŞARILI çekim zamanı — kaynağın ne zamandır ölü olduğunu gösterir';
COMMENT ON COLUMN feeds.gorsel_oran IS
  'Son çekimde görseli olan öğelerin yüzdesi (0-100); görselsiz kaynak kart düzenini bozuyor';
COMMENT ON COLUMN feeds.dil IS
  'Kaynağın yayın dili (tr/en/...); Türkçe siteye İngilizce içerik çevrilmeden girmemeli';
COMMENT ON COLUMN feeds.otomatik_kapandi IS
  'true ise kaynağı bot kapattı (elle değil); yeniden açılınca sayaç sıfırlanmalı';

-- Mevcut kaynakların dili: Türkçe olanlar ada ve adrese göre işaretlenir.
UPDATE feeds SET dil = 'tr'
 WHERE url ~* '\.com\.tr|\.tr/|gzt\.com|hurriyet|ntv\.com|cnnturk|webtekno|
               shiftdelete|donanimhaber|tamindir|technopat|log\.com|chip\.com'
    OR name ~* 'GZT|Hürriyet|NTV|CNN|Webtekno|ShiftDelete|Donanım|Tamindir|Technopat';

-- Çalıştığı bilinen kaynakların son başarı zamanı, son kontrolden kopyalanır
-- (geçmiş yok; en iyi tahmin bu).
UPDATE feeds SET last_ok_at = last_checked_at
 WHERE last_status = 'ok' AND last_checked_at IS NOT NULL;
