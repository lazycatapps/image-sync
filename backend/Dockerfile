# syntax=docker/dockerfile:1

ARG commit_id=dev

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o image-sync-server ./cmd/server

FROM alpine:latest AS runtime-base

ARG commit_id
ENV COMMIT_ID=${commit_id}

# Install skopeo and configure non-root user
RUN apk add --no-cache skopeo ca-certificates \
    && addgroup -g 65532 -S nonroot \
    && adduser -u 65532 -S nonroot -G nonroot

WORKDIR /app

# Prepare config directory with restrictive permissions
RUN mkdir -p /configs \
    && chown 65532:65532 /configs \
    && chmod 700 /configs

USER 65532:65532

# Set HOME environment variable for non-root user
ENV HOME=/home/nonroot

EXPOSE 8080

ENTRYPOINT ["./image-sync-server"]

FROM runtime-base AS dev

# Copy pre-built binary from local build context
COPY --chown=nonroot:nonroot image-sync-server /app/image-sync-server

FROM runtime-base AS prod

# Copy binary from builder stage for production image
COPY --from=builder --chown=nonroot:nonroot /app/image-sync-server /app/image-sync-server
