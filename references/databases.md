# Veritabanı lehçeleri

Tek `DATABASE_URL`, tek `repository.Open`, tek `Rebind`. Repository kodu ortak;
farklar migration SQL'inde ve birkaç noktada toplanır. Bu dosyayı migration ya da
alışılmadık bir sorgu yazarken oku.

## URL biçimleri ve sürücüler

| Lehçe | URL | Sürücü paketi | `driver_*.go` |
|---|---|---|---|
| postgres | `postgres://u:p@host:5432/db?sslmode=disable` | `github.com/jackc/pgx/v5/stdlib` (adı `pgx`) | `driver_postgres.go` |
| mysql | `mysql://u:p@host:3306/db` | `github.com/go-sql-driver/mysql` | `driver_mysql.go` |
| sqlite | `sqlite://./data/app.db` · `sqlite:///mutlak/yol.db` · `sqlite://:memory:` | `modernc.org/sqlite` (saf Go, CGO yok) | `driver_sqlite.go` |

- MySQL: `Open` DSN'yi `u:p@tcp(host:3306)/db` biçimine çevirir, `parseTime=true`
  (DATETIME → time.Time) ve `multiStatements=true` (çok ifadeli migration) ekler.
  Unix soketi: `mysql://u:p@unix(/var/lib/mysql/mysql.sock)/db`.
- SQLite: `Open` üst dizini oluşturur; `busy_timeout(5000)`, `journal_mode(WAL)`,
  `foreign_keys(1)` PRAGMA'larını ekler; `MaxOpenConns=1` (tek yazıcı).
  Kendi PRAGMA'nı vermek istersen `?_pragma=...` yaz, varsayılanlar eklenmez.
- Sürücü değiştirmek/eklemek: `assets/db/<lehçe>/` altındaki `driver_<lehçe>.go` ve
  `migrations/<lehçe>/` dizinini projeye kopyala, `go get` yap. Aynı ikili, URL'ye göre
  hangisi verilirse ona bağlanır.

## Rebind

Sorgular `?` ile yazılır; `r.db.Rebind(q)` Postgres'te `$1,$2…` üretir, diğerlerinde
dokunmaz. Sorgu içindeki string sabitlerinde `?` kullanma (o da çevrilir).
Aynı parametreyi iki kez kullanman gerekiyorsa iki kez geç.

## ID üretimi ve INSERT

| | Sütun | Geri alma |
|---|---|---|
| postgres | `BIGSERIAL PRIMARY KEY` (ya da `GENERATED ALWAYS AS IDENTITY`) | `INSERT … RETURNING id` + `QueryRow.Scan` |
| mysql | `BIGINT AUTO_INCREMENT PRIMARY KEY` | `res.LastInsertId()` |
| sqlite | `INTEGER PRIMARY KEY AUTOINCREMENT` | `res.LastInsertId()` |

Repository'de tek dal: `if r.db.Dialect == Postgres { RETURNING } else { LastInsertId }`
(bkz. feature-example.md `Create`). Postgres'te `LastInsertId` çalışmaz.

## Tarih/saat

| | Sütun | Not |
|---|---|---|
| postgres | `TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP` | UTC sakla |
| mysql | `DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP` | `ON UPDATE CURRENT_TIMESTAMP` eklenebilir |
| sqlite | `DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP` | **`TEXT` yazma** — modernc sürücüsü yalnızca `DATE/DATETIME/TIMESTAMP` bildirilen sütunları `time.Time`'a çevirir; `TEXT` bildirirsen Scan hatası alırsın. |

`UPDATE … SET updated_at = CURRENT_TIMESTAMP` üçünde de çalışır.

## Boolean, JSON, metin

| | bool | JSON | metin |
|---|---|---|---|
| postgres | `BOOLEAN` | `JSONB` | `TEXT` |
| mysql | `TINYINT(1)` / `BOOLEAN` | `JSON` | `VARCHAR(n)` (indeksli) / `TEXT` |
| sqlite | `INTEGER` (0/1) | `TEXT` | `TEXT` |

JSON sütunlarını Go'da `[]byte`/`string` olarak tara, `json.Unmarshal` ile çöz.

## Upsert / çakışma

- postgres: `INSERT … ON CONFLICT (kolon) DO UPDATE SET …`
- mysql: `INSERT … ON DUPLICATE KEY UPDATE …`
- sqlite: `INSERT … ON CONFLICT (kolon) DO UPDATE SET …` (3.24+; modernc güncel)

Tekil kısıt ihlali (UNIQUE / PRIMARY KEY) → `repository.IsUniqueViolation(err)`
(db.go'da hazır, üç sürücüde test edildi). Repository'de:

```go
if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
	if IsUniqueViolation(err) {
		return nil, domain.ErrConflict
	}
	return nil, fmt.Errorf("orders insert: %w", err)
}
```

Önce `SELECT` ile "var mı" bakmak yarış durumunda yine de ihlale düşer; indeks
son güvencedir, o yüzden `IsUniqueViolation` her zaman olmalı. `pgconn.PgError`
gibi sürücüye özel tipleri import etme — projeyi tek sürücüye bağlar.

## Transaction

`tx, err := r.db.BeginTx(ctx, nil)`; `defer tx.Rollback()`; sonunda `tx.Commit()`.
Bir service metodu birden çok repo çağrısını tek transaction'da istiyorsa
repository'ye `WithTx(ctx, func(tx *sql.Tx) error)` gibi tek bir yardımcı ekle;
service `*sql.Tx` görmemeli. MySQL'de DDL transaction dışıdır — migration'da yarım
kalan dosya elle temizlenir.

## Migration disiplini

- Dosya adı sürümdür (`0007_orders.sql`). Uygulandıktan sonra ad ve içerik değişmez.
- Geri alma dosyası yok; gerekirse yeni bir ileri migration yaz.
- Yalnızca `--` yorum içeren dosya "uygulandı" sayılır, Exec edilmez.
- Yeni dosya: `~/.claude/skills/go-katmanli/scripts/new_migration.sh <ad>`.
