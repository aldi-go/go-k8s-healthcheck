# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.24-alpine AS builder

# Install CA certificates for HTTPS calls (if needed downstream)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy dependency files first — leverages Docker layer cache
COPY go.mod ./
RUN go mod download

# Copy source and build a fully static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION:-dev}" \
    -o /app/server \
    ./cmd/server

# =============================================================================
# Stage 2: Final — distroless (no shell, no package manager, non-root)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and CA certs from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/server /server

# Distroless nonroot image already runs as uid=65532
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
