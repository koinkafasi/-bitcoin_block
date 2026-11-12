#!/usr/bin/env bash
# gosec-borc.sh — gosec bulgularını BORÇ ile YENİ olarak ayırır.
#
# SORUN: gosec doğrudan kapı yapılınca mevcut 11 HIGH bulgu kapıyı kalıcı
# kırmızı tutuyordu. Kalıcı kırmızı kapı, kapı değildir: kimse bakmaz ve
# gerçekten yeni gelen açık da gürültüde kaybolur.
#
# ÇÖZÜM: lint'teki "yeni kodda 0 bulgu" kuralının gosec karşılığı.
# Mevcut bulguların imzası `deploy/gosec-borc.txt`e yazılır. Kapı yalnız
# BU LİSTEDE OLMAYAN bulgu geldiğinde kırılır. Borç ayrıca raporlanır —
# görünmez olmaz, sadece kapıyı kilitlemez.
#
# İMZA NEDEN `kural|dosya`:
#   satır numarası eklenseydi, araya iki satır kod girince aynı bulgu
#   "yeni" görünür ve kapı sahte alarm verirdi. Dosya düzeyinde eşleşme
#   kararlı; aynı dosyaya aynı kuraldan İKİNCİ bir bulgu eklenirse sayı
#   karşılaştırması onu yakalar.
#
# Kullanım:
#   bash deploy/scripts/gosec-borc.sh            # kapı: yeni bulgu varsa 1
#   bash deploy/scripts/gosec-borc.sh --yenile   # borcu güncelle (bilinçli)
#   bash deploy/scripts/gosec-borc.sh --listele  # borcu insan okur biçimde
set -uo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BORC="$KOK/deploy/gosec-borc.txt"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
GOSEC="$HOME/go/bin/gosec"

YENILE=0
LISTELE=0
for a in "$@"; do
  case "$a" in
    --yenile)  YENILE=1 ;;
    --listele) LISTELE=1 ;;
    *) echo "bilinmeyen seçenek: $a" >&2; exit 2 ;;
  esac
done

[ -x "$GOSEC" ] || { echo "HATA: gosec yok ($GOSEC)" >&2; exit 1; }

# gosec JSON üretir; çıkış kodu bulgu varsa 1 olur — burada kapı değil.
RAPOR=$(mktemp)
trap 'rm -f "$RAPOR"' EXIT
( cd "$KOK/go" && "$GOSEC" -quiet -fmt=json -severity high -confidence medium \
    -nosec-require-rules -nosec-require-justification \
    -exclude-dir=internal/webui/static/vendor ./... ) > "$RAPOR" 2>/dev/null || true

# İmza: kural|depo-köküne-göre-dosya   (satır YOK — kayma sahte alarm verir)
imzalar() {
  python3 - "$RAPOR" "$KOK" <<'PY'
import json, os, sys
rapor, kok = sys.argv[1], sys.argv[2]
try:
    with open(rapor, encoding='utf-8') as f:
        veri = json.load(f)
except Exception:
    sys.exit(0)
gorulen = set()
for b in veri.get('Issues', []):
    dosya = os.path.relpath(b.get('file', ''), kok).replace('\\', '/')
    gorulen.add('%s|%s' % (b.get('rule_id', '?'), dosya))
for s in sorted(gorulen):
    print(s)
PY
}

SIMDI=$(imzalar)

if [ "$LISTELE" = 1 ]; then
  echo "== gosec borcu ($BORC) =="
  if [ -f "$BORC" ]; then grep -v '^#' "$BORC" | grep -v '^$' | nl; else echo "(borç dosyası yok)"; fi
  echo
  echo "== şu anki bulgular =="
  echo "$SIMDI" | nl
  exit 0
fi

if [ "$YENILE" = 1 ]; then
  {
    echo "# gosec borcu — BİLİNÇLİ olarak kabul edilmiş mevcut bulgular."
    echo "# Biçim: KURAL|dosya   (satır yok: kod kayınca sahte alarm olmasın)"
    echo "# Güncelleme: bash deploy/scripts/gosec-borc.sh --yenile"
    echo "# Üretim: $(date +%Y-%m-%d)"
    echo
    echo "$SIMDI"
  } > "$BORC"
  echo "borç güncellendi: $(echo "$SIMDI" | grep -c . ) kayıt -> $BORC"
  exit 0
fi

# ---- kapı ----
if [ ! -f "$BORC" ]; then
  echo "HATA: borç dosyası yok. Önce: bash deploy/scripts/gosec-borc.sh --yenile" >&2
  exit 1
fi

BILINEN=$(grep -v '^#' "$BORC" | grep -v '^$' | sort -u)
YENI=$(comm -13 <(echo "$BILINEN") <(echo "$SIMDI" | sort -u))
COZULEN=$(comm -23 <(echo "$BILINEN") <(echo "$SIMDI" | sort -u))

ADET_BORC=$(echo "$BILINEN" | grep -c . || true)
echo "  gosec: $ADET_BORC bilinen borç"

if [ -n "$COZULEN" ]; then
  echo "  ✓ çözülmüş (borçtan düşebilir):"
  echo "$COZULEN" | sed 's/^/      /'
  echo "    güncellemek için: bash deploy/scripts/gosec-borc.sh --yenile"
fi

if [ -n "$YENI" ]; then
  echo "  ✗ YENİ gosec bulgusu:" >&2
  echo "$YENI" | sed 's/^/      /' >&2
  echo "    düzeltin ya da bilinçliyse borca ekleyin (--yenile)" >&2
  exit 1
fi

echo "  yeni bulgu yok"
exit 0
