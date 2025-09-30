# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o image-sync-backend ./cmd/server

# Final stage
FROM alpine:latest

# Set the commit ID for version tracking
ARG commit_id=dev
ENV COMMIT_ID=${commit_id}

# Install skopeo and set up non-root user (UID 65532)
RUN apk add --no-cache skopeo ca-certificates \
	&& addgroup -g 65532 -S nonroot \
	&& adduser -u 65532 -S nonroot -G nonroot

WORKDIR /app

# Copy the binary from builder
COPY --from=builder --chown=65532:65532 /app/image-sync-backend /app/image-sync-backend

# Prepare config directory with restrictive permissions
RUN mkdir -p /configs \
	&& chown 65532:65532 /configs \
	&& chmod 700 /configs

USER 65532:65532

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["./image-sync-backend"]
