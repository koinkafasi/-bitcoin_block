-- 0131: Trafik güvenliği — ziyaretçi risk profili + ASN önbelleği.
--
-- NEDEN: site yalnız "kaç okunma" sayıyordu; sayının NE KADARININ gerçek
-- olduğunu bilmiyorduk. Olay yeri: sayfa önbelleği isabetinde
-- `okunmaTamponu.ekle` istek sahibine BAKMADAN çağrılıyordu
-- (sayfa_onbellek.go). Yani tek bir bot, aynı sayfayı 10 000 kez isteyip
-- 10 000 okunma yazdırabiliyordu. Bu tablo, sayımın önüne konan kararın
-- kalıcı kaydıdır: kim (tarayıcı kimliği), hangi ağdan (ASN/sağlayıcı),
-- hangi sinyallerle, hangi puanda.
--
-- SINIFLANDIRMA NEDEN VAR: "bot" tek kutu değil. Ölçülen üç ayrı şey var:
--   1) VERİ MERKEZİ (hosting) — Hetzner/DigitalOcean/OVH. Gerçek okuyucu
--      neredeyse hiç gelmez; sayılmaz, varsayılan olarak engellenir.
--   2) TİCARİ VPN / Tor — NordVPN, Mullvad, Cloudflare WARP… Bunların
--      arkasında GERÇEK insan var. Engellemek okuyucu kaybettirir; bu
--      yüzden varsayılan "işaretle + sayma + reklam gösterme", engel
--      yönetici açarsa (guvenlik_vpn_engel) devreye girer.
--   3) BOT — UA boş/otomasyon, tuzak yolu, sahte Googlebot, aşırı hız.
--      Sayılmaz ve engellenir.
--
-- GİZLİLİK: IP saklanır (tekrar/rotasyon tespiti için) ama panelde
-- MASKELENİR. Satır tarayıcı kimliği (tb_vid çerezi) başına TEKTİR;
-- okunma sayısıyla birlikte büyümez. Kişisel kimlik (ad/e-posta/oturum)
-- ile birleştirilmez.
--
-- `ip_asn` NEDEN AYRI TABLO: ASN sorgusu Team Cymru DNS'e gider (ücretsiz,
-- anahtarsız). Ölçüldü: süreç içi `net.LookupTXT` ortanca 1.4 ms, soğuk
-- DNS'te 234 ms. Cevap /24 başına aynı olduğundan burada saklanır; süreç
-- yeniden başladığında önbellek soğuk başlamasın.

CREATE TABLE IF NOT EXISTS visitor_risk (
    -- anahtar: tb_vid çerezi; çerez yoksa "ip:<adres>". PK metin çünkü
    -- iki tür kimliği tek tabloda tutuyoruz ve çerezsiz istek için UUID
    -- uydurmak yanlış olurdu (uydurma kimlik gerçek ziyaretçiyi böler).
    anahtar       TEXT PRIMARY KEY,
    visitor_id    UUID,
    ip            INET,
    ip_ag         TEXT,             -- ASN'in gerçek CIDR'i (Team Cymru)
    asn           INT,
    asn_ad        TEXT,             -- "HETZNER-AS - Hetzner Online GmbH, DE"
    asn_ulke      TEXT,
    -- asn_tur: hosting | vpn | tor | mobil | kurumsal | konut | bilinmiyor
    asn_tur       TEXT NOT NULL DEFAULT 'bilinmiyor',
    ulke          TEXT,             -- CF-IPCountry
    puan          INT  NOT NULL DEFAULT 0,
    -- sinif: temiz | supheli | yuksek | bot | arama
    sinif         TEXT NOT NULL DEFAULT 'temiz',
    vpn           BOOLEAN NOT NULL DEFAULT FALSE,
    proxy         BOOLEAN NOT NULL DEFAULT FALSE,
    hosting       BOOLEAN NOT NULL DEFAULT FALSE,
    tor           BOOLEAN NOT NULL DEFAULT FALSE,
    bot           BOOLEAN NOT NULL DEFAULT FALSE,
    -- sinyaller: [{"k":"asn_hosting","p":45,"n":"Hetzner"}, …]
    -- JSONB: yeni sinyal eklemek şema değişikliği gerektirsin istemedik.
    sinyaller     JSONB NOT NULL DEFAULT '[]'::jsonb,
    webrtc_ip     INET,             -- tarayıcının bildirdiği, sunucudan farklı IP
    ip_degisim    INT NOT NULL DEFAULT 0,
    ulke_degisim  INT NOT NULL DEFAULT 0,
    istek_sayisi  BIGINT NOT NULL DEFAULT 0,
    ilk_gorulme   TIMESTAMPTZ NOT NULL DEFAULT now(),
    son_gorulme   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Panelin ana sorgusu: "son N günde en riskli ziyaretçiler".
CREATE INDEX IF NOT EXISTS ix_visitor_risk_puan
    ON visitor_risk (puan DESC, son_gorulme DESC);
-- KPI sayımları son_görülme penceresiyle yapılır.
CREATE INDEX IF NOT EXISTS ix_visitor_risk_son
    ON visitor_risk (son_gorulme DESC);
-- "VPN/hosting oranı" filtresi.
CREATE INDEX IF NOT EXISTS ix_visitor_risk_sinif
    ON visitor_risk (sinif, son_gorulme DESC);

COMMENT ON TABLE visitor_risk IS
    'Ziyaretçi risk profili: ASN/sağlayıcı sınıfı + sinyal puanı. Sayım kapısının kaydı.';
COMMENT ON COLUMN visitor_risk.anahtar IS
    'tb_vid çerezi; çerez yoksa "ip:<adres>". Kişi değil tarayıcı/istek.';
COMMENT ON COLUMN visitor_risk.sinif IS
    'temiz | supheli | yuksek | bot | arama (doğrulanmış arama motoru)';
COMMENT ON COLUMN visitor_risk.webrtc_ip IS
    'Tarayıcı WebRTC ile sızdırdığı ve sunucunun gördüğünden FARKLI genel IP.';

CREATE TABLE IF NOT EXISTS ip_asn (
    -- ag: Team Cymru'nun döndürdüğü CIDR. /24 yerine gerçek önek: aynı
    -- sağlayıcı 5.9.0.0/16 gibi geniş blokta tek satırda durur.
    ag          TEXT PRIMARY KEY,
    asn         INT NOT NULL,
    asn_ad      TEXT NOT NULL DEFAULT '',
    ulke        TEXT,
    tur         TEXT NOT NULL DEFAULT 'bilinmiyor',
    -- kaynak: 'cymru' | 'elle' (ileride itibar servisi eklenirse)
    kaynak      TEXT NOT NULL DEFAULT 'cymru',
    guncelleme  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_ip_asn_guncelleme ON ip_asn (guncelleme DESC);

COMMENT ON TABLE ip_asn IS
    'ASN önbelleği (Team Cymru DNS). Ücretsiz, anahtarsız; /24 yerine gerçek önek.';
