# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files from root
COPY go.mod go.sum ./
RUN go mod download

# Copy internal packages
COPY internal ./internal
COPY pkg ./pkg

# Copy web-app source
COPY examples/web-app ./examples/web-app

# Build
WORKDIR /app/examples/web-app
RUN CGO_ENABLED=0 GOOS=linux go build -o web-app .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/examples/web-app/web-app .

EXPOSE 8080

CMD ["./web-app"]
