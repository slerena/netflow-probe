BINARY  := netflow-probe
PKG     := ./cmd/netflow-probe
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet fmt tidy clean dist windows linux darwin macapp

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

## macOS GUI bundle (.app) with the Pandora icon. Requires cgo + Xcode CLT.
macapp:
	bash scripts/build-macos-app.sh
