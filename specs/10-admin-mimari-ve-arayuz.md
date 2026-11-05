# theblock — Admin Mimarisi ve Arayüz Kararı

**Sürüm:** 1.0 — 28 Ekim 2026
**Kapsam:** Admin (panel) arayüzünün teknoloji envanteri, karmaşanın kök nedenleri, hedef mimari
kararı ve uygulama fazları (A1–A7).
**İlişkili:** `09-eksikler-ve-yapilacaklar.md` (genel denetim, Faz 125–137) · `03-spec-plan.md` §1.3
("yeni geliştirme Go'da") · `docs/ADMIN-REDIZAYN.md` · `docs/ADMIN-IA.md` · `docs/ADMIN-TEMPL.md` ·
`docs/ADR.md`

> **Yöntem:** Her bulgu komut çıktısıyla kanıtlanmıştır. Ölü şablon kümesi, şablon çağrı
> grafiğinin **geçişli kapanışı** ile hesaplandı (Go kodundaki tüm tırnaklı adlar kök kabul edildi —
> yani hesap *aleyhte*, gerçek ölü küme en az bu kadardır). Kanıtsız kalem listeye girmemiştir.

---

## 1. Yönetici özeti

Admin tarafında **teknoloji seçimi sorunu yok; katman birikmesi sorunu var.**

Yaygın varsayımın aksine **projede React, shadcn/ui, Tailwind veya Radix yok.** `admin-shadcn.css`
269 satır **elle yazılmış** CSS'tir (dosyanın kendi başlığı: *"shadcn/ui (New York, zinc tabanlı)
token'ları ve bileşen kalıpları"*). `editor-src` ve `collab-server` bağımlılıkları yalnızca
TipTap, Yjs, esbuild ve marked. Dolayısıyla **"React'ten çıkmak" diye bir iş yoktur.**

Gerçek sorun iki tamamlanmamış geçiş:

1. **`html/template` → `templ` geçişi bitti ama eski dosyalar silinmedi** → her sayfanın iki
   sürümü var; biri canlı, biri ölü. **4.580 satır / 36 dosya ölü kod.**
2. **shadcn CSS katmanı üçüncü katman olarak eklendi ama eskiler birleştirilmedi** → aynı
   değişken ve aynı bileşen sınıfı 2–3 dosyada tanımlı; yükleme sırası son sözü söylüyor →
   öngörülemeyen görünüm ("düzen bozuk").

Buna bir de **koşulsuz varlık yükleme** (her admin sayfası 1.087 KB, %71'i tek bir editör bundle'ı)
ve **üç paralel gezinme paradigması** eklenince kullanıcının tarif ettiği karmaşa oluşuyor.

**Karar:** Go + `templ` + HTMX tamamlanır. shadcn **tasarım dili** olarak tutulur, **bağımlılık**
olarak alınmaz. Yeniden yazım yoktur; A1–A3 işinin tamamı **silme ve kablolama**dır.

---

## 2. Teknoloji envanteri (ölçülmüş gerçek durum)

| Katman | Nerede | Boyut | Durum |
|---|---|---|---|
| Go `html/template` | `webui/templates/*.html` | **12.554 satır / 72 dosya** | Eski katman — yalnız **6** canlı giriş noktası kaldı |
| `templ` | `webui/templkit/*.templ` | **9.798 satır / 27 dosya** · 65 bileşen · 46 panel | **Fiili yeni standart** — 42 `renderTemplPage`, 20 `*_view.go` |
| HTMX | `static/vendor/htmx.min.js` | 50.917 bayt | Panel etkileşim modeli (Faz 49–99) |
| "shadcn" | `static/admin-shadcn.css` | 269 satır / 10.836 bayt | **React değil** — elle yazılmış token taklidi |
| Diğer admin CSS | `app.css` · `admin-theme.css` | 858 + 158 satır | shadcn ile **çakışıyor** (§3.3) |
| Admin JS | `static/*.js` (18 dosya) | 5.552 satır | 16'sı **her** sayfada yükleniyor |
| İçerik editörü | `editor-src/src` → `static/editor.js` | 90 KB kaynak → **770.939 bayt** bundle | TipTap + 7 özel düğüm; istemci↔sunucu **uyumlu** |
| Ortak düzenleme | `collab-server/` (Node) | ayrı süreç | Yjs / Hocuspocus |

