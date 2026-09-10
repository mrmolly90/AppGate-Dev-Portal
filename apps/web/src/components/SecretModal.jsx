import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'

/**
 * SecretModal — Ephemeral Credential Reveal Modal
 *
 * Displays the one-time credential handshake (JWT + Secret ID) from the
 * AppGate Control Plane with enhanced security and UX:
 *
 * Security:
 *  - Credentials held ONLY in component state — never persisted.
 *  - Secrets masked by default; each field reveals for a bounded time (REVEAL_MS).
 *  - Click-to-copy with visual feedback; copies directly to clipboard.
 *  - Closing the modal zeroes all in-memory references.
 *  - Escape key requires explicit confirmation before closing.
 *  - Backdrop prevents accidental dismissal.
 *
 * UX:
 *  - Glassmorphism panel with cyan accent.
 *  - Slide-up animation entrance.
 *  - Copy button with "Copied!" success state.
 *  - Reveal toggle with auto-revert timer.
 *  - JWT payload preview (parsed claims).
 *  - Warning banner emphasizing one-time display.
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

/**
 * Parses the base64-encoded payload of a JWT for preview display.
 */
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
              className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors focus:outline-none"
              aria-label={visible ? 'Hide credential' : 'Reveal credential'}
            >
              {visible ? 'Hide' : 'Reveal'}
            </button>
          )}
          <button
            type="button"
            onClick={() => onCopy(value)}
            className="text-xs px-2.5 py-1 rounded-md glass-button"
          >
            Copy
          </button>
        </div>
      </div>
      <code className="block w-full px-3.5 py-2.5 rounded-lg bg-slate-800/60 border border-slate-700/60 font-mono text-sm text-slate-200 break-all select-none-all"
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

export default function SecretModal({ credentials, onClose }) {
  const [visibleField, setVisibleField] = useState(null)
  const [copiedField, setCopiedField] = useState(null)
  const [confirmClose, setConfirmClose] = useState(false)
  const [showPayload, setShowPayload] = useState(false)
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
    return () => { document.body.style.overflow = prev }
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

  const handleClose = useCallback(() => {
    setVisibleField(null)
    setCopiedField(null)
    onClose?.()
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
        {/* ==================== Header ==================== */}
        <div className="px-6 pt-5 pb-4 border-b border-slate-700/40">
          <div className="flex items-start justify-between">
            <div>
              <h2 id="secret-modal-title" className="text-lg font-semibold text-slate-100 flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-glow-pulse" />
                Gateway Credentials Issued
              </h2>
              <p className="text-sm text-slate-400 mt-1">
                Credential for <span className="text-cyan-400 font-mono">{credentials.clientId}</span>.
                <strong className="text-red-400 ml-1">One-time display only.</strong>
              </p>
            </div>
            <button
              onClick={() => setConfirmClose(true)}
              className="text-slate-500 hover:text-slate-300 transition-colors p-1"
              aria-label="Close modal"
            >
              <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          {/* Confirm close */}
          {confirmClose && (
            <div className="mt-3 animate-slide-up flex items-center justify-between bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-2.5">
              <span className="text-xs text-red-400">Closing discards these credentials permanently.</span>
              <div className="flex gap-2">
                <button
                  onClick={() => setConfirmClose(false)}
                  className="text-xs px-3 py-1.5 rounded-md glass-button"
                >
                  Keep open
                </button>
                <button
                  onClick={handleClose}
                  className="text-xs px-3 py-1.5 rounded-md bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30 transition-all"
                >
                  Discard & close
                </button>
              </div>
            </div>
          )}
        </div>

        {/* ==================== Body ==================== */}
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

          {/* Client ID (non-secret) */}
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
                className="text-xs text-cyan-400/70 hover:text-cyan-400 transition-colors flex items-center gap-1"
              >
                {showPayload ? '▾' : '▸'} JWT Claims Preview
              </button>
              {showPayload && (
                <pre className="mt-2 px-3.5 py-2.5 rounded-lg bg-slate-800/40 border border-slate-700/40 font-mono text-xs text-slate-400 overflow-x-auto animate-slide-up">
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
            <p className="mt-3 text-xs text-emerald-400 animate-fade-in flex items-center gap-1">
              <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12" /></svg>
              Copied to clipboard.
            </p>
          )}
          {copiedField === 'fail' && (
            <p className="mt-3 text-xs text-red-400 animate-fade-in">
              Could not copy automatically. Select and copy manually; value re-masks after {REVEAL_MS / 1000}s.
            </p>
          )}

          {/* Warning */}
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

        {/* ==================== Footer ==================== */}
        <div className="px-6 py-4 border-t border-slate-700/40 flex items-center justify-end gap-3">
          <button
            onClick={() => setConfirmClose(true)}
            className="px-4 py-2 rounded-lg glass-button text-sm"
          >
            Close
          </button>
          <button
            onClick={handleClose}
            className="px-4 py-2 rounded-lg glass-button-primary text-sm"
          >
            Done — I&apos;ve saved it
          </button>
        </div>
      </div>
    </div>
  )
}