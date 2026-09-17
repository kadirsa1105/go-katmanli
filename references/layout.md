# Dizin yerleşimi ve sorumluluklar

```
<app>/
├── cmd/<app>/main.go              flag'ler, komutlar (serve|migrate|version), bağımlılık kurulumu
├── internal/
│   ├── config/config.go           Config struct, Load(.env), Prefix sabiti — tek os.Getenv noktası
│   ├── domain/
│   │   ├── errors.go              ErrNotFound, ErrConflict, ValidationError, Invalid()
│   │   └── <varlık>.go            struct, <Varlık>Input, Validate()
│   ├── repository/
│   │   ├── db.go                  Open(url), Dialect, Rebind, DSN çözümleme
│   │   ├── migrate.go             gömülü migration çalıştırıcı
│   │   ├── repos.go               Repos{...} — tüm depoların kaydı, Ping
│   │   ├── driver_<lehçe>.go      sürücü blank import
│   │   ├── testdb_test.go         test için DB helper
│   │   ├── <varlık>.go            <Varlık>Repo — SQL
│   │   └── migrations/<lehçe>/NNNN_ad.sql
│   ├── service/
│   │   ├── service.go             Services{...} — tüm servislerin kaydı, Health
│   │   └── <varlık>.go            <Varlıklar> — iş kuralı
│   └── handler/
│       ├── server.go              Server, routes(), Run() (graceful shutdown), healthz
│       ├── respond.go             writeJSON, writeError (hata→HTTP kodu), decodeJSON
│       ├── middleware.go          withLog, withRecover
│       └── <varlık>.go            handle<Varlık>* + pathID gibi yardımcılar
├── deploy/systemd/<app>.service
├── Makefile · Dockerfile · .env.example · .gitignore · README.md
└── data/                          SQLite dosyası vb. (git dışı)
```

## Neden `internal/`?

Go, `internal/` altındaki paketlerin modül dışından import edilmesini engeller.
Böylece repository/service dışarıya API olmaz; yalnızca `cmd/` giriş noktasıdır.
Dışarıya kütüphane sunmak gerekirse `pkg/` açılır — varsayılan olarak açma.

## Yeni bir alt sistem (HTTP olmayan)

Cron benzeri arka plan işi, kuyruk tüketicisi vb.: `internal/worker/` paketi,
`service`'i çağırır, `main.go`'da `serve` içinde goroutine olarak başlatılır ve
`ctx` iptalinde durur. Repository'ye doğrudan gitmez.

## Birden çok ikili

İkinci bir komut satırı aracı gerekiyorsa `cmd/<araç>/main.go`; aynı
`config.Load` + `repository.Open` + `service.New` zincirini kullanır. Makefile'da
`APP` değişkenini çoğaltmak yerine `go build ./cmd/<araç>` satırı ekle.

## Ne nereye gitmez

- `*http.Request`, `http.ResponseWriter` → yalnızca handler
- `database/sql`, SQL metni → yalnızca repository
- `os.Getenv` → yalnızca config
- `log.Fatal`/`os.Exit` → yalnızca `main()`; diğer her yer `error` döner
