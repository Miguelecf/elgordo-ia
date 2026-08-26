GO ?= go
GOFILES := $(shell git ls-files '*.go')

.PHONY: check fmt mod test vet build installer race coverage

check: fmt mod test vet build installer

fmt:
	@unformatted="$$(gofmt -l $(GOFILES))"; test -z "$$unformatted" || { printf '%s\n' "Run gofmt on:" >&2; printf '%s\n' "$$unformatted" >&2; exit 1; }

mod:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build ./cmd/elgordo

installer:
	sh -n scripts/install.sh

race:
	$(GO) test -race ./...

coverage:
	$(GO) test -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
