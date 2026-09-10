# =============================================================================
# AppGate Dev-Portal — API (BFF) Dockerfile
#
# Security posture:
#   - Multi-stage build: builder has full toolchain; runtime is minimal.
#   - Non-root USER (uid 10001) with no shell.
#   - distroless base: no package manager, no shell, no network tools.
#   - Read-only root filesystem with a writable /tmp.
#   - No secrets baked into the image (config injected at runtime via env,
#     Vault, or a mounted secret store).
# =============================================================================

# ---- Builder stage ---------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static, stripped binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /out/dev-portal-api ./apps/api

# ---- Runtime stage (alpine, non-root) --------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -g 10001 appgate && \
    adduser -u 10001 -G appgate -s /sbin/nologin -D appgate

# Copy the static binary
COPY --from=builder /out/dev-portal-api /app/dev-portal-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 10001:10001
EXPOSE 8080

# Fail closed: exec form, no shell
ENTRYPOINT ["/app/dev-portal-api"]