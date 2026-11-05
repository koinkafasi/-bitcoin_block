#!/bin/bash
# theblock-backup.sh — günlük yedekleme (GAZETE-PLAN.md §7.1)
#
# 1) PostgreSQL tam dump (pg_dump, container içinden)
# 2) Depolama ağacı (media + gazete çıktıları) kopyası
# 3) Eski yedeklerin temizliği (varsayılan 30 gün)
#
# Kullanım: theblock-backup.sh [HEDEF_DIZIN]   (varsayılan: /home/alper/theblock-backups)
set -euo pipefail

BACKUP_ROOT="${1:-/home/alper/theblock-backups}"
STAMP="$(date +%Y%m%d-%H%M%S)"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
PG_CONTAINER="${PG_CONTAINER:-theblock-postgres-1}"
PGUSER="${PGUSER:-theblock}"
PGDB="${PGDB:-theblock}"
STORAGE_DIR="${STORAGE_DIR:-/home/alper/theblock/storage}"

DEST="$BACKUP_ROOT/$STAMP"
mkdir -p "$DEST"

echo "[yedek] hedef: $DEST"

# 1) PostgreSQL
if docker ps --format '{{.Names}}' | grep -q "^$PG_CONTAINER$"; then
  docker exec "$PG_CONTAINER" pg_dump -U "$PGUSER" -d "$PGDB" \
    --format=custom --compress=6 > "$DEST/db.dump"
  echo "[yedek] veritabanı: $(du -h "$DEST/db.dump" | cut -f1)"
else
  echo "[yedek] UYARI: $PG_CONTAINER bulunamadı, DB yedeği atlandı" >&2
fi

# 2) Depolama (media + gazete) — artımlı rsync
if [ -d "$STORAGE_DIR" ]; then
  rsync -a --delete "$STORAGE_DIR/" "$DEST/storage/"
  echo "[yedek] depolama: $(du -sh "$DEST/storage" | cut -f1)"
fi

# 3) Eski yedekleri temizle
find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -mtime "+$RETENTION_DAYS" -exec rm -rf {} +
echo "[yedek] tamamlandı ($RETENTION_DAYS günden eskiler temizlendi)"