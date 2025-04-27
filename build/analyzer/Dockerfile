FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git ca-certificates gcc g++ libc-dev

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o analyzer ./cmd/analyzer/main.go

# Create a minimal image
FROM alpine:latest

WORKDIR /app

# Install required runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/analyzer .

# Set environment variables
ENV ANALYZER_SERVICE_PORT=8081 \
    DB_HOST=postgres \
    DB_PORT=5432 \
    DB_USER=postgres \
    DB_PASSWORD=postgres \
    DB_NAME=ecommerce \
    KAFKA_BROKERS=kafka:9092 \
    ENVIRONMENT=production

# Expose the port
EXPOSE 8081

# Run the application
CMD ["./analyzer"]