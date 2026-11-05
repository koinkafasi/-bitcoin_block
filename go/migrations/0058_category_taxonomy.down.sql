-- 0058 geri alma: görünürlük/sıralamayı eski hâle döndürmek yerine
-- tüm alt kategorileri menüde görünür bırakır.
UPDATE categories SET in_menu = TRUE, in_footer = TRUE WHERE parent_id IS NOT NULL;
UPDATE categories SET in_header = TRUE WHERE slug = 'diger';
