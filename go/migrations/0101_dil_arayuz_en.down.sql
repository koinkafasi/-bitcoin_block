-- Faz 189 geri alma: 0101'in eklediği arayüz anahtarları silinir.
DELETE FROM translations
WHERE key IN (
  'skip_to_content','search_hint','subcategories','services','preferences','other','sections',
  'related_more','related_less',
  'news_count','category','menu','search_title','archive_title','no_results',
  'empty_category','load_more','read_article','published','updated','min_read',
  'share_whatsapp','share_x','share_facebook','share_linkedin','copy_link','copied'
)
AND locale IN ('en','tr');

DELETE FROM translations WHERE locale = 'en' AND key LIKE 'footer.%';
