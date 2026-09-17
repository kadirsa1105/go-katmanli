// Package domain uygulamanın iş nesnelerini (modeller) ve katmanlar arası
// ortak hata türlerini barındırır. Hiçbir katmana bağımlı değildir; her katman
// buraya bağımlı olabilir.
package domain

import (
	"errors"
	"fmt"
)

// Katmanlar arası ortak hatalar. Repository bunları döner, service iletir,
// handler HTTP koduna çevirir (bkz. handler/respond.go).
var (
	ErrNotFound = errors.New("kayıt bulunamadı")
	ErrConflict = errors.New("kayıt zaten var")
)

// ValidationError kullanıcı girdisi hatalarını taşır; handler 400 döner.
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

// Invalid kısa yoldan ValidationError üretir.
func Invalid(field, msg string) error { return &ValidationError{Field: field, Msg: msg} }
