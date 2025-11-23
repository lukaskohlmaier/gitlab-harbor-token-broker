# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o broker ./cmd/broker

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 broker && \
    adduser -D -u 1000 -G broker broker

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/broker .

# Copy example config (can be overridden via volume mount)
COPY config.yaml config.example.yaml

# Change ownership
RUN chown -R broker:broker /app

# Switch to non-root user
USER broker

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/broker"]
CMD ["-config", "/app/config.yaml"]
