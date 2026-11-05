# SPEC — Gazete Sayfa Düzenleyici ve Baskıya Hazırlık

**Durum:** taslak → uygulanıyor
**Kapsam sahibi:** `/admin/gazete/sayilar/{id}/duzenle` ve baskı çıktısı zinciri
**Üst plan:** [`docs/GAZETE-PLAN.md`](../docs/GAZETE-PLAN.md) Faz 2 ("Şablon Yapısı & Editör") bu dokümanla değiştirilir.

> Bu doküman birden fazla AI ajanının sırayla çalışabilmesi için yazıldı.
> **Kural:** Bir faza başlamadan §2'deki ölçülmüş gerçekleri ve §4'teki kararları oku.
> Fazı bitirince §7'deki ilerleme tablosunu güncelle ve kabul ölçütlerini kanıtla.
> Karar değiştirmek gerekiyorsa §4'e gerekçeli not ekle — sessizce sapma.

---

## 1. Neden

Günlük yayımlanan site içeriklerinin, gazete sayfası peyzajına göre düzenlenip
baskıya hazır hale getirilmesi gerekiyor. Bugün editör var, kaydediyor, PDF
üretiyor — ama **çıktının sayfaya sığdığına dair hiçbir garanti yok** ve
**uygulanacak hazır bir peyzaj (şablon) yok**. Yani "var olan plana göre
düzenleme" denen şeyin planı ortada değil.

İhtiyaç üç cümlede:

1. Hazır gazete peyzajları arasından seç.
2. Günün içeriklerini o peyzaja yerleştir, sığmayanı **gör**.
3. Baskıya giderken sürprizsiz, gerçek bir dijital nüsha al.

---

## 2. Ölçülmüş mevcut durum

Aşağıdakiler varsayım değil; bu spec yazılırken kod ve veritabanından okundu.
Tarih: 2026-10-30.

### 2.1 Veri

| Tablo | Satır | Not |
|---|---|---|
| `gazete_issues` | 6 | **6'sının da `template_id` NULL** |
| `gazete_pages` | 60 | yüzde tabanlı blok JSON'u dolu |
| `gazete_templates` | **0** | şema var, içerik yok → galeri boş |
| `gazete_subscriptions` / `_deliveries` / `_user_roles` | 0 | kapsam dışı |

Blok JSON şekli (`gazete_pages.blocks`, gerçek örnek):

```json
{"h":52,"w":96,"x":2,"y":3,"id":"p1m","tip":"manset","stil":null,
 "metin":"","baslik":"","gorsel":"",
 "icerik":{"haberler":[{"id":557,"baslik":"…","spot":"…","gorsel":"media/202610/….png",
                        "kaynak":"Sözcü","kategori":"Gündem","tarih":"15.10.2026","url":"/haber/…"}]},
 "kaynak":{"id":0,"tip":"manset","limit":1,"icerikler":null}}
```

Model: **yüzde tabanlı serbest konumlu bloklar + içerik sorgusuna bağlanma**
(`kaynak`) + **çözümlenmiş içeriğin blokta saklanması** (`icerik`). Bu sağlam bir
temel; değiştirilmiyor.

### 2.2 Geometri

| Yer | Ölçü |
|---|---|
| Referans tasarım (`Newspaper #6003.svg`) | `viewBox 793.7 × 1122.5` = **A4 @96dpi** (210×297mm) |
| Editör tuvali (`gazete.css:22`) | `aspect-ratio: 297/420`, `max-width:720px` |
| Baskı CSS (`gazete_fill.go`) | `@page { size: A3 portrait }`, `.gazete-sayfa { 297mm × 420mm }` |

A3 ve A4 aynı √2 oranını paylaşır, bu yüzden **yüzdelik blok geometrisi iki boy
arasında bozulmadan taşınır**. Ama baskı CSS'indeki tipografi `pt` cinsinden
sabit (`.gb-manset h1 { font-size: 26pt }`). Sayfa boyu değişince tipografi
ölçeklenmiyor → aynı blok JSON'u A4'te apayrı görünür. Referans A4, sistem A3
basıyor.

### 2.3 Baskı kalitesini bozan iki şey

**(1) Sessiz metin kesmesi.** Hem editörde (`gazete.css:26 .gz-blok{overflow:hidden}`)
hem baskıda (`.gb{position:absolute;overflow:hidden}`) taşan içerik hiçbir uyarı
vermeden kırpılıyor. Bir manşet spotu bloğa sığmazsa yarısı yok olur ve bunu
ancak PDF'i açınca görürsün.

**(2) Editör WYSIWYG değil.** Editör kendi yer tutucularını çiziyor
(`gazete.js:109 blokIcerik` → `"— haber —"`, `"Manşet haberi (otomatik doldurulur)"`),
baskı ise Go tarafında ayrı bir üreticiden geçiyor (`gazete_fill.go:317 gazeteBlockHTML`).
**İki ayrı render yolu, iki ayrı tipografi.** Editörde doğru görünen, baskıda
sığmıyor. Blok JSON'unda çözümlenmiş gerçek içerik (`icerik.haberler`) zaten
duruyor ama editör onu hiç göstermiyor.

