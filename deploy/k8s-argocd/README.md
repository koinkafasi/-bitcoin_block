# GitOps: Argo CD + Kyverno

Bu dizin kümedeki **sevk ve admission** katmanını tanımlar. Uygulama
manifest'leri `deploy/k8s-bitcosar/` içinde; Argo CD orayı okur.

## Akış

```
git push main
   │
   ├─ imaj.yml ──► imaj derle → Trivy tara → SBOM → cosign imzala
   │                 → GHCR'a bas (sha-<7hane>)
   │               ──► cosign verify (4 imaj)
   │               ──► etiket-yaz: kustomization.yaml'daki newTag'i
   │                   güncelle + commit [skip ci]
   │
   └─ Argo CD o commit'i görür ──► kümeyi eşitler
                                     │
                                     └─ Kyverno admission:
                                        imza + :latest + root denetimi
```

Sıra **bas → doğrula → yaz**. Doğrulanmamış imajın etiketi depoya
girerse Argo CD onu canlıya sürer ve Kyverno pod'u reddeder; site
sessizce sevk edilemez hâle gelir.

## Kurulum (tekrarlanabilir)

Sürümler bilerek sabit. `--server-side` zorunlu: düz `kubectl apply`
büyük CRD'lerde 262144 baytlık annotation sınırına takılıyor.

```bash
# Argo CD
kubectl create namespace argocd
kubectl -n argocd apply --server-side --force-conflicts \
  -f https://raw.githubusercontent.com/argoproj/argo-cd/v3.6.1/manifests/install.yaml

# Kyverno
kubectl apply --server-side --force-conflicts \
  -f https://github.com/kyverno/kyverno/releases/download/v1.20.0/install.yaml

# Uygulama tanımı (bootstrap — Argo CD bunu kendi yönetmiyor)
kubectl apply -f deploy/k8s-argocd/bitcosar-app.yaml
```

## Arayüze erişim

Argo CD sunucusu **ClusterIP**, yani dışarı kapalı. Bilinçli: yönetim
arayüzünü internete açmak ayrı bir karar.

```bash
bash ~/bitcosar-betikler/argocd_eris.sh      # port-forward + parola
# sonra: https://localhost:8080  (kullanıcı: admin)
```

Parola `argocd/argocd-initial-admin-secret` içinde. İlk girişten sonra
değiştirip o secret'ı silin.

## Senkron davranışı

`prune: true` + `selfHeal: true`.

- **prune** — depodan silinen kaynak kümeden de silinir. Argo CD yalnız
  KENDİ izlediği kaynakları siler; namespace'teki secret'lar
  (`bitcosar-env`, `bitcosar-pg-tls`, `ghcr`) depoda olmadığı ve izleme
  etiketi taşımadığı için dokunulmaz.
- **selfHeal** — elle yapılan `kubectl edit` geri alınır.

`ignoreDifferences` iki kayıt taşıyor; ikisi de gürültü bastırmak için:

- **Deployment imaj alanı** — Kyverno `mutateDigest` ile etiketi
  digest'e çeviriyor, yani kümedeki değer depodakinden FARKLI yazılır
  ve bu İSTENEN davranış (çalışan imaj = imzalanan baytlar). Yoksa
  uygulama sürekli `OutOfSync` görünür ve gerçek sapma fark edilmez.
- **spec/replicas** — ölçekleme kararı işletmede verilebiliyor.

## Acil sevk (Argo CD'yi beklemeden)

```bash
bash deploy/scripts/bitcosar-sevk.sh sha-abc1234
```

Bu betik **depodaki `kustomization.yaml`'a dokunmaz** — yani acil sevk
Argo CD'nin hedefini sessizce kaydırmaz. `selfHeal` açık olduğu için
Argo CD kümeyi bir sonraki turda depodaki etikete geri çeker.
**Kalıcı olması gereken bir geri dönüşte etiketi depoya da yazın.**

## Kyverno politikaları

`deploy/k8s-bitcosar/05-kyverno.yaml` — üçü de `bitcosar-site`'a bağlı:

| Politika | Ne yapar |
|---|---|
| `bitcosar-imza-zorunlu` | cosign keyless imza + digest'e sabitleme |
| `bitcosar-latest-yasak` | `:latest` etiketini reddeder |
| `bitcosar-pod-sertlestirme` | root ve ayrıcalık yükseltme yasak |

Politikalar **CEL tabanlı yeni tiplerde** (`policies.kyverno.io/v1`).
Kyverno v1.20 eski `kyverno.io/v1 ClusterPolicy` tipinin
çalıştırılmasını kaldırdı ve oluşturma isteklerini reddediyor.

### İmza doğrulaması kayıt defteri kimliği ister

Paketler private; Kyverno imaj manifest'ini okuyamazsa pod'u REDDEDER:

```
failed to resolve digest for ghcr.io/koinkafasi/theblock-site:...
UNAUTHORIZED: authentication required
```

Bu yüzden `spec.credentials.secrets: [ghcr]` ve aynı dosyada dar
kapsamlı bir Role var (yalnız `ghcr` secret'i, yalnız `get`). Secret
**politikanın** namespace'inde aranır, Kyverno'nun kendi namespace'inde
değil.

### Politikaları test et

```bash
bash ~/bitcosar-betikler/kyverno_test.sh sha-abc1234
```

`--dry-run=server` admission webhook'undan geçer ama pod oluşturmaz.
Taban pod PSS ve ResourceQuota ile **tam uyumlu**; her sınav yalnız tek
bir şeyi bozar ve reddedenin kim olduğunu yazdırır. Bu önemli: eksik
bir test pod'u Kyverno'dan önce PodSecurity tarafından reddedilir ve
"politika çalışıyor" yanılgısı üretir.
