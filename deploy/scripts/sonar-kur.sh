#!/bin/bash
# SonarQube'u ayaga kaldir (yoksa) ve tarama icin hazirla.
#
# NEDEN BETIK: kurulum bir kez yapilir ama port, ulimit, parola ve token
# ayrintilari unutulur. Betik idempotenttir: tekrar kosarsa mevcut kurulumu
# kullanir, bozmaz.
#
# Kullanim:  bash deploy/scripts/sonar-kur.sh
set -uo pipefail

PORT="${SONAR_PORT:-9100}"
KONTEYNER=sonarqube
VERI="$HOME/sonarqube"
TOKEN_DOSYA="$HOME/.sonar-token"
PAROLA_DOSYA="$HOME/.sonar-admin-pass"
VARSAYILAN_PAROLA='Bitcosar2026!Guclu'

# ---- 1) konteyner -------------------------------------------------------
if docker ps --format '{{.Names}}' | grep -qx "$KONTEYNER"; then
  echo "==> SonarQube zaten calisiyor (:$PORT)"
else
  if docker ps -a --format '{{.Names}}' | grep -qx "$KONTEYNER"; then
    echo "==> mevcut konteyner baslatiliyor"
    docker start "$KONTEYNER" >/dev/null
  else
    echo "==> SonarQube kuruluyor (:$PORT, veri: $VERI)"
    mkdir -p "$VERI/data" "$VERI/extensions" "$VERI/logs"
    # 9000 portu bu makinede arastam-dev tarafindan kullaniliyor -> 9100.
    docker run -d --name "$KONTEYNER" \
      -p "${PORT}:9000" \
      -v "$VERI/data:/opt/sonarqube/data" \
      -v "$VERI/extensions:/opt/sonarqube/extensions" \
      -v "$VERI/logs:/opt/sonarqube/logs" \
      --ulimit nofile=65536:65536 \
      --ulimit nproc=4096:4096 \
      sonarqube:community >/dev/null
  fi
fi

# ---- 2) hazir olmasini bekle -------------------------------------------
echo "==> hazir olmasi bekleniyor (en fazla 5 dk)"
for i in $(seq 1 60); do
  if curl -fsS -o /dev/null "http://127.0.0.1:$PORT/api/system/status" 2>/dev/null; then
    D=$(curl -fsS "http://127.0.0.1:$PORT/api/system/status" | sed -n 's/.*"status":"\([A-Z]*\)".*/\1/p')
    if [ "$D" = "UP" ]; then echo "==> hazir"; break; fi
  fi
  sleep 5
done

# ---- 3) parola + token (yalniz ilk kurulumda) ---------------------------
if [ -s "$TOKEN_DOSYA" ]; then
  echo "==> token mevcut ($TOKEN_DOSYA)"
else
  echo "==> ilk kurulum: admin parolasi + token"
  if curl -fsS -o /dev/null -u admin:admin "http://127.0.0.1:$PORT/api/system/status" 2>/dev/null; then
    curl -fsS -u admin:admin -X POST "http://127.0.0.1:$PORT/api/users/change_password" \
      -d "login=admin" \
      --data-urlencode "previousPassword=admin" \
      --data-urlencode "password=$VARSAYILAN_PAROLA" >/dev/null
    echo "$VARSAYILAN_PAROLA" > "$PAROLA_DOSYA"
    chmod 600 "$PAROLA_DOSYA"
  fi
  P=$(cat "$PAROLA_DOSYA" 2>/dev/null || echo "$VARSAYILAN_PAROLA")
  T=$(curl -fsS -u "admin:$P" -X POST "http://127.0.0.1:$PORT/api/user_tokens/generate" \
        -d "name=bitcosar-cli" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
  if [ -z "$T" ]; then
    curl -fsS -u "admin:$P" -X POST "http://127.0.0.1:$PORT/api/user_tokens/revoke" -d "name=bitcosar-cli" >/dev/null
    T=$(curl -fsS -u "admin:$P" -X POST "http://127.0.0.1:$PORT/api/user_tokens/generate" \
          -d "name=bitcosar-cli" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
  fi
  if [ -z "$T" ]; then echo "HATA: token uretilemedi" >&2; exit 1; fi
  printf '%s' "$T" > "$TOKEN_DOSYA"
  chmod 600 "$TOKEN_DOSYA"
  echo "==> token yazildi ($TOKEN_DOSYA)"
fi

# ---- 4) proje ----------------------------------------------------------
P=$(cat "$PAROLA_DOSYA" 2>/dev/null || echo "$VARSAYILAN_PAROLA")
curl -fsS -u "admin:$P" -X POST "http://127.0.0.1:$PORT/api/projects/create" \
  -d "project=bitcosar" -d "name=bitcosar" >/dev/null 2>&1 || true

echo
echo "SonarQube: http://127.0.0.1:$PORT/dashboard?id=bitcosar"
echo "Simdi:     bash deploy/scripts/sonar-tara.sh"
