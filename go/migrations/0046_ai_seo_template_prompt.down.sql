-- 0046 geri alma: SEO Meta şablonuna yer tutucuyu geri koy.
UPDATE ai_templates
   SET prompt = '{baslik} için SEO başlığı (40-70 kr) ve meta açıklaması (120-160 kr) üret.'
 WHERE kind = 'sohbet'
   AND prompt = 'Verilen metin için SEO başlığı (40-70 kr) ve meta açıklaması (120-160 kr) üret.';
