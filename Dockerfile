# Build stage
FROM golang:1.24.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gofer ./cmd/gofer

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates git

# Create non-root user
RUN adduser -D -g '' gofer

# Copy binary from builder
COPY --from=builder /build/gofer /usr/local/bin/

# Set ownership
RUN chown gofer:gofer /usr/local/bin/gofer

# Switch to non-root user
USER gofer

# Set working directory
WORKDIR /workspace

# Expose volume for configuration
VOLUME ["/home/gofer/.gofer"]

# Set entrypoint
ENTRYPOINT ["gofer"]

# Default command
CMD ["--help"]