# Coolify kurulumu — __APP__

## 1. Veritabanı

Paylaşımlı veritabanı servisinde şema ve kullanıcı aç, sonra varsa dökümü içe aktar.

```sql
CREATE DATABASE __APP___db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER '__APP__'@'%' IDENTIFIED BY 'yalnizHarfRakamSifre';
GRANT ALL PRIVILEGES ON __APP___db.* TO '__APP__'@'%';
FLUSH PRIVILEGES;
```

## 2. Uygulama

| Ayar | Değer |
|---|---|
| Kaynak | Private Repository → bu repo, branch `main` |
| Build Pack | **Dockerfile** |
| Ports Exposes | **8080** (container içi port; dışarısı 80/443) |
| Network | **Connect to the predefined Coolify network** |
| Domain | `__APP__.example.com` |

## 3. Ortam değişkenleri

| Anahtar | Değer |
|---|---|
| `__PREFIX___DATABASE_URL` | `mysql://__APP__:<şifre>@{{team.MARIADB_HOST}}:3306/__APP___db` |
| `__PREFIX___PUBLIC_URL` | `https://__APP__.example.com` |
| `__PREFIX___ENV` | `prod` |
| `__PREFIX___LOG_LEVEL` | `info` |

Tekrar eden değerler (veritabanı host/kullanıcı/şifre) Shared Variables → Team altında bir kez
tanımlanır, burada `{{team.…}}` ile çağrılır. Şifrede `@ : / ? %` varsa yüzde-kodla.

`__PREFIX___LISTEN` Dockerfile'da ayarlı (`0.0.0.0:8080`), panele girilmez.

## 4. Kalıcı depolama

| Name | Destination path |
|---|---|
| `__APP__-data` | `/app/data` |

SQLite kullanılıyorsa veya dosya yükleniyorsa zorunlu. **İlk deploy'dan önce** ekle.

## 5. Healthcheck

Dockerfile'da tanımlı (`/healthz`). Coolify Healthcheck sekmesinde ayrıca: GET http
`127.0.0.1:8080/healthz`, interval 5, timeout 3, retries 3, start period 5.

## 6. Güncelleme

`git push` → Redeploy (Restart değil). Migration'lar `serve` açılışında kendiliğinden uygulanır.
