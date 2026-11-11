#!/usr/bin/env bash
# k8s-sevk.sh — theblock'u Kubernetes'e SABİT İMAJ ETİKETİYLE sevk eder.
#
# Neden bu betik var: manifest'lerdeki imaj etiketi `${IMAJ_ETIKETI}` yer
# tutucusudur, `kubectl apply -f` onu çözmez. Eskiden `:latest` yazıyordu;
# o hâlde kümede hangi commit'in koştuğu BİLİNEMİYORDU:
#   · geri dönüş yapılamıyor (etiket hep aynı yeri gösteriyor),
#   · imaj taramasının anlamı kalmıyor (taranan imaj ≠ dağıtılan imaj).
#
# Kullanım:
#   bash deploy/scripts/k8s-sevk.sh                 # HEAD commit'inin etiketi
#   bash deploy/scripts/k8s-sevk.sh sha-abc1234     # belirli etiket
#   bash deploy/scripts/k8s-sevk.sh --uret-only     # yalnız YAML bas (tarama/inceleme)
#   bash deploy/scripts/k8s-sevk.sh --site-only sha-abc1234
#       # yalnız site imajını sevk et. Yalnız Go sitesi değiştiyse doğru yol:
#       # diğer üç imaj o etikette YOKTUR; 30-mikroservisler.yaml'ı uygulamak
#       # onları ImagePullBackOff'a düşürür (5 Kasım 2026'da yaşandı).
#
# Ortam:
#   KUBECONFIG   varsayılan: D:\talos\.kube\config karşılığı (WSL yolu)
#   NAMESPACE    varsayılan: theblock  (ASLA başka namespace'e dokunmaz)
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
K8S="$KOK/deploy/k8s"
NS="${NAMESPACE:-theblock}"

# kubectl: WSL'de kurulu değil; Docker Desktop'ınki (Windows .exe) kullanılır.
WINDOWS_KUBECTL=0
if command -v kubectl >/dev/null 2>&1; then
  KUBECTL=kubectl
elif [ -x "/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe" ]; then
  KUBECTL="/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe"
  WINDOWS_KUBECTL=1
else
  echo "HATA: kubectl bulunamadı." >&2
  exit 1
fi

: "${KUBECONFIG:=/mnt/d/talos/.kube/config}"
# TUZAK: kubectl.exe bir WINDOWS programıdır; `/mnt/d/...` biçimindeki
# WSL yolunu açamaz ve "couldn't get current server API group list" ile
# düşer (hata mesajı yolu hiç anmaz, bu yüzden yanıltıcı). Windows
# ikilisi kullanılıyorsa yol `D:\talos\...` biçimine çevrilir.
if [ "$WINDOWS_KUBECTL" = 1 ] && [ "${KUBECONFIG#/mnt/}" != "$KUBECONFIG" ]; then
  KUBECONFIG=$(wslpath -w "$KUBECONFIG")
fi
export KUBECONFIG
echo "==> kubectl: $KUBECTL"
echo "==> KUBECONFIG: $KUBECONFIG"

# ÖLÇÜM (5 Kasım 2026): WSL'den başlatılan kubectl.exe, `KUBECONFIG` ortam
# değişkenini GÖRMÜYOR; varsayılan `%USERPROFILE%\.kube\config`'e düşüp
# "couldn't get current server API group list" veriyor. Aynı ikili
# `--kubeconfig D:\talos\.kube\config` ile sorunsuz çalışıyor. Bu yüzden
# her çağrıda açıkça verilir.
kctl() {
  if [ "$WINDOWS_KUBECTL" = 1 ]; then
    "$KUBECTL" --kubeconfig "$KUBECONFIG" "$@"
  else
    "$KUBECTL" "$@"
  fi
}

# kubectl.exe WSL yollarını (`/tmp/...`) açamaz; apply dosya yolunu
# Windows biçimine çevir.
uygula() {
  if [ "$WINDOWS_KUBECTL" = 1 ]; then
    kctl apply -f "$(wslpath -w "$1")"
  else
    kctl apply -f "$1"
  fi
}

URET_ONLY=0
SITE_ONLY=0
ETIKET=""
for a in "$@"; do
  case "$a" in
    --uret-only) URET_ONLY=1 ;;
    --site-only) SITE_ONLY=1 ;;
    -*) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
    *) ETIKET="$a" ;;
  esac
done

if [ -z "$ETIKET" ]; then
  cd "$KOK"
  ETIKET="sha-$(git rev-parse --short HEAD)"
  KIRLI=$(git status --porcelain | wc -l)
  if [ "$KIRLI" -gt 0 ]; then
    echo "UYARI: ağaçta $KIRLI commit'lenmemiş dosya var."
    echo "       Etiket HEAD'i ($ETIKET) gösteriyor; o dosyalar imajda YOK."
  fi
fi

export IMAJ_ETIKETI="$ETIKET"
echo "==> imaj etiketi: $IMAJ_ETIKETI   namespace: $NS"

# Yer tutucuyu doldur. envsubst'a YALNIZ bu değişken verilir: manifest'teki
# k8s tarafı `$(POSTGRES_USER)` ifadeleri olduğu gibi kalsın.
URETIM=$(mktemp -d)
temizle() { rm -rf "$URETIM"; }
trap temizle EXIT

