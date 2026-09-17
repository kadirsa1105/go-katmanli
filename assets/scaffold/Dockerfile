FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/__APP__ ./cmd/__APP__

FROM alpine:3.20
RUN adduser -D -H app && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=build /out/__APP__ /app/__APP__
USER app
EXPOSE 8080
ENV __PREFIX___LISTEN=0.0.0.0:8080
ENTRYPOINT ["/app/__APP__"]
CMD ["serve"]
