#!/usr/bin/env bash
# `make e2e` canlı siteye karşı koşmasın diye kapı.
#
# Port 8070'te theblock-site.service çalışır ve cloudflared onu
# theblock.com.tr'ye tünneller. E2E testlerinin çoğu admin paneline POST eder,
# yani o adrese koşan bir paket üretim veritabanına içerik yazıp silerdi.
#
# Kullanım: e2e-uretim-korumasi.sh <E2E_BASE_URL> [FORCE]
set -euo pipefail

HEDEF="${1:-}"
ZORLA="${2:-}"

if [ "$HEDEF" != "http://127.0.0.1:8070" ] && [ "$HEDEF" != "http://localhost:8070" ]; then
  exit 0
fi
if ! systemctl --user is-active --quiet theblock-site 2>/dev/null; then
  # 8070 boşsa (servis kapalı) koşmasında sakınca yok.
  exit 0
fi

cat >&2 <<'MESAJ'
DURDURULDU: E2E_BASE_URL canlı siteyi işaret ediyor (port 8070,
theblock-site.service, theblock.com.tr'ye tünellenmiş).

Testlerin büyük bölümü admin paneline POST eder — bu koşum üretim
veritabanına yazardı.

Bunun yerine:
  make e2e-izole                 ayrı veritabanı + ayrı port + ayrı ikili
  make e2e E2E_BASE_URL=http://127.0.0.1:8099    kendi test örneğiniz varsa

Riski bilerek üstleniyorsanız:
  make e2e FORCE=1
MESAJ
[ -n "$ZORLA" ] || exit 1
echo "UYARI: FORCE=1 — canlı siteye karşı koşuluyor." >&2
