#!/usr/bin/env bash
# bitcosar-sevk.sh — bitcosar-site namespace'ini kümeye sevk eder.
#
# `k8s-sevk.sh` theblock'a sabit (K8S dizini, rollout adları, smoke URL).
# Bu betik aynı güvenlik sırasını `deploy/k8s-bitcosar/` için uygular.
#
# TRAFİK ALMAZ: ingress varsayılan olarak UYGULANMAZ. Trafiği devralmak
# bilinçli ve ayrı bir adımdır (bkz. deploy/BITCOSAR-GECIS.md):
#   INGRESS=1 bash deploy/scripts/bitcosar-sevk.sh sha-XXXXXXX
#
# Kullanım:
#   bash deploy/scripts/bitcosar-sevk.sh sha-2976bef
#   INGRESS=1 bash deploy/scripts/bitcosar-sevk.sh sha-2976bef   # trafiği de al
#   bash deploy/scripts/bitcosar-sevk.sh --uret-only sha-XXXX    # yalnız YAML bas
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
K8S="${K8S_DIR:-$KOK/deploy/k8s-bitcosar}"
NS="${NAMESPACE:-bitcosar-site}"
ONEK="bitcosar"
INGRESS="${INGRESS:-0}"

KUBECTL=""
WINDOWS_KUBECTL=0
if command -v kubectl >/dev/null 2>&1; then
  KUBECTL=kubectl
elif [ -x "/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe" ]; then
  KUBECTL="/mnt/c/Program Files/Docker/Docker/resources/bin/kubectl.exe"
  WINDOWS_KUBECTL=1
else
  echo "HATA: kubectl bulunamadı." >&2; exit 1
fi
: "${KUBECONFIG:=/mnt/d/talos/.kube/config}"
if [ "$WINDOWS_KUBECTL" = 1 ] && [ "${KUBECONFIG#/mnt/}" != "$KUBECONFIG" ]; then
  KUBECONFIG=$(wslpath -w "$KUBECONFIG")
fi
kctl() {
  if [ "$WINDOWS_KUBECTL" = 1 ]; then
    "$KUBECTL" --kubeconfig "$KUBECONFIG" "$@"
  else
    "$KUBECTL" "$@"
  fi
}
uygula() {
  if [ "$WINDOWS_KUBECTL" = 1 ]; then
    kctl apply -f "$(wslpath -w "$1")"
  else
    kctl apply -f "$1"
  fi
}

URET_ONLY=0
ETIKET=""
for a in "$@"; do
  case "$a" in
    --uret-only) URET_ONLY=1 ;;
    -*) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
    *) ETIKET="$a" ;;
  esac
done

if [ -z "$ETIKET" ]; then
  cd "$KOK"
  ETIKET="sha-$(git rev-parse --short HEAD)"
  KIRLI=$(git status --porcelain | wc -l)
  [ "$KIRLI" -gt 0 ] && echo "UYARI: ağaçta $KIRLI commit'lenmemiş dosya var — imajda YOK."
fi

export IMAJ_ETIKETI="$ETIKET"
echo "==> imaj etiketi: $IMAJ_ETIKETI   namespace: $NS   ingress: $INGRESS"

[ -d "$K8S" ] || { echo "HATA: $K8S yok — önce bitcosar-manifest-uret.sh" >&2; exit 1; }

