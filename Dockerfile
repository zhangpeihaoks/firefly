# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o firefly main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/firefly .
COPY --from=builder /app/config ./config

# Expose ports
EXPOSE 8080 9090

# Set default timezone
ENV TZ=Asia/Shanghai

# Run the binary
ENTRYPOINT ["./firefly"]
CMD ["-config", "./config/config.yaml"]
