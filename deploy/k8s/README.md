# theblock — Kubernetes kurulum ve taşıma runbook'u

Hedef küme: `bitcosar` — tek düğüm (`talos-t3g-r50`, 173.249.30.38),
Talos v1.13.8, Kubernetes v1.36.2, ingress-nginx, StorageClass `local-path`.

> **Dikkat:** Bu kümede başka üretim yükleri çalışıyor (bitcosar/wordpress,
> supabase, ibrahimcosar, omniroute). theblock kendi namespace'inde durur ve
> `ResourceQuota` ile sınırlıdır; hiçbirine dokunulmaz.

Kubeconfig: `D:\talos\.kube\config` (context `admin@bitcosar`).

> **Manifest'ler artık `kubectl apply -f` ile DOĞRUDAN uygulanmaz.**
> İmaj etiketi `${IMAJ_ETIKETI}` yer tutucusudur; `deploy/scripts/k8s-sevk.sh`
> onu commit SHA'sı ile doldurur. `:latest` 5 Kasım 2026'da kaldırıldı:
> o hâlde kümede hangi commit'in koştuğu bilinemiyordu, geri dönüş
> yapılamıyordu ve taranan imaj ile dağıtılan imaj ayrışabiliyordu.
> Ayrıntı: `docs/guvenlik-hatti.md`.

```powershell
$env:KUBECONFIG = "D:\talos\.kube\config"
```

---

## 0. Ölçülen başlangıç durumu (2026-11-04)

| | |
|---|---|
| Düğüm kapasitesi | 6 CPU · 11.3Gi allocatable · 415Gi disk |
| **Gerçek** bellek kullanımı | 3.5Gi (%31) — boşta ~7.8Gi |
| theblock'un isteyeceği | ~1.2Gi istek · ~3.4Gi limit tavanı |
| Taşınacak veri | Postgres 409 MB · storage 276 MB · storage-private 23 MB |

Not: `kubectl describe node` "memory limits %91" der. Bu **limit** toplamıdır,
kullanım değil; tek başına `omniroute` 6Gi limit isteyip 441 MB kullanıyor.
Planlamayı belirleyen **istek** toplamı yalnız %23.

---

## 1. İmajları derle ve GHCR'ye gönder

Depo private olduğu için imajlar da private paket olur; küme `ghcr` adlı
pull secret ile çeker (adım 3).

**Normal yol: CI yapar.** `main`'e push'ta `.github/workflows/imaj.yml`
4 imajı derler, **Trivy ile tarar** ve ancak kapı geçerse
`ghcr.io/koinkafasi/theblock-<servis>:sha-<kısa>` olarak gönderir.
Sıra önemli — push-sonra-tara yapılsa açıklı imaj registry'ye girerdi.

Elle yapmak gerekirse (kapıyı atlamayın):

```bash
cd ~/theblock
ETIKET="sha-$(git rev-parse --short HEAD)"

for s in site image-svc search-svc collab; do
  docker build -f deploy/docker/$s.Dockerfile \
    -t ghcr.io/koinkafasi/theblock-$s:$ETIKET .
done

# KAPI: düzeltilebilir CRITICAL/HIGH varsa gönderme.
for s in site image-svc search-svc collab; do
  trivy image --scanners vuln --severity CRITICAL,HIGH --ignore-unfixed \
    --exit-code 1 ghcr.io/koinkafasi/theblock-$s:$ETIKET || exit 1
done

echo "$GHCR_TOKEN" | docker login ghcr.io -u koinkafasi --password-stdin
for s in site image-svc search-svc collab; do
  docker push ghcr.io/koinkafasi/theblock-$s:$ETIKET
done
```

`GHCR_TOKEN`: GitHub'da `write:packages` yetkili bir PAT.
`gh auth token` ile üretilen oturum jetonu paket yazmaya yetmez.

**İmaj boyutları:** site 822 MB (ffmpeg dahil), collab 359 MB,
search-svc 135 MB, image-svc 128 MB.

---

## 2. Sırları Secret'a çevir

`deploy/.env` repoya **girmez**; Secret ondan üretilir.

```bash
kubectl -n theblock create secret generic theblock-env \
  --from-env-file=deploy/.env
```

Manifest'ler `DATABASE_URL`, `REDIS_URL`, `COLLAB_URL`, `SEARCH_SVC_URL` ve
`APP_ENV` değerlerini `env:` ile **ezer** — `.env` içindeki 127.0.0.1
değerleri küme içinde geçersizdir, ezme bilinçlidir.

