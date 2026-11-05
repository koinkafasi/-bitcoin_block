-- Faz 195 geri alma: BİLİNÇLİ OLARAK BOŞ.
--
-- Bu migration var olan çeviri satırlarının BOŞ değerlerini doldurdu. Geri
-- alma "değerleri yeniden boşalt" demek olurdu ve panelden yazılmış Türkçe
-- metinleri silerdi. Hangi değerin bu dosyadan, hangisinin panelden geldiği
-- ayırt edilemez.
--
-- Gerekirse elle tek tek düzeltilir (panel → Çeviriler). Tablo/kolon
-- değişikliği yok, şema geri alınacak bir şey bırakmıyor.
SELECT 1;
