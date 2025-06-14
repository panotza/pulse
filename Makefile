.PHONY: build test lint help

# Build the application
build:
	go build -o bin/pulse .

# Run tests
test:
	go test ./...

# Run linter using Docker
lint:
	docker run -t --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v2.1.6 golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter using Docker"
	@echo "  help          - Show this help message"
