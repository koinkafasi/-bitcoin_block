#!/bin/bash
# SonarQube taramasi — yerdel SonarQube (Docker, :9100) uzerinde.
#
# Kullanim:
#   bash deploy/scripts/sonar-tara.sh
#
# ONKOSUL:
#   1) SonarQube ayakta:  docker ps --filter name=sonarqube
#      Yoksa: docker run -d --name sonarqube -p 9100:9000 ... sonarqube:community
#   2) Token:  ~/.sonar-token  (chmod 600)
#
# KAPSAM: once `kapsam-uret.sh` kosarsa `coverage.out` taranir. Kosmazsa
# Sonar kapsami %0 gosterir ve kalite kapisi "new_coverage" nedeniyle kirmizi
# kalir — bu yuzden betik kapsami KENDISI uretir.
set -uo pipefail
KOK=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$KOK" || exit 1

TOKEN_DOSYA="$HOME/.sonar-token"
if [ ! -s "$TOKEN_DOSYA" ]; then
  echo "HATA: $TOKEN_DOSYA yok. SonarQube token'i gerekli." >&2
  exit 1
fi
TOKEN=$(cat "$TOKEN_DOSYA")

SONAR_URL="${SONAR_URL:-http://127.0.0.1:9100}"
if ! curl -fsS -o /dev/null "$SONAR_URL/api/system/status" 2>/dev/null; then
  echo "HATA: SonarQube $SONAR_URL yanit vermiyor." >&2
  exit 1
fi

# Kapsam taze olsun: bayat coverage.out yanlis rapor uretir.
bash "$KOK/deploy/scripts/kapsam-uret.sh" || true

echo "==> tarama"
docker run --rm \
  --network host \
  -e SONAR_HOST_URL="$SONAR_URL" \
  -e SONAR_TOKEN="$TOKEN" \
  -v "$KOK:/usr/src" \
  -w /usr/src \
  sonarsource/sonar-scanner-cli:latest \
  -Dsonar.projectBaseDir=/usr/src
