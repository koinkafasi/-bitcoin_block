-- theblock — kurumsal sayfa seed (0007): Künye + İletişim.
-- Kurumsal sayfalar kilitlidir (silinemez — 02 §17).
INSERT INTO pages (kind, title, slug, spot, detail, status, in_menu, in_footer, is_locked) VALUES
('kurumsal','Künye','kunye', 'theblock.com.tr künye bilgileri.',
 '[{"type":"p","text":"theblock.com.tr — güncel haberler, analiz ve yerel gelişmeler. İnternet haber sitesi kategorisinde yayın yapar."},{"type":"p","text":"İçerikler editör onayından geçer; kaynaklar belirtilir."}]'::jsonb,
 'yayinda', false, true, true),
('kurumsal','İletişim','iletisim', 'Reklam, haber ve teknik destek için iletişim.',
 '[{"type":"p","text":"İletişim: info@theblock.com.tr"},{"type":"p","text":"Reklam: reklam@theblock.com.tr"}]'::jsonb,
 'yayinda', false, true, true)
ON CONFLICT (slug) DO NOTHING;
