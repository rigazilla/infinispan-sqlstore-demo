# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o inmemory-catalogue-go cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/inmemory-catalogue-go .

# Copy XML configuration files
COPY --from=builder /app/tableStore.xml .
COPY --from=builder /app/queryStore.xml .

# Expose port
EXPOSE 8280

# Run the application
CMD ["./inmemory-catalogue-go"]
