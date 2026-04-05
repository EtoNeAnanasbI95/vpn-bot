.PHONY: build run-dev run-prod tidy lint

build:
	go build -o ./bin/bot ./cmd/bot

run-dev:
	DEBUG=true go run ./cmd/bot

run-prod:
	go run ./cmd/bot

tidy:
	go mod tidy

lint:
	golangci-lint run ./...