URETIM=$(mktemp -d)
trap 'rm -rf "$URETIM"' EXIT
for f in "$K8S"/*.yaml; do
  envsubst '${IMAJ_ETIKETI}' < "$f" > "$URETIM/$(basename "$f")"
done
if grep -rn '\${IMAJ_ETIKETI}' "$URETIM"/*.yaml; then
  echo "HATA: doldurulmamış yer tutucu kaldı." >&2; exit 1
fi
if [ "$URET_ONLY" = 1 ]; then
  KAYIT="${URET_DIZIN:-$KOK/.k8s-uretim-bitcosar}"
  mkdir -p "$KAYIT"; cp "$URETIM"/*.yaml "$KAYIT"/
  echo "==> basılan YAML: $KAYIT"; exit 0
fi

echo "==> namespace + kota"
uygula "$URETIM/00-namespace.yaml" || exit 1
echo "==> servis hesapları"
uygula "$URETIM/07-rbac.yaml" || exit 1

# Postgres TLS: Secret yoksa postgres de site de BAŞLAMAZ (fail closed).
bash "$KOK/deploy/scripts/bitcosar-pg-tls-uret.sh" || {
  echo "HATA: Postgres TLS üretilemedi." >&2; exit 1; }

echo "==> PVC + ağ politikaları"
uygula "$URETIM/10-pvc.yaml" || exit 1
uygula "$URETIM/06-networkpolicy.yaml" || exit 1

# Kyverno kurulu değilse politika uygulamak sevki kırar.
if kctl get crd clusterpolicies.kyverno.io >/dev/null 2>&1; then
  uygula "$URETIM/05-kyverno.yaml" || exit 1
  echo "==> Kyverno politikaları uygulandı"
else
  echo "    Kyverno kurulu değil — imza zorunluluğu AKTİF DEĞİL"
fi

echo "==> veri katmanı"
uygula "$URETIM/20-postgres.yaml" || exit 1
uygula "$URETIM/21-redis.yaml"    || exit 1
kctl -n "$NS" rollout status deploy/${ONEK}-postgres --timeout=240s || exit 1
kctl -n "$NS" rollout status deploy/${ONEK}-redis    --timeout=180s || exit 1

# VERİ GERİ YÜKLEME (isteğe bağlı): postgres ayağa kalktıktan SONRA,
# uygulamadan ÖNCE. Uygulama önce kalksaydı migrate init'i boş şemayı
# kurar, ardından dump'ın şeması "zaten var" diye çakışırdı.
#
# Veritabanı DROP/CREATE edilir: `pg_restore --clean` uzantıları (pg_trgm
# vb.) düşüremeyip --exit-on-error ile yarıda kalıyordu.
if [ -n "${VERI_DOSYA:-}" ]; then
  [ -f "$VERI_DOSYA" ] || { echo "HATA: $VERI_DOSYA yok" >&2; exit 1; }
  echo "==> veri geri yükleniyor: $VERI_DOSYA"
  POD=$(kctl -n "$NS" get pod -l app=${ONEK}-postgres -o name | head -1)
  [ -n "$POD" ] || { echo "HATA: postgres pod'u yok" >&2; exit 1; }
  POD=${POD#pod/}
  kctl -n "$NS" cp "$VERI_DOSYA" "$POD:/tmp/gecis.dump" || exit 1
  kctl -n "$NS" exec "$POD" -- sh -c '
    set -e
    export PGPASSWORD="$POSTGRES_PASSWORD"
    psql -h 127.0.0.1 -U "$POSTGRES_USER" -d postgres -q \
      -c "DROP DATABASE IF EXISTS \"$POSTGRES_DB\" WITH (FORCE)" \
      -c "CREATE DATABASE \"$POSTGRES_DB\" OWNER \"$POSTGRES_USER\""
    pg_restore -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
      --no-owner --no-acl --exit-on-error /tmp/gecis.dump
    rm -f /tmp/gecis.dump
    psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAF"|" -c \
      "select (select count(*) from contents) icerik,
              (select count(*) from contents where status = '"'"'yayinda'"'"') yayinda,
              (select count(*) from feeds) kaynak,
              (select count(*) from schema_migrations) migration"
  ' || { echo "HATA: veri yüklenemedi" >&2; exit 1; }
fi

echo "==> uygulama"
uygula "$URETIM/30-mikroservisler.yaml" || exit 1
uygula "$URETIM/40-site.yaml"           || exit 1

for d in ${ONEK}-image-svc ${ONEK}-search-svc ${ONEK}-collab ${ONEK}-site; do
  kctl -n "$NS" rollout status "deploy/$d" --timeout=600s || {
    echo "HATA: $d açılmadı. Geri dönüş: $KUBECTL -n $NS rollout undo deploy/$d" >&2
    exit 1
  }
done

if [ "$INGRESS" = 1 ]; then
  echo "==> INGRESS uygulanıyor (trafik devralınıyor)"
  uygula "$URETIM/50-ingress.yaml" || exit 1
else
  echo "==> ingress UYGULANMADI (trafik hâlâ WordPress'te)"
fi

echo
echo "==> sevk edilen imajlar"
kctl -n "$NS" get deploy -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'
echo
echo "sevk tamam: $IMAJ_ETIKETI"