### Hâlâ `html/template` ile basılan canlı sayfalar (yalnız 6)
| Şablon | Dosya | Satır | Not |
|---|---|---|---|
| `tasarimci` | `site_builder.html` | **2.564** | Sayfa Tasarımcısı — §5'te ayrı karar |
| `admin_gorunum` | `site_gorunum.html` | 142 | küçük |
| `merkez` | `site_center.html` | 101 | iframe hub'ı — §3.4 |
| `admin_bilesenler` | `site_bilesenler.html` | 6 | **templ sayfalarının kabuğu** (köprü) |
| `login` | `site_admin1.html` | 12 | küçük |
| `2falogin` | `site_admin10.html` | 12 | küçük |

`admin_bilesenler` kritik: templ içeriği bu 6 satırlık kabuğa enjekte edilip `admin_head` /
`admin_side` / `admin_top` ile sarılıyor. Yani templ ve `html/template` **düşman değil**,
biri diğerinin içinde çalışıyor.

---

## 3. Bulgular

### 3.1 · S2 · 4.580 satır ölü admin HTML (36 dosya, 50 ulaşılamaz şablon)

**Kanıt:** 177 `{{define}}` · Go'dan çağrılan 75 kök · geçişli ulaşılabilir 127 → **50 ölü**.
Hiçbir Go dosyası `*_panel` şablonunu adıyla çağırmıyor; eski `render*Panel` yardımcılarının
tamamı artık templ V2 görünümlerine yönleniyor (örn. `renderCatsPanel` → `renderKategorilerPanel`,
`renderSEOPanel` → `renderSeoMerkezPanelV2`, `renderMansetPanel` → `renderMansetPanelV2`).
Şablon adı hiçbir yerde dinamik üretilmiyor (tüm render `renderTemplPage` + tipli bileşen),
bu yüzden ölü küme güvenilir.

**En büyük ölü dosyalar ve canlı karşılıkları:**
| Ölü dosya | Satır | Canlı karşılığı |
|---|---|---|
| `content_form_partial.html` | 844 | `templkit/icerikform.templ` (1.599) |
| `manset_partial.html` | **646** | `templkit/manset.templ` (202) |
| `media_partial.html` | 324 | `templkit/medya.templ` (335) |
| `bot_partial.html` | 284 | templ paneli |
| `menu_partial.html` | 252 | `templkit/menu.templ` (257) |
| `settings_partial.html` | 182 | `templkit/sistem.templ` (384) |
| `footer_partial.html` | 165 | templ paneli |
| `dashboard_partial.html` | 156 | `templkit/panel.templ` (258) |
| `tabcats_partial.html` | 135 | templ paneli |
| `contents_partial.html` | 129 | `templkit/icerikler.templ` (405) |
| `ai_source_partial.html` · `share_partial.html` · `indexing_partial.html` | 117 · 114 · 105 | templ panelleri |
| `reports_partial.html` · `marka_partial.html` · `categories_partial.html` | 101 · 90 · 89 | templ panelleri |
| + 20 dosya daha (≤ 80 satır) | | |

**Etkisi:** Bu, kullanıcının tarif ettiği *"aynı mantık işi yapan alanlar"*ın birebir kaynağıdır.
Bir sayfayı düzenlemek isteyen (insan veya ajan) iki dosya görür ve yanlış olanı düzenlerse
değişiklik canlıya yansımaz — sessiz, tekrar eden bir zaman kaybı.

### 3.2 · S2 · Her admin sayfası 1.087 KB varlık indiriyor; %71'i gereksiz

**Kanıt:** `partials.html` `admin_head` (satır 272–295) **koşulsuz** 3 CSS + 16 JS yüklüyor.

