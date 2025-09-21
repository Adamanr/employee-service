FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd
RUN ls -l /app/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/employes_service .

FROM alpine:latest
RUN apk add --no-cache postgresql-client ca-certificates wget netcat-openbsd

WORKDIR /app

# Download goose binary directly for the correct architecture
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        GOOSE_ARCH="x86_64"; \
    elif [ "$ARCH" = "aarch64" ]; then \
        GOOSE_ARCH="arm64"; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi && \
    wget -O /usr/local/bin/goose "https://github.com/pressly/goose/releases/download/v3.25.0/goose_linux_${GOOSE_ARCH}" && \
    chmod +x /usr/local/bin/goose

# Copy application files
COPY --from=builder /app/employes_service .
COPY migrations /app/migrations
COPY configs/config.toml /app/config.toml

# Create startup script
COPY <<'EOF' /app/startup.sh
#!/bin/sh
set -e

# Configuration
MAX_RETRIES=30
RETRY_DELAY=2
DB_TIMEOUT=60
REDIS_TIMEOUT=30

echo "=== Starting application setup ==="

# Function to wait for service
wait_for_service() {
    local host=$1
    local port=$2
    local service_name=$3
    local timeout=$4
    local retries=0
    
    echo "Waiting for $service_name at $host:$port..."
    while [ $retries -lt $timeout ]; do
        if nc -z "$host" "$port"; then
            echo "$service_name is ready!"
            return 0
        fi
        echo "Waiting for $service_name... (attempt $((retries + 1))/$timeout)"
        sleep $RETRY_DELAY
        retries=$((retries + 1))
    done
    
    echo "ERROR: $service_name is not available after ${timeout} attempts" >&2
    return 1
}

# Wait for database
if ! wait_for_service "${DB_HOST:-postgres}" "${DB_PORT:-5432}" "Database" $DB_TIMEOUT; then
    exit 1
fi

# Wait for Redis (if configured)
if [ -n "$REDIS_HOST" ]; then
    if ! wait_for_service "$REDIS_HOST" "${REDIS_PORT:-6379}" "Redis" $REDIS_TIMEOUT; then
        echo "WARNING: Redis is not available, continuing without it..."
    fi
fi

# Build database URL
DB_URL="postgres://${DB_USER:-postgres}:${DB_PASSWORD:-admin21}@${DB_HOST:-postgres}:${DB_PORT:-5432}/${DB_NAME:-employee_dev}?sslmode=disable"

# Run database migrations with retries
echo "=== Running database migrations ==="
migration_retries=0
while [ $migration_retries -lt 5 ]; do
    if goose -dir /app/migrations postgres "$DB_URL" up; then
        echo "✓ Migrations completed successfully!"
        break
    else
        migration_retries=$((migration_retries + 1))
        if [ $migration_retries -lt 5 ]; then
            echo "Migration attempt $migration_retries failed, retrying in ${RETRY_DELAY}s..."
            sleep $RETRY_DELAY
        else
            echo "ERROR: Migration failed after 5 attempts!" >&2
            exit 1
        fi
    fi
done

# Show migration status
echo "=== Current migration status ==="
goose -dir /app/migrations postgres "$DB_URL" status || echo "Could not get migration status"

# Start the application
echo "=== Starting application ==="
exec ./employes_service
EOF

RUN chmod +x /app/startup.sh

# Verify goose works
RUN /usr/local/bin/goose --version

EXPOSE 8080
CMD ["/app/startup.sh"]