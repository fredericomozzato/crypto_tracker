.PHONY: fmt lint test vuln build check run

fmt:
	gofumpt -w .

lint:
	gofumpt -l . | grep . && exit 1 || true
	golangci-lint config verify
	golangci-lint run ./...
	govulncheck ./...

test:
	go test -race -coverprofile=coverage.out ./...

vuln:
	govulncheck ./...

build:
	go build -o crypto-tracker ./cmd/crypto-tracker

run: build
	./crypto-tracker

check: fmt lint test vuln