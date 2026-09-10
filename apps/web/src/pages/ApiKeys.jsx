import React from 'react'
import Skeleton from '../components/Skeleton.jsx'
import { useGateways } from '../hooks/useDevPortal.js'

/**
 * ApiKeys — displays registered gateway secrets and their lifecycle status.
 * Each key row shows status, model, creation date, and expiry.
 */
export default function ApiKeys() {
  const { gateways, loading, error, refresh } = useGateways()

  return (
    <div className="space-y-8">
      {/* Page header */}
      <div className="flex items-start justify-between animate-fade-in">
        <div>
          <h2 className="text-xl font-semibold text-slate-100">API Keys</h2>
          <p className="text-sm text-slate-400 mt-1">
            Manage issued gateway credentials and their lifecycle status
          </p>
        </div>
        <button
          onClick={refresh}
          disabled={loading}
          className="px-4 py-2 rounded-lg glass-button text-sm flex items-center gap-2"
        >
          <svg className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
          {loading ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>

      {/* Keys table */}
      <div className="glass-card p-0 overflow-hidden">
        {loading && gateways.length === 0 ? (
          <div className="p-6 space-y-4">
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
          </div>
        ) : error ? (
          <div className="p-6">
            <div className="rounded-lg bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400">
              Failed to load API keys: {error}
            </div>
          </div>
        ) : gateways.length === 0 ? (
          <div className="p-6 text-center py-12">
            <svg className="w-12 h-12 mx-auto text-slate-700 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="8" cy="21" r="1" /><circle cx="16" cy="21" r="1" />
              <path d="M3 3h18l-1.68 10.08A2 2 0 0 1 17.34 15H6.66a2 2 0 0 1-1.98-1.92L3 3z" />
            </svg>
            <p className="text-sm text-slate-500">No API keys issued yet.</p>
            <p className="text-xs text-slate-600 mt-1">Register a gateway from the Dashboard first.</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-700/40">
                <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Client ID</th>
                <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Model</th>
                <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Status</th>
                <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Created</th>
                <th className="text-left px-6 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Expires</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/40">
              {gateways.map((gw, i) => (
                <tr key={gw.id} className={`hover:bg-slate-800/30 transition-colors animate-fade-in`} style={{ animationDelay: `${i * 50}ms` }}>
                  <td className="px-6 py-3.5 font-mono text-xs text-slate-200">{gw.id}</td>
                  <td className="px-6 py-3.5 text-slate-300">{gw.model}</td>
                  <td className="px-6 py-3.5">
                    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                      gw.status === 'active' ? 'text-emerald-400' : 'text-slate-400'
                    }`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${
                        gw.status === 'active' ? 'bg-emerald-400' : 'bg-slate-500'
                      }`} />
                      {gw.status}
                    </span>
                  </td>
                  <td className="px-6 py-3.5 text-slate-400 text-xs">
                    {new Date(gw.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-3.5 text-slate-400 text-xs">
                    {new Date(gw.expires_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Info card */}
      <div className="glass-card">
        <h3 className="text-sm font-semibold text-slate-100 mb-2">About API Key Lifecycle</h3>
        <ul className="space-y-2 text-sm text-slate-400">
          <li className="flex items-start gap-2">
            <span className="text-cyan-400 mt-0.5">•</span>
            <span>Keys are issued with a default 24-hour expiry. Configure via <code className="font-mono text-xs text-cyan-400">CP_JWT_TOKEN_EXPIRY</code>.</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-cyan-400 mt-0.5">•</span>
            <span>Revoked or expired keys can no longer authenticate with the data plane.</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-cyan-400 mt-0.5">•</span>
            <span>AppGate never stores the raw JWT after issuance. Store it securely.</span>
          </li>
        </ul>
      </div>
    </div>
  )
}