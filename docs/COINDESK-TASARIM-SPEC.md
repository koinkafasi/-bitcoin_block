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

**İkinci tur eklemeleri (6 Kasım):** kenarlık/ayırıcı haritası (33 desen),
bant genişliği alternasyonu, başlık+şerit iç kapsayıcı hizası, reklam
yuvası, hero tam kompozisyonu, footer iç yapısı, modül dikey ritmi,
slider mekaniği, **şeridin kaydırma durumu dönüşümü ve sekme veri
değişimi**.

**Üçüncü tur (6 Kasım):** semantik font ölçeği (`font-*` sınıf haritası),
makale sayfasının tam sıralı anatomisi, **Özet/Summary açılır kutusu**,
aksiyon satırı üç pill'i (okuma süresi · Paylaş · Google News), tarih
bloğunun `::before` ayırıcısı, TR↔EN dil farkları, **hisse detay şablonu**
(`/price/stock/*`) ve coin sayfasından farkları.
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

### 2.1 Tam genişlik ↔ kapsanmış bant alternasyonu

**Sayfanın en belirgin yapısal kuralı; kolay kaçırılıyor.** Sayfa iki tür
yatay bandın sırayla dizilmesinden oluşuyor.

Ölçülen dizilim (ana sayfa, 1920px viewport):

```
y   73   1905px  TAM GENİŞLİK   fiyat şeridi      (iç kapsayıcı 1512px)
y  140   1512px  KAPSANMIŞ      hero — 3 kolon [290px 608px 290px]
y 1936   1905px  TAM GENİŞLİK   «Advertisement»   (iç kapsayıcı YOK)
y 2000   1512px  KAPSANMIŞ      «Explore more…» — TEK kolon + slider
y 2721   1512px  KAPSANMIŞ      «Opinion» — 2 kolon [427.5px 598.5px]
y 3248   1905px  TAM GENİŞLİK   «Advertisement»
y 3312   1512px  KAPSANMIŞ      2 kolon [724px 724px]
```

**Kritik ayrım — başlık ve şerit:** dış kutuları tam genişlik ama
**içerikleri gövdeyle birebir hizalı.**

```
header   dış 1905px (sol 0)   →   iç sarmalayıcı 1512px, sol 197, padX 24px
şerit    dış 1905px (sol 0)   →   iç sarmalayıcı 1512px, sol 197, padX 24px
gövde          —              →   kapsayıcı      1512px, sol 197, padX 24px
logo                              sol 221  (= 197 + 24)  ✓ hizalı
```

Yani **tam genişlik olan yalnızca zemin ve alt çizgi.** Menüdeki logo,
gezinme ve şeritteki fiyatlar gövde sütunuyla aynı hizada başlıyor.
"Menü geniş, gövde dar" görünümünün kaynağı bu: arka plan kenardan kenara
akıyor, içerik akmıyor.

**Reklam yuvaları farklı:** iç kapsayıcı YOK, gerçekten kenardan kenara —
`1905×64`, `background: transparent`, `padding: 32px 0`, metin
"Advertisement". İçerik modüllerinin arasına girip ritmi kapsanmıştan tam
genişliğe çeviriyorlar.

### 2.2 Ölçülen kolon düzenleri

| Bölge | Kolon | Ölçüm |
|---|---|---|
| Hero | **3** | `290px 608px 290px`, gap 24px/80px, **≥1284px** |
| «Explore more» | **1** (ızgara yok) | tek kolon + yatay slider |
| «Opinion» | 2 | `427.5px 598.5px` — **eşit değil** |
| Alt blok | 2 | `724px 724px` — eşit |
| Bölüm sayfası | 12 | `100.656px` × 12, gap 16px → 1384px |
| Footer | 5 | `236.797px` × 5, gap 24px → 1280px |

**Üç ayrı genişlik kademesi var:** gövde 1512 · bölüm ızgarası 1384 ·
footer 1280.

### 2.3 Slider bölgesi (yatay kaydırmalı)

«Explore more from CoinDesk» ölçümü:

```
bölge      1512px kapsanmış (iç 1464px)
grid-template-columns: none          ← ızgara YOK, tek kolon
kaydırıcı  1464px
           overflow-x: auto
           gap: 16px
           scroll-snap-type: x mandatory
           çocuk: 6
kart       400px   scroll-snap-align: start
oklar      24×24   border-radius: 9999px
           aria-label "Previous cards" / "Next cards"
```

