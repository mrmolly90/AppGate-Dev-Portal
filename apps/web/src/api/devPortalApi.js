/**
 * Dev-Portal BFF API Client
 *
 * Production-grade fetch-based client for the AppGate Dev-Portal BFF.
 * No mock data — all calls go to the real backend.
 *
 * Security:
 *  - Provider keys are NEVER logged or stored client-side.
 *  - Credentials are returned once and must be discarded after display.
 *  - Cache-Control: no-store on all credential-bearing requests.
 *  - X-Request-Id correlation header on every request.
 */

const API_BASE = import.meta.env.VITE_API_BASE || ''

// ---------------------------------------------------------------------------
// Error class
// ---------------------------------------------------------------------------

class ApiError extends Error {
  constructor(message, code, status) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

async function apiFetch(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: {
      Accept: 'application/json',
      'X-Request-Id': crypto.randomUUID(),
      ...options.headers,
    },
    cache: 'no-store',
    ...options,
  })

  const body = await res.json().catch(() => null)

  if (!res.ok) {
    const err = new ApiError(
      body?.message || `Request failed (HTTP ${res.status})`,
      body?.error || 'unknown',
      res.status
    )
    throw err
  }

  return body
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Submit a tenant registration to the BFF.
 * Returns { clientId, clientSecretId, jwtToken, expiresAt, tokenType, issuer }
 */
export async function registerTenant(payload) {
  const body = await apiFetch('/v1/tenant/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      project_name: payload.projectName,
      model: payload.model,
      llm_model_name: payload.llmModelName,
      provider_key: payload.providerKey,
      provider_url: payload.providerUrl,
      monthly_spend_usd: payload.monthlySpendUSD,
      rate_limit_rps: payload.rateLimitRPS ?? 1000,
      rate_limit_rpm: payload.rateLimitRPM ?? 60000,
      rate_limit_burst: payload.rateLimitBurst ?? 2000,
      webhook_url: payload.webhookURL ?? '',
      tags: payload.tags ?? {},
    }),
  })

  if (!body || !body.jwt_token || !body.client_secret_id || !body.client_id) {
    throw new Error('Invalid response from server: missing credentials')
  }

  return {
    clientId: body.client_id,
    clientSecretId: body.client_secret_id,
    jwtToken: body.jwt_token,
    expiresAt: body.expires_at,
    tokenType: body.token_type || 'Bearer',
    issuer: body.issuer || 'appgate',
  }
}

/**
 * Query the lifecycle status of an issued secret.
 */
export async function fetchSecretStatus(clientId) {
  const body = await apiFetch(`/v1/tenant/status/${encodeURIComponent(clientId)}`)
  return body
}

/**
 * Health check for the BFF.
 */
export async function fetchHealth() {
  const body = await apiFetch('/health')
  return body
}

/**
 * List all registered gateways from the control plane via the BFF.
 */
export async function fetchGateways() {
  const body = await apiFetch('/v1/gateways')
  return (body?.gateways || []).map((gw) => ({
    id: gw.id || gw.client_id,
    model: gw.model,
    status: gw.status,
    created_at: gw.created_at,
    expires_at: gw.expires_at,
    provider_url: gw.provider_url,
    monthly_spend_usd: gw.monthly_spend_usd,
    rate_limit_rps: gw.rate_limit_rps,
    tags: gw.tags,
  }))
}