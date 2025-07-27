BINARY_NAME=labtime

.PHONY: all
all: lint cspell test generate build

.PHONY: clean
clean:
	go clean -i ./...
	rm -rf build

.PHONY: fmt
fmt:
	@echo "Formatting code"
	go fmt ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint"
	golangci-lint run

.PHONY: cspell
cspell:
	docker run -it --rm -v ./:/workdir ghcr.io/streetsidesoftware/cspell:9.2.0 lint /workdir

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

.PHONY: generate
generate:
	go generate ./...
