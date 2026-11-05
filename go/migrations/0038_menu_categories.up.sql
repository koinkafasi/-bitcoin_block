-- 0038_menu_categories: Üst menü + akış sekmeleri için kategori seed (Güncel/Teknoloji/Oyun/Bilim/Otomobil/Diğer)
INSERT INTO categories (name, slug, in_header, in_menu, status, sort_order, icon) VALUES
('Teknoloji','teknoloji', true, true, 'yayinda', 1, 'technology'),
('Oyun','oyun',         true, true, 'yayinda', 2, 'gaming'),
('Bilim','bilim',       true, true, 'yayinda', 3, 'science'),
('Otomobil','otomobil', true, true, 'yayinda', 4, 'car')
ON CONFLICT (slug) DO UPDATE SET in_header=true, in_menu=true, sort_order=EXCLUDED.sort_order, status='yayinda';