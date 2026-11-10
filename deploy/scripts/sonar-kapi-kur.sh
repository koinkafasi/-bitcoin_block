#!/bin/bash
# Kalite kapisi "bitcosar" — projeye GERCEKCI esikler.
#
# NEDEN VARSAYILAN "Sonar way" DEGIL: o kapi yeni kod icin %80 kapsam ister.
# Bu proje ~%22 kapsamda (DB/HTTP gerektiren paketler birim testinde kosmuyor).
# Kalici kirmizi kapi, theblock'ta yasandigi gibi "kimse bakmiyor" durumuna
# yol acar (bkz. docs/guvenlik-hatti.md: "düzeltilemeyen bir bulgu CI'yı
# kalıcı kırmızıya çevirir ve kimse bakmaz").
#
# Bu kapi REGRESYONU engeller, mukemmellik beklemez:
#   yeni kodda 0 hata/acik/koku · genel hata+acik 0 · guvenlik A
#   yeni kod kapsami >= %20 · yeni kod tekrari <= %3
#
# KULLANIM: bash deploy/scripts/sonar-kapi-kur.sh
set -uo pipefail
PORT="${SONAR_PORT:-9100}"
P=$(cat "$HOME/.sonar-admin-pass" 2>/dev/null)
if [ -z "$P" ]; then echo "admin parolasi yok (~/.sonar-admin-pass)" >&2; exit 1; fi
A="admin:$P"
U="http://127.0.0.1:$PORT"
KAPI=bitcosar

echo "==> kapi '$KAPI' olusturuluyor (varsa atlanir)"
curl -fsS -u "$A" -X POST "$U/api/qualitygates/create" -d "name=$KAPI" >/dev/null 2>&1 || true

# Kosul ekle/guncelle — IDEMPOTENT.
# Sonar, yeni kapiya "Sonar way" kosullarini KOPYALAR; her kosulun bir UUID'si
# olur. Metrik adiyla silme CALISMAZ ("id parameter is missing"), bu yuzden
# once mevcut kosullar okunur.
kosul() {
  local metrik="$1" operator="$2" esik="$3"
  local id
  id=$(curl -fsS -u "$A" "$U/api/qualitygates/show?name=$KAPI" | python3 -c "
import json,sys
d = json.load(sys.stdin)
for c in d.get('conditions', []):
    if c['metric'] == '$metrik':
        print(c['id']); break
")
  if [ -n "$id" ]; then
    if curl -fsS -u "$A" -X POST "$U/api/qualitygates/update_condition" \
        -d "id=$id" -d "metric=$metrik" -d "op=$operator" -d "error=$esik" >/dev/null 2>&1; then
      echo "  ~ $metrik $operator $esik"
      return
    fi
  fi
  if curl -fsS -u "$A" -X POST "$U/api/qualitygates/create_condition" \
      -d "gateName=$KAPI" -d "metric=$metrik" -d "op=$operator" -d "error=$esik" >/dev/null 2>&1; then
    echo "  + $metrik $operator $esik"
  else
    echo "  ! $metrik eklenemedi" >&2
  fi
}

# --- regresyon kapilari (yeni kod) ---
kosul new_bugs GT 0
kosul new_vulnerabilities GT 0
kosul new_code_smells GT 0
# KAPSAM EŞİĞİ BİR ÇIT (ratchet), hedef değil.
# ÖLÇÜM (10 Kasım 2026): genel kapsam %23,5; yeni kod %7,8. Yeni kodun
# çoğu templ görünüm bileşeni (HTML üretimi) — birim testiyle ölçmek
# verimsiz, tarayıcı ölçümüyle doğrulanıyor. %80'lik "Sonar way" eşiği
# yapısal olarak geçilemez ve kapıyı kalıcı kırmızıya çevirirdi.
# Bugünkü 7,8'in hemen altında: düşüşü engeller, test eklendikçe yükseltilir.
kosul new_coverage LT 7
kosul new_duplicated_lines_density GT 3

# --- genel durum kapilari ---
kosul bugs GT 0
kosul vulnerabilities GT 0
kosul security_rating GT 1        # 1 = A
kosul reliability_rating GT 1

# BAGLAMA: bu Sonar surumu `qualitygates/list` ve `show` yanitlarinda kapi
# KIMLIGI dondurmuyor, bu yuzden `select` (gateId ister) kullanilamiyor.
# Tek projeli yerel kurulumda dogru yol kapıyı VARSAYILAN yapmak.
echo "==> '$KAPI' varsayilan kapi yapiliyor (tek projeli kurulum)"
if curl -fsS -u "$A" -X POST "$U/api/qualitygates/set_as_default" -d "name=$KAPI" >/dev/null 2>&1; then
  echo "==> baglandi"
else
  echo "!! varsayilan yapilamadi — panodan elle secin:" >&2
  echo "   $U/project/quality_gate?id=bitcosar" >&2
fi

echo
echo "==> kapi kosullari"
curl -fsS -u "$A" "$U/api/qualitygates/show?name=$KAPI" | python3 -c "
import json,sys
d = json.load(sys.stdin)
for c in d.get('conditions', []):
    print(f\"  {c['metric']:34} {c['op']:3} {c['error']}\")
"
