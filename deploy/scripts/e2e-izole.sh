#!/usr/bin/env bash
# E2E testlerini ÜRETİME DOKUNMADAN koşar.
#
# Neden gerekli: `make e2e` varsayılan olarak http://127.0.0.1:8070 adresine
# gider ve o portta canlı site servisi (theblock-site.service) çalışır —
# cloudflared ile theblock.com.tr'ye tünellenmiş durumda. E2E testlerinin
# büyük bölümü panele POST eder, yani yerel `make e2e` canlı veritabanında
# içerik oluşturup silerdi.
#
# Bu betik CI'ın (.github/workflows/ci.yml, e2e işi) yaptığını birebir yerelde
# kurar ve üretimden dört noktada ayrışır:
#   1. Ayrı veritabanı  (theblock_e2e, sıfırdan migration + e2e_seed.sql)
#   2. Ayrı ikili       (go/bin/site ÜZERİNE YAZMAZ — o çalışan servisin ikilisi)
#   3. Ayrı port + depolama dizini
#   4. deploy/.env YÜKLENMEZ — üretim Redis'i ve gerçek API anahtarları devrede
#      değildir; yalnız CI'ın verdiği dört değişken geçilir.
#
# Kullanım:
#   bash deploy/scripts/e2e-izole.sh                 # tam paket
#   bash deploy/scripts/e2e-izole.sh tests/smoke.test.js
set -euo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK"

PORT="${E2E_IZOLE_PORT:-8099}"
DB_ADI="${E2E_IZOLE_DB:-theblock_e2e}"
IS_DIZIN="${E2E_IZOLE_DIR:-/tmp/theblock-e2e}"
PG_KAP="${E2E_IZOLE_PG_CONTAINER:-theblock-postgres-1}"

if [ ! -f deploy/.env ]; then
  echo "deploy/.env yok — Postgres parolası okunamıyor" >&2
  exit 1
fi
# Parolayı yalnız bağlantı dizesini kurmak için oku; ortama yayma.
PGPASS=$(grep '^POSTGRES_PASSWORD=' deploy/.env | cut -d= -f2-)
PGKUL=$(grep '^POSTGRES_USER=' deploy/.env | cut -d= -f2-)
PGKUL="${PGKUL:-theblock}"
if [ -z "$PGPASS" ]; then
  echo "deploy/.env içinde POSTGRES_PASSWORD bulunamadı" >&2
  exit 1
fi
E2E_DB_URL="postgres://${PGKUL}:${PGPASS}@127.0.0.1:5433/${DB_ADI}?sslmode=disable"

# Üretim portuna yanlışlıkla bağlanmayı imkânsız kıl.
if [ "$PORT" = "8070" ]; then
  echo "E2E_IZOLE_PORT=8070 üretim portudur; başka bir port seçin" >&2
  exit 1
fi
if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
  echo "port ${PORT} dolu — orada bir şey çalışıyor, durdurun ya da E2E_IZOLE_PORT verin" >&2
  exit 1
fi

echo "==> 1/5 test veritabanı sıfırlanıyor (${DB_ADI})"
docker exec "$PG_KAP" psql -U "$PGKUL" -d postgres \
  -c "DROP DATABASE IF EXISTS ${DB_ADI}" >/dev/null
docker exec "$PG_KAP" psql -U "$PGKUL" -d postgres \
  -c "CREATE DATABASE ${DB_ADI}" >/dev/null

echo "==> 2/5 migration + test verisi"
( cd go && DATABASE_URL="$E2E_DB_URL" go run ./cmd/migrate up >/dev/null )
( cd go && DATABASE_URL="$E2E_DB_URL" go run ./cmd/dbexec ../deploy/e2e_seed.sql >/dev/null )

echo "==> 3/5 ayrı ikili derleniyor (${IS_DIZIN}/site)"
mkdir -p "${IS_DIZIN}/storage/media/202611"
# E2E kapak fikstürü: e2e_seed.sql media_files satırı bu yolu bekler.
cp deploy/e2e-media/kapak.png "${IS_DIZIN}/storage/media/202611/e2e-kapak.png"
( cd go && go build -mod=mod -o "${IS_DIZIN}/site" ./cmd/site )

echo "==> 4/5 test örneği başlatılıyor (port ${PORT})"
# env -i: üretim ortam değişkenleri sızmasın. CI ile aynı dört değişken.
env -i PATH="$PATH" HOME="$HOME" \
    DATABASE_URL="$E2E_DB_URL" \
    PORT="$PORT" \
    APP_ENV=test \
    APP_BASE_URL="http://127.0.0.1:${PORT}" \
    STORAGE_DIR="${IS_DIZIN}/storage" \
    "${IS_DIZIN}/site" > "${IS_DIZIN}/site.log" 2>&1 &
SITE_PID=$!
temizle() {
  if kill -0 "$SITE_PID" 2>/dev/null; then
    kill "$SITE_PID" 2>/dev/null || true
    wait "$SITE_PID" 2>/dev/null || true
  fi
}
trap temizle EXIT

for i in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then break; fi
  if ! kill -0 "$SITE_PID" 2>/dev/null; then
    echo "test örneği öldü:" >&2; tail -30 "${IS_DIZIN}/site.log" >&2; exit 1
  fi
  sleep 1
done
if ! curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
  echo "test örneği ayağa kalkmadı:" >&2; tail -30 "${IS_DIZIN}/site.log" >&2; exit 1
fi

echo "==> 5/5 E2E koşuyor"
# e2e_seed.sql deterministik admin oturumu ekler; admin panelleri de kapsanır.
cd e2e
set +e
if [ "$#" -gt 0 ]; then
  E2E_BASE_URL="http://127.0.0.1:${PORT}" \
  TB_SESSION=e2ec0000-0000-4000-8000-000000000001 \
  node --test --test-concurrency=1 --test-timeout=180000 "$@"
else
  E2E_BASE_URL="http://127.0.0.1:${PORT}" \
  TB_SESSION=e2ec0000-0000-4000-8000-000000000001 \
  npm test
fi
SONUC=$?
set -e
echo "==> bitti (çıkış kodu ${SONUC}); sunucu günlüğü: ${IS_DIZIN}/site.log"
exit $SONUC
