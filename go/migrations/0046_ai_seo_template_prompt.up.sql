-- theblock — AI sohbet "SEO Meta" şablonundaki işlenmeyen {baslik}
-- yer tutucusunu kaldır (0046). Sohbet formunda başlık bağlamı yok;
-- diğer şablonlar gibi "Verilen metin için..." ifadesi kullanılır.
UPDATE ai_templates
   SET prompt = 'Verilen metin için SEO başlığı (40-70 kr) ve meta açıklaması (120-160 kr) üret.'
 WHERE kind = 'sohbet'
   AND prompt LIKE '%{baslik}%';