| Dosya | Bayt | Gerçekten gerekli olduğu yer |
|---|---|---|
| **editor.js** | **770.939** | yalnız içerik formu |
| app.js | 76.363 | genel |
| app.css | 64.394 | genel |
| vendor/htmx.min.js | 50.917 | genel |
| manset.js | 32.592 | yalnız manşet editörü |
| ai_editor.js | 18.880 | yalnız içerik formu |
| admin-ui.js | 18.265 | genel |
| admin-shadcn.css · admin-theme.css | 10.836 · 10.281 | genel |
| audio_panel.js · ses_player.js | 10.421 · 8.776 | yalnız ses paneli |
| menu.js · footer.js · media.js · tabcats.js · header.js | 7.463 · 7.034 · 6.818 · 5.466 · 4.596 | yalnız ilgili sayfa |
| ai-sohbet.js · marka.js · raporlar.js · kumeler.js | 3.463 · 2.966 · 2.045 · 1.061 | yalnız ilgili sayfa |
| **TOPLAM** | **1.113.576 (1.087 KB)** | |

Kontrol paneli, kategoriler, ayarlar gibi sayfalar TipTap + Yjs bundle'ını ve manşet motorunu
boşuna indiriyor. Sayfa başına yüklemeye geçildiğinde tipik sayfa **~300 KB**'a iner.

### 3.3 · S2 · Üç CSS katmanı çakışıyor — "düzen bozuk"un kök nedeni

Yükleme sırası: `app.css` → `admin-theme.css` → `admin-shadcn.css` (son yüklenen kazanır).

**Çakışan değişkenler (app.css ∩ admin-shadcn.css):** `--bg` · `--border` · `--radius` ·
`--shadow` · `--text`
*(`admin-theme.css` hiç değişken tanımlamıyor ama sınıf tanımlıyor.)*

**İki veya üç dosyada tanımlı bileşen sınıfları (22 adet):**
`.btn` `.card` `.card-pad` `.input` `.table` `.badge` `.flash` `.admin-body` `.admin-side`
`.admin-top` `.admin-crumb` `.kpi` `.icon-btn` `.ui-modal` `.status-pill` `.qtile` `.form-bar`
`.tbl-tools` `.tbl-pager` `.cmdk-box` `.cmdk-list` `.auto-row`

**Etkisi:** `app.css` değerleriyle tasarlanmış bileşenler shadcn değerleriyle çiziliyor.
Bir rengi veya yarıçapı değiştirmek için hangi dosyaya dokunulacağı belirsiz; bir yerde
düzeltilen şey başka sayfada bozuluyor. Kullanıcının *"manşet arayüz yönetim düzen bozuk"*
tarifi bu belirsizliğin semptomu.


> ### ⚠ DÜZELTME (29 Ekim 2026) — yukarıdaki §3.3 bulgusu kısmen yanlıştı
>
> "22 bileşen sınıfı iki-üç dosyada tanımlı → öngörülemeyen görünüm" tespiti,
> ölçüm yapılmadan yazılmıştı. Ölçüldüğünde tablo değişti:
>
> **1) Değişken çakışması bir hata değil, bilinçli tema katmanı.**
> `app.css` public sitenin **tek** token kaynağıdır: `--bg --text --border
> --radius --shadow` orada tanımlıdır ve app.css içinde **98 kez** kullanılır
> (`--border` 54, `--text` 26, `--radius` 10, `--shadow` 7, `--bg` 1).
> `public.css` bu beşinden hiçbirini kullanmaz. `admin-shadcn.css` yalnız
> admin'de yüklenir ve bu beşini **kasıtlı olarak** shadcn değerleriyle ezer.
> Admin ve public'in farklı görünmesi tasarımdır. Bu tanımları app.css'ten
> çıkarmak **public siteyi bozardı**.
>
> **2) 22 sınıfın 8'i gerçekten paylaşımlıdır.** Public şablonlardaki kullanım
> sayıları: `.btn` 82, `.card` 106, `.card-pad` 59, `.input` 67, `.table` 8,
> `.badge` 21, `.flash` 14 (ayrıca `site_header` içinde `.btn` 8, `.icon-btn`
> 5). Yani `app.css` = public+admin **ortak temel**, `admin-shadcn.css` =
> admin **derisi**. Bu "base + skin" deseni doğrudur; birleştirilecek bir
> çakışma değildir.
>
> Dolayısıyla §3.3'ün "tek token kaynağına indir" önerisi ve M3 kararı bu
> hâliyle uygulanamaz.

