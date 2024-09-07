# syntax=docker/dockerfile:1

FROM golang:1.23.1 AS builder

WORKDIR /app

COPY go.mod go.sum main.go ./
RUN go mod download
RUN go build

FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /app/labtime /labtime

# This is the port currently hardcoded in the application
EXPOSE 2112

# For now the config file path/name are hardcoded in the application
VOLUME ["/config"]

USER nonroot:nonroot

ENTRYPOINT ["/labtime"]