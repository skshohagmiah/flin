# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the CLI
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o flin ./cmd/flin

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kvserver ./cmd/kvserver

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binaries from builder
COPY --from=builder /app/flin .
COPY --from=builder /app/kvserver .

# Create data directory
RUN mkdir -p /data

# Set environment variables
ENV FLIN_DATA_DIR=/data

# Expose server port
EXPOSE 6380

# Default command (can be overridden)
CMD ["./flin", "help"]
