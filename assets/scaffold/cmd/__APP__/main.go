package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"__MODULE__/internal/config"
	"__MODULE__/internal/handler"
	"__MODULE__/internal/repository"
	"__MODULE__/internal/service"
)

// version derleme sırasında -ldflags ile doldurulur (bkz. Makefile).
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hata:", err)
		os.Exit(1)
	}
}

// run tüm bağımlılıkları tek yerde kurar: config → repository → service → handler.
// Katmanlar birbirini doğrudan tanımaz; bağlantı yalnızca burada yapılır.
func run(args []string) error {
	fs := flag.NewFlagSet("__APP__", flag.ContinueOnError)
	envFile := fs.String("env", ".env", "ortam değişkenleri dosyası (yoksa sessizce atlanır)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cmd := "serve"
	if fs.NArg() > 0 {
		cmd = fs.Arg(0)
	}
	if cmd == "version" {
		fmt.Println(version)
		return nil
	}

	cfg, err := config.Load(*envFile)
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch cmd {
	case "migrate":
		return repository.Migrate(ctx, db, log)
	case "serve":
		// Açılışta migration: servis her zaman güncel şemayla kalkar,
		// ayrı bir "migrate çalıştırmayı unuttum" adımı olmaz.
		if err := repository.Migrate(ctx, db, log); err != nil {
			return err
		}
		repos := repository.New(db)
		svcs := service.New(repos, log)
		srv := handler.New(cfg, svcs, log.With("bileşen", "http"))
		log.Info("başlıyor", "sürüm", version, "veritabanı", db.Dialect)
		return srv.Run(ctx)
	default:
		return fmt.Errorf("bilinmeyen komut: %q (serve | migrate | version)", cmd)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
