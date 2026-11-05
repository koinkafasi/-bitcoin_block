#!/usr/bin/env bash
# İkon kütüphanesini YENİDEN ÜRETİR (vendor aracı).
#
# Çalışma anında dış bağımlılık YOK: bu betik bir kez çalışır, çıktıyı
# depoya yazar. Üretilen dosyalar:
#   static/vendor/tabler-icons.svg   tam set (panel)
#   static/vendor/ikon-cekirdek.svg  çekirdek set (public, ~3 KB)
#   static/vendor/ikonlar.txt        manifest (galeri + testler okur)
#   static/vendor/spinner/*.svg      animasyonlu yükleniyor SVG'leri
#   static/vendor/IKON-LISANS.txt    MIT lisans metinleri
#
# KULLANIM (ağ gerekir; yalnız ikon eklerken):
#   bash deploy/scripts/ikon-vendor.sh
#   cd go && go test ./internal/httpapi/ -run Ikon
#
# Yeni ikon gerektiğinde: deploy/scripts/ikon-uret.py içindeki KURATORLU
# listesine adı ekleyin ve bu betiği yeniden çalıştırın. Şablonlarda
# kullanılan adlar otomatik taranır; eksik kalırsa test yakalar.
set -euo pipefail

KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
VENDOR="$KOK/go/internal/webui/static/vendor"
IS=$(mktemp -d)
trap 'rm -rf "$IS"' EXIT

echo "==> 1/4 Tabler ikonları indiriliyor"
TABLER_VER=$(curl -sf https://registry.npmjs.org/@tabler/icons \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['dist-tags']['latest'])")
echo "    sürüm: $TABLER_VER"
curl -sfL "https://registry.npmjs.org/@tabler/icons/-/icons-${TABLER_VER}.tgz" -o "$IS/tabler.tgz"
tar xzf "$IS/tabler.tgz" -C "$IS"
TABLER_SRC="$IS/package"

echo "==> 2/4 SVG Spinners indiriliyor"
SPIN_VER=$(curl -sf https://registry.npmjs.org/svg-spinners \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['dist-tags']['latest'])")
echo "    sürüm: $SPIN_VER"
curl -sfL "https://registry.npmjs.org/svg-spinners/-/svg-spinners-${SPIN_VER}.tgz" -o "$IS/spin.tgz"
tar xzf "$IS/spin.tgz" -C "$IS"

echo "==> 3/4 sprite üretiliyor"
mkdir -p "$VENDOR/spinner"
python3 "$KOK/deploy/scripts/ikon-uret.py" \
  --kaynak "$TABLER_SRC" --cikti "$VENDOR" --sadece-outline

# Çekirdek set: public head'de kullanılan az sayıda ikon. Tam sprite 233 KB;
# public'te 4 ikon için indirmek anlamsız.
python3 - "$TABLER_SRC" "$VENDOR" <<'PY'
import io, os, re, sys
kaynak, hedef_dir = sys.argv[1], sys.argv[2]
outline = os.path.join(kaynak, 'icons', 'outline')
adlar = ['home', 'moon', 'sun', 'search', 'external-link', 'rss', 'share-3', 'mail']
parca = ['<svg xmlns="http://www.w3.org/2000/svg" style="display:none" aria-hidden="true">',
         '<!-- theblock çekirdek ikon seti (public). Tabler Icons, MIT. '
         'Tamamı: vendor/tabler-icons.svg (panel). Üretici: ikon-vendor.sh -->']
n = 0
for a in adlar:
    p = os.path.join(outline, a + '.svg')
    if not os.path.exists(p):
        print('    atlandı (tabler\'de yok):', a); continue
    s = io.open(p, encoding='utf-8').read()
    i = s.find('>', s.find('<svg')); j = s.rfind('</svg>')
    ic = re.sub(r'\s*\n\s*', ' ', s[i+1:j]).strip()
    parca.append('<symbol id="ti-%s" viewBox="0 0 24 24" fill="none" stroke="currentColor" '
                 'stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</symbol>' % (a, ic))
    n += 1
parca.append('</svg>')
hedef = os.path.join(hedef_dir, 'ikon-cekirdek.svg')
io.open(hedef, 'w', encoding='utf-8').write('\n'.join(parca) + '\n')
print('    çekirdek: %d ikon · %.1f KB' % (n, os.path.getsize(hedef)/1024))
PY

echo "==> 4/4 animasyonlu spinner'lar + lisans"
python3 - "$IS/package/svg" "$VENDOR/spinner" <<'PY'
import glob, io, os, re, sys
kaynak, hedef = sys.argv[1], sys.argv[2]
# Küratörlü: yükleniyor/kaydetme/bekleme durumlarında kullanılanlar.
adlar = ['ring-resize', '90-ring', '180-ring', '270-ring', '3-dots-bounce',
         '3-dots-fade', 'bars-scale', 'pulse-ring', 'pulse-2', 'eclipse', 'dot-revolve']
n = 0
for a in adlar:
    p = os.path.join(kaynak, a + '.svg')
    if not os.path.exists(p):
        print('    atlandı:', a); continue
    s = io.open(p, encoding='utf-8').read()
    # Sabit siyahı temaya bağla.
    s = s.replace('stroke="#000"', 'stroke="currentColor"').replace('fill="#000"', 'fill="currentColor"')
    # Bazı spinner'lar renk özniteliği taşımıyor (varsayılan siyah dolgu).
    # Kök <svg> etiketinde ne fill ne stroke varsa fill="currentColor" eklenir.
    kok_son = s.find('>', s.find('<svg'))
    kok = s[:kok_son]
    if 'fill=' not in kok and 'stroke=' not in kok:
        s = kok + ' fill="currentColor"' + s[kok_son:]
    io.open(os.path.join(hedef, a + '.svg'), 'w', encoding='utf-8').write(s)
    n += 1
print('    spinner: %d dosya' % n)
PY

{
  echo "theblock ikon kütüphanesi — lisanslar"
  echo "====================================="
  echo
  echo "Tabler Icons (vendor/tabler-icons.svg, vendor/ikon-cekirdek.svg)"
  echo "  https://github.com/tabler/tabler-icons  · MIT"
  echo "  Copyright (c) 2020-2026 Paweł Kuna"
  echo
  sed 's/^/  /' "$TABLER_SRC/LICENSE"
  echo
  echo "SVG Spinners (vendor/spinner/*.svg)"
  echo "  https://github.com/n3r4zzurr0/svg-spinners  · MIT"
  echo "  Copyright (c) Utkarsh Verma"
  echo
  sed 's/^/  /' "$IS/package/LICENSE"
  echo
  echo "Her iki paket de MIT lisanslıdır: ticari kullanım, değiştirme ve kendi"
  echo "sunucumuzdan dağıtma serbesttir; telif bildirimi bu dosyada korunur."
  echo "Çalışma anında hiçbir dış servise istek gitmez — tüm dosyalar yereldir."
} > "$VENDOR/IKON-LISANS.txt"

echo
echo "==> bitti"
ls -la "$VENDOR/tabler-icons.svg" "$VENDOR/ikon-cekirdek.svg" "$VENDOR/ikonlar.txt" \
  | awk '{printf "  %-22s %7.1f KB\n", $9, $5/1024}'
echo "  spinner: $(ls "$VENDOR/spinner" | wc -l) dosya"
echo
echo "Doğrulama:  cd go && go test ./internal/httpapi/ -run Ikon"
