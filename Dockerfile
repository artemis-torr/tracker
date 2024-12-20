# Start with a base image containing Go and a Linux distro
FROM golang:1.23.4-alpine as builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY ./src .

# Build the Go app
RUN go build -o torrent-tracker

# Start a new stage from scratch
FROM alpine:latest

# Set the current working directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/torrent-tracker .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./torrent-tracker"]
