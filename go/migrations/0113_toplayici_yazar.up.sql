-- 0113: toplayıcı sistem yazarı
--
-- NEDEN: `contents.author_id` NOT NULL. Toplanan haberlerin tek tek yazarı
-- yok (kaynak gösterilir), ama kayıt bir yazara bağlanmak zorunda. Bu yüzden
-- TEK bir sistem kullanıcısı tanımlanır; içerikte yazar alanı
-- "bitcosar / kaynak adı" olarak gösterilir.
--
-- `password_hash` bilinçli olarak GEÇERSİZ bir değerdir ('!'); bu hesapla
-- giriş yapılamaz (bcrypt doğrulaması başarısız olur).

INSERT INTO users (email, name, password_hash, role, status)
VALUES ('toplayici@bitcosar.com', 'bitcosar Toplayıcı', '!', 'Editör', 'aktif')
ON CONFLICT (email) DO NOTHING;

-- İçerik üretiminde kullanılacak kimliği sabitle (göçlerde tekrar
-- aranmasın): email üzerinden erişilir, sabit id varsayılmaz.
