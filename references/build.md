# Derleme, dağıtım, platformlar

## Makefile hedefleri

| Hedef | Ne yapar |
|---|---|
| `make build` | `bin/<app>` — yerel platform, `CGO_ENABLED=0`, `-trimpath`, `-s -w`, sürüm gömülü |
| `make run` | build + `serve` |
| `make migrate` | build + yalnızca migration |
| `make test` / `vet` / `fmt` / `tidy` | kalite |
| `make release` | `dist/<app>_<os>_<arch>[.exe]` — linux/amd64, linux/arm64, windows/amd64, darwin/amd64, darwin/arm64 |
| `make release PLATFORMS="linux/amd64"` | tek platform |
| `make clean` | bin/ ve dist/ sil |

Sürüm `git describe --tags --always --dirty`'den gelir; `VERSION=1.2.0 make release`
ile ezilir. İkili `<app> version` ile gösterir.

CGO kapalı olduğu için cross-compile için C derleyici gerekmez. Bu, SQLite sürücüsü
olarak `modernc.org/sqlite` seçilmesinin sebebi; `mattn/go-sqlite3` CGO ister ve
Windows/ARM hedeflerinde derleme zinciri kurdurur. `modernc` ekleme.

## systemd (Linux sunucu)

İskelet `deploy/systemd/<app>.service` içerir. Kurulum:

```bash
sudo useradd -r -s /sbin/nologin <app>
sudo mkdir -p /opt/<app>/data && sudo chown -R <app>: /opt/<app>
sudo cp dist/<app>_linux_amd64 /opt/<app>/<app>
sudo cp .env /opt/<app>/.env            # üretim değerleriyle
sudo cp deploy/systemd/<app>.service /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable --now <app>
journalctl -u <app> -f
```

Birim dosyası `ProtectSystem=strict` kullanır; yazma yalnızca `/opt/<app>/data`.
SQLite dosyası ya da başka yazılabilir yol gerekiyorsa `ReadWritePaths`'e ekle.

## Docker

`Dockerfile` çok aşamalı: `golang:1.26-alpine` derler, `alpine` çalıştırır, root dışı
kullanıcı. Konteyner içinde `LISTEN=0.0.0.0:8080` (ENV ile ayarlı); dışarıdan
ayarlar `-e <PREFIX>_DATABASE_URL=...` ya da `--env-file .env` ile.

```bash
docker build --build-arg VERSION=$(git describe --tags --always) -t <app> .
docker run --rm -p 8080:8080 --env-file .env -v $PWD/data:/app/data <app>
```

## Windows

`make release PLATFORMS="windows/amd64"` → `dist/<app>_windows_amd64.exe`.
`.env` dosyası ikilinin yanında ya da `-env C:\yol\.env`. SQLite yolu için
`sqlite://C:/veri/app.db` (ileri eğik çizgi).

## macOS

`darwin/arm64` (Apple Silicon) ve `darwin/amd64`. İmzasız ikili Gatekeeper'a takılırsa
`xattr -d com.apple.quarantine <ikili>`.
