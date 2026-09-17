package repository

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

// migrationFS migrations/<lehçe>/NNNN_ad.sql dosyalarını ikiliye gömer;
// dağıtımda ayrı bir dosya taşımak gerekmez.
//
//go:embed all:migrations
var migrationFS embed.FS

// Migrate uygulanmamış .sql dosyalarını ada göre sıralı çalıştırır.
// Her dosya kendi transaction'ında koşar (MySQL DDL'i transaction dışıdır,
// orada yarım kalan dosya elle düzeltilir). Uygulananlar schema_migrations
// tablosuna dosya adıyla yazılır; dosya adı değişirse yeniden koşar — adları değiştirme.
func Migrate(ctx context.Context, db *DB, log *slog.Logger) error {
	if _, err := db.ExecContext(ctx, migrationTableDDL(db.Dialect)); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	dir := "migrations/" + string(db.Dialect)
	entries, err := fs.ReadDir(migrationFS, dir)
	if err != nil {
		return fmt.Errorf("%s dizini yok: bu lehçe için migration yazılmamış", dir)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			continue
		}
		body, err := migrationFS.ReadFile(dir + "/" + name)
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if sqlText := stripComments(string(body)); sqlText != "" {
			if _, err := tx.ExecContext(ctx, sqlText); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %s: %w", name, err)
			}
		}
		if _, err := tx.ExecContext(ctx, db.Rebind(`INSERT INTO schema_migrations (version) VALUES (?)`), name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Info("migration uygulandı", "dosya", name)
	}
	return nil
}

func migrationTableDDL(d Dialect) string {
	switch d {
	case Postgres:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`
	case MySQL:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`
	default:
		return `CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now')))`
	}
}

// stripComments yalnızca "--" ile başlayan satırları atar; dosya sadece
// yorumdan oluşuyorsa boş döner ve Exec çağrılmaz.
func stripComments(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "--") {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
