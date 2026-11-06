#!/usr/bin/env bash
# dagit.sh — site servisini çalışma ağacından derleyip yeniden başlatır.
#
# Servis ve port: DAGIT_SERVIS / DAGIT_PORT ile ezilebilir.
# Varsayılan: bitcosar-site.service · 8090
#
# Neden betik: dağıtım elle iki komutla yapılıyordu ve her seferinde iki şey
# ters gidiyordu. Birincisi, komut Windows terminalinde çalıştırılınca
# "/home/alper/..." yolu bulunamıyordu (proje WSL'de). İkincisi, derleme
# doğrudan bin/site üzerine yazdığı için başarısız bir derleme çalışan
# ikiliyi bozabiliyordu.
#
# Burada derleme önce geçici bir dosyaya yapılır, ancak başarılıysa yerine
# taşınır; servis yeniden başlatılır ve sağlık kontrolü yapılır. Sağlık
# gelmezse çıkış kodu sıfır değildir ve günlüğün son satırları basılır.
#
# Kullanım (WSL içinden):   bash deploy/scripts/dagit.sh
# Kullanım (Windows'tan):   wsl -d Ubuntu -- bash -lc "cd /home/alper/bitcosar && bash deploy/scripts/dagit.sh"
#
# Seçenekler:
#   --atla-test    testleri atla (acil düzeltme için; varsayılan koşar)
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK"

# TEHLIKELI VARSAYILAN DUZELTILDI (6 Kasim 2026): burada
# `SERVIS=theblock-site.service` ve `PORT=8070` yaziliyordu — theblock'tan
# kalma. O servis bu makinede GERCEKTEN var ve ayri bir projeyi (~/theblock)
# calistiriyor. Betik bu depodan kosturulunca bitcosar'in ikilisini derleyip
# THEBLOCK'u yeniden baslatiyordu; sagligi da 8070'ten (theblock'un portu)
# okuyup "TAMAM" diyordu. Yani yanlis servisi yeniden baslatip basarili
# rapor veren bir dagitim.
#
# Artik depo adindan turetiliyor, ortam degiskeniyle ezilebiliyor.
SERVIS="${DAGIT_SERVIS:-bitcosar-site.service}"
PORT="${DAGIT_PORT:-8090}"
ATLA_TEST=0
[ "${1:-}" = "--atla-test" ] && ATLA_TEST=1

echo "==> 1/5 depo durumu"
echo "    HEAD: $(git log -1 --format='%h %s' | cut -c1-72)"
KIRLI=$(git status --porcelain)
if [ -n "$KIRLI" ]; then
  echo "    UYARI: çalışma ağacı temiz değil — commit'lenmemiş değişiklikler de dağıtılacak:"
  echo "$KIRLI" | sed 's/^/      /'
fi

cd go

echo "==> 2/5 biçim ve vet"
BICIM=$(gofmt -l ./internal/ ./cmd/)
if [ -n "$BICIM" ]; then
  echo "    DURDURULDU: gofmt'lenmemiş dosyalar var:" >&2
  echo "$BICIM" | sed 's/^/      /' >&2
  exit 1
fi
if ! go vet ./... > /tmp/dagit-vet.log 2>&1; then
  echo "    DURDURULDU: go vet temiz değil" >&2
  head -20 /tmp/dagit-vet.log >&2
  exit 1
fi
echo "    temiz"

if [ "$ATLA_TEST" -eq 0 ]; then
  echo "==> 3/5 testler"
  if ! go test ./... > /tmp/dagit-test.log 2>&1; then
    echo "    DURDURULDU: testler geçmiyor (ayrıntı /tmp/dagit-test.log)" >&2
    grep -E '^(FAIL|---)' /tmp/dagit-test.log | head -10 >&2
    exit 1
  fi
  echo "    geçen paket: $(grep -c '^ok' /tmp/dagit-test.log)"
else
  echo "==> 3/5 testler ATLANDI (--atla-test)"
fi

echo "==> 4/5 derleme"
GECICI=$(mktemp /tmp/site-XXXXXX)
if ! go build -o "$GECICI" ./cmd/site; then
  rm -f "$GECICI"
  echo "    DERLEME BAŞARISIZ — çalışan ikiliye dokunulmadı" >&2
  exit 1
fi
mkdir -p bin
mv "$GECICI" bin/site
chmod +x bin/site
echo "    bin/site $(stat -c%s bin/site) bayt"

echo "==> 5/5 servis"
ONCE=$(pgrep -f "$KOK/go/bin/site" | head -1)
systemctl --user restart "$SERVIS" || { echo "    RESTART BAŞARISIZ" >&2; exit 1; }

HAZIR=0
for _ in $(seq 1 60); do
  if curl -fsS -o /dev/null "http://127.0.0.1:${PORT}/healthz" 2>/dev/null; then HAZIR=1; break; fi
done
SONRA=$(pgrep -f "$KOK/go/bin/site" | head -1)
echo "    pid $ONCE -> $SONRA · servis: $(systemctl --user is-active "$SERVIS")"

if [ "$HAZIR" -ne 1 ]; then
  echo "    SAĞLIK GELMEDİ — son günlük:" >&2
  journalctl --user -u "$SERVIS" --since '2 min ago' --no-pager 2>/dev/null | tail -20 >&2
  exit 1
fi
curl -fsS -o /dev/null -w "    /healthz HTTP %{http_code}\n" "http://127.0.0.1:${PORT}/healthz"
curl -fsS -o /dev/null -w "    /        HTTP %{http_code}  %{size_download} bayt\n" "http://127.0.0.1:${PORT}/"

echo
echo "==> zamanlayıcı günlüğü"
journalctl --user -u "$SERVIS" --since '2 min ago' --no-pager 2>/dev/null \
  | grep -iE 'içerik skoru|bot poller|başladı' | tail -6
echo
echo "DAĞITIM TAMAM"
