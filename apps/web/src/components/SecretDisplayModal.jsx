import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'

/**
 * SecretDisplayModal — Ephemeral Credential Display Modal
 *
 * High-security, non-dismissible (until acknowledged) modal that displays
 * the one-time credential handshake from the AppGate Control Plane.
 *
 * Security:
 *  - Credentials held ONLY in component state — never persisted.
 *  - JWT masked by default; eye icon reveals with auto-revert (15s).
 *  - Click-to-copy with "Copied!" visual feedback.
 *  - Closing the modal actively wipes all credential state.
 *  - NO localStorage, sessionStorage, or cookies are written.
 *  - Escape key requires explicit confirmation before closing.
 *  - Backdrop prevents accidental dismissal.
 *
 * UX:
 *  - Glassmorphism dark panel with cyan accent.
 *  - Stark red/orange warning banner: one-time display only.
 *  - "I have saved my credentials" button as the only close path.
 *  - Copy buttons with success state animation.
 *  - JWT payload preview (parsed claims).
 */

const REVEAL_MS = 15000
const COPY_OK_MS = 2000

function maskToken(token) {
  if (!token) return ''
  if (token.length <= 10) return '••••••••••'
  return `${token.slice(0, 8)}••••••••••${token.slice(-6)}`
}

function maskSecretId(id) {
  if (!id) return ''
  if (id.length <= 8) return '••••••••'
  return `${id.slice(0, 4)}••••••••${id.slice(-4)}`
}

async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    try {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(ta)
      return ok
    } catch {
      return false
    }
  }
}