### 2.4 Editör eksikleri (`gazete.js` 533 satır okundu)

| Eksik | Kanıt |
|---|---|
| Sütun ızgarası yok | yakalama `Math.round(dx*2)/2` → 0.5% adım; A3'te 1.485mm, sütunla ilgisiz |
| Tipografi/stil denetimi yok | `Stil map[string]any` **yalnız `gazete_fill.go:52`'de tanımlı, hiçbir yerde okunmuyor/yazılmıyor** |
| Katman sırası yanıltıcı | tuval dizi sırasında boyar, katman paneli `y,x`'e göre sıralar → panel sırası ≠ boyama sırası; `z` alanı yok |
| Tek tutamak | sağ-alt köşe; kenar tutamağı, oran kilidi yok |
| Sayfa yönetimi yok | ekle/sil/sırala/adlandır rotası **yok**; yalnız `POST …/sayfa` (tek sayfa bloklarını kaydet) |
| Kayıp riski | `Kaydet` sayfa başına ayrı POST (`Promise.all`), işlem (transaction) yok, sürüm denetimi yok → kısmi kayıt mümkün |
| Kirli durum takibi yok | `beforeunload` yok; sekmeyi kapatınca sessiz veri kaybı |
| Panel dışı diyaloglar | 7 adet `prompt()`/`alert()`/`confirm()` |
| Yakınlaştırma yok | A4 detayını ayarlamak için büyütme yok |
| Kopyala/yapıştır/çoğalt yok | yalnız sil + ok tuşlarıyla taşıma |
| Baskı önizleme yüzeysel | `body.classList.toggle("gz-preview")`; kesim/taşma payı/kenar kılavuzu yok |
| PDF akışı kör | `alert("PDF üretildi. Panele dönüp indirebilirsiniz.")` — ilerleme ve bağlantı yok |

### 2.5 Çalışan ve dokunulmayacak yerler

Altyapı sanıldığından sağlam; yeniden yazılmayacak:

- 25 adet `/admin/gazete*` rotası ayakta.
- Şablon handler'larının **tamamı** mevcut ve çalışıyor: galeri
  (`handleGazeteSablonlar`), JSON okuma (`handleGazeteSablonJSON`), varsayılan
  atama (`handleGazeteSablonVarsayilan`), sayıdan şablon üretme
  (`handleGazeteSayiSablonaKaydet`), silme. **Boş olan yalnızca tablonun kendisi.**
- Otomatik doldurma (`gazeteAutoFill`) ve içerik sorguları (`gazeteQuery`) çalışıyor.
- Chromium headless PDF üretimi ve sayfa görüntüsü çıkarma çalışıyor.
- StPageFlip okuyucu (`/gazete/{tarih}`) çalışıyor.

Blok tipleri Go ve JS tarafında **tutarlı** (9 tip: `manset, liste, kart, baslik,
metin, gorsel, cizgi, reklam, kunye`). `kategori` ve `sabit` blok tipi değil,
**kaynak** tipidir — karıştırma.

---

## 3. Referans peyzaj anatomisi

`Newspaper #6003.svg` düzleştirilmiş görsel bir maket: 107 `<path>`, 3 `<image>`,
**0 `<text>`**. Yani dosya doğrudan şablona çevrilemez; yapısı elle okunup bizim
şablon formatımıza yazılır. Okunan anatomi (yukarıdan aşağı):

1. Sayı no / tarih şeridi
2. Lacivert künye bandı (gazete adı)
3. Kategori şeridi — ECONOMY · WORLD NEWS · TRAVEL · CULTURE · TECHNOLOGY · SPORTS
4. 2 kolon kahraman alan: görsel + üzerine beyaz metin kutusu / turuncu metin kutusu
5. 3 kolon orta şerit: koyu başlık kutusu · görsel · açık başlık kutusu
6. 2 kolon alt blok: çok büyük manşet + gövde / görsel üzerine koyu bindirme
7. Alt bilgi: "Page 1" / "Main page", ve devam rozetleri ("Page N")

Bundan çıkan ve bugün **karşılanamayan** iki gereksinim:

- **Görsel üzerine metin bindirme** (overlay) — bugün `gorsel` bloğu yalnız resim basar.
- **Renkli zemin kutusu** (lacivert bant, turuncu kutu, koyu/açık başlık kutusu) —
  bugün blok zemini/rengi ayarlanamaz çünkü `stil` ölü.

---

## 4. Kararlar

Numaralı, gerekçeli ve bağlayıcı. Değiştirmek isteyen buraya not bırakır.

### K1 — Tek render yolu: baskı HTML'i editörün de kaynağıdır

İki render yolu (JS `blokIcerik` + Go `gazeteBlockHTML`) birleştirilir. Baskı
CSS'i gerçek bir statik dosyaya (`static/gazete-baski.css`) taşınır; editör
tuvali aynı `.gb*` sınıflarını, gerçek sayfa ölçüsünde bir kutuda
`transform: scale()` ile küçültülerek kullanır.

