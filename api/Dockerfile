# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# Use CGO_ENABLED=1 if using SQLite with CGO, but here we prefer pure Go sqlite if possible.
# Requirement says modernc.org/sqlite, so CGO_ENABLED=0 is fine.
RUN CGO_ENABLED=0 GOOS=linux go build -o opus ./cmd/opus

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/opus .

# Create directory for config and db
RUN mkdir -p /root/.opus

EXPOSE 8080

ENTRYPOINT ["./opus", "start"]
