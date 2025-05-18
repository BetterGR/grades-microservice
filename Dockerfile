# syntax=docker/dockerfile:1

# Base image with Go
FROM golang:1.24.3-alpine AS base

# Install build dependencies
RUN apk --no-cache add git

# Set work directory
WORKDIR /app

# =====================
# Dependencies stage
# =====================
FROM base AS deps

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# =====================
# Build stage
# =====================
FROM base AS builder

# Copy dependencies
COPY --from=deps /go/pkg /go/pkg

# Copy the source code
COPY . .

# Build Go binary
WORKDIR /app/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /microservice

# =====================
# Production stage
# =====================
FROM alpine:latest

# Set executable binary
COPY --from=builder /microservice /microservice

# Expose the server port
EXPOSE 50051

# Run the Go application
ENTRYPOINT ["/microservice"]