**JS'siz çalışıyor:** yerel kaydırma + CSS snap; oklar yalnız kolaylık.
Dikey klip karuseli aynı desen ama `aspect-[9/16]` ve `rounded-lg`.

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

### 4.1 Semantik font sınıfları — CoinDesk'in KENDİ adlandırılmış ölçeği

Sınıf adlarından çıkarıldı (`font-*`), hesaplanmış değerlerle eşlendi.
**Bu, frekans tablosundan daha değerli: hiyerarşinin kendi adları.**

| Sınıf | Ölçüm | Nerede | Sıklık |
|---|---|---|---|
| `font-serif` | **40px / 48px w400** | büyük teşhir başlığı | 10× |
| `font-headline` | **36px / 44px w500** | **makale H1**, bölüm H1 | 3× |
| `font-headline-sm` | **28px / 36px w500** | makale ara başlığı (H2/H3) | 2× |
| `font-body-lg` | **22px / 30px w400** | **makale gövdesi** | 4× |
| `font-headline-2xs` | **20px / 26px w500** | kart başlığı | 13× |
| `font-body` | **18px / 24px w400** | — | 3× |
| `font-title` | **16px / 32px w500** | modül başlığı, kicker | 2× |
| `font-label` | **14px / 24px w500** | kutu başlığı ("Summary") | 2× |
| `font-metadata-lg` | **14px / 22px w400** | yazar, tarih, pill — **en sık** | 45× |
| `font-sans` | **14px / 22px w400** | gövde dışı metin | 39× |
| `font-metadata` | **12px / 16px w400** | küçük meta | 13× |

`!text-md !leading-6` gibi `!` önekli ezmeler de var — özet kutusunda
`font-body-lg`'yi 16/24'e düşürmek için (§10.2.2).

> **Bizde:** adlandırılmış ölçek yok, `cd.css`'te her yerde ham `rem`
> değerleri. Bu tabloyu jetonlaştırmak (`--cd-font-headline` vb.) hem
> tutarlılığı hem değişikliği kolaylaştırır.

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

## 6. Çizgi, ayırıcı, köşe, gölge sistemi

Kenarlık taşıyan **33 farklı öğe deseni** tarandı. Tek cümleyle:
**her kenarlık 1px solid.** 2px, 3px, `dashed`, `dotted` hiç yok.

### 6.1 İki kademeli ayırıcı — bu bir hiyerarşi

| Kademe | Jeton | Açık | Koyu | Kullanım |
|---|---|---|---|---|
| **Güçlü** | `border-strong` | #A6A6A6 | #585858 | **`<section>` üst çizgisi** = modül ayırıcı |
| Normal | `border-default` | #D9D9D9 | #262626 | `<div>` iç ayırıcı, kart çerçevesi, tablo satırı |

```
section.border-t.border-strong       modül ayrımı                10 kez
div.my-4.border-t.border-default     iç ayırıcı, margin-y 16px    8 kez
div.mb-4.border-t.border-default     iç ayırıcı, margin-b 16px    5 kez
div.mb-2.border-t.border-default     iç ayırıcı, margin-b 8px     4 kez
div.border-t.border-default          çıplak iç ayırıcı            8 kez
div.border-b.border-default          alt çizgi                    1 kez
hr.border-default                    3 adet; 1px, margin 0
```

**Kural:** modülü diğerinden ayıran çizgi **güçlü**, modül İÇİNDEKİ
bölmeleri ayıran çizgi **normal**.

### 6.2 Dikey tüy çizgi (hairline)

```
div.h-8.border-l.border-default     1×32px     6 kez
```

Satır içi öğeleri ayıran **32px yüksekliğinde 1px dikey çizgi**,
`border-left` ile, margin yok.

### 6.3 Kolon ayırıcısı — `::before`, boşluğun tam ortasında

Hero'nun merkez kolonu:

```css
section::before {
  content: "";
  position: absolute;
  width: 1px;
  height: 1732px;          /* kolon boyu */
  background: #262626;     /* border-default — BORDER DEĞİL, BACKGROUND */
  inset: 0 647px 0 -40px;
  left: -40px;             /* 80px kolon boşluğunun TAM ORTASI */
}
```

