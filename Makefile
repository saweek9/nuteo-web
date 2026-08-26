.PHONY: build run test fmt vet tidy docker-build docker-up docker-down clean

BIN := bin/nuteo-web

build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BIN) ./cmd/server
	@echo "Built $(BIN)"

run: build
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example — please edit it."; exit 1; fi
	@set -a; . ./.env; set +a; $(BIN)

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	@mkdir -p coverage
	go test -coverprofile=coverage/cover.out ./...
	go tool cover -html=coverage/cover.out -o coverage/cover.html
	@echo "Coverage report: coverage/cover.html"
	@go tool cover -func=coverage/cover.out | grep total

bench:
	go test -bench=. -benchmem ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# --- Docker ---

docker-build:
	docker build -f deploy/Dockerfile -t nuteo-web:latest .

docker-up:
	cd deploy && docker compose up -d

docker-down:
	cd deploy && docker compose down

docker-logs:
	cd deploy && docker compose logs -f app

# --- Clean ---

clean:
	rm -rf bin/
