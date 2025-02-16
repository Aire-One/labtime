BINARY_NAME=labtime

.PHONY: all
all: lint test build

.PHONY: clean
clean:
	go clean -i ./...
	rm -rf build

.PHONY: lint
lint:
	@echo "Running golangci-lint"
	golangci-lint run

.PHONY: test
test:
	@echo "Running tests"
	go test -v ./...

.PHONY: build
build:
	@echo "Building binary"
	go build -o build/$(BINARY_NAME) cmd/labtime/main.go

.PHONY: dev
dev:
	go run cmd/labtime/main.go --config configs/example-config.yaml