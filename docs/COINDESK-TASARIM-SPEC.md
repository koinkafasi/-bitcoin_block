# CoinDesk tasarım spesifikasyonu — ölçüm (6 Kasım 2026)

Bu belge **görünümü** tanımlar: palet, tipografi, boşluk, ızgara, bileşen
anatomisi ve sayfa bazında geometri. Yapı/URL/içerik tipleri ayrı belgede:
`COINDESK-SPEC.md`. İkisi birbirini tamamlar, çakışmaz.

## Ölçüm künyesi

Her sayı **gerçek Chrome'da hesaplanmış stilden** okundu
(`getComputedStyle`), kaynaktan tahmin edilmedi.

| | |
|---|---|
| Tarih | 6 Kasım 2026 |
| Viewport | 1920×871, DPR 1 |
| Tema | **ikisi de** (`html.light` ve `html.dark`) |
| Ölçülen sayfalar | `/` · `/tech/2026/11/05/<slug>` · `/markets` · `/price` · `/price/bitcoin` |
| Yöntem | 822 farklı sınıf, ~600 metin düğümü tarandı; sınıf adı → hesaplanmış değer eşlemesi |

**Ölçülmeyenler** (bilerek, dürüstlük için işaretli):

- `/videos`, `/podcasts`, `/research`, `/newsletters`, `/tag/*`, `/index/*`
  sayfalarının iç geometrisi