*Gerekçe:* WYSIWYG'in tek güvenilir yolu aynı HTML+CSS'i göstermektir. Yan
kazanç: birbirinden sapan iki uygulama tek uygulamaya iner.

*Sonuç:* `gazete.js:blokIcerik` silinir. Editör blok gövdesini sunucudan alır
(`GET /admin/gazete/sayilar/{id}/blok-html`) veya — tercih edilen — Go'daki
üretici templ bileşenine çevrilip her iki yoldan çağrılır.

### K2 — Sayfa boyu ve tipografi ölçeği şablonun özelliğidir; varsayılan A4

Baskı CSS'i parametrik olur: `--gz-sayfa-w`, `--gz-sayfa-h`, `--gz-tip-olcek`.
Şablon bunları bildirir. Varsayılan **A4 dikey** (referans bu).

*Gerekçe:* A3 ve A4 gazeteler birbirinin ölçekli kopyası değildir — A4 daha az
ve daha büyük blok taşır. Tipografiyi sayfa boyundan matematikle türetmek A4'te
6–7pt gövde gibi okunamaz sonuç verir. Bu yüzden ölçek **türetilmez, bildirilir**;
oransal ölçekleme yalnızca makul bir varsayılandır.

*Sonuç:* Mevcut A3 çıktısı bozulmaz; A3 şablonu kendi ölçeğini bildirerek bugünkü
görünümü korur.

### K3 — Taşma açık, görünür ve PDF'i engeller (Scribus deseni)

Taşma sihirli biçimde çözülmez; **gösterilir**.

- Editör: blok gerçek içeriğiyle render edilir, `scrollHeight > clientHeight`
  ölçülür, bloğun sağ-altına taşma rozeti + araç çubuğunda sayaç konur.
- PDF öncesi geçit: `POST …/dogrula` baskı HTML'ini **aynı** headless
  Chromium'da açıp blok başına taşmayı ölçer ve rapor eder.
- Taşma varsa PDF üretimi uyarır; `zorla=1` olmadan üretmez.

*Gerekçe:* Scribus da taşan metin için çerçeveye işaret koyar ve sayfayı
kendiliğinden açmaz — kullanıcı kararı gerektirir. Otomatik küçültme/kırpma
baskıda telafisi olmayan hata üretir. Tarayıcı ölçümünü Chromium'da tekrarlamak,
"editörde sığdı ama PDF'te taştı" sınıfını tamamen kapatır.

### K4 — Sütun ızgarası şablonda tanımlı, yakalama ona göre

Şablon `izgara: {sutun, oluk, kenar}` bildirir (mm). Sürükleme/boyutlandırma en
yakın sütun kenarına yakalanır; `Alt` serbest bırakır. Bugünkü 0.5% yuvarlama
kaldırılır.

*Gerekçe:* Gazete sayfası sütun ürünüdür; keyfi 0.5% adım hizalı sayfa üretemez.

### K5 — Yeni ağır bağımlılık yok

`fabric.js`, `interact.js`, `gridstack` eklenmez. Mevcut ~50 satırlık pointer
event katmanı genişletilir. StPageFlip (okuyucu) ve Chromium (PDF) kalır.

*Gerekçe:* `GAZETE-PLAN.md` §3b canvas/raster modelini baskı, SEO ve otomatik
doldurma için yanlış bulmuş — hâlâ doğru. `interact.js` yakalamayı koordinat
değiştiricisiyle yapar, görsel hizalama kılavuzu sunmaz; zaten elle yazılacak.
Pacely işinde de elle yazılmış JS, hazır çözümden iyi sonuç verdi.

### K6 — Kayıt tek işlemde ve sürüm denetimli

`POST …/sayfalar` tüm sayfaları tek gövdede, tek transaction'da alır ve
`updated_at` tabanlı iyimser kilit uygular. Sayfa başına POST kaldırılır.

*Gerekçe:* Bugün 8 sayfalık bir sayı 8 bağımsız POST; üçüncüsü düşerse sayı
tutarsız kalır ve kullanıcı bunu bilmez.

### K8 — "Baskıya hazır" iki ayrı çıktı demektir

Tek bir PDF hem ekran nüshası hem matbaa dosyası olamaz. İkiye ayrılır:

- **Dijital nüsha** — bugünkü Chromium çıktısı. RGB, ekran/indirme için. Değişmiyor.
- **Matbaa dosyası** — ayrı adımda üretilir: PDF/X, CMYK, kesim/taşma kutuları,
  gömülü font. Chromium bunu üretemez; üretilen PDF dönüştürülür.

Zincir: Chromium PDF → **Ghostscript** (PDF/X + CMYK + ICC) → **doğrulama**.

