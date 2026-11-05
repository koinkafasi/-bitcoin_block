-- theblock — hazır AI sohbet şablonları (0008)
INSERT INTO ai_templates (kind, title, prompt, icon, sort_order) VALUES
('sohbet','Güncel Gündem','Güncel gündemdeki önemli başlıkları 5 maddeyle listele.','📰',1),
('sohbet','Son Dakika Taslağı','Son dakika haberi için kısa ve net bir taslak yaz (başlık + 3 paragraf).','⚡',2),
('sohbet','Başlık Öner','Verilen metin için 5 haber başlığı öner.','✍️',3),
('sohbet','SEO Meta','{baslik} için SEO başlığı (40-70 kr) ve meta açıklaması (120-160 kr) üret.','🔍',4),
('sohbet','Sosyal Medya','Haberi X/Facebook için paylaşım metnine dönüştür (kısa + hashtag).','📣',5),
('sohbet','Metni Özetle','Verilen metni 3 maddelik özete çevir.','📌',6),
('sohbet','Yeniden Yaz','Verilen metni daha akıcı ve özgün biçimde yeniden yaz.','🔄',7)
ON CONFLICT DO NOTHING;
