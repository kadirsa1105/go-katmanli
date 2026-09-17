# __APP__

Katmanlı Go servisi. Yerleşim:

```
cmd/__APP__/            giriş noktası, bağımlılıkların kurulduğu tek yer
internal/config/        ayarlar (.env + __PREFIX___* ortam değişkenleri)
internal/domain/        modeller ve ortak hatalar (hiçbir katmana bağımlı değil)
internal/repository/    SQL — bağlantı, migration, depolar
internal/service/       iş kuralları
internal/handler/       HTTP (net/http)
deploy/                 systemd birimi
```

Bağımlılık yönü tek taraflı: `handler → service → repository → domain`.

## Çalıştırma

```bash
cp .env.example .env     # ayarları düzenle
make run                 # derler, migration'ları uygular, dinlemeye başlar
curl localhost:8080/healthz
```

## Derleme

```bash
make build                          # bin/__APP__
make release                        # dist/ altında tüm platformlar
make release PLATFORMS="linux/arm64"
```

## Yeni özellik eklerken

1. `internal/repository/migrations/<lehçe>/NNNN_ad.sql` — şema
2. `internal/domain/<ad>.go` — model
3. `internal/repository/<ad>.go` — SQL; `repos.go`'ya kaydet
4. `internal/service/<ad>.go` — iş kuralı; `service.go`'ya kaydet
5. `internal/handler/<ad>.go` — endpoint'ler; `server.go` `routes()`'a ekle