**80px kolon boşluğunun varlık sebebi bu:** çizgiyi ortalamak, iki yanda
40px nefes payı bırakmak. `border` değil `background` olduğu için ızgara
hücresinin dışına, boşluğun içine taşabiliyor — `border` ile yapılamaz.

### 6.4 Modül dikey ritmi — margin YOK

Ölçülen 8 modülün hepsinde `margin-top: 0`, `margin-bottom: 0`:

```
   ⟨ebeveynin gap'i: 24px⟩
───────────────────────────  border-top: 1px border-strong
   ⟨padding-top: 16px⟩
   modül içeriği
```

**Üstteki boşluk ebeveynin `gap`'inden, çizgi altındaki 16px modülün
kendi `padding-top`'ından.** Modül kendi dışına margin yazmadığı için
yer değiştirdiğinde ritim bozulmuyor (`order-*` ile dizilim mümkün —
bkz. §7A).

Modül içi `gap`: **12px · 16px · 20px**. Ana içerik bandı
`padding-top: 0`, `padding-bottom: 32px`.

### 6.5 Köşe yarıçapı

```
4px      (120×)  kart görseli (`rounded`) — varsayılan
8px      ( 72×)  `rounded-lg`: dikey video, büyük kutular
9999px   ( 66×)  pill: düğme, rozet, yuvarlak ok
1px      ( 19×)  neredeyse keskin
6px      (  4×)  şerit sekmesi
50%      (  3×)  avatar
```

### 6.6 Gölge

```
0 0 10px rgba(0,0,0,0.4)    21 öğe
0 0 10px rgba(0,0,0,0.5)     1 öğe
```

Kaydırma (offset) ve yayılma (spread) **yok** — yalnız 10px bulanıklık.
Kartlarda gölge yok; bindirme/açılır katmanda var.

### 6.7 Dört kenarı çerçeveli öğeler

```
div.overflow-hidden.rounded.border.border-default             kart çerçevesi
div.relative.h-[632px].rounded-lg.border            (5×)      632px sabit kutu
span.rounded.border.border-default.px-1.5                     rozet, yan dolgu 6px
button.size-6.rounded-full  border 1px rgba(255,255,255,.3)   24px yuvarlak ok
li.flex.items-center.gap-2.p-[7px]  border rgba(255,255,255,.06)  tablo satır kartı
```

Yuvarlak düğme ve liste satırı kenarlıkları **yarı saydam beyaz** —
jeton değil, bağlama özel.

---

## 7. Global bileşenler — tam iç yapı

### 7.1 Başlık (header)

```
header#main-navigation
dış:  1905×73   sol 0      max-width: none   padding: 0
iç:   1512px    sol 197    padding-x: 24px        ← gövdeyle AYNI
position: sticky · top: 0 · z-index: 102
```

| Parça | Ölçüm |
|---|---|
| Logo | 169×32, **sol 221** (= 197 + 24, gövde içeriğiyle hizalı) |
| Gezinme | 642px, sol 422; 7 öğe, **16px/24px w400** |
| Görünürlük | `hidden h-full lg:block` — 1024 altı gizli, yerine hamburger |

### 7.2 Fiyat şeridi (ticker) — kaydırmaya göre DEĞİŞİYOR

```
dış:  1905×67   sol 0     padding: 0
iç:   1512px    sol 197   padding-x: 24px
position: sticky · top: 73px · z-index: 40
border-bottom: 1px solid border-default   (tam genişlik)
```

**İki katmanlı sticky yığın:**

```
header   top 0     yükseklik 73   z-index 102
şerit    top 73    yükseklik 67   z-index  40      ← header'ın ALTINA kenetlenir
                   ─────────────
                   toplam 140px sabit
```

Header z-index'i daha yüksek: şerit header'ın altından geçer.
Çıpa kaydırmasında `scroll-margin-top: 140px` gerekir.

#### Kaydırma durumu — öğe iki satırdan tek satıra iner

Şeridin kapsayıcısı sabit (1905×67, top 73) ama **içindeki öğenin
yerleşimi değişiyor.** Ölçüldü:

