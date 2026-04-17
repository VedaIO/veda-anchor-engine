# Makefile for Veda Anchor Engine

VERSION ?= $(shell git describe --tags --always --dirty --first-parent 2>/dev/null || echo "dev")

.PHONY: all build generate fmt lint clean

all: build
generate:
	@echo "Generating version info..."
	cd src && go generate

build: generate
	@echo "Building Veda Anchor Engine for windows..."
	CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu -Wl,--subsystem,windows" GOOS=windows GOARCH=amd64 go build -ldflags="-w -H=windowsgui -X main.Version=$(VERSION)" -o ./bin/veda-anchor-engine.exe ./src/

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu" GOOS=windows go vet ./...
	CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu" GOOS=windows go fix ./...
	CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu" GOOS=windows golangci-lint run

clean:
	@echo "Cleaning..."
	rm -rf ./bin/veda-anchor-engine.exe
	rm -f src/resource.syso

