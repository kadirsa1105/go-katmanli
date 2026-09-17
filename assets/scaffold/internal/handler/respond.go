package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"__MODULE__/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// writeJSON gövdeyi JSON olarak yazar.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// writeError domain hatasını HTTP koduna çevirir. Beklenmeyen hatalar
// loglanır; üretimde istemciye ayrıntı sızdırılmaz.
func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusBadRequest, errBody(ve.Error()))
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errBody(err.Error()))
	case errors.Is(err, domain.ErrConflict):
		writeJSON(w, http.StatusConflict, errBody(err.Error()))
	default:
		s.log.Error("istek başarısız", "yöntem", r.Method, "yol", r.URL.Path, "hata", err)
		msg := "sunucu hatası"
		if !s.cfg.IsProd() {
			msg = err.Error()
		}
		writeJSON(w, http.StatusInternalServerError, errBody(msg))
	}
}

func errBody(msg string) map[string]string { return map[string]string{"hata": msg} }

// decodeJSON istek gövdesini v'ye çözer; boyut sınırı ve bilinmeyen alan
// kontrolü yapar. Hata ValidationError olarak döner (→ 400).
func decodeJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return domain.Invalid("gövde", fmt.Sprintf("geçersiz JSON: %v", err))
	}
	if dec.More() {
		return domain.Invalid("gövde", "tek bir JSON nesnesi bekleniyor")
	}
	_, _ = io.Copy(io.Discard, r.Body)
	return nil
}
