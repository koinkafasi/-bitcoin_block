# theblock — Bot ve İçerik Akışı: Denetim, Yapılanlar ve Kalanlar

**Sürüm:** 1.0 — 28 Ekim 2026
**Kapsam:** Bot (RSS/ajans) hattı, trend merkezi kapsama ölçümü ve içerik
triyajı. B1–B6 bulguları, uygulama durumları ve U1–U4 olarak adlandırılan
arayüz işleri.
**İlişkili:** `09-eksikler-ve-yapilacaklar.md` (genel denetim, E-n) ·
`10-admin-mimari-ve-arayuz.md` (admin mimarisi, A1–A7) ·
`docs/ILERLEME.md` (uygulama kayıtları) · `docs/TREND-MERKEZI-PLANI.md`

> **Yöntem:** Her bulgu canlı veritabanı sorgusu, 24 saatlik servis günlüğü
> veya `curl` ile doğrulanmıştır. Sayılar 28 Ekim 2026, 05:45 itibarıyladır.
> Bu dosya, önceki oturumda yalnız sohbet geçmişinde kalan denetim
> bulgularını kalıcı hale getirir.

---

## 1. Durum özeti

| # | Bulgu | Önem | Durum |
|---|---|---|---|
| B1 | Trend kapsama göstergesi tek pencereye bakıyordu | S2 | ✅ **Tamam** |
| B2 | Bot aynı haberi farklı URL'lerle tekrar tekrar taslağa düşürüyordu | S2 | ✅ **Tamam** (bir bilinen boşlukla) |
| B3 | Kaynak hiyerarşisi tersine çalışıyordu (agregatör "orijinal" sayılıyordu) | S2 | ✅ **Tamam** |
| B4 | `ai_score` yalnız panelde hesaplanıyordu → kapı hiç çalışmıyordu | S2 | ✅ **Tamam** |
| B5 | Taslakların %77'si tek kategoride toplanıyordu | S2 | ✅ **Tamam** |
| B6 | "11 gündür yanlış spor verisi" iddiası | — | ⛔ **Geri çekildi** (yanlış alarm) |
| E-2 | RSS çekimlerinin %52'si hata veriyordu | S2 | ✅ **Tamam** (veri düzeltmesi kısmi) |
| — | 10.442 mükerrer kaydın birleştirilmesi | S3 | ✅ **Uygulandı** (9.668 arşivlendi) |

**Ölçülen sonuç:**

| Gösterge | Önce | Sonra |
|---|---|---|
| Gündem kategorisindeki taslak payı | %77 (30.021) | **%29** (8.590) |
| `ai_score = 0` olan taslak | 39.209 | **2** |
| Ortalama `ai_score` | ~0 | **72** |
| Mükerrer taslak | 10.442 fazla | **9.668 arşivde**, 361 grup dokunulmadı |
| Feed çekim hatası (24 saat) | 3.321 / 6.378 | 8 kaynak (hepsi RSS adresi olmayan) |
| Taslak toplamı | 39.211 | 29.634 (arsivleme sonrası) |

---

## 2. B1 — Kapsama penceresi

**Bulgu.** Kapsama analizi tek bir 72 saatlik pencereye bakıyordu; site o
pencerede yayın yapmadığı için 1.635 terimin tamamı `coverage_count=0`
görünüyordu. Panel "hiçbir konuda içeriğimiz yok" diyordu.

**Karar.** İki soru iki ayrı sayıdır:

- `Total` (180 gün): "bu konuda arşivimizde içerik var mı?"
- `Recent` (72 saat): "bu haber döngüsünde işledik mi?" — skorlamadaki
  boşluk sinyali yalnız buna bakar.

**Kanıt.** `internal/trends/coverage.go` + 5 test; `repo.TrendCoverageHours`
/ `TrendRecentHours`; `ListTrendMatchContents` `published_at` döndürür ve
limit 500 → 5.000. Migration gerekmedi.

---

## 3. B2 — Yakın-kopya denetimi

