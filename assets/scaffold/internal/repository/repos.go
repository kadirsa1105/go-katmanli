package repository

import "context"

// Repos tüm depoları bir arada tutar; main.go bunu service.New'e verir.
// Yeni bir depo eklerken: alan ekle, New içinde kur. Bkz. skill referansı
// "feature-example.md".
type Repos struct {
	db *DB
	// Örnek: Notes *NoteRepo
}

func New(db *DB) *Repos {
	return &Repos{
		db: db,
		// Örnek: Notes: &NoteRepo{db: db},
	}
}

// Ping bağlantının canlı olduğunu doğrular (healthz için).
func (r *Repos) Ping(ctx context.Context) error { return r.db.PingContext(ctx) }
