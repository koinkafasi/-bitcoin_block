# 14 — İçerik listesinde dil tekilleştirme ve dil etiketleri

**Durum:** ✅ uygulandı (Faz 192) — bu belge tasarım kaydı olarak kalıyor
**Tarih:** 1 Kasım 2026
**İlgili:** Faz 190 (Dil / Çeviriler paneli), Faz 189 (çok dilli ön yüz)

## Kullanıcının tarifi (kelimesi kelimesine)

> "bu çevirisi yapılan içeriğin içerikler tablosunda listelemede çevirisi
> yapılan içerik ayrıca bir listede tablo olarak değil de Aynı temel dili olan
> içeriğin listedeki Tip kolon öncesinde Dil diye bir alan yapıp kısa
> isimleriyle etiket ufak bir sembol olarak listede görünsün o içeriğin diğer
> dili düzenlemek istersem o etikete girip direkt o dil versiyonu düzenleme
> yapabileyim. Aynı makale için tablo alanında bu kadar tekrar eden farklı
> dillerde listeleme olmasın çok karışık oluyor. Var olan o makale içeriğini
> açtığımda diğer dil seçenekleri seçerek o makalenin diğer dil seçeneğini
> açıp düzenleme yapabilmeliyim."

Üç istek:

1. **Tek satır = tek makale.** Aynı makalenin TR/EN/DE sürümleri listede
   ayrı satır olarak tekrarlanmasın.