for f in "$K8S"/*.yaml; do
  envsubst '${IMAJ_ETIKETI}' < "$f" > "$URETIM/$(basename "$f")"
done

# Yer tutucu kaldıysa dur: eksik değişkenle küme bozulmasın.
#
# Yalnız ${IMAJ_ETIKETI} aranır. Manifest'lerin içindeki CronJob kabuk
# betiklerinde ${damga}, ${hedef}, ${SAKLAMA_GUN} gibi DEĞİŞKENLER var;
# bunlar envsubst'ın değil, pod içindeki `sh`'in değişkenleridir. Hepsini
# "doldurulmamış yer tutucu" saymak sevki tamamen kilitlerdi.
if grep -rn '\${IMAJ_ETIKETI}' "$URETIM"/*.yaml; then
  echo "HATA: doldurulmamış yer tutucu kaldı (yukarıda)." >&2
  exit 1
fi

if [ "$URET_ONLY" = 1 ]; then
  KAYIT="${URET_DIZIN:-$KOK/.k8s-uretim}"
  mkdir -p "$KAYIT"
  cp "$URETIM"/*.yaml "$KAYIT"/
  echo "==> basılan YAML: $KAYIT"
  exit 0
fi

echo "==> altyapı: namespace + PVC"
uygula "$URETIM/00-namespace.yaml" || exit 1

# SERVICE ACCOUNT'LAR ÖNCE: Deployment'lar bunlara `serviceAccountName` ile
# bağlanıyor; SA yoksa pod Pending'de kalır ("serviceaccount not found").
uygula "$URETIM/07-rbac.yaml"      || exit 1

# POSTGRES TLS: `theblock-pg-tls` Secret'ı YOKSA postgres de site de
# BAŞLAMAZ (fail closed — şifresiz bağlantıdan iyidir). Betik idempotent:
# sertifika geçerliyse dokunmaz, süresi yaklaşmışsa yeniler.
if command -v openssl >/dev/null 2>&1; then
  bash "$KOK/deploy/scripts/pg-tls-uret.sh" || {
    echo "HATA: Postgres TLS materyali üretilemedi." >&2
    exit 1
  }
else
  echo "UYARI: openssl yok — mevcut theblock-pg-tls Secret'ı kullanılacak." >&2
  kctl -n "$NS" get secret theblock-pg-tls >/dev/null 2>&1 || {
    echo "HATA: openssl yok ve theblock-pg-tls Secret'ı da yok." >&2
    exit 1
  }
fi

uygula "$URETIM/10-pvc.yaml"       || exit 1

# AĞ POLİTİKALARI: varsayılan reddet. Deployment'lardan ÖNCE uygulanır ki
# pod'lar ilk saniyeden itibaren segmentli doğsun.
uygula "$URETIM/06-networkpolicy.yaml" || exit 1

# KYVERNO: yalnız CRD kuruluysa. Kurulu değilse politika uygulamak
# "no matches for kind ClusterPolicy" ile sevki kırar — küme Kyverno'suz
# da çalışabilsin (politika kurulumu ayrı bir adım, bkz. dosya başı).
if kctl get crd clusterpolicies.kyverno.io >/dev/null 2>&1; then
  uygula "$URETIM/05-kyverno.yaml" || exit 1
  echo "==> Kyverno politikaları uygulandı"
else
  echo "UYARI: Kyverno kurulu değil — imza zorunluluğu kümede AKTİF DEĞİL." >&2
  echo "       Kurulum: deploy/k8s/05-kyverno.yaml başındaki helm komutu." >&2
fi

if [ "$SITE_ONLY" = 1 ]; then
  echo "==> yalnız site"
  uygula "$URETIM/40-site.yaml" || exit 1
  kctl -n "$NS" rollout status deploy/theblock-site --timeout=300s || {
    echo "HATA: theblock-site açılmadı. Geri dönüş:  $KUBECTL -n $NS rollout undo deploy/theblock-site" >&2
    exit 1
  }
else
  echo "==> veri katmanı (postgres, redis)"
  uygula "$URETIM/20-postgres.yaml" || exit 1
  uygula "$URETIM/21-redis.yaml"    || exit 1
  kctl -n "$NS" rollout status deploy/theblock-postgres --timeout=240s || exit 1
  kctl -n "$NS" rollout status deploy/theblock-redis    --timeout=180s || exit 1

  echo "==> uygulama (mikroservisler, site, ingress)"
  uygula "$URETIM/30-mikroservisler.yaml" || exit 1
  uygula "$URETIM/40-site.yaml"           || exit 1
  uygula "$URETIM/50-ingress.yaml"        || exit 1

  for d in theblock-image-svc theblock-search-svc theblock-collab theblock-site; do
    kctl -n "$NS" rollout status "deploy/$d" --timeout=300s || {
      echo "HATA: $d açılmadı. Geri dönüş:  $KUBECTL -n $NS rollout undo deploy/$d" >&2
      exit 1
    }
  done
fi

echo
echo "==> sevk edilen imajlar"
kctl -n "$NS" get deploy -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'

echo
echo "==> sağlık"
bash "$KOK/deploy/smoke.sh" https://theblock.com.tr || {
  echo "UYARI: smoke geçmedi. Geri dönüş için:" >&2
  echo "  $KUBECTL -n $NS rollout undo deploy/theblock-site" >&2
  exit 1
}
echo "sevk tamam: $IMAJ_ETIKETI"