### 3.3b · S3 · Asıl bulgu: admin CSS'i her ziyaretçiye iniyor

Ölçüm (29 Ekim 2026) gerçek sorunu başka yerde gösterdi.

`app.css` **64.982 bayt** ve `public_head` ile **her public sayfa
görüntülemesinde** indirilir. İçindeki 346 üst düzey sınıfın dağılımı:

| | Sayı |
|---|---|
| Yalnız admin'de kullanılan (public şablonlarda **0**) | **178** |
| Public'te de kullanılan (gerçekten paylaşımlı) | 97 |
| Kalan (iki tarafta da eşleşmeyen) | 71 |

Admin-only sınıflar app.css'in **~281 / 865 satırını (~%32)** kaplıyor.
Örnekler: `ai-drawer*` (12 sınıf), `ai-select-*`, `be-btn`, `bg-preview*`,
`danger-zone`, `cat-row-child`, `chip-view`, `count-badge`, `bar-*`, `bars`.
Bunların hiçbiri ziyaretçiye lazım değil.

Kaba tahminle **~20 KB admin CSS'i her ziyaretçiye gidiyor.** Faz 131'de
admin tarafında yapılan iş (1.093 KB → 266 KB) burada public tarafın
karşılığıdır.

**Revize edilmiş A3:** "tek token kaynağı" değil, **katman ayrımı** —
admin-only kuralları `app.css`'ten alıp admin katmanına taşımak.
Paylaşımlı sınıflar (`.btn`, `.card`, `.input` …) ve token'lar app.css'te
**kalır**; base+skin deseni korunur.

**Risk:** CSS taşımada sıra ve özgüllük önemlidir. Taşınan kural admin'de
`app.css`'ten **sonra** yüklenen bir dosyaya giderse, aynı özgüllükteki
çakışmalarda kazanan değişebilir. Bu yüzden taşıma hedefi, yükleme sırasında
`app.css`'in hemen ardından gelen bir dosya olmalı ve her taşınan sınıf için
etkin özellik kümesi testle sabitlenmelidir.

**Durum: tamamlandı — Faz 135 (29 Ekim 2026).** `static/admin-only.css`
ayrıldı; 383 kural taşındı, public sayfa CSS yükü 335.872 → 298.527 ham
(gzip 71.768 → 64.490, %10,1). Sıra riski ölçüldü: **0 çift**. Kural düzeyinde
kayıp yok (667 = 667, yorum 73 = 73). Koruma:
`internal/webui/admincss_test.go` — iki yönlü sızma ve yükleme sırası.
Ayrıntı: `docs/ILERLEME.md` Faz 135.

Açık kalan: gözle doğrulama otomatik değil; hem admin hem public görünümü tek
dosyada tutan 6 şablon yüzünden 4 kural (`.admin-body`, scrollbar) `app.css`'te
kaldı. **Sıradaki asıl hedef `public.css`:** 270.890 bayt (gzip 57.168), artık
`app.css`'in 10 katı ve public CSS yükünün %91'i.

**Kabul:** `app.css` içinde public şablonlarda kullanılmayan sınıf kalmamış ·
public sayfa CSS yükü ölçülüp kaydedilmiş · admin ve public görünümü gözle
doğrulanmış · test, admin-only sınıfın app.css'e geri sızmasını engelliyor.

### 3.4 · S3 · Üç paralel gezinme paradigması

61 parametresiz admin sayfası (80 admin GET rotası) için:

| Yol | Kapsam | Kabuk |
|---|---|---|
| Kenar menü (`admin_side`) | **11** link | normal |
| "Merkez" hub'ı (`sitecenter.go`) | **23** kart | **iframe** (`?embed=1`, `tmpl.go:221` — menü/üst bar gizlenir) |
| Komut paleti (`admin-ui.js`) | **12** `/admin/` referansı | normal |

