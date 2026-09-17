# Özellik ekleme örneği: Note varlığı

Aşağıdaki kod `new_project.sh` ile üretilmiş bir projeye eklenmiş, SQLite,
PostgreSQL ve MariaDB'de birebir aynı Go koduyla çalıştırılmıştır. Modül yolu
`github.com/kadir/stokapp` — kendi projenin modül yolunu kullan.

Sıra: migration → domain → repository (+ repos.go kaydı) → service (+ service.go
kaydı) → handler (+ routes) → test.

## 1. Migration

`scripts/new_migration.sh notes` çalıştır; her lehçe dizininde `000N_notes.sql` açılır.
Yalnızca projenin kullandığı lehçe dizini vardır; ikinci bir lehçe eklenirse o dizin de yazılır.

SQLite (`migrations/sqlite/0002_notes.sql`):
```sql
CREATE TABLE notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

PostgreSQL:
```sql
CREATE TABLE notes (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

MySQL/MariaDB:
```sql
CREATE TABLE notes (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    body       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 2. domain/note.go

Model + girdi tipi + doğrulama. Doğrulama burada olduğu için service, CLI ya da
başka bir giriş noktası aynı kuralı paylaşır.

```go
package domain

import (
	"strings"
	"time"
)

// Note basit bir not kaydı.
type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NoteInput oluşturma/güncelleme girdisi. Doğrulama burada: hem service
// hem de ileride başka giriş noktaları aynı kuralı kullanır.
type NoteInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (in *NoteInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return Invalid("title", "boş olamaz")
	}
	if len(in.Title) > 200 {
		return Invalid("title", "en fazla 200 karakter")
	}
	return nil
}
```

## 3. repository/note.go

Yalnızca SQL. `?` yer tutucu + `r.db.Rebind`. `sql.ErrNoRows` → `domain.ErrNotFound`
çevirisi tek bir `scanNote` içinde. Etkilenen satır 0 ise `ErrNotFound`.
INSERT'te Postgres `RETURNING`, diğerleri `LastInsertId` — bu tek lehçe dalı normaldir.

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kadir/stokapp/internal/domain"
)

// NoteRepo notes tablosuna erişim.
type NoteRepo struct{ db *DB }

const noteCols = `id, title, body, created_at, updated_at`

func scanNote(row interface{ Scan(...any) error }) (*domain.Note, error) {
	var n domain.Note
	if err := row.Scan(&n.ID, &n.Title, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepo) Get(ctx context.Context, id int64) (*domain.Note, error) {
	row := r.db.QueryRowContext(ctx, r.db.Rebind(`SELECT `+noteCols+` FROM notes WHERE id = ?`), id)
	return scanNote(row)
}

func (r *NoteRepo) List(ctx context.Context, limit, offset int) ([]*domain.Note, error) {
	rows, err := r.db.QueryContext(ctx,
		r.db.Rebind(`SELECT `+noteCols+` FROM notes ORDER BY id DESC LIMIT ? OFFSET ?`), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("notes list: %w", err)
	}
	defer rows.Close()
	notes := []*domain.Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Create ekler ve kaydı geri okur; RETURNING her lehçede olmadığından
// LastInsertId (MySQL/SQLite) ya da Postgres için RETURNING kullanılır.
func (r *NoteRepo) Create(ctx context.Context, in domain.NoteInput) (*domain.Note, error) {
	var id int64
	if r.db.Dialect == Postgres {
		err := r.db.QueryRowContext(ctx,
			r.db.Rebind(`INSERT INTO notes (title, body) VALUES (?, ?) RETURNING id`), in.Title, in.Body).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("notes insert: %w", err)
		}
	} else {
		res, err := r.db.ExecContext(ctx,
			r.db.Rebind(`INSERT INTO notes (title, body) VALUES (?, ?)`), in.Title, in.Body)
		if err != nil {
			return nil, fmt.Errorf("notes insert: %w", err)
		}
		if id, err = res.LastInsertId(); err != nil {
			return nil, err
		}
	}
	return r.Get(ctx, id)
}

func (r *NoteRepo) Update(ctx context.Context, id int64, in domain.NoteInput) (*domain.Note, error) {
	res, err := r.db.ExecContext(ctx,
		r.db.Rebind(`UPDATE notes SET title = ?, body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`),
		in.Title, in.Body, id)
	if err != nil {
		return nil, fmt.Errorf("notes update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, domain.ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *NoteRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, r.db.Rebind(`DELETE FROM notes WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("notes delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
```

`repos.go` kaydı:
```go
type Repos struct {
	db    *DB
	Notes *NoteRepo
}

func New(db *DB) *Repos {
	return &Repos{db: db, Notes: &NoteRepo{db: db}}
}
```

## 4. service/note.go

Doğrulama çağrısı, sınır düzeltmeleri (limit/offset), loglama. HTTP yok, SQL yok.

```go
package service

import (
	"context"
	"log/slog"

	"github.com/kadir/stokapp/internal/domain"
	"github.com/kadir/stokapp/internal/repository"
)

// Notes not iş kuralları.
type Notes struct {
	repo *repository.NoteRepo
	log  *slog.Logger
}

func (s *Notes) Get(ctx context.Context, id int64) (*domain.Note, error) {
	return s.repo.Get(ctx, id)
}

func (s *Notes) List(ctx context.Context, limit, offset int) ([]*domain.Note, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *Notes) Create(ctx context.Context, in domain.NoteInput) (*domain.Note, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	n, err := s.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	s.log.Info("not oluşturuldu", "id", n.ID)
	return n, nil
}

func (s *Notes) Update(ctx context.Context, id int64, in domain.NoteInput) (*domain.Note, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, in)
}

