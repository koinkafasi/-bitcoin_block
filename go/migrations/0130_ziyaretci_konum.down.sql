-- 0130 geri alma: ziyaretçi konumu tablosunu kaldırır.
--
-- VERİ KAYBI BİLİNÇLİ: bu tablo türetilmiş/analitik veridir, içerik değil.
-- Geri alma yalnız konum geçmişini siler; içerik, üyelik ve okunma
-- sayıları (hit_daily) dokunulmaz.
DROP TABLE IF EXISTS visitor_locations;
