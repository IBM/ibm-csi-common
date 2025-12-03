GOPACKAGES=$(shell go list ./... | grep -v /vendor/ | grep -v /tests)
GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./tests/e2e/*")
VERSION := latest
GOPATH := $(shell go env GOPATH)
LINT_BIN=$(GOPATH)/bin/golangci-lint

GIT_COMMIT_SHA="$(shell git rev-parse HEAD 2>/dev/null)"
GIT_REMOTE_URL="$(shell git config --get remote.origin.url 2>/dev/null)"
BUILD_DATE="$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")"
ARCH=$(shell docker version -f {{.Client.Arch}})

GO111MODULE_FLAG?=on
export GO111MODULE=$(GO111MODULE_FLAG)

export LINT_VERSION="1.60.1"

COLOR_YELLOW=\033[0;33m
COLOR_RESET=\033[0m

OSS_FILES := go.mod

.PHONY: all
all: deps fmt vet test

.PHONY: deps
deps:
	@echo "Installing dependencies ..."

	@go mod download

	@if ! command -v gotestcover >/dev/null; then \
		echo "Installing gotestcover ..."; \
		go install github.com/pierrre/gotestcover@latest; \
	fi

	@if ! command -v $(LINT_BIN) >/dev/null || ! golangci-lint --version 2>/dev/null | grep -q "$(LINT_VERSION)"; then \
		echo "Installing golangci-lint $(LINT_VERSION) ..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
			| sh -s -- -b $(shell go env GOBIN 2>/dev/null || echo $$(go env GOPATH)/bin) v$(LINT_VERSION); \
	fi

.PHONY: fmt
fmt: 
	gofmt -l -w ${GOFILES}

.PHONY: dofmt
dofmt:
	$(LINT_BIN) run --disable-all --enable=gofmt --fix --skip-dirs=tests

.PHONY: lint
lint:
	$(LINT_BIN) run --timeout 600s --skip-dirs=tests

.PHONY: vet
vet:
	go vet ${GOPACKAGES}

.PHONY: coverage
coverage:
	go tool cover -html=cover.out -o=cover.html

.PHONY: test
test:
	$(GOPATH)/bin/gotestcover -v -race -short -coverprofile=cover.out ${GOPACKAGES}
