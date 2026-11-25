BINARY  := undelete
DIST    := dist

# Pure Go, no cgo: lets us cross compile from any host.
export CGO_ENABLED := 0

# Windows: force cmd.exe so behaviour does not depend on whether a POSIX sh
# happens to be on PATH. Recipes below avoid POSIX-only commands (mkdir -p,
# rm -rf, VAR=x prefixes); go build creates the output directory itself.
ifeq ($(OS),Windows_NT)
SHELL    := cmd.exe
NULL     := NUL
EXE      := .exe
CLEAN_CMD = if exist $(DIST) rmdir /s /q $(DIST)
else
NULL     := /dev/null
EXE      :=
CLEAN_CMD = rm -rf $(DIST)
endif

VERSION ?= $(shell git describe --tags --always --dirty 2>$(NULL) || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

TARGETS := \
	windows/amd64 \
	windows/arm64 \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64

.PHONY: all build windows linux darwin test vet fmt check clean help

help:
	@echo Targets:
	@echo   build     build for the host into $(DIST)/
	@echo   all       cross compile every target into $(DIST)/
	@echo   windows   windows/amd64 and windows/arm64
	@echo   linux     linux/amd64 and linux/arm64
	@echo   darwin    darwin/amd64 and darwin/arm64
	@echo   test      go test ./...
	@echo   vet       go vet ./...
	@echo   fmt       gofmt -w .
	@echo   check     fail if gofmt would change anything, then vet
	@echo   clean     remove $(DIST)/
	@echo Override version with: make all VERSION=1.2.3

all: $(TARGETS)

windows: windows/amd64 windows/arm64
linux:   linux/amd64 linux/arm64
darwin:  darwin/amd64 darwin/arm64

# build_target(os,arch,exe-suffix): one rule per os/arch pair. GOOS/GOARCH are
# target-specific exported variables so no shell env-prefix syntax is needed.
define build_target
.PHONY: $(1)/$(2)
$(1)/$(2): export GOOS := $(1)
$(1)/$(2): export GOARCH := $(2)
$(1)/$(2):
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-$(1)-$(2)$(3) .
endef

$(eval $(call build_target,windows,amd64,.exe))
$(eval $(call build_target,windows,arm64,.exe))
$(eval $(call build_target,linux,amd64,))
$(eval $(call build_target,linux,arm64,))
$(eval $(call build_target,darwin,amd64,))
$(eval $(call build_target,darwin,arm64,))

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)$(EXE) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# gofmt -l is evaluated by make itself, so no shell test/grep is needed.
check: vet
	$(if $(strip $(shell gofmt -l .)),$(error gofmt needed on: $(shell gofmt -l .)))
	@echo ok

clean:
	$(CLEAN_CMD)