Aynı sayfaya üç farklı yoldan, üç farklı kabukla ulaşılıyor; biri iframe içinde. Kenar menü
61 sayfanın yalnız 11'ini kapsıyor. Üç liste birbirinden bağımsız elle bakımda — tek kaynak yok.
Kullanıcının *"bağlam alakası olmayan alanlar"* hissi buradan geliyor.

### 3.5 · S3 · Sayfa Tasarımcısı geçişin dışında kaldı

`site_builder.html` **2.564 satır** + `builder_handlers.go` **2.562 satır** = en büyük admin
yüzeyi, hâlâ `html/template`. Detaylı analiz ve karar §5'te.

### 3.6 · S3 · İçerik editörü: alan kaybı yok, 5 ölü bağımlılık var

**İyi haber (ölçülmüş):** templ geçişinde **form alanı kaybedilmemiş**. Eski ölü
`content_form_partial.html` 30 alan adı, canlı `icerikform.templ` 36 alan adı; eski küme
yeninin **tam alt kümesi**. Yeni eklenenler: `dil`, `konusmaci`, `min_tier_weight`, `onay`,
`q`, `ses`.

**İstemci ↔ sunucu düğüm kümeleri de uyumlu.** Editör şu özel düğümleri tanımlıyor:
`callout` · `embed` · `toggle` · `bookmark` · `mediaCard` · `uploadPlaceholder` · `gallery`
(+ `slashCommand`, öneri/imleç altyapısı). `tiptap.go` bunların hepsini ve ek olarak
`table*`, `audio`, `video`, `youtube`, `x` varyantlarını basıyor. **Uyumsuzluk yok.**

**Gerçek eksikler:**
| Sorun | Kanıt |
|---|---|
| 5 ölü npm bağımlılığı | `extension-text-align` · `extension-youtube` · `extension-table-cell` · `extension-table-header` · `extension-table-row` — `package.json`'da var, hiç `import` edilmiyor |
| **Metin hizalama özelliği yok** | `TextAlign` yüklenmiyor **ve** `tiptap.go` `textAlign` basmıyor. Tutarlı ama eksik bir yetenek |

*(Tablo desteği `TableKit` üzerinden çalışıyor, YouTube gömme özel `embed` düğümüyle çalışıyor —
o yüzden ilgili bağımlılıklar gereksiz, bozuk değil.)*

### 3.7 · S4 · Yetim sayfa sorunu neredeyse yok — önceki varsayım yanlıştı

Yalnız kenar menü tarandığında 28 sayfa "yetim" görünüyor. Tüm `templkit` + `templates` +
`static/*.js` kaynakları tarandığında **gerçek yetim 3**:

| Rota | Değerlendirme |
|---|---|
| `/admin/adsense/callback` | OAuth callback — linksiz olması **doğru** |
| `/admin/2fa` | Gerçek yetim — menüye eklenmeli (hesap güvenliği) |
| `/admin/tasarim/kategori-gorunum` | Gerçek yetim — Tasarım menüsüne bağlanmalı |

**Sonuç:** Sorun ölü *rota* değil, ölü *şablon* (§3.1) ve dağınık *IA* (§3.4). Bu ayrım önemli:
A5'in kapsamı "sayfa silmek" değil, "var olan 61 sayfayı tek menüde toplamak"tır.

---

## 4. Mimari karar

> **Admin, Go + `templ` + HTMX ile tamamlanır. shadcn tasarım dili olarak tutulur, bağımlılık
> olarak alınmaz. Ağır istemci yüzeyleri (içerik editörü, manşet tuvali, sayfa tasarımcısı)
> framework değil, kapsamı sınırlı JS adaları olarak kalır.**

### Gerekçeler (kanıta dayalı)

1. **Geçiş fiilen bitmiş.** Canlı admin sayfalarının neredeyse tamamı templ'den basılıyor
   (42 `renderTemplPage`, 65 bileşen, 46 panel, 20 `*_view.go`). React'e geçmek 9.798 satır
   templ'i çöpe atıp 63.059 satır Go handler'ı API'ye çevirmek demek.
