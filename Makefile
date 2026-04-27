.PHONY: build test fmt lint run dump

GOLANGCI_LINT_VERSION ?= v2.10.1

build:
	go build -o bin/whoiscached ./cmd/whoiscached

test:
	go test -count=1 ./...

fmt:
	gofmt -s -w .

lint:
	CGO_ENABLED=0 go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

run:
	go run ./cmd/whoiscached -config local/config.ini

dump:
	go run ./cmd/whoiscached -config local/config.ini -dump-keys
