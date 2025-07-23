# syntax=docker/dockerfile:1

FROM golang:1.24.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o labtime cmd/labtime/main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /app/labtime /labtime

EXPOSE 2112

VOLUME ["/config"]

USER nonroot:nonroot

ENTRYPOINT ["/labtime"]
