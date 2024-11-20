FROM golang:1.22-bullseye AS builder

WORKDIR /build/app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY main.go .
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o demo

# run
FROM debian:bullseye-slim

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/app/demo .

ENV GIN_MODE=release

ENTRYPOINT ["/app/demo"]