2. **Projenin kendi ilkesi** (`03-spec-plan.md` §1.3): *"Yeni geliştirme Go'da."*
3. **Ekip ölçeği tek kişi + ajanlar.** İkinci ekosistem CI/build/bağımlılık yüzeyini ikiye
   katlar. Mevcut JS yüzeyinin bakımsızlığı bunu zaten gösteriyor: 5 ölü bağımlılık,
   her sayfada 770 KB bundle, `editor.js` dışında test edilmeyen 18 dosya.
4. **shadcn denemesi React olarak değil CSS taklidi olarak indi.** Bu bir başarısızlık değil,
   **sinyal**: gerçek shadcn Tailwind + React + bundler zinciri getirir; bu projede karşılığı
   ölçülebilir bir kazanç değil. Token sözlüğü ise zaten işe yarıyor — tutulmalı.
5. **Yeniden yazım gerekmiyor.** A1–A3 işinin tamamı silme ve kablolama; davranış değişmediği
   için mevcut **569 Go testi** ve **56 E2E dosyası** doğrudan güvenlik ağı.

### Hedef mimari (7 karar)

| # | Karar |
|---|---|
| **M1** | **Tek şablon dili: `templ`.** Admin'de `html/template` yalnız `admin_bilesenler` kabuğu + giriş sayfaları olarak kalır; panel gövdeleri tamamen templ. (Public taraf bu spec'in kapsamı dışında.) |
| **M2** | **Etkileşim: HTMX.** Yeni JS framework eklenmez. |
| **M3** | **Tek token kaynağı:** token'lar `admin-tokens.css`'e çıkar. `app.css` ve `admin-theme.css` token **tanımlamaz**, yalnız tüketir. `.btn/.card/.input/.table` gibi bileşen sınıfları **tek dosyada** tanımlı olur. |
| **M4** | **Bileşen kütüphanesi:** `templkit/bilesenler.templ` pilotu gerçek kütüphaneye çevrilir (Button · Card · Input · Table · Badge · Dialog · Toast). Sayfalar ham HTML yazmaz, bileşen çağırır. **shadcn vokabüleri, templ gövdesi.** |
| **M5** | **Varlık yükleme sayfa başına:** templ layout'a `Scripts []string` / `Styles []string`. `admin_head` koşulsuz liste tutmaz. `editor.js` yalnız içerik formunda, `manset.js` yalnız manşet editöründe. |
| **M6** | **Tek IA:** iframe hub'ı (`?embed=1`) kaldırılır veya gerçek sayfalara yönlendiren bir dizine dönüşür. Kenar menü 61 sayfayı **gruplu** kapsar. Komut paleti menü tanımından **türetilir** (tek kaynak). |
| **M7** | **Ağır yüzeyler ada:** içerik editörü, manşet tuvali ve sayfa tasarımcısı kendi CSS/JS kapsamında kalır; paylaşılan **token**'ları tüketir, paylaşılan **sınıf**ları kullanmaz. |

---

## 5. Sayfa Tasarımcısı: taşınmaz, "izole ada" ilan edilir

**Bağlam:** Kullanıcı bu aracı aktif kullanıyor ve değerli buluyor. Bu, kararı belirler.

**Ölçülmüş durum — zaten fiilen bir ada:**
| Bulgu | Değer | Anlamı |
|---|---|---|
| Kendi `<style>` bloğu | 243 satır | Stili kendi içinde |
| Kendi `<script>` bloğu | **2.011 satır, şablona gömülü** | `app.js`'te builder kodu **0** referans |
| Kendi ön ekli sınıf aileleri | `opt-` 49 · `bld-` 47 · `bp-` 29 · `ab-` 9 · `kp-` 8 · `cm-` 5 · `hs-` 4 · `catview-` 3 | Kendi isim alanı var |
| Paylaşılan 22 sınıftan kullanımı | **yalnız 46 yer**: `.btn` 28 · `.card`+`.card-pad` 12 · `.input` 2 · `.badge` 2 · `.flash` 1 · `.admin-body` 1 | Çakışma yüzeyi küçük; hepsi genel çerçeve, tuval değil |
| Harici varlık | yalnız `public.css` + vendor | Admin CSS'ine zayıf bağlı |
| HTTP ucu | 10 handler | Sınır net |
| Test kapsamı | `builder_test.go` **57 test** + `builder.test.js` E2E + `_verify_builder.js` | **Güçlü koruma mevcut** |

