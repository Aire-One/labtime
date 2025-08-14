BINARY_NAME=labtime

.PHONY: all
all: lint cspell yamllint markdownlint tidy-check test generate build build-generator

.PHONY: clean
clean:
	go clean -i ./...
	rm -rf build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: yamllint
yamllint:
	yamllint --strict .

.PHONY: markdownlint
markdownlint:
	markdownlint-cli2 **/*.md

.PHONY: cspell
cspell:
	docker run -it --rm -v ./:/workdir ghcr.io/streetsidesoftware/cspell:9.2.0 lint /workdir

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	go build -o build/$(BINARY_NAME) cmd/labtime/main.go

.PHONY: build-generator
build-generator:
	go build -o build/$(BINARY_NAME)-generator cmd/generator/main.go

.PHONY: dev
dev:
	go run cmd/labtime/main.go --config configs/example-config.yaml

.PHONY: generate
generate:
	go generate ./...

.PHONY: tidy-check
tidy-check:
	go mod tidy -diff

.PHONY: tidy
tidy:
	go mod tidy
