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

**Dördüncü tur (6 Kasım):** mega menü — tetikleyici düğme, panel
geometrisi ve menü başına genişlikler, üç katmanlı gölge, panel başlığı
satırı, iki içerik tipi (bağlantı listesi / **açıklamalı kart**) ve
**açılış efektinin olmadığının ölçülmesi**. Hover ile açılma
doğrulanamadı (sentetik olay CSS `:hover` tetiklemiyor).
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

### 7.2A Mega menü — tetikleyici, panel, içerik, efekt

#### Tetikleyici (üst gezinme öğesi)

```html
<nav aria-label="Primary" class="hidden h-full lg:block">      <!-- 642×40 -->
  <ul class="flex h-full items-center justify-center gap-1">   <!-- gap 4px -->
    <li class="group/disc hidden lg:flex">
      <button type="button" aria-expanded="false" aria-controls="_R_qkcqnivb_"
        class="appearance-none border-0 bg-transparent p-0
               hover:bg-surface-subtle flex h-full items-center
               justify-center rounded px-4">
        <span class="flex h-[2.5rem] …">News</span>
```

| | Ölçüm |
|---|---|
| Öğe | **`<button>`**, bağlantı değil |
| Tipografi | **16/24 w400** |
| Dolgu | `padding: 0 16px` |
| Köşe | `rounded` = **4px** |
| Zemin | saydam · **hover: `surface-subtle`** |
| Öğe arası | `ul` **gap 4px** |
| Erişilebilirlik | `aria-expanded` + `aria-controls` **var**; `aria-haspopup` **YOK** |
| İç yükseklik | `h-[2.5rem]` = 40px |

**8 üst başlık** ölçüldü: News · Video · Prices · Research · Events ·
Data & Indices · **More** · Sponsored.
(Daha önce 7 saymıştım; "More" taşan öğeleri toplayan sekizinci başlık.)

#### Panel

```
div.absolute.z-10.left-[12rem].top-[64px].w-[750px]
position: absolute · left: 192px (12rem) · top: 64px · z-index: 10
padding: 20px 24px            (Research'te 20px 16px)
border-radius: 8px
border-left: 1px + border-top: 1px   solid border-default
background: surface-default
box-shadow:
   0  2px  2px -1px rgba(10,10,10,0.04),
   0  4px  6px -2px rgba(10,10,10,0.03),
   0 12px 16px -4px rgba(10,10,10,0.08)
```

**Üç katmanlı yükselti gölgesi** — sayfadaki diğer gölge (`0 0 10px`)
ile aynı değil; menü kendi yükselti jetonunu kullanıyor.

**Panel genişlikleri menü başına FARKLI** — içeriğe göre sabitlenmiş:

| Menü | Genişlik | Konum |
|---|---|---|
| Video · Prices · Events · Sponsored | **589px** | `left: 192px` |
| Data & Indices | **728px** | `left: 192px` |
| **News** | **750px** | `left: 192px` |
| Research | **836px** | `left: 192px` |
| More | **960px** | `left: 192px` |

İstisna: **Sponsored** `left: auto` + `top: 100%` — son başlık olduğu
için sağa hizalı, yoksa ekrandan taşardı.

#### Açılış efekti — YOK (ölçüldü)

```
transition-property: all        transition-duration: 0s
animation-name: none            animation-duration: 0s
transform: none                 backdrop-filter: none
will-change: auto
```

Kapalı hâl: `display: none` + `opacity: 0` (`visibility: visible`).
Açık hâl: `display: block` + `opacity: 1`.

**Yani solma, kayma veya ölçek animasyonu yok; panel anında görünüyor.**
`display` animasyonlanamadığı için `transition: all` yazılı olsa da
etkisiz. Tetikleyici düğmede de `transition: all 0s` — hover zemini de
anında.

> Ölçüm notu: panel `click` ile açıldı ve `aria-expanded` `true` oldu.
> JS ile gönderilen `mouseover`/`mouseenter` paneli açmadı — gerçek
> işaretçi gerektiği için **hover ile açılıp açılmadığı doğrulanamadı**.
> `li` üzerindeki `group/disc` sınıfı grup-hover düzeneğini düşündürüyor
> ama bu çıkarım, ölçüm değil.