*Gerekçe (2026-10-30'da canlı PDF üzerinde ölçüldü):*

| Ölçüm | Değer | Anlamı |
|---|---|---|
| Producer | `Skia/PDF m155` | Chromium çıktısı |
| Sayfa kutusu | 594.96 × 841.92 pt | A4 ✓ doğru |
| TrimBox / BleedBox | **0 / 0** | kesim ve taşma payı kutusu yok |
| OutputIntent | **0** | PDF/X değil, çıktı amacı tanımsız |
| PDF sürümü | 1.4 | PDF/X-4 en az 1.6 ister |
| Gömülü font | **DejaVuSerif / DejaVuSans** | CSS Georgia istiyor; **sunucuda Georgia yok** |

Yani bugünkü dosya ekran için iyi, **matbaaya verilecek dosya değil** — ve
üstüne gazete amaçlanan yazı tipiyle basılmıyor.

*Araç durumu (aynı makinede doğrulandı):* Ghostscript **10.06.0 kurulu**;
poppler araçları (`pdfinfo`, `pdffonts`, `pdfimages`) tam — preflight'ın
büyük kısmı bunlarla yapılabilir. veraPDF kurulu değil (JVM tabanlı); zorunlu
tutulmuyor, isteğe bağlı ikinci denetim olarak kalır.

*K5 ile ilişkisi:* K5 ön yüze ağır JS bağımlılığı almama kararıydı; burada
söz konusu olan sunucu tarafı. Ghostscript zaten kurulu bir sistem aracı.
`pdfcpu` (saf Go) eklenirse ikiliye girer, yeni servis gerekmez — ama önce
poppler araçlarıyla yetinilip yetinilmediğine bakılır, gereksiz bağımlılık
alınmaz.

### K7 — Kapsam dışı (bu spec'te yapılmayacak)

Döndürme, serbest çokgen/çizim, blok içi zengin metin editörü, otomatik metin
akışı (bağlı çerçeve zinciri), CMYK ayrıştırma, imposition/forma düzeni,
gerçek zamanlı çok kullanıcılı düzenleme. Gerekirse ayrı spec.

*Not:* Bağlı çerçeve (linked frames) bilinçli olarak dışarıda — Scribus'ta bile
elle kurulan bir mekanizma ve taşma rozeti (K3) ihtiyacın büyük kısmını karşılar.

---

## 5. Veri modeli değişiklikleri

### 5.1 Blok (ek alanlar)

```go
type gazeteBlock struct {
    // … mevcut alanlar …
    Z    int            `json:"z"`    // YENİ: boyama sırası, 0 = en altta
    Stil map[string]any `json:"stil"` // ARTIK KULLANILIYOR, bkz. 5.2
}
```

`Z` yokken (eski kayıtlar) dizi indeksi kullanılır → geriye dönük uyumlu.

### 5.2 `stil` sözlüğü — izin verilen anahtarlar

Serbest `map[string]any` güvensiz (CSS enjeksiyonu). **Beyaz liste** uygulanır;
listede olmayan anahtar sessizce atılır ve `TestGazeteStilBeyazListe` bunu korur.

| Anahtar | Değer | Etki |
|---|---|---|
| `zemin` | sınırlı renk jetonu (`lacivert`,`turuncu`,`koyu`,`acik`,`beyaz`,`yok`) | blok zemini |
| `metinRenk` | `acik` \| `koyu` | metin rengi |
| `hiza` | `sol` \| `orta` \| `sag` | yatay hiza |
| `sutun` | 1–4 | blok içi metin kolonu |
| `olcek` | 0.6–2.0 | bloğun tipografi çarpanı |
| `bindirme` | `yok` \| `alt` \| `ust` \| `tam` | görsel üzerine metin (§3) |
| `kenarlik` | `yok` \| `ince` \| `kalin` | çerçeve |

Renk **jetonu**, serbest hex değil: baskıda tutarlı palet ve CSS enjeksiyonuna
kapalı yüzey için.

### 5.3 Şablon belgesi (`gazete_templates.pages`)

```json
{
  "surum": 1,
  "sayfa": { "boyut": "A4", "yon": "dikey", "tipOlcek": 1.0,
             "tasmaPayi": 3, "kesimPayi": 5 },
  "izgara": { "sutun": 12, "oluk": 4, "kenar": 12 },
  "sayfalar": [ { "no": 1, "ad": "Kapak", "bloklar": [ /* gazeteBlock[] */ ] } ]
}
```

Ölçüler mm. `surum` alanı gelecekteki şema göçü için zorunlu.

---

## 6. Fazlar

Her faz **tek başına sevk edilebilir** ve kendi kabul ölçütünü taşır. Sıra
bağlayıcı: her faz öncekinin garantisine yaslanır.

### Faz 1 — Ölçü gerçeği ve tek render yolu  `[K1, K2]`

**Yap**
1. `gazetePrintCSS` sabitini `go/internal/webui/static/gazete-baski.css`'e taşı;
   `--gz-sayfa-w/h`, `--gz-tip-olcek` değişkenleriyle parametrik yap.
2. `@page size` ve `.gazete-sayfa` ölçüsünü şablon/sayı bildiriminden üret.
3. Blok gövdesi üreticisini tek yere al (templ bileşeni veya tek Go fonksiyonu);
   baskı ve editör aynı yerden çağırsın.
4. `gazete.js:blokIcerik` ve `.gz-blok-*` görsel taklit CSS'ini sil; editör
   tuvali gerçek sayfa ölçüsünde kutu + `transform: scale()` kullansın.
5. Varsayılan sayfa boyu A4; A3 şablonu kendi ölçeğini bildirerek korunsun.

**Kabul ölçütü**
- Aynı blok JSON'u editörde ve üretilen PDF'te **aynı** kırılma noktalarıyla görünür.
- `/admin/gazete/sayilar/78/duzenle` gerçek haber başlıklarını gösterir, `— haber —` yazmaz.
- A3 tipografisi **yeniden üretilebilir**: taban 9.5pt bildirildiğinde parametrik
  çarpanlar eski sabit puntolara (16/26/11/7.5…) 0.01pt içinde denk gelir.
  Testle kanıtlanır.

**Bilinçli davranış değişikliği:** Şablon şeması Faz 3'te geldiği ve
`gazete_templates` bugün boş olduğu için Faz 1 sonrası **mevcut 6 sayı A4
basar** (önceden A3). İstenen yön bu — referans tasarım A4. A3 isteyen sayı,
Faz 3'te `A3 — Mevcut Düzen` şablonunu seçerek bugünkü görünümü birebir alır.

**Koruyucu testler**
- `TestGazeteTekRenderYolu` — `gazete.js` içinde blok HTML'i üreten kod kalmadığını doğrular.
- `TestGazeteBaskiCSSParametrik` — baskı CSS'inde sabit `A3`/`297mm`/`420mm` geçmediğini doğrular.
- `TestGazeteBlokTipleriEslesir` — Go üreticisi, JS paleti ve şablon şeması aynı 9 tipi tanır.

---

### Faz 2 — Taşma görünürlüğü ve baskı geçidi  `[K3]`

**Yap**
1. Editörde blok başına taşma ölçümü (`scrollHeight > clientHeight`) + rozet
   (Scribus deseni: bloğun sağ-alt köşesinde işaret) + araç çubuğunda sayaç.
2. `POST /admin/gazete/sayilar/{id}/dogrula` — baskı HTML'ini headless
   Chromium'da açar, blok başına taşmayı ölçer, JSON rapor döner.
3. `handleGazetePDF` doğrulamayı çağırır; taşma varsa `zorla=1` olmadan üretmez
   ve hangi sayfa/blokta olduğunu söyler.
4. Taşan blok için düzeltme önerisi: "yükseklik +Xmm" veya "ölçek 0.9".

**Kabul ölçütü**
- Bir manşet spotu bloğa sığmadığında editörde rozet çıkar.
- Taşma varken PDF düğmesi sebebini söyleyerek durur.
- `zorla=1` ile üretim yine mümkün (kullanıcı kararı).

**Koruyucu testler**
- `TestGazeteDogrulaTasmaYakalar` — kasten taşan sabit blok JSON'u ile raporda o blok görünür.
- `TestGazetePDFTasmadaDurur` — taşma varken `zorla` olmadan 4xx/uyarı döner.

---

### Faz 3 — Şablon kütüphanesi: gerçek peyzajlar  `[§3, K2, K4]`

**Yap**
1. Şablon belge şemasını (§5.3) Go tarafında doğrulayıcıyla tanımla.
2. **Referans peyzajını** (§3) şablona çevir: `A4 — Klasik Manşet`.
3. En az 4 şablon daha: `A4 — İki Kolon Ağırlıklı`, `A4 — Görsel Ağırlıklı`,
   `A4 — Kategori Sayfası`, `A3 — Mevcut Düzen` (bugünkü çıktıyı korumak için).
4. Şablonları tohum (seed) göçüyle yükle — `gazete_templates` artık boş olmayacak.
5. Şablon galerisini küçük resimlerle gerçek bir seçiciye çevir; sayı
   oluşturulurken şablon seçimi zorunlu olsun ve `template_id` yazılsın.
6. Mevcut 6 sayıya uygun `template_id` ata (geriye dönük doldurma).

**Kabul ölçütü**
- `select count(*) from gazete_templates` ≥ 5.
- Yeni sayı oluşturma şablon seçmeden geçmez; oluşan sayıda `template_id` dolu.
- `A4 — Klasik Manşet` uygulandığında referans anatomisinin 7 bölgesi sayfada yerinde.

**Koruyucu testler**
- `TestGazeteSablonSemasi` — her tohum şablonu şemayı geçer; hiçbir blok sayfa dışına taşmaz (`x+w<=100`, `y+h<=100`).
- `TestGazeteSablonTipOlcek` — her şablon `sayfa.boyut` ve `tipOlcek` bildirir.

---

### Faz 4 — Editör etkileşimi: ızgara, hizalama, katman, stil  `[K4, K5, 5.1, 5.2]`

**Yap**
1. Sütun ızgarası çizimi + sütun kenarına yakalama; `Alt` serbest.
2. Komşu bloklara hizalama kılavuzu (kenar ve merkez), yakalama eşiği ~2mm.
3. 8 tutamak (4 kenar + 4 köşe), `Shift` oran kilidi.
4. `z` alanı: katman panelinde sürükleyerek sıralama, öne/arkaya al; panel sırası
   **boyama sırasıyla aynı** olsun.
5. Stil denetimleri (§5.2): zemin jetonu, metin rengi, hiza, kolon, ölçek,
   bindirme, kenarlık — `stil` alanı canlanır.
6. Çoğalt (`Ctrl+D`), kopyala/yapıştır, çoklu seçim (`Shift`+tık) ve çoklu taşıma.
7. Yakınlaştırma (`25%–200%`, `Ctrl+0` sığdır).

**Kabul ölçütü**
- İki blok yan yana bırakıldığında kenarları sütuna ve birbirine hizalanır.
- Referans peyzajındaki lacivert bant ve turuncu kutu **stil denetimleriyle** üretilebilir.
- Katman panelindeki sıra tuvaldeki örtüşmeyle birebir.

**Koruyucu testler**
- `TestGazeteStilBeyazListe` — liste dışı anahtar kaydedilmez, CSS'e sızmaz.
- `TestGazeteZSirasiKorunur` — kaydet/yükle turunda `z` değişmez; `z` yoksa indeks sırası korunur.

---

### Faz 5 — Sayfa yönetimi ve güvenli kayıt  `[K6]`

**Yap**
1. Sayfa ekle / sil / sırala / adlandır — rota + editör UI (bugün hiç yok).
2. `POST …/sayfalar` tek gövde, tek transaction, `updated_at` iyimser kilidi.
3. Kirli durum takibi: `beforeunload` uyarısı + otomatik kaydetme taslağı.
4. 7 adet `prompt`/`alert`/`confirm` yerine panel bileşenleri.
5. PDF akışı: ilerleme göstergesi ve bitince indirme bağlantısı (kör `alert` gider).

**Kabul ölçütü**
- Sayfa eklenip sıralanabilir; sayfa numaraları ve "Sayfa N" etiketleri tutarlı kalır.
- İki sekmede aynı sayıyı düzenleyip kaydetmek **sessiz üzerine yazma üretmez**; ikincisi çakışma bildirir.
- Kaydedilmemiş değişiklikle sekme kapatılırken uyarı çıkar.

**Koruyucu testler**
- `TestGazeteSayfalarTekIslem` — bir sayfa hatalıysa hiçbiri yazılmaz (atomiklik).
- `TestGazeteIyimserKilit` — eski `updated_at` ile kayıt çakışma döner.

---

### Faz 6 — Baskı önizleme ve dijital nüsha  `[K8]`

**Yap**
1. **Yazı tipini düzelt.** Baskı CSS'i Georgia istiyor ama sunucuda yok; çıktı
   DejaVuSerif'e düşüyor. Sunucuda gerçekten bulunan bir serif seçilir **veya**
   gazete için bir yazı tipi projeye alınıp `@font-face` ile gömülür. Seçim
   şablonda bildirilir (`sayfa.yaziTipi`), tipografi tabanı gibi.
2. Kesim (trim) / taşma payı (bleed) / güvenli alan kılavuzları; şablondan okunur.
3. Baskı işaretleri: `marks: crop cross` ve `bleed` — CSS Paged Media yazımıyla.
4. Tam sayfa önizleme kipi: gerçek ölçekte, sayfa sayfa, kılavuzlu.
5. Dijital nüsha zinciri (PDF + sayfa görüntüleri + çevirerek okuyucu)
   doğrulamadan geçmiş çıktıyla üretilir.

**Kabul ölçütü**
- `pdffonts` çıktısında amaçlanan yazı tipi görünür; DejaVu yedeğine düşmez.
- A4 şablonundan üretilen PDF'in sayfa kutusu 210×297mm + bildirilen taşma payı.
- `pdfinfo` TrimBox ve BleedBox bildirir (bugün ikisi de yok).
- Önizlemedeki kesim çizgisi PDF'teki kesim işaretiyle aynı yerde.

**Koruyucu testler**
- `TestGazetePDFSayfaKutusu` — PDF sayfa boyutu şablonun bildirdiğiyle eşleşir.
- `TestGazeteBaskiYaziTipiMevcut` — baskı CSS'inin istediği yazı tipi ya
  sunucuda kurulu ya da projede gömülü; yedeğe düşme sessizce olamaz.

---

### Faz 7 — Matbaa dosyası: PDF/X + preflight  `[K8]`

Faz 6'nın dijital nüshasının yanına, **ayrı** bir matbaa çıktısı ekler.
Dijital nüsha zinciri bundan etkilenmez.

**Yap**
1. `POST /admin/gazete/sayilar/{id}/matbaa` — dijital PDF'i Ghostscript'ten
   geçirip PDF/X üretir ve depolamaya ayrı anahtarla yazar
   (`gazete-matbaa.pdf`), mevcut `gazete.pdf`'in üstüne **yazmaz**.
2. Ghostscript çağrısı: `-dPDFX`, `-sColorConversionStrategy=CMYK`,
   `-sDEVICE=pdfwrite`, ICC profili `-sOutputICCProfile=`.
   **PDF/X sürümü ve ICC profili ayardan okunur** — matbaa hangi sürümü ve
   ICC v2/v4 istiyorsa o. Varsayılan kabul edilmez, sorulur.
3. TrimBox/BleedBox şablonun kesim ve taşma payından üretilir.
4. Preflight: üretilen dosya `pdfinfo` + `pdffonts` ile denetlenir —
   sayfa kutusu, TrimBox/BleedBox varlığı, tüm fontların gömülü olması,
   OutputIntent varlığı. Eksik varsa panelde sebebiyle bildirilir.
5. Panelde "Matbaa dosyası" bölümü: üret, preflight raporu, indir.

**Kabul ölçütü**
- Üretilen dosyada `OutputIntent` var (bugün yok).
- `pdffonts` çıktısında gömülü olmayan font yok.
- TrimBox ve BleedBox şablonun bildirdiği değerlerle eşleşir.
- Dijital nüsha (`gazete.pdf`) değişmeden yerinde durur.

**Koruyucu testler**
- `TestGazeteMatbaaPreflight` — bilerek bozuk bir girdide (font gömülü değil,
  TrimBox yok) preflight hata bildirir, sessiz geçmez.
- `TestGazeteMatbaaDijitaliBozmaz` — matbaa üretimi `pdf_path` kaydını
  değiştirmez.

**Açık soru (kullanıcıya sorulacak):** Matbaa hangi PDF/X sürümünü
(X-1a / X-4), hangi ICC profilini ve ICC v2 mi v4 mü istiyor? Bu cevap
gelmeden varsayılanla üretim yapılmaz — yanlış profil baskıda renk
kaymasına yol açar ve telafisi matbaada olur.

*Not:* veraPDF (JVM) ikinci bir denetim katmanı olarak eklenebilir ama
zorunlu değil; poppler araçları yukarıdaki ölçütleri karşılıyor.
Forma düzeni (imposition) gerekirse `pdfcpu`'nun `booklet`/`nup` komutları
saf Go olduğu için ikiliye girebilir — ayrı karar.

---

## 7. İlerleme

Fazı bitiren ajan bu tabloyu günceller ve `ILERLEME.md`'ye kanıt yazar.

| Faz | Durum | Commit | Kanıt |
|---|---|---|---|
| 1a — Parametrik baskı CSS'i, A4 varsayılan | ✅ | `78957d2` | `gazete_baski_test.go`; A3 tipografisi 0.01pt içinde korunuyor |
| 1b — Editör tek render yoluna geçti | ✅ | | `blokIcerik` silindi; gövde `/blok-html` ucundan geliyor. Ölçüldü: A4 kutusu 794×1123px = referans SVG; tuval ölçeği tipografiyi değiştirmiyor (editör ≡ baskı) |
| 2 — Taşma görünürlüğü, baskı geçidi | ✅ | | Editörde rozet + sayaç, `/dogrula` ucu, PDF geçidi. Chromium'da uçtan uca ölçüldü. Yol boyunca bulundu: sarmalayıcı/iç gövde sınıf çakışması uzun metni yana akıtıp kırpıyordu |
| 3 — Şablon kütüphanesi | ✅ | | 5 hazır peyzaj (referans dahil), şema + doğrulayıcı, stil beyaz listesi. Kapak tarayıcıda ölçüldü: 11 blok yerinde, taşma yok. **Sapma:** sayı oluşturmada sert hata yerine varsayılan şablona düşülüyor |
| 4 — Editör etkileşimi | ✅ | | Sütun ızgarası + hizalama kılavuzu, 8 tutamak, z sırası, stil denetimleri, çoklu seçim, çoğalt/yapıştır, yakınlaştırma. Tarayıcıda ölçüldü. Yol boyunca: editör tuvali 0×0'dı (okuyucu CSS'i baskıyı eziyordu), stil sınıfları tuvalde kayboluyordu |
| 5 — Sayfa yönetimi, güvenli kayıt | ✅ | | Sayfa ekle/sil/sırala/adlandır; tek transaction + iyimser kilit (409). Tarayıcıda ölçüldü: ekle 2→3, sırala, sil 3→2, son sayfa korunuyor |
| 6 — Baskı önizleme, dijital nüsha | ✅ | | Yazı tipi sunucuda kurulu olanlardan seçiliyor (sessiz yedek yok); kesim/güvenli alan kılavuzları; tam önizleme. **Not:** sunucuda yalnız DejaVu Serif var — daha iyi bir serif kurulursa tercih listesi kendiliğinden alır |
| 7 — Matbaa dosyası: PDF/X + preflight | ✅ | | Taşma paylı + kesim işaretli HTML → Chromium → Ghostscript PDF/X + CMYK + Trim/BleedBox → preflight. Uçtan uca ölçüldü: OutputIntent ✓, TrimBox ✓, BleedBox ✓, 658×904pt. **Açık:** ICC profili genel CMYK; matbaanın profili sorulmalı |

