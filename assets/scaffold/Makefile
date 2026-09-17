# __APP__ — derleme ve geliştirme hedefleri.
#
#   make build              yerel ikili → bin/__APP__
#   make run                derle + serve
#   make migrate            derle + yalnızca migration
#   make test / vet / fmt   kalite
#   make release            tüm platformlar → dist/
#   make release PLATFORMS="linux/amd64"   tek platform
#
# CGO_ENABLED=0: cross-compile için C derleyici gerekmez (SQLite sürücüsü saf Go).

APP      := __APP__
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
PLATFORMS ?= linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

.PHONY: build run migrate test vet fmt tidy release clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(APP) ./cmd/$(APP)

run: build
	./bin/$(APP) serve

migrate: build
	./bin/$(APP) migrate

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

release:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/$(APP)_$${os}_$${arch}$${ext}"; \
		echo "-> $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags '$(LDFLAGS)' -o "$$out" ./cmd/$(APP) || exit 1; \
	done

clean:
	rm -rf bin dist