**Bulgu.** 1.923 başlık grubunda 10.442 fazladan kayıt; en çok tekrarlayan
haber 122 kopya. Kök neden: yalnız `crawled_items.url_hash` denetimi.

**Uygulama.** `repo/bot_dedupe.go`: trigram indeksiyle aday daraltma (SQL,
indeksli) + `quality.CheckDuplicates` (shingle + Jaccard/kapsama, saf
fonksiyon). Mükerrer öğe kaydedilmez, URL "çekildi" işaretlenir.

**Bilinen boşluk (kapatılmadı, kayıtlı).** `ShingleSize = 5` olduğu için
başlık **ve** spot birlikte yeniden yazıldığında benzerlik 0.60'ın altında
kalıyor. Test `t.Skip` ile gerekçesi ve çözüm adaylarıyla duruyor
(`bot_dedupe_test.go`). Ölçüm, 10.442 kaydın tamamının aynı/çok yakın metin
sınıfında olduğunu gösteriyor; yani bu boşluk kayıtların küçük bir bölümünü
etkiliyor.

---

## 4. B3 — Kaynak hiyerarşisi

**Bulgu.** MSN Türkiye 10.764 taslakla en büyük kaynaktı ve bir agregatördür.
Bot onu sıradan kaynak gibi çektiği için aynı haber önce MSN'den taslağa
düşüyor, sonra asıl kaynaktan gelen kopya "mükerrer" sayılıyordu.

**Karar.**
1. `feeds.aggregator` (migration **0089**); MSN, Google News ve `bingnews`
   türü işaretlendi (91 feed'in 2'si).
2. Çekim sırası: `aggregator ASC, trust DESC` — orijinaller önce.
3. Agregatörlerde benzerlik eşiği **0.45**, orijinallerde 0.60.
4. `DedupeDecision` (saf fonksiyon): kayıtlı kopya agregatörden, gelen öğe
   orijinalden ise **orijinal kazanır**; agregatör kopyası `arsiv`'e alınır.

**Neden arşiv, neden silme değil.** Yanlış bir hiyerarşi kararı veri kaybına
dönüşmemeli. `UPDATE contents SET status='taslak' WHERE status='arsiv'` ile
geri alınabilir.

---

## 5. B4 — Alaka puanı

**Bulgu.** 39.211 taslağın 39.209'u `ai_score = 0`. Kategori yayın kapısı
(`categories.min_score`) bu alana baktığı için hiçbir taslakta çalışmıyordu.
Eşiği açmak, tüm taslakları sessizce durdururdu — yani kapı "yeşil" görünüp
hiçbir şey yapmıyordu.

**Karar.** Hesap tek kaynakta: `internal/scoring.Relevance`. Panel
(`httpapi.aiRelevanceScore`) ve bot (`repo.CrawlFeed`) aynı fonksiyonu
çağırır; `CreateContent` `ai_score` yazar. `cmd/backfill -is=skor` mevcut
kayıtları doldurur.

**Sınır.** Bu puan bir *alaka/seçilebilirlik* göstergesidir, editör yargısı
değildir. Eşik açmadan önce dağılım incelenmeli (ortalama 72; 60–79 bandında
26.428 taslak var).

