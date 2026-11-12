# bitcosar.com geçiş planı

**Durum (12 Kasım 2026):** altyapı hazırlandı, **uygulanmadı**.
Geçiş canlı bir siteyi (WordPress) değiştirir; tetikleme bilinçli bir karar.

---

## Kümedeki mevcut tablo

Ölçüldü (`kubectl get ingress -A`):

| Alan adı | Şu an nereye gidiyor | Namespace |
|---|---|---|
| **bitcosar.com**, www | **WordPress** (canlı) | `bitcosar` |
| demo / ghost.bitcosar.com | Ghost CMS | `koinkafasi` |
| omn.bitcosar.com | OmniRoute | `omniroute` |
| sb / traders / rust.bitcosar.com | Supabase, traders, RustDesk | çeşitli |
| theblock.com.tr | Yeni Go sitesi | `theblock` |

DNS **Cloudflare arkasında** (104.21.13.173 / 172.67.200.222).
Küme giriş IP'si: `173.249.30.38`.

## İki veritabanı — karıştırılmamalı

| | Yerel (geliştirme) | Küme `theblock` |
|---|---|---|
| İçerik | 5.568 / 3.814 yayında | 58.579 / 986 yayında |
| Tür | **kripto** (TR + EN) | **genel haber** |
| Örnek | "Bitcoin ETF onayı…" | "Ardahan'da kış lastiği…" |
| Migration | **125** | 109 |

Kümedeki `theblock` veritabanı **theblock.com.tr'ye ait**. bitcosar.com'un
içeriği yereldedir. Bu yüzden geçiş ayrı bir namespace'e yapılır
(`bitcosar-site`) ve `theblock`'a dokunulmaz.

---

## Hazırlanan araçlar

| Dosya | Ne yapar |
|---|---|
| `scripts/bitcosar-manifest-uret.sh` | `k8s/` → `k8s-bitcosar/` (namespace + alan adı dönüşümü, doğrulamalı) |
| `scripts/bitcosar-veri-tasi.sh` | yerel veritabanı → küme (yedek al → yükle) |
| `k8s-bitcosar/` | üretilmiş manifestler (elle düzenlenmez) |

Manifestler aynı imajı kullanır (`ghcr.io/koinkafasi/theblock-site`):
iki site aynı Go binary'sini çalıştırır, fark veritabanındadır.

---

## Geçiş adımları

### 1. İmaj hazırla

```bash
git -C ~/bitcosar push              # GitHub Actions imajı üretir
# imaj etiketi: sha-<commit>
```

### 2. Manifestleri üret ve incele

```bash
bash deploy/scripts/bitcosar-manifest-uret.sh
less deploy/k8s-bitcosar/50-ingress.yaml
```

### 3. Sevk et (ingress HARİÇ ilk turda)

```bash
NAMESPACE=bitcosar-site bash deploy/scripts/k8s-sevk.sh sha-XXXXXXX
```

Bu adım `bitcosar-site` namespace'ini kurar; **bitcosar.com hâlâ
WordPress'tedir** çünkü yeni ingress henüz trafiği almıyor
(aynı host'u iki ingress isterse nginx eskisini korur — sıra
`creationTimestamp`'e göre).

### 4. Secret'ı ayarla

`theblock-env` içindeki değerler kopyalanır, **ikisi değişir**:

```bash
APP_BASE_URL=https://bitcosar.com
DATABASE_URL=postgres://...@bitcosar-postgres:5432/...
```

`OMNIROUTE_BASE_URL` **küme içi adres olarak kalmalı**:
`http://omniroute.omniroute.svc.cluster.local:20128/v1`
(dışarıdan `omn.bitcosar.com` DNS + TLS katmanına bağımlı olur).

### 5. Veriyi taşı

```bash
bash deploy/scripts/bitcosar-veri-tasi.sh --yedek-al
bash deploy/scripts/bitcosar-veri-tasi.sh --yukle ~/bitcosar-gecis/bitcosar-YYYYMMDD-HHMM.dump
```

Betik `theblock` namespace'ine yüklemeyi **reddeder**.

### 6. Küme içinden doğrula (trafik hâlâ WordPress'te)

```bash
kubectl -n bitcosar-site port-forward svc/bitcosar-site 8099:8070
curl -s -H "Host: bitcosar.com" http://127.0.0.1:8099/ | head -40
```

Kontrol: ana sayfa geliyor mu, `<link rel="canonical">` `bitcosar.com`
mu, fiyat şeridi dolu mu, `/admin` açılıyor mu.

### 7. Trafiği devral

```bash
# WordPress ingress'lerini kaldır (SİLME — yedekle)
kubectl -n bitcosar get ingress wordpress wp-engelli wp-giris -o yaml > ~/wp-ingress-yedek.yaml
kubectl -n bitcosar delete ingress wordpress wp-engelli wp-giris

# yeni ingress devreye girer
kubectl -n bitcosar-site apply -f deploy/k8s-bitcosar/50-ingress.yaml
```

### 8. Cloudflare

- SSL/TLS modu: **Full** (küme içinde TLS yoksa Flexible, ama Full tercih edilir)
- Proxy (turuncu bulut) açık kalsın
- Geçişten sonra cache temizle: *Caching → Purge Everything*

---

## Geri dönüş

Her adım geri alınabilir:

```bash
# trafiği WordPress'e geri ver
kubectl -n bitcosar-site delete ingress bitcosar-site
kubectl apply -f ~/wp-ingress-yedek.yaml

# dağıtımı geri al
kubectl -n bitcosar-site rollout undo deploy/bitcosar-site
```

WordPress dağıtımına hiç dokunulmaz — yalnız ingress'i kaldırılır,
pod'ları çalışmaya devam eder.

---

## Geçiş sonrası yapılacaklar

- [ ] `sites` tablosundaki `domain` değeri `bitcosar.com` (zaten öyle)
- [ ] Sitemap ve robots.txt yeni alan adını veriyor mu
- [ ] Google Search Console'a yeni mülk + sitemap
- [ ] WordPress URL'lerinden yeni yapıya 301 yönlendirme (eski
      `/%postname%/` → yeni `/kategori/YYYY/MM/DD/slug`)
- [ ] `theblock.com.tr` yaşamaya devam edecek mi — edecekse iki site
      ayrı veritabanlarıyla paralel çalışır, etmeyecekse `theblock`
      namespace'i kapatılabilir

---

## Açık soru

**theblock.com.tr'nin geleceği belirlenmedi.** İki olasılık:

1. **Paralel yaşar** — hazırlanan yapı tam bunu destekler (ayrı
   namespace, ayrı veritabanı, aynı imaj).
2. **Kapanır** — o zaman `theblock` namespace'i silinebilir ve
   kaynak (CPU/bellek/disk) serbest kalır.

Karar verilene kadar ikisi de ayakta kalabilir; düğümde yer var
(ölçüm: 6 CPU / 11.3Gi, kullanım %31).
