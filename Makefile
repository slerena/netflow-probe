BINARY  := netflow-probe
PKG     := ./cmd/netflow-probe
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet fmt tidy clean dist windows linux darwin macapp dmg linux-gui

all: build

## build the binary for the current platform
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

## run the unit tests
test:
	go test ./...

## static analysis
vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -rf dist $(BINARY) $(BINARY).exe

## cross-compile the whole matrix (pure Go, no cgo)
dist:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(PKG)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-arm64.exe $(PKG)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64        $(PKG)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64        $(PKG)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64       $(PKG)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64       $(PKG)

windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY).exe $(PKG)

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

## macOS GUI bundle (.app) + ZIP with the Pandora icon. Requires cgo + Xcode CLT.
macapp:
	bash packaging/macos/make_app.sh

## macOS installer (.dmg, arrastrar a Aplicaciones). Requires dmgbuild:
##   python3 -m venv .venv && .venv/bin/pip install dmgbuild pillow
dmg:
	bash packaging/macos/make_dmg.sh

## Linux GUI. Must run ON Linux with cgo + Fyne deps (see docs/gui-linux.md).
linux-gui:
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-gui ./cmd/netflow-probe-gui
