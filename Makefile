BINARY  := netflow-probe
PKG     := ./cmd/netflow-probe
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet fmt tidy clean dist windows linux darwin macapp dmg linux-gui linux-deb linux-rpm windows-zip windows-installer bump

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

## Linux .deb installer (Ubuntu/Debian). Must run ON Linux.
linux-deb:
	bash packaging/linux/make_deb.sh

## Linux .rpm installer (Fedora). Must run ON Fedora with rpm-build.
linux-rpm:
	bash packaging/linux/make_rpm.sh

## Windows portable ZIP (needs packaging/windows/payload + Mesa; see docs/gui-windows.md)
windows-zip:
	bash packaging/windows/make_zip.sh

## Windows NSIS installer (needs makensis + payload; see docs/gui-windows.md)
windows-installer:
	bash packaging/windows/make_installer.sh

## Bump the internal version everywhere: make bump V=1.0.33
bump:
	@test -n "$(V)" || (echo "uso: make bump V=x.y.z" >&2; exit 1)
	python3 scripts/bump_version.py $(V)
