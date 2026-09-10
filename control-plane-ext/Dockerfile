# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY control-plane-ext/go.mod control-plane-ext/go.sum ./
RUN go mod download

COPY control-plane-ext/ .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /out/control-plane ./cmd/server

# Runtime stage — minimal alpine, non-root
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -g 10001 appgate && \
    adduser -u 10001 -G appgate -s /sbin/nologin -D appgate

COPY --from=builder /out/control-plane /app/control-plane
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 10001:10001
EXPOSE 8081
ENTRYPOINT ["/app/control-plane"]