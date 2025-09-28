# Use the official Go image with version 1.24
FROM golang:1.24.0

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod tidy

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -ldflags="-s -w" -o main main.go

# Create a non-root user and set permissions
RUN useradd -m appuser && \
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