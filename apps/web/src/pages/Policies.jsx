import React from 'react'

/**
 * Policies — displays gateway routing policies and rate limit configurations.
 * In a production AppGate deployment, these policies are synchronized from
 * the Control Plane's policy engine to the data plane.
 *
 * This page shows the current policy contract enforced by the gateway:
 * rate limits, spend ceilings, upstream allowlist, and audit rules.
 */
export default function Policies() {
  const policies = [
    {
      name: 'Rate Limiting',
      description: 'Per-identity token bucket rate limiting enforced by the data plane.',
      rules: [
        'Default: 1,000 req/s with 2,000 burst',
        'Configurable per gateway registration',
        'Enforced via GCRA (Generic Cell Rate Algorithm)',
      ],
      icon: GaugeIcon,
    },
    {
      name: 'Spend Ceiling',
      description: 'Hard monthly budget caps per upstream gateway.',
      rules: [
        'Set during registration (monthly_spend_usd)',
        'Tracked by the control plane audit system',
        'Automatic rate shaping when approaching limit',
      ],
      icon: DollarIcon,
    },
    {
      name: 'Upstream Allowlist',
      description: 'SSRF defense — only allowlisted endpoints receive traffic.',
      rules: [
        'Provider URL must use HTTPS',
        'Private/reserved IPs blocked at network level',
        'DNS resolution timeouts treated as unsafe (fail closed)',
      ],
      icon: ShieldIcon,
    },
    {
      name: 'Audit & Observability',
      description: 'Every proxied request is logged with metadata.',
      rules: [
        'Timestamps, identity, model, token count, latency',
        'Asynchronous audit events to control plane',
        'Prometheus metrics + OpenTelemetry tracing',
      ],
      icon: ActivityIcon,
    },
  ]

  return (
    <div className="space-y-8">
      {/* Page header */}
      <div className="animate-fade-in">
        <h2 className="text-xl font-semibold text-slate-100">Policies</h2>
        <p className="text-sm text-slate-400 mt-1">
          Gateway security policies enforced by the AppGate data plane
        </p>
      </div>

      {/* Policy cards grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {policies.map((policy, i) => (
          <div
            key={policy.name}
            className="glass-card animate-slide-up"
            style={{ animationDelay: `${i * 100}ms` }}
          >
            <div className="flex items-start gap-3 mb-3">
              <div className="p-2 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
                <policy.icon className="w-5 h-5 text-cyan-400" />
              </div>
              <div>
                <h3 className="text-sm font-semibold text-slate-100">{policy.name}</h3>
                <p className="text-xs text-slate-400 mt-0.5">{policy.description}</p>
              </div>
            </div>
            <ul className="space-y-1.5">
              {policy.rules.map((rule, j) => (
                <li key={j} className="flex items-start gap-2 text-xs text-slate-400">
                  <span className="text-cyan-400/60 mt-0.5">→</span>
                  <span>{rule}</span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      {/* Fail-closed note */}
      <div className="glass-card">
        <div className="flex items-start gap-3">
          <div className="p-2 rounded-lg bg-red-500/10 border border-red-500/20">
            <svg className="w-5 h-5 text-red-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100">Fail-Closed Enforcement</h3>
            <p className="text-xs text-slate-400 mt-1 leading-relaxed">
              All policy violations result in immediate request denial — no fallback to permissive defaults.
              The data plane evaluates policies before any upstream request is made. If the control plane
              is unreachable, the last-known-good policy set is cached and applied.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inline icons
// ---------------------------------------------------------------------------

function GaugeIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z" />
      <path d="M12 12l2.5-5" />
      <path d="M12 12a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5z" />
    </svg>
  )
}

function DollarIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="12" y1="1" x2="12" y2="23" /><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
    </svg>
  )
}

function ShieldIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  )
}

function ActivityIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  )
}