**Karar ve gerekçe:** templ'e taşımak, her gün kullanılan 5.126 satırlık bir aracı görünür hiçbir
kazanç olmadan yeniden yazmak olur — planın en yüksek riskli, en düşük getirili işi. Araç zaten
izole ve zaten iyi test edilmiş. **Taşımak yerine, doğru olanı resmîleştir:**

1. **Ada ilan et ve `docs/ADR.md`'ye yaz.** Böylece "neden burası farklı?" sorusu karmaşa değil,
   kayıtlı bir karar olur.
2. **Token'ları paylaş, sınıfları paylaşma.** 243 satırlık `<style>` bloğu renk/yarıçap/gölgeyi
   `admin-tokens.css` değişkenlerinden okusun → tema değişince uyar, çakışma yaşamaz.
3. **Tek gerçek iyileştirme: 2.011 satırlık gömülü `<script>`'i `static/tasarimci.js`'e çıkar.**
   Mekanik iş, davranış değişmez, 57 Go testi + E2E koruyor. Kazanç: önbelleğe girer, her sayfa
   yüklemesinde yeniden ayrıştırılmaz, lint edilebilir ve şablonun içinde kaybolmaz.
4. **A3 sırasında o 46 paylaşılan sınıf kullanımını gözle doğrula.** Liste yukarıda, az sayıda.

Bu yaklaşımla A6, "yüksek riskli 5.126 satır taşıma"dan **yarım günlük JS çıkarma + bir ADR
kaydı**na iner.

---

## 6. Faz planı (A1–A7)

Sıra bağımlılığa ve risk/getiri oranına göre belirlendi.

| Faz | İş | Bulgu | Risk | Neden bu sırada |
|---|---|---|---|---|
| **A1** | **Ölü HTML katmanını sil** (36 dosya, 4.580 satır, 50 şablon) | §3.1 | **Düşük** | Karmaşanın en büyük tek kaynağı; davranış değişmiyor; A3'ün ön koşulu |
| **A2** | **Varlık yüklemeyi sayfa başına indir** (`Scripts []string`) | §3.2 | Düşük | Mekanik; 1.087 KB → ~300 KB; anında hissedilir |
| **A3** | **CSS tek token kaynağı + 22 sınıf çakışmasını bitir** | §3.3 | **Orta** | "Düzen bozuk" kökten çözülür; görsel regresyon riski gerçek |
| **A4** | **templkit bileşen kütüphanesini tamamla**; sayfalar bileşen kullansın | M4 | Orta | Token'lar A3'te sabitlenmeden bileşen yazmak kayan zemine inşa etmek olur |
| **A5** | **IA birleştirme**: iframe hub'ını kaldır, tek gruplu menü, komut paletini menüden türet, 2 yetim sayfayı bağla | §3.4, §3.7 | Orta | A1 sonrası sayfa envanteri netleşince anlamlı |
| **A6** | **Sayfa Tasarımcısı**: ada ilanı + ADR + gömülü JS'i `tasarimci.js`'e çıkar + token paylaşımı | §3.5, §5 | Düşük | A3'ten sonra: token'lar hazır olunca bağlanır |
| **A7** | **Editör temizliği**: 5 ölü npm bağımlılığını sil; TextAlign kararı (ekle ya da kaldır) | §3.6 | Düşük | Bağımsız, her an yapılabilir |

### A1'in ön koşulu (atlanamaz)
`09-eksikler-ve-yapilacaklar.md` **E-12**: E2E paketi (56 dosya) hiç çalıştırılmadı, durumu
bilinmiyor. **A1'den önce `make e2e-install && make e2e` bir kez çalıştırılıp sonuç
`docs/ILERLEME.md`'ye yazılmalı.** Silme işleminin referans noktası budur; yeşil olduğu
bilinmeyen bir pakete güvenerek 4.580 satır silinmez.

