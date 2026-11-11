#!/usr/bin/env bash
# pg-tls-uret.sh — Postgres TLS sertifikalarını üretir ve Secret'a yazar.
#
# NEDEN: `sslmode=disable` aynı düğümde bile ağ katmanında trafiği AÇIK
# bırakır. NetworkPolicy segmentasyon sağlar ama dinleme (sniffing) ve
# yedek trafiği hâlâ düz metindir. TLS, "ağa güveniyoruz" varsayımını
# ortadan kaldırır — defense in depth.
#
# TASARIM: küme içi CA yönetimi (cert-manager) TEK düğümlü kümede fazla
# ağırlık. Bunun yerine sertifikalar bir kez burada üretilir, Secret'a
# yazılır, pod'lar bağlar. Yenileme = betiği tekrar koş + rollout.
#
# KULLANIM:
#   bash deploy/scripts/pg-tls-uret.sh              # üret (varsa atla)
#   bash deploy/scripts/pg-tls-uret.sh --yenile     # zorla yeniden üret
#
# ROTASYON (yıllık veya sızıntı şüphesinde):
#   1) bash deploy/scripts/pg-tls-uret.sh --yenile
#   2) kubectl -n theblock rollout restart deploy/theblock-postgres
#   3) kubectl -n theblock rollout restart deploy/theblock-site
#      (2 ve 3 sırası ÖNEMLİ: CA yenilenirse postgres ve site AYNI CA'yı
#       görmeli; arada kesinti ~10 sn.)
set -uo pipefail

NS="${NAMESPACE:-theblock}"
SIR="theblock-pg-tls"
GECERLILIK_GUN=825          # 2 yıl ömür, ~1 yıl sonra yenile
YENILE=0
[ "${1:-}" = "--yenile" ] && YENILE=1

command -v kubectl >/dev/null 2>&1 || { echo "kubectl gerekli" >&2; exit 1; }
command -v openssl >/dev/null 2>&1 || { echo "openssl gerekli" >&2; exit 1; }

# Mevcut sertifika geçerliyse ve --yenile verilmediyse dokunma.
if [ "$YENILE" = 0 ] && kubectl -n "$NS" get secret "$SIR" >/dev/null 2>&1; then
  bitis=$(kubectl -n "$NS" get secret "$SIR" -o jsonpath='{.data.server\.crt}' \
    | base64 -d | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
  if [ -n "$bitis" ]; then
    kalan=$(( ( $(date -d "$bitis" +%s) - $(date +%s) ) / 86400 ))
    if [ "$kalan" -gt 60 ]; then
      echo "==> sertifika geçerli (${kalan} gün kaldı) — atlanıyor"
      exit 0
    fi
    echo "==> sertifika ${kalan} gün içinde bitiyor — yenileniyor"
  fi
fi

D=$(mktemp -d)
trap 'rm -rf "$D"' EXIT
cd "$D" || exit 1

# --- CA ------------------------------------------------------------------
openssl req -x509 -newkey rsa:4096 -sha256 -days "$GECERLILIK_GUN" -nodes \
  -keyout ca.key -out ca.crt \
  -subj "/CN=theblock-postgres-ca/O=bitcosar" 2>/dev/null

# --- Sunucu sertifikası --------------------------------------------------
# SAN'lar ZORUNLU: `sslmode=verify-full` CN'e değil SAN'a bakar. Eksik SAN
# "certificate is valid for ..., not theblock-postgres" hatası verir.
cat > srv.cnf <<'CNF'
[req]
distinguished_name = dn
req_extensions = ext
prompt = no
[dn]
CN = theblock-postgres
O = bitcosar
[ext]
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @san
[san]
DNS.1 = theblock-postgres
DNS.2 = theblock-postgres.theblock
DNS.3 = theblock-postgres.theblock.svc
DNS.4 = theblock-postgres.theblock.svc.cluster.local
DNS.5 = localhost
IP.1 = 127.0.0.1
CNF

openssl req -newkey rsa:4096 -sha256 -nodes -keyout server.key -out server.csr \
  -config srv.cnf 2>/dev/null
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days "$GECERLILIK_GUN" -sha256 \
  -extensions ext -extfile srv.cnf 2>/dev/null

# Postgres, anahtarın yalnız sahibi tarafından okunmasını ister.
chmod 600 server.key

echo "==> Secret yazılıyor: $NS/$SIR"
kubectl -n "$NS" create secret generic "$SIR" \
  --from-file=ca.crt=ca.crt \
  --from-file=server.crt=server.crt \
  --from-file=server.key=server.key \
  --from-file=pg_hba.conf=/dev/stdin \
  --dry-run=client -o yaml <<'HBA' | kubectl apply -f - >/dev/null
# Yalnız TLS'li TCP bağlantı kabul edilir (`hostssl`); düz TCP REDDEDİLİR.
# Unix soketi (konteyner içi) local satırıyla çalışır.
local   all             all                                     scram-sha-256
hostssl all             all             0.0.0.0/0               scram-sha-256
hostssl all             all             ::/0                    scram-sha-256
hostnossl all           all             0.0.0.0/0               reject
hostnossl all           all             ::/0                    reject
HBA

echo "==> doğrulama"
openssl verify -CAfile ca.crt server.crt
bitis=$(openssl x509 -in server.crt -noout -enddate | cut -d= -f2)
echo "    geçerlilik bitişi: $bitis"
echo
echo "SIRADAKI ADIM (CA yenilendiyse ZORUNLU):"
echo "  kubectl -n $NS rollout restart deploy/theblock-postgres"
echo "  kubectl -n $NS rollout restart deploy/theblock-site"
