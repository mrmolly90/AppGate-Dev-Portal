import React from 'react'
import Skeleton from '../components/Skeleton.jsx'
import { useBffHealth, useGateways, useSecretStatus } from '../hooks/useDevPortal.js'

/**
 * Dashboard — management view showing live gateway registrations,
 * health status, and credential lifecycle. Real-time data from the control plane.
 */
export default function Dashboard() {
  const { health } = useBffHealth()
  const { gateways, loading: gwLoading, error: gwError } = useGateways()

  const totalActive = gateways.filter((g) => g.status === 'active').length
  const totalSpend = gateways.reduce((sum, g) => sum + (g.monthly_spend_usd || 0), 0)

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div className="animate-fade-in">
        <h2 className="text-[17px] font-semibold text-slate-100">Dashboard</h2>
        <p className="text-[13px] text-slate-400 mt-1">
          Live gateway management — real-time data from the control plane
        </p>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-3 gap-3">
        <StatCard label="Gateways" value={String(gateways.length)} accent="#22d3ee" loading={gwLoading} />
        <StatCard label="Active" value={String(totalActive)} accent="#34d399" loading={gwLoading} />
        <StatCard label="Budget/mo" value={`$${totalSpend.toLocaleString()}`} accent="#fbbf24" loading={gwLoading} />
      </div>

      {/* Health panel */}
      <div className="panel p-4">
        <div className="flex items-center justify-between">
          <span className="text-[12px] font-medium text-slate-400">System Health</span>
          <span className="flex items-center gap-2">
            <span
              className="w-2 h-2 rounded-full"
              style={{
                background:
                  health.status === 'healthy' ? '#34d399'
                  : health.status === 'loading' ? '#64748b'
                  : '#f87171',
              }}
            />
            <span className="text-[12px] font-medium text-slate-300">
              {health.status === 'healthy' ? 'Operational' : health.status === 'loading' ? 'Checking…' : 'Unreachable'}
            </span>
          </span>
        </div>
        <div className="mt-3 grid grid-cols-2 gap-2 text-[11px]">
          <div className="rounded-lg bg-slate-800/40 border border-slate-700/40 px-3 py-2 flex justify-between">
            <span className="text-slate-500">BFF</span>
            <span className="text-slate-300 font-medium">{health.component || 'dev-portal-bff'}</span>
          </div>
          <div className="rounded-lg bg-slate-800/40 border border-slate-700/40 px-3 py-2 flex justify-between">
            <span className="text-slate-500">Control Plane</span>
            <span className="text-slate-300 font-medium">{health.control_plane ? 'Up' : 'Down'}</span>
          </div>
        </div>
      </div>

      {/* Registered gateways — real-time list */}
      <div className="panel p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-[14px] font-semibold text-slate-100">Registered Gateways</h3>
          {!gwLoading && !gwError && (
            <span className="text-[11px] text-slate-500">{gateways.length} total</span>
          )}
        </div>

        {gwLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-14" />
            <Skeleton className="h-14" />
            <Skeleton className="h-14" />
          </div>
        ) : gwError ? (
          <div className="rounded-xl bg-[rgba(248,113,113,0.08)] border border-[rgba(248,113,113,0.25)] px-4 py-3 text-sm text-red-400">
            Failed to load gateways: {gwError}
          </div>
        ) : gateways.length === 0 ? (
          <div className="text-center py-10">
            <svg className="w-10 h-10 mx-auto text-slate-700 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="3" y="3" width="7" height="9" rx="1" />
              <rect x="14" y="3" width="7" height="5" rx="1" />
              <rect x="14" y="12" width="7" height="9" rx="1" />
              <rect x="3" y="16" width="7" height="5" rx="1" />
            </svg>
            <p className="text-[13px] text-slate-500">No gateways registered yet.</p>
            <p className="text-[11px] text-slate-600 mt-1">Use the Register tab to issue your first credentials.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {gateways.map((gw) => (
              <GatewayRow key={gw.id} gw={gw} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StatCard({ label, value, accent, loading }) {
  return (
    <div className="panel p-3.5 text-center">
      {loading ? (
        <Skeleton className="h-6 w-16 mx-auto" />
      ) : (
        <div className="text-[18px] font-bold tabular-nums" style={{ color: accent }}>
          {value}
        </div>
      )}
      <div className="text-[10px] text-slate-500 uppercase tracking-wider mt-1">{label}</div>
    </div>
  )
}

function GatewayRow({ gw }) {
  const { data: status } = useSecretStatus(gw.id)
  const displayStatus = status?.status || gw.status || 'active'

  return (
    <div className="flex items-center justify-between px-3.5 py-3 rounded-xl bg-slate-800/30 border border-slate-700/40 animate-slide-up">
      <div className="flex items-center gap-3 min-w-0">
        <span
          className="w-2 h-2 rounded-full flex-shrink-0"
          style={{
            background: displayStatus === 'active' ? '#34d399' : displayStatus === 'expired' ? '#fbbf24' : '#f87171',
            boxShadow: `0 0 6px ${displayStatus === 'active' ? '#34d399' : '#f87171'}`,
          }}
        />
        <div className="min-w-0">
          <p className="text-[13px] font-medium text-slate-200 truncate">{gw.project_name || gw.id}</p>
          <p className="text-[11px] text-slate-500">{gw.model} · {gw.id}</p>
        </div>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0 ml-3">
        {displayStatus === 'active' ? (
          <span className="badge badge-success">{displayStatus}</span>
        ) : displayStatus === 'expired' ? (
          <span className="badge badge-warning">{displayStatus}</span>
        ) : (
          <span className="badge badge-danger">{displayStatus}</span>
        )}
        {gw.expires_at && (
          <span className="text-[10px] text-slate-500 hidden xs:inline">
            {new Date(gw.expires_at).toLocaleDateString()}
          </span>
        )}
      </div>
    </div>
  )
}