build:
	@echo "Building the application..."
	go build -o librascan ./cmd/librascan

run:
	@echo "Running the application..."
	./librascan

docker-build:
	@echo "Building the Docker image..."
	docker build -t gouthampersonal/librascan:latest .

docker-push:
	@echo "Pushing the Docker image to the registry..."
	docker push gouthampersonal/librascan:latest

dev:
	@echo "Starting rapid iteration mode..."
	air

setup-deps:
	@echo "Installing dependencies..."
	go install github.com/air-verse/air@latest
	go install github.com/gokrazy/tools/cmd/gok@main
	go mod tidy

download-db:
	@echo "Downloading the latest database..."
	rm -rf ./.db
	mkdir -p ./.db
	litestream restore -config litestream.yml ./.db/librascan.db

update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

test:
	@echo "Running all tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --timeout=5m
