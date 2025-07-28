# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY landing/go.mod landing/go.sum ./landing/
COPY go.mod go.sum ./
COPY demo/go.mod demo/go.sum ./demo/

# Copy source code
COPY . .

# Change to landing directory
WORKDIR /workspace/landing

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /workspace/landing/main .

# Copy static files
COPY --from=builder /workspace/landing/static ./static

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]