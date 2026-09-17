#!/usr/bin/env bash
# Katmanlı Go proje iskeleti üretir.
#
#   new_project.sh <hedef-dizin> <modül-yolu> <postgres|mysql|sqlite>
#   new_project.sh ./stokapp github.com/kadir/stokapp sqlite
#
# Uygulama adı hedef dizinin son parçasıdır (bin/<ad>, cmd/<ad>).
# Ortam değişkeni öneki uygulama adının büyük harfli halidir (STOKAPP_*).
# Sonunda go mod tidy + go build çalıştırır; ağ yoksa tidy başarısız olur,
# dosyalar yine de yerinde kalır.
set -euo pipefail

if [ $# -ne 3 ]; then
    sed -n '2,10p' "$0"
    exit 1
fi

TARGET=$1
MODULE=$2
DB=$3
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"

case "$DB" in
    postgres|mysql|sqlite) ;;
    *) echo "hata: veritabanı postgres | mysql | sqlite olmalı, verilen: $DB" >&2; exit 1 ;;
esac

APP=$(basename "$TARGET")
if ! [[ "$APP" =~ ^[a-z][a-z0-9_-]*$ ]]; then
    echo "hata: uygulama adı küçük harf/rakam/-/_ olmalı: $APP" >&2; exit 1
fi
PREFIX=$(echo "$APP" | tr 'a-z-' 'A-Z_')

case "$DB" in
    postgres) DATABASE_URL="postgres://$APP:$APP@127.0.0.1:5432/$APP?sslmode=disable" ;;
    mysql)    DATABASE_URL="mysql://$APP:$APP@127.0.0.1:3306/$APP" ;;
    sqlite)   DATABASE_URL="sqlite://./data/$APP.db" ;;
esac

if [ -e "$TARGET" ] && [ -n "$(ls -A "$TARGET" 2>/dev/null)" ]; then
    echo "hata: $TARGET boş değil" >&2; exit 1
fi

mkdir -p "$TARGET"
cp -r "$SKILL_DIR/assets/scaffold/." "$TARGET/"
cp -r "$SKILL_DIR/assets/db/$DB/." "$TARGET/"
mv "$TARGET/cmd/__APP__" "$TARGET/cmd/$APP"
mv "$TARGET/deploy/systemd/__APP__.service" "$TARGET/deploy/systemd/$APP.service"
mkdir -p "$TARGET/data"

# Yer tutucuları doldur. Sıra önemli: __DATABASE_URL__ içinde __APP__ yok ama
# yine de önce en özel olanı değiştiriyoruz.
find "$TARGET" -type f | while read -r f; do
    sed -i \
        -e "s|__DATABASE_URL__|$DATABASE_URL|g" \
        -e "s|__MODULE__|$MODULE|g" \
        -e "s|__PREFIX__|$PREFIX|g" \
        -e "s|__APP__|$APP|g" \
        "$f"
done

cd "$TARGET"
go mod init "$MODULE" >/dev/null 2>&1
case "$DB" in
    postgres) go get github.com/jackc/pgx/v5@latest >/dev/null ;;
    mysql)    go get github.com/go-sql-driver/mysql@latest >/dev/null ;;
    sqlite)   go get modernc.org/sqlite@latest >/dev/null ;;
esac
go mod tidy >/dev/null
gofmt -l -w . >/dev/null
go build ./... && go vet ./...

echo
echo "hazır: $TARGET  (modül $MODULE, veritabanı $DB, önek ${PREFIX}_)"
echo "  cd $TARGET && cp .env.example .env && make run"
