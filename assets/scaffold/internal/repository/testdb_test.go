package repository

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// testDB testler için taze bir veritabanı açar ve migration'ları uygular.
// __PREFIX___TEST_DATABASE_URL verilmişse onu kullanır (Postgres/MySQL'e karşı
// koşmak için); yoksa SQLite bellek içi veritabanı açar.
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("__PREFIX___TEST_DATABASE_URL")
	if url == "" {
		url = "sqlite://:memory:"
	}
	db, err := Open(context.Background(), url)
	if err != nil {
		t.Skipf("test veritabanı açılamadı: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(context.Background(), db, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
