# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations for production
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /app/main \
    ./cmd/app/main.go

# Production stage
FROM scratch

# Copy binary
COPY --from=builder /app/main /main

# Use non-root user for security
USER 1000:1000

# Expose port
EXPOSE 8080

CMD ["/main"]