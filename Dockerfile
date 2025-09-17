# This version is based on your go.mod file.
FROM golang:1.24-alpine

# Set the working directory inside the container.
WORKDIR /app

# Copy the files that define our project's dependencies.
COPY go.mod go.sum ./

# Download the dependencies.
RUN go mod download

# Copy the rest of your project's source code into the container.
COPY . .

# Compile your Go application.
RUN go build -o server ./cmd/server

# Expose port 8080 for the application.
EXPOSE 8080

# The command to run when the container starts.
CMD ["./server"]