> ### ÖLÇÜM (29 Ekim 2026) — yukarıdaki sayılar eskidi, sonuç da değişti
>
> **Doldurma yapılmış.** 34.156 taslağın yalnız **2'sinde** `ai_score = 0`.
> Yani "39.209 taslakta puan yok" bulgusu kapandı; `cmd/backfill -is=skor`
> kuru koşumu artık 2 kayıt buluyor. Buna bağlı olarak, Faz 132'de eklenen
> triyaj sıralaması/eşiği için daha önce yazılan "skorlar dolmadan anlamsız"
> uyarısı **geçersizdir** — veri hazır.
>
> **Ama eşik hâlâ açılamaz, ve sebebi dağılımın darlığı:**
>
> | bant | taslak | pay |
> |---|---|---|
> | 0 | 2 | %0,0 |
> | 1–29 | 49 | %0,1 |
> | 30–59 | 2.082 | %6,1 |
> | **60–79** | **30.017** | **%87,9** |
> | 80+ | 2.006 | %5,9 |
>
> Taslakların **%88'i tek bir 20 puanlık bantta**. `min_score` için kullanışlı
> bir ara değer yok: 60 eşiği %94'ü geçirir (kapı yok sayılır), 80 eşiği
> %94'ü durdurur (hat tıkanır). Aradaki her değer rastgele kesim yapar.
>
> **Kök neden — puan alaka ölçmüyor, üstveri eksiksizliği ölçüyor.**
> `scoring.Relevance` sinyalleri: başlık uzunluğu (+20), spot uzunluğu (+25),
> etiket sayısı (+15), kapak görseli var mı (+10), kaynak bağlantısı var mı
> (+10), başlıkta tekrar eden sözcük (+5/adet). Hiçbiri konuyla, güncellikle
> ya da trend uyumuyla ilgili değil. Bot hattı her taslağı aynı biçimde
> ürettiği için bu sinyaller neredeyse sabit; puan da öyle.
>
> **Sonuç:** Dağılımı genişletmeden eşiği açmanın anlamı yok. Puanın
> ayırt edici olması için konuya bağlı bir sinyal (trend terimi eşleşmesi,
> kaynak güveni, yayın yoğunluğu) gerekir. Bu, §7.2'deki **U3** ile aynı
> kökten: editöre 72 sayısı değil gerekçe gösterilmeli.
>
> **Ayrıca arayüz etiketi yanıltıcı.** `icerikform.templ:1146` alanı
> "AI Alaka Puanı (yayın kapısı)", düğme "AI puanını hesapla" diyor. Kodun
> kendi notu (`scoring.Note`) ise dürüst: *"yerel sezgisel (başlık/spot/
> etiket/görsel/kaynak sinyalleri)"* — ama bu not panelde hiç gösterilmiyor.
> Ortada AI yok; yerel bir sezgisel var. U3 bunu da kapsamalı.


---

## 5c · Yayın önceliği: tek sayıdan dört eksene (29 Ekim 2026)

§5'teki ölçüm şunu göstermişti: tek bir `ai_score` hem "yayına hazır mı"
hem "yayınlamaya değer mi" sorusunu yanıtlamaya çalışıp ikisini de
yanıtlayamıyordu. Yapı dört eksene ayrıldı.

| Eksen | Soru | Kaynak | Saklanıyor mu |
|---|---|---|---|
| **Tamlık** | Yayına hazır mı? | başlık/spot/etiket/kapak/kaynak | `contents.ai_score` |
| **Trend** | Şu an aranıyor mu? | `trend_terms` eşleşmesi | `contents.score_trend` |
| **Güven** | Kaynağa güveniliyor mu? | `feeds.trust` | `contents.score_guven` |
| **Tazelik** | Hâlâ haber mi? | `created_at` | **hayır** — sorgu anında |

**Tazelik neden saklınmıyor.** Zamanın fonksiyonu. Sütuna yazılsaydı her saat
bayatlar, 34 bin satırı sürekli güncelleyen bir iş gerekir ve iki koşum
arasında sıralama yanlış kalırdı. `created_at` zaten tabloda; hesap sorgu
anında yapılıyor (`scoring.TazelikSQL`). Ağırlıklar Go sabitlerinden SQL'e
üretiliyor, elle yazılmıyor — aksi hâlde panelde görünen puanla listenin
sırası ayrışırdı. Integration testi (geçici tablo, gerçek `contents`'e
dokunmaz) 55 durumda iki tarafın aynı sonucu verdiğini sabitliyor.

