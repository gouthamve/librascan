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
	reflex -r '\.go$$' -s -- sh -c "go build -o librascan . && ./librascan"

setup-deps:
	@echo "Installing dependencies..."
	go install github.com/cespare/reflex@latest
