-- 0073: bölüm bilgisi eksik/genel kalan HTML ve sitemap öğelerini yeniden kontrol et
-- JSON-LD breadcrumb ve unquoted script desteği eklendikten sonra bu öğeler
-- kontrollü kategori kuyruğunda yeniden işlenir.
UPDATE ai_source_items i
SET category_checked_at = NULL
FROM feeds f
WHERE f.id = i.feed_id
  AND f.kind IN ('html','sitemap')
  AND i.enriched_at IS NOT NULL
  AND (i.section = '' OR lower(i.section) IN ('news','haber','haberler','genel','general','default'));
