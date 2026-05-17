VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
LDFLAGS := -X github.com/jinkp/outlook-go-mcp/internal/version.Version=$(VERSION)

BIN_DIR     := bin
BUILD_TARGET := ./cmd/outlook-mcp
# Windows-only: outlook-mcp wraps Outlook COM automation
RELEASE_PLATFORMS := windows/amd64 windows/arm64

.PHONY: build build-all test lint clean install

## build: compile for the current platform (Windows only)
build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/outlook-mcp.exe $(BUILD_TARGET)

## build-all: cross-compile release binaries for all supported platforms
build-all:
	@mkdir -p $(BIN_DIR)
	@for target in $(RELEASE_PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
			-o "$(BIN_DIR)/outlook-mcp-$$os-$$arch.exe" $(BUILD_TARGET); \
	done

## test: run the full test suite
test:
	go test ./...

## lint: run go vet
lint:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## install: install to GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" $(BUILD_TARGET)
