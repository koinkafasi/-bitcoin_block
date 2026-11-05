-- Faz 197 — Gazete Manşetleri modülü (ana sayfa, sağ sütun).
--
-- NEDEN MIGRATION: blok düzeni veritabanında (page_layouts.blocks) tutuluyor;
-- kod dağıtımı tek başına modülü ana sayfaya yerleştirmez. Bu migration
-- düzeni BİR KEZ bozar — tekrar çalışmaz (idempotent).
--
-- ÖLÇÜM (üretim, 2 Kasım 2026, dağıtım öncesi):
--   page_layouts.home  → 19 blok, hepsi zone'suz (→ "ana")
--   .home-side         → BOŞ, ama ızgara sağ sütuna %33 yer ayırıyor
--   http://127.0.0.1:8070/  → <div class="home-grid" style="--pg-side-w:33%">
--                             <aside class="home-side"></aside>
-- Yani sayfanın sağ üçte biri bugün boş duruyordu. Gazete manşetleri
-- modülü tam olarak oraya (zone=yan) yerleşir.
--
-- GERİ ALMA: aşağıda eski düzen page_layout_versions'a yazılır. Panelde
-- "Sayfa Tasarımcısı → Sürümler" üzerinden o sürüm geri yüklenebilir.
DO $$
DECLARE
    mevcut jsonb;
    yeni   jsonb;
BEGIN
    SELECT blocks INTO mevcut
      FROM page_layouts
     WHERE site_id = 1 AND page_key = 'home';

    -- Düzen yoksa (yeni kurulum) varsayılan akış kullanılır; dokunma.
    IF mevcut IS NULL THEN
        RETURN;
    END IF;

    -- Modül zaten varsa ikinci kez ekleme.
    IF EXISTS (
        SELECT 1 FROM jsonb_array_elements(mevcut) b
         WHERE b->>'type' = 'gazete_manset'
    ) THEN
        RETURN;
    END IF;

    yeni := mevcut || jsonb_build_array(jsonb_build_object(
        'id',      'gzm-' || substr(md5(random()::text), 1, 8),
        'type',    'gazete_manset',
        'visible', true,
        'opts', jsonb_build_object(
            'zone',          'yan',
            'count',         12,
            'gazete_aralik', 6,
            'gazete_baslik', 'Gazete Manşetleri'
        )
    ));

    -- Geri alma izi: önceki düzen sürüm geçmişine yazılır.
    INSERT INTO page_layout_versions (site_id, page_key, blocks, note)
    VALUES (1, 'home', mevcut, 'Faz 197: gazete manşetleri modülü eklenmeden önceki düzen');

    UPDATE page_layouts
       SET blocks = yeni, updated_at = now()
     WHERE site_id = 1 AND page_key = 'home';
END $$;
