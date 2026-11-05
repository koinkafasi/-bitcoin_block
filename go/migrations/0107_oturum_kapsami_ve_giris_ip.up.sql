-- Oturum kapsamı + giriş denemesi IP'si (2026-11-02 güvenlik turu)
--
-- 1) sessions.scope: personel ("panel") ve okuyucu ("uye") oturumları aynı
--    tabloyu ve aynı tb_session çerezini paylaşıyordu. Panel ara katmanı
--    yalnız rolü denetlediği için, rolü sonradan yükseltilen bir kullanıcının
--    eski üye çerezi kendiliğinden panel yetkisi kazanıyordu. Kapsam kolonu
--    oturumun hangi kapıdan açıldığını kalıcı olarak kaydeder.
--    Varsayılan 'panel': mevcut oturumlar personel oturumuydu; okuyucu
--    oturumları zaten requireAdmin'de StaffRoles denetimine takılır.
--
-- 2) login_attempts.ip: hız sınırı yalnız e-posta bazlıydı; bilinen bir
--    e-postaya 5 hatalı deneme yapmak hesabı 10 dakika kilitliyordu
--    (hesap hedefli hizmet engelleme). IP kolonu, sayacın yalnız aynı
--    kaynaktan gelen denemeleri saymasını sağlar.

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS scope text NOT NULL DEFAULT 'panel';

ALTER TABLE login_attempts ADD COLUMN IF NOT EXISTS ip text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_login_attempts_email_ip
    ON login_attempts (email, ip, attempted_at);