2. **Dil kolonu (Tip'ten önce).** Satırda o makalenin hangi dillerde
   olduğunu gösteren küçük etiketler (`TR` `EN` `DE`). Etikete tıklayınca
   o dilin sürümü düzenlemede açılır.
3. **Editörde dil geçişi.** Makaleyi açtığımda diğer dillere oradan
   geçebilmeliyim (bugün "Dil / Çeviriler" paneli var ama satırlar
   listenin üstünde gizli bir `<details>` içinde; editörde sekme gibi
   değil).

## Bugünkü durum (ölçüm)

Üretim veritabanı, 1 Kasım 2026 12:55:

```
toplam satır                     13.350
  tr                             12.857
  en                                493
çeviri satırı (translation_of)      493
distinct kök                     12.857
etkin dil sayısı                      2

durum      satır    kök
taslak     12.294   12.293
yayında       984      492      ← 492 makale, 984 satır (her biri 2 kez)
arşiv          72       72

"Yayında" sekmesinin İLK 40 SATIRI:
  satır 40 · çeviri 40 · en 40     ← %100'ü İngilizce çeviri
```

**Bu kozmetik bir karışıklık değil.** Sıralama
`published_at DESC NULLS LAST, id DESC` olduğu için en yeni 40 satırın
hepsi EN çeviri; TR asılları sonraki sayfalara düşüyor. Editör "Yayında"
sekmesini açtığında **Türkçe içerikleri göremiyor**, yalnız çevirilerini
görüyor. Kullanıcının "çok karışık oluyor" dediğinin ölçülebilir karşılığı
bu.

Kök bazlı gruplamayla aynı sekme **492 satır** olur (bugün 984) — tam yarısı;
her yeni dil eklendiğinde liste o kadar kat büyümeyecek.

Kod tarafı: `repo.ListContents` (contents.go:373) dil filtresi uygulamıyor;
sıralama `contentOrderBy` ve `LIMIT/OFFSET` doğrudan tüm satırlara uygulanıyor.

Yani çeviri üretildikçe liste **şişiyor** — kullanıcının "çok karışık"
dediği şey bu.

## Karar özeti

| Konu | Karar | Gerekçe |
|---|---|---|
| Listeleme birimi | **çeviri kökü** (`COALESCE(translation_of, id)`) | aynı makalenin tüm dilleri tek satır |
| Satırın gösterdiği sürüm | **kök dil** (yoksa en yeni sürüm) | başlık/spot tutarlı; editör varsayılanı da bu |
| Dil etiketi | küçük pill: `TR` `EN` `DE` + eksik diller için `+` | tek bakışta hangi diller var |
| Etikete tıklama | o dilin editörü (yoksa "çeviri yok" durumu → AI ile çevir) | Faz 190 paneliyle aynı sözleşme |
| Sayfalama | kök sayısı üzerinden | 24 dil artık sayfa boyutunu yemiyor |
| Filtre | `dil` parametresi **korunur** ama anlamı değişir: "bu dili olan makaleler" | mevcut `?dil=` kırılmasın |
| Tip kolonu | yerinde kalır, Dil ondan **önce** gelir | kullanıcının istediği sıra |

## Teknik tasarım

### 1) Sorgu: kök bazlı toplama

`ListContents`'a `GrupDil bool` (ya da `KokBazli bool`) alanı eklenir.
Açıkken:

```sql
WITH kok AS (
  SELECT COALESCE(c.translation_of, c.id) AS kok_id, c.*
  FROM contents c
  WHERE <filtreler>          -- durum, tip, kategori, q, tarih...
),
sec AS (
  SELECT DISTINCT ON (kok_id) kok_id, id
  FROM kok
  ORDER BY kok_id,
           (locale = $kokDil) DESC,      -- önce kök dil
           updated_at DESC
)
SELECT <contentCols>, count(*) OVER(), sec.kok_id
FROM sec JOIN contents c ON c.id = sec.id
LEFT JOIN ... (mevcut join'ler)
ORDER BY <contentOrderBy>
LIMIT $n OFFSET $m;
```

- `DISTINCT ON` sıralaması deterministik olmalı: `kok_id`, sonra kök dil,
  sonra `updated_at DESC`, sonra `id DESC`.
- Durum filtresi **kök satırına** uygulanır (bugünkü davranış), böylece
  "Taslak" sekmesi çevirileri değil makaleleri sayar.
- `q` araması **tüm dillere** bakar: `EXISTS (SELECT 1 FROM contents t
  WHERE root(t)=root(c) AND (t.title ILIKE ...))`. Aksi hâlde İngilizce
  başlıkla arayan editör hiçbir şey bulamaz.

### 2) Dil etiketleri: tek ek sorgu (N+1 yok)

Liste 40 satır olduğu için satır başına sorgu **yasak**. Tek sorgu:

```sql
SELECT COALESCE(translation_of, id) AS kok_id, locale, id, status
FROM contents
WHERE COALESCE(translation_of, id) = ANY($1)   -- sayfadaki kökler
ORDER BY locale;
```

Sonuç `map[int64][]DilSurum` olarak view'a taşınır. `ix_contents_translation`
kısmi indeksi bu sorguyu karşılar; 40 kök için ölçülecek hedef **< 5 ms**.

### 3) Veri modeli (Go)

```go
// repo
type DilSurum struct {
    Locale string
    ID     int64
    Status string
}
// ListFilter'a: KokBazli bool; ListContents dönüşüne
//   diller map[int64][]DilSurum  (ya da ayrı fonksiyon: DillerOf)
```

`templkit.IcerikRow`'a:

```go
Diller []IcerikDil   // {Kod, ID, Durum, Var, Aktif}
```

### 4) Şablon

Başlık satırı:

```html
<th>Dil</th><th>Tip</th>...
```

Hücre:

```html
<td class="dil-hucre">
  <a class="dil-pill is-var" href="/admin/icerikler/58141" title="İngilizce · taslak">EN</a>
  <a class="dil-pill is-yok" href="#" hx-post="/admin/icerikler/554/cevir/de"
     hx-vals='{"ai":"1"}' title="Almanca çeviri yok — AI ile çevir">DE</a>
</td>
```

- Var olan dil → **düz bağlantı** (editörü açar). HTMX gerekmez.
- Olmayan dil → Faz 190'daki AI kuyruğu (aynı uç, aynı bekleme paneli).
- 24 dilin hepsini basmak satırı şişirir: **en çok 4 etiket** + `+N`
  (açılır liste). Kural: kök dil her zaman ilk, sonra dolu diller, sonra
  boşlar.

CSS: mevcut `.ceviri-rozet` yeniden kullanılmaz (o panel içi, farklı
yoğunluk). Yeni `.dil-pill`, 18px yükseklik, 10px yazı, `--surface` zemin.

### 5) Editörde dil geçişi (3. istek)

Bugün: "Dil / Çeviriler" kapalı bir `<details>`.

Değişiklik: **sayfa başlığının yanına** aynı pill'ler taşınır
(`.icerik-dil-seridi`), `<details>` paneli kalır (AI/boş taslak eylemleri
orada). Böylece:

- makaleyi açınca üstte `TR EN DE` görünür,
- tıklayınca o dilin editörü açılır,
- panel yalnız "üret" işleri için kullanılır.

Bu, Faz 190'ın `langRows` verisini yeniden kullanır — yeni sorgu yok.

## Uygulama sırası (her adım ölçülebilir)

1. **Sorgu katmanı** (`repo.KokBazli` + `DillerOf`), testler: aynı kökün 3
   dili tek satır; sayfalama toplam sayısı kök sayısı; `q` İngilizce
   başlıkla da bulur.
2. **Dil kolonu** (templ + CSS), testler: `TH sırası`, `dil-pill` sınıfı,
   eksik dilde AI ucu.
3. **Editör dil şeridi**, test: `langRows` aynı veriyi besliyor.
4. **Ölçüm**: 11.372 içerik / 24 dil varsayımıyla liste sayısı ve sorgu
   süresi; `EXPLAIN ANALYZE` ile `ix_contents_translation` kullanımı.

## Riskler ve kaçınma

| Risk | Kaçınma |
|---|---|
| `DISTINCT ON` + `count(*) OVER()` birlikte yanlış toplam verir | toplam **ayrı** sayılır (`SELECT count(DISTINCT root)`), sayfalama ondan |
| Kök satırı silinmiş çeviriler listede "yetim" kalır | `WHERE translation_of IS NULL OR EXISTS(kök)` |
| Sıralama kök dışı alana göre bozulur | `contentOrderBy` kök satırın alanlarıyla çalışır (join'li) |
| Aynı makale iki kez görünür (translation_of zinciri) | zincir Faz 190'da köke sabitlendi; `translation_of` her zaman kök |
| Toplu işlemler (yayınla/sil) yanlış sürümü vurur | toplu işlem **kök** kimliğiyle çalışır; çeviriler bilinçli olarak etkilenmez — panelde uyarı metni |

## Açık sorular (kullanıcıya)

1. Satırda **hangi sürümün** başlığı/durumu görünsün? Önerim: kök dil (TR);
   çeviri yayındaysa küçük `EN` etiketi yeşil kalır.
2. Boş dillere **AI düğmesi** konsun mu, yoksa etiket sadece "yok" mu
   göstersin? Önerim: etiket tıklanınca Faz 190 panelindeki akış açılsın
   (istek kuyruğa girer) — ayrı düğme satırı şişirir.
3. 24 dilde `+N` açılır listesi mi, yoksa yalnız "dolu diller" mi?
   Önerim: dolu diller + `+` (tıklayınca panel).

## Uygulama sonrası ölçüm (1 Kasım 2026)

```
                         ÖNCE        SONRA
"Yayında" ilk 40 satır   tr 1 · en 39   tr 40 · en 0
yayında satır sayısı     984            492   (492 makale)
sayfa başına 40 satır    karışık        tek makale = tek satır
arama (q)                yalnız temsilci dil   tüm diller (ölçüm: "Digital" → 11 kök)
```

Tarayıcıda doğrulama (izole):

```
başlık sırası            İçerik | Dil | Tip | Durum | …   (Dil, Tip'ten önce)
40 satır · tekrar eden başlık YOK
1. satır etiketleri      TR(aktif) EN(dolu) DE(kesikli) FR(kesikli)
dolu etikete tıklama     /admin/icerikler/53600 açıldı
JS hatası                yok
```

## Bilinçli sınırlar

- Satırda en çok **4** etiket: aktif dil + dolu çeviriler + eksiklerden 2.
  24 dilin tamamını basmak satırı okunamaz yapıyordu.
- Eksik dil etiketi (`DE` kesikli) tıklanınca **AI kuyruğu** çalışır
  (Faz 190). Ayrı "AI ile çevir" düğmesi satıra konmadı.
- Temsilci satır **kök dilde** (tr). Yayında olmayan çeviri varsa etiketi
  yeşil değil, ipucu metninde durumu yazar.
- Silinmiş köke işaret eden çeviri kendi başına grup olur (yetim satır);
  ayrı temizlik işi.
