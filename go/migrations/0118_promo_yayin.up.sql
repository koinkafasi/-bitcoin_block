-- 0118: promo bandının ilk yayını (gerçek hedef: bülten)
--
-- 0117 bandı KAPALI kurdu. Bu migration bandı gerçek bir sayfaya bağlar:
-- `/newsletters` — sayfa var, kayıt formu çalışıyor. Uydurma "etkinlik
-- yakında" metni yazmıyoruz; bant var olan bir hizmeti duyuruyor.
--
-- Geri sayım (bitis) BOŞ: sahte aciliyet üretmemek için. Editör panelden
-- bir tarih girerse bant gerçek geri sayım gösterir.
--
-- Editör sonradan /admin/ayarlar?seke=promo üzerinden metni değiştirebilir
-- veya bandı kapatabilir; bu yalnız başlangıç değeridir.

UPDATE settings SET value = 'true'::jsonb, updated_at = now()
WHERE section = 'promo' AND key = 'aktif';

UPDATE settings SET value = to_jsonb('Günün kripto özeti her sabah e-postanızda'::text), updated_at = now()
WHERE section = 'promo' AND key = 'metin';

UPDATE settings SET value = to_jsonb('/newsletters'::text), updated_at = now()
WHERE section = 'promo' AND key = 'baglanti';

UPDATE settings SET value = to_jsonb('Abone ol'::text), updated_at = now()
WHERE section = 'promo' AND key = 'baglanti_metin';

UPDATE settings SET value = to_jsonb(''::text), updated_at = now()
WHERE section = 'promo' AND key = 'bitis';
