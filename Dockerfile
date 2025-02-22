# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
# Use the official lightweight Golang Alpine image as the build environment.
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker layer caching.
# This ensures that dependency downloads are only re-run when these files change.
COPY go.mod go.sum ./

# Download all dependencies.
RUN go mod download

# Copy the entire source code into the container.
# This step should come after dependency download to optimize build speed.
COPY . .
COPY .env .env
RUN go mod tidy


# Build the application:
# - CGO_ENABLED=0 produces a statically-linked binary.
# - GOOS=linux ensures the binary is built for Linux.
# - The output binary is named 'api' and is built from the 'cmd/api' package.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o api ./cmd/api

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
# Use a minimal Alpine image to run the statically-linked binary.
FROM alpine:latest

# Install CA certificates and PostgreSQL client (for pg_isready)
RUN apk --no-cache add ca-certificates postgresql-client curl netcat-openbsd

# Install CA certificates, which are required for HTTPS requests.
RUN apk --no-cache add ca-certificates netcat-openbsd curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
RUN mv migrate /usr/local/bin/migrate

# Set the working directory for the final container.
WORKDIR /root/

# Copy the compiled binary from the builder stage into the runtime container.
COPY --from=builder /app/api .

#  copy the .env file for local development.
# COPY .env .
COPY --from=builder /app/.env .env


COPY --from=builder /app/migrations ./migrations
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
# Expose the port on which your application listens.
# Adjust the port if your application uses a different one.
EXPOSE 4000

# Specify the command to run the application.
ENTRYPOINT ["./entrypoint.sh"]
