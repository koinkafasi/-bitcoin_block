-- 0119: uzman/editoryal profil görseli (avatar_url)
--
-- NEDEN: "Görüş" modülü yazar adı + 48 px yuvarlak avatar basıyor.
-- Avatar şimdiye kadar `media_files` üzerinden geliyordu; uzman
-- profilleri dış kaynaklı (CoinDesk yazar fotoğrafı) olduğu için
-- medya kütüphanesine dosya yüklemeden mutlak adres saklanıyor.
--
-- Avatar yalnız GÖRÜNÜM içindir; boşsa modül avatar basmaz.

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';
