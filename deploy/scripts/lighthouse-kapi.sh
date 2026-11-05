#!/usr/bin/env bash
# lighthouse-kapi.sh — performans/SEO bütçesi kapısı (yerel ve CI).
#
# NEDEN: "hızlı" demek yetmez; ölçülebilir eşik gerekir. CoinDesk hedefi:
# TTFB < 80 ms (edge), LCP < 1.5 s, HTML < 60 KB gzip, JS ~0.
#
# Kullanım:
#   bash deploy/scripts/lighthouse-kapi.sh              # CI eşiği
#   bash deploy/scripts/lighthouse-kapi.sh --rapor      # kırmaz
#   URL=https://demo.bitcosar.com bash ...              # başka hedef
#
# Lighthouse kurulu değilse ATLAR (sessizce yeşil göstermez, açıkça yazar).
set -uo pipefail

HEDEF="${URL:-http://127.0.0.1:8090/}"
RAPOR=0
[ "${1:-}" = "--rapor" ] && RAPOR=1

# Eşikler (Lighthouse 0-100). SEO ve erişilebilirlik 100'e yakın olmalı;
# performans mobil ağda 90 altına düşerse gerçek kullanıcı etkilenir.
ESIK_PERF=90
ESIK_SEO=95
ESIK_A11Y=90
ESIK_BP=90

if ! command -v lighthouse >/dev/null 2>&1 && ! command -v npx >/dev/null 2>&1; then
  echo "ATLANDI: lighthouse/npx yok (kurulum: npm i -g lighthouse)"
  if [ "$RAPOR" = 1 ]; then exit 0; fi
  exit 0
fi

OUT=$(mktemp -d)
trap 'rm -rf "$OUT"' EXIT

LH=lighthouse
command -v lighthouse >/dev/null 2>&1 || LH="npx --yes lighthouse"

# Chrome yolu: Lighthouse varsayılan olarak sistem Chrome'unu arar; bu
# makinede kurulu değil ama Puppeteer'ın indirdiği Chrome var. Yol
# verilmezse "Unable to connect to Chrome" der ve rapor üretmez — bu,
# kapıyı sessizce atlatmak olurdu (ölçüm: 5 Kasım 2026).
CHROME="${CHROME_PATH:-}"
if [ -z "$CHROME" ]; then
  for aday in \
    "$(command -v google-chrome 2>/dev/null)" \
    "$(command -v chromium 2>/dev/null)" \
    $(ls -1 "$HOME"/.cache/puppeteer/chrome/*/chrome-linux64/chrome 2>/dev/null | sort -V | tail -1) \
    "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"; do
    if [ -n "$aday" ] && [ -x "$aday" ]; then CHROME="$aday"; break; fi
  done
fi
if [ -z "$CHROME" ]; then
  echo "ATLANDI: Chrome bulunamadı (CHROME_PATH verin)"
  [ "$RAPOR" = 1 ] && exit 0
  exit 0
fi
echo "==> Chrome: $CHROME"

export CHROME_PATH="$CHROME"

echo "==> Lighthouse: $HEDEF"
$LH "$HEDEF" \
  --quiet --chrome-path="$CHROME" \
  --chrome-flags="--headless=new --no-sandbox --disable-dev-shm-usage --disable-gpu" \
  --only-categories=performance,seo,accessibility,best-practices \
  --preset=desktop \
  --output=json --output-path="$OUT/lh.json" >/dev/null 2>&1

if [ ! -s "$OUT/lh.json" ]; then
  echo "HATA: Lighthouse raporu üretilemedi"
  [ "$RAPOR" = 1 ] && exit 0
  exit 1
fi

python3 - "$OUT/lh.json" "$ESIK_PERF" "$ESIK_SEO" "$ESIK_A11Y" "$ESIK_BP" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
esik = {"performance": int(sys.argv[2]), "seo": int(sys.argv[3]),
        "accessibility": int(sys.argv[4]), "best-practices": int(sys.argv[5])}
kaldi = 0
for k, esik_deger in esik.items():
    puan = (d.get("categories", {}).get(k) or {}).get("score")
    if puan is None:
        print(f"  {k:16s} — (ölçülmedi)")
        continue
    yuz = round(puan * 100)
    ok = yuz >= esik_deger
    print(f"  {'OK ' if ok else 'X  '} {k:16s} {yuz:3d}  (eşik {esik_deger})")
    if not ok:
        kaldi += 1
# Ağırlıklar (HTML, LCP) bilgi amaçlı
audits = d.get("audits", {})
def say(name, birim=""):
    a = audits.get(name) or {}
    v = a.get("displayValue")
    return v if v else "—"
print(f"  bilgi: LCP {say('largest-contentful-paint')} · CLS {say('cumulative-layout-shift')} · TBT {say('total-blocking-time')}")
sys.exit(1 if kaldi else 0)
PY
DURUM=$?

if [ "$DURUM" = 0 ]; then
  echo "✓ Lighthouse bütçesi geçti"
else
  echo "✗ Lighthouse bütçesi aşıldı"
  [ "$RAPOR" = 1 ] && exit 0
fi
[ "$RAPOR" = 1 ] && exit 0
exit "$DURUM"
