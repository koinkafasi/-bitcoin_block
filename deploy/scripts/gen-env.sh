#!/usr/bin/env bash
# deploy/.env üretir (mevcut ise DOKUNMAZ). Sırlar yalnız bu dosyada.
set -euo pipefail
cd "$(dirname "$0")/.."   # deploy/
ENV_FILE=".env"
if [ -f "$ENV_FILE" ]; then
  echo "deploy/.env zaten var — dokunulmadı."
  exit 0
fi
PW="$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 28)"
sed -e "s|POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${PW}|" \
    -e "s|DATABASE_URL=.*|DATABASE_URL=postgres://theblock:${PW}@127.0.0.1:5433/theblock?sslmode=disable|" \
    .env.example > "$ENV_FILE"
chmod 600 "$ENV_FILE"
echo "deploy/.env oluşturuldu (600)."