| | Sayfa başı (scrollY 0) | Kaydırılmış (scrollY 1500) |
|---|---|---|
| Öğe kutusu | **157×50** | **196×32** |
| `flex-direction` | **column** | **row** |
| `gap` | 2px | **8px** |
| `align-items` | normal | **baseline** |
| `padding` | 4px 8px | 4px 8px (değişmez) |
| Sembol kutusu | 141×18 (tam genişlik) | **33×18** (içerik kadar) |
| Fiyat kutusu | 141×22 | 139×22 |

Yani: başta **sembol üstte / fiyat altta** (iki satır, 50px yüksek),
kaydırınca **yan yana** (tek satır, 32px yüksek). Dikeyde 18px kazanılıyor,
yatayda öğe 39px genişliyor. Sembol kutusu `w-full`'dan içerik genişliğine
düşüyor.

#### Sekmeler — veri kaynağını değiştiriyor

İki sekme: `Markets Live` (Crypto) ve `Crypto Stocks`.

| | Ölçüm |
|---|---|
| Tipografi | 12px/16px **w500** (aktif) / **w400** (pasif) |
| Aktif zemin | `surface-default` · Pasif: `transparent` |
| Köşe | `border-radius: 6px` |
| Dolgu | `padding: 0 8px` |
| `aria-selected` | **YOK** — aktiflik yalnız ağırlık + zeminle anlatılıyor |

**Sekme değişince şeridin verisi tamamen değişiyor** (ölçüldü, 7 öğe → 7 öğe):

```
Markets Live (kripto + kendi endeksi):
  CD20 $2,052.08 ▼6.49%   BTC $75,486.03 ▼6.02%   ETH $2,222.27 ▼8.08%
  XRP $1.20 ▼8.23%        SOL $89.22 ▼6.78%       HYPE $77.58 ▼6.75%
  TRX $0.31 ▼0.46%

Crypto Stocks (ABD hisseleri):
  SPY $733.22 ▼4.38%   BLSH $20.67 ▼10.56%   COIN $136.68 ▼9.78%
  CRCL $69.38 ▼8.91%   MSTR $129.32 ▼9.21%   NVDA $181.87 ▼9.16%
  HOOD $78.19 ▼10.57%
```

**URL değişmiyor** (`location.search` boş) — istemci tarafı durum, sorgu
parametresi yok. Yani sekme seçimi paylaşılabilir/derin bağlanabilir değil.

| Parça | Ölçüm |
|---|---|
| Şerit öğesi bağlantısı | `/price/{sembol}` ve `/index/{endeks}` |
| Yüzde | 12px/18px w500, pos/neg renkleri |
| Erişilebilirlik | görselde `6.33%`, okuyucuya ayrı düğümde `Down 6.33 percent` |