**Ağırlıklar ölçülen yayılıma göre:** Trend 45, Tazelik 35, Güven 10,
Tamlık 10. İlk denemede Tamlık 15 + Güven 20 idi; ölçünce bu iki eksenin
korpusta neredeyse sabit olduğu, yani ağırlıklarının %35'inin sıralamaya hiç
katkı vermeyip yalnız puan aralığını daralttığı görüldü. Bir eksenin
katkısı = ağırlığı × gerçek yayılımı. Tamlık asıl işini yayın
**kapısında** yapıyor (`categories.min_score`), sıralamada değil.

**Tazelik eğrisi de ölçümle düzeltildi.** Önce 24 saat yarı ömürlü üstel
sönüm denendi; taslak yaşları 0–182 saate yayıldığı için bu, 72 saatten
sonrasını ~0'a indirip korpusun çoğunda ayırt etmeyi bırakıyordu — öncelik
yine tek bantta (%61) toplanıyordu. İki parçalı eğriye geçildi:
0–24 saat 100→70, 24–168 saat 70→0.

### Trend eşleştirmesi artık taslaklara da bakıyor

**Asıl boşluk buydu:** `trend_matches` yalnız **yayındaki** içeriği
kapsıyordu (185 kayıt, hiç taslak yok). Yani trend uyumu yayından *sonra*
ölçülüyor, yayın kararında kullanılmıyordu.

Düzeltme **ayrı bir geçişle** yapıldı, mevcut eşleştirme değiştirilmeden:
`trend_matches` kapsama analizinindir ("bu konuyu yayınladık mı?") ve bilerek
yalnız yayındakilere bakar; taslakları oraya eklemek `coverage_count`'u
şişirir ve editöre işlenmemiş konuyu "kapsandı" gösterirdi. İki soru ayrı,
iki depo ayrı: taslağın trend puanı `contents.score_trend`'e yazılıyor.

`trends.EnIyiEslesme` belge merkezlidir (`Coverage` terim merkezli) ve aynı
ilkelleri kullanır; tek fark terimleri bir kez ayrıştırmasıdır.
`TestHazirEslestirmeCoveredIleAyni` iki yolun birebir aynı sonucu verdiğini
sabitler.

### Üretim verisinde sonuç (34.884 taslak)

| | eski `ai_score` | yeni `öncelik` |
|---|---|---|
| en kalabalık bant | **%87,9** | **%26,2** |
| dolu bant sayısı | 3 | **10** |
| trend sinyali olan taslak | 0 | **12.905 (%37)** |

```
 0- 9 :    52 (%0,1)     50-59 : 2.597 (%7,4)
10-19 : 9.131 (%26,2)    60-69 : 2.743 (%7,9)
20-29 : 6.371 (%18,3)    70-79 : 1.376 (%3,9)
30-39 : 7.378 (%21,2)    80-89 :   427 (%1,2)
40-49 : 4.759 (%13,6)    90-99 :    50 (%0,1)
```

Artık hem sıralama hem `min_score` eşiği için kullanışlı aralık var.

### Kalanlar

- `scheduler.RunIcerikSkor` 15 dakikada bir güven ve trend eksenlerini
  tazeliyor; tazelik penceresi dışındaki (7 günden eski) taslaklar atlanıyor.
- Panelde sıralama seçeneği eklendi ("Yayın önceliği"), "AI skoru" etiketi
  "Tamlık" oldu.
- **Açık:** öncelik ifadesi hesaplanmış olduğu için indeksle sıralanamıyor;
  korpus büyüdükçe tam sıralama maliyeti ölçülmeli.
- **Açık:** eksenler panelde içerik formunda tek tek gösterilmiyor; editör
  şu an yalnız tamlığın gerekçesini görüyor (U3).

---

## 6. B5 — Kategori ataması

**Bulgu.** %77 Gündem. Kök neden: feed'in kendi kategorisi atanıyordu ve
agregatör akışlarında o kategori Gündem'di.

**Karar.** `internal/classify` üç kademeli: kaynak bölümü → editörün feed
ataması → anahtar sözcük → Gündem.

**Tasarım notları.**
- Türkçe normalizasyon `strings.ToLower` ile yapılamaz: "İ" → "i̇"
  (i + birleşik nokta) üretir, "İSTANBUL" ile "istanbul" eşleşmez.
