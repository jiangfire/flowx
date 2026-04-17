.PHONY: build run test clean deps migrate swagger docker-build docker-up docker-down docker-logs docker-restart all test-short

# Go parameters
BINARY_NAME=flowx-server
GO=go
MAIN_PATH=./cmd/server

# Build
build:
	CGO_ENABLED=0 $(GO) build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run
run:
	$(GO) run $(MAIN_PATH)

# Test
test:
	$(GO) test ./... -count=1 -v

test-short:
	$(GO) test ./... -count=1 -short

# Clean
clean:
	rm -rf bin/

# Dependencies
deps:
	$(GO) mod tidy

# Database migration
migrate:
	$(GO) run $(MAIN_PATH) --migrate

# Swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Docker
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f flowx-server

docker-restart:
	docker compose restart flowx-server

# All
all: test build
