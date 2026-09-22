# Derleme bağlamı proje kökü, Dockerfile deploy/ altında:
#   docker build -f deploy/Dockerfile -t __APP__ .
# Coolify: Build pipeline → Dockerfile location = /deploy/Dockerfile

FROM golang:__GOVER__-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/__APP__ ./cmd/__APP__

FROM alpine:3.20
# ca-certificates: dış HTTPS çağrıları · tzdata: yerel saat dilimi
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H app && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=build /out/__APP__ /app/__APP__
USER app
EXPOSE 8080
# Container'da 0.0.0.0 şart: 127.0.0.1 kalırsa reverse proxy (Traefik) erişemez.
ENV __PREFIX___LISTEN=0.0.0.0:8080
# SQLite dosyası ve yüklenen dosyalar burada; dağıtımda kalıcı volume bağla.
VOLUME ["/app/data"]
# Coolify/Docker rolling update'te kullanır: yeni container sağlıklı olmadan eski kapanmaz.
# Alpine'da curl yok, wget (busybox) var.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/__APP__"]
CMD ["serve"]