- Geniş kategorilerde tek sinyal yetmez (broadRules): "Yeni spor salonu
  açıldı" cümlesi Spor'a düşmemeli.
- Editörün atadığı kategori heuristiği ezer: "GZT Spor" akışındaki haberde
  "dolar" geçse bile kategori spordur.

**Risk.** Anahtar sözcük tablosu Türkçe haber diline göre elle bakımlıdır;
yeni kategoriler eklendiğinde `classify` tablosu güncellenmelidir. Yanlış
pozitifler arşivde kalır, yayına çıkmaz (taslak durumu).

> ### ÖLÇÜM (29 Ekim 2026) — B5 işe yaradı, ama Gündem hâlâ dolu
>
> **İyi haber: %77 Gündem bulgusu kapandı.** Taslakların kategori dağılımı
> artık gerçek: Gündem **9.745 / 34.156 = %28,5**. Kalanı ekonomi 3.923,
> dünya 3.349, siyaset 2.247, spor 2.088, futbol 2.055, teknoloji 1.626,
> asayiş 1.556, afet-kaza 1.098, yaşam 1.086, eğitim-kültür 935 … (43 kategori).
>
> **Kötü haber: Gündem'deki 9.745 taslağı mevcut araç kurtaramaz.**
> `cmd/backfill -is=kategori` kuru koşumu: *9.744 tarandı, 0 taşındı.*
> Bu bir hata değil, **tasarım gereği**: backfill yalnız Gündem'dekilere bakar
> ve onları yutumda kullanılan sınıflandırıcının **daha azıyla** (feed bölümü
> ve editör ataması olmadan, yalnız başlık+spot) yeniden geçirir. Gündem'de
> olmalarının sebebi zaten hiçbir kuralın tutmamasıdır; aynı kural yine
> tutmaz. Backfill ancak `classify` tablosu genişletilirse iş görür.
>
> **Anahtar sözcük kuralları ne kadar taşıyor?** 43 kategoriden her birinden
> en fazla 400 taslak (toplam 8.640) alınıp sınıflandırıcıya **ipucusuz**
> (feed/bölüm verilmeden) verildi:
>
> - %76,9'unda mevcut kategoriyle aynı sonuç çıktı,
> - %14,7'si hiçbir kurala takılmayıp varsayılan Gündem'e düştü.
>
> Yani kurallar geneli taşıyor. Sorun belirli kategorilerde yoğunlaşıyor:
>
> | kategori | adet | varsayılana düşen | mevcutla aynı | en sık tahmin |
> |---|---|---|---|---|
> | mimari | 76 | %57,9 | %2,6 | gundem |
> | yaşam | 400 | %41,0 | %3,2 | gundem |
> | avrupa | 16 | %31,2 | %0,0 | gundem |
> | dünya | 400 | %31,0 | %42,2 | dunya |
> | amerika | 10 | %30,0 | %0,0 | dunya |
> | ekonomi | 400 | %29,0 | %3,2 | gundem |
> | teknoloji | 400 | %28,8 | %55,0 | teknoloji |
> | spor | 400 | %28,2 | %3,8 | futbol |
>
> Buna karşılık asayiş, otomotiv, sağlık, afet-kaza, siyaset, futbol, enerji,
> döviz, altın, tarım, faiz, basketbol, müzik, emlak, borsa, enflasyon %99–100
> isabetle çalışıyor.
>
> **İki ayrı sorun var, karıştırılmamalı:**
>
> 1. **Kural boşluğu** — `mimari`, `yaşam`, `avrupa`, `amerika` için kural
>    yok ya da çok dar. Bunlar Gündem'e düşüyor. Çözüm: `classify` tablosuna
>    kural eklemek.
> 2. **Taksonomi çakışması** — `ekonomi` %3,2, `spor` %3,8 "isabet" gösteriyor
>    ama bu yanlışlık değil: kurallar geniş ebeveyn yerine dar çocuğu seçiyor
>    (`ekonomi` → döviz/altın/borsa/faiz, `spor` → futbol/basketbol). Burada
>    karar gereken şey kuralın değil, **taksonominin kendisi**: ebeveyn
>    kategori mi kalsın, yoksa içerik hep en dar çocuğa mı gitsin?
>
> Ölçüm yöntemi: feed ipucu bilerek verilmedi; amaç "ipucu yoksa ne oluyor"
> sorusunu yalıtmaktı. Mevcut kategoriler feed bölümünden geldiği için bu
> karşılaştırma *hangisinin doğru olduğunu* söylemez — yalnız kuralların tek
> başına nereyi taşıyabildiğini gösterir.