- Mobil/tablet **render**'ı: tarayıcı penceresi yeniden boyutlandırılamadı
  (araç başarı döndü, `innerWidth` 1920'de kaldı). Duyarlı davranış bunun
  yerine **sınıf adlarından ve `sizes` özniteliğinden** çıkarıldı — bunlar
  da ölçülmüş kanıt, aşağıda kaynağıyla veriliyor.
- Hover/focus durumları
- Ana stylesheet `cssRules` ile okunamadı (yalnız 92 kural erişilebilir);
  bu yüzden "CSS'te şöyle yazıyor" değil "tarayıcı şunu hesapladı" deniyor.

---

## 0. ÖNEMLİ — mevcut spec'teki yanlış değerler

`COINDESK-SPEC.md` §2'deki token listesi **bugünkü siteyle uyuşmuyor**.
O liste eski bir build'den çıkarılmış. Bugün CoinDesk'te:

> **DURUM (6 Kasım 2026, akşam): bu bölümün tamamı UYGULANDI.**
> `cd.css` artık aşağıdaki ölçülen değerleri taşıyor; eski yanlış değerler
> `cd_tasarim_test.go` içindeki `TestCDEskiYanlisDegerlerYok` kapısıyla
> geri gelmeye karşı korunuyor. Canlıda `getComputedStyle` ile doğrulandı:
> kapsayıcı 1512, yan boşluk 24, köşe 4 px, metin #262626, ayırıcı #D9D9D9,
> yükseliş #218410, düşüş #BB020A, sticky toplam 140 px, makale sütunu 680,
> gövde 22/30, etiketler düz bağlantı.

**CSS özel özelliği (custom property) HİÇ YOK.** Tailwind, değerleri
derleyerek basıyor. Semantik sınıf adları (`bg-surface-subtle`,
`text-subtle`) var ama arkasında `--degisken` yok. Yani §2'deki
`--bg`, `--fg`, `--muted` … adları CoinDesk'te mevcut değil.

Düzeltilmesi gereken somut değerler:

| Mevcut spec §2 | Gerçek (ölçüm) | Not |
|---|---|---|
| `--font "proxima-nova"` | **`"Neue Haas", sans-serif`** | Font tamamen değişmiş |
| `--link #4d90fe` (mavi) | **bağlantılar metin rengi**, altı çizili değil | Sitede mavi bağlantı yok; 317 bağlantıda `none`, 6'sında `underline` |
| `--up #68b631` | **#218410** (açık) / **#18E589** (koyu) | |
| `--down #ff0062` | **#BB020A** (açık) / **#FF454D** (koyu) | |
| `--warn #ffd200` | **marka rengi** #FFD200 | Uyarı değil; `bg-surface-brand-fixed` ve `text-brand-fixed` |
| `--border #e9e9e9` | **#D9D9D9** | #E9E9E9 başka şey: `surface-disable` |
| `--muted #696969` | **#676767** | |
| "gölge yok" | **gölge var**: `0 0 10px rgba(0,0,0,0.4)` (21 öğe) | Kartlarda yok, bindirme/açılır katmanlarda var |
| "yarıçap 0–4px" | **4px baskın** (120×), 8px (72×), pill 9999px (66×) | 2px kullanılmıyor |
| "açık tema varsayılan" | **kullanıcı anahtarı var** (`html.light` / `html.dark`) | `prefers-color-scheme` ilk değeri veriyor, anahtar eziyor |

---

## 1. Kırılma noktaları

CSS'ten okunan medya sorgusu merdiveni (tamamı):

```
min-width:  320  379  441  640  728  767  768  1024  1080  1280  1440  1728  1920  2048
max-width:  1079  1439
```

Tasarımı yönetenler (sınıf öneklerinden): **768 (md)**, **1024 (lg)**,
**1440**. Diğerleri reklam yuvası ve ince ayar.

Davranış (sınıf adlarından — ölçülmüş kanıt):

| Sınıf | Anlam |
|---|---|
| `px-4 md:px-6` | kapsayıcı yan boşluğu 16px → **768'den** 24px |
| `flex-row gap-4 md:flex-col` | kart mobilde **yatay** (görsel solda), 768'den **dikey** |
| `hidden h-full lg:block` | mega gezinme **1024'ün altında gizli** |
| `aria-label="Open menu"` | 1024 altı için hamburger |

---

## 2. Kapsayıcı ve ızgara

```
Kapsayıcı:  div.mx-auto.max-w-[var(--page-max,1512px)].px-4.md:px-6
```

| | Değer |
|---|---|
| max-width | **1512px** (`--page-max`) |
| yan boşluk | 16px (<768) · **24px** (≥768) |
| içerik genişliği | **1464px** (1512 − 48) |
| ortalama | `margin-inline: auto` |

**Ana sayfa ana ızgarası:** 3 sütun, `gap: 24px 80px`
(satır arası 24, sütun arası **80px** — geniş bir boşluk, dikkat).

**Bölüm sayfası:** **12 sütunlu** ızgara, sütun 100.656px, gap 16px
→ 12×100.656 + 11×16 = **1384px** içerik. Ana sayfadan farklı (1464).

**Alt bilgi:** 5 sütun × 236.797px, gap 24px → **1280px** içerik.

Ölçülen diğer ızgaralar: 4 sütun gap 24px (240.5px); 2 sütun gap 16px (en sık).

---

## 3. Boşluk ölçeği

Kullanım sıklığıyla (flex/grid `gap`):

```
4px (146×)  8px (133×)  16px (70×)  12px (33×)  6px (13×)
20px (11×)  24px (9×)   2px (7×)    32px (6×)   10px (2×)
```

Yani ölçek: **2 · 4 · 6 · 8 · 10 · 12 · 16 · 20 · 24 · 32**
(Tailwind 0.5/1/1.5/2/2.5/3/4/5/6/8). Kart içi 16px, satır içi 4–8px.

Alt bilgi dolgusu: `48px 32px`.

---

## 4. Tipografi

**Font:** `"Neue Haas", sans-serif` (429 düğüm). İkincil: `Arial, Helvetica,
sans-serif` (15 düğüm, muhtemelen gömülü/reklam).

**Ağırlıklar:** 400 (364×) · 500 (164×) · 700 (19×) · 600 (2×)
→ Sistem **400 ve 500** üzerine kurulu. 700 yalnız vurgu (tablo başlığı,
bölüm H1, gezinme). 600 kullanılmıyor.

### Ölçek (açık tema, 1920px)

| Boyut/satır | Ağırlık | Harf aralığı | Nerede | Sıklık |
|---|---|---|---|---|
| 14 / 22 | 400 | — | **gövde varsayılanı**, meta, yazar | 86× |
| 12 / 16 | 400 | **0.4px** | küçük meta, figcaption | 59× |
| 14 / 21 | 400 | — | liste metni | 42× |
| 12 / 22 | 400 | — | sıkışık meta | 40× |
| 12 / 18 | 400 | — | tablo hücresi | 38× |
| 14 / 20 | 500 | — | kart alt başlığı | 28× |
| 14 / 21 | 500 | — | vurgulu satır | 20× |
| **20 / 26** | 500 | — | **kart başlığı (orta)** | 20× |
| **18 / 26** | 500 | — | **kart başlığı** | 18× |
| 12 / 16 | 400 | 0.4px | **UPPERCASE** — kicker/etiket | 12× |
| 14 / 22 | 500 | — | capitalize bağlantı | 12× |
| 16 / 24 | 500 | — | modül başlığı (H2/H3) | 10× |
| 16 / 24 | 400 | — | makale deck'i, alt bilgi | 9× |
| 14 / 21 | 700 | — | güçlü vurgu | 9× |
| 18 / 24 | 500 | — | kart başlığı (küçük) | 7× |
| 11 / 16 | 400 | — | en küçük | 6× |

**Dikkat:** Modül başlıkları (`Prices`, `Videos`, `Today's Videos`) yalnız
**16px/24–32px w500**. Büyük bölüm başlığı değil — CoinDesk modül
başlıklarını küçük tutuyor, hiyerarşiyi kart başlıklarıyla kuruyor.

### Sayfa bazında başlıklar

| Sayfa | Öğe | Ölçüm |
|---|---|---|
| Makale | H1 | **36 / 44 w500** |
| Makale | gövde `<p>` | **22 / 30 w400**, `margin-bottom: 32px` |
| Makale | ara başlık (H2/H3) | **28 / 36 w500** |
| Makale | blockquote | 22 / 30 w400 (gövdeyle aynı) |
| Makale | figcaption | 12 / 16 w400 **ls 0.4px**, `text-subtle` |
| Makale | yazar | 14 / 22 w400 |
| Makale | tarih `<time>` | 14 / 22 w400, `text-subtle` |
| Bölüm | H1 | **36 / 44 w700** ← makaleden farklı ağırlık |
| Bölüm | bölüm başlığı | 28 / 36 w700 |
| Fiyat listesi | H1 | **24 / 36 w500** |
| Coin detay | H1 | **20 / 30 w500** (küçük!) |
| Coin detay | büyük fiyat | **36 / 38 w500** |

---

## 5. Palet (semantik ad → gerçek değer)

Adlar CoinDesk'in kendi Tailwind sınıflarından; değerler ölçüldü.

### Açık tema (`html.light`)

```
bg-surface-default        #FFFFFF    gövde zemini
bg-surface-subtle         #F4F4F4    kart/blok zemini (en sık, 52×)
bg-surface-action         #D9D9D9    etkileşim zemini
bg-surface-disable        #E9E9E9    devre dışı
bg-surface-brand-fixed    #FFD200    MARKA sarısı (7×)
bg-fill-strong            #FFFFFF
bg-fill-strong-inverted   #000000

text-default              #262626    ana metin (284×)
text-subtle               #676767    ikincil (144×)
text-strong               #000000    vurgulu (40×)
text-pos                  #218410    fiyat yükseliş
text-neg                  #BB020A    fiyat düşüş
text-brand-fixed          #FFD200
text-white-fixed          #FFFFFF
text-black-fixed          #000000

border-default            #D9D9D9    ayırıcı (85×)
border-strong             #A6A6A6    güçlü ayırıcı (18×)
```

### Koyu tema (`html.dark`)

```
bg-surface-default        #161616
bg-surface-subtle         #262626
bg-surface-action         #262626
bg-surface-disable        #0F0F0F
bg-surface-brand-fixed    #FFD200    ← marka DEĞİŞMEZ ("fixed")
bg-fill-strong            #000000
bg-fill-strong-inverted   #FFFFFF

text-default              #D4D4D4
text-subtle               #949494
text-strong               #ECECEC
text-pos                  #18E589
text-neg                  #FF454D

border-default            #262626
border-strong             #585858
```

**`-fixed` son eki bir sözleşme:** temaya göre değişmeyen renkler
(marka sarısı, mutlak siyah/beyaz). Kendi jeton adlandırmamızda bunu
kopyalamak değerli — hangi rengin tema ile döneceği adından okunuyor.

**Tema anahtarı:** `aria-label="Switch to dark mode"` / `"Switch to light
mode"` düğmesi; `<html>` üzerinde `light`/`dark` sınıfını değiştiriyor.
İlk değer `prefers-color-scheme`'den geliyor, anahtar onu eziyor.

---

## 6. Köşe, gölge, çizgi

```
Yarıçap:  4px (120×)  ·  8px (72×)  ·  9999px/pill (66×)  ·  1px (19×)  ·  6px (4×)
Gölge:    0 0 10px rgba(0,0,0,0.4)  (21×)   — yalnız bindirme/açılır katman
          0 0 10px rgba(0,0,0,0.5)  (1×)
```

**Kart ve blok ayrımı çizgiyle:** `section.flex.flex-col.border-t.border-solid`
— modüller **üst çizgiyle** ayrılıyor, gölge veya kutu yok. "Gölge yok"
kuralı **kartlar için doğru**, sayfa genelinde değil.

---

## 7. Global bileşenler

### 7.1 Başlık (header)

```
#main-navigation   1905×73   position: STICKY   bg: #000000 (koyu) 
logo               169×32
gezinme            7 öğe, 16/24 w400, hidden lg:block
```

Gezinme öğeleri (ölçülen sıra): **News · Video · Prices · Research ·
Events · Data & Indices · Sponsored**

> bitcosar'da Events ve Data & Indices **bilerek yok** (veri/metodoloji
> olmadığı için; bkz. commit `8ecf27f`). Bu bilinçli bir sapma, hata değil.

