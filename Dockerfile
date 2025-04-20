FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o traefik-log-processor ./cmd/main.go

# Use a minimal alpine image for the final container
FROM alpine:3.17

# Add ca-certificates for any HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /build/traefik-log-processor .

# Copy default config
COPY config.yaml .

# Create directories for logs
RUN mkdir -p /logs /output

# Set user to non-root
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -h /app -s /bin/sh -D appuser && \
    chown -R appuser:appuser /app /logs /output

USER appuser

# Command to run
ENTRYPOINT ["/app/traefik-log-processor"]
CMD ["--config", "/app/config.yaml"]