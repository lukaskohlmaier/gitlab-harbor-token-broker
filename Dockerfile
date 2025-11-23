# Build stage for Go application
FROM golang:1.21-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o broker ./cmd/broker

# Build stage for UI
FROM node:20-alpine AS ui-builder

WORKDIR /build/ui

# Copy package files
COPY ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy UI source
COPY ui/ ./

# Build UI
RUN npm run build

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and wget for healthcheck
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 broker && \
    adduser -D -u 1000 -G broker broker

WORKDIR /app

# Copy binary from go-builder
COPY --from=go-builder /build/broker .

# Copy UI build from ui-builder
COPY --from=ui-builder /build/ui/dist ./ui/dist

# Copy migrations
COPY migrations ./migrations

# Copy example configs (can be overridden via volume mount)
COPY config.yaml config.example.yaml
COPY config.db.yaml config.db.example.yaml

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
