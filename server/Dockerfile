# Use the official Go image with version 1.24
FROM golang:1.24.0-alpine AS builder 

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod tidy

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o main main.go && \
    go clean -cache -modcache

# Create a non-root user and set permissions
RUN adduser -D -h /home/appuser appuser && \
    mkdir -p logs && \
    chown -R appuser:appuser config/ && \
    chown -R appuser:appuser logs/ && \
    chmod +r config/* && \
    chmod +w logs && \
    chmod +x main

# EXPOSE 8080

USER appuser

# Command to run the application
CMD ["./main"]