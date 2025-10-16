.PHONY: build test clean install docker-build help

BINARY_NAME=zkbackup
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -s -w"

help: ## Show help information
	@echo "zkbackup Makefile usage"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build binary
	go build ${LDFLAGS} -o ${BINARY_NAME} main.go
	@echo "✅ Build complete: ${BINARY_NAME}"

build-all: ## Build binaries for all platforms
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 main.go
	@echo "✅ Multi-platform build complete"

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Generate test coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

clean: ## Clean build files
	rm -f ${BINARY_NAME} ${BINARY_NAME}-*
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

install: ## Install to $GOPATH/bin
	go install ${LDFLAGS}
	@echo "✅ Installed to: $$(which ${BINARY_NAME})"

docker-build: ## Build Docker image
	docker build -t ${BINARY_NAME}:${VERSION} -t ${BINARY_NAME}:latest .
	@echo "✅ Docker image build complete: ${BINARY_NAME}:${VERSION}"

fmt: ## Format code
	go fmt ./...
	@echo "✅ Code formatting complete"

lint: ## Run linter
	golangci-lint run ./...
	@echo "✅ Linting complete"

deps: ## Download dependencies
	go mod download
	go mod tidy
	@echo "✅ Dependencies downloaded"

run-example: build ## Run example
	./${BINARY_NAME} --help

.DEFAULT_GOAL := help
