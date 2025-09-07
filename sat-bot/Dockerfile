# Use the official Golang image as the base image
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Start a new stage from scratch for a smaller final image
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy the context.txt file
COPY --from=builder /app/context.txt .

# Copy the .env file (optional, can also be passed via environment variables)
COPY --from=builder /app/.env .

# Expose port 8080
EXPOSE 8080

# Set default environment variables (can be overridden)
ENV PORT=8080

# Run the binary
CMD ["./main"]