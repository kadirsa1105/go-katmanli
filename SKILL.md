---
name: go-katmanli
description: Katmanlı mimaride (handler → service → repository → domain) Go projesi kurar ve mevcut Go projelerine bu düzende özellik ekler. Sabit config (.env + UYGULAMA_* ortam değişkenleri), tek bir DATABASE_URL ile Postgres/MySQL/SQLite, gömülü SQL migration'lar, stdlib net/http, Makefile ile çok platformlu derleme (linux/windows/darwin, amd64/arm64) ve systemd/Docker dağıtımı. Kullanıcı Go ile yeni bir servis/API/daemon/araç yazmak istediğinde; "go projesi başlat", "iskelet çıkar", "katmanlı", "repository/service/handler" dediğinde; bir Go projesine tablo, kolon/alan, endpoint ya da migration eklemek istediğinde ("migration aç/ekle", "yeni tablo", "kolon ekle", "endpoint ekle"); Go projesini cross-compile/Makefile/systemd ile derleyip dağıtmak istediğinde; ya da internal/repository + internal/service + internal/handler düzenindeki bir projede çalışırken — "skill" ya da "katmanlı" demese bile — bu skill'i kullan. Tek dosyalık script'ler, Go dışı diller ve salt hata ayıklama/soru-cevap için gerekmez.
---

# go-katmanli — Katmanlı Go projeleri

Bu skill iki iş yapar: (A) sıfırdan proje iskeleti kurar, (B) var olan projeye
katmanlara uygun özellik ekler. Amaç, her projede aynı yerde aynı şeyi bulmak:
ayar `internal/config`'de, SQL `internal/repository`'de, iş kuralı
`internal/service`'te, HTTP `internal/handler`'da. Yeni birinin (ya da altı ay
sonra senin) projeyi açtığında yolunu bulabilmesi bu tutarlılığa bağlı.

## A) Yeni proje

```bash
~/.claude/skills/go-katmanli/scripts/new_project.sh <hedef-dizin> <modül> <postgres|mysql|sqlite>
# örn: ~/.claude/skills/go-katmanli/scripts/new_project.sh ./stokapp github.com/kadir/stokapp sqlite
```

Script iskeleti kopyalar, yer tutucuları doldurur, sürücüyü `go get` eder,
`go build ./... && go vet ./...` çalıştırır. Elle dosya yazma — script
deterministik, her proje birebir aynı başlar. Kullanıcı modül yolu ya da
veritabanı söylemediyse sor; küçük tek makine araçları için sqlite, çok
kullanıcılı servisler için postgres öner. Uygulama adı = hedef dizin adı
(küçük harf, `-`/`_` serbest); env öneki bunun büyük harfli hali (`stok-app` → `STOK_APP_`).

