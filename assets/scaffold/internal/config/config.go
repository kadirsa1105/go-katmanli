// Package config uygulamanın tüm ayarlarını tek yerde toplar.
//
// Kaynak sırası: .env dosyası → ortam değişkenleri (ortam değişkeni kazanır).
// Kodun başka hiçbir yerinde os.Getenv çağrılmaz; yeni bir ayar gerekiyorsa
// buraya alan olarak eklenir. Böylece "bu servis hangi ayarlarla çalışıyor"
// sorusunun cevabı her zaman bu dosyadır.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prefix tüm ortam değişkenlerinin başına gelir: __PREFIX___DATABASE_URL gibi.
const Prefix = "__PREFIX___"

type Config struct {
	// DatabaseURL şemasıyla sürücüyü belirler:
	//   postgres://user:pass@host:5432/db?sslmode=disable
	//   mysql://user:pass@host:3306/db
	//   sqlite://./data/__APP__.db
	DatabaseURL string
	// Listen HTTP dinleme adresi. Varsayılan 127.0.0.1: yanlışlıkla dışarı açılmasın.
	Listen string
	// LogLevel: debug | info | warn | error
	LogLevel string
	// Env: dev | prod. Hata ayrıntılarının istemciye gidip gitmeyeceğini belirler.
	Env string
}

// Load .env dosyasını (varsa) okur, sonra ortam değişkenleriyle ezer.
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := loadEnvFile(envFile); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("env dosyası: %w", err)
		}
	}

	c := &Config{
		DatabaseURL: getenv("DATABASE_URL", "__DATABASE_URL__"),
		Listen:      getenv("LISTEN", "127.0.0.1:8080"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		Env:         getenv("ENV", "dev"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("%sDATABASE_URL tanımlı değil", Prefix)
	}
	return c, nil
}

// IsProd üretim ortamında mıyız?
func (c *Config) IsProd() bool { return c.Env == "prod" }

func getenv(key, def string) string {
	if v := os.Getenv(Prefix + key); v != "" {
		return v
	}
	return def
}

// loadEnvFile KEY=VALUE satırlarını okur; zaten tanımlı ortam değişkenlerini ezmez.
func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
	}
	return sc.Err()
}
