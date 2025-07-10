# Multi-stage build
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/server .

# Create downloads directory
RUN mkdir -p downloads

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]