### 7.2 Fiyat şeridi (ticker)

```
1905×67   position: STICKY   bg: surface-default   border-bottom: 1px
```

İki sekme: `Markets Live` | `Crypto Stocks`. Ardından CD20, BTC, ETH, XRP…

**Başlık + şerit ikisi de sticky → toplam sabit yükseklik 140px.**
Çıpa kaydırmasında (`scroll-margin-top`) bu hesaba katılmalı.

Erişilebilirlik ayrıntısı: yüzde değeri görsel olarak `6.33%` yazarken
ekran okuyucuya `Down 6.33 percent` veriliyor — ayrı metin düğümü.

### 7.3 Alt bilgi (footer)

```
1905×966   padding: 48px 32px   bg: surface-default
ızgara: 5 × 236.797px, gap 24px → 1280px
bağlantı: 16/24 w400
```

Sütun 1 "About" 46 bağlantı, sütun 2 "Cryptocurrencies" 25 bağlantı
→ sütunlar **eşit uzunlukta değil**, içerik kadar.

---

## 8. Kart anatomisi (en kritik bileşen)

Ölçülen DOM:

```html
<article class="flex flex-row gap-4 md:flex-col">   <!-- mobil yatay, 768+ dikey -->
  <div class="relative aspect-video overflow-hidden rounded">
    <img ... loading="lazy" sizes="…" srcset="…">   <!-- object-fit: cover -->
  </div>
  <!-- kicker · başlık · spot · meta -->
</article>
```

