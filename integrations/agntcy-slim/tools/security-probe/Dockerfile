# context: integrations/
FROM golang:1.24-alpine AS builder
WORKDIR /src

# Copy module files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY agntcy-slim/tools/security-probe ./agntcy-slim/tools/security-probe

# Build the binary
RUN go build -o /bin/security-probe ./agntcy-slim/tools/security-probe

# Minimal runtime image
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/security-probe .
ENTRYPOINT ["./security-probe"]
