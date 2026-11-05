-- 0094 down: iletişim mesajları tablosu ve iletişim ayarı kaldırılır.
-- Sayfa metinlerindeki yer tutucular olduğu gibi kalır (varsayılan değerler
-- koddan üretildiği için sayfalar çalışmaya devam eder).
DROP TABLE IF EXISTS contact_messages;
DELETE FROM settings WHERE section = 'iletisim' AND key = 'ana';
