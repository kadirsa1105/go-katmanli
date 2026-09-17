package repository

// SQLite sürücüsü (saf Go, CGO gerektirmez → cross-compile sorunsuz).
// database/sql'e "sqlite" adıyla kaydolur.
import _ "modernc.org/sqlite"
