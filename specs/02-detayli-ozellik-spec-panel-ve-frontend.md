# theblock — Detaylı Özellik Spec (Panel Kılavuzu + Frontend/Demo) v0.2

**Tarih:** 2026-10-14
**Kaynaklar:**
- (S4) https://www.haberyazilimiai.com/panel-kullanim-kilavuzu.html — 26 bölümlük yönetim paneli kılavuzu
- (S5) https://medyamugla.com — canlı demo site (frontend yapısı, URL desenleri, servisler)
- (S1) özellik sayfası + (S2) video altyazıları + (S3) songazete.com incelemesi
- (S6) https://www.haberyazilimai.com/assets/img/neden-{2,3,4,5}.png — rakip ürün ekran görüntüleri (AI asistan · analitik · görsel manşet editörü · SEO/yayın)
- (S7) https://kantan.news — AI-küratörlü, reklamsız çok dilli (tr/en/es) teknoloji/gündem platformu (kategori AI-skor kapıları, trends API, bülten/podcast/ping, topluluk+gamification) — ayrıntı: `06-kantan-news-referans.md`

> Bu doküman GELİŞTİRME referansıdır: her satır kodlanabilir seviyede kesinleştirilmiştir.
> Öncelik/dalga eşlemesi: `03-spec-plan.md`.

---

## 1. İçerik Türleri (12 tür, tek akıllı form) [S4 §05]
| Tür | Özel alan/not |
|---|---|
| Haber | Mahreç + galeri + manşet+seo görseli + tüm manşet rozetleri |
| Fotoğraf (Galeri) | Galeri (dikey/yatay) |
| Video | Video ekleme (3 yol) |
| Makale | Mahreç gizli; köşe yazısı rozeti |
| Resmi İlan | Basın İlan Kodu alanı (BİK) |
| E-Gazete | Basım Tarihi, Sayı; sayfalar galeriye görsel yüklenir |
| Röportaj | Haber formu + "Canlı Akış" eklentisi kapalı |
| Anket | Soru + sürüklenebilir cevap listesi (metin, opsiyonel görsel, elle **Oy** sayısı, varsayılan 2 cevap) |
| Biyografi | Sadeleştirilmiş form (galeri vurgulu) |
| Firma Rehberi | Harita konumu, Adres, Telefon, Web, E-posta |
| Seri İlan | İlan Sahibi, Fiyat (₺), Adres, Telefon, Web, E-posta |
| Vefat | Sadeleştirilmiş form |

**Ortak form alanları [S4 §05]:**
- Başlık, Spot (özet), Detay (zengin metin)
- 3 görsel kutusu: **İçerik Resmi** (kapak), **Manşet Resmi**, **Sosyal Medya Resmi** — "Sürükle, tıkla veya çek"
- **Otomatik 16:9 kırpma** — yükleme anında kırpma penceresi açılır, "Kırp ve Kaydet"
- **Dosyalarımdan Seç** — geçmiş görseller, oran uymazsa turuncu uyarı
- Kategori, Mahreç, Galeri, Video, Devmanşet, Kısayollar
- Kaydetme: **Arşive Kaydet** / **Yayınla**; ileri tarih → Planlanmış listesine

## 2. İçerik Listesi & Hızlı İşlemler [S4 §06]
- 5 durum sekmesi + sayaç rozetleri: **Yayında · Taslak · Planlanmış · Arşiv · Çöp Kutusu** (rozetler hafif istekle sonradan yüklenir)
- Manşet rozetleri (tek tık, onaysız): **AM** (Anamanşet) · **ÜM** · **DM** (Devmanşet) · **SD** (Son Dakika) · **FH** (Flaş) · **HK** (Hikaye)
- Paylaş (FB/X/WA/TG/LinkedIn/Kopyala; IG uyarılı), Görüntüle, Arşive Taşı/Yayınla, Sil
- **3 saniyelik geri sayım** onayı (silme/arşivleme); ikon geri-oka döner
- **Hit (okunma) + paylaşım sayaçları** satır rozetleri (otomatik)
- **Çöp kutusu 30 gün**, sonra kalıcı silme; geri yükleme (Düzenle→Kaydet/Yayınla)
- **Dışa Aktar (.csv)**: aktif filtreye göre, ≤1000 satır, Türkçe karakter uyumlu; yalnız İçerikler/Yorumlar/Kullanıcılar listelerinde (ID, Başlık, Spot, Kategori, Editör, Mahreç, Hit, SEO Skoru)

