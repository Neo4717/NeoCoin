# NeoCoin Makefile
# Reproducible build system

.PHONY: build build-reproducible test docker-build docker-push clean install-deps

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GOFLAGS := -trimpath -mod=readonly

# Default build
build:
	go build ${GOFLAGS} -o blockchain .

# Reproducible build (deterministic)
build-reproducible:
	GOWORK=off go build -ldflags="-s -w \
		-X main.version=${VERSION} \
		-X main.buildTime=${BUILD_TIME}" \
		-mod=vendor -o blockchain .

# Build with debug info
build-debug:
	go build -o blockchain .

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Vulnerability scan
vuln:
	go sec ./...

# Docker build (production)
docker-build:
	docker build -t neocoin/blockchain:${VERSION} -t neocoin/blockchain:latest .

# Docker build (reproducible)
docker-build-reproducible:
	docker build --build-arg VERSION=${VERSION} --build-arg BUILD_TIME=${BUILD_TIME} \
		-t neocoin/blockchain:${VERSION} -t neocoin/blockchain:latest -f Dockerfile.reproducible .

# Clean build artifacts
clean:
	rm -f blockchain coverage.out

# Install dependencies
install-deps:
	go mod download

# Run smoke tests
smoke:
	docker compose -f docker-compose.smoke.yml up --build --abort-on-container-exit

# Run testnet
testnet:
	docker compose -f docker-compose.testnet.yml up -d

# Run mainnet
mainnet:
	docker compose -f docker-compose.mainnet.yml up -d