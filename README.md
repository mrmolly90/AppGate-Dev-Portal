# AppGate Dev-Portal

**Enterprise BFF for LLM Gateway Credential Issuance**

The Dev-Portal is a Backend-For-Frontend (BFF) management interface that delegates core security tasks to the [AppGate Control Plane](https://github.com/mrmolly90/AppGate). It implements strict separation of concerns between the control plane and data plane.

## Architecture

```
┌──────────────┐     ┌──────────────────┐     ┌───────────────────┐     ┌──────────────┐
│   React SPA  │────→│  Go BFF (BFF)   │────→│ Go Control Plane  │────→│ Rust Data    │
│  (Portal UI) │     │  apps/api        │     │ (JWT signing)     │     │ Plane        │
│              │     │  :8080           │     │ :8081             │     │ (Gateway)    │
└──────────────┘     └──────────────────┘     └───────────────────┘     └──────────────┘
     Web                    Internal REST            Auth/JWT                   Proxy
    (HTTPS)                 (mTLS/token)              RS256/ES256               (tokio)
```

## Flow

1. **Developer** submits provider credentials, model selection, and spend limits via the React UI
2. **BFF** validates payloads, never logs secrets, forwards to control plane
3. **Control Plane** authenticates, signs a JWT with RSA-2048/ES256, returns `{ client_id, client_secret_id, jwt_token }`
4. **SecretModal** displays credentials once — ephemeral, auto-masking, copy-to-clipboard

## Project Structure

```
appgate-dev-portal/
├── apps/
│   ├── api/                    # Go BFF (HTTP handlers, middleware, services)
│   │   ├── config/             # Env-based config loader
│   │   ├── handlers/           # /v1/tenant/register, /v1/gateways, /health
│   │   ├── middleware/         # Rate limiter, security headers, session auth, CORS
│   │   └── services/           # Orchestration — registration, status, gateway listing
│   └── web/                    # React SPA (Vite + Tailwind)
│       └── src/
│           ├── api/            # Production fetch-based API client (no mock data)
│           ├── components/     # SecretModal, RegistrationForm, BudgetSlider, Layout
│           ├── hooks/          # useRegistration, useSecretStatus, useBffHealth, useGateways
│           └── pages/          # Dashboard, ApiKeys, Policies
├── control-plane-ext/          # AppGate Go Control Plane (JWT signing server)
│   └── internal/
│       ├── api/                # Gin HTTP server, gateway store
│       ├── config/             # Env-based config
│       └── jwt/                # RS256/ES256 JWT signer
├── pkg/
│   ├── controlplane/           # Internal REST client for the control plane
│   └── security/               # Masking, token parsing, SSRF validation
├── deploy/
│   ├── docker/                 # Multi-stage Dockerfiles, docker-compose
│   └── terraform/              # AWS ECS Fargate provisioning (private subnets only)
├── .github/workflows/
│   ├── ci.yml                  # Lint, test, build Go + Web + Docker
│   └── cd.yml                  # Push Docker images to ghcr.io
├── Makefile
└── .env.example
```

## Security Properties

| Principle | Implementation |
|-----------|---------------|
| **Zero Trust** | Every request is validated at each layer; no implicit trust between BFF, CP, or client |
| **Fail Closed** | All validation errors, transport failures, and non-2xx responses deny access |
| **Least Privilege** | Each SG/NACL allows only the minimum required traffic; no `0.0.0.0/0` egress |
| **No Secrets in Logs** | Provider keys and JWTs are NEVER logged — only digests and identifiers |
| **Ephemeral Credentials** | JWT shown once in an auto-masking modal; never persisted to localStorage/cookies |
| **SSRF Defense** | Upstream URLs forced to HTTPS; private/local host resolution blocked |
| **Input Validation** | Payload size limited (1MB), content-type enforced, fields length-bounded |

## Quick Start (Development)

```bash
# Prerequisites: Go 1.22+, Node 22+, Docker

# 1. Start the control plane
cd control-plane-ext
go run ./cmd/server/main.go &

# 2. Start the BFF
cd ..
$env:DP_CP_BASE_URL="http://localhost:8081"
$env:DP_CP_TLS_INSECURE="true"
$env:DP_ALLOWED_ORIGINS="http://localhost:3000"
go run ./apps/api

# 3. Start the web portal
cd apps/web
npm install && npm run dev
# → http://localhost:3000
```

## Docker Compose (Full Stack)

```bash
docker compose -f deploy/docker/docker-compose.yml up --build
```

## Register a Gateway (API Test)

```bash
curl -s -X POST http://localhost:8080/v1/tenant/register \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-4o",
    "provider_key": "sk-super-secret-key-123456",
    "provider_url": "https://api.openai.com",
    "monthly_spend_usd": 500,
    "rate_limit_rps": 100
  }' | jq
```

## Run Tests

```bash
# Go unit + integration tests (24 tests)
go test -v -count=1 ./...

# Web build
cd apps/web && npm run build
```

## CI/CD

- **CI** (`.github/workflows/ci.yml`): Runs on every push/PR — lints Go, runs tests, builds Go binaries, builds React, verifies Docker images
- **CD** (`.github/workflows/cd.yml`): On push to `main` — builds and pushes Docker images to GitHub Container Registry (ghcr.io)

## Deployment

See `deploy/terraform/main.tf` for the AWS ECS Fargate provisioning:
- **VPC**: Private subnets only (no blanket `0.0.0.0/0` egress)
- **Security Groups**: Each service locked to its callers only
- **Secrets Manager**: JWT signing key and provider credentials
- **No public subnets** for application workloads