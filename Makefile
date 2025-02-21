build:
	@echo "Building the application..."
	go build -o librascan .

run:
	@echo "Running the application..."
	./librascan

docker-build:
	@echo "Building the Docker image..."
	docker build -t librascan .

dev:
	@echo "Starting rapid iteration mode..."
	reflex -r '\.go$$' -r '\.html$$' -r '\.js$$' -s -- sh -c "go build -o librascan ./cmd/librascan && ./librascan serve"

setup-deps:
	@echo "Installing dependencies..."
	go install github.com/cespare/reflex@latest
	go mod tidy
