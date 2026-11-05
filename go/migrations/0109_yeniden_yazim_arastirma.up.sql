-- Faz 205 — yeniden yazımın "araştırmalı" modu.
--
-- ÖLÇÜM (3 Kasım 2026, üretim): yayındaki 982 ince içeriğin TAMAMI tek
-- cümlelik RSS özetinden türemiş (kaynak 94-228 karakter). Pilot, "yeni olgu
-- uydurma" kuralıyla bu kaynaktan 700 karakterlik haber üretilemediğini
-- gösterdi: 20/20 kapıda kaldı. Metni büyütmenin tek dürüst yolu GERÇEK
-- KAYNAK eklemek.
--
-- Bu yüzden öneriye iki şey eklendi:
--   mod      'yeniden' (yalnız mevcut metni düzenle) | 'arastirmali' (web
--            aramasıyla kaynak topla, haber oradan genişlet)
--   sources  kullanılan gerçek kaynaklar; editör onaydan önce görebilir
--
-- Kaynaklar öneriyle birlikte saklanır: onaylandıktan sonra "bu haberi neye
-- dayanarak yazdın" sorusunun cevabı veritabanında kalır (BİK / düzeltme).
ALTER TABLE yeniden_yazim
    ADD COLUMN IF NOT EXISTS mod     TEXT  NOT NULL DEFAULT 'arastirmali',
    ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Eski öneriler modsuz üretildi; onlar 'yeniden' sayılır.
UPDATE yeniden_yazim SET mod = 'yeniden' WHERE mod = 'arastirmali' AND id <= 20;

CREATE INDEX IF NOT EXISTS yeniden_yazim_mod_idx ON yeniden_yazim (mod, status);
