BINARY := agentspace
PREFIX ?= /usr/local
INSTALL_DIR := $(PREFIX)/bin

# Detect Windows for the .exe suffix.
ifeq ($(OS),Windows_NT)
	EXT := .exe
else
	EXT :=
endif

VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w

.PHONY: all build install clean test vet fmt tidy

all: build

## build: compile the agentspace binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)$(EXT) .

## install: build and copy the binary into $(INSTALL_DIR)
install: build
	install -d $(INSTALL_DIR)
	install -m 0755 $(BINARY)$(EXT) $(INSTALL_DIR)/$(BINARY)$(EXT)

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format the code
fmt:
	go fmt ./...

## tidy: tidy module dependencies
tidy:
	go mod tidy
