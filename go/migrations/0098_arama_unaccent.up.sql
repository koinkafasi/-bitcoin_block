-- Türkçe arama katlaması (Faz 165 devamı).
--
-- NEDEN: panel aramasında "saglik" yazan kullanıcı "Sağlık Bakanlığı"
-- içeriğini bulamıyordu. PostgreSQL'in ILIKE'ı aksan/nokta farkına duyarlı:
-- 'Sağlık' ILIKE '%saglik%' → false (ölçüldü).
--
-- unaccent, ğ→g ve ı→i eşlemesini yapıyor (ölçüldü:
-- unaccent('Sağlık Bakanlığı') = 'Saglik Bakanligi').
CREATE EXTENSION IF NOT EXISTS unaccent;

-- neden sarmalayıcı: unaccent() STABLE işaretli, IMMUTABLE değil. PostgreSQL
-- indeks ifadesinde IMMUTABLE olmayan fonksiyonu reddediyor:
--   "functions in index expression must be marked IMMUTABLE" (42P17)
-- Bu hata bir kez yaşandı (migration geri alındı). Sözlük sabit olduğu için
-- (unaccent.rules dosyadan yüklenir, oturum boyunca değişmez) sarmalayıcıyı
-- IMMUTABLE işaretlemek güvenli; yine de iki argümanlı biçim kullanılıyor ki
-- arama_yolu'na bağlı kalmasın.
CREATE OR REPLACE FUNCTION public.immutable_unaccent(text)
RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
STRICT
AS $$ SELECT public.unaccent('public.unaccent', $1) $$;

-- Arama ifadesi indeksten yararlansın diye ifade indeksi. Sorgu da AYNI
-- ifadeyi kullanmalı (lower(public.immutable_unaccent(...))), yoksa indeks
-- kullanılmaz.
CREATE INDEX IF NOT EXISTS contents_title_unaccent_trgm
  ON contents USING gin (lower(public.immutable_unaccent(title)) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS contents_slug_unaccent_trgm
  ON contents USING gin (lower(public.immutable_unaccent(slug)) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS categories_name_unaccent_trgm
  ON categories USING gin (lower(public.immutable_unaccent(name)) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS media_filename_unaccent_trgm
  ON media_files USING gin (lower(public.immutable_unaccent(filename)) gin_trgm_ops);