function parseJwtPayload(jwt) {
  try {
    const parts = jwt.split('.')
    if (parts.length !== 3) return null
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
    const decoded = atob(padded)
    return JSON.parse(decoded)
  } catch {
    return null
  }
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function CopyField({ label, value, secret = true, visible, onToggle, onCopy, mask }) {
  const display = visible && value ? value : mask ? mask(value) : value || '—'
  return (
    <div className="mb-4 animate-fade-in">
      <div className="flex items-center justify-between mb-1.5">
        <label className="text-xs font-semibold uppercase tracking-wider text-slate-400">
          {label}
        </label>
        <div className="flex items-center gap-2">
          {secret && (
            <button
              type="button"
              onClick={onToggle}
              className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors duration-200 focus:outline-none"
              aria-label={visible ? 'Hide credential' : 'Reveal credential'}
            >
              {visible ? (
                <span className="flex items-center gap-1">
                  <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                    <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                  Hide
                </span>
              ) : (
                <span className="flex items-center gap-1">
                  <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                  Reveal
                </span>
              )}
            </button>
          )}
          <button
            type="button"
            onClick={() => onCopy(value)}
            className="text-xs px-2.5 py-1 rounded-md glass-button transition-all duration-200"
          >
            Copy
          </button>
        </div>
      </div>
      <code
        className="block w-full px-3.5 py-2.5 rounded-lg bg-slate-800/60 border border-slate-700/60 font-mono text-sm text-slate-200 break-all select-none-all transition-all duration-200"
        style={{ wordBreak: 'break-all' }}
      >
        {display}
      </code>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function SecretDisplayModal({ credentials, onClose }) {
  const [visibleField, setVisibleField] = useState(null)
  const [copiedField, setCopiedField] = useState(null)
  const [confirmClose, setConfirmClose] = useState(false)
  const [showPayload, setShowPayload] = useState(false)
  const [hasAcknowledged, setHasAcknowledged] = useState(false)
  const copyTimer = useRef(null)
  const revealTimer = useRef(null)

  // Auto re-mask after reveal duration
  useEffect(() => {
    if (visibleField) {
      revealTimer.current = setTimeout(() => setVisibleField(null), REVEAL_MS)
    }
    return () => {
      if (revealTimer.current) clearTimeout(revealTimer.current)
    }
  }, [visibleField])

  // Clear copy feedback timer
  useEffect(() => {
    return () => {
      if (copyTimer.current) clearTimeout(copyTimer.current)
    }
  }, [])

  // Prevent background scroll
  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [])

  const handleToggle = useCallback((field) => {
    setVisibleField((cur) => (cur === field ? null : field))
  }, [])

  const handleCopy = useCallback((value) => {
    copyToClipboard(value).then((ok) => {
      setCopiedField(ok ? 'ok' : 'fail')
      if (copyTimer.current) clearTimeout(copyTimer.current)
      copyTimer.current = setTimeout(() => setCopiedField(null), COPY_OK_MS)
    })
  }, [])

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      setConfirmClose(true)
    }
  }, [])

  // Wipe ALL credential state on close — zero trust enforcement
  const handleAcknowledgeAndClose = useCallback(() => {
    setVisibleField(null)
    setCopiedField(null)
    setHasAcknowledged(true)
    // Delay slightly to allow state to settle before parent wipes
    setTimeout(() => {
      onClose?.()
    }, 50)
  }, [onClose])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const expiryLabel = useMemo(() => {
    if (!credentials?.expiresAt) return ''
    try {
      return new Date(credentials.expiresAt).toLocaleString(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short',
      })
    } catch {
      return credentials.expiresAt
    }
  }, [credentials?.expiresAt])

  const jwtPayload = useMemo(() => {
    if (!credentials?.jwtToken) return null
    return parseJwtPayload(credentials.jwtToken)
  }, [credentials?.jwtToken])

  if (!credentials) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm p-4 animate-fade-in"
      role="dialog"
      aria-modal="true"
      aria-labelledby="secret-modal-title"
    >
      <div className="w-full max-w-xl glass-panel-strong rounded-xl animate-scale-in">
        {/* ================================================================ */}
        {/* HEADER                                                          */}
        {/* ================================================================ */}
        <div className="px-6 pt-5 pb-4 border-b border-slate-700/40">
          <div className="flex items-start justify-between">
            <div>
              <h2
                id="secret-modal-title"
                className="text-lg font-semibold text-slate-100 flex items-center gap-2"
              >
                <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-glow-pulse" />
                Gateway Credentials Issued
              </h2>
              <p className="text-sm text-slate-400 mt-1">
                Credential for{' '}
                <span className="text-cyan-400 font-mono">{credentials.clientId}</span>.
              </p>
            </div>
          </div>

          {/* ================================================================ */}
          {/* STARK WARNING — ONE-TIME DISPLAY                                */}
          {/* ================================================================ */}
          <div className="mt-3 rounded-lg bg-orange-500/10 border border-orange-500/30 px-4 py-3 animate-slide-up">
            <div className="flex items-start gap-2.5">
              <svg
                className="w-5 h-5 flex-shrink-0 mt-0.5 text-orange-400"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
              <div>
                <strong className="block text-sm text-orange-300 mb-0.5">
                  This JWT token will only be displayed once.
                </strong>
                <p className="text-xs text-orange-400/80 leading-relaxed">
                  Copy it now. <strong className="text-orange-300">AppGate does not store this token.</strong>{' '}
                  If you close this modal without saving, you will need to register a new gateway.
                </p>
              </div>
            </div>
          </div>

          {/* Confirm close warning */}
          {confirmClose && (
            <div className="mt-3 animate-slide-up flex items-center justify-between bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-2.5">
              <span className="text-xs text-red-400">
                Closing discards these credentials permanently.
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setConfirmClose(false)}
                  className="text-xs px-3 py-1.5 rounded-md glass-button transition-all duration-200"
                >
                  Keep open
                </button>
                <button
                  onClick={handleAcknowledgeAndClose}
                  className="text-xs px-3 py-1.5 rounded-md bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30 transition-all duration-200"
                >
                  Discard & close
                </button>
              </div>
            </div>
          )}
        </div>

        {/* ================================================================ */}
        {/* BODY — Credential Fields                                        */}
        {/* ================================================================ */}
        <div className="px-6 py-5">
          {/* JWT Token */}
          <CopyField
            label="JWT Token"
            value={credentials.jwtToken}
            secret
            visible={visibleField === 'jwt'}
            onToggle={() => handleToggle('jwt')}
            onCopy={handleCopy}
            mask={maskToken}
          />

          {/* Client Secret ID */}
          <CopyField
            label="Client Secret ID"
            value={credentials.clientSecretId}
            secret
            visible={visibleField === 'secret'}
            onToggle={() => handleToggle('secret')}
            onCopy={handleCopy}
            mask={maskSecretId}
          />

          {/* Client ID (non-secret, informational) */}
          <CopyField
            label="Client ID"
            value={credentials.clientId}
            secret={false}
            visible={false}
            onToggle={() => {}}
            onCopy={handleCopy}
            mask={null}
          />

          {/* JWT payload preview */}
          {jwtPayload && (
            <div className="mb-4">
              <button
                type="button"
                onClick={() => setShowPayload((p) => !p)}
                className="text-xs text-cyan-400/70 hover:text-cyan-400 transition-colors duration-200 flex items-center gap-1"
              >
                <svg
                  className={`w-3.5 h-3.5 transition-transform duration-200 ${
                    showPayload ? 'rotate-90' : ''
                  }`}
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                >
                  <polyline points="9 18 15 12 9 6" />
                </svg>
                JWT Claims Preview
              </button>
              {showPayload && (
                <pre className="mt-2 px-3.5 py-2.5 rounded-lg bg-slate-800/40 border border-slate-700/40 font-mono text-xs text-slate-400 overflow-x-auto animate-slide-up transition-all duration-200">
                  {JSON.stringify(jwtPayload, null, 2)}
                </pre>
              )}
            </div>
          )}

          {/* Metadata grid */}
          <div className="grid grid-cols-2 gap-3 text-xs">
            <div className="rounded-lg bg-slate-800/40 border border-slate-700/40 p-3">
              <div className="text-slate-500 uppercase tracking-wider mb-0.5">Expires</div>
              <div className="text-slate-200 font-mono">{expiryLabel || '—'}</div>
            </div>
            <div className="rounded-lg bg-slate-800/40 border border-slate-700/40 p-3">
              <div className="text-slate-500 uppercase tracking-wider mb-0.5">Type</div>
              <div className="text-slate-200 font-mono">{credentials.tokenType || 'Bearer'}</div>
            </div>
          </div>

          {/* Copy feedback */}
          {copiedField === 'ok' && (
            <p className="mt-3 text-xs text-emerald-400 animate-fade-in flex items-center gap-1 transition-all duration-200">
              <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              Copied to clipboard.
            </p>
          )}
          {copiedField === 'fail' && (
            <p className="mt-3 text-xs text-red-400 animate-fade-in transition-all duration-200">
              Could not copy automatically. Select and copy manually; value re-masks after{' '}
              {REVEAL_MS / 1000}s.
            </p>
          )}

          {/* Security info */}
          <div className="mt-4 flex items-start gap-2.5 rounded-lg bg-cyan-500/8 border border-cyan-500/20 p-3">
            <span className="text-cyan-400 mt-0.5 flex-shrink-0">ⓘ</span>
            <p className="text-xs text-slate-400 leading-relaxed">
              Store this credential in your secrets manager now. AppGate never stores the raw
              JWT after issuance. Present it as{' '}
              <code className="font-mono text-cyan-400">Authorization: Bearer &lt;jwt&gt;</code>{' '}
              to the gateway to proxy requests.
            </p>
          </div>
        </div>

        {/* ================================================================ */}
        {/* FOOTER — Acknowledge & Close                                    */}
        {/* ================================================================ */}
        <div className="px-6 py-4 border-t border-slate-700/40 flex items-center justify-end">
          <button
            onClick={handleAcknowledgeAndClose}
            className="px-6 py-2.5 rounded-lg glass-button-primary text-sm font-semibold transition-all duration-200"
          >
            I have saved my credentials
          </button>
        </div>
      </div>
    </div>
  )
}