#### Panel içeriği — iki yerleşim tipi

Ortak iskelet:

```
div > div.pb-3 > div.flex.flex-col.gap-4        (700×392, gap 16px, pb 12px)
  1) başlık satırı   div.flex.items-center.gap-2    700×32
       svg 32×32 (ikon) + span.font-sans.font-bold 16/24 w700 + svg 9×18 (ok)
  2) içerik bloğu
```

**Tip A — bağlantı listesi** (News, Video, Prices):

```
div.mt-1.flex.w-full.flex-wrap        700×340   gap 20px · padding-left 8px
  div.flex.w-1/3.flex-col   231×340   gap 20px      ← 3 sütun (w-1/3)
    div.flex.flex-col.gap-3  231×223  gap 12px      ← satır grubu
    a.font-metadata-lg.font-bold  231×21            ← "See all …"
```

| Öğe | Ölçüm |
|---|---|
| Panel başlığı | **16/24 w700** (`font-sans font-bold`) |
| Grup başlığı | **14/21 w700** (`font-metadata-lg font-bold`) — ör. "Latest Crypto News", "Newsletters" |
| Normal satır | **14/22 w400** (`font-metadata-lg font-normal`) — ör. "Markets", "Tech" |
| "See all …" | **14/21 w700** — sütun sonunda, vurgulu |
| Sütun | 3 × `w-1/3` = 231px · sütun arası 20px · satır grubu 12px |

**Tip B — açıklamalı kart** (Research, muhtemelen Data & Indices / More):

```
ızgara: grid-template-columns: 256.656px 256.656px 256.656px · column-gap 16px
kart:   a   257×96   flex row · padding 12px 16px · radius 8px · bg saydam
```

| Öğe | Ölçüm |
|---|---|
| Kart başlığı | **14/20 w500**, `text-default` (`font-sans font-medium`) |
| Kart açıklaması | **12/16 w400**, `text-subtle` (`font-sans font-normal`) |

Ölçülen örnekler:
"**Research Hub** / Bespoke research, primers, and quarterly upd…" ·
"**Exchange Benchmark** / The institutional standa…"

**İki tip arasındaki fark:** liste tipi yalnız ad verir; kart tipi ada
bir satır açıklama ekler ve kendi kutusunu (8px yarıçap, 12/16 dolgu)
taşır.

> **bitcosar farkı:** bizim mega menü tek tip — düz bağlantı listesi
> (`cd_menu.templ`, `cd-mega__sutun`). Açıklamalı kart tipi, panel başlığı
> satırı (ikon + ad + ok), menü başına değişen panel genişliği ve üç
> katmanlı gölge yok. Panelimiz `:hover`/`:focus-within` ile JS'siz
> açılıyor — CoinDesk `aria-expanded` ile JS kullanıyor; **bizimki
> erişilebilirlik ve CSP açısından daha iyi**, korunmalı. Bizde
> `aria-haspopup` da var, onlarda yok.

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

### 7A.1 bitcosar'da uygulanışı — ölçülmüş (6 Kasım 2026)

Izgara birebir: `290px minmax(0,608px) 290px`, `gap 24px 80px`,
`justify-content: center`, `padding: 32px 0 16px`, kırılma **1283 px**.
Gerçek Chrome'da ölçülen: sol ray **x 286 · 290 px**, merkez **608 px**,
sağ ray **x 1344 · 290 px** — CoinDesk ile aynı x konumları.

| Ray | CoinDesk modülleri | bitcosar'da ne var |
|---|---|---|
| Sol (290) | Son Haberler · Düşünce Liderliği | **Son Haberler** — 5 satır: başlık 16/24 w500 (3 satır clamp) + saat 12/18 w400. Birebir anatomi; modül 290×571 (CoinDesk 575). |
| Merkez (608) | manşet + 4 blok | **Manşet** — metin üstte (36/44 w700), görsel altta 608×342 (16/9). |
| Sağ (290) | Bugünün Videoları · Öne Çıkan Araştırma · Bugünün Klipleri | **Üç bölüm kartı** (Markets · Finance · Tech): `border-top` strong + `padding-top` 16, başlık 14/21 w500 + ok, kart 290 (görsel 16/9 + başlık 20/26 w500 + meta 12/18, iç boşluk 8). |

