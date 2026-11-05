# Pacely Fark Analizi ve Yol Haritası

**Sürüm:** 1.0 — 30 Ekim 2026
**Kapsam:** Pacely demosu ile `theblock` yönetim panelinin **kalıp düzeyinde**
karşılaştırması; eksik işlevsellik listesi; faz planı.
**İlişkili:** `03-spec-plan.md`, `09-eksikler-ve-yapilacaklar.md`,
`10-admin-mimari-ve-arayuz.md`, `docs/ILERLEME.md` (Faz 146–161)

---

## 1. Yöntem ve kritik uyarı

Pacely bir **fizyoterapi kliniği** uygulamasıdır (`/patients`, `/schedule`,
`/billing/claims`). `theblock` bir **haber CMS**'idir. Bu yüzden alan
özellikleri (hasta, randevu, sigorta talebi) karşılaştırma konusu **değildir**
ve kopyalanmamalıdır.

Karşılaştırılan şey **arayüz ve etkileşim kalıplarıdır**: bir panelin
veriyi bulma, tarama, düzenleme ve geri bildirim verme biçimi. Bunlar
alan bağımsızdır.

Kanıt toplama yöntemi:

1. Pacely'nin **15 rotası** headless Chrome ile açıldı, her rotada
   `getComputedStyle` + DOM envanteri çıkarıldı (`e2e/pacely-kaliplar.js`).
2. `theblock` panelinde aynı kalıplar **kaynak aramasıyla** arandı
   (`templkit/*.templ`, `static/*.js`, `templates/*.html`).
3. Her bulgu bir sayaç ya da dosya/satır referansıyla kanıtlandı.
   Kanıtsız kalem listeye girmedi.

Ölçüm çıktısı (Pacely, 15 rota):

```
her rotada:  ⌘K arama hapı · "New" birincil düğme · bildirim zili (3)
             · tema anahtarı · rol/avatar
/home:       7 bölüm, 25 KPI, 16 akış öğesi, 10 rozet
/patients:   tek tablo 118 satır, sütun başlıkları tıklanabilir filtre
             (Patient, Plan, Pain, Next visit), 6 filtre çipi sayaçlı
/outcomes:   15 grafik öğesi, 6 KPI sekmesi (delta'lı: "Visits798↓ 9.5%")
/settings:   Discard / Save changes — kirli-durum eylem çubuğu
/help:       SSS akordeonu + iletişim formu + talep listesi
```

---

## 2. Kalıp karşılaştırma tablosu

| # | Kalıp | Pacely | theblock | Durum |
|---|---|---|---|---|
| P1 | Komut paleti (⌘K) | Varlık arar: hasta, ziyaret, talep | Varlık arar (Faz 162) | ✅ |
| P2 | Palet içeriği | Gruplu, delta/rozet, son kullanılanlar | Gruplu, vurgu, son kullanılan (Faz 162) | ✅ |
| P3 | Boş durum | Ortak bileşen, ikon + CTA | `BosDurum` bileşeni, 10 ekran (Faz 163) | ✅ |
| P4 | Yükleme durumu | İskelet/shimmer | `IskeletTablo/Kart` + halka (Faz 163) | ✅ |
| P5 | Tablo sistemi | Ortak: araç çubuğu, sütun filtresi, sayaçlı çip | Yapışkan başlık + kaydırma (Faz 164); sütun filtre kaldı | ⚠️ Kısmi |
| P6 | Yapışkan tablo başlığı | Var | Var (Faz 164) | ✅ |
| P7 | Zaman serisi grafiği | 12 aylık seri + delta | `panoAlanChart` 7/14/30 gün (N-7 yanlış ölçümdü) | ✅ |
| P8 | Yardım merkezi | `/help`: SSS + form + talepler | `/admin/yardim` (Faz 165) | ✅ |
| P9 | Birleşik gelen kutusu | `/messages`: All/Unread/Starred + hızlı yanıt | `/admin/gelen-kutusu` salt okuma (Faz 167) | ✅ |
| P10 | Genel "Yeni" düğmesi | Üst barda bağlama duyarlı | Yok (her sayfada yerel) | ❌ Yok |
| P11 | Sütun düzeyi filtre | Her tabloda | Yalnız `/admin/icerikler` | ⚠️ Kısmi |
| P12 | Kirli-durum çubuğu | Discard / Save changes | Otomatik kaydet + beforeunload | ✅ Eşdeğer |
| P13 | Bildirim zili | Var (3 okunmamış) | Var (`notif-dd` + sayaç) | ✅ |
| P14 | Tema anahtarı | Var | Var (`data-theme-btn`) | ✅ |
| P15 | Yoğunluk / kayıtlı görünüm | Var | Var | ✅ |
| P16 | Toplu işlem çubuğu | — | Var (`topluBar`) | ✅ |
| P17 | KPI delta | Var | Var (`pn-delta`) | ✅ |
| P18 | Sekmeler + sayaç | Var | Var (`ws-tabs`) | ✅ |

