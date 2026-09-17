package handler

import (
	"net/http"
	"runtime/debug"
	"time"
)

// statusWriter yanıt kodunu yakalar (loglama için).
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// withLog her isteği yöntem/yol/durum/süre ile loglar.
func (s *Server) withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.log.Info("istek",
			"yöntem", r.Method, "yol", r.URL.Path,
			"durum", sw.status, "süre", time.Since(start).Round(time.Millisecond),
			"ip", r.RemoteAddr)
	})
}

// withRecover panic'i 500'e çevirir; tek bir handler tüm süreci düşürmesin.
func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "hata", rec, "yığın", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, errBody("sunucu hatası"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