| | Değer |
|---|---|
| En-boy oranı | **16/9 sabit** (`aspect-video`) — ölçülen tüm görseller 1.77–1.79 |
| Görsel yerleşimi | `object-fit: cover` |
| Görsel yarıçapı | `rounded` = **4px** (sarmalayıcıda, `<img>`'de 0) |
| Kart içi boşluk | **gap 16px** (`gap-4`) |
| Taşma | `overflow-hidden` |
| Yükleme | `loading="lazy"` |

### Ölçülen görsel yuvaları

| Genişlik×yükseklik | Nerede |
|---|---|
| **689×388** | bölüm sayfası lead |
| **608×342** | ana sayfa hero lead |
| **334×188** | — |
| **327×185** | bölüm ızgara kartı |
| **290×163** | ana sayfa yan ray kartı |
| **241×135** | bölüm küçük kart |
| **175×128** | 1.37 oran (istisna — kare'ye yakın yuva) |
| **152×86** | liste satırı küçük görseli |
| **48×48** | avatar (1.00) |

Hepsi 16:9, iki istisna: 175×128 (1.37) ve 48×48 avatar.

---

## 9. Görsel hattı — `sizes`/`srcset` tekniği

**Bu bölüm bizim için en değerli bulgu.** CoinDesk'in hero görselinden
alınan **tam** `sizes` dizesi:

```
(min-width: 1440px) and (min-resolution: 3dppx) calc(608px * 0.433),
(min-width:  768px) and (min-resolution: 3dppx) calc(60vw  * 0.433),
                         (min-resolution: 3dppx) calc(100vw * 0.583),
(min-width: 1440px) and (min-resolution: 2dppx) calc(608px * 0.65),
(min-width:  768px) and (min-resolution: 2dppx) calc(60vw  * 0.65),
                         (min-resolution: 2dppx) calc(100vw * 0.875),
(min-width: 1440px) 608px,
(min-width:  768px) 60vw,
                    100vw
```

Yapı: **3 DPR katmanı × 3 genişlik katmanı**.

| DPR | Masaüstü/tablet çarpanı | Mobil çarpanı | Etkin yoğunluk |
|---|---|---|---|
| 3dppx | 0.433 | 0.583 | 3 × 0.433 ≈ **1.3x** / 3 × 0.583 ≈ **1.75x** |
| 2dppx | 0.65 | 0.875 | 2 × 0.65 ≈ **1.3x** / 2 × 0.875 = **1.75x** |
| 1x | 1.0 | 1.0 | 1.0x |

**Yani etkin piksel yoğunluğunu masaüstünde ~1.3x, mobilde ~1.75x'te
sabitliyorlar.** Retina ekranda tam 2x/3x indirmek yerine bilinçli olarak
daha az indiriyorlar; gözle fark edilmiyor, bayt yarıya iniyor.

Genişlik katmanları: `1440px+ → 608px sabit`, `768–1440 → 60vw`,
`<768 → 100vw`.

**srcset basamakları** (Next.js varsayılanları):
`384 448 640 750 828 896 1024 1080 1200` (9–10 giriş)

**Hat:** `coindesk.com/_next/image?url=<sanity-cdn-url>&w=…&q=…`
→ Next.js görsel iyileştirici, Sanity CDN'in önünde.

> **bitcosar farkı:** `/kg/{hash}/{genislik}` ucumuz düz DPR-2 kapsamı
> kullanıyor (yuva × 2). CoinDesk'in çarpan tekniği daha iyi: aynı görsel
> kalitesinde belirgin daha az bayt. Uygulanması kolay — `cd_gorsel_yuva.go`
> içindeki `sizes` üretiminde `calc(<px> * 0.65)` dalları eklenir, basamak
> listesi değişmez. Bkz. `DENETIM-2026-11-05.md`.

---

## 10. Sayfa bazında geometri

### 10.1 Ana sayfa

Kapsayıcı 1512 / içerik 1464. Ana ızgara **3 sütun, gap 24px 80px**.

Modül sırası `COINDESK-SPEC.md` §6'da (23 modül) — o liste geçerli.
Ölçülen yerleşim bandları: hero bloğu 1464×1796, "Explore more from
CoinDesk" 1512×721, Opinion 1512×527, Most Read 1074×495.

Yan ray (`aside`) genişliği **290px**.

### 10.2 Makale sayfası

```
Sütun:    680px, YATAY ORTALANMIŞ (sol 597px @1920 → (1920-680)/2 ≈ 620)
Yan ray:  YOK — tek sütun
```

| Öğe | Ölçüm |
|---|---|
| H1 | 36/44 w500 |
| Deck (H1 altı) | 16/24 w400 |
| Gövde `<p>` | **22/30 w400, mb 32px** |
| Ara başlık | 28/36 w500 |
| Blockquote | 22/30 w400 — ayrı stil ölçülmedi |
| Yazar | 14/22 w400 |
| Tarih | `Nov 5, 2026, 8:30 p.m. EST` — 14/22, `text-subtle` |
| `<time datetime>` | `2026-11-06T01:30:00.000Z` (ISO, UTC) |
| figcaption | 12/16 w400 ls 0.4px |
| Etiketler | **düz bağlantı** 16/24 — pill değil, zemin/kenarlık/dolgu yok |
| Paylaşım | X (`x.com/intent/tweet`), LinkedIn, paylaşım diyaloğu |
| Breadcrumb | **bulunamadı** — mevcut spec §7 "Home / Markets" diyor; bugün yok |

Gövde 17 paragraf; metin görünen genişliği 636px (680 sütun içinde).

### 10.3 Bölüm sayfası (`/markets`)

```
İçerik:  1384px, 12 sütunlu ızgara, gap 16px
H1:      36/44 w700  ("Markets")
H1 altı açıklama:  YOK  (mevcut spec §8 "tek satır açıklama" diyor — yok)
Lead görseli:      689×388
Izgara kartları:   327×185 (5 adet) · 241×135 (4 adet)
Kart başlıkları:   H2 24/30 w500 · 20/26 w500 · 16/24 w500
Sayfalama:         "More stories" DÜĞMESİ — sonsuz kaydırma değil
```

Ek ızgara: 4 sütun × 240.5px, gap 24px.

### 10.4 Fiyat listesi (`/price`)

**Gerçek `<table>`** (ızgara değil).

```
Genişlik:         1857px      border-collapse: collapse   table-layout: auto
Sütun sayısı:     10
Satır sayısı:     50
Satır yüksekliği: 69px
th:  12/18 w700, padding 8px 1px
td:  12/18 w400, padding 8px
tr:  border-bottom 1px (border-default)
H1:  24/36 w500 — "Live Crypto Prices & Market…"
```

Sütunlar (ölçülen sıra):
`# · Asset · (Buy) · Price · Market Cap · 24h Volume · Direct 24h trading ·
24h trading vol · 24h% · Last 7 Days`

Son sütun **sparkline** (7 günlük mini grafik).
Yüzde hücresi: 12/18 **w500**, pos/neg renkleriyle.

### 10.5 Coin detay (`/price/bitcoin`)

```
H1:          20/30 w500 ("Bitcoin") — başlık küçük, fiyat büyük
Büyük fiyat: 36/38 w500 ("$75,813.08")
Grafik:      <canvas> 1385×500
Zaman sekmeleri: 1M · YTD · 1Y · All
```

**URL deseni farkı:** CoinDesk **`/price/bitcoin`** kullanıyor (tam ad).
bitcosar'da `/price/btc` (sembol). Mevcut spec §4 `/price/{coin}` diyor,
hangisi olduğunu söylemiyor. Karar gerek: tam ad SEO'da daha iyi
("bitcoin price" aramasıyla eşleşiyor), sembol kısa.

---

## 11. bitcosar ↔ CoinDesk fark tablosu

Bizim değerler `go/internal/webui/static/cd.css`'ten **okundu**, tahmin değil.

> **DURUM (6 Kasım 2026): tablo UYGULANDI.** Aşağıdaki "aksiyon" kolonundaki
> kalemlerin tamamı yapıldı; kalan üç bilinçli sapma §14'te gerekçesiyle
> yazılı. Doğrulama gerçek Chrome'da `getComputedStyle` ile ölçüldü.

| Konu | bitcosar (bugün) | CoinDesk (ölçüm) | Aksiyon |
|---|---|---|---|
| Font | `"Neue Haas", Inter, system-ui…` | **`"Neue Haas", sans-serif`** | ✅ lisans varsa Neue Haas, yoksa Inter/sistem; webfont GÖMÜLMEDİ (bütçe) |
| Kapsayıcı | `--cd-max: 1512px`, pad 16/24 → **1464** | **1512px**, pad 16/24 → **1464** | ✅ |
| Köşe yarıçapı | `--cd-radius: 4px` (+8, pill 9999) | **4px** (+8px, pill) | ✅ |
| Gölge | yalnız `--cd-shadow-overlay` (mega menü) | kart yok, **bindirmede var** | ✅ |
| Bağlantı rengi | metin rengi (`currentColor`), alt çizgi yok | **metin rengi**, altı çizili değil | ✅ |
| "Tümü →" bağlantısı | metin/ikincil renk | metin/ikincil renk | ✅ |
| Ana metin | `--cd-fg: #262626` | **#262626** | ✅ |
| İkincil metin | `--cd-muted: #676767` | **#676767** | ✅ |
| Ayırıcı | `--cd-border: #d9d9d9` | **#D9D9D9** | ✅ |
| Yükseliş | `--cd-up: #218410` / koyu #18E589 | **#218410** / koyu #18E589 | ✅ |
| Düşüş | `--cd-down: #bb020a` / koyu #FF454D | **#BB020A** / koyu #FF454D | ✅ |
| Marka | `--cd-brand: #ffd200` | **marka** #FFD200 | ✅ ad düzeltildi |
| Yüzeyler | default #FFF · subtle #F4F4F4 · action #D9D9D9 · disable #E9E9E9 | aynı | ✅ |
| Koyu tema | **kullanıcı anahtarı** + `html.light/.dark` | anahtar + sistem | ✅ |
| Makale sütunu | **680px** | **680px** | ✅ |
| Makale gövdesi | **22px / 30px, mb 32px** | **22px / 30px, mb 32px** | ✅ |
| Makale ara başlık | **28/36 w500** | **28px / 36px w500** | ✅ |
| Makale H1 | **36/44 w500** | 36/44 w500 | ✅ |
| Bölüm H1 | **36/44 w700** | 36/44 **w700** | ✅ |
| Modül başlığı | **16/24 w500, büyük harf YOK** | **16/24 w500** | ✅ |
| Dikey ritim | `--cd-section-gap: 32px` | boşluk ölçeği 2–32px | ✅ |
| Etiketler | **düz bağlantı** 16/24 | **düz bağlantı** 16/24 | ✅ |
| Sticky | **başlık + şerit** (140px) | header **+** ticker (140px) | ✅ |
| `sizes` | **çarpanlı** (0.433/0.65) | **çarpanlı** | ✅ |
| Sayfalama | **"Daha fazla haber" düğmesi** | "More stories" düğmesi | ✅ |
| `/price` URL | **`/price/bitcoin`** (sembol → 301) | **`/price/bitcoin`** | ✅ |
| Gezinme | 5 başlık | 7 başlık | **Bilinçli sapma** — veri yok, değiştirme |

---

## 12. Kopyalanması önerilen sözleşmeler

Ölçüm sırasında CoinDesk'in iyi kararları:

1. **`-fixed` son eki** — temayla dönmeyen rengi adından belli ediyor
   (`bg-surface-brand-fixed`). Bizde `--cd-warn` gibi adlar ne olduğunu
   söylemiyor.
2. **`aspect-video` sarmalayıcı** — en-boy oranı CSS'te sabit, görselin
   gerçek oranı bilinmese de CLS sıfır. Bizde kartlarda var (`.cd-card__media`
   `aspect-ratio: 16/9`), **makale kapağında YOK** — doğrulandı:
   `.cd-article__cover` yalnız `margin` taşıyor, `img`'de `width: 100%` var
   ama oran yok. Kaynak görselin oranı bilinmediği için `width`/`height` de
   basılamıyor → makale sayfasında CLS riski. Sarmalayıcıya `aspect-ratio`
   eklenmesi hem CLS'i sıfırlar hem CoinDesk'le eşler.
3. **Mobilde yatay kart** (`flex-row md:flex-col`) — küçük ekranda görsel
   solda, metin sağda; dikey listede daha çok başlık sığıyor.
4. **Erişilebilir fiyat metni** — `6.33%` görsel, `Down 6.33 percent`
   okuyucuya.
5. **Sticky yüksekliğinin bilinmesi** — iki katman sticky olduğu için
   çıpa kaydırması hesaplı.
6. **`sizes` çarpanı** — §9.

---

## 13. Açık kararlar — KAPANDI (6 Kasım 2026)

| # | Soru | Karar | Gerekçe |
|---|---|---|---|
| 1 | Font: lisans mı, ikame mi? | **Yığın: Neue Haas → Inter → sistem** | Lisans alınırsa ilk sıra zaten devreye girer, kod değişmez. Webfont GÖMÜLMEDİ: 4 ağırlık ≈120 KB ve Lighthouse 100'ü korumak istiyoruz; sistem groteskleri Neue Haas'a yeterince yakın. |
| 2 | Kapsayıcı 1512'ye çıksın mı? | **Evet, 1512 px** | Spec'in tamamı bu genişlikte ölçüldü; 1280'de kalırsak `sizes` hesabı, ızgara oranları ve boşluk ölçeği tutmuyor. 1920'de yatay taşma yok (doğrulandı). |
| 3 | Makale gövdesi 22 px mi? | **22 px (mobilde 20 px)** | Masaüstünde satır uzunluğu 680 px sütunda 22 px'te ~48 karakter — Türkçe için kabul edilebilir üst sınır. Mobilde 22 px tek satırda 20-25 karaktere düşüyordu; 20 px'e indirildi. |
| 4 | `/price/{sembol}` mı `/price/{tam-ad}` mı? | **Tam ad kanonik**, sembol **301** | "bitcoin price" araması tam adla birebir eşleşiyor. Eski `/price/btc` adresleri 301 ile taşındı: bağlantı kırılmadı, iki adres ayrı sayfa olarak indekslenmiyor (kanonik tekilleşti). |
| 5 | Koyu tema tercihi saklanacak mı? | **Evet, `localStorage('tema')`** | CoinDesk deseni. İlk sınıf `<head>` içinde bloklayıcı minik betikle yazılıyor (FOUC yok); `localStorage` kapalıysa `prefers-color-scheme` devralıyor. Düğme JS yoksa gizli (`html.cd-js`). |

## 14. Bilinçli sapmalar (değiştirilmeyecek)

| Konu | CoinDesk | bitcosar | Neden |
|---|---|---|---|
| Gezinme başlıkları | 7 (News · Video · Prices · Research · **Events** · **Data & Indices** · Sponsored) | 5 | Etkinlik takvimimiz ve kendi endeks metodolojimiz yok — uydurma sayfa 404 demektir (bkz. §9 menü sözleşmesi) |
| Makale breadcrumb | yok | **var** | Toplayıcı sitede kaynak/atıf zinciri SEO ve şeffaflık için gerekli |
| Makalede kaynak kutusu | yok | **var (zorunlu)** | Telif/atıf yükümlülüğü; tam metin yayınlanmıyor |
| Yazar kutusu | yok | **var** | E-E-A-T sinyali + düzeltme kanalı |
| Paylaşım | X · LinkedIn · diyalog | X · WhatsApp · LinkedIn | Türkiye'de WhatsApp baskın paylaşım kanalı |
| Yorumlar | var | yok | Moderasyon yükü; Faz 3 |

---

## 15. Uygulama sırasında bulunan GERÇEK hatalar

Bu tur yalnız görsel değerleri düzeltmedi; **iki sessiz hata** ortaya çıktı.
İkisi de "sayfa 200 dönüyor, ama bir şey çalışmıyor" sınıfındaydı — bu
yüzden aylarca görünmeden kalabilirdi.

### 15.1 `cspWriter.Flush` HTML tamponunu bırakıyordu → satır içi betikler engelli

`templ`, bileşeni bitirirken kendi tamponunu boşaltıyor ve yazıcı
`http.Flusher` ise `Flush()` çağırıyor (`runtime.ReleaseBuffer`).
`cspWriter.Flush` bu çağrıyı "akışa geç" sayıp `akis = true` yapıyordu;
`finish()` akışta CSP üretmediği için yanıt **hash'siz** politika alıyordu.

Sonuç: templ ile render edilen **her sayfada** satır içi `<script>`
engellendi. Konsolda:

```
Executing inline script violates … 'script-src 'self' 'sha256-…'
Either the 'unsafe-inline' keyword, a hash ('sha256-yuDK…'), or a nonce …
```

Düzeltme: `Flush`, HTML yanıtında tamponu **korur** (hash ancak nihai
gövdeden üretilebilir); HTML dışı akışlar (zip/csv/SSE) eskisi gibi akar.
Regresyon testi: `TestCSPWriterHTMLFlushSonrasiHash`.

### 15.2 CSP hash'i HTML **yorumundaki** `<script>` kelimesini sayıyordu

Hash'leyici metinde `<script` arıyordu. Templ'in basacağı yorum metninde
"`buildCSP gövdedeki her <script>'in sha256'sını…`" yazıyordu; yorum +
gerçek betik tek parça olarak hash'lendi:

| | bayt | hash |
|---|---|---|
| sunucu (yorum dahil) | 478 | `sha256-+5vpu0VH…` (yanlış) |
| tarayıcı (yalnız gerçek) | 400 | `sha256-yuDKVlqu…` (doğru) |

Hash hiç tutmadı. Düzeltme: `htmlYorumAtla` — yorum içindeki etiket metni
sayılmaz (`<script>` ve `<style>` için). Regresyon testleri:
`TestCSPYorumIcindekiEtiketSayilmaz`, `…StilSayilmaz`, `…ScriptTamamenAtlanir`.

### 15.3 Statik varlık sürümü `?v=1` sabitti

Tasarım sistemi baştan yazıldığında tarayıcılar eski `cd.css`'i önbellekten
servis etti; ölçüm "hiçbir şey değişmemiş" gibi göründü ve **yanlış teşhis**
riski doğdu (değerler doğruydu, dosya eskiydi). Sürüm artık süreç başına
değişiyor (`assetVer` → `templkit.VarlikSurum`).

### 15.4 Doğrulama

Gerçek Chrome, 1920×871, `getComputedStyle` — spec §11'deki kalemlerin
tamamı ölçüldü; **25/25 uyumlu**. Tema anahtarı: açık → koyu geçiş,
`localStorage` kalıcılığı, yeniden yükleme sonrası tercih korunuyor.
Lighthouse: `/` 100 · `/markets` 100 · `/price` 100 (SEO 100, BP 96, CLS 0).

---

*Ölçen: Claude Opus 5, 6 Kasım 2026, gerçek Chrome, hesaplanmış stiller.
Ölçülmeyen kalemler §Ölçüm künyesi'nde işaretli.*
