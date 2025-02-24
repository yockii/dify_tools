# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Build the application
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o dify-tools ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache tzdata ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/dify-tools .
COPY --from=builder /app/config.example.yaml ./config.yaml

# Create necessary directories
RUN mkdir -p /app/logs

# Expose the application port
EXPOSE 8080

# Set environment variables
ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

# Run the application
CMD ["./dify-tools"]