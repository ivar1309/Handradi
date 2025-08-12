# Stage 1 — Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy Go modules first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the server binary
RUN go build -o server ./cmd/server/server.go

# Build the CLI binary
RUN go build -o cli ./cmd/cli/cli.go

# Stage 2 — Runtime
FROM alpine

# Create app folder
WORKDIR /app

# Copy binaries
COPY --from=builder /app/server ./server
COPY --from=builder /app/cli ./cli

# Create storage folder
RUN mkdir -p /app/storage

# Default command — run the server
CMD ["/app/server"]