**Basılmayanlar (veri yok, uydurma olurdu):** video oynatıcı (yayında video
0), "Öne Çıkan Araştırma" metrik kartı (yayında araştırma 0), 9/16 dikey
klipler (klip 0), sponsorlu "Düşünce Liderliği" (sponsorlu içerik 0).
Modül iskeleti hazır: ilgili içerik türü yayına girdiğinde şablon aynı
anatomiyi kullanacak.

**Bu turda bulunan iki gerçek hata** (ikisi de sessizdi):

1. `body { margin: 8px }` hiç sıfırlanmamıştı — tüm sayfa 8 px'lik çerçeve
   içindeydi; header x=8 w=1889 (olması gereken 0/1905). Ayrıca
   `overflow-x: clip` `body`'deydi, `html`'de değil → tam genişlik bant
   `scrollWidth` 1913 üretiyordu.
2. `.cd h1…h4` (özgüllük 0,1,1) bileşen sınıflarını eziyordu: kart başlığı
   20/24 (yazılan 20/26), manşet başlığı 28/36 (yazılan 36/44). Temel
   kurallar `:where()` ile sıfır özgüllüğe alındı.

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

### 10.3 Kategori / bölüm sayfası (`/tr/business`) — tam ölçüm

Ölçüm 6 Kasım 2026, 1920px, **Türkçe** sayfa (`/tr/business`, H1 "Finans").

#### Ağaç (gerçek DOM, geometriyle)

```
main.relative.flex.min-h-screen.flex-col            1905×2240
 section.flex-grow.border-t.border-default          1905×2240   ← ÜST ÇİZGİ
  div.container.mx-auto                             1432×953    maxW 1432 · grid-cols-16 · gap 24
   div.flex.flex-col.justify-center                 1384×873
    div.border-b.border-solid.border-strong.mb-6    1384×61     ← H1 bandı
    div.grid.xl:grid-cols-12.sm:grid-cols-4         1384×788    ← 12 kolon · gap 16
  div.container.mx-auto                             1432×1326   ← İKİNCİ bant
   div                                              1384×1246
    div.flex.flex-col.gap-6                         1032×1134   ← liste kolonu
    div.flex.justify-center.self-center.mt-6        1384×80     ← sayfalama
```

