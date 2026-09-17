#!/usr/bin/env bash
# Sıradaki numarayla boş migration dosyası açar (her lehçe dizini için bir tane).
#
#   new_migration.sh <ad>          örn: new_migration.sh notes
#
# Proje kökünden (go.mod'un olduğu yerden) ya da alt dizinlerinden çalıştırılır.
set -euo pipefail

if [ $# -ne 1 ]; then
    sed -n '2,6p' "$0"; exit 1
fi
NAME=$(echo "$1" | tr 'A-Z -' 'a-z__' | tr -cd 'a-z0-9_')

ROOT=$PWD
while [ ! -f "$ROOT/go.mod" ]; do
    ROOT=$(dirname "$ROOT")
    [ "$ROOT" = "/" ] && { echo "hata: go.mod bulunamadı" >&2; exit 1; }
done
MIG="$ROOT/internal/repository/migrations"
[ -d "$MIG" ] || { echo "hata: $MIG yok — bu proje go-katmanli düzeninde değil" >&2; exit 1; }

# Tüm lehçe dizinlerindeki en büyük numara + 1
LAST=$(find "$MIG" -name '[0-9][0-9][0-9][0-9]_*.sql' -printf '%f\n' | cut -c1-4 | sort -n | tail -1)
NEXT=$(printf '%04d' $((10#${LAST:-0} + 1)))

for d in "$MIG"/*/; do
    dialect=$(basename "$d")
    f="$d${NEXT}_${NAME}.sql"
    printf -- '-- %s (%s)\n' "$NAME" "$dialect" > "$f"
    echo "$f"
done
