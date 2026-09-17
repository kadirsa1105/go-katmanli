// Package service iş kurallarının yaşadığı katmandır. HTTP'yi bilmez,
// SQL yazmaz: girdiyi doğrular, repository'yi çağırır, domain hatası döner.
// Böylece aynı servis ileride CLI'dan, cron'dan ya da başka bir API'den
// değişiklik gerekmeden çağrılabilir.
package service

import (
	"context"
	"log/slog"

	"__MODULE__/internal/repository"
)

// Services tüm servisleri bir arada tutar; main.go bunu handler.New'e verir.
// Yeni bir servis eklerken: alan ekle, New içinde kur.
type Services struct {
	Health *Health
	// Örnek: Notes *Notes
}

func New(repos *repository.Repos, log *slog.Logger) *Services {
	return &Services{
		Health: &Health{repos: repos},
		// Örnek: Notes: &Notes{repo: repos.Notes, log: log.With("servis", "notes")},
	}
}

// Health sağlık kontrolü.
type Health struct {
	repos *repository.Repos
}

// Check veritabanı erişilebilir mi?
func (h *Health) Check(ctx context.Context) error { return h.repos.Ping(ctx) }