---

## 8. Dış referanslar

Desen kaynakları — kopyalanacak kod değil, doğrulanmış yaklaşım:

- **Paged.js** — CSS Paged Media polyfill'i; `@page` içinde `bleed`, `marks: crop cross`
  ve tarayıcıda canlı baskı önizlemesi. Faz 6'nın yazım referansı. Uyarı: `file://`
  üzerinde çalışmıyor, HTTP gerekiyor (bizde sorun değil).
  <https://pagedjs.org/documentation/>
- **Scribus** — taşan metin çerçevesinin sağ-alt köşesine işaret koyar ve taşmada
  **kendiliğinden sayfa/çerçeve açmaz**; akış elle kurulur. K3'ün dayanağı.
  <https://www.linuxjournal.com/content/text-frames-scribus> ·
  <https://archive.flossmanuals.net/scribus-2/composing.html>
- **Affinity Publisher / InDesign** — çerçeve metni ve taşma göstergesi aynı deseni izler.
  <https://affinity.help/publisher2/en-US.lproj/pages/Text/frameText.html>
- **interact.js** — sürükleme/boyutlandırma ve ızgara yakalama "modifier" mantığı;
  görsel hizalama kılavuzu sunmadığı için K5'te alınmadı, yakalama mantığı örnek alındı.
  <https://github.com/taye/interact.js>
