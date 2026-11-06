-- 0117: promo banner (§3) ayarları
--
-- CoinDesk ana sayfa sırası §3: fiyat şeridinin hemen altında tek satırlık
-- duyuru bandı ("etkinlik geri sayımı"). Bizde etkinlik takvimi YOK, bu
-- yüzden uydurma geri sayım basmıyoruz; bant **ayarla** yönetilir ve
-- yalnız gerçek bir hedefe (bülten, fiyatlar, araştırma…) bağlanır.
--
-- Neden settings: metin/bağlantı değişince kod deploy'u gerekmesin.
-- Bant kapalıyken (aktif=false) ana sayfada HİÇ basılmaz — boş bir çerçeve
-- bırakmak düzeni bozar.
--
-- bitis: isteğe bağlı ISO-8601 zaman. Doluysa sunucu "3 gün 4 saat kaldı"
-- yazar (JS YOK — CSP satır içi script'e izin vermiyor). Geçmişse bant
-- otomatik kapanır, yani süresi dolmuş kampanya kendiliğinden düşer.

INSERT INTO settings (section, key, value) VALUES
    ('promo', 'aktif',          'false'::jsonb),
    ('promo', 'metin',          '""'::jsonb),
    ('promo', 'baglanti',       '""'::jsonb),
    ('promo', 'baglanti_metin', '""'::jsonb),
    ('promo', 'bitis',          '""'::jsonb)
ON CONFLICT (section, key) DO NOTHING;
