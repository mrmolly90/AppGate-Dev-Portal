# =============================================================================
# AppGate Dev-Portal — Web (React) Dockerfile
#
# Multi-stage: builds with Node, serves with nginx (non-root).
# The built assets talk to the BFF over a reverse proxy path (/v1).
# =============================================================================

# ---- Builder stage ---------------------------------------------------------
FROM node:22-alpine AS builder

WORKDIR /app

COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm install --no-audit --no-fund --loglevel=error

COPY apps/web/ .
RUN npm run build

# ---- Runtime stage (nginx, non-root) --------------------------------------
FROM nginxinc/nginx-unprivileged:1.27-alpine

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Minimal nginx config: SPA + API proxy to BFF service
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf

# nginx-unprivileged runs on port 8080 as uid 101
EXPOSE 8080

USER 101

CMD ["nginx", "-g", "daemon off;"]