Sonrasında kullanıcının istediği alanları (tablolar, endpoint'ler) B bölümündeki
sırayla ekle. Bitirince `make build` ve `go test ./...` geçtiğini göster.

## B) Mevcut projeye özellik ekleme

Bir varlık (örn. `Order`) eklemek her zaman aynı beş dokunuş:

| # | Dosya | İçerik | Kayıt yeri |
|---|-------|--------|------------|
| 1 | `internal/repository/migrations/<lehçe>/NNNN_orders.sql` | tablo/indeks | `scripts/new_migration.sh orders` numarayı verir |
| 2 | `internal/domain/order.go` | struct + `Input.Validate()` | — |
| 3 | `internal/repository/order.go` | `OrderRepo` — yalnızca SQL | `repos.go` → `Repos.Orders` |
| 4 | `internal/service/order.go` | `Orders` — iş kuralı | `service.go` → `Services.Orders` |
| 5 | `internal/handler/order.go` | `handleOrder*` | `server.go` → `routes()` |

Dizin sorumlulukları ve "ne nereye gitmez" listesi: **references/layout.md**.
Tam çalışan örnek (Note varlığı, üç veritabanında test edildi): **references/feature-example.md**.
Yeni bir varlık yazmadan önce oku; adlandırma, hata akışı ve Rebind kullanımı oradaki gibi olsun.

Projede bu düzen yoksa (ör. eski bir proje) önce kullanıcıya sor: düzene mi
taşıyalım, yoksa mevcut düzenine mi uyalım? Sormadan büyük yeniden yapılandırma yapma.

## Katman kuralları

Bağımlılık yönü tek taraflı: `cmd → handler → service → repository → domain`.

- **domain**: modeller, `Input` tipleri ve `Validate()`, ortak hatalar
  (`ErrNotFound`, `ErrConflict`, `ValidationError`). Hiçbir iç pakete import yok.
- **repository**: SQL yalnızca burada. `domain` tiplerini döner; `sql.ErrNoRows` →
  `domain.ErrNotFound`. Sorgular `?` ile yazılır ve `r.db.Rebind(...)`'den geçer.
- **service**: doğrulama, iş kuralı, loglama. HTTP'yi bilmez (`*http.Request` yok),
  SQL yazmaz. Bugün HTTP'den, yarın CLI/cron'dan çağrılabilmeli.
- **handler**: istek çöz → service çağır → `writeJSON`/`s.writeError`. Hata→HTTP kodu
  eşlemesi `respond.go`'da; handler'da `http.StatusNotFound` gibi kodları elle seçme.
- **cmd/main.go**: bağımlılıkları kuran tek yer. Global değişken yok.

Handler'ın repository'ye, service'in handler'a uzanması kural ihlalidir; ihtiyaç
duyuluyorsa eksik olan bir service metodudur.

## Sabit noktalar (her projede aynı)

- Ayarlar: `.env` (git dışı) + `<PREFIX>_*` ortam değişkenleri, `internal/config/config.go`.
  Kodda başka yerde `os.Getenv` kullanma; yeni ayar = Config'e alan + `getenv` satırı + `.env.example`.
- `<PREFIX>_DATABASE_URL` şeması sürücüyü seçer:
  `postgres://u:p@host:5432/db?sslmode=disable` · `mysql://u:p@host:3306/db` · `sqlite://./data/app.db`
  Tek giriş `repository.Open`. Sürücü `internal/repository/driver_<lehçe>.go` içinde blank import.
- Migration: `internal/repository/migrations/<lehçe>/NNNN_ad.sql`, ikiliye gömülü,
  `serve` açılışında ve `migrate` komutuyla uygulanır. Dosya adı = sürüm; adı sonradan değiştirme.
  Uygulanmış bir dosyayı düzenleme — yeni dosya aç.
- Komutlar: `<app> serve` (varsayılan) · `<app> migrate` · `<app> version` · `-env <dosya>`.
- Derleme: `make build` / `make release` (linux, windows, darwin × amd64, arm64; `CGO_ENABLED=0`).
  Ayrıntı ve deploy: **references/build.md**.
- Container dağıtımı (Coolify/Traefik): port, ağ, kalıcı volume, healthcheck, Shared Variables ile
  ortak veritabanı bilgisi, Cloudflare/TLS tuzakları → **references/coolify.md**. İskeletteki
  `COOLIFY.md` projeye özgü kurulum notudur; doldur, silme.

Lehçe farkları (ID üretimi, tarih tipleri, RETURNING, SQLite tuzakları): **references/databases.md** — migration
ya da repository yazarken ilgili bölümü oku; özellikle SQLite'ta tarih sütunu `DATETIME` olmalı, yoksa `time.Time` taranmaz.

## Kod stili

- Tanımlayıcılar İngilizce, yorumlar ve kullanıcıya dönen mesajlar Türkçe (`ErrNotFound = "kayıt bulunamadı"`).
- Yorumlar "ne"yi değil "neden"i anlatır (ör. varsayılan `127.0.0.1` — yanlışlıkla dışarı açılmasın).
- Hatalar `fmt.Errorf("orders insert: %w", err)` ile sarılır; `errors.Is/As` ile ayrıştırılır.
- Her DB/servis metodu `ctx context.Context` alır ve ilk parametredir.
- Log: `log/slog`, anahtar-değer; `fmt.Println` yok.
- Bağımlılık eklemeden önce stdlib'de var mı bak; router/ORM/config kütüphanesi ekleme.
- Tekil kısıt ihlali → `repository.IsUniqueViolation(err)` → `domain.ErrConflict` (409). Sürücüye özel hata tipi import etme.
- Testler: `internal/repository/testdb_test.go` helper'ı `<PREFIX>_TEST_DATABASE_URL` ile (yoksa `sqlite://:memory:` ile)
  gerçek DB açar ve migration'ları uygular; repository testleri buna dayanır. Postgres/MySQL projesinde SQLite sürücüsü
  yoksa test **skip** eder — bu beklenen davranış. Testleri koşturmak için projeye SQLite sürücüsü ya da ikinci bir
  `migrations/sqlite` dizini ekleme: her migration'ı iki kez yazmak demektir. Bunun yerine test komutunu
  `<PREFIX>_TEST_DATABASE_URL=postgres://... go test ./...` biçiminde ver. Service/handler testinde de aynı helper'la
  gerçek repo kullan; interface + sahte depo yazma — domain doğrulaması gibi DB'siz mantık zaten ayrı test edilebilir.

## Bitirmeden

Yeni import eklediysen `go mod tidy` (go.sum eksikliği build'i düşürür). `gofmt -l .` boş,
`go vet ./...` ve `go test ./...` temiz, `make build` geçiyor.
Yeni ayar eklediysen `.env.example` güncel — panelden de girileceği için `COOLIFY.md`
tablosuna aynı satırı ekle. Yeni endpoint eklediysen README'deki listeye bir satır ekle.