**Sonuç:** temel kabuk tamam. Eksikler **keşif (discovery)**, **durum
iletişimi (empty/loading)** ve **tutarlılık** katmanlarında.

---

## 3. İhtiyaç listesi

Her kalem: kök neden → çözüm → kabul kriteri.

### N-1 · Komut paleti yalnız rota listesi (Kritik)

**Kanıt:** `static/app.js` içinde `ROUTES` dizisi 38 sabit rota. Sunucuda
`/admin/ara` yok; `grep -r 'cmdk' ` yalnız `app.js`, `pano.js`,
`partials.html` döndürüyor.

**Kök neden:** Palet, menü kısayolu olarak yazılmış; arama altyapısına
bağlanmamış. Kullanıcı "Kadıköy depremi" diye yazınca içerik değil,
"İçerikler" sayfası açılıyor.

**Çözüm:** `GET /admin/ara?q=` uç noktası; içerik, kategori, etiket,
medya, kullanıcı araması; gruplu sonuç; son kullanılanlar; eylem bölümü.

**Kabul:** 2+ karakterde 200 ms altında yanıt; boş sorguda son
kullanılanlar; `↓↑ Enter Esc`; sonuç yoksa boş durum; emoji yok.

### N-2 · Palet içinde emoji (Kritik, kolay)

**Kanıt:** `app.js`: `"⚡ Yeni haber oluştur"`, `"🔎 İçeriklerde ara"`.

**Kök neden:** Faz 161 emoji temizliği `templkit` ve `editor-src` ile
sınırlı kaldı; `app.js` taranmadı.

**Çözüm:** Emoji öneklerini kaldır, simgeyi `ikon` alanıyla ver.

**Kabul:** `app.js` ve `admin-ui.js` içinde emoji aralığında karakter yok
(testle kilitli).

### N-3 · Ortak boş durum bileşeni yok (Yüksek)

**Kanıt:** `grep 'Liste boş\|Henüz .* yok\|Kayıt yok' templkit/*.templ` → **31**
eşleşme; ortak bileşen 0.

**Kök neden:** Her ekran boş durumu kendi cümlesiyle yazmış. Bazısı CTA
veriyor ("Yukarıdan Yeni Sayı ile oluşturun"), bazısı yalnız "Liste boş."
diyor. Ton ve yapı tutarsız; bir kısmı ikonsuz ve eylemsiz.

**Çözüm:** `templkit.BosDurum(ikon, baslik, aciklama, ctaEtiket, ctaHref)`.

**Kabul:** En az 6 liste ekranı bileşene geçer; her boş durumda ikon +
tek cümle + (varsa) CTA; `role="status"` ile ekran okuyucuya duyurulur.

### N-4 · İskelet yükleme yok (Yüksek)

**Kanıt:** `static/*.css` içinde yalnız `app.css:228` "Feed iskelet";
panelde 0. 11 templ dosyasında düz `Yükleniyor…` metni.

**Kök neden:** HTMX değişiminde hedef boşalıp tek satır metin geliyor;
tablo yüksekliği ~600 px'den ~20 px'e düşüp sayfa zıplıyor (CLS).