## 3. Başlık & Metin Araçları [S4 §07]
- **Otomatik başlık tamamlama**: ≥3 karakter → Google suggest listesi (slug'ı doldurur) — kural tabanlı, AI değil
- **Aa** büyük/küçük harf dönüşümü (Türkçe İ/ı uyumlu); ikincil buton: yalnız ilk harf
- **Etiket önerisi**: başlık+spot'tan kural tabanlı kelime çıkarımı (AI değil)
- **Kelime sayacı**: "Paragraf: X — Kelime: Y" canlı

## 4. SEO Puanlama — 10 kriter, ağırlıklı %100 [S4 §09]
| Kriter | Ağırlık | İdeal |
|---|---|---|
| Başlık uzunluğu | %20 | 40–70 karakter |
| Ana görsel | %15 | ≥1 |
| Spot uzunluğu | %15 | 120–160 |
| Etiket sayısı | %12 | 3–5 |
| Kelime sayısı | %10 | 300–600 |
| H2 sayısı | %10 | 2–4 |
| Paragraf sayısı | %5 | ≥4 |
| İç görsel | %5 | 1–2 (3+ puan düşürür) |
| Kalın (bold) | %5 | 3–5 (aşırı → sıfır) |
| İtalik | %3 | 1–2 |

- Renkli ilerleme çubukları (kırmızı→sarı→yeşil) + dairesel 0–100 gösterge
- **SERP Önizleme**: başlık 70ch sayaç (40-60 yeşil, 61-70 sarı, 71+ kırmızı); açıklama 160ch (120-160 yeşil, 161-200 sarı, 201+ kırmızı) — engel değil, öneri

## 5. Sesli Yazma [S4 §10]
- Tarayıcı Web Speech; **Otomatik** mod (cümle sonu durur) ve **Toplantı modu** (manuel); kısayol Ctrl+Shift+V
- Otomatik noktalama yok; yaygın yazım düzeltmesi; uyarılar: mikrofon izni/ses/mikrofon/internet

## 6. Video & Galeri [S4 §11-12]
- Video 3 yol: dosya (mp4/webm/mov) · **YouTube** (manuelle link/kod + **YouTube'da Ara** sekmesi: Göster/Kaydet) · platform (Vimeo, Dailymotion, Twitch, FB, IG, TikTok, generic Embed)
- "İçerikte resim yerine bu video görünsün" (liste/manşette oynatıcı)
- Galeri görünümleri: **Dikey** (kare ızgara, varsayılan haber/makale) / **Yatay** (görsel+spot liste); canlı önizleme; her görsele başlık+açıklama; drag sıralama; **kapak = ilk sıradaki görsel**

## 7. Kısayol Ekleme (editör içi gömme) [S4 §13]
- Tür: Haber/Makale/Foto Galeri/Video (**İçerik Ara** ≥4 karakter) · İletişim Formu/Anket/Puan Durumu/Fikstür (sabit link)
- Renk seçimi (hazır palet + özel); renkli kutu olarak gömülür

## 8. Görsel Manşet & Başlık Tasarım Editörü (katmanlı canvas) [S4 §14][S6]
> Kapsam: yalnız Dev Manşet değil; **her görsel kutusu** için (İçerik Resmi / Manşet Resmi / Sosyal Medya Resmi / galeri kapağı) ortak **katmanlı başlık editörü**. Dev Manşet (S4 §14) bu editörün sabit "zemin/rozet/3 başlık" şablonu gibi davranır.

**İki mod:**
- **Katman (canvas) modu** — serbest düzenleme [S6, neden-4] · **Dev Manşet form modu** — mevcut S4 §14 (sabit şablon olarak yüklenir)

**Katman listesi & davranış:**
- Metin katmanları: **Üst Başlık · Ana Başlık · Alt Başlık** (ayrı katmanlar) + opsiyonel rozet/logo/grafik katmanı
- **Sürükle ile konumlandırma** (serbest X/Y), seçim + köşe tutamaçları, ok tuşlarıyla 1px kaydırma, katman silme
- Görünürlük (göster/gizle) + **kilit** (konum/özellik değiştirilemez) + z-sırası (öne/arkaya) yeniden sıralama
- Katman seçiliyken yalnız o katmanın özellik paneli açılır (modal menü yok)

**Metin özellikleri (katman başına):**
- Metin · yazı tipi (vars. Montserrat/Merriweather + site görünüm fontu) · boyut · kalınlık (regular→black) · renk · hizalama (sol/orta/sağ) · **satır aralığı** · harf aralığı · gölge (renk/blur/offset)
- Varsayılan boyutlar: üst 24 / ana 36 / alt 18 px (S4 §14)

**Arka plan & kaplama:**
- Görsel seçimi (yeni yükle / Dosyalarımdan Seç / galeri) + arka plan rengi
- **Koyu/açık kaplama**: karartma gradyanı — renk, yön (sol/sağ), genişlik (% , vars. 70), iç boşluk (px, vars. 32) — Dev Manşet "Zemin" ile aynı kurallar
- **Rozet** katmanı: tür (Yok/SonDakika/Flaş/FotoHab/VideoHab/Köşe/Günün), animasyon (Yok/Nefes/Ritim/Shake/Flip/Tonyhawk/Blur), font+kalın, renk, iç boşluk 0.2–2, boyut (S4 §14)

**Şablonlar:**
- Hazır şablonlar: Manşet, Son Dakika, Foto Haber, Video Haber, Köşe, Spor, Ekonomi, Kültür-Sanat…
- **Kaydet/Güncelle/Sil + adlandırma**; seçince tüm katman ayarları yüklenir
- **Kategori/brand bazlı varsayılan şablon** (Ayarlar→Görünüm eşlemesi; ilgili kategorinin editöründe otomatik yüklenir)
- **AI "Öner"**: mevcut başlık+spot'tan 3'er seçenek; prompt ayarlardan ({baslik},{spot}, S4 §14)

**Önizleme & çıktı:**
- Canlı önizleme; boyut varyantlarında kontrol: **16:9 (manşet) · 4:3 içerik · 1:1 / 9:16 (sosyal)** — otomatik yeniden yerleşim veya varyant başına elle şablon
- **Dışa aktarma**: tasarım → sunucu tarafı PNG/WebP render (`image-svc`); önizleme ile dışa aktarılan görsel aynı kuralları kullanır

**Teknik (tasarım belgesi):**
- Tarayıcıda Canvas tabanlı editör; tasarım **JSON belge** olarak DB'de (`headline_designs`) saklanır; sunucuda render için `image-svc`'ye gider
- Metin taşması kontrolü (font/uzunluk, auto-uyarlama), font yönetimi (web-safe + site fontu), güvenli medya işleme (EXIF/script temizliği)
- Dev Manşet form modu bu JSON'a birebir maplenir (geriye uyum)

Örnek tasarım belgesi:
```json
{
  "version": 1,
  "canvas": { "width": 1200, "height": 675, "mode": "katman" },
  "background": { "asset_id": "media_123", "overlay": { "color": "#000000", "direction": "left", "spread": 70 } },
  "layers": [
    { "type": "text", "role": "ust_baslik", "text": "SON DAKİKA", "x": 80, "y": 60, "font": "Montserrat", "fontSize": 24, "weight": 700, "color": "#FFD700", "align": "left", "lineHeight": 1.2, "visible": true, "locked": false },
    { "type": "text", "role": "ana_baslik", "text": "Örnek haber başlığı", "x": 80, "y": 120, "font": "Merriweather", "fontSize": 48, "weight": 700, "color": "#FFFFFF", "align": "left", "lineHeight": 1.1 },
    { "type": "text", "role": "alt_baslik", "text": "Örnek haber özeti…", "x": 80, "y": 225, "font": "Merriweather", "fontSize": 18, "weight": 400, "color": "#FFFFFF", "align": "left" }
  ]
}
```

- **Kabul:** Tasarım kaydedilip yeniden açılınca konum/biçim birebir geri yüklenir (JSON deterministik); önizleme ile dışa aktarılan PNG/WebP aynı kurallarla render edilir; tüm boyut varyantlarında taşma/kırpılma olmaz; varsayılan şablon doğru kategoride otomatik yüklenir

## 9. AI Sohbet Ekranı [S4 §16]
- Haber paneli asistanı kimliği; gerekirse **otomatik web araması** (döviz/borsa/gündem); doğrudan yanıt (yazma/SEO)
- **Sohbet geçmişi**: başlık otomatik (ilk mesaj), mesaj sayısı, tarih; devam/tek tek sil/Tümünü Sil; **bağlam son birkaç mesajla sınırlı** (maliyet/hız)
- **Hazır şablonlar**: gündem, son dakika, siyaset, ekonomi, döviz, borsa, dünya, spor, sağlık, teknoloji, eğitim, magazin, hava, trafik, viral, manşet, başlık önerisi, sosyal medya uyarlaması (placeholder'lar \[tarih\]\[ilce\]\[sehir\]\[site_adi\] otomatik)
- **Sohbet→Haber**: seçili metin sağ tık → "Bunu Haber Yap" / "Araştırarak Haber Yap" → yeni haber formu

## 10. AI Şablon Yönetimi [S4 §15]
- **Sohbet şablonları** (`Ayarlar→Yapay Zeka`): başlık(buton), prompt (\[ilce\]\[sehir\]\[tarih\]\[site_adi\]), 24 hazır ikon, sıralama ok
- **Sağ tık özel şablonları**: menü başlığı, prompt ({kaynak} tüm haber, {secilen} seçili metin), ikon; mavi renkli alt menüde

## 11. AI ile Üretim (editör içi) [S4 §08]
- Model: **Anthropic Claude** + kullanıcı API anahtarı (referans üründe). **theblock kararı:** OmniRoute varsayılan + çoklu provider "API anahtarı ekle"
- **Alan bazlı menüler**: Başlık {Özgünleştir · Özete göre başlık · Detaya göre başlık} · Spot {Özgünleştir · Başlığa göre özet · Detaya göre özet} · Detay {Özgünleştir · Başlığa göre · Özete göre}; öneri tıklanır link → alana yazılır (otomatik değil, onaylı)
- **12 sağ tık komutu**: Seçimden Haber Oluştur · Genişlet (50-150 kelime) · İki Paragraf Ekle · Farklı Perspektiften Yaz · H2 Başlıklı Bölümler Ekle · Nedenleri Analiz Et · Sonuçlar & Etkiler · Karşılaştırma Paragrafı · Artı/Eksi Listesi · Adım Adım Uygulama · Pratik İpuçları · Kimdir? Biyografi
- **Prompt özelleştirme**: "Üst düzey bilgi sahibiyim" kilidi → komut başına prompt düzenleme ({kaynak},{secilen}); "Varsayılanları Geri Yükle"

## 12. Bot Modülü (Otomatik ajans haberi) [S4 §22]
- **10 ajans**: AA, İHA, İGF, Kapsül, BHA, BİH, ANKA, PERRE, Hibya, İLKHA; İGF/Kapsül/BHA/BİH ücretsiz; AA/İHA/ANKA/PERRE lisanslı (kullanıcı adı/şifre/API anahtarı)
- 3 sekme: **Haber Listesi** (5 dk otomatik yenileme sayacı, ham metin düzenleme, kategori/editör/mahreç/manşet rozetleri, **Kelime Değiştirici** sağ tık → listeye ekle ve tüm metinden temizle) · **Ayarlar** (kategori eşleştirme, varsayılan editör, varsayılan mahreç, kelime değiştirme listesi, AI prompt {baslik}{detay}{kategori}{kaynak}) · **Otomatik** (Ekleme Süresi 5 dk–1 sa, "Yapay Zeka ile Ekle" anahtarı)
- **Kaydet** vs **AI Kaydet**: AI Kaydet metni kendi promptunla yeniden yazar (özgünlük/SEO)

## 13. Yorumlar [S4 §23]
- Yorumlar **asla otomatik yayınlanmaz**: Onay Bekliyor (sarı)/Onaylı (yeşil); rozete tık da durum değiştirir; "reddet" yok
- Detaylı filtre: kelime, içerik türü, durum, **IP**, tarih
- Yorum Detayı: Onayla / Sil / Arşive Taşı; **toplu onay yok**, toplu sil var
- Cevap yazma ve kelime bazlı spam engelleme YOK (referansta)
- Bekleyen rozeti: Anasayfa Keşfet + sol menü; CSV export

## 14. Kullanıcılar & Yetkiler (RBAC) [S4 §24]
- 5 rol: **Yönetici** (tam) · **Editör** (tam) · **Muhabir** (sadece Haber) · **Yazar** (sadece Makale) · **Üye** (panel yetkisi yok)
- Form: profil resmi, ad soyad, e-posta (canlı kullanım kontrolü), şifre (≥8, güç göstergesi, "Şifre Değiştir"), sosyal linkler (FB/TW/IG/LN/WA/TG/YT), yetki, **2FA** anahtarı + "2FA Sıfırla"
- **Yetki özelleştirme**: 12 içerik modülü + Diğer (Botlar/Yorumlar/Reklamlar/Kullanıcılar/Sayfalar/Araçlar) × {Tümü, Ekle, Düzenle, Sil}; kilitli ana yönetici değiştirilemez
- Yönetici filtreleri; CSV export

## 15. Dosya Yöneticisi [S4 §25]
- Upload: jpg/jpeg/png/gif/webp/pdf/zip/rar/mp4; **≤50 MB, eşzamanlı ≤5**; filtre (Görsel/Video/Doküman/Dosya); yıldızlı favori
- **Resim Düzenle**: 16:9 kırpma, döndürme, yatay/dikey çevir, "Orijinaliyle değiştir"
- **İnternette Ara (Google görsel)**: kelime ile, **Görsel Boyutu / En-Boy Oranı (16:9 / 9:16 / Kare)** filtreleri, **"Sadece Haber Siteleri"** kutusu, Kaydet; klasörler tarihe göre otomatik (elle klasör yok); Instagram desteklenmez

## 16. Reklamlar [S4 §20-21]
- **5 reklam tipi**: Görsel (URL + rel nofollow/sponsored/ugc) · Kod (HTML/JS/Ads panosu) · Slayt (geçiş süresi) · **Splash (pop-up)** (geri sayım + "kaç saat sonra tekrar") · **Sticky (sabit)** (dakika)
- **Reklam Alanları**: konum checkbox'lı (header/sidebar/içerik arası vd), salt-okunur liste; önizleme "Aç/Kapat" (etiketlerle alan gösterimi)
- Kategori opsiyonel; **başlangıç/çıkış tarihi** (boş = süresiz); Aktif anahtarı; gösterim/tıklama istatistiği yok (referansta)
- **Kod Oluşturucu**: başlık/görsel/URL/boyut/hedef/alt metin/rel → `<a><img>` kodu + canlı önizleme

## 17. Kategoriler & Sayfalar [S4 §17-19]
- Kategori: **düz yapı (alt kategori YOK)**, ad (≤255, sayaç), Özel URL + açılış hedefi, görünürlük (Header/sol menü/Footer), Yayın durumu; slug otomatik; drag sıra; "Bağlı İçerik" sayısı; çöp 30 gün
- Sayfa: Genel/Kurumsal; başlık+canlı slug önizleme (döngü=yeniden üret/elle düzenle), Özel URL, spot, detay, yayın durumu/menü/footer; **kurumsal + kilitli sayfalar silinemez**

## 18. Ayarlar — 7 sekme [S4 §26]
- **Genel**: yayın dili (**24 dil**), site başlığı/slogan, iletişim/sosyal linkler, **Bakım Modu** (süreli/süresiz + özel mesaj), CDN adresi
- **Görünüm**: site/favicon/footer logo; site/içerik/başlık font+boyut; header/ana-manşet/üst-manşet/beşli manşet/kategori/görsel-video galeri gösterim tipleri; ayrı site+footer renk paletleri
- **Modüller**: anasayfa blok sırası **drag + otomatik kaydet**; görünüm (Hepsi/Mobil/Masaüstü); aktif; ⚙ modül-özel seçenekler
  - Modüller listesi: Ana Manşet, Üst Manşet, Devmanşet(5'li), Son Dakika, Trend, Kategori bazlı, Orta Blok, Tab Menü (≤5 kat), İmsakiye, Piyasalar, Hava Durumu, Röportaj, Yazarlar, Vefat, Biyografi, Rehber, İlan, Resmi İlan, Sinemalar, Anket, Canlı Yayın (özel pencere), Seçim Sayfası (özel pencere), BİK Resmi İlan (widget kod)
- **İzinler**: 10 içerik türü × 9 anahtar: **Okunma, Paylaşım, Yorumlar, Sonsuz Scroll, Editör, Gözden Kaçmasın, İlginizi Çekebilir, Trend, Son Eklenenler** (otomatik kaydet)
- **Varsayılan Manşetler**: AM/ÜM/DM/SD/FH/HK anahtarları (yeni içerikte ön işaretli); "Google'da Takip Et"; çerez bildirimi metinleri
- **Bağlantılar**: arama motoru doğrulama/izleme kodları, AdSense, robots.txt, web push anahtarları, **Google servis hesabı**, Google Haberler bağlantısı; **Servis Konumları (şehir/ilçe)** → hava/namaz/eczane + AI \[sehir\]\[ilce\]
- **Yapay Zeka**: kilit (Üst düzey bilgi sahibiyim) → **API Anahtarı + Model seçimi**, prompt editörleri (Dev Manşet + 12 komut), sohbet şablon yönetimi
- **Loglar**: kim/zaman/modül/işlem filtresi; temizleme yetkisi
- **Güncelleme**: sürüm geçmişi

## 19. Dashboard (Anasayfa) Özeti [S4 §03][S2]
- **Keşfet**: sayılar (Haber/Makale/Galeri/Video), çöp adetleri, onay bekleyen yorum, okunmamış mesaj; kartlar tıklanınca listeye; **Hızlı Ekle** (Haber/Makale/AI sohbet)
- **Gerçek Zamanlı GA**: aktif kullanıcı, giren/çıkan (halka), en çok okunan sayfa, ülke; **120 sn yenileme**
- **Gelişmiş Raporlama**: 7/15/30 gün; aktif kullanıcı/oturum/PV/bounce/süre/sayfa-oturum (7 metrik, mini grafik + % değişim rozeti); günlük trafik + cihaz dağılımı grafikleri
- **Reklam Geliri (AdSense) kartı**: Kazanç ₺, Gösterim, Tıklama, CTR, RPM, Tıklama başı kazanç
- **Hava Durumu** kutusu (şehir; Servis Konumu'na bağlı, anlık şehir değiştirme)
- **Google Trends** kutusu (⋮ → mevcut haberler / Google'da Ara); **otomatik çeviri/üretim YOK**
- **Ulusal Gündem** (kaynak gruplu kartlar → orijinal sayfa)
- **Trend/Gündem/Etkinlik → sağ tık menüsü**: Yapay Zekaya Sor · Haber Başlığı Öner (5) · Haber Yaz (SEO: başlık+spot+≥3 paragraf) · **Araştır & Haber Yaz** (web araştırmalı)
- **Yaklaşan Etkinlikler** şeridi ("Bugün:"/"Yarın:"/"X gün sonra:")

## 20. Frontend Yapısı (demo: medyamugla.com) [S5][S3]
**URL desenleri:** `/`, `/kategori/{slug}`, `/haber-slug-{id}`, `/fotograflar`, `/videolar`, `/roportajlar`, `/yazarlar`, `/gazete-arsiv`, `/biyografiler`, `/vefat-edenler`, `/rehber`, `/ilanlar`, `/resmi-ilanlar`, `/anketler`, `/yerel-haberler`, `/son-dakika`, `/hava-durumu`, `/namaz-vakitleri`, `/trafik`, `/sinemalar`, `/gazete-mansetleri?ac={n}`, `/astroloji?burc=0..11`, `/nobetci-eczaneler`, `/lig-fikstur`, `/futbol/super-lig-puan-durumu-ve-fikstur`, `/arsiv`, `/rss-baglantilari`, `/sayfa/{slug}`, `/gazeteler/thumb/{gazete}`, `/assets/images/site/futbol/super-lig/*.png`

**Anasayfa modülleri:** header nav (kategoriler) · hava durumu (81 il) · son dakika kayan şerit · **piyasa bantı (BİST/DOLAR/EURO/ALTIN, canlı %)** · anamanşet/üst manşet/devmanşet · kategori blokları · Trend Haberler · Son Eklenenler (saatli) · foto galeri · E-Gazete · **Süper Lig puan durumu (takım logo'lu)** · **BİK ilan → Gazete Manşetleri (~30 gazete, thumb)** · burçlar (12) · sinemalar (süre+tür, fragmanlı) · yerel haberler · footer (künye, kurumsal, sosyal, iletişim, RSS) · "Haber Yazılımı AI v4.13.0" imzası

**Teknik notlar [S5/S3]:** statik CDN (`cdn.webaksiyon.com/uploads/{siteid}/...`), AutoWebP thumb `_large` varyantları, Cloudflare edge cache (`s-maxage=60`), muzik/metin fotoğraflar, `__cf_email__` koruması, PWA + service worker, koyu mod.

## 21. Servis Sayfaları (veri kaynakları) [S1/S3/S5]
- **Hava Durumu** 81 il (dış api) · **Namaz vakitleri** (il; iftara kalan süre seçeneği) · **Nöbetçi eczane** (il/ilçe) · **Trafik** (İstanbul vb.) · **Sinemalar** (film listesi: süre+tür+fragman) · **Astroloji** (12 burç, günlük/haftalık) · **Döviz/Altın/Borsa/Kripto** (canlı piyasa + otomatik çevirici sayfa) · **Puan durumu & Fikstür** (ligler, takım logoları) · **Gazete Manşetleri** (~30 ulusal gazete thumb) · **E-Gazete** · **Resmi İlanlar (BİK)** · **Vefat** · **Anket** · **Arşiv** · **RSS** (alma/verme) · **Canlı haber akışı** (son dakika otomatik yenileme) · **Mahreç etiketleme+raporlama**

## 22. AI Kararı (theblock)
- **Varsayılan sağlayıcı: OmniRoute** (OpenAI-uyumlu chat/completions uç noktası)
- **"API Anahtarı Ekle" özelliği**: kullanıcı kendi anahtarını/provider'ını ekler (OpenAI, Anthropic Claude, Gemini, OmniRoute, özel OpenAI-uyumlu URL) → soğuk başlangıç, sıcak anahtar yönetimi
- Prompt şablonları + model seçimi ayardan; token bazlı faturalandırma notu kullanıcıya
- Referans ürün Claude tabanlıydı; biz mimariyi çoklu-provider'a açıyoruz

## EKSİK / AÇIK SORULAR
- (S2) "Ulusal Gündem" + "7-Hazırlanıyor" video altyazıları (kullanıcı verecek)
- Referans üründe "Hazırlanıyor" modülü + yol haritasında ~ modül sayısı
- Canlı Yayın & Seçim Sayfası widget detayları (özel pencere görseli gerekli)
- Google News sitemap içerik kuralları, Discover kriterleri — ayrıca incelenecek
- Abonelik/paywall net spesifikasyonu yok (referansta liste var)
- E-posta bülten yok gibi görünüyor — teyit
