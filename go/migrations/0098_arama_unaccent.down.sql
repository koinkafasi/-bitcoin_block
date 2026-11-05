DROP INDEX IF EXISTS contents_title_unaccent_trgm;
DROP INDEX IF EXISTS contents_slug_unaccent_trgm;
DROP INDEX IF EXISTS categories_name_unaccent_trgm;
DROP INDEX IF EXISTS media_filename_unaccent_trgm;
DROP FUNCTION IF EXISTS public.immutable_unaccent(text);
-- unaccent uzantısı BIRAKILMAZ: başka sorgular da kullanabilir ve
-- uzantıyı düşürmek ona bağlı nesneleri de etkiler.
