FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install required dependencies
RUN apk add --no-cache bash zsh

# Copy go.mod and go.sum first for caching dependencies
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o server .

FROM alpine:latest

WORKDIR /app

# Install bash and zsh (required for command execution)
RUN apk add --no-cache bash zsh

# Copy the built binary from the builder stage
COPY --from=builder /app/server ./

# Run with a default of no allowed commands
ENTRYPOINT ["./server"]
CMD ["--allowed-commands=echo,ls,cat,pwd"]
