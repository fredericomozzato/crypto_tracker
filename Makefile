.PHONY: fmt lint test vuln build check run

fmt:
	gofumpt -w .

lint:
	golangci-lint run ./...

test:
	go test -race -coverprofile=coverage.out ./...

vuln:
	govulncheck ./...

build:
	go build -o crypto-tracker ./cmd/crypto-tracker

run: build
	./crypto-tracker

check: fmt lint test vuln