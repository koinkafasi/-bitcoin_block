-- Faz 197 geri alma: gazete manşetleri modülünü düzenden çıkarır.
-- Kapak verileri (newspaper_pages) SİLİNMEZ: modül yeniden eklenirse
-- aynı günün kapakları yerinde olur.
DO $$
DECLARE
    mevcut jsonb;
    yeni   jsonb;
BEGIN
    SELECT blocks INTO mevcut
      FROM page_layouts
     WHERE site_id = 1 AND page_key = 'home';

    IF mevcut IS NULL THEN
        RETURN;
    END IF;

    SELECT jsonb_agg(b) INTO yeni
      FROM jsonb_array_elements(mevcut) b
     WHERE b->>'type' <> 'gazete_manset';

    IF yeni IS NULL THEN
        yeni := '[]'::jsonb;
    END IF;

    UPDATE page_layouts
       SET blocks = yeni, updated_at = now()
     WHERE site_id = 1 AND page_key = 'home';
END $$;
