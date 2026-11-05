-- 0091 down: eklenen kurumsal sayfaları kaldır, güncellenenlerin içeriğini geri al.
-- Kilitli yasal sayfalar silinmez; yalnızca bu migration'da eklenenler kaldırılır.
DELETE FROM pages WHERE slug IN ('kvkk', 'kvkk-basvuru', 'telif-haklari', 'editoryal-ilkeler', 'bilgi-toplum-hizmetleri');
DELETE FROM redirects WHERE source_path IN ('/bize-ulasin', '/sayfa/bize-ulasin', '/kvkk-aydinlatma', '/sayfa/kvkk-aydinlatma', '/iletisim-formu');