---

## 3. GHCR pull secret

```bash
kubectl -n theblock create secret docker-registry ghcr \
  --docker-server=ghcr.io \
  --docker-username=koinkafasi \
  --docker-password="$GHCR_TOKEN"
```

---

## 4. Manifest'leri uygula

```bash
kubectl apply -f deploy/k8s/00-namespace.yaml
# Secret'lar (adım 2-3) namespace oluştuktan SONRA yaratılır.
kubectl apply -f deploy/k8s/10-pvc.yaml
kubectl apply -f deploy/k8s/20-postgres.yaml
kubectl apply -f deploy/k8s/21-redis.yaml
kubectl -n theblock rollout status deploy/theblock-postgres

kubectl apply -f deploy/k8s/30-mikroservisler.yaml
kubectl apply -f deploy/k8s/40-site.yaml   # initContainer migration'ları uygular
kubectl apply -f deploy/k8s/50-ingress.yaml
kubectl -n theblock rollout status deploy/theblock-site
```

> **Yukarıdaki `kubectl apply -f` çağrıları artık çalışmaz** — imaj etiketi
> yer tutucu. Hepsinin yerine tek komut:
>
> ```bash
> bash deploy/scripts/k8s-sevk.sh            # HEAD commit'inin etiketi
> bash deploy/scripts/k8s-sevk.sh sha-abc1234
> ```
>
> Betik yer tutucuyu doldurur, sırayla uygular (namespace → PVC → veri
> katmanı → uygulama → ingress), her rollout'u bekler ve sonunda
> `deploy/smoke.sh` koşar. Yalnız YAML'i görmek için: `make k8s-uret`.

---

## 5. Veriyi taşı

