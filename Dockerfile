# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download the Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o go-page-processor .

# Stage 2: Create the final image
FROM alpine:latest

# Copy the built Go application from the builder stage
COPY --from=builder /app/go-page-processor /

# Command to run the application
CMD ["/go-page-processor"]

# Workid /data a volume
VOLUME /data
VOLUME /data/pages

WORKDIR /data

