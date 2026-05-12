# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o firefly main.go

# Runtime stage
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -h /app firefly
USER firefly

# Copy binary from builder
COPY --from=builder /app/firefly .
COPY --from=builder /app/config ./config

# Expose ports
EXPOSE 8080 9090

# Set default timezone
ENV TZ=Asia/Shanghai

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ./firefly -check-health || exit 1

# Run the binary
ENTRYPOINT ["./firefly"]
CMD ["-config", "./config/config.yaml"]
