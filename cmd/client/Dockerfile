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
COPY pkg/ ./pkg/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client cmd/client/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ffmpeg yt-dlp ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/client .

# Create downloads directory
RUN mkdir -p downloads

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./client"]
