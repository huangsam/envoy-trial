# Use the official Golang image as the base image
FROM golang:1.24-bookworm

# Set the working directory inside the container
WORKDIR /app
COPY . /app

ENV CGO_ENABLED=0
ENV GOOS=linux

# Copy the Go module files and download dependencies
RUN go mod download

# Build the Go application
RUN go build -o main main.go

EXPOSE 161

# Set the entry point for the container
CMD ["./main"]
