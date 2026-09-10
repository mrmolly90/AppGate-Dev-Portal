import React from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { useBffHealth } from '../hooks/useDevPortal.js'

/**
 * Layout — persistent sidebar + main content area with health indicator.
 * Uses glassmorphism panels and cyan-accented navigation.
 */
export default function Layout() {
  const { health } = useBffHealth()

  const navItems = [
    { to: '/', label: 'Dashboard', icon: DashboardIcon },
    { to: '/api-keys', label: 'API Keys', icon: KeyIcon },
    { to: '/policies', label: 'Policies', icon: ShieldIcon },
  ]

  const healthColor =
    health.status === 'healthy'
      ? 'bg-emerald-400'
      : health.status === 'loading'
      ? 'bg-slate-500'
      : 'bg-red-400'

  const healthLabel =
    health.status === 'healthy'
      ? 'All systems operational'
      : health.status === 'loading'
      ? 'Checking connection…'
      : 'Backend unreachable — fail closed'

  return (
    <div className="flex h-screen bg-slate-950 overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 flex-shrink-0 border-r border-slate-800/60 bg-slate-900/40 backdrop-blur-lg flex flex-col">
        {/* Logo */}
        <div className="h-16 flex items-center gap-3 px-5 border-b border-slate-800/40">
          <div className="w-9 h-9 rounded-lg bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center shadow-sm shadow-cyan-500/10">
            <span className="text-cyan-400 font-bold text-sm tracking-tight">AG</span>
          </div>
          <div>
            <h1 className="text-sm font-semibold text-slate-100 leading-tight">AppGate</h1>
            <p className="text-[11px] text-slate-500 leading-tight">Dev Portal</p>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `nav-link ${isActive ? 'nav-link-active' : ''}`
              }
            >
              <item.icon className="w-5 h-5 flex-shrink-0" />
              {item.label}
            </NavLink>
          ))}
        </nav>

        {/* Health indicator */}
        <div className="px-4 py-4 border-t border-slate-800/40">
          <div className="flex items-center gap-2.5">
            <span className="relative flex h-2.5 w-2.5">
              <span
                className={`absolute inline-flex h-full w-full rounded-full opacity-75 animate-ping ${healthColor}`}
              />
              <span className={`relative inline-flex h-2.5 w-2.5 rounded-full ${healthColor}`} />
            </span>
            <div className="flex-1 min-w-0">
              <p className="text-[11px] font-medium text-slate-400 truncate">{healthLabel}</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        <div className="route-enter p-6 lg:p-8 max-w-5xl mx-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inline SVG icons (no dependency)
// ---------------------------------------------------------------------------

function DashboardIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
      <rect x="14" y="14" width="7" height="7" rx="1" />
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