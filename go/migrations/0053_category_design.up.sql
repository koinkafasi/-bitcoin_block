-- 0053: kategoriye özel liste görünümü ve tasarım şablonu
--
-- ListView: kategori sayfasında genel "kategori görünümü" ayarını ezer
--           (klasik | liste | izgara | mozaik; boş = genel ayar).
-- LayoutPreset: uygulanan hazır şablonun anahtarı (bilgi/geri izleme).

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS list_view TEXT NOT NULL DEFAULT '';

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS layout_preset TEXT NOT NULL DEFAULT '';