func (s *Notes) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
```

`service.go` kaydı:
```go
type Services struct {
	Health *Health
	Notes  *Notes
}

func New(repos *repository.Repos, log *slog.Logger) *Services {
	return &Services{
		Health: &Health{repos: repos},
		Notes:  &Notes{repo: repos.Notes, log: log.With("servis", "notes")},
	}
}
```

## 5. handler/note.go

Her handler aynı üç adım: çöz → çağır → yaz. Hata kodunu `s.writeError` seçer.

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/kadir/stokapp/internal/domain"
)

func (s *Server) handleNoteList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	notes, err := s.svc.Notes.List(r.Context(), limit, offset)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) handleNoteGet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	n, err := s.svc.Notes.Get(r.Context(), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleNoteCreate(w http.ResponseWriter, r *http.Request) {
	var in domain.NoteInput
	if err := decodeJSON(r, &in); err != nil {
		s.writeError(w, r, err)
		return
	}
	n, err := s.svc.Notes.Create(r.Context(), in)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleNoteUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var in domain.NoteInput
	if err := decodeJSON(r, &in); err != nil {
		s.writeError(w, r, err)
		return
	}
	n, err := s.svc.Notes.Update(r.Context(), id, in)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleNoteDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.svc.Notes.Delete(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pathID {id} yol parametresini okur; geçersizse ValidationError (→ 400).
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.Invalid("id", "pozitif tam sayı olmalı")
	}
	return id, nil
}
```

`server.go` içinde `routes()`:
```go
mux.HandleFunc("GET /api/v1/notes", s.handleNoteList)
mux.HandleFunc("POST /api/v1/notes", s.handleNoteCreate)
mux.HandleFunc("GET /api/v1/notes/{id}", s.handleNoteGet)
mux.HandleFunc("PUT /api/v1/notes/{id}", s.handleNoteUpdate)
mux.HandleFunc("DELETE /api/v1/notes/{id}", s.handleNoteDelete)
```

## 6. Test: repository/note_test.go

`testdb_test.go` iskelette hazır gelir: `<PREFIX>_TEST_DATABASE_URL` yoksa
`sqlite://:memory:` açar ve migration'ları uygular. Projede SQLite sürücüsü yoksa
(`driver_sqlite.go` yok) test `Skip` eder; o zaman env değişkeniyle gerçek DB ver:
`ENVANTER_TEST_DATABASE_URL=postgres://u:p@127.0.0.1:5432/envanter_test?sslmode=disable go test ./...`.
Test için projeye SQLite sürücüsü ekleme (migration'lar ikilenir).

```go
package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/kadir/stokapp/internal/domain"
)

func TestNoteRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	r := &NoteRepo{db: testDB(t)}

	n, err := r.Create(ctx, domain.NoteInput{Title: "a", Body: "b"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if n.ID == 0 || n.Title != "a" {
		t.Fatalf("beklenmeyen kayıt: %+v", n)
	}

	if _, err := r.Update(ctx, n.ID, domain.NoteInput{Title: "c"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := r.Get(ctx, n.ID)
	if got.Title != "c" {
		t.Fatalf("update yansımadı: %q", got.Title)
	}

	if err := r.Delete(ctx, n.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := r.Get(ctx, n.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("silinen kayıt için ErrNotFound bekleniyordu, gelen: %v", err)
	}
}
```

## Doğrulanan davranış

```
POST /api/v1/notes {"title":"  "}       → 400 {"hata":"title: boş olamaz"}
POST /api/v1/notes {"title":"x","foo":1} → 400 {"hata":"gövde: geçersiz JSON: json: unknown field \"foo\""}
GET  /api/v1/notes/abc                   → 400 {"hata":"id: pozitif tam sayı olmalı"}
GET  /api/v1/notes/99                    → 404 {"hata":"kayıt bulunamadı"}
DELETE /api/v1/notes/1                   → 204
```