---

## 7. Kalan işler

### 7.1 Küçük kalemler (B6 artığı)

| İş | Açıklama |
|---|---|
| `services_cache` yetim satırları | Eski anahtar şemasından kalan `puan|ana`, `fikstur|meta`, `fikstur-r|2026-2027-N` okunmuyor; temizlenmeli |
| `Demo` rozeti | `puan` ve `fikstur` sayfalarında `demo:false` iken rozet gösterilmemeli (namaz/eczane/borsa'da doğru davranış var) |
| RSS'siz 7 kaynak | DHA, İHA, Fanatik, Sporx, NTV Spor, Bigpara, Hürriyet Gündem: adresleri RSS değil. Doğru adres bulununca panelden düzeltilmeli; bot 6 saatte bir deniyor |

### 7.2 U1–U4 — içerik triyajı (tasarım)

Bunlar bot hattının ürettiği hacmi **editörün işine** çeviren arayüz işleridir.
Kod tarafı (B1–B5) tamamlandığı için artık veri bunları destekliyor.

| # | İş | Neden | Kabul |
|---|---|---|---|
| **U1** | Triyaj ekranı: taslakları `ai_score`, kategori ve kaynak güvenine göre tek listede, klavye ile "yayınla / arşivle / birleştir" | 29.634 taslak liste hâlinde gezilemez; editör öncelik göremiyor | Tek ekranda 50 taslak, her satırda skor + kategori + kaynak; eylemler tek tuş |
| **U2** | İçerik formu sadeleştirme: ilk açılışta yalnız başlık/spot/gövde/kategori; SEO, YMYL, sosyal alanları katlanır | Form 40+ alan; editör kayboluyor | İlk görünümde ≤ 8 alan; kaydetme davranışı değişmez; E2E yeşil |
| **U3** | AI etkileşim dili: "AI skor" yerine "Yayına uygunluk" + gerekçe satırı; kullanıcıya sayı değil gerekçe | Editör 72'nin ne anlama geldiğini bilmiyor | Her skorun yanında gerekçe (hangi sinyaller eksik) |
| **U4** | Trend → içerik köprüsü: Trend Merkezi'ndeki "Taslağa Al" sonrası taslak, trend terimine ve kapsama bilgisine bağlı açılsın | Bağlam kayboluyor, editör trendi tekrar aramak zorunda | Taslakta `trend_term_id`; panelde "bu trendden geldi" rozeti |

U1, `10-admin-mimari-ve-arayuz.md` A1–A3 ile aynı dosyalara dokunur
(admin şablonları, CSS token'ları); sıralamada **A3'ten sonra** ele alınmalı.

---

## 8. Bu dosyanın kapsamadıkları

- **E2E paketi çalıştırılmadı** (56 dosya); U1–U4 kabulü buna dayanır.
- **Görsel denetim yapılmadı**: triyaj ekranı ve form sadeleştirmesi
  tarayıcıda gezinilmedi.
- RSS'siz 7 kaynağın adresi araştırılmadı (yalnız yaygın kalıplar denendi).
- Botun **içerik kalitesi** (AI özetinin doğruluğu, başlık kalitesi) bu
  denetimin dışında; kalite kapıları `ILERLEME.md` Faz 115–122'de.
