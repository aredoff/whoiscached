.PHONY: build test fmt lint

build:
	go build -o bin/whoiscached ./cmd/whoiscached

test:
	go test -count=1 ./...

fmt:
	gofmt -s -w .

lint:
	golangci-lint run ./... 2>/dev/null || go vet ./...