Sıra önemli: önce şema (adım 4'teki `migrate` initContainer), sonra veri.

### 5a. Postgres

```bash
# Kaynak: yereldeki docker konteyneri (port 5433)
docker exec theblock-postgres-1 pg_dump -U theblock -d theblock -Fc \
  > /tmp/theblock.dump          # ~409 MB

# Hedef: kümedeki pod
POD=$(kubectl -n theblock get pod -l app=theblock-postgres -o name | head -1)
kubectl -n theblock cp /tmp/theblock.dump ${POD#pod/}:/tmp/theblock.dump

# --clean --if-exists: migration'ın kurduğu boş şemanın üstüne yazar.
kubectl -n theblock exec "$POD" -- \
  pg_restore -U theblock -d theblock --clean --if-exists --no-owner \
  /tmp/theblock.dump
```

`pg_restore` birkaç "does not exist" uyarısı basar (temiz şemada silinecek
nesne yok); bunlar beklenir, hata değildir.

### 5b. Medya

```bash
SITE=$(kubectl -n theblock get pod -l app=theblock-site -o name | head -1)
kubectl -n theblock cp ~/theblock/storage         ${SITE#pod/}:/data/
kubectl -n theblock cp ~/theblock/storage-private ${SITE#pod/}:/data/
kubectl -n theblock exec "$SITE" -- ls /data/storage | head
```

### 5c. Arama indeksi

Yeniden üretilebilir; kopyalamaya gerek yok. İndeks boşsa search-svc
ilk taramada doldurur.

---

## 6. DNS'ten önce doğrula

Ingress'e `Host` başlığıyla doğrudan vurarak, DNS değiştirmeden test et:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  -H 'Host: theblock.com.tr' http://173.249.30.38/healthz
curl -s -o /dev/null -w '%{http_code} %{size_download}\n' \
  -H 'Host: theblock.com.tr' http://173.249.30.38/
```

Kontrol listesi:

- [ ] `/healthz` 200
- [ ] Ana sayfa 200 ve ~260 KB
- [ ] `/static/public.css` 200 (gömülü varlıklar)
- [ ] Bir medya görseli 200 (`/media/variant/<id>/480`)
- [ ] Yönetim girişi açılıyor
- [ ] `kubectl -n theblock logs deploy/theblock-site` temiz

---

## 7. DNS geçişi (Cloudflare)

1. Cloudflare → theblock.com.tr → DNS
2. `A  @  173.249.30.38` (Proxied, turuncu bulut)
3. `A  www  173.249.30.38` (Proxied)
4. SSL/TLS modunu ayarla — bkz. `50-ingress.yaml` başındaki not.

**Eski tüneli hemen kapatma.** `cloudflared-theblock.service` çalışır
kalsın; geri dönüş gerekirse DNS'i tünel CNAME'ine çevirmek yeterli olur.
En az 24 saat beklenmesi önerilir.

---

## 8. Geri dönüş

| Sorun | Çözüm |
|---|---|
| Site kümede bozuk | Cloudflare DNS'i eski tünel CNAME'ine çevir (dakikalar) |
| Şema göçü hatalı | `kubectl -n theblock exec <pg-pod> -- pg_restore ...` yeniden; ya da yerel yedekten |
| İmaj hatalı | `kubectl -n theblock rollout undo deploy/theblock-site` |
| Tamamen vazgeç | `kubectl delete ns theblock` — PVC'ler `Retain`, veri düğümde kalır |

---

## 9. Bu kurulumun bilinen sınırları

- **Tek kopya, tek düğüm.** HA yok; düğüm düşerse site düşer.
- **`local-path` genişletilemez.** PVC'ler baştan bol verildi (40/60/10/5 Gi).
- **Postgres ve medya yedeği otomatik.** `deploy/k8s/60-yedek.yaml`:
  günlük DB dump'ı 03:15, haftalık medya arşivi pazar 04:30; ikisi de ayrı
  `theblock-backup` PVC'sine yazılır. Doğrulama ve geri yükleme: §9.1.
- **`site` yatay ölçeklenemez:** arka plan işçileri süreç içinde; iki kopya
  aynı işi iki kez yapar. Ölçekleme için işçiler ayrılmalı.
- **collab tek kopya:** Yjs belge durumu bellekte.

---

## 9.1 Otomatik yedek (5 Kasım 2026)

| İş | Zaman (Europe/Istanbul) | Ne | Saklama |
|---|---|---|---|
| `theblock-yedek-db` | her gün **03:15** | `pg_dump -Fc` + `pg_restore --list` | 14 gün |
| `theblock-yedek-medya` | pazar **04:30** | `tar.gz` + `tar -tzf` | 56 gün |

Tasarım:

- **Ayrı PVC** (`theblock-backup`, 20 Gi). Medya PVC'sine yazılsaydı yedek
  ile canlı veri aynı diski paylaşırdı; disk dolunca ikisi birlikte giderdi.
- **Atomik yazım:** arşiv önce `<hedef>.tmp` dosyasına yazılır, okunabilirlik
  doğrulanır, sonra `mv` ile hedefe taşınır. Hata olursa `trap` yarım dosyayı
  siler — "boyutu var ama açılmıyor" yedeği kalmaz.
- **Kimlikler:** DB pod'u `70:70` (postgres imajı); medya pod'u **`10001:999`**
  (site konteyneriyle aynı). `local-path` bir hostPath hacmidir; kubelet
  `fsGroup`'u var olan dosyalara uygulamaz. Medya dosyaları `uid 10001`,
  mod `600` olduğu için yedek pod'u farklı kullanıcıyla koşarsa `tar`
  "Permission denied" ile düşer (5 Kasım'da gerçekten düştü).
- Her iki pod da `readOnlyRootFilesystem`, `drop: ALL`,
  `allowPrivilegeEscalation: false`, `runAsNonRoot`, `seccompProfile:
  RuntimeDefault` ile koşar; yazılabilir tek yol `/tmp` (emptyDir).

Doğrulama (elle tetikleme):

```bash
$env:KUBECONFIG = "D:\talos\.kube\config"
kubectl -n theblock create job --from=cronjob/theblock-yedek-db yedek-db-elle
kubectl -n theblock create job --from=cronjob/theblock-yedek-medya yedek-medya-elle
kubectl -n theblock logs job/yedek-db-elle
kubectl -n theblock logs job/yedek-medya-elle
```

Ölçülen (5 Kasım 2026): **DB 46,9 MB**, **medya 288,5 MB**; ikisi de
`pg_restore --list` / `tar -tzf` kapısından geçti. İşler `Complete 1/1`.

Geri yükleme: dump'lar `theblock-backup` PVC'sinde.

- **DB:** §5'teki `pg_restore --clean --if-exists` akışını kullan. Dosyayı
  yedek PVC'sini bağlayan geçici bir pod'dan postgres pod'una kopyala;
  geri yükleme sırasında siteyi `kubectl -n theblock scale deploy/theblock-site
  --replicas=0` ile durdur, bitince geri aç.
- **Medya:** siteyi aynı şekilde durdur, arşivi site konteyneriyle aynı
  kimlikle (`uid 10001`) `/data` altına aç, `chown -R 10001:999 /data/storage`
  uygula, sonra siteyi aç.