- **storytold/designcraft** — Rust'ta sıfırdan yazılmış açık kaynak masaüstü DTP
  motoru (MIT/Apache-2.0, pre-alpha). Bağlı metin çerçeveleri, master sayfalar,
  paragraf/karakter stilleri, heceleme ve blok hizalamalı dizgi; **PDF/X-4 ve
  PDF/A-2b** çıktısı, IDML/INDD içe aktarma, MCP sunucusu.
  **Gömülemez** — Go+templ web paneline Rust masaüstü uygulaması girmez. Değeri
  ikisi: (a) doğru DTP veri modeli referansı (master sayfa + stil + çerçeve
  zinciri), (b) K8'deki PDF/X eksiğini görünür kılması. IDML dışa aktarımı
  ileride "panelde kur, tasarımcı InDesign'da bitirsin" köprüsü olabilir.
  Pre-alpha; üretim zincirine konmaz. <https://github.com/storytold/designcraft>
- **Ghostscript** — PDF → PDF/X + CMYK dönüşümü (`-dPDFX`,
  `-sColorConversionStrategy=CMYK`, `-sOutputICCProfile=`). Faz 7'nin motoru.
  Bu makinede **10.06.0 kurulu**. Çıktı amacı (output intent) gömülü ICC
  profiliyle veya anahtar sözcükle verilebilir.
  <https://ghostscript.readthedocs.io/en/latest/GhostscriptColorManagement.html>
