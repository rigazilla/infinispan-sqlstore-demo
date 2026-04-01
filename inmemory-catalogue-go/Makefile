.PHONY: help run build clean test fmt vet docker-build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Run the application
	go run cmd/server/main.go

build: ## Build the binary
	go build -o inmemory-catalogue-go cmd/server/main.go

build-linux: ## Build for Linux (for Docker)
	GOOS=linux GOARCH=amd64 go build -o inmemory-catalogue-go cmd/server/main.go

clean: ## Remove build artifacts
	rm -f inmemory-catalogue-go

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

docker-build: ## Build Docker image
	docker build -t infinispan/inmemory-catalogue-go:latest .

docker-run: ## Run Docker container
	docker run -p 8280:8280 --network infinispan-sqlstore-demo_mynetwork infinispan/inmemory-catalogue-go:latest