### Bağımlılık haritası
```
(E2E referansı) ──► A1 (ölü HTML sil) ──► A2 (varlık yükleme) ──► A3 (CSS tek kaynak)
                      │                                              │
                      └──────────► A5 (IA birleştirme)               ├──► A4 (bileşen kütüphanesi)
                                                                     └──► A6 (tasarımcı adası)
A7 (editör temizliği) — bağımsız, her an
```

### `09`'un faz hattıyla ilişki
`09`'daki Faz 125–137 (commit, yedekleme, RSS, smoke/CI, constraint…) ile A1–A7 **çakışmaz**;
farklı dosyalara dokunurlar. Ancak sıralamada şu uyarı geçerlidir:

> **`09` E-1 (yedekleme hiç çalışmamış) hâlâ projenin tek en büyük riskidir.** Arayüz karmaşası
> can sıkıcıdır; yedeksizlik **geri dönüşsüzdür**. Önerilen: Faz 126'yı (yarım günlük iş) araya
> al, sonra A1–A3'e tam odaklan; `09`'un 127–129'u (RSS, smoke/CI, constraint) arayüz işinden
> sonra gelebilir.

---

## 7. Kabul kriterleri

| Faz | Kabul |
|---|---|
| A1 | `go test ./...` + `go vet` temiz · E2E A1 öncesiyle **aynı** sonuç · `templates/` altında ulaşılamaz `{{define}}` **0** (geçişli analiz tekrar koşar) · canlı smoke bozulmamış |
| A2 | Kontrol paneli yanıtında `editor.js` **yok** · içerik formunda **var** · tipik admin sayfası varlık toplamı **< 400 KB** · E2E yeşil |
| A3 | Hiçbir CSS değişkeni birden fazla dosyada tanımlı **değil** · hiçbir bileşen sınıfı birden fazla dosyada tanımlı **değil** · 6 canlı sayfa + tasarımcı gözle doğrulanmış · E2E yeşil |
| A4 | `templkit` bileşen kütüphanesi en az 7 bileşen · yeni/düzenlenen panellerde ham `<button>`/`<input>` yok |
| A5 | Tek menü tanımı 61 sayfayı kapsar · komut paleti aynı tanımdan türer · `?embed=1` iframe yolu kalkmış veya belgelenmiş · `/admin/2fa` ve `/admin/tasarim/kategori-gorunum` bağlı |
| A6 | `site_builder.html` içinde gömülü `<script>` **yok** · `static/tasarimci.js` mevcut · `builder_test.go` 57 testi + `builder.test.js` yeşil · ADR kaydı yazılı |
| A7 | `package.json`'da import edilmeyen bağımlılık **0** · TextAlign ya çalışıyor (istemci + `tiptap.go`) ya da bağımlılık kaldırılmış |

---

## 8. Bu denetimin kapsamadıkları

Dürüstlük adına — "yeşil" varsayılmamalı:

- **E2E paketi çalıştırılmadı** (56 dosya). A1'in ön koşulu olarak işaretlendi.
- **Görsel/kullanılabilirlik denetimi yapılmadı.** Sayfalar tarayıcıda tek tek gezilmedi;
  bulgular statik analiz, varlık ölçümü ve şablon çağrı grafiğine dayanıyor. Kullanıcının
  tarif ettiği *"manşet arayüz yönetim düzen bozuk"* ve *"içerik editörü eksikleri"*nin
  **yapısal** nedenleri (§3.1, §3.3, §3.6) tespit edildi; **görsel** kusurların tam listesi
  için sayfa sayfa gezinti gerekir.
- **Erişilebilirlik (a11y) denetimi** kapsam dışıydı.
- **Public taraf** (`site_public.html` 2.208 satır, `public.css` 4.677 satır) incelenmedi;
  bu spec yalnız admin'i kapsar.
- **Yetki matrisi / çok-kiracılı izolasyon** incelenmedi — `09` §7'de ayrı denetim olarak işaretli.
