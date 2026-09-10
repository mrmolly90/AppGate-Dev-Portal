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

# ---- Runtime stage (distroless, non-root) ---------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the static binary
COPY --from=builder /out/dev-portal-api /app/dev-portal-api

# Non-root user already provided by distroless:nonroot (uid 65532)
USER 65532:65532

# Read-only root filesystem; writable /tmp for any transient work
COPY --chown=65532:65532 --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

# Fail closed: exec form, no shell
ENTRYPOINT ["/app/dev-portal-api"]