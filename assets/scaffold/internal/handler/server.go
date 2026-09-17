// Package handler HTTP katmanıdır: isteği çözer, service'i çağrır, yanıtı yazar.
// İş kuralı ya da SQL burada yazılmaz. Router stdlib net/http (Go 1.22+ desenleri).
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"__MODULE__/internal/config"
	"__MODULE__/internal/service"
)

type Server struct {
	cfg  *config.Config
	svc  *service.Services
	log  *slog.Logger
	http *http.Server
}

func New(cfg *config.Config, svc *service.Services, log *slog.Logger) *Server {
	s := &Server{cfg: cfg, svc: svc, log: log}
	s.http = &http.Server{
		Addr:              cfg.Listen,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s
}

// routes tüm yolları tek yerde toplar; yeni endpoint buraya eklenir.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	// Örnek:
	// mux.HandleFunc("GET /api/v1/notes", s.handleNoteList)
	// mux.HandleFunc("POST /api/v1/notes", s.handleNoteCreate)
	// mux.HandleFunc("GET /api/v1/notes/{id}", s.handleNoteGet)
	return s.withRecover(s.withLog(mux))
}

// Run sunucuyu başlatır; ctx iptal olunca (SIGINT/SIGTERM) düzgün kapatır.
func (s *Server) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() { errc <- s.http.ListenAndServe() }()
	s.log.Info("dinleniyor", "adres", s.cfg.Listen)

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.log.Info("kapatılıyor")
		return s.http.Shutdown(shutdownCtx)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Health.Check(r.Context()); err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"durum": "ok"})
}
