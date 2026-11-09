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

> **bitcosar UYGULANDI (9 Kasım 2026):** `/price/stock/{sembol}` şablonu
> yazıldı. Veri kaynağı **Yahoo Finance grafik ucu** — BIST için zaten
> kullanılan uç (`query1.finance.yahoo.com/v8/finance/chart/…`); ABD
> hisseleri aynı uçtan gelir, yeni bağımlılık/anahtar YOK.
>
> | Ölçülen alan | Uygulama |
> |---|---|
> | `<h1>` yok, ad küçük | görünür h1 yok; a11y/SEO için `.cd-visually-hidden` h1 |
> | Büyük fiyat 28/42 w500 | aynı |
> | Izgara 392 + 976 (gap 32) | aynı (`--cd-max-section` 1432) |
> | Sekmeler 1D·5D·1M·YTD·1Y·All | aynı; `?aralik=` |
> | Basic / Advanced | Basic alan+çizgi, **Advanced gerçek OHLC mumları** |
> | OHLC okuması | O/H/L basılır (H/L borsa gününden) |
> | Performans satırı 88 px | 24s·5g·1a·YTD·1y — 1 yıllık seriden |
> | **"Key Stats"** | Piyasa değeri · Gün aralığı · 52 hafta · Hacim · Önceki kapanış · Sektör · Çalışan |
> | **"About …"** | ✅ gerçek iş tanımı (326 px sol ray, 14/21) |
>
> **CRUMB AKIŞI (9 Kasım 2026):** Market Cap ve "About" ilk sürümde
> "veri yok" diye atlanmıştı; `v10/quoteSummary` doğrudan 401 "Invalid
> Crumb" veriyordu. Çerez + `/v1/test/getcrumb` akışı eklendi
> (`repo/hisse_profil.go`): `fc.yahoo.com` çerezi → crumb → `?crumb=…`.
> Ölçüldü — META ve THYAO.IS dahil **her iki pazarda** geliyor:
>
> | | META | THYAO.IS |
> |---|---|---|
> | Piyasa değeri | $1.63T | ₺425.41B |
> | Sektör | Communication Services · Internet Content & Information | Industrials · Airlines |
> | Çalışan | 66,775 | 66,549 |
> | İş tanımı | 1.988 karakter | 804 karakter |
>
> Crumb süreç içinde 6 saat önbellekte; 401 alınırsa bir kez yenilenir.
> Profil önbelleği 10 dakika + 2 sn zaman aşımı: profil gelmezse About ve
> piyasa değeri satırları **hiç basılmaz**, sayfa yine açılır.
> İş tanımı kaynak dilinde (İngilizce) basılır ve altında "Kaynak: Yahoo
> Finance" yazar — çeviri hattı gelince çevrilecek, uydurma çeviri yok.

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
| Hero sol/sağ ray modülleri (izleme listesi, Today's Videos, Featured Research, dikey klipler) | **UYGULANDI** — 3 kolon (290·608·290). Sol ray: Son Haberler + canlı yayın modülü + Düşünce Liderliği. Sağ ray: Bugünün Videoları · Öne Çıkan Araştırma · Bugünün Klipleri. Video/klip verisi artık GERÇEK (YouTube kanal senkronu, `youtube_icerik` tablosu); veri yoksa modül basılmaz. |
| Slider ("Explore more") | Slider'ın dolacağı içerik türü (video/koleksiyon) yok |
| `order-*` mobil dizilim | Yan ray olmadığı için yeniden dizilecek modül yok |
| `display: contents` | Ray sarmalayıcı olmadığı için gereksiz |
| Şerit kaydırma dönüşümü (157×50 → 196×32) | CoinDesk bunu JS ile scroll durumuna bağlıyor; bizde CSP nedeniyle JS'siz ilke var. Şerit zaten tek satır. |
| Google News pill | `google.com/preferences/source` hedefi site yayına alınınca eklenecek |
| Hisse keşfedilebilirliği | ✅ `/price` hisse tablosu, `/borsa/hisseler` satır bağlantıları, şerit, mega menü "Hisse Senetleri" sütunu, `/sitemaps/stocks-1.xml` (632 URL), WebPage+FinancialProduct JSON-LD, `/price/stock` → 301 `/price` |
| Çeviri etiketi ("AI tarafından çevrildi") | EN çeviri hattı gelince; şu an çevrilmiş içerik yok |
| Hisse şablonu (`/price/stock/*`) | ✅ **UYGULANDI** (9 Kasım 2026) — Yahoo Finance gerçek verisi (§10.6) |
| Dördüncü genişlik kademesi (1432) | ✅ hisse sayfasında kullanılıyor (`--cd-max-section`) |

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
| Video · araştırma · klip modülleri | var | **UYGULANDI** — YouTube veri hattı bağlandı (`youtube_icerik`, 15 dk senkron); üç modül gerçek videoyla doluyor | ✅ |
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

### 11.7 Dokuzuncu tur (ana sayfa modülleri + iki fontlu tipografi) — DURUM

Tur 8'de iskeleti kurulan ana sayfa bu turda **ölçümle piksel hizalaması**
yapıldı (7 Kasım 2026, gerçek Chrome, 1920×1400, `getComputedStyle`).

#### 11.7.1 Font sistemi — TÜRDEN YANLIŞ KURULMUŞTU

Önceki turlarda CoinDesk'in **tek** fontu olduğu (Neue Haas) ve ikamesinin
Inter Tight olduğu varsayıldı. `document.fonts` + 522 öğenin hesaplanmış
stili ölçüldüğünde tablo başka çıktı:

| Aile | Öğe | Nerede | Boyut/ağırlık örnekleri |
|---|---|---|---|
| **Neue Haas** (sans) | 428 | UI: etiket, saat, ticker, deck, sponsor, menü, düğme | 14/21 w500, 12/18 w400, 10/14 w500 |
| **Family** (serif) | 94 | Editoryal: manşet, kart başlığı, makale gövdesi | 20/26 w500, 18/26 w500, 36/44 w500, **gövde 22/30 w400** |
| DM Mono | 2 | yalnız sayısal rozet | — |

Yani CoinDesk **iki fontlu**: sans UI'da, serif editoryal içerikte. Biz tek
sans fontuyla başlıkları çiziyorduk — bu, tasarımın en görünür sapmasıydı.

Ticari lisanslı oldukları için repoya konamazlar. Aynı 6 metinlik korpusla
(20 px, genişlik + x-yükseklik + cap) ücretsiz adaylar ölçüldü:

| İkame | Genişlik farkı | x-yüksekliği | cap | Karar |
|---|---|---|---|---|
| **Inter** 400/500 | Neue Haas'a **+3,4 % / +2,1 %** | 0,546 vs 0,550 | 0,728 | ✅ sans |
| Inter Tight 400/500 | −7,0 % / −6,8 % (DAR) | 0,546 | 0,728 | ❌ değiştirildi |
| **Alegreya** 400/500/700 | Family'ye **+2,0 / +1,2 / +0,5 %** | **0,452 vs 0,464** | 0,637 vs 0,660 | ✅ serif |
| Crimson Pro | +0,1 % | 0,420 — metni küçük gösteriyor | 0,573 | ❌ cap çok düşük |
| EB Garamond | +0,2 % | 0,400 | 0,650 | ❌ x-yüksekliği düşük |
| Newsreader / Petrona / Source Serif | +9 % … +18 % | 0,426–0,475 | 0,641–0,670 | ❌ genişlik |

Gömülen dosyalar: `inter-400/500/700.woff2` + `alegreya-400/500/700.woff2`
(Türkçe + Latin + para/sembol alt kümesi, **~254 KB** toplam). Lisanslar
`INTER-OFL.txt` / `ALEGREYA-OFL.txt`. Eski `inter-tight-*.woff2` silindi.

`--cd-font` = `"Neue Haas", …, "Inter", system-ui` ·
`--cd-serif` = `"Family", "Alegreya", Georgia, serif`.
Serif sınıfları: `.cd-card__title`, `.cd-list__title`, `.cd-ray__baslik`,
`.cd-dusunce__baslik`, `.cd-tekgorsel__baslik`, `.cd-akis__baslik`,
`.cd-basin__baslik`, `.cd-video__baslik`, `.cd-arastirma__baslik`,
`.cd-arastirma__rapor`, `.cd-hero-lead__baslik`, `.cd-blok__manset`,
`.cd-blok__baslik`, `.cd-opinion__baslik`, `.cd-section-title h1`,
`.cd-article h1`, `.cd-article__body`.

#### 11.7.2 Ölçülen modül geometrisi — 1920 px

| Modül | CoinDesk | bitcosar (tur 9) | Durum |
|---|---|---|---|
| Düşünce Liderliği | 290×306 | 290×**306** | ✅ |
| Bugünün Videoları | 290×360 | 290×**360** | ✅ |
| Öne Çıkan Araştırma | 290×503 | 290×**503** | ✅ |
| Bugünün Klipleri | 290×566 | 290×**566** | ✅ |
| Basın Bültenleri | 608×181 | 608×**181** | ✅ |
| Hero sol ray | 290×1077 | 290×1111 | ⚠️ +34 |
| Görüş satırı / avatar | 74 / 48 | 74 / 48 | ✅ |
| Akış satırı / görsel | 608×102 / 152×86 | 608×102 / 152×86 | ✅ |
| Yatay taşma | 0 | **0** | ✅ |

Hero sol ray farkı **veri kaynaklı**, tasarım değil: CoinDesk'in son haber
satırları 3 satıra kırılıyor, bizdeki Türkçe başlıklar 2 satırda kalıyor;
ayrıca tek görselli manşetin metni kısa. Aynı sayıda satırla geometri aynı.

#### 11.7.3 Bu turda düzeltilen GERÇEK tasarım hataları

| Konu | Ölçülen sorun | Düzeltme |
|---|---|---|
| Araştırma metriği | "50.8%" + ayrı **serif sarı çip** sanıldı; gerçekte metrik + **birimi** tek vurgu span'ında, sarı bant **gradyan** | `.cd-arastirma__metrik` inline + `linear-gradient(transparent 52%, brand 52% … 86%)`; sayı 56/56 w500 ls −1 px, birim 18/26 w500 alt satıra sarar |
| Araştırma kartı | zemin `surface-soft`, radius yok, etiketler **hap** (9999), rapor bağlantısı alt çizgili sans | zemin saydam, radius 4, etiket radius 4 + siyah metin + `max-height 58` kırpma, rapor **serif** 16/22 alt çizgisiz |
| Modül başlığı oku | dört modülde de "→" **karakteri** (13×21, dikey ortalanmıyor, ekran okuyucu "sağ ok" okuyor) | `cdRayOk()`: 16×16 SVG chevron, `currentColor`, `aria-hidden` |
| Başlık türü | Video / Araştırma / Görüş / En Çok Okunan'da **bağlantı + ok**; CoinDesk'te düz `h2` | `.cd-ray__duz` — yalnız "Son Haberler", "Düşünce Liderliği" ve "Bugünün Klipleri" bağlantılı + ok |
| Video kutusu | çerçevesiz; modül 359 (hedef 360) | `border: 1px + radius 4` → 360 |
| Basın modülü | modül sonu çizgisi yok; 180 (hedef 181) | `.cd-basin__cizgi` (1 px #d9d9d9) |
| Basın gezinme | nokta **8 px** / gap 8, oklar "‹ ›" karakteri | nokta **10 px** / gap **16**, oklar 20 px SVG, 32 px yuvarlak |
| Basın sponsoru | `letter-spacing: 0.4px`, 12/18 | `0.06em` (=0.72 px) — CoinDesk ölçümü |
| Düşünce sponsoru | 12/18 (hedef 14/21) | 14/21 → modül 303 → **306** |
| Sol ray listesi | satır arası 16 px, **ayırıcı yok** | 8 px + `border-top` 1 px (ilk satır hariç) |
| Tek görsel manşet | başlık 20/26, meta 12/18, aralık yok | başlık 22/30, meta 14/21, kart `gap: 8px` |
| "Daha fazla" düğmesi | `.cd-btn` 14/20 pad 8×14; CoinDesk 14/21 pad 8×12 | modül içi `.cd-video__alt .cd-btn` eşitlemesi |

Font geçişiyle birlikte **Basın başlığı 3 satırdan 2 satıra** indi
(139 karakter, 608 px): daha önce 78 px yer kaplıyordu, CoinDesk'te 52 px.

#### 11.7.4 Doğrulama

```
go vet ./...      temiz
go test ./...     tüm paketler ok (2 test değeri bilinçli güncellendi:
                  serif font ailesi + CSS bütçesi 52 → 80 KB)
gofmt -l          boş
templ generate    temiz
4xx istek         0
konsol hatası     0
font yükleme      inter-var + alegreya-var → 2/2 HTTP 200, ikisi de "loaded"
1920 taşma        0
390 taşma         0 (sağ ray gizli, sol ray flex)
kaydırma durumu   html.cd-kaydi=true, yön=row, yüzde=block
```

### 11.8 Onuncu tur (gerçek hatalar: kontrast, ARIA, font ağırlığı) — DURUM

Lighthouse 12 ölçümleri bu turda **üç gerçek hata** ortaya çıkardı. Hiçbiri
görsel incelemeyle fark edilmezdi.

| Konu | Ölçülen sorun | Düzeltme |
|---|---|---|
| **Birincil düğme kontrastı** | `.cd a { color: inherit }` özgüllüğü (0,1,1) `.cd-btn`'inkinden (0,1,0) büyük olduğu için üst çubuktaki "Ücretsiz üye ol" düğmesi **siyah zeminde #262626** yazıyla çiziliyordu — kontrast **1,4:1** (AA eşiği 4,5:1). Mobil ölçümde görünmüyordu çünkü o düğme ≤767 px'te gizli | `a.cd-btn { color: var(--cd-bg) }` / `a.cd-btn--ghost { color: var(--cd-fg) }` |
| **ARIA yasağı** | Basın ve video oklarında `<label aria-label="…">` kullanılmıştı; ARIA spesifikasyonunda `label` öğesinde adlandırma **yasak** (`aria-prohibited-attr`, 4 düğüm) | metin `<span class="cd-visually-hidden">` içine alındı; oklar da "‹ ›" karakteri yerine 20/16 px SVG chevron oldu |
| **Font ağırlığı** | Altı sabit ağırlık dosyası (173 KB, 6 istek); LH mobilde 5 font isteği LCP görseliyle yarışıyordu | İki **değişken** font (94 KB, 2 istek): `inter-var.woff2` + `alegreya-var.woff2`, `font-weight: 400 700` |

Bonus: Inter'in `opsz` (optik boyut) ekseni korundu — `font-optical-sizing`
varsayılanı 36 px manşette daha sıkı, 14 px UI metninde daha açık aralık seçer.
Neue Haas da optik boyutlu bir aileydi.

LCP iyileştirmeleri (ölçümle doğrulandı):

| Değişiklik | Etki |
|---|---|
| `<link rel="preload" as="image" imagesrcset imagesizes>` (manşet görseli) | LH "load delay" **942 → 447 ms** |
| Alt katlama görsellerine `fetchpriority="low"` (akış + bölüm blokları) | LCP görseliyle bant genişliği yarışı azaldı |
| Türkçeye daraltılmış alt küme | 254 → 173 KB (sabit), sonra değişken fontla 94 KB |

**Alt küme tuzağı:** ilk daraltmada `U+0100-017F` unutuldu ve **Ğ/Ş/ğ/ş**
glifleri düştü — Türkçe başlıklar bozuk çizilecekti. Glif kapsam testi
yakaladı; aralık eklendi. Bu kontrol (`dar_kapsam.py`) her font üretiminden
sonra koşulmalı.

Lighthouse 12 sonuçları:

| Profil | Perf | A11y | BP | SEO | LCP | CLS | TBT |
|---|---|---|---|---|---|---|---|
| **Masaüstü** (`--preset=desktop`) | **100** | **100** | **100** | **100** | 0,7 s | 0,003 | 0 ms |
| Mobil (simüle 1.6 Mbps) | 94 | **100** | **100** | **100** | 2,9 s | 0,003 | 0 ms |

Mobil performansın 94'te kalması **tasarımın kendisinden**: ana sayfa
CoinDesk'ten devralınan görsel ağırlıklı bir düzen (24 istek, 11 görsel).
Aynı profilde LCP görseli 94 KB JPEG (960w). Gerçek tarayıcıda (kısıtlama
yok) LCP 0,86 s masaüstü / 0,08 s mobil. JPEG kalitesini düşürmek
(q82 → q78) tasarım sadakatini bozacağı için yapılmadı.

### 11.9 Onbirinci tur (`/tr` fark taraması — dil ve yapı) — DURUM

Kullanıcı "coindesk.com/tr ile kalan farklar" dedi; 1920 px'te iki sayfa
yan yana ölçüldü (7 Kasım 2026, gerçek Chrome).

#### 11.9.1 EN BÜYÜK HATA: arayüz İngilizceydi

Türkçe sitede gezinme, bölüm adları, menü ve alt bilgi **İngilizce**
görünüyordu: "News · Prices · Research", "Markets · Policy · Tech ·
Finance", "POLICY" kicker'ı. CoinDesk `/tr` ise "Haberler · Fiyatlar ·
Araştırma", "Piyasalar · Politika · Teknoloji · Finans" kullanıyor.

Kök neden **veritabanı**: `categories.name` alanı 16–30 arası kayıtlarda
İngilizceydi ve bu ad hem bölüm bloğuna, hem makale kicker'ına, hem menüye
yayılıyordu. Migration `0121` adları Türkçeleştirdi (**slug'lar
değişmedi** — adresler kanonik ve indekslenmiş). Ayrıca kodda sabit yazılı
etiketler de çevrildi.

#### 11.9.2 İKİNCİ GERÇEK HATA: Türkçe büyük harf

Bölüm kicker'ı `strings.ToUpper` ile büyütülüyordu; Go yerel ayar bilmediği
için **"POLITIKA"** çıkıyordu (olması gereken "POLİTİKA" — `i → İ`,
`ı → I`). `i18n.BuyukHarf` eklendi; üç çağrı yeri ona bağlandı.

#### 11.9.3 Ölçülen yapısal farklar

| Konu | CoinDesk `/tr` | bitcosar (önce) | Düzeltme |
|---|---|---|---|
| "Canlı Piyasalar" etiketi | **10/15** w500, 82×15, büyük harf DEĞİL | 12/16 + uppercase + 0.4 px, 144×50 | 10/15, uppercase kaldırıldı |
| Canlı noktası | 8 px nokta + 21 px yarı saydam halka | yalnız 8 px nokta | `::after` halka eklendi |
| Ticker sekmesi | kap **gri** zemin + aktif sekme **beyaz** | tersi (aktif gri, kap saydam) | ölçüme çevrildi |
| Ticker öğesi | sembol 12/18, fiyat 14/21, **yüzde rozeti renkli zeminli**, öğeler arası çizgi YOK | köşe yok, `border-left` çizgi, rozetsiz | rozet `--cd-up-bg`/`--cd-down-bg`, çizgi kaldırıldı |
| Sekme etiketi | "Hisse Senetleri" | "Hisse" | düzeltildi |
| Modül düğmesi | "Daha Fazlasını Keşfet" | "Daha fazla" | düzeltildi |
| Sponsor satırı | "Sponsor Hex Trust" | "Sponsorlu: Hex Trust" | iki nokta + "lu" kaldırıldı |
| Üst çubuk düğmeleri | arama ikonu · ay (tema) · TR · **Oturum Aç** · **Kaydol** (sarı) | "Ara" metni · yarım daire · Giriş · siyah "Ücretsiz üye ol" | ikon · ay SVG · Oturum Aç · `--cd-brand` sarı |
| **En Çok Okunan** | sol manşet görselli+spotlu; **sağdaki yedi kart GÖRSELSİZ ve SPOTSUZ** (yalnız 20/26 başlık + 14/21 saat) | küçük kartlarda da görsel + spot → modül **1191 px** | küçük kartlar düz başlık+saat → **470 px** (CoinDesk 561) |
| **Alt bilgi** | **4 sütun**, her sütunda **2–3 bölüm üst üste** (toplam 12 başlık), 942 px | 5 sütun × tek bölüm, 497 px | 4 sütun × çok bölüm, 11 başlık |
| Alt bilgi yasal satırı | solda 10/16 bağlantılar + **sağda 8 sosyal ikon** | yalnız bağlantılar, ikon yok | X · Facebook · LinkedIn · RSS · Instagram · YouTube · Telegram |
| Reklam bandı | dış 366 px (32 dikey dolgu), iç 302 px `#f4f4f4` | 250 px kesikli çerçeve | 302 px, zemin dolgusu, çerçevesiz |

#### 11.9.4 Alt bilgi süzgeci kaldırıldı — gerçek hata

Alt bilgi satırları `MenuBolumler` haritasıyla süzülüyordu; o harita yalnız
**içeriği olan** bölümleri sayıyor. `/hakkimizda`, `/kariyer`, `/iletisim`
gibi **statik sayfalar** haritada olmadığı için elendi: alt bilgi 942 px
yerine 497 px'e düşüyor, sütunlar seyreltik kalıyordu. Liste artık elle
doğrulanmış — 31 rotanın hepsi `curl` ile **200** döndüğü kontrol edildi;
ölü olan 7 yol (blog · erisilebilirlik · reklam · site-haritasi · index ·
api · web-seminerleri) listeye **alınmadı**.

#### 11.9.5 Ölçüm sonrası

```
yatay taşma            0 (1920 ve 390)
sayfa yüksekliği       5341 px (CoinDesk /tr: 5469)
sütun yapısı           3 kolon korunuyor (290 + 608 + 290, gap 80)
modül geometrisi       Düşünce 306 · Video 360 · Araştırma 503 ·
                       Klipler 566 · Basın 181 — hepsi CoinDesk ile aynı
alt bilgi              4 sütun, 11 bölüm başlığı
4xx / konsol           0 / 0
gofmt · vet · test · build   temiz
Lighthouse masaüstü    100 · 100 · 100 · 100 (LCP 0,7 s, CLS 0,001)
Lighthouse mobil       95 · 100 · 100 · 100 (LCP 2,9 s, CLS 0,001)
```

Kalan bilinçli fark: "Kripto Fiyatları" tablosu. CoinDesk `/tr` ana
sayfasında fiyat tablosu **yok**; bizde var çünkü kullanıcı ilk turda
canlı piyasa şeridinin yanında tabloyu da istedi. Bu bir eksik değil,
istenmiş ek modül.

### 11.10 Onikinci tur (modül kaldırma + akış boşluğu) — DURUM

Kullanıcı isteği: Kripto Fiyatları modülünü ve reklam alanını kaldır;
hero manşeti ile haber akışı arasındaki bitişikliği düzelt.

#### 11.10.1 Kaldırılan modüller

| Modül | Önce | Sonra |
|---|---|---|
| Kripto Fiyatları tablosu (§14) | 1464×779, ana sayfanın ortasında | **kaldırıldı** — `cdPriceTable` çağrısı, `Prices` alanı ve `cdFiyatSatirlari` beslemesi söküldü |
| Reklam bandı (§13) | 1920×302 tam genişlik | **kaldırıldı** |

Sayfa yüksekliği 5341 → **4182 px**. Bu iki modül zaten CoinDesk `/tr`
ana sayfasında yoktu; kullanıcı ilk turda istediği için duruyorlardı,
şimdi kaldırıldı.

#### 11.10.2 GERÇEK HATA: manşet akışa yapışıktı

Ölçüm (bizim sayfa): manşet `y=172…742`, akış `y=742…` — **sıfır boşluk**,
arada çizgi yok. CoinDesk'te ise:

```
manşet biter (1047) → 16 px → ÇİZGİ (1063) → 16 px → satır (1080)
→ 86 içerik + 16 alt dolgu → ÇİZGİ (1182) → …
```

Satır adımı **119 px** = 1 (çizgi) + 16 (üst dolgu) + 86 (içerik) +
16 (alt dolgu). Üç ayrı neden vardı:

1. **Orta kolonda aralık yoktu.** `.cd-hero__orta` düz bloktu; CoinDesk
   `flex-col gap-4`. `display:flex; gap:16px` eklendi.
2. **Çizgi yalnız satır aralarındaydı.** Ayrı bir `.cd-akis__ayirici`
   div'i ikinci satırdan itibaren basılıyordu; manşetle ilk satır
   arasında ne çizgi ne boşluk kalıyordu. Çizgi satırın kendi
   `border-top`'una taşındı — artık her satırın üstünde var.
3. **Alt dolgu unutulmuştu.** Satır 103 px kalıyordu (86 + 16 + 1);
   `padding: 16px 0` ile 119 px'e çıktı.

`.cd-basin` üzerindeki `margin-top: 16px` kaldırıldı: orta kolonun `gap`'i
zaten 16 px veriyor, ikisi birden 32 px yapacaktı.

#### 11.10.3 Doğrulama

```
orta kolon            gap 16 px, akış ilk satırı manşetten 16 px sonra
akış satır adımı      119 px (CoinDesk 119) — çizgi + 16 + 86 + 16
akış ilk satırında    1 px border-top VAR (manşeti ayıran çizgi)
yatay taşma           0 (1920 ve 390)
kripto fiyat tablosu  YOK · reklam bandı YOK (masaüstü ve mobilde doğrulandı)
4xx / konsol          0 / 0
gofmt · vet · test · build   temiz
Lighthouse masaüstü   100 · 100 · 100 · 100 (LCP 0,7 s, CLS 0,003)
Lighthouse mobil      94 · 100 · 100 · 100 (LCP 2,9 s, CLS 0,001)
```

### 11.11 Onüçüncü tur (fiyat şeridi + üst çubuk + koyu tema zeminleri) — DURUM

Kullanıcı üç şey istedi: fiyat şeridi CoinDesk gibi olsun (hover, renkli
yüzde rozetleri, ayırıcılar, Kaydol hizası, kaydırma yüksekliği, alt
bilgide kalma), üst çubuk sağ kümesi birebir olsun (Kaydol · Oturum Aç ·
dil · tema · arama, aralarında çizgi) ve koyu tema zeminleri düzelsin.

#### 11.11.1 Fiyat şeridi — ölçülen değerler (CoinDesk `/tr`, 1920 px)

| Konu | CoinDesk | bitcosar (önce) | Şimdi |
|---|---|---|---|
| Kapsayıcı | `justify-between` + `gap: 12px`, 1191×50 | `gap: 0` + `margin-right` | `justify-between` + `gap: 12px` |
| Kalem arası | **1×32 px dikey çizgi** (`role="separator"`, `border-default`) | yok, yalnız boşluk | `.cd-ticker__ayirici` 1×32 `#d9d9d9` |
| Yüzde rozeti | 60×22, radius 4, `padding: 0 6px`, **renkli zemin** (`#ffe5e5` / `#e5ffe0`) | zeminsiz, 18 px | `min-height: 22px` + renkli zemin |
| Kalem | 157×50, `padding: 4px 8px`, radius 4 | 152×50 | 157×50 |
| Sağ hiza | son kalem sağ kenarı **1692 = Kaydol sağ kenarı** | 1449 | **1692 = 1692** |
| Kaydırılmış şerit | 46 px (45 + 1 çizgi), kalem **32 px**, satıra SABİT 45 px | 49 px, kalem 36 px (esniyordu) | 46 px, kalem 32 px |
| Alt bilgide | şerit görünür kalır (`position: sticky`) | kayboluyordu | kalıyor (y=73, en altta doğrulandı) |
| Hover | kalemde zemin değişimi (çerçeve değil) | yok | `--cd-surface-soft`, 120 ms |

Kaydırılmış yükseklik hatasının kökü: satıra yalnız `min-height: 45px`
verilmişti ve `align-items: stretch` kalemleri 36 px'e şişiriyordu.
CoinDesk satıra **sabit 45 px** verip çocukları `center` ile hizalıyor.

#### 11.11.2 Üst çubuk sağ kümesi — ölçülen değerler

| Öğe | CoinDesk | bitcosar |
|---|---|---|
| Küme | 334×40, `gap: 8px` | 369×40 (arama ikonu fazla), `gap: 8px` |
| Arama / tema düğmesi | 36×36, radius 6, `padding: 8px`, ikon 20×20 | 36×36, radius 6, `padding: 8px`, ikon 20×20 |
| Dil kutusu | 75×40, `padding: 0 16px`, küre 20×20 + kod, `gap: 4px` | 74×40, aynı |
| Dil ↔ Oturum çizgisi | **1×40 px `#d9d9d9`**, `role="separator"` | 1×40, aynı |
| Oturum Aç | 104×40, `padding: 0 16px`, 14/21 w500, şeffaf | 104×40, aynı |
| Kaydol | 77×40, `#ffd200`, siyah metin, hover `opacity: .9` | 78×40, aynı |
| Sağ kenar | 1692 | 1692 |

#### 11.11.3 Koyu tema zeminleri

| Yüzey | Önce | Şimdi |
|---|---|---|
| Gövde (`.cd`) | `#0a0b0e` | **`#161616`** (spec §3 `bg-surface-default`) |
| Fiyat şeridi | `#0a0b0e` | **`#161616`** (ayrı jeton, gövdeden bağımsız) |
| Üst çubuk | `#0a0b0e` | **`#000000`** (`--cd-topbar-bg`, açık temada `#ffffff`) |
| Mega menü | açık tema jetonları | değişmedi — koyu temada **açık** kalıyor |

#### 11.11.4 Doğrulama

```
yatay taşma            0 (1920 · 768 · 390)
şerit yüksekliği       67 px üstte / 46 px kaydırılmış (CoinDesk 67 / 46)
rozet                  22 px, zeminli (açık #ffe5e5, koyu #3a1214)
son kalem sağ kenarı   1692 = Kaydol sağ kenarı 1692
üst çubuk              tema ile dönüyor (açık #fff, koyu #000)
gofmt · vet · test · build   temiz
Lighthouse masaüstü    100 · 100 · 100 · 100 (LCP 0,7 s, CLS 0,005)
```

Bilinçli fark: arama ikonu. CoinDesk'in **Türkçe** masaüstü üst
çubuğunda arama düğmesi **0×0** (boş `aria-label="Ara"` yuvası); İngilizce
sayfada ise 200×40 hap var. On beşinci turda İngilizce desen uygulandı —
bkz. §11.12.

### 11.12 On dördüncü tur (tema simgesi + arama hapı ve paneli) — DURUM

Kullanıcı iki hata bildirdi: (1) açık/koyu tema geçişinde simge
değişmiyordu, (2) arama kutusu CoinDesk gibi değildi — sayfa değiştirip
ayrı bir sonuç sayfası açıyordu. İkisi de düzeltildi.

**Referans sayfa:** `coindesk.com/` (İngilizce). `coindesk.com/tr`
üst çubuğunda arama yuvası boş (0×0); hap + panel deseni İngilizce
sayfada ölçüldü (7 Kasım 2026, 1920 px).

#### 11.12.1 Tema anahtarı

| Konu | CoinDesk | bitcosar (önce) | Şimdi |
|---|---|---|---|
| Açık tema simgesi | **ay** (20×20) | ay | ay (`.cd-tema__ay`) |
| Koyu tema simgesi | **güneş** (20×20) | ay (değişmiyordu) | güneş (`.cd-tema__gunes`) |
| Görünürlük | tema sınıfı | JS simge değiştirmiyordu | CSS: `html.dark` ile takas |
| `aria-label` | "Karanlık moda geç" / "Aydınlık moda geç" | "Koyu/Açık temaya geç" | CoinDesk metinleri |

İki SVG de DOM'da; ilk boyamada doğru simge çizilir (FOUC yok).

#### 11.12.2 Arama hapı — ölçülen değerler

| Konu | CoinDesk | bitcosar |
|---|---|---|
| Kutu | **200×40**, köşe 9999 | 200×40, köşe 9999 |
| Zemin | `#f4f4f4` (surface-subtle) | `--cd-surface-soft` |
| Dolgu / aralık | `0 10px` / `gap: 4px` | aynı |
| Büyüteç | 16×16 | 16×16 |
| Metin | "Search" 14/22 w500 `#676767` | "Ara" 14/22 w500 `--cd-muted` |
| "/" ipucu | 24×24 daire, 1 px çerçeve, beyaz zemin | aynı |
| Sağ kenar | 1692 | 1692 |
| Küme | 533×40 (hap dahil) | 533×40 |

Orta genişlikte (768–1279) hap 40×40 ikon kutusuna iner; mobilde
(≤767) gizlenir — hamburger panelinde "Ara" bağlantısı var.
JS kapalıysa hap `/arama` sayfasına gider (bağlantı olarak yazıldı).

#### 11.12.3 Arama paneli — ölçülen değerler

| Konu | CoinDesk | bitcosar |
|---|---|---|
| Panel | **560 px**, üstten **64 px**, köşe 16 | aynı |
| Gölge | `shadow-lg` | `--cd-shadow-overlay` (tasarım kuralı) |
| Başlık | 64 px: 24 büyüteç + 16/24 girdi + 32×32 kapat | aynı |
| Alt çizgi | 1 px | `--cd-border` |
| Çipler | 28 px, 9999 köşe, 12 px yatay dolgu | aynı |
| Gövde | kaydırılır | `overflow-y: auto` |
| Perde | `rgba(0,0,0,.5)` | aynı |
| Davranış | tıkla aç, Esc kapat, Ctrl+K / "/" | aynı + çip süzgeci |

Panel `hidden` gelir; form doğrudan `GET /arama?q=…` adresine gider.
Açma/kapama, çip süzgeci ve kısayollar `csp-delegated.js`'te
(CSP `unsafe-inline` yok). Panelde **yalnız gerçek veri** var: şeritten
gelen 7 fiyat ve gezinmedeki bölüm bağlantıları; haber sonucu verisi
olmadığı için uydurulmadı.

#### 11.12.4 Doğrulama

```
yatay taşma            0 (1920 · 768 · 390)
hap                    200×40, sağ kenar 1692 = Kaydol 1692
tema simgesi           açık: ay / koyu: güneş (display block/none)
panel                  560×758, x=680, y=64, radius 16, başlık 64
form                   GET /arama?q=bitcoin → 200
Esc / "/" / Ctrl+K     çalışıyor; çip süzgeci grupları gizliyor
gofmt · vet · test · build   temiz
Lighthouse masaüstü    100 · 100 · 100 · 100 (LCP 0,7 s, CLS 0,007)
```

CSS bütçesi 92 → **100 KB** (`TestCDButce`): hap + panel + çift tema
simgesi eklendi; gzip'te ~14 KB.

---

### 11.13 On beşinci tur (SEO katmanı — CoinDesk kaynak koduyla birebir) — DURUM

Kullanıcı isteği: "CoinDesk'in tüm sayfa kaynak kodlarını incele; SEO ile
ilgili bizim yapmadığımız HER şeyi aynı desen ve yapıda kur."

**Yöntem:** gerçek Chrome (Puppeteer, gerçek UA, 1920 px) ile 7 sayfa tipi
açıldı, `page.content()` + `document.querySelectorAll('script[type=ld+json]')`
ile ham kaynak toplandı: ana sayfa, bölüm (`/policy`), makale
(`/policy/2026/11/06/…`), etiket (`/tag/tokenization`), yazar
(`/author/krisztian-sandor`), varlık (`/price/bitcoin`), ayrıca
`/robots.txt`, `/sitemap-index.xml`, `/sitemaps/static-1.xml`,
`/sitemaps/articles-1.xml`, `/sitemaps/crypto-1.xml`,
`/arc/outboundfeeds/news-sitemap-index`, `/arc/outboundfeeds/rss`,
`/manifest.webmanifest`.

#### 11.13.1 Head meta seti (CoinDesk → bizde)

| Alan | CoinDesk | bitcosar |
|---|---|---|
| `googlebot` | `index, follow, max-video-preview:-1, max-image-preview:large, max-snippet:-1` | **aynı** (noindex sayfada yerine `robots noindex,follow`) |
| `page_url` | kanonik adres | **aynı** |
| `media_type` | `html` | **aynı** |
| `page_category` | homepage · section_landing_page · article_page · tag_page · author_page · price_page | **aynı** değerler |
| `content_category` | homepage · policy · market · tag · author · price | **aynı** (bölüm slug'ı) |
| `content_language` | `en` | `tr` |
| `og:locale` | `en` | `tr_TR` |
| `og:image:width/height` | 1200 / 675 | **aynı** |
| `og:image:alt` · `twitter:image:alt` | var (makale) | **aynı** (görsel kredisi varsa) |
| `twitter:image:width/height` | 1200 / 675 | **aynı** |
| `apple-itunes-app` | var | **YOK** — mobil uygulamamız yok (uydurma kimlik yazılmaz) |
| `fb:app_id` | var | **YOK** — Facebook uygulaması yok |
| `twitter:site` / `creator` | `@coindesk` | **YOK** — X hesabı yok |
| `theme-color` | tek değer `rgb(255,255,255)` | iki media sorgusu: açık `#ffffff`, koyu `#161616` |
| hreflang | 15 dil + x-default | yalnız **gerçek** çeviriler + x-default (şu an tr) |
| `link rel=hub` | pubsubhubbub | **aynı** |
| `link rel=self` | RSS | **aynı** |
| ikonlar | ico + svg + 16/32/48/64/192/512 + 5 apple-touch | 192 + svg + 512 + apple-touch |
| origin-trial meta'ları | 5 adet (reklam) | **YOK** — reklam ağı yok |
| Arc/Parse.ly meta seti (makale) | `content_*`, `authors`, `author_1`, `tags`, `tag_N`, `publish_*`, `create_*`, `display_*`, `reg_enabled`, `sponsored`, `featured_media_*`, `language_selection`, `parsely-*` | **aynı anahtarlar**, yalnız dolu olanlar |

#### 11.13.2 JSON-LD — sayfa tipi → tip (ölçüldü)

| Sayfa | CoinDesk | bitcosar |
|---|---|---|
| Ana sayfa | JSON-LD **yok** | Organization + WebSite (bilinçli ek: site bağlantıları + arama kutusu) |
| Bölüm | `CollectionPage` + `mainEntity.ItemList` (11 `NewsArticle`) | **aynı** (11 öğe) |
| Makale | `NewsArticle` **tek nesne** (`@graph` yok): `inLanguage, url, identifier, publisher, headline, abstract, mainEntityOfPage, thumbnailUrl, image, datePublished, dateModified, articleSection, author[Person], keywords` | **aynı alanlar** + ayrı `BreadcrumbList` bloğu (Google SERP yolu; görünür kırıntı yok) |
| Etiket | JSON-LD yok | `CollectionPage` (eksik bırakmak yerine doğru tip) |
| Yazar | `WebPage` + `mainEntity.Person` (`jobTitle, worksFor, description, image, sameAs`) | **aynı** (boş alan basılmaz) |
| `/price` | — | WebSite + `BreadcrumbList` + `ItemList` |
| `/price/{coin}` | `[WebPage, FinancialProduct]` + `BreadcrumbList` + `FAQPage` + `ItemList` | **aynı** eksi `FAQPage` (soru-cevap metnimiz yok; uydurma FAQ spam sayılır) ve eksi `Volume-to-Market-Cap Ratio` (hesaplanmıyor) |

Makale `author` bizde `Person{name,url}`; CoinDesk ek olarak `jobTitle`,
`description`, `image`, `sameAs` basıyor — bu alanlar veritabanında
doğrulanmış biyografi/sosyal hesap olarak yoksa **basılmaz**.

#### 11.13.3 robots.txt

CoinDesk sırası birebir: genel `Allow: /` + API/dahili/arama yolları kapalı
→ **AI tarayıcı bloğu** (22 bot; `Allow: /price/` + `Disallow: /`)
→ sitemap listesi. Bizde kapatılan yollar: `/api/`, `/admin/`,
`/internal/`, `/hits/`, `/webhooks/`, `/arama?q=`, `/?q=`, `/?s=`,
`/giris`, `/kayit`, `/arsiv/dizin`, `/gizli-dizin`, `/*/amp`.

#### 11.13.4 Sitemap ailesi

| Yol | İçerik | CoinDesk parametreleri |
|---|---|---|
| `/sitemap-index.xml` | `static-1` + `articles-1…N` + `crypto-1` | aynı |
| `/sitemaps/static-1.xml` | sabit sayfalar + dolu bölümler + dolu kategoriler | `daily` / `1` |
| `/sitemaps/articles-N.xml` | 1000'lik dilimler | `lastmod` ISO-8601 + `yearly` / `1` |
| `/sitemaps/crypto-1.xml` | `/price` + her varlık | `hourly` / `1` |
| `/news-sitemap-index` | haber sitemap indeksi | aynı |
| `/sitemap-news.xml` | son 48 saat | `lastmod` + `news:news` + `image:image` + CDATA başlık |
| `/sitemap.xml` | **korundu** (BİK resmî ilanları + hreflang) | — |

`articles-N.xml` içinde hreflang alternates **yok** — CoinDesk kaynak
dosyasında da yok; sayfa düzeyindeki canonical/x-default doğru adresi veriyor.

#### 11.13.5 RSS 2.0

CoinDesk kanal alanları: `title, link, atom:link self, atom:link hub,
description, lastBuildDate, language, copyright, generator, ttl=5,
sy:updatePeriod, sy:updateFrequency, image`. Öğe alanları: CDATA başlık,
`media:content` (görsel), `guid isPermaLink="false"` (içerik kimliği —
slug değişse de okuyucu aynı öğeyi tekrar göstermez), `pubDate`,
CDATA `description`, `dc:creator`, boş `content:encoded` / `dc:description`,
`category` (bölüm + etiket + `News`). Etiketler **tek sorguda** gelir
(`repo.EtiketlerCoklu`, `= ANY($1)`).

#### 11.13.6 manifest.webmanifest

`name/short_name` sunucudaki `site_baslik` ayarından **dinamik** okunur
(iki yerde farklı ad kalmasın); `start_url`, `scope`, 4 ikon (any +
maskable), `shortcuts` (Son Haberler · Piyasalar · Fiyatlar),
`theme_color`/`background_color` `#ffffff`, `lang: tr`.

#### 11.13.7 Doğrulama (7 Kasım 2026, gerçek sunucu)

```
JSON-LD blok sayısı     makale 2 (NewsArticle + BreadcrumbList)
                        bölüm 1 (CollectionPage)
                        fiyat 3 (WebSite + Breadcrumb + ItemList)
                        varlık 3 ([WebPage,FinancialProduct] + Breadcrumb + ItemList)
                        etiket 1 (CollectionPage) · yazar 1 (WebPage/Person)
page_category           homepage · section_landing_page · article_page
                        tag_page · price_page (ölçüldü)
og:locale               tr_TR (tüm tiplerde)
og:image                MUTLAK (https://…/kg/…); önceden göreliydi → X/Facebook
                        kartı görselsiz çıkıyordu. JSON-LD thumbnailUrl de mutlak.
og:image:width/height   1200 / 675
hreflang (makale)       tr + gerçek çeviri (en) + x-default; çeviri yoksa yalnız x-default
robots.txt              200; 22 AI botu + 4 sitemap satırı
sitemap-index           200; static-1 + articles-1 + crypto-1 (133 TR içerik)
news-sitemap-index      200; → /sitemap-news.xml
sitemap-news            lastmod + CDATA + image:loc (mutlak)
rss                     dc:creator + media:content + guid + category + ttl
manifest                name/short_name = BitCoŞaR
yatay taşma             0 (1920 · 768 · 390; 6 sayfa tipi)
Lighthouse masaüstü     performance/accessibility/best-practices/seo = 100
gofmt · vet · build     temiz
```

**Kapak görseli kuralı:** `image_url` (kaynağın görseli) → yoksa
`cover_media_id` (editörün yüklediği kapak) → yoksa **görsel basılmaz**.
Uydurma/yer tutucu görsel Open Graph'ta yanlış izlenim verir.

---

### 11.14 On altıncı tur (klip slider'ı + hesap merkezi + kamu sayfaları) — DURUM

İki kullanıcı isteği: (a) ana sayfadaki video modülü CoinDesk'te slider
olmalı, bizde "3 tane alt alta" basılıyor; (b) `/hesabim` CoinDesk tasarımına
geçmeli ve theblock'tan kalan eski tasarım yerleri yeni tasarıma taşınmalı.

#### 11.14.1 Bugünün Klipleri — ölçülenle birebir

**Sorun (ölçüldü):** sağ rayda `.cd-klipler` **1621 px** yükseklikteydi;
3 adet 9:16 kart alt alta basılıyordu (her kart ~516 + 12 boşluk).

**CoinDesk ölçümü** (coindesk.com, 1920 px, gerçek Chrome, 7 Kasım 2026):

| Öğe | Ölçü |
|---|---|
| section `aria-label="Today's Clips"` | 290×566, `gap:12`, 1 px üst çizgi, 16 px üst dolgu |
| başlık bağı | 107×21, 14/21 w500 |
| kart | **290×516** = 9/16, köşe 8, zemin `#000` |
| dikey oklar | **kartın SAĞ kenarında**, 24 px yuvarlak, yarı saydam beyaz çerçeve, dikeyde ortalı |
| başlık | kartın **ÜSTÜNDE** beyaz overlay (siyah geçişli zemin), 18/24 w500, 3 satır |
| oynat düğmesi | sol altta 40 px |

**Uygulama:** slider yine **JS'siz** — radyo + `label` deseni (video ve
araştırma modülleriyle aynı). Yukarı ok = önceki, aşağı ok = sonraki;
dikey slider yönü CoinDesk'ten alındı.

**Yeni ölçüm (bitcosar):** bölüm **290×566**, kart **290×516** (oran
1,778), görünen kart sayısı **1**, oklar 24×24 sağ kenarda.

#### 11.14.2 Hesap merkezi (`/hesabim` + 8 sekme)

**Sorun (ölçüldü):** gövde yazı tipi Poppins (179 elemanda), üst çubuk
`rgb(209,0,21)` (theblock kırmızısı), 12 px köşeli kartlar, kırmızı alt bilgi.

**Tasarım kaynağı — uydurma yok:** CoinDesk'in hesap alanı Auth0
barındırmalı (`auth.coindesk.com/u/login`) ve girişsiz ölçülemiyor. Bu
yüzden geometri tahmin edilmedi; sitenin **kendi ölçülmüş jetonları**
kullanıldı (spec §5, §4). Yapı CoinDesk'in hesap deseniyle aynı:
sol kenar menü + sağ içerik.

| Öğe | Değer |
|---|---|
| ızgara | `240px minmax(0,1fr)`, gap 40 |
| kenar menü | yapışkan, aktif satır **2 px siyah sol çizgi** |
| kart | 1 px `#d9d9d9`, köşe 4, gölge yok |
| başlık | serif 32/40 w700 |
| ikincil metin | 16/24 `#676767` |
| birincil düğme | siyah zemin / beyaz metin, köşe 4 |
| plan rozeti | `#ffd200` zemin, siyah 11/16 büyük harf |
| koyu tema | zemin `#161616`, topbar `#000000` |

Sekmeler: hub · profil · abonelik · kaydedilenler · yorumlar · takip ·
bildirimler · rozetler. Tümü `noindex` (hesap sayfaları dizine girmemeli).

**Yeni ölçüm:** 8 sekmede Poppins **0**, `cd-topbar`/`cd-footer` var,
yatay taşma **0** (1920 px).

#### 11.14.3 Kamu sayfaları — ortak kabuk çevrildi

**Kapsam (19 rota × 2 tema ölçüldü):** `/giris`, `/kayit`, `/sifremi-unuttum`,
`/abonelik`, `/bulten`, `/yazarlar`, `/arsiv`, `/arama`, `/etkinlikler`,
`/anketler`, `/hava-durumu`, `/namaz-vakitleri`, `/imsakiye`,
`/nobetci-eczaneler`, `/borsa`, `/fikstur`, `/astroloji`, `/resmi-ilanlar`,
`/gazete-mansetleri`, `/sinemalar`, 404.

**Yöntem:** 30 şablonu tek tek yeniden yazmak yerine **ortak kabuk**
çevrildi (`go/internal/webui/static/cd-public.css`): jetonlar, tipografi,
üst çubuk, alt bilgi, düğmeler, formlar, kartlar. Sayfa gövdeleri kendi
sınıflarını koruyor; tek noktada düzeltme → otuz sayfada tutarlılık.

| Öğe | theblock | CoinDesk |
|---|---|---|
| marka rengi | `#d10015` kırmızı | siyah `#000` + sarı `#ffd200` vurgu |
| üst çubuk | kırmızı zemin, beyaz menü | **beyaz**, menü `#262626` (koyuda `#000`/`#f4f4f4`) |
| gövde zemini | `#f3f3f3` + ağ deseni | **`#ffffff`** düz (koyuda `#161616`) |
| gövde fontu | Poppins | **Inter** (sans) |
| başlık fontu | Poppins | **Alegreya** (serif) |
| köşe | 8 px | **4 px** |
| gölge | `0 1px 3px` | **yok** (ayırım 1 px çizgi) |
| alt bilgi | kırmızı şerit + `#18181b` | `#000` şerit + **`#161616`** |
| logo | metin "the block" | **marka görseli** (1x/2x/3x, iki tema varyantı) |

Giriş/kayıt/şifre sıfırlama sayfaları ayrı kabuk kullanıyor
(`uye-giris.css`); düzen (iki sütunlu illüstrasyon + form) **korundu**,
yalnız görsel dil çevrildi (koyu bordo zemin → beyaz, 28 px → 4 px,
kırmızı → siyah birincil düğme).

#### 11.14.4 Bu turda bulunan GERÇEK hatalar

1. **Döngüsel CSS değişkeni.** `--hp-header-bg: var(--hp-header-bg)` —
   CSS kendine çözümlenemeyen değişkeni geçersiz sayar; `.header-skin-accent`
   içindeki kırmızı kazandı (headerBg `rgb(209,0,21)`). Açık değer yazıldı.
2. **Yükleme sırası.** `cd-public.css` panelin `DesignCSS` `<style>`
   bloğundan ÖNCE geliyordu; satır içi blok aynı özgüllükte ve SONRA
   geldiği için jetonları geri yazıyordu. Bağlantı blokların altına taşındı.
3. **Özgüllük tuzağı (logo, iki yerde).** `.site-header .brand-img img`
   ve `.auth-brand img` kuralları (0,2,1 / 0,1,1) `display:block` veriyor;
   tek sınıflı gizleme (0,1,0) kaybediyor → **iki logo yan yana** basıldı.
   Gizleme kuralları aynı özgüllükle yazıldı.
4. **Koyu temada kontrast 1:1.** `.abon-toggle-btn.is-active`
   `background: var(--hp-text)` + `color:#fff`; koyu temada `--hp-text`
   beyaz olduğu için "Aylık" düğmesi görünmezdi. Jeton ters çevrildi →
   **21:1** (iki temada da ölçüldü).
5. **Sabit renk, tema değişince kayboldu.** `.auth-hint` `#cfc8d2`,
   `.auth-hint__note` `#8f8995` — koyu zemin için seçilmiş açık griler;
   beyaz zeminde ~1.4:1. Tema jetonuna bağlandı.
6. **Çift kaçış.** Alt bilgide `"Piyasa &amp; Borsa"` etiketi HTML
   kaçışıyla birlikte metne yazılmıştı → ekranda `Piyasa &amp; Borsa`.
   `"Piyasa ve Borsa"` yapıldı.
7. **Dekoratif başlık emojileri.** `☀ 🕌 📅 🏥 📈 🔮 📍 🔍 📬 🎧` başlık ve
   menüden kaldırıldı (CoinDesk hiç kullanmıyor). **İşlevsel işaretler
   (`✓ ✕ ❤`) dokunulmadı** — ilk denemede hepsi silinince
   `TestAIEditorResultRendersQuoteSources` kırıldı; temizlik
   `page-title` ve menü çekmecesiyle sınırlandırıldı.

#### 11.14.5 Doğrulama

```
klip modülü            290×566 bölüm · 290×516 kart (1,778) · görünen 1 · oklar 24×24
hesap merkezi          8 sekmede Poppins 0 · cd-topbar/cd-footer var · taşma 0
kamu sayfaları         19 rota × 2 tema: Poppins 0 · taşma 0 · emoji 0
                       headerBg #fff / #000 · footerBg #161616
abonelik segment       kontrast 21:1 (açık ve koyu)
JSON-LD                ana sayfa 1 · bölüm 1 · makale 2 · fiyat 3 · etiket 1 · yazar 1
taşma                  0 (1920 · 768 · 390)
Lighthouse masaüstü    100 · 100 · 100 · 100 (LCP 0,7 s, CLS 0,006)
gofmt · vet · test · build   temiz (HEAD ayrı worktree'de doğrulandı)
```

---

### 11.15 On yedinci tur (makale üst boşluğu + özet kutusu) — DURUM

Kullanıcı bildirimi (yazı sayfası): "yukarıdaki etiket menüden boşluk aynı
değil" ve "Summary/Özet alanı çerçeve rengi aynı değil".

**Ölçüm kaynağı:** `coindesk.com/markets/2026/11/05/the-crypto-industry-…`,
gerçek Chrome, 1920 px, 8 Kasım 2026 (`getComputedStyle` + `outerHTML`).

#### 11.15.1 Üst boşluk ve kicker

| Öğe | CoinDesk | bitcosar (önce) | bitcosar (şimdi) |
|---|---|---|---|
| şerit alt → kicker üst | **73 px** | **0 px** (yapışık) | 73 px |
| kicker alt → H1 üst | **2 px** | 8 px | 2 px |
| H1 alt → deck üst | **8 px** | 12 px | 8 px |
| kicker biçimi | `text-transform: none` ("Markets"), 16/32 w500 | `uppercase` + `i18n.BuyukHarf` ("POLİTİKA") | `none`, 16/32 w500 |
| kart kicker'ı | **hiç yok** | "POLİTİKA" 12/16 | "Politika" 12/16 |

73 px'in kaynağı iki katmanlı dolgu: içerik kapsayıcısı 24 px + ızgara
32 px (+ şeridin kendi alt dolgusu). Katmanları tek tek kurmak kırılgan
olduğu için `.cd-main--makale { padding-top: 73px }` olarak sabitlendi.

**İKİ KATMANLI HATA:** CSS'te `text-transform: none` yapmak yetmedi —
metnin **kendisi** sunucuda `i18n.BuyukHarf` ile çevriliyordu. Ölçüm bunu
yakaladı: computed değer `none` ama `textContent` "POLİTİKA". Bu yüzden
`cd_article.go`, `cd_home.go` ve `cd_section.go`'daki üç çağrı da
kaldırıldı.

#### 11.15.2 Özet kutusu — çerçeve gradyan

CoinDesk'te çerçeve **düz çizgi değil, marka gradyanı**:

| Öğe | Değer |
|---|---|
| dış kap | 680×116 · radius **7** · padding **2** · margin 16 0 32 |
| zemin | `linear-gradient(102deg, #FFD200 -0.2%, #18E589 171.06%)` |
| iç kutu | 676×112 · radius **5** · padding **20 20 12** · zemin beyaz |
| başlık satırı | 15×15 **sparkle ikon** + "Summary" 16/24 **w500** solda; sağda "Show" + chevron, mb 16 |
| önizleme | **TEK satır + ellipsis** (`truncate`); alt solma gradyanı **yok** |
| chevron | açılınca **180°** döner |

Bizde 1 px `--cd-border` çizgisi vardı → kullanıcı bildirimi haklı.

#### 11.15.3 Bulunan GERÇEK yapı hatası

`<summary>` bir `<div class="cd-ozet__ic">` **içine** konmuştu. HTML'de
`<summary>`, `<details>`in **doğrudan çocuğu** olmak zorunda; tarayıcı
geçersiz summary'yi yok sayıp yerine **"Details"** yazıyordu (ekran
görüntüsünde görüldü). Gövde artık summary'nin **kardeşi**;
`.cd-ozet__govde` beyaz zemini sürdürüp alt köşeleri yuvarlıyor.

#### 11.15.4 Doğrulama

```
şerit → kicker         73 px
kicker → H1            2 px
H1 → deck              8 px
kicker metni           "Politika" · text-transform: none
kart kicker            "Politika" · text-transform: none
özet kutusu            680 genişlik · radius 7 · padding 2 · border 0
                       background-image: linear-gradient(102deg, …)
özet açık              chevron matrix(-1,0,0,-1,0,0) · gövde görünür
"Details" metni        YOK
yatay taşma            0
gofmt · vet · test · build   temiz (HEAD ayrı worktree'de doğrulandı)
```

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

### 11.16 On sekizinci tur ("Explore more" şeridi + bölüm panelleri) — DURUM

CoinDesk ana sayfasındaki **"Explore more from CoinDesk"** şeridi kuruldu.
Ölçüm: gerçek Chrome 1920 px, `getComputedStyle`, 8 Kasım 2026.

**Yerleşim sırası (CoinDesk'ten ölçüldü):** hero → Explore → Görüş +
En Çok Okunan → bölüm blokları. Yani şerit "en çok okunanlar"ın **üstünde**.

| Öğe | CoinDesk (ölçüldü) | bitcosar |
|---|---|---|
| kapsayıcı | 1464×672, `overflow: hidden` | 1464×672 ✅ |
| başlık | 220×24 · 16/24 w500 | 16/24 w500 ✅ |
| başlık → şerit | 16 px | 16 px ✅ |
| şerit | 1464×632 · flex · gap 16 · `padding: 0 16` | aynı ✅ |
| kaydırma | `overflow-x:auto` + `snap-x mandatory` + çubuk gizli | aynı ✅ |
| panel | 400×632 · `flex-shrink:0` · `scroll-snap-align:start` | 400×632 ✅ |
| kart | radius 8 · zemin `--cd-bg` · **dört kenar** 1 px `#d9d9d9` | aynı ✅ |
| iç dolgu | 16 · flex column · yükseklik %100 | aynı ✅ |
| panel başlığı | 366×32 · gap 8 · 24×24 ikon + (16/32 w500 + 16×16 chevron) | aynı ✅ |
| fiyat satırı | 382×36 (pad 6/8, radius 8, gap 8) · değişim rozeti 22 px | 382×34 ✅ |
| video satırı | 366×94 · kapak 168×94 · metin 182 | aynı ✅ |
| alt satır | 366×22 · 14/22 | aynı ✅ |

**Panel sayısı — bilinçli fark.** CoinDesk 6 panel basıyor: Prices · Stocks ·
Exchange Benchmark · Videos · CoinDesk Indices · Podcast. Bizde hisse
(BIST dışı), endeks ve borsa-benchmark **verisi yok**; bülten içeriği de
yayında değil. "Veri yoksa modül basılmaz" kuralı gereği o paneller
uydurulmadı. Yerine **gerçek veriyle dolan 7 panel** kuruldu:

| # | Panel | Kaynak |
|---|---|---|
| 1 | Fiyatlar (10 satır) | canlı fiyat şeridi (`/price`) |
| 2 | Videolar (5 satır) | YouTube `video` + `stream` arşivi |
| 3 | Klipler (3×2, 9:16) | YouTube `short` |
| 4–7 | Piyasalar · Politika · Teknoloji · Finans (5'er satır) | bölüm listeleri (`/markets`, `/policy`, `/tech`, `/business`) |

Bölüm panelleri ana sayfa havuzundan değil **bölümün kendi listesinden**
besleniyor (`cdListeIcerikleri`): havuz 60 kayıtla sınırlı olduğu için
Piyasalar paneli 2 satırda kalıyordu.

**Videolar panelinde açıklama ve alt bağlantı yok** — CoinDesk'te de yok
(ölçüm: başlık 32 → 16 → 5×94 = 534). Eklenince 5. satır kartın dışına
taşıyordu. Klipler panelinde de aynı gerekçeyle ikisi basılmıyor: 2 satır
(2×248 + 8) ancak sığıyor.

**JS yok:** kaydırma tarayıcının kendi `overflow-x` + `scroll-snap`
davranışından geliyor (dokunmatik, Shift+tekerlek, klavye, sürükleme).
CoinDesk'te de ok düğmesi yok.

Doğrulama (1920 · 768 · 390): yatay taşma **0**, panel 7, kırık görsel 0
(görseller `loading="lazy"` — ekrana gelince yükleniyor), koyu temada kart
`#161616` / kenar `#262626` / metin `#d4d4d4`. `cd.css` bütçesi aşılmadı.

### 11.17 On dokuzuncu tur (mobil CoinDesk eşleşmesi + koyu palet) — DURUM

Ölçüm: gerçek Chrome, 9 Kasım 2026. Mobil 390×844, `isMobile`, iPhone UA.
CoinDesk tarafı JS KAPALI ölçüldü (no-JS varyantı birebir alınıyor).

#### 11.17.1 Mobil başlık ve menü

CoinDesk iki mobil menü varyantı basıyor: JS açıkken kayan panel, JS
kapalıyken `<details name="cd-nav-section">` akordiyonu. Projenin JS'siz
ilkesi gereği **no-JS varyantı** alındı.

| Öğe | CoinDesk (ölçüldü) | bitcosar |
|---|---|---|
| header | 57 px (56 + 1 px çizgi) | 57 ✅ |
| iç dolgu | 16 px, logo ORTADA | aynı ✅ |
| hamburger | 32×32 x=16, ikon 20×20 | aynı ✅ |
| arama ikonu | 36×32 x=48 | 36×32 x=16+32 ✅ |
| logo | 127×25, mutlak orta | 118×26, mutlak orta ✅ |
| Oturum Aç | 77×40, 16/24 w400, 0 16 | 113×40 aynı tipografi ✅ |
| panel | header altı, tam genişlik, `max-h calc(100vh − 57px)` | aynı ✅ |
| panel iç dolgu | 16 16 192 | aynı ✅ |
| satır | 64 px (16 + 32 + 16) | aynı ✅ |
| başlık | 14/32 w700, x=32 | aynı ✅ |
| chevron | 24×24 sağda, **DÖNMEZ** | aynı ✅ |
| alt bağlantı | 14/32 w400, dolgu 16 0 16 24 → 64 px | aynı ✅ |
| ayraç | tam genişlik 1 px | aynı ✅ |
| alt blok | Dil · Tema · Oturum, 32 px aralık | aynı ✅ |

**Panel yüksekliği:** liste kısa olduğunda altında sayfa içeriği görünüyordu
(kullanıcı bildirimi). `min-height: calc(100vh − var(--cd-header-h))`
eklendi; içerik uzunsa `max-height` sınırı kaydırmayı devreye sokuyor.

**Mobilde yapışkanlık yok:** CoinDesk mobilde üst çubuk `static`, şerit de
akışta. Bizde de öyle (`--cd-header-h: 57px`, `.cd-ticker { position: static }`).
Tek satırlık kaydırılmış şerit (`html.cd-kaydi`) yalnız ≥1024 px'te eklenir
(`matchMedia` ile), mobilde sınıf hiç yazılmaz.

**Tema anahtarı** iki seçenekli segment (`role="radiogroup"` +
`aria-checked`); `csp-delegated.js` `data-tema-deger` ile işler. Tema
düğmesinin mobil satırı CoinDesk'teki gibi: kap `surface-soft` 4 px dolgu,
8 px köşe, seçili düğme panel zemini + gölge.

#### 11.17.2 Mobil makale

| Öğe | CoinDesk (390 px) | bitcosar |
|---|---|---|
| şerit → kicker | 32 px | 32 ✅ |
| kicker | 14/32 w500 | aynı ✅ |
| H1 | 28/36 **w500** | aynı ✅ |
| deck | 16/24 w400, mt 16 | 16/24, mt 8 → **fark bilinçli** (deck H1'e 8 px) |
| gövde | 20/28, paragraf arası 32 | aynı ✅ |
| özet kutusu | 358×116, iç 354×112 (dolgu 20/20/12) | 358×116 / 354×112 ✅ |
| özet başlığı | 14/24 w500 | 16/24 w500 → **fark bilinçli** (masaüstüyle tutarlı) |
| görünür breadcrumb | yok | yok ✅ |
| yatay taşma | 0 | 0 ✅ |

**Bulunan gerçek hata:** `.cd-article` yalnız `max-width: 680px` taşıyordu;
flex ebeveyn içinde küçülmediği için 390 px viewport'ta **680 px** kalıyor ve
sayfa yatay taşıyordu. `width: 100%` eklendi (masaüstünde `max-width` zaten
sınırlıyor, görünüm değişmiyor).

#### 11.17.3 Koyu palet (9 Kasım 2026 — kullanıcı isteği)

| Katman | Renk |
|---|---|
| gövde | `#0a0b0e` |
| üst çubuk | `#0e1014` |
| şerit (`--cd-ticker-bg`) | `#0e1014` |
| alt bilgi (`--cd-footer-bg`) | `#0e1014` |
| açılır panel (`--cd-surface-panel`; mobil menü, arama, mega menü) | `#0e1014` |
| kart/blok (`--cd-surface-soft`) | `#14171d` |
| etkileşim (`--cd-surface-action`) | `#1c2027` |
| çizgi | `#1f2329` / güçlü `#3a4048` |
| metin | `#eaecef` / sönük `#848e9c` |
| yükseliş / düşüş | `#0ecb81` / `#f6465d` |

Şerit, alt bilgi ve panel **ayrı jeton** (`--cd-ticker-bg`, `--cd-footer-bg`,
`--cd-surface-panel`). Ayrı yazılmalarının nedeni: gövde rengi ileride
değişirse bu üç yüzeyin sessizce onunla birlikte kaymasını engellemek.
Açık temada üçü de beyaz — görünüm değişmiyor.

`theme-color` meta'sı da güncellendi (`#0a0b0e`). Kamu sayfaları
(`cd-public.css`, `uye-giris.css`) aynı skalaya taşındı; orada koyu tema
`html[data-theme="dark"]` ile geliyor.

**Bulunan hata:** koyu temada giriş sayfasının gövde metni `#262626`
kalıyordu (taban kural `color: #262626` yazıyor, koyu blok yalnız zemin
veriyordu). `color: var(--a-text)` eklendi — 1,1:1 kontrasttı, WCAG AA'nın
çok altında.

#### 11.17.4 Marka sarısı → `#fcc334`

`#ffd200` sitede **tek yerden** yönetiliyordu ama dört dosyaya dağılmıştı:
`--cd-brand`, `cd-public.css` (2 kural + okuma çubuğu), `favicon.svg`,
`google-preferred.svg`. Hepsi `#fcc334` yapıldı; tasarım testi de yeni
değeri doğruluyor.

#### 11.17.5 Özet kutusu gradyanı

CoinDesk'in sarı→yeşil gradyanı korunmuyor (kullanıcı isteği):
`linear-gradient(44deg, #007effe0 -0.2%, #ff00a3 171.06%)` — mavi → magenta.
Yapı aynı: dış kap radius 7 / 2 px dolgu, iç kutu radius 5 / dolgu 20 20 12.

#### 11.17.6 Doğrulama (390 px + 1512 px, gerçek Chrome)

Yatay taşma **0**: `/`, makale, `/policy`, `/price`, `/price/bitcoin`,
`/hesabim`, `/uye-giris`, `/arama`. Ölçülen koyu tema değerleri:
gövde `rgb(10,11,14)`, üst çubuk `rgb(14,16,20)`, şerit `rgb(14,16,20)`,
alt bilgi `rgb(14,16,20)`, panel `rgb(14,16,20)`, metin `rgb(234,236,239)`.
Mobil menü: Escape kapatıyor, dışarı tıklama kapatıyor, arama paneli
menüyü kapatıyor ve odağı arama girdisine veriyor.

`templ` / `gofmt` / `go vet` / `go test` / `go build` temiz. `cd.css`
bütçesi 172 → **176 KB** (ölçüldü 177.685 bayt); yeni CSS'in tamamı koyu
tema jetonları ve mobil düzeltmeleri, harici istek eklenmedi.

---

## 16. Gerçek veri katmanı (8 Kasım 2026)

Bu turda **sahte/demo veri siteden çıkarıldı** ve grafikler gerçek kaynaklara
bağlandı. İlke: *veri yoksa modül basılmaz* — uydurma sayı yazılmaz.

### 16.1 Piyasa şeridi artık HER sayfada

**Bildirim:** "diğer sayfalarda kur alanı neden yok sadece ana sayfada var".

**Kök neden:** şablon (`cdTicker`) dokuz sayfa ailesinde çağrılıyordu ama
`Shell.Ticker` yalnız `cd_home.go` içinde dolduruluyordu. Diğer sayfalarda
dilim boş olduğu için şerit **hiç basılmıyordu**.

**Çözüm:** şerit `cdShellCtx` içinde doldurulur (tek yer, tüm sayfalar).
Ölçüm: `/`, `/markets`, `/price`, `/videos`, `/business`, `/tech`, `/policy`,
`/opinion`, `/latest-crypto-news`, `/business?sayfa=2` → **7 kalem / sayfa**.

**Hisse sekmesi** (`?serit=stocks`) ölü bağlantıydı: parametre hiçbir yerde
okunmuyordu. Artık okunuyor, `sh.TickerTab` şablona geçiyor ve BIST verisi
(`services` işi → Yahoo Finance `.IS`) listeleniyor. Kripto satırları
`borsa/ana` önbelleğinde de bulunduğu için sekmede **süzülür** — aksi hâlde
"Hisse Senetleri" sekmesi BTC/ETH gösteriyordu (ölçüldü).

Şerit kalemleri artık `Href` taşır: kripto → `/price/{slug}`, BIST → `/markets`
(ölü `/price/THYAO` adresi basılmaz).

### 16.2 CryptoQuant — zincir-üstü veri

İstemci: `go/internal/cryptoquant` (yalnız `net/http`, ek bağımlılık yok).
Anahtar `CRYPTOQUANT_KEY` ortam değişkeninden okunur; **repoya yazılmaz**.
Anahtar yoksa istemci kapalı davranır ve hiçbir istek atılmaz.

| Metrik | Uç | Kapsam (ölçüldü) |
|---|---|---|
| Fiyat OHLCV | `/{coin}/market-data/price-ohlcv` | btc, eth, xrp, trx (diğerleri 404) |
| MVRV | `/btc/market-indicator/mvrv` | yalnız btc |
| SOPR | `/btc/market-indicator/sopr` | yalnız btc (`sopr/a_sopr/sth_sopr/lth_sopr`) |
| Borsa giriş/çıkış/rezerv | `/{coin}/exchange-flows/{inflow|netflow|reserve}` | btc, eth — **`exchange=` zorunlu** |
| Borsa balina oranı | `/btc/flow-indicator/exchange-whale-ratio` | yalnız btc |

Notlar:
- `GET /v1/` **404** döner (kök uç yok) — doğrudan metrik yolu çağrılır.
- Fiyat ve piyasa değeri **CoinGecko'da kalır** (`repo.Fiyatlar`, 60 sn
  önbellek): CryptoQuant yalnız 4 varlık için fiyat veriyor, şerit ve fiyat
  tablosu 25 varlık gösteriyor. İki kaynak karıştırılmaz.
- Yanıtlar `services_cache` (`kind = zincir`) içinde **10 dakika** önbellekli.
  Seriler paralel çekilir (6 istek → ~1 tur gecikme).

### 16.3 Grafikler — bağımlılıksız inline SVG

CSP `script-src 'self'` olduğu için harici grafik kütüphanesi yüklenemez.
Grafikler **sunucuda** üretilir (`templkit.CDGrafikYolu`): `viewBox 640×180`,
`preserveAspectRatio="none"`, çizgi kalınlığı `vector-effect:
non-scaling-stroke` ile sabit. JS kapalıyken de görünür, ilk boyamada hazır,
istemci maliyeti sıfır.

| Sayfa | Modül |
|---|---|
| `/` | Piyasa Nabzı (BTC fiyat + MVRV/SOPR/net akış/rezerv) |
| `/markets` | Piyasa Nabzı (manşetin üstünde) |
| `/price` | Piyasa Nabzı |
| `/price/bitcoin` | Fiyat grafiği + 5 zincir-üstü kart |
| `/price/ethereum` | Fiyat grafiği + net akış + rezerv |
| diğer coinler | **hiç basılmaz** (CryptoQuant o varlık için veri döndürmüyor) |

Ölçüm (Chrome 1440 px): 5 kart / `/markets`, SVG 755×180 (büyük) + 4×232×84
(mini), yatay taşma 0. `cd.css` bütçesi 136 → **140 KB** (ölçüldü 141.787
bayt); harici istek eklenmediği için ağ maliyeti artmadı.

Sayı biçimi tek kaynağa alındı: `repo.SayiBicim` ("1,234.56"). Önce
`piyasa.FormatTR` ile aynı kartta "1,40" ve "$75,778.14" yan yana görünüyordu.

### 16.4 Tasarım tablosundaki kalan boşluklar

| Konu | Durum |
|---|---|
| Video · araştırma · klip modülleri | ✅ gerçek YouTube verisiyle dolu |
| Hero ray modülleri | ✅ üç kolon, sağ ray üç modül |
| Piyasa şeridi | ✅ tüm sayfalarda |
| Hisse sekmesi | ✅ gerçek BIST verisi |
| Grafik paneli | ✅ yeni bileşen ailesi (CoinDesk'te grafik kartı ölçüsü: başlık 12 px uppercase, değer 28 px w700, eksen 12 px) |
| Hisse şablonu `/price/stock/*` | ✅ gerçek ABD hisse verisiyle (Yahoo Finance; Basic + Advanced mum) |
| Market Cap satırı | ✅ crumb akışıyla geliyor ($1.63T / ₺425.41B) |
| Google News pill | ⛔ yayın sonrası |
| Hisse keşfedilebilirliği | ✅ sitemap + JSON-LD + mega menü + /price/stock 301 |
| Çeviri etiketi | ⛔ EN çeviri hattı gelince |

---

*Ölçen: Claude Opus 5, 6 Kasım 2026, gerçek Chrome, hesaplanmış stiller.
Ölçülmeyen kalemler §Ölçüm künyesi'nde işaretli.*
