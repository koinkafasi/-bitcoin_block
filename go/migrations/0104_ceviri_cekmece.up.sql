-- Faz 195 — çekmece ve hesap sayfası için yeni arayüz anahtarları.
--
-- Neden: çekmecede "Yazı boyutu" ve başlıktaki "menü" sabit Türkçeydi;
-- İngilizce sitede de Türkçe görünüyordu. Personel için eklenecek
-- "Yönetim paneli" kısayolunun da çevirisi olmalı.
--
-- ON CONFLICT DO NOTHING: panelden düzenlenmiş çeviri ezilmez.
INSERT INTO translations (locale, key, value) VALUES
  ('tr', 'font_size',    'Yazı boyutu'),
  ('en', 'font_size',    'Font size'),
  ('tr', 'staff_panel',  'Yönetim paneli'),
  ('en', 'staff_panel',  'Admin panel'),
  ('tr', 'admin_group',  'Yönetim'),
  ('en', 'admin_group',  'Administration'),
  ('tr', 'menu_close',   'Kapat'),
  ('en', 'menu_close',   'Close')
ON CONFLICT (locale, key) DO NOTHING;
