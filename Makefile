APP := nextclone
CMD := ./cmd/nextclone
DIST := dist
GO_BUILD_FLAGS := -trimpath
GO_LDFLAGS := -s -w
VERSION := $(shell tr -d '[:space:]' < VERSION)
VERSION_LDFLAG := -X github.com/marvinscham/nextclone/internal/version.Version=$(VERSION)

.PHONY: help run test build build-linux build-windows build-all clean

help:
	@printf 'Available targets:\n'
	@printf '  run           Run the app with go run\n'
	@printf '  test          Run all Go tests\n'
	@printf '  build         Build the Linux executable\n'
	@printf '  build-linux   Build dist/nextclone-linux-amd64\n'
	@printf '  build-windows Build dist/nextclone-windows-amd64.exe\n'
	@printf '  build-all     Build Linux and Windows executables\n'
	@printf '  clean         Remove build artifacts\n'

run:
	go run $(CMD)

test:
	go test ./...

build: build-linux

build-linux:
	mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -ldflags "$(GO_LDFLAGS) $(VERSION_LDFLAG)" -o $(DIST)/$(APP)-linux-amd64 $(CMD)

build-windows:
	mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build $(GO_BUILD_FLAGS) -ldflags "$(GO_LDFLAGS) $(VERSION_LDFLAG) -H windowsgui" -o $(DIST)/$(APP)-windows-amd64.exe $(CMD)

build-all: build-linux build-windows

clean:
	rm -rf $(DIST)
