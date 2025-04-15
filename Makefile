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