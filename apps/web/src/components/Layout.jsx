import React, { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { useBffHealth } from '../hooks/useDevPortal.js'

/**
 * Layout — Android-frame shell with status bar, scrollable screen, bottom nav.
 * Desktop shows a framed mobile device; mobile is full-bleed.
 */
export default function Layout() {
  const { health } = useBffHealth()
  const [now, setNow] = useState(new Date())

  // Live clock in status bar
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  const dateStr = now.toLocaleDateString([], { month: 'short', day: 'numeric' })

  const navItems = [
    { to: '/', label: 'Register', icon: RegistrationIcon },
    { to: '/dashboard', label: 'Dashboard', icon: DashboardIcon },
    { to: '/api-keys', label: 'API Keys', icon: KeyIcon },
    { to: '/policies', label: 'Policies', icon: ShieldIcon },
  ]

  const healthColor =
    health.status === 'healthy'
      ? '#34d399'
      : health.status === 'loading'
      ? '#64748b'
      : '#f87171'

  const healthLabel =
    health.status === 'healthy'
      ? 'Live'
      : health.status === 'loading'
      ? 'Connecting…'
      : 'Offline'

  return (
    <div className="app-shell">
      <div className="phone-frame">
        {/* Android-style status bar */}
        <header className="status-bar">
          <div className="flex items-center gap-2">
            <span className="w-7 h-7 rounded-lg bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center shadow-sm">
              <span className="text-cyan-400 font-bold text-[11px] tracking-tight">AG</span>
            </span>
            <div>
              <div className="text-[12px] font-semibold text-slate-100 leading-none">AppGate</div>
              <div className="text-[9px] text-slate-500 leading-none mt-0.5">Dev Portal</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-[10px] text-slate-400 hidden xs:inline">{healthLabel}</span>
            <span className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full" style={{ background: healthColor, boxShadow: `0 0 8px ${healthColor}` }} />
              <span className="text-[13px] font-semibold text-slate-100 tabular-nums">{timeStr}</span>
            </span>
          </div>
        </header>

        {/* Scrollable screen (Android content) */}
        <main className="phone-screen">
          <div className="px-4 py-4 pb-8 route-enter">
            {/* Date strip */}
            <div className="flex items-center justify-between mb-3">
              <span className="text-[11px] text-slate-500 font-medium uppercase tracking-wider">{dateStr}</span>
              <span className="text-[11px] text-cyan-400/80 font-medium flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full" style={{ background: healthColor }} />
                Control plane {healthLabel.toLowerCase()}
              </span>
            </div>
            <Outlet />
          </div>
        </main>

        {/* Android-style bottom navigation */}
        <nav className="bottom-nav">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `bottom-nav-item ${isActive ? 'active' : ''}`
              }
            >
              <item.icon className="bottom-nav-icon w-5 h-5" />
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inline SVG icons (no dependency)
// ---------------------------------------------------------------------------

function RegistrationIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 20h9" />
      <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />
    </svg>
  )
}

function DashboardIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="9" rx="1" />
      <rect x="14" y="3" width="7" height="5" rx="1" />
      <rect x="14" y="12" width="7" height="9" rx="1" />
      <rect x="3" y="16" width="7" height="5" rx="1" />
    </svg>
  )
}

function KeyIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="8" cy="21" r="1" />
      <circle cx="16" cy="21" r="1" />
      <path d="M3 3h18l-1.68 10.08A2 2 0 0 1 17.34 15H6.66a2 2 0 0 1-1.98-1.92L3 3z" />
      <path d="M7 21l-1-6h12l-1 6" />
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