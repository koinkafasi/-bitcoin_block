-- 0117 geri alma: promo banner ayarları silinir.
--
-- Yalnız bu migration'ın eklediği satırlar silinir; settings bölümü genel
-- bir tablo olduğu için başka anahtarlara dokunulmaz.
DELETE FROM settings WHERE section = 'promo'
  AND key IN ('aktif', 'metin', 'baglanti', 'baglanti_metin', 'bitis');