| | Değer |
|---|---|
| Kapsayıcı | **max-width 1432px** (ana sayfa 1512'den DAR) |
| İçerik | **1384px** (1432 − 48) |
| Dış ızgara | `grid-cols-16`, gap 24px |
| Kart ızgarası | **12 kolon**, gap **16px** → sütun 100.656px |
| Duyarlı | `xl:grid-cols-12` · `sm:grid-cols-4` · `xs:grid-cols-2` |
| Liste kolonu | **1032px** = 12 kolonun **9'u** (9×100.656 + 8×16) |
| `<section>` | `border-top: 1px border-default` |

#### H1 bandı

```
div.border-b.border-solid.border-strong.mb-6        1384×61
  h1.font-headline.font-serif.font-bold             1384×44   36px/44px w700
```

| | |
|---|---|
| H1 | **36/44 w700** (`font-headline font-serif font-bold`) |
| Alt çizgi | **1px `border-strong`** — modül ayırıcı kademesi (§6.1) |
| Alt boşluk | `mb-6` = **24px** |
| Açıklama paragrafı | **YOK** (hem `/markets` hem `/tr/business`'te yok) |

> Mevcut spec §8'de "bölüm adı + tek satır açıklama" yazıyordu — ölçümde
> açıklama yok. Düzeltildi.

#### Görsel yuvaları ve kart kademeleri

| Yuva (görsel ölçümü) | Izgara hücresi (KESİN) | Adet |
|---|---|---|
| 689×388 | **6 kolon = 683.9px** | 1 — öne çıkan |
| 327×185 | **3 kolon = 334.0px** | 5 |
| 241×135 | liste satırı küçük görseli | 4 |

> **DİKKAT — hangi sayı doğru:** soldaki sütun `getBoundingClientRect`
> ile GÖRSELDEN okundu; sağdaki ızgara matematiğinden. Kolon
> 100.656px, gap 16px, içerik 1384:
>
> ```
>  6 kolon + 5 gap = 6×100.656 +  5×16 =  683.9   (öne çıkan)
>  3 kolon + 2 gap = 3×100.656 +  2×16 =  334.0   (kart)
>  9 kolon + 8 gap = 9×100.656 +  8×16 = 1033.9   (liste)
> 12 kolon + 11 gap = 12×100.656 + 11×16 = 1383.9 (içerik)
> ```
>
> **Karşılaştırmada ızgara sayıları kullanılmalı.** Görsel ölçümleri
> (689/327) birkaç piksel sapıyor — görselin kendi oranı ve `object-fit`
> yuvayı tam doldurmayabiliyor. bitcosar 684/334/1034 ölçüyor, yani
> ızgaraya **tam isabet** ediyor; 689'a karşı 684 görmek bir fark değil,
> yanlış referanstır. (Bu tuzağa bir kez düşüldü ve beş sahte fark
> raporlandı.)

Hepsi 16:9. Kart başlıkları: **24/30 w500** (öne çıkan) · **20/26 w500** ·
**16/24 w500**. Bölüm başlığı 28/36 w700. Kart spotu **16/24 w400**.

#### Sayfalama

```
div.flex.justify-center.self-center.mt-6.md:mt-8     1384×80
  button  166×48 · radius 8px · padding 8px 24px · 14px w600 · border 1px
  metin: "Daha fazla hikaye"   (EN: "More stories")
```

**Sonsuz kaydırma YOK** — açık düğme. Üst boşluk `mt-6` (24px), 768px+
`md:mt-8` (32px). Ortalanmış (`justify-center self-center`).

#### Kategori sayfasında HERO YOK

CoinDesk'in kategori sayfası şu sırayla: üst çizgi → H1 bandı → 12 kolonlu
öne çıkan ızgara → liste kolonu (1032) → sayfalama. **`.cd-hero` benzeri
asimetrik manşet bloğu yok.**

---

### 10.3.1 bitcosar kategori sayfası — fark tablosu (ölçüldü)

`demo.bitcosar.com/business`, aynı viewport. **1 eşleşme / 5 fark.**

| Konu | CoinDesk | bitcosar | Aksiyon |
|---|---|---|---|
| H1 tipografi | 36/44 w700 | **36/44 w700** | ✅ |
| Kapsayıcı | **max-width 1432px** | 1432px | ✅ |
| İçerik genişliği | **1384** | 1384 | ✅ |
| Kart ızgarası | **12 kolon · gap 16** | 12 kolon · gap 16 | ✅ |
| H1 alt çizgisi | **1px `border-strong`** | `border-strong` | ✅ |
| H1 alt boşluğu | **24px** | 24px | ✅ |
| Sayfalama | r8px · 14px w600 · p8/24 | r8px · w600 · p8/24 | ✅ |
| Liste kolonu | **9 kolon = 1033.9** | 1034 | ✅ |
| Öne çıkan kart | **6 kolon = 683.9** | 684 | ✅ |
| Kategori kartı | **3 kolon = 334.0** | 334 | ✅ |
| Kart başlığı | 24/30 · 20/26 · 16/24 | 20/24 (19×) | Üç kademeyi kur |
| Görsel yuvaları | 689×388 · 327×185 · 241×135 | 429×241 · 596×335 · 290×163 | Izgara düzelince oturur |

#### GERÇEK HATA — lead kartı en dar kolonda (ÇÖZÜLDÜ, commit dd5f75b)

> **DURUM:** hero kategori sayfasından kaldırıldı, 12 kolonlu ızgara
> kuruldu. Canlı ölçüm: lead 684 > kart 334, boş kolon yok.
> `cd_kategori_test.go` / `TestKategoriHeroKullanmaz` geri dönüşü
> engelliyor. Aşağıdaki kayıt, niye yapıldığının belgesi.

Ölçülen ağaç:

```
section.cd-mod
  div.cd-hero                      1336×1275  [grid cols:3  gap:24px 80px]
    article.cd-card.cd-card--lead    290×477   ← LEAD, en dar kolonda
    div.cd-hero__side                596×1227  ← yan kartlar DAHA GENİŞ
    (üçüncü kolon BOŞ)
```

`.cd-hero` ızgarası ana sayfa için `290px 608px 290px` kurulmuş. Kategori
sayfası o ızgaraya **yalnız iki çocuk** veriyor; sonuç:

- Lead kartı **290px** — yan kartlardan (596px) **dar**. Manşet en küçük
  görünüyor, hiyerarşi tersine dönmüş.
- Üçüncü 290px kolon **boş** duruyor (1336 − 290 − 80 − 596 = 370 ≈ 290+80).

Bu bir tasarım tercihi değil, ızgara/çocuk uyuşmazlığı. İki yol var:

1. **CoinDesk'e uy:** kategori sayfasında hero'yu kaldır, 12 kolonlu öne
   çıkan ızgara + 9 kolonlu liste kur (§10.3).
2. **Hero'yu koruyacaksan:** kategori için ayrı ızgara ver
   (örn. `grid-template-columns: 689px 1fr` ya da `6fr 3fr`), lead'i geniş
   kolona yerleştir ve boş kolon bırakma.

Ölçüm (1) lehine: CoinDesk'in kategori sayfasında hero yok ve öne çıkan
görsel 689px (bizdeki 290'ın iki katından fazla).

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

### 11.1 İkinci ve üçüncü turda bulunanlar — DURUM (6 Kasım 2026, akşam)

**Uygulananlar** (gerçek Chrome'da `getComputedStyle` ile doğrulandı):

| Konu | Uygulanan |
|---|---|
| Bant genişliği | ✅ `.cd-tam` — tam genişlik zemin, içerik 1512'de hizalı (logo 221 px'te) |
| Reklam yuvası | ✅ tam genişlik, `padding: 32px 0`, iç kapsayıcı yok |
| Genişlik kademeleri | ✅ üç kademe: gövde 1512 · bölüm 1384 · footer 1280 |
| Ayırıcı kademesi | ✅ `border-strong` (modül) + `border-default` (iç) |
| Kolon ayırıcısı | ✅ hero sağ kolonunda `::before` 1 px, 80 px boşluğun ortası (`background`, border değil) |
| Dikey tüy çizgi | ✅ `.cd-tuy` 1×32 px |
| Modül ritmi | ✅ margin YOK: `gap 24` + `border-top` + `padding-top 16` |
| Hero ızgarası | ✅ `minmax(0,1fr) 290px`, aralık 80, `padding: 32px 0 16px` |
| Hero kırılması | ✅ **1283 px** altında tek kolon (1284 sınırı) |
| Yan ray başlığı | ✅ 14 px w500 (ana kolon 16 px) |
| Semantik font ölçeği | ✅ 11 `--cd-font-*` jetonu |
| Özet kutusu | ✅ 7/5 px iç içe yarıçap, 5 px nokta maddesi; **JS'siz `<details>`** |
| Okuma süresi pill | ✅ 32 px yüksek, r 9999 |
| Paylaş düğmesi | ✅ pill + `<details>` menüsü (JS'siz) |
| Tarih bloğu | ✅ Güncellendi + Yayınlandı, `·` `::before` ile (8 px) |
| Footer tipografi | ✅ 14/22 w500 capitalize başlık · 12/16 w400 bağlantı |
| Footer bantları | ✅ logo → 5 sütun → yasal → feragat, `<hr>` ile ayrık |
| Makale kicker | ✅ 16/32 w500 |
| Bölüm genişliği | ✅ `/markets` 1384 px |

**Bilinçli olarak UYGULANMAYANLAR** (veri/karşılık yok — uydurma olurdu):

| Konu | Neden |
|---|---|
| Hero sol/sağ ray modülleri (izleme listesi, Today's Videos, Featured Research, dikey klipler) | **KISMEN UYGULANDI** — 3 kolon kuruldu (290·608·290). Sol ray: "Son Haberler" (başlık + saat, görselsiz). Sağ ray: üç bölüm kartı. CoinDesk'in video/araştırma/klipler modülleri veri kaynağı olmadığı için basılmıyor (bkz. §7A.1). |
| Slider ("Explore more") | Slider'ın dolacağı içerik türü (video/koleksiyon) yok |
| `order-*` mobil dizilim | Yan ray olmadığı için yeniden dizilecek modül yok |
| `display: contents` | Ray sarmalayıcı olmadığı için gereksiz |
| Şerit kaydırma dönüşümü (157×50 → 196×32) | CoinDesk bunu JS ile scroll durumuna bağlıyor; bizde CSP nedeniyle JS'siz ilke var. Şerit zaten tek satır. |
| Google News pill | `google.com/preferences/source` hedefi site yayına alınınca eklenecek |
| Çeviri etiketi ("AI tarafından çevrildi") | EN çeviri hattı gelince; şu an çevrilmiş içerik yok |
| Hisse şablonu (`/price/stock/*`) | ABD hisse verisi bağlanmadı (§14) |
| Dördüncü genişlik kademesi (1432) | Yalnız hisse sayfasında; veri yok |

### 11.2 Dördüncü tur (mega menü + mobil taşma) — DURUM (6 Kasım 2026)

Gerçek Chrome'da `getComputedStyle` + `getBoundingClientRect` ile doğrulandı.

| Konu | CoinDesk (ölçülen) | bitcosar | Durum |
|---|---|---|---|
| Panel genişliği | menü başına sabit: 589/728/750/836/960 | menü başına sınıf: `--589` / `--750` / `--836` | ✅ (960/728 karşılığı içerik yok) |
| Panel gölgesi | üç katmanlı yükselti `0 2/4/12px` | birebir aynı üç katman | ✅ |
| Panel yüzeyi | r 8 · border 1 px · padding 20/24 · bg surface-default | aynı | ✅ |
| Panel başlığı | ikon 32×32 + ad **16/24 w700** + ok svg | aynı (ölçüldü: ikon 32 px) | ✅ |
| Grup başlığı | **14/21 w700** (`font-metadata-lg font-bold`) | aynı (önceki 12/16 w500 yanlıştı) | ✅ |
| Menü satırı | **14/22 w400** | aynı | ✅ |
| "Tümünü gör" | **14/21 w700**, sütun sonunda, üst ayraç | aynı | ✅ |
| **Açıklamalı kart tipi** | 257×96 · p 12/16 · r 8 · başlık 14/20 w500 + açıklama 12/16 w400 subtle | Research panelinde birebir (ölçüldü: 256×96) | ✅ |
| Açılış efekti | **YOK** (transition 0s, animation none, transform none) | aynı — geçiş tamamen kaldırıldı | ✅ |
| Tetikleyici | `<button>` + `aria-expanded` + JS | `<a>` + `aria-haspopup` + CSS `:hover`/`:focus-within` | ✅ **bilinçli sapma** (§14) |
| Sağdaki panel | Sponsored sağa hizalı | nav referanslı `right: 0` (grubun değil, nav'ın sağ kenarı) | ✅ |
| Üst başlık sayısı | 8 (More dahil) | 5 | ✅ **bilinçli sapma** (§14) |

**4. turda bulunan GERÇEK hata — mobilde 190 px yatay taşma.** Üst çubuk
aksiyon şeridi (Ara · dil · Giriş · Ücretsiz üye ol) 390 px'lik görüntü
alanında ne sarılıyor ne katlanıyordu (`clientWidth` 390 / `scrollWidth`
580). Aynı üç bağlantı hamburger panelinde zaten olduğu için üst çubukta
**kopyalar gizlendi**; tema düğmesi kaldı. Doğrulama: taşma **0 px**.

**Ölçüm yöntemi hatası da düzeltildi.** Önceki turda "taşma yok" demiştim;
yanlış ölçüt kullanmıştım — `window.innerWidth` içerikle birlikte büyüyor,
`scrollWidth` ile karşılaştırınca fark görünmüyor. Doğru ölçüt
`documentElement.scrollWidth - clientWidth`.

### 11.3 Beşinci tur (hero rayları + gövde hatası) — DURUM

| Konu | CoinDesk (ölçülen) | bitcosar | Durum |
|---|---|---|---|
| Hero ızgarası | `290px 608px 290px`, gap 24/80, justify center | birebir; sol x 286, sağ x 1344 | ✅ |
| Sol ray modülü | Son Haberler: başlık 14/21 w500, satır 16/24 w500 + saat 12/18 w400 | birebir (5 satır, 290×571) | ✅ |
| Sağ ray modülü | border-top strong + pt 16, başlık 14/21 w500, kart gap 8 | birebir | ✅ |
| Ray kartı tipografisi | başlık 20/26 w500, kicker YOK, iç boşluk 8 | birebir (kicker ray kartında gizli) | ✅ |
| Manşet başlığı | 36/44 **w700** | birebir (önceki 24/30 w500) | ✅ |
| Manşet sırası | metin üstte, görsel altta | birebir | ✅ |
| Video · araştırma · klip modülleri | var | **basılmıyor** — veri 0 (uydurma olmaz) | ⛔ |
| Mobil | tek kolon, sol ray manşetin altında, sağ ray gizli | birebir; taşma 0 | ✅ |
| 1283 sınırı | tek kolon | birebir | ✅ |

**Beşinci turda bulunan iki sessiz hata:** `body { margin: 8px }` sıfırlanmamıştı
(tüm sayfa çerçeveliydi, tam genişlik bant `scrollWidth` 1913 üretiyordu);
`.cd h1…h4` temel kuralı (0,1,1) bileşen sınıflarını eziyordu — kart başlığı
20/24, manşet 28/36 çıkıyordu. İkisi de `:where()` + `html { overflow-x: clip }`
ile düzeltildi; Lighthouse `/` **100**, CLS 0, LCP 0.3 s.

### 11.4 Altıncı tur (bölüm blokları 2×2 + kategori kapsayıcısı) — DURUM

Ana sayfanın alt bölümü **tamamen yanlış düzendeydi**: bizde kategori başına
tam genişlikte bir modül (923 + 461, sütun arası 80) vardı. CoinDesk'te ise
**tek bir 2×2 ızgara** var: 724 + 724, gap 16, her hücre kendi `border-top`
çizgisini taşıyor.

| Konu | CoinDesk (ölçülen) | bitcosar | Durum |
|---|---|---|---|
| Dış düzen | 2×2 ızgara, 724/724, gap 16, alt dolgu 64 | birebir (724/724, gap 16, pb 64) | ✅ |
| Hücre | `border-top` 1 px #a6a6a6, `padding: 20px 20px 0`, gap 16 | birebir | ✅ |
| Hücre başlığı | bağlantı 14/21 w500 + 16 px ok, gap 4 | birebir | ✅ |
| İç ızgara | 334 + 334, gap 16 | birebir (334/334) | ✅ |
| Görselli kart | görsel 16/9 (334×188) + başlık 24/30 w500 + saat 14/21 | birebir | ✅ |
| Sağ satırlar | 3 satır: başlık 18/26 w500 + saat 14/21; aralarında 1 px çizgi, iki yanı 16 px | birebir (`my-4`) | ✅ |
| Kategori sırası | Piyasalar · Politika · Teknoloji · Finans | birebir | ✅ |
| Kapsayıcı kademesi | bölüm sayfası kapsayıcı **1432**, içerik **1384** | 1432 / 1384 (önceki 1384/1336) | ✅ |
| 1024 sınırı | `lg:grid-cols-2` → 1024 ve üstü İKİ kolon | birebir (eşik 1023'e çekildi) | ✅ |
| Mobil | tek kolon; hücre içi de tek kolon (`md` altı) | birebir; taşma 0 | ✅ |

**Bu turda bulunan iki şey:**
1. `.cd-wrap--bolum` jetonu **1384**'e ayarlıydı; bu değer KAPSAYICIYA
   veriliyordu, içerik 1336'ya düşüyordu. CoinDesk ölçümü kapsayıcı 1432 /
   içerik 1384. Düzeltildi (test de 1432'yi doğruluyor).
2. Tailwind `lg` bir ALT sınır: `lg:grid-cols-2` 1024'te İKİ kolon demek.
   `@media (max-width: 1024px)` yazmak tam 1024'te tek kolona düşürüyordu;
   eşik 1023'e çekildi.

Lighthouse `/`: performans **100** · SEO **100** · a11y 97 · BP **100** ·
LCP 0.4 s · CLS 0. CSS bütçesi 48 → **52 KB** (beşinci tur bileşenleri;
gzip'te ~9 KB).

### 11.5 Yedinci tur (promosyon şeridi kaldırıldı) — DURUM

CoinDesk TR ana sayfasında **"Günün kripto özeti her sabah e-postanızda"**
diye bir şerit YOK. Ölçüm (6 Kasım 2026): `main` içindeki ilk şey doğrudan
hero; hero `border-top: 0`, `padding: 32px 0`; önünde promo/duyuru bandı
hiç yok.

Bizde iki fazlalık vardı:

1. `.cd-promo` bandı — y 156, h 52, `margin-top: 16px`, 1 px #d9d9d9
   çerçeve + 3 px sol vurgu. Bu şerit uydurmaydı.
2. Hemen altındaki hero `<section class="cd-mod">` → `border-top:
   1px #a6a6a6`. Kullanıcının gördüğü "kalın çizgi" buydu.

| Konu | CoinDesk (ölçülen) | bitcosar | Durum |
|---|---|---|---|
| Promo şeridi | yok | kaldırıldı | ✅ |
| Hero `border-top` | 0 | 0 (`cd-mod--cizgisiz`) | ✅ |
| Hero `padding` | `32px 0` | `32px 0` | ✅ |
| Hero içeriği | `main` + 32 px | `main` + 32 px | ✅ |

Kaldırılan kod (ölü kontrol bırakılmadı):

- `cd_home.templ` içindeki promo bloğu,
- handler'da `repo.AktifPromo` çağrısı ve `CDHomeView.Promo` alanı,
- `templkit.CDPromo` tipi ve `Gorunur()` yöntemi,
- `.cd-promo*` CSS kuralları (iki blok),
- admin ayarlarındaki **promo** sekmesi ve `promo_*` alanları,
- `repo/promo.go` + `promo_test.go` (artık çağıranı yok).

Aynı turda **§7 "Günlük bülten" formu** da ana sayfadan kaldırıldı:
CoinDesk TR ana sayfasında hiç form yok (yalnız footer'da "Newsletters"
bağlantıları var; ölçümde `form` sayısı 0). `cdNewsletter` bileşeni
`/newsletters` sayfasında kalıyor — orada form gerçekten var.

Doğrulama (1920 px): `promoVar: false`, `news: false`, hero section
`border-top: 0px`, `.cd-hero` `padding: 32px 0`; mobil 390 px taşma **0**.
Lighthouse `/`: performans **100** · SEO **100** · a11y 97 · BP **100** ·
LCP 0.4 s · CLS 0.

### 11.6 Sekizinci tur (Görüş paneli mobilde + ölü menü bağlantıları) — DURUM

AI özet/çeviri hattı kurulurken ortaya çıkan iki **gerçek** hata ölçümle
bulundu (6 Kasım 2026, gerçek Chrome).

| Konu | Ölçülen sorun | Düzeltme | Durum |
|---|---|---|---|
| Görüş paneli, 390 px | panel genişliği **90 px** — başlıklar 2-3 karakterde kırılıyordu (`.cd-ikili__sol` `flex: 0 0 25%` mobilde de uygulanıyordu) | `≤1023px`: `.cd-ikili { flex-direction: column }`, iki sütun `width: 100%` | ✅ 358 px |
| Boş bölüm menüsü | Focus · Videos · Podcasts · News Analysis · Press Releases bölümlerinde **0 yayında içerik**; mega menü + üst gezinme + alt bilgi **11 ölü bağlantı** veriyordu | `repo.DoluBolumler` + `CDShell.MenuBolumler`; satır yalnız yayında içerik varsa basılır | ✅ 0 ölü bağlantı |
| Kaynak görseli | `/kg/…` uçları **404**; günlükte `kaynak görsel indirilemedi … HTTP 429` | `repo.GorselAdresiCoz`: CoinDesk `/_next/image?url=…` proxy'si gerçek `cdn.sanity.io` adresine indirgenir (proxy 429, CDN 200) | ✅ 4xx yok |
| `ai_summary` | yayındaki **0/126** içerikte özet kutusu boştu | AI yazım promptu + `cmd/ozet-doldur` | ✅ dolduruldu |

Hata ölçümleri:

```
Görüş paneli genişliği   390px önce 90px  → sonra 358px   (taşma 0)
/kg 4xx                  2 istek/ana sayfa → 0
focus/videos/... menüde  11 satır          → 0 satır
```

Menü süzgecinin sözleşmesi `internal/webui/templkit/cd_menu_test.go` içinde
üç testle korunuyor: boş bölüm elenir, dolu bölüm kalır, veri yoksa filtre
uygulanmaz.

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
