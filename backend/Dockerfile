# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN apk add --no-cache build-base git

# Enable Go modules and allow toolchain auto-upgrade if needed
ENV GO111MODULE=on \
    GOTOOLCHAIN=auto

# First copy just go.mod for better layer caching (avoid requiring go.sum)
COPY go.mod ./

# Use Go proxy for reproducible checksums and avoid mirror drift
ENV GOPROXY=https://proxy.golang.org,direct

# Download deps based on go.mod (cache layers)
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Now copy the rest of the backend source code
COPY . ./
# Populate go.sum for all imported packages (no codegen)
RUN go mod tidy

# Build api binary (verbose to surface errors in Docker logs)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /out/api ./cmd/api

# --- Runtime stage ---
FROM gcr.io/distroless/base-debian12
WORKDIR /app

# Non-root user
USER 65532:65532

# Copy binary and migrations
COPY --from=builder /out/api /app/api
# Note: migrations are embedded via go:embed in the binary, but keep a copy for visibility.
COPY internal/db/migrations /app/db/migrations

# App listens on 8080
EXPOSE 8080

# Environment defaults (overridden by compose)
ENV BACKEND_PORT=8080

ENTRYPOINT ["/app/api"]
