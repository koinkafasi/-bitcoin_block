-- 0131 geri alımı: yalnız bu migration'ın yarattığı iki tablo.
-- Başka tabloya dokunulmaz (0130 visitor_locations KALIR).
DROP TABLE IF EXISTS ip_asn;
DROP TABLE IF EXISTS visitor_risk;
