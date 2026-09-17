# go-katmanli

Claude Code skill'i: katmanlı mimaride Go projesi kurar ve mevcut projelere aynı düzende özellik ekler.

- `cmd/<app>` → `internal/handler` → `internal/service` → `internal/repository` → `internal/domain`
- Ayarlar `.env` + `<APP>_*` ortam değişkenleri, tek `internal/config`
- Tek `DATABASE_URL` ile PostgreSQL / MySQL / SQLite (`?` yer tutucu + `Rebind`, aynı repository kodu üçünde de çalışır)
- Gömülü, numaralı `.sql` migration'lar; `serve` açılışında otomatik uygulanır
- stdlib `net/http`, graceful shutdown, JSON hata eşlemesi
- Makefile ile `CGO_ENABLED=0` cross-compile (linux/windows/darwin × amd64/arm64), systemd birimi, Dockerfile
- Türkçe yorum, İngilizce tanımlayıcı

## Kurulum

```bash
git clone git@github.com:kadirsa1105/go-katmanli.git ~/.claude/skills/go-katmanli
```

Claude Code bir sonraki oturumda skill'i otomatik görür. Yeni projede yalnızca
"Go ile X servisi yaz, postgres, modül github.com/..." demek yeterli.

## Elle kullanım

```bash
~/.claude/skills/go-katmanli/scripts/new_project.sh ./stokapp github.com/kadir/stokapp sqlite
cd stokapp && cp .env.example .env && make run
curl localhost:8080/healthz

# yeni migration
~/.claude/skills/go-katmanli/scripts/new_migration.sh notes
```

## İçerik

```
SKILL.md                 Claude'un okuduğu talimatlar
references/layout.md     dizin sorumlulukları, "ne nereye gitmez"
references/feature-example.md  tam örnek varlık (Note) — 3 veritabanında test edildi
references/databases.md  lehçe farkları: ID, tarih, upsert, tekil ihlal
references/build.md      Makefile, systemd, Docker, Windows/macOS notları
scripts/new_project.sh   deterministik iskelet üretici (go build + vet ile doğrular)
scripts/new_migration.sh sıradaki numarayla migration dosyası açar
assets/scaffold/         iskelet şablonu (__APP__, __MODULE__, __PREFIX__ yer tutucuları)
assets/db/<lehçe>/       sürücü import'u ve ilk migration
evals/evals.json         skill'in test istekleri ve kontrol maddeleri
```

## Lisans

MIT