- **poppler** (`pdfinfo`, `pdffonts`, `pdfimages`) — preflight'ın büyük kısmı.
  Zaten kurulu ve `pdftoppm` ile hâlihazırda kullanılıyor.
- **pdfcpu** — saf Go PDF kütüphanesi; `validate`, `booklet`, `nup` (forma
  düzeni). Yığına doğrudan girer, yeni servis gerekmez. Fork'ları çok, upstream
  `pdfcpu/pdfcpu` kullanılmalı. Şimdilik alınmadı — poppler yetiyor.
- Tarandı ve **uygun bulunmadı**: `newspaper-front-page-maker` (canvas tabanlı, hediyelik
  ön sayfa), `forme` (PowerPoint→LibreOffice dönüşümü), `leaflet-editor` (bülten, jsPDF),
  `laydown` / `PageDrop` (imposition ve sayfa yönetimi, tasarım değil).
- **Polotno — açık kaynak DEĞİL.** npm paketi ticari kullanımda ücretli lisans
  istiyor; "açık kaynak Canva" diye anılması yanıltıcı. Planın canvas modelini
  reddetmesi (K5) bu yüzden de doğru. Topluluk alternatifi `OpenPolotno`'nun
  kökeni tartışmalı (kodun Polotno'dan kopyalanıp kopyalanmadığı soruldu) —
  önerilmiyor.
- **Superdesk (Sourcefabric)** — açık kaynak haber odası CMS'i (AGPL v3), ama
  baskı hazır özellik değil: Sourcefabric baskı için "iletişime geçin" diyor.
  Haber üretimi için referans, **baskı mizanpajı için değil**.

---

## 9. Ajanlar için notlar

- `templ` değişirse `make templ` (`templ generate go/internal/webui/templkit/`) çalıştır.
- Baskı CSS'i artık statik dosya: `static/gazete-baski.css`. Sürümlü varlık
  yüklemesine takılma — `staticCacheControl` sürümsüz dosyayı 5 dakika tutar.
- `admin-only.css` / `app.css` ayrımı korunuyor: gazete editörü **yalnız** panelde
  kullanılıyor, kuralları public CSS'e sızmasın. `admincss_test.go` bunu zaten
  koruyor, testleri kırma.
- Veritabanı erişimi: `deploy/.env` içindeki `POSTGRES_PASSWORD` ile
  `docker exec theblock-postgres-1 psql -U theblock -d theblock`.
- WSL üzerinden komut geçerken `$değişken` ve backtick bozulur; betiği dosyaya
  yazıp çalıştır.
