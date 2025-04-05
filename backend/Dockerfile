# ---- Build Stage ----
# Use a specific Go version for reproducibility. Alpine variants are smaller.
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies if needed (e.g., for CGO)
# RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum first to leverage Docker layer caching
COPY go.mod go.sum ./

# Download dependencies. This layer is cached if go.mod/go.sum haven't changed.
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go application
# - CGO_ENABLED=0: Build statically linked binary without CGO (recommended for alpine)
# - ldflags="-w -s": Reduce binary size by removing debug info (-w) and symbol table (-s)
# -o /app/server: Output the executable named 'server' in the /app directory
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server .

# ---- Runtime Stage ----
# Use a minimal base image like Alpine
FROM alpine:latest

# Install runtime dependencies if any (e.g., CA certificates for HTTPS, timezone data)
RUN apk add --no-cache ca-certificates tzdata

# Set the working directory
WORKDIR /app

# Create a non-root user and group for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the compiled binary from the builder stage
# Change ownership to the non-root user
COPY --from=builder --chown=appuser:appgroup /app/server /app/server

# Set default environment variables
# PORT defaults to 8000 if not provided externally
ENV PORT=8000
# Assume Production environment by default inside the container
# This prevents loading .env file unless overridden
ENV ENV=PROD
# DB_URL MUST be provided at runtime

# Expose the port the application listens on (using the ENV variable)
# Note: This is documentation; use `docker run -p` to actually publish the port.
EXPOSE ${PORT}

# Switch to the non-root user
USER appuser

# Add a basic healthcheck using the /health endpoint
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Command to run the application
# Use the absolute path for clarity
CMD ["/app/server"]