**Çözüm:** `templkit.IskeletTablo(satir, sutun)` + `.iskelet` CSS
(shimmer, `prefers-reduced-motion` altında statik).

**Kabul:** İçerik listesi, medya, yorumlar iskelet kullanır; geçiş
sırasında yükseklik korunur (ölçüm: |Δh| < 40 px).

### N-5 · Tablo sistemi dağınık, sayfalama eksik (Yüksek)

**Kanıt:** 17 templ dosyasında `<table>`; 7'sinde sayfalama. 10 tablo
sınırsız büyüyebilir.

**Kök neden:** Tablo her sayfada yeniden yazılmış; ortak sarmalayıcı yok.

**Çözüm:** `.table-wrap` sarmalayıcı + yapışkan `thead` + satır sayısı
altlığı; sayfalaması olmayan listelere `LIMIT` + "ilk N / toplam M".

**Kabul:** Yapışkan başlık 17 tabloda; sınırsız liste kalmaz.

### N-6 · Yapışkan tablo başlığı yok (Orta)

**Kanıt:** `admin-pano.css` içinde `position: sticky` tablo başlığına
uygulanmıyor; yalnız yan menü ve üst çubukta var.

**Kök neden:** Uzun listede (118 satır) aşağı inince sütun anlamı kayboluyor.

**Çözüm:** `.table thead th { position: sticky; top: 0 }` + zemin/gölge.

### N-7 · ~~Zaman serisi grafiği yok~~ → **YANLIŞ ÖLÇÜM, DÜZELTİLDİ**

**İlk iddia (hatalı):** "Pano'da yalnız sparkline var; eksenli seri grafiği
yok."

**Gerçek durum (30 Ekim 2026, kodu okuyunca):** `httpapi/pano_svg.go` içinde
`panoAlanChart` **zaten var**: alan + çizgi, 7/14/30 gün aralığı seçici,
her veri noktasına `data-v`/`data-l` (JS fare balonu), `role="img"` +
`aria-label`. `panoSpark` KPI döşemeleri için ayrı mini grafik üretiyor.
Veri kaynağı `hit_daily` (günlük kova), yani gerçek zaman serisi.

**Neden yanlış ölçtüm:** Yalnız `raporlar.templ` içindeki "svg" kelimesini
aradım; grafik üreticisi templ dosyasında değil, Go tarafında (`template.HTML`
olarak dönen sunucu-taraflı SVG). Arama deseni eksikti.

**Ders:** "yok" demeden önce üretici fonksiyonu aramak gerekir
(`grep -rn 'polyline\|template.HTML.*svg'`). Bu, spec'e yazılmış bir kalemin
kodla çürütülmesinin örneği; kalem kapatıldı, iş açılmadı.

**Kalan fark (isteğe bağlı, düşük öncelik):** Raporlar ekranı Chart.js
(205 KB, vendor) kullanıyor; pano ise bağımlılıksız SVG. İki yaklaşım
birbirine yakınsatılabilir ama çalışan bir şeyi yeniden yazmak riskli
olduğu için **yapılmadı**.

### N-8 · Panel içi yardım merkezi yok (Orta)

**Kanıt:** Pacely `/help` (SSS + form + talepler). Bizde `yardim`
eşleşmeleri yalnız dış bağlantı.

**Kök neden:** Bilgi `docs/` içinde; editör panelde aramıyor.

