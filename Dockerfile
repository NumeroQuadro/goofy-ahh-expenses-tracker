# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOPROXY=direct go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Create certs directory
RUN mkdir -p /app/certs

# Copy the binary
COPY --from=builder /app/main .

# Copy static files
COPY --from=builder /app/static ./static

# Create data directory
RUN mkdir -p /app/data

# Expose ports (app defaults to 0.0.0.0:8082)
EXPOSE 8082 443

CMD ["./main"]
