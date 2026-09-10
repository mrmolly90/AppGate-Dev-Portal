import React from 'react'
import ModelRegistrationForm from '../components/ModelRegistrationForm.jsx'
import SecretDisplayModal from '../components/SecretDisplayModal.jsx'
import Skeleton from '../components/Skeleton.jsx'
import { useRegistration, useBffHealth, useGateways } from '../hooks/useDevPortal.js'

/**
 * Dashboard — the main registration and credential management view.
 *
 * Features:
 *  - Multi-segment gateway registration form with Zod validation.
 *  - Ephemeral SecretDisplayModal on successful issuance.
 *  - Recent gateways list with live status.
 *  - Health indicator connection status.
 *  - Loading skeletons during data fetch.
 *  - Fail-closed error states (provider secret wiped on error).
 */
export default function Dashboard() {
  const {
    form,
    updateField,
    submit,
    isSubmitting,
    error,
    clearError,
    fieldErrors,
    credentials,
    clearCredentials,
  } = useRegistration()

  const { health } = useBffHealth()
  const { gateways, loading: gwLoading, error: gwError } = useGateways()

  const handleSubmit = async (payload) => {
    const result = await submit(payload)
    if (result) clearError()
  }

  const isBackendDown = health.status === 'unreachable'

  return (
    <div className="space-y-8">
      {/* Page header */}
      <div className="animate-fade-in">
        <h2 className="text-xl font-semibold text-slate-100">Dashboard</h2>
        <p className="text-sm text-slate-400 mt-1">
          Register upstream LLM gateways and issue execution credentials
        </p>
      </div>

      {/* Connection warning banner */}
      {isBackendDown && (
        <div className="animate-slide-up rounded-lg bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400 flex items-center gap-3">
          <svg className="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <div>
            <strong>Backend unreachable.</strong> Registration is disabled. Check your connection and try again.
          </div>
        </div>
      )}

      {/* Registration form panel */}
      <div className="glass-card">
        <div className="mb-6">
          <h3 className="text-base font-semibold text-slate-100">Register an Upstream LLM Gateway</h3>
          <p className="text-sm text-slate-400 mt-0.5">
            Submit provider credentials and spend limits to receive a signed execution JWT
          </p>
        </div>
        <ModelRegistrationForm
          form={form}
          updateField={updateField}
          onSubmit={handleSubmit}
          isSubmitting={isSubmitting || isBackendDown}
          error={error}
          fieldErrors={fieldErrors}
          onClearError={clearError}
        />
      </div>

      {/* Recent gateways */}
      <div className="glass-card">
        <h3 className="text-base font-semibold text-slate-100 mb-4">Recent Gateways</h3>
        {gwLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
            <Skeleton className="h-12" />
          </div>
        ) : gwError ? (
          <p className="text-sm text-red-400">Failed to load gateways: {gwError}</p>
        ) : gateways.length === 0 ? (
          <div className="text-center py-8">
            <svg className="w-12 h-12 mx-auto text-slate-700 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="3" y="3" width="7" height="7" rx="1" />
              <rect x="14" y="3" width="7" height="7" rx="1" />
              <rect x="3" y="14" width="7" height="7" rx="1" />
              <rect x="14" y="14" width="7" height="7" rx="1" />
            </svg>
            <p className="text-sm text-slate-500">No gateways registered yet.</p>
            <p className="text-xs text-slate-600 mt-1">Use the form above to register your first gateway.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {gateways.map((gw) => (
              <div
                key={gw.id}
                className="flex items-center justify-between px-4 py-3 rounded-lg bg-slate-800/40 border border-slate-700/40 animate-slide-up"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
                    gw.status === 'active' ? 'bg-emerald-400' : 'bg-slate-500'
                  }`} />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-200 truncate">{gw.project_name || gw.id}</p>
                    <p className="text-xs text-slate-500">{gw.model}</p>
                  </div>
                </div>
                <div className="text-xs text-slate-500 text-right flex-shrink-0 ml-4">
                  <div>{gw.status}</div>
                  <div>{new Date(gw.created_at).toLocaleDateString()}</div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Ephemeral credential modal */}
      {credentials && (
        <SecretDisplayModal credentials={credentials} onClose={clearCredentials} />
      )}
    </div>
  )
}