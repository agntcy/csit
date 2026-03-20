# context: benchmarks/
FROM golang:1.24-alpine AS builder
WORKDIR /src

# Copy module files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY agntcy-slim/tools/slim-bench ./agntcy-slim/tools/slim-bench

# Build the binary
RUN go build -o /bin/slim-bench ./agntcy-slim/tools/slim-bench

# Minimal runtime image
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/slim-bench .
ENTRYPOINT ["./slim-bench"]