**Çözüm:** `/admin/yardim`: klavye kısayolları, sık sorulanlar, ekran
başına ipucu, sürüm notları (ILERLEME'den özet).

### N-9 · Birleşik gelen kutusu yok (Orta)

**Kanıt:** `/admin/iletisim`, `/admin/geri-bildirim`, `/admin/ai-sohbet`
ayrı; ortak "okunmamış" görünümü yok.

**Çözüm:** Üç kaynağı tek listede toplayan "Gelen Kutusu" görünümü
(yıkıcı değil, salt okuma + yönlendirme).

### N-10 · Genel "Yeni" düğmesi yok (Düşük) — ✅ Faz 168

**Kanıt:** Pacely üst barda `New` (bağlama duyarlı). Bizde yeni içerik
düğmesi yalnız liste ekranında.

**Çözüm:** Üst barda `+ Yeni` menüsü: içerik, gazete sayısı, kategori,
medya, kullanıcı — mevcut rotalara bağlanır.

### N-11 · Sütun düzeyi filtre (Düşük) — ✅ Faz 168 (arama çubuğu)

**Kanıt:** Pacely `/patients` sütun başlıkları tıklanabilir filtre.
Bizde yalnız `/admin/icerikler` filtre çubuğu.

**Çözüm:** Kademeli: önce en çok kullanılan 3 tabloya (medya, yorumlar,
üyeler) filtre çubuğu.

**Sonuç:** Medyada arama zaten vardı. Yorumlara (yazan/metin/başlık) ve
üyelere (ad/e-posta/rol) Türkçe katlamalı arama eklendi. Sütun başlığına
tıklama yerine arama çubuğu seçildi: 200 satırlık listede başlık sıralaması
kayıp kişiyi buldurmuyordu.

---

## 4. Mimari boşluklar

Bunlar Pacely farkı değil, **kendi borcumuz**; faz planına alındı.

| # | Bulgu | Kanıt | Etki | Durum |
|---|---|---|---|---|
| A-1 | `UpdateContent` tamlık skorunu yeniden hesaplamıyor | Faz 138 notu | Düzenlenen içeriğin skoru bayat kalır | ✅ Faz 169 |
| A-2 | `go/cmd/tmpsession` kullanılmıyor | `ls go/cmd` | Ölü kod + yönetici oturumu üreten araç | ✅ Faz 169 (silindi) |
| A-3 | Grup açık/kapalı durumu yalnız `localStorage` | Faz 147 | Cihazlar arası taşınmaz | ✅ Faz 173 (`user_prefs`) |
| A-4 | `editor.js` 753 KB, her içerik formunda | `ls -la static/editor.js` | Mobilde yavaş ilk yükleme | ✅ Faz 158 (kapsam) + Faz 172 (gzip) |
| A-5 | Paylaşılan bileşen katmanı zayıf | `bilesenler.templ` demo amaçlı | Her ekran tablo/boş/yükleme tekrar yazıyor | ✅ Faz 163 (boş/yükleme) + Faz 174 (tablo) |
| A-6 | `--muted` metin rengi olarak kullanılıyordu | Erişilebilirlik taraması | Panelde görünmez yazı (kontrast ≈ 1.0), public'te tanımsız token | ✅ Faz 175 (`--muted-fg`) |
| A-7 | Küçük metinlerde kontrast 4.5 altı | Tarama: 6 çift (3.19–4.20) | Okunabilirlik | ✅ Faz 180 (türetilmiş tonlar; marka korundu) |
| A-8 | `.field > <label>` alanlarında `for`/`id` bağı yok | Tarama: 61 alan; 214 alan zaten sarmalanmış | Ekran okuyucu alanı adlandıramaz | ✅ Faz 176 (tarih) + Faz 177 (tamamı) |
| A-9 | Alanlar yalnızca `placeholder` ile adlandırılmış | Tarama: 1163 kontrol → 114 adsız | Ekran okuyucu alanı adlandıramaz; yazı girilince ipucu kaybolur | ✅ Faz 179 |
| A-10 | Manşet editörü `.me-*` sınıflarının CSS'i yok | Tarayıcı: 0 kural; kaplama 0 px | Tuval düzensiz, "Izgara" düğmesi ölü | ✅ Faz 178 |
| A-11 | İkon sprite'ı sürümsüz URL'de | CDN: 41 sembol (8 gün bayat), sunucu: 641 | İkonlar sessizce boş kutu | ✅ Faz 181 (`?v=<özet>` + testler) |
| A-12 | Sayfa Tasarımcısı public.css çakışması | Panel 171px, alanlar kırpılıyor, marka rengi kayması | Sayfa çok uzun, alanlar görünmüyor, koyu temada yazı okunmuyor | ✅ Faz 182 |
| A-13 | Seçilen metin rengi iki temaya da uygulanıyordu | Koyu temada kontrast 1.2; 23 ekranda 17 bulgu | Koyu temada metin okunmuyor | ✅ Faz 183 |
| A-14 | `.be-bubble` konumlanmamış ve gizli değil | Ölçüm: static, 1440×40, y=2249 | İçerik formunun altında tam genişlikte düğme şeridi | ✅ Faz 185 |
| A-15 | Medya ekranı düz form + sabit ızgara; sürükle-bırak ve ilerleme yok | Ölçüm: `auto-fill minmax(150px)`, yükleme yalnız form gönderimi | Görsel ağırlıklı bir kütüphanede dosya seçmek ve taramak zor | ✅ Faz 186 |
| A-16 | Medyada sayfalama var, yükleme alanları resim detayıyla tekrar ediyor | Ölçüm: 60'lık sayfalar; üstte 6 alan + dosya seç düğmesi | Yükleme akışı ile düzenleme akışı çakışıyor, tarama kesintili | ✅ Faz 187 |
| A-17 | Otomatik çekilen haberlerde `seo_title`/`seo_description` boş kalıyordu | Ölçüm: içerik 48439'da iki alan da boş; SERP önizlemesi hiç güncellenmiyordu | Google'da başlıksız/meta açıklamasız haber | ✅ Faz 188 |
| A-18 | 24 dil açık ama tek dilde içerik; `/en` boş manşet, hreflang yok, dil geçişi ana sayfaya atıyor | Ölçüm: EN manşet 0 (TR 150), canonical `/en`, hreflang yok | İngilizce site boş görünüyor; kopya dizinleme riski | ✅ Faz 189 |
| A-19 | İçerik düzenlemedeki "Dil / Çeviriler" paneli çalışmıyor, AI çeviri tetikleyicisi yok | Ölçüm: 24 satır, 23 "+" bağlantısı, tıklama **404** (rota POST, şablon GET); AI düğmesi 0 (KaynakID boş); senkron AI isteği 160 sn → HTTP 000 (WriteTimeout 60 sn, Cloudflare ~100 sn) | Editör çeviri yapamıyor; düğme "çalışmıyor" | ✅ Faz 190 |

A-1 ve A-2 küçük ve somut; A-5 bu belgenin ana gerekçesi.

---

## 5. Faz planı

| Faz | Ad | Kapsam | Öncelik | Durum |
|---|---|---|---|---|
| **162** | Komut paleti 2.0 | N-1, N-2 | Kritik | ✅ tamam |
| **163** | Boş durum + iskelet | N-3, N-4 | Yüksek | ✅ tamam |
| **164** | Tablo sistemi | N-5, N-6 | Yüksek | ✅ tamam |
| **165** | Yardım merkezi | N-8 | Orta | ✅ tamam |
| ~~166~~ | ~~Zaman serisi grafiği~~ | N-7 | — | ❌ kalem geçersiz (zaten var) |
| **167** | Gelen kutusu | N-9 | Orta | ✅ tamam |
| **168** | Genel "Yeni" + arama filtreleri | N-10, N-11 | Düşük | ✅ tamam |
| **169** | Teknik borç kapanışı | A-1, A-2 | Düşük | ✅ (A-3 açık) |
| **170** | Panel genelinde emoji temizliği | — (tutarlılık) | Düşük | ✅ tamam |
| **171** | `data-confirm` çakışması (silme 405) | — (hata) | Kritik | ✅ tamam |
| **172** | JS gzip (`text/javascript`) | A-4 | Yüksek | ✅ tamam |
| **173** | Arayüz tercihleri sunucuda | A-3 | Orta | ✅ tamam |
| **174** | Tablo boş durumu tek bileşende | A-5 | Orta | ✅ tamam |
| **175** | `--muted` metin rengi düzeltmesi | A-6 (yeni) | Yüksek | ✅ tamam |
| **176** | Tarih alanı etiketleri + flex genişliği | A-8 (yeni) | Orta | ✅ tamam |
| **177** | Form etiket bağı (for/id) + çift id | A-8 | Yüksek | ✅ tamam |
| **178** | Manşet editörü tuval CSS'i (ızgara ölüydü) | A-10 (yeni) | Orta | ✅ tamam |
| **179** | Placeholder etiket yerine geçmez | A-9 | Yüksek | ✅ tamam |
| **180** | Küçük metin kontrastı (marka korunarak) | A-7 | Yüksek | ✅ tamam |
| **181** | İkon sprite önbelleği (boş ikonlar) | A-11 (yeni) | Kritik | ✅ tamam |
| **182** | Sayfa Tasarımcısı yerleşimi + koyu tema | A-12 (yeni) | Yüksek | ✅ tamam |
| **183** | Koyu tema metin rengi (tüm panel) | A-13 (yeni) | Kritik | ✅ tamam |
| **184** | Gazete şablon galerisi (tablo → kart) | — (kullanıcı isteği) | Orta | ✅ tamam |
| **185** | Seçim balonu konumlandırma (alt çubuk) | A-14 (yeni) | Yüksek | ✅ tamam |
| **186** | Medya galerisi: kart, masonry, sürükleme, yükleme halkası | A-15 (yeni) | Orta | ✅ tamam |
| **187** | Medya: sonsuz kaydırma, kategori çipleri, sade üst alan | A-16 (yeni) | Yüksek | ✅ tamam |
| **188** | Otomatik haberlerde AI ile SEO dolumu | A-17 (yeni) | Yüksek | ✅ tamam |
| **189** | Çok dilli ön yüz: İngilizce içerik + canonical/hreflang | A-18 (yeni) | Kritik | ✅ tamam |

Teslim edilen fazların ayrıntısı `docs/ILERLEME.md` Faz 162–170.

### Faz 162 — Komut paleti 2.0 · Definition of Done

1. `GET /admin/ara?q=` gruplu JSON döner: içerik, kategori, etiket,
   medya, kullanıcı. Site kapsamlı (`site_id`), en çok 5/grup.
2. Palet: gruplu başlık, ikon, alt satır (kategori · durum · tarih),
   vurgulanan eşleşme; boş sorguda **son kullanılanlar** (localStorage,
   en çok 6); `↓↑` gruplar arası geçer, `Enter` açar, `Esc` kapatır.
3. Emoji yok; simge yerel sprite'tan.
4. Sorgu 200 ms altında (indeksli ILIKE; `title` üzerinde trigram
   indeksi varsa kullanılır, yoksa `ILIKE` ile sınırlı LIMIT).
5. Testler: uç nokta grupları döner; boş sorgu 400/boş grup; XSS
   kaçışı (arama terimi HTML'e basılmaz); emoji yok.

### Faz 163 — Boş durum + iskelet · DoD

1. `BosDurum` ve `IskeletTablo` templ bileşenleri.
2. En az 6 liste ekranı `BosDurum` kullanır.
3. 3 ekran `IskeletTablo` kullanır; `aria-busy` ve `role="status"`.
4. `prefers-reduced-motion` altında shimmer durur.
5. Testler: bileşen çıktısı; emoji yok; CSS sınıfları tanımlı.

### Faz 164 — Tablo sistemi · DoD

1. `.table-wrap` + yapışkan `thead` (zemin, alt çizgi, z-index).
2. Uzun listelerde yatay kaydırma sarmalayıcı içinde kalır (taşma yok).
3. Satır sayısı altlığı: "N / M satır".
4. Testler: 17 tablonun tamamı sarmalayıcıda; sticky kuralı sabit.

---

## 6. Bilinçli olarak yapılmayacaklar

- **React'e geçiş.** Ölçüm tersini söylüyor (Faz 150: ilk yükleme
  ~300 ms vs Pacely ~4.164 ms). Kalıplar HTMX ile de uygulanabilir.
- **Alan özellikleri** (randevu, sigorta, hasta). Farklı ürün.
- **Pacely kodu/assets kopyalamak.** Lisans yok; yalnız ölçülmüş değer
  ve kalıp uyarlanır.
