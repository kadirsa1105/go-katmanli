# Coolify ile dağıtım

systemd yerine Coolify kullanılan kurulumlar için. Varsayım: tek sunucu, önde Traefik,
veritabanları ayrı bir projede paylaşımlı servis olarak duruyor.

## Kaynağı oluşturma

- New Resource → Private Repository (GitHub App ya da deploy key) → repo + branch.
- Build Pack: **Dockerfile**. Nixpacks/Railpack seçme; repodaki Dockerfile derlensin.
- Ports Exposes: uygulamanın **container içinde** dinlediği port (iskelette `8080`).
  Bu dışarı açılan port değil, Traefik'in hedefi; dışarısı her zaman 80/443.
- Network: **Connect to the predefined Coolify network**. Aksi halde uygulama veritabanı
  container'ını çözemez (`dial tcp: lookup … no such host`).
- Ports Mappings boş kalsın — dışarı port açmak proxy'yi devre dışı bırakır.

## Ortam değişkenleri

Dockerfile'da sabitlenenler (panele girilmez): `<PREFIX>_LISTEN=0.0.0.0:8080`.
Yerelde varsayılan `127.0.0.1` doğrudur, container'da `0.0.0.0` şarttır; 127.0.0.1 kalırsa
Traefik erişemez ve healthcheck dışında her istek 502 döner.

Panele girilenler: `<PREFIX>_DATABASE_URL`, `<PREFIX>_PUBLIC_URL`, `<PREFIX>_ENV`, `<PREFIX>_LOG_LEVEL`
ve varsa gizli anahtarlar (`<PREFIX>_JWT_SECRET`, `<PREFIX>_ENCRYPTION_KEY`).

Birden çok uygulamada tekrar eden değerler (veritabanı adresi, kullanıcı, şifre) **Shared Variables**
altında Team kapsamında bir kez tanımlanır, uygulamada `{{team.MARIADB_HOST}}` biçiminde kullanılır:

```
<PREFIX>_DATABASE_URL = mysql://{{team.MARIADB_USER}}:{{team.MARIADB_PASSWORD}}@{{team.MARIADB_HOST}}:3306/<db>
```

Veritabanı yeniden kurulduğunda (container adı değişir) tek bir yeri düzeltip uygulamaları
redeploy etmek yeter. Her uygulamada ayrı ayrı yazılmış değer, taşıma gününde saatler yer.

Tuzaklar:
- Şifre DSN içinde URL kuralına tabidir: `@ : / ? %` varsa yüzde-kodla (`@` → `%40`) ya da
  baştan yalnız harf-rakam şifre üret. `Access denied … (using password: YES)` hatasının olağan sebebi budur.
- Şablondaki `<…>` yer tutucularını değerle birlikte yapıştırma; `<` `>` şifrenin parçası olur.
- Coolify her değişkeni Production ve Preview olarak iki kez tutar. Preview deploy kullanmıyorsan
  ikinci takımı görmezden gel, silmene gerek yok.

## Veritabanı adresi

Coolify'da veritabanı container'ının adı **kaynağın UUID'sidir**; panelde yazdığın ad yalnızca etikettir.
Doğru host: DB → General → "… URL (internal)" alanındaki `@` sonrası, ya da sunucuda
`docker network inspect coolify --format '{{range .Containers}}{{.Name}}{{"\n"}}{{end}}'`.

## Kalıcı depolama

Container silinince içindeki her şey gider. SQLite dosyası, yüklenen medya, üretilen PDF'ler için
Storages → volume ekle: `/app/data` (iskelette çalışma dizini `/app`, sqlite varsayılanı `./data/<app>.db`).

Volume'u **ilk deploy'dan önce** ekle. Sonradan eklersen aradaki veriler kaybolur; ayrıca boş bir
volume mount edilirken Docker image'daki mevcut dosyaları bir kez kopyalar — dizin doluysa bu olmaz.

## Healthcheck ve kesintisiz deploy

Dockerfile'daki `HEALTHCHECK`'i Coolify görür ve rolling update'te kullanır: yeni container sağlıklı
olmadan eskisi kapatılmaz, kesinti olmaz. Healthcheck yoksa geçişte birkaç saniye 502 alınır.
Alpine'da `wget` (busybox) vardır, `curl` yoktur — komutu ona göre yaz.

Kod güncellerken **Restart değil Redeploy**: Restart tek container'ı durdurup başlatır, rolling update yapmaz.

## Domain ve TLS

- Domain alanına protokolü ayrı seç, alan adını şemasız yaz (`app.example.com`). Virgülle çoklu
  giriş kabul edilmez; her alan adı ayrı satır.
- Cloudflare arkasındaysa SSL modu **Full** olmalı. **Flexible** + Coolify'ın "Redirect HTTP to HTTPS"
  açık olması sonsuz döngü üretir (`ERR_TOO_MANY_REDIRECTS`). Flexible'da kalınacaksa o yönlendirmeyi kapat.
- İki seviyeli alt alan adı (`app.c.example.com`) Cloudflare'in ücretsiz sertifikasının kapsamı dışındadır
  (`*.example.com` yalnız bir seviye) → `ERR_SSL_VERSION_OR_CIPHER_MISMATCH`. Ya tek seviye kullan,
  ya o kaydı DNS-only yap.
- Gerçek ziyaretçi IP'si `X-Forwarded-For` başlığındadır; `RemoteAddr` Traefik'i gösterir.

## Build hızı

Dockerfile önce `go.mod`/`go.sum` kopyalayıp `go mod download` çalıştırır; bağımlılık değişmedikçe
bu katman cache'ten gelir. Kod değişikliğinde yalnız derleme adımı tekrarlanır (~20-40 sn).
Coolify "Build configuration changed" derse cache atlanır, bu normaldir.

## Repoda COOLIFY.md

Kurulum adımları ve değişken tablosu projeyle birlikte dursun: iskelette hazır gelen `COOLIFY.md`
dosyasını doldur. Altı ay sonra sunucu taşırken tek bakılacak yer orası olsun.
