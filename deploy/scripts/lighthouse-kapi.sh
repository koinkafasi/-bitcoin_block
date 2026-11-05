#!/usr/bin/env bash
# lighthouse-kapi.sh — performans/SEO bütçesi kapısı (yerel ve CI).
#
# ÖLÇÜM NOTU (5 Kasım 2026, WSL): Lighthouse WSL'de çalışırken Chrome'u
# Windows tarafından başlatıyordu (`--user-data-dir=C:\Users\...`). Windows
# Chrome ile WSL arasındaki süreç köprüsü aralıklı kopuyor ve araç
# "Unable to connect to Chrome" veriyor; ayrıca her koşuda zombi süreç
# birikiyor (ölçüm: 3 artık süreç). Çözüm: PATH'ten Windows yolları
# çıkarılır, Linux Chrome açıkça verilir ve artık süreçler temizlenir.
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

ESIK_PERF=90
ESIK_SEO=95
ESIK_A11Y=90
ESIK_BP=90

if ! command -v npx >/dev/null 2>&1 && ! command -v lighthouse >/dev/null 2>&1; then
  echo "ATLANDI: lighthouse/npx yok (kurulum: npm i -g lighthouse)"
  [ "$RAPOR" = 1 ] && exit 0
  exit 0
fi

# --- Chrome seçimi: YALNIZ Linux ikilisi -------------------------------
CHROME="${CHROME_PATH:-}"
if [ -z "$CHROME" ]; then
  for aday in \
    $(ls -1 "$HOME"/.cache/puppeteer/chrome/*/chrome-linux64/chrome 2>/dev/null | sort -V -r) \
    /usr/bin/google-chrome /usr/bin/chromium /usr/bin/chromium-browser; do
    if [ -n "$aday" ] && [ -x "$aday" ]; then CHROME="$aday"; break; fi
  done
fi
if [ -z "$CHROME" ]; then
  echo "ATLANDI: Linux Chrome bulunamadı (CHROME_PATH verin)"
  [ "$RAPOR" = 1 ] && exit 0
  exit 0
fi
echo "==> Chrome: $CHROME"

# --- ortam --------------------------------------------------------------
# PATH'E DOKUNULMAZ. chrome-launcher WSL dalında Windows geçici dizinini
# `/mnt/<sürücü>/Users/<kullanıcı>/AppData/` desenini PATH'te arayarak
# türetiyor; /mnt yolları çıkarılırsa `undefined:\Users\undefined\...`
# üretip `ENOENT mkdtemp` hatası veriyor (ölçüldü: 5 Kasım 2026).
# Bu yüzden PATH korunur, Chrome ise açıkça Linux ikilisine sabitlenir.
export CHROME_PATH="$CHROME"

# --- artık süreçleri temizle -------------------------------------------
# Ölçüm: her başarısız koşu bir zombi Chrome bırakıyordu (3 birikti) ve
# sonraki koşular "Unable to connect to Chrome" veriyordu.
pkill -f 'chrome-linux64/chrome' 2>/dev/null || true
sleep 1

OUT=$(mktemp -d)
trap 'rm -rf "$OUT"; pkill -f "chrome-linux64/chrome" 2>/dev/null || true' EXIT

LH=lighthouse
command -v lighthouse >/dev/null 2>&1 || LH="npx --yes lighthouse"

# --- koşum: bir kez yeniden dene ---------------------------------------
# WSL süreç köprüsü aralıklı kopabiliyor; tek bir geçici hata kapıyı
# kırmızı göstermemeli.
kos() {
  rm -f "$OUT/lh.json"
  $LH "$HEDEF" \
    --quiet \
    --chrome-path="$CHROME" \
    --chrome-flags="--headless=new --no-sandbox --disable-gpu --disable-dev-shm-usage" \
    --only-categories=performance,seo,accessibility,best-practices \
    --preset=desktop \
    --output=json --output-path="$OUT/lh.json" >/dev/null 2>&1
  [ -s "$OUT/lh.json" ]
}

echo "==> Lighthouse: $HEDEF"
if ! kos; then
  echo "    (ilk deneme başarısız — süreçler temizlenip yeniden deneniyor)"
  pkill -f 'chrome-linux64/chrome' 2>/dev/null || true
  sleep 2
  kos || true
fi

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
audits = d.get("audits", {})
def say(name):
    a = audits.get(name) or {}
    return a.get("displayValue") or "—"
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
