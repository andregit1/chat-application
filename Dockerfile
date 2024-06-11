# Use a base image with both Go and Node.js
FROM golang:1.22.4-alpine3.20 AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum to enable dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o chatapp main.go

# Start a new stage for the final image
FROM alpine:3.20

# Install Node.js for wscat
RUN apk add --no-cache nodejs npm

# Set the working directory
WORKDIR /app

# Copy the built Go binary from the builder stage
COPY --from=builder /app/chatapp /app/chatapp

# Install wscat globally using npm
RUN npm install -g wscat

# Expose the port that the Go server listens on
EXPOSE 8080

# Set the entrypoint to the Go application
ENTRYPOINT ["/app/chatapp"]
