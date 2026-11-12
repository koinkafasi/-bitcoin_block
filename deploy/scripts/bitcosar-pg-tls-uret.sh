#!/usr/bin/env bash
# bitcosar-pg-tls-uret.sh — bitcosar-site için Postgres TLS materyali üretir.
#
# NEDEN AYRI BETİK: `pg-tls-uret.sh` theblock'a sabit. Sertifikadaki SAN'lar
# `theblock-postgres.theblock.svc…` biçiminde; `sslmode=verify-full` host adını
# SAN ile karşılaştırır, yanlış SAN'lı sertifika ile site BAĞLANAMAZ.
# Bu betik aynı üretim mantığını bitcosar adlarıyla yapar.
#
# KULLANIM:
#   bash deploy/scripts/bitcosar-pg-tls-uret.sh            # üret (varsa atla)
#   bash deploy/scripts/bitcosar-pg-tls-uret.sh --yenile   # zorla yeniden üret
set -uo pipefail

NS="${NAMESPACE:-bitcosar-site}"
SIR="bitcosar-pg-tls"
SUNUCU="bitcosar-postgres"
GECERLILIK_GUN=825
YENILE=0
[ "${1:-}" = "--yenile" ] && YENILE=1

# kubectl: WSL'de kurulu değil; Docker Desktop'ın Windows ikilisi kullanılır.
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

command -v openssl >/dev/null 2>&1 || { echo "openssl gerekli" >&2; exit 1; }

if [ "$YENILE" = 0 ] && kctl -n "$NS" get secret "$SIR" >/dev/null 2>&1; then
  bitis=$(kctl -n "$NS" get secret "$SIR" -o jsonpath='{.data.server\.crt}' \
    | base64 -d | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
  if [ -n "$bitis" ]; then
    kalan=$(( ( $(date -d "$bitis" +%s) - $(date +%s) ) / 86400 ))
    if [ "$kalan" -gt 60 ]; then
      echo "==> sertifika geçerli (${kalan} gün kaldı) — atlanıyor"
      exit 0
    fi
  fi
fi

D=$(mktemp -d)
trap 'rm -rf "$D"' EXIT
cd "$D" || exit 1

openssl req -x509 -newkey rsa:4096 -sha256 -days "$GECERLILIK_GUN" -nodes \
  -keyout ca.key -out ca.crt \
  -subj "/CN=${SUNUCU}-ca/O=bitcosar" 2>/dev/null

# SAN listesi: kısa ad (DATABASE_URL'de geçen), namespace'li ve FQDN biçimleri.
cat > srv.cnf <<CNF
[req]
distinguished_name = dn
req_extensions = ext
prompt = no
[dn]
CN = ${SUNUCU}
O = bitcosar
[ext]
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @san
[san]
DNS.1 = ${SUNUCU}
DNS.2 = ${SUNUCU}.${NS}
DNS.3 = ${SUNUCU}.${NS}.svc
DNS.4 = ${SUNUCU}.${NS}.svc.cluster.local
DNS.5 = localhost
IP.1 = 127.0.0.1
CNF

openssl req -newkey rsa:4096 -sha256 -nodes -keyout server.key -out server.csr \
  -config srv.cnf 2>/dev/null
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days "$GECERLILIK_GUN" -sha256 \
  -extensions ext -extfile srv.cnf 2>/dev/null
chmod 600 server.key

printf '%s\n' \
  'local   all             all                                     scram-sha-256' \
  'hostssl all             all             0.0.0.0/0               scram-sha-256' \
  'hostssl all             all             ::/0                    scram-sha-256' \
  'hostnossl all           all             0.0.0.0/0               reject' \
  'hostnossl all           all             ::/0                    reject' > pg_hba.conf

echo "==> Secret yazılıyor: $NS/$SIR"
kctl -n "$NS" create secret generic "$SIR" \
  --from-file=ca.crt=ca.crt \
  --from-file=server.crt=server.crt \
  --from-file=server.key=server.key \
  --from-file=pg_hba.conf=pg_hba.conf \
  --dry-run=client -o yaml | kctl apply -f - >/dev/null

echo "==> doğrulama"
openssl verify -CAfile ca.crt server.crt
echo "    SAN: $(openssl x509 -in server.crt -noout -ext subjectAltName | tail -1 | sed 's/^ *//')"
echo "    bitiş: $(openssl x509 -in server.crt -noout -enddate | cut -d= -f2)"
