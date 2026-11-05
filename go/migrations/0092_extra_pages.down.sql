-- 0092 down: ek sayfaları ve yönlendirmeleri kaldır.
DELETE FROM pages WHERE slug IN ('insan-kaynaklari', 'sikca-sorulan-sorular');
DELETE FROM redirects WHERE source_path IN ('/hakkimizda', '/about', '/kariyer', '/sss');