> **bitcosar farkı:** bizim `cdTicker` iki sekmeyi `?serit=markets` /
> `?serit=stocks` sorgu parametresiyle çözüyor (sunucu tarafı, derin
> bağlanabilir — CoinDesk'ten **iyi**). Ama kaydırma durumu dönüşümü
> (iki satır → tek satır) ve header'ın sticky olması bizde yok.

### 7.3 Reklam yuvası

```
1905×64   background: transparent   padding: 32px 0
iç kapsayıcı YOK — gerçekten kenardan kenara
metin: "Advertisement"
```

Ölçülen iki yuva: y1936 ve y3248 (bkz. §2.1).

### 7.4 Alt bilgi (footer)

```
footer#footer   1905×966
padding: 48px 32px · flex column · items-center
iç kapsayıcı: max-w 1280px · flex column · gap 32px
```

Dört yatay bant (ölçülen sıra):

| # | İçerik | Ölçüm |
|---|---|---|
| 1 | Logo | 1280×29 (logo 150×29) |
| 2 | Bağlantı sütunları | 1280×562 — **grid 5 × 236.797px, gap 24px** |
| 3 | Yasal bağlantılar | 534×24 + 288×24, `justify-between`, gap `8px 24px` |
| 4 | Feragat + telif | 1280×21 ve 1280×48, `flex column gap: 24px` |

Mobilde sütunlar `grid-cols-2`, 768+ **5 sütun**.

| Öğe | Ölçüm |
|---|---|
| Sütun başlığı | **14px/22px w500**, `text-strong`, **`capitalize`** |
| Sütun bağlantısı | **12px/16px w400**, `text-subtle` |
| Telif | 12px/21px w400, `text-subtle` — "© 2026 CoinDesk, Inc." |
| Sosyal ikon | 5 adet |
| `<hr>` | 1px `border-default`, margin 0, 1280px (2 adet) |

Sütunlar **eşit uzunlukta değil** — içerik kadar ("About" 46 bağlantı,
"Cryptocurrencies" 25 bağlantı).

**Footer iç genişliği 1280px** — gövdenin 1512'sinden dar, üçüncü kademe.

---

## 7A. Hero gövdesi — tam kompozisyon

Menünün hemen altındaki **3 kolona bölünmüş** alan.

```
div.group/hero   1464×1796
  flex flex-col gap-6 pt-8 pb-4
  [@media(min-width:1284px)]:  grid-cols-[290px_608px_290px]   gap: 24px 80px
```

**Kırılma 1284px** — ızgara merdiveninde 1280 değil **1284**. Altında tek
kolon, `flex column`, `gap: 24px`.

| | |
|---|---|
| Kolon genişlikleri | **290px · 608px · 290px** (açıkça px, `fr` değil) |
| Boşluk | satır 24px · **kolon 80px** (ortasında 1px çizgi, §6.3) |
| Dolgu | `padding-top: 32px` · `padding-bottom: 16px` |
| Toplam | 290+608+290 + 2×80 = **1448** (1464 içeriğe oturur) |

### Sol ray (290px) — `aside.contents`

`display: contents` → çocuklar doğrudan ebeveyn ızgarasına katılıyor, ara
sarmalayıcı ızgarayı bozmuyor. `gap: 24px`. Üç blok (`order-2`, `order-8`,
`order-10`), her biri `section.border-t.border-strong.pt-4` → 455 / 306 / 208px.

### Merkez (608px) — `section.contents`

Sol kenarında `::before` dikey ayırıcı (§6.3).

| Sıra | İçerik | Ölçüm |
|---|---|---|
| `order-1` | **lead kart** | `article.flex.flex-col.gap-4` 608×571 (görsel 608×342) |
| `order-5` | blok | 608×119, gap 16px |
| `order-7` | blok | 608×237 |
| `order-9` | blok | 608×237 |
| `order-11` | blok | 608×181 |

### Sağ ray (290px) — `aside.hidden ... 1284px:flex`

1284px altında **tamamen gizli**. `gap: 24px`.

| Modül | Ölçüm |
|---|---|
| "Today's Videos" | `section.gap-4` 290×360; başlık **14px w500**; çerçeveli kutu 290×268 |
| "Featured Research" | `section.border-t.border-strong.pt-4.gap-5` 290×503; kart 290×386 |
| Dikey klipler | `section.border-t.border-strong.gap-3` 290×566; **`aspect-[9/16]`** 290×516, `rounded-lg`, `bg-black-fixed` |

**Yan ray modül başlıkları 14px w500, ana kolon modül başlıkları 16px w500**
— iki ayrı kademe.

### `order-*` ile yeniden dizilim

Masaüstü yerleşimi ızgarada, mobil okuma sırası `order` ile
(1, 2, 5, 7, 8, 9, 10, 11). **Tek HTML, iki kompozisyon** — modüllerin
margin yazmaması (§6.4) bunu mümkün kılıyor.

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

### 10.2 Makale sayfası — tam sıralı anatomi

```
Sütun:    680px, YATAY ORTALANMIŞ (sol 597px @1920)
Yan ray:  YOK — tek sütun
```

Ölçülen sıra (Türkçe makale, y koordinatıyla):

| y | Öğe | Ölçüm |
|---|---|---|
| 212 | kicker | `div.font-title` 680×32 — **16/32 w500**, bölüm adı ("Piyasalar") |
| 246 | **H1** | `h1.font-headline.font-medium.mt-1` **36/44 w500** |
| 386 | deck | `h2.font-sans.text-base.leading-6` **16/24 w400** |
| 458 | "Yazan" | `div.font-metadata-lg.text-sm` 680×22 — 14/22 w400 |
| 460 | yazar adları | `a.underline` — **ALTI ÇİZİLİ** (sitede nadir!) |
| 488 | tarih bloğu | `div.font-metadata-lg.text-subtle` 680×47 |
| 510 | çeviri etiketi | `a` 160×25 — **"AI tarafından çevrildi"** (yalnız çeviri dillerinde) |
| 551 | aksiyon satırı | `flex gap-3 flex-wrap` 680×32 — üç pill |
| 615 | kapak | `figure` 680×383 |
| 1092 | **Özet kutusu** | §10.2.2 |
| 1178 | gövde | `div.document-body.font-body-lg` **22/30 w400** |

#### 10.2.1 Tarih bloğu — `·` ayırıcısı `::before` ile

```html
<span>Güncellendi</span> <time>5 Kas 2026 ÖS 9:26 GMT-5</time>
<span class="before:content-['·'] before:mx-2 before:text-subtle">Yayınlandı</span>
<time>5 Kas 2026 ÖS 1:48 GMT-5</time>
```

**İki tarih birlikte gösteriliyor:** güncelleme + yayın. Aradaki `·`
ayırıcı bir karakter değil, `::before` ile basılıyor; `mx-2` = **8px**
yan boşluk, rengi `text-subtle`. Saat dilimi **GMT-5** (UTC değil).

#### 10.2.2 Özet kutusu (açılır) — "Summary" / "Özet"

Kapak görselinin **altında**, gövdenin **üstünde**.

```
dış çerçeve   div.mt-4.mb-8.rounded-[7px].p-[2px]     680×116
                margin: 16px 0 32px · radius 7px · padding 2px
iç kutu       div.rounded-[5px].bg-surface-default     676×112
                radius 5px · padding 20px 20px 12px · bg surface-default
başlık satırı div.flex.items-center.justify-between.mb-4   636×24
                span.font-label.font-medium   **14/24 w500**  "Summary"/"Özet"
                (sağda) "Show" / "Göster"  →  "Hide" / "Gizle"
içerik        div.transition-all.duration-300.ease-in-out
                div.document-body.font-body-lg.!font-sans.!text-md.!leading-6
                  → **16/24 w400** (gövdeden KÜÇÜK; `!` ile eziliyor)
                ul.unordered-list.pl-6
                  flex column · gap 8px · list-style none · pl 24px · mb 16px
                  li → pl 24px (toplam girinti 48px)
solma katmanı div.absolute.bottom-0.left-0.right-0.h-8   636×32  (kapalıyken)
```

**İki iç içe yarıçap (7px dış / 5px iç + 2px dolgu)** — 2px çerçeve
etkisi yaratıyor; düz `border` değil, bu yüzden gradyan çerçeve mümkün.

**Madde işareti gerçek bullet değil:**

```css
li::before { content: ""; width: 5px; background: #ECECEC; border-radius: 9999px; }
```

`list-style: none` + **5px yuvarlak nokta** `::before` ile.

Açık/kapalı ölçümü:

| | Kapalı | Açık |
|---|---|---|
| `ul` yüksekliği | **24px** (1 satır) | **184px** |
| Görünen `li` | 1 | 3 |
| `li` yükseklikleri | — | 72 / 48 / 48px |
| Düğme | "Show" / "Göster" | "Hide" / "Gizle" |
| Solma katmanı | **var** (`h-8` = 32px) | yok |
| Geçiş | `transition-all 300ms ease-in-out` | aynı |

İçerik AI üretimi özet; 3 madde.

#### 10.2.3 Aksiyon satırı — üç pill

`div.flex.items-center.gap-3.flex-wrap` 680×32, **gap 12px**.

| Pill | Ölçüm |
|---|---|
| Okuma süresi | 122×32 (TR 129×32) · `r9999px` · `border 1px border-default` · `padding 4px 16px 4px 12px` · `gap 4px` · 14/22 w400 · ikon + "3 min read" / "3 dk okuma" |
| Paylaş | `button.share-dialog-button.h-8` 91×32 · `r9999px` · "Share" / "Paylaş" |
| **Google News** | 300×32 (TR 319×32) · `r9999px` · `border 1px border-default` · `padding 4px 12px` · `background: transparent` · 14/22 w400 · "Make … preferred on Google News" / "…'ı …'da tercih edilen yap" · `href: google.com/preferences/source?q=coindesk.com` |

Üçü de **pill (9999px) ve 32px yüksek** — tutarlı bir aksiyon şeridi.

#### 10.2.4 Gövde ve diğer öğeler

| Öğe | Ölçüm |
|---|---|
| Gövde `<p>` | **22/30 w400**, `margin-bottom: 32px` |
| Ara başlık | **28/36 w500** (`font-headline-sm`) |
| Blockquote | 22/30 w400 |
| figcaption | 12/16 w400 **ls 0.4px**, `text-subtle` |
| Etiketler | **düz bağlantı** 16/24 — pill değil |
| Breadcrumb | makalede **YOK** (hisse sayfasında var) |

#### 10.2.5 Dil farkları (TR ↔ EN)

| | EN | TR |
|---|---|---|
| H1 yüksekliği | 88px (**2 satır**) | **132px (3 satır)** |
| Google News pill | 300×32 | 319×32 |
| Çeviri etiketi | yok | **"AI tarafından çevrildi"** 160×25 |
| Tarih | "Nov 5, 2026, 8:30 p.m. EST" | "5 Kas 2026 ÖS 9:26 GMT-5" |
| `<html lang>` | `en` | `tr` |

**Türkçe başlık aynı puntoda %50 daha uzun.** 22px gövde kararı
alınırken bu hesaba katılmalı (§13).

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

### 10.6 Hisse detay (`/price/stock/meta`) — coin sayfasından FARKLI şablon

Aynı değil: ölçüm iki ayrı şablon olduğunu gösteriyor.

```
İçerik bandı:  1432px        ← DÖRDÜNCÜ genişlik kademesi
Izgara:        392px 976px   gap 32px   → 1400px
```

| | Coin (`/price/bitcoin`) | **Hisse (`/price/stock/meta`)** |
|---|---|---|
| `<h1>` | 20/30 w500 ("Bitcoin") | **YOK** |
| Büyük fiyat | **36/38 w500** | **28/42 w500** |
| Grafik | `<canvas>` 1385×500 | `<canvas>` **976×500** |
| Düzen | tek kolon | **iki kolon 392 + 976** |
| Zaman sekmeleri | 1M · YTD · 1Y · All | **1D · 5D · 1M · YTD · 1Y · All** |
| Grafik kipi | — | **Basic / Advanced** |
| Breadcrumb | — | **var**: "Prices > Meta Platforms, Inc." |
| İçerik genişliği | 1512 | **1432** |

#### Sol ray (392px)

```
flex column · gap 40px · padding 12px 32px 32px
border-top 1px + border-bottom 1px (border-default)
iç genişlik 326px
```

| Bölüm | Ölçüm |
|---|---|
| Breadcrumb + ad | 326×245 |
| **"Key Stats"** | `section` 326×328 |
| **"About …"** | `section` 326×407 |

İstatistik satırı anatomisi (ölçüldü):

```
157×64   flex · gap 4px · padding 8px 0 · border-bottom 1px border-default
  etiket:  14/21 w400  text-subtle   "Market Cap"
  değer:   14/21 w700  text-default  "$1610.805B"
```

**Etiket w400 subtle, değer w700 default** — iki kademeli vurgu, her
satır alt çizgiyle ayrılıyor. Ölçülen alanlar: Market Cap, Volume (24h),
52-Week Range.

#### Sağ kolon (976px)

```
Grafik kipi:  «Basic» / «Advanced»  — 12/22 w400 · r6px · padding 4px 12px
              aktif: bg surface-default · pasif: transparent
Zaman:        1D · 5D · 1M · YTD · 1Y · All
OHLC okuma:   "O:$632.30  H:$…"  grafiğin üstünde
Performans:   976×88 · border-top 1px
              "24 hours · 5 days · 1 month · YTD · 1 year"  16/24 w400
              yüzdeler 14/21 w400, pos/neg renkleriyle
```

Sekme stili fiyat şeridindeki sekmeyle **aynı desen** (r6px, aktif zemin)
ama tipografi 12/22 (şeritte 12/16).

> **bitcosar:** bizde `/price/stock/*` şablonu **hiç yok**; ABD hisse
> verisi bağlanmadığı için menüden de çıkarıldı (§14). Bu bölüm veri
> kaynağı bağlanırsa uygulanmak üzere kayıtlı.

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

### 11.1 İkinci turda bulunanlar (6 Kasım, henüz uygulanmadı)

| Konu | bitcosar (bugün) | CoinDesk (ölçüm) | Aksiyon |
|---|---|---|---|
| **Bant genişliği** | `.cd-wrap` her yerde tek kapsayıcı | **tam genişlik zemin + 1512 hizalı iç içerik** | Başlık/şerit zeminini tam genişlik yap, içeriği 1512'de hizala |
| Reklam yuvası | kapsanmış | **tam genişlik**, `padding: 32px 0`, iç kapsayıcı yok | Düzelt |
| Genişlik kademeleri | tek (1512) | **üç**: gövde 1512 · bölüm 1384 · footer 1280 | Footer ve bölüm ızgarasını ayır |
| Ayırıcı kademesi | tek `--cd-border` | **`border-strong` (modül) + `border-default` (iç)** | İkinci jeton ekle |
| Kolon ayırıcısı | yok | `::before` 1px, 80px boşluğun ortasında (`background`) | Hero'ya ekle |
| Dikey tüy çizgi | yok | `border-l` 1×32px satır içi ayırıcı | Ekle |
| Modül ritmi | margin (`--cd-section-gap: 40px`) | **gap 24 + border-strong + `pt-4`**, margin YOK | Yaklaşımı değiştir |
| Hero ızgarası | `1.55fr 1fr` gap 32 | **`290px 608px 290px`** gap 24/80, ≥**1284px** | Düzelt |
| Hero yan ray | her zaman basılır | **1284px altında `hidden`** | Karar gerek |
| Mobil dizilim | sabit sıra | `order-*` ile ayrı mobil sırası | Teknik yok |
| Slider | yok | `scroll-snap-type: x mandatory`, kart 400px, **JS'siz** | Ekle |
| `display: contents` | yok | ray sarmalayıcıları ızgarayı bozmasın diye | Teknik yok |
| **Şerit kaydırma durumu** | tek görünüm | **`column`→`row`**, 157×50 → 196×32, gap 2→8 | Ekle |
| Şerit sticky | yalnız şerit | **header + şerit** (0/73, z 102/40) | Header'ı da sticky yap |
| Şerit sekme a11y | — | `aria-selected` **YOK** (CoinDesk eksiği) | **Bizde ekle** — onlardan iyi olalım |
| Şerit sekme durumu | `?serit=` sorgu param. | istemci tarafı, URL değişmiyor | **Bizimki daha iyi**, koru |
| Yan ray başlığı | — | **14px w500** (ana kolon 16px w500) | İki kademe ayır |
| Semantik font ölçeği | ham `rem` değerleri | **adlandırılmış**: `font-headline` 36/44, `font-body-lg` 22/30, `font-metadata-lg` 14/22 … | Jetonlaştır |
| **Özet kutusu** | yok | açılır, 7px/5px iç içe çerçeve, 5px nokta `::before`, 300ms geçiş | Ekle |
| Okuma süresi pill | yok | 122×32 `r9999px` border 1px, ikon + "3 min read" | Ekle |
| Google News pill | yok | 300×32 `r9999px`, `google.com/preferences/source` | Ekle |
| Paylaş düğmesi | bağlantı listesi | `button.h-8.rounded-full` 91×32 + diyalog | Pill'e çevir |
| Yazar bağlantısı | — | **`underline`** (sitede nadir istisna) | Eşle |
| Tarih bloğu | tek tarih | **Güncellendi + Yayınlandı**, `·` `::before` `mx-2` | Ekle |
| Çeviri etiketi | yok | **"AI tarafından çevrildi"** (çeviri dillerinde) | **Bizde gerekli** — içerik çevrilecek |
| Saat dilimi | — | **GMT-5 / EST** (UTC değil) | Karar gerek |
| Makale kicker | — | `font-title` 16/32 w500, bölüm adı | Eşle |
| Hisse şablonu | yok | `/price/stock/*` — 2 kolon 392+976, içerik **1432** | Veri gelirse |
| Genişlik kademeleri | tek (1512) | **dört**: 1512 · 1432 · 1384 · 1280 | Dördünü ayır |

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
