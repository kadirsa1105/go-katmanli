// Package repository veritabanına dokunan tek katmandır. SQL yalnızca burada
// yazılır; service katmanı buradaki metotları çağırır, handler asla doğrudan
// buraya inmez.
//
// Sabit kurallar:
//   - Bağlantı yalnızca Open ile, DATABASE_URL üzerinden açılır.
//   - Sorgular "?" yer tutucusuyla yazılır; db.Rebind lehçeye göre çevirir.
//     Böylece aynı repository kodu Postgres, MySQL ve SQLite'ta çalışır.
//   - Şema değişiklikleri migrations/<lehçe>/NNNN_ad.sql dosyalarındadır.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Dialect desteklenen veritabanı lehçeleri.
type Dialect string

const (
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
	SQLite   Dialect = "sqlite"
)

// DB *sql.DB'yi lehçe bilgisiyle sarar.
type DB struct {
	*sql.DB
	Dialect Dialect
}

// Open DATABASE_URL'yi çözer, sürücüyü seçer, havuzu ayarlar ve ping atar.
//
//	postgres://user:pass@host:5432/db?sslmode=disable
//	mysql://user:pass@host:3306/db            (parseTime ve multiStatements otomatik eklenir)
//	sqlite://./data/app.db  |  sqlite://:memory:
func Open(ctx context.Context, rawURL string) (*DB, error) {
	dialect, driver, dsn, err := parseURL(rawURL)
	if err != nil {
		return nil, err
	}

	sqldb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sürücü %q açılamadı (internal/repository/driver_%s.go var mı?): %w", driver, dialect, err)
	}

	switch dialect {
	case SQLite:
		// SQLite tek yazıcı ister; birden çok bağlantı SQLITE_BUSY üretir.
		sqldb.SetMaxOpenConns(1)
	default:
		sqldb.SetMaxOpenConns(10)
		sqldb.SetMaxIdleConns(5)
		sqldb.SetConnMaxLifetime(time.Hour)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqldb.PingContext(pingCtx); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("veritabanına bağlanılamadı (%s): %w", dialect, err)
	}
	return &DB{DB: sqldb, Dialect: dialect}, nil
}

// Rebind "?" yer tutucularını lehçenin beklediği biçime çevirir.
// Postgres $1,$2… ister; MySQL ve SQLite "?" ile çalışır.
// Not: String sabitlerinin içindeki "?" de çevrilir; sorgularda literal "?" kullanma.
func (d *DB) Rebind(query string) string {
	if d.Dialect != Postgres {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// parseURL şemadan lehçe ve sürücüyü seçer, DSN'yi sürücünün beklediği biçime getirir.
func parseURL(rawURL string) (Dialect, string, string, error) {
	scheme, rest, ok := strings.Cut(rawURL, "://")
	if !ok {
		return "", "", "", fmt.Errorf("DATABASE_URL şeması yok: %q (postgres:// | mysql:// | sqlite:// bekleniyor)", rawURL)
	}
	switch strings.ToLower(scheme) {
	case "postgres", "postgresql":
		return Postgres, "pgx", rawURL, nil
	case "mysql":
		return MySQL, "mysql", mysqlDSN(rest), nil
	case "sqlite", "sqlite3":
		dsn, err := sqliteDSN(rest)
		return SQLite, "sqlite", dsn, err
	default:
		return "", "", "", fmt.Errorf("desteklenmeyen DATABASE_URL şeması: %q", scheme)
	}
}

// mysqlDSN "user:pass@host:3306/db?x=y" biçimini go-sql-driver'ın
// "user:pass@tcp(host:3306)/db?x=y" biçimine çevirir; zaten o biçimdeyse dokunmaz.
// parseTime=true (DATETIME → time.Time) ve multiStatements=true (migration
// dosyaları birden çok ifade içerir) yoksa eklenir.
func mysqlDSN(rest string) string {
	dsn := rest
	if !strings.Contains(rest, "@tcp(") && !strings.Contains(rest, "@unix(") {
		cred, hostPart, hasCred := strings.Cut(rest, "@")
		if !hasCred {
			cred, hostPart = "", rest
		}
		hostPort, dbAndQuery, _ := strings.Cut(hostPart, "/")
		dsn = cred + "@tcp(" + hostPort + ")/" + dbAndQuery
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	if !strings.Contains(dsn, "parseTime=") {
		dsn += sep + "parseTime=true"
		sep = "&"
	}
	if !strings.Contains(dsn, "multiStatements=") {
		dsn += sep + "multiStatements=true"
	}
	return dsn
}

// sqliteDSN dosya yolunu çözer, üst dizini oluşturur ve makul PRAGMA'ları ekler.
func sqliteDSN(rest string) (string, error) {
	path, query, _ := strings.Cut(rest, "?")
	if path == "" {
		return "", fmt.Errorf("sqlite:// sonrası dosya yolu boş")
	}
	if path != ":memory:" {
		if p, err := url.PathUnescape(path); err == nil {
			path = p
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", fmt.Errorf("sqlite dizini: %w", err)
		}
	}
	params := url.Values{}
	if query != "" {
		var err error
		if params, err = url.ParseQuery(query); err != nil {
			return "", fmt.Errorf("sqlite parametreleri: %w", err)
		}
	}
	if len(params["_pragma"]) == 0 {
		params["_pragma"] = []string{"busy_timeout(5000)", "journal_mode(WAL)", "foreign_keys(1)"}
	}
	return "file:" + path + "?" + params.Encode(), nil
}

// IsUniqueViolation tekil kısıt (UNIQUE / PRIMARY KEY) ihlali mi?
// Repository'de `if IsUniqueViolation(err) { return domain.ErrConflict }` diye
// kullanılır; handler bunu 409'a çevirir. Sürücü paketlerini import etmemek
// için (yalnızca seçili sürücü derlenir) hata metnine bakılır — üç sürücünün
// mesajı da sabittir:
//   pgx:     "... (SQLSTATE 23505)"
//   mysql:   "Error 1062 (23000): Duplicate entry ..."
//   sqlite:  "constraint failed: UNIQUE constraint failed: ... (1555|2067)"
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "Error 1062") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}
