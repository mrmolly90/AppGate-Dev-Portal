import React from 'react'
import ModelRegistrationForm from '../components/ModelRegistrationForm.jsx'
import SecretDisplayModal from '../components/SecretDisplayModal.jsx'
import { useRegistration, useBffHealth } from '../hooks/useDevPortal.js'

/**
 * Register — landing page. Shows "Register an Upstream LLM Gateway".
 * Full registration form with all 3 segments (Identity, Provider, Quota).
 * After successful issuance, shows the SecretDisplayModal with credentials.
 */
export default function Register() {
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
  const isBackendDown = health.status === 'unreachable'

  const handleSubmit = async (payload) => {
    const result = await submit(payload)
    if (result) clearError()
  }

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div className="animate-fade-in">
        <h2 className="text-[17px] font-semibold text-slate-100">Register an Upstream LLM Gateway</h2>
        <p className="text-[13px] text-slate-400 mt-1 leading-relaxed">
          Submit provider credentials and spend limits through AppGate's secure
          control plane to receive a signed execution JWT for the reverse proxy.
        </p>
      </div>

      {/* Backend warning */}
      {isBackendDown && (
        <div className="animate-slide-up rounded-xl bg-[rgba(248,113,113,0.08)] border border-[rgba(248,113,113,0.25)] px-4 py-3 text-sm text-red-400 flex items-start gap-3">
          <svg className="w-5 h-5 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <div>
            <strong className="block mb-0.5">Control plane unreachable</strong>
            <span className="text-red-400/80">Registration is disabled until the backend reconnects.</span>
          </div>
        </div>
      )}

      {/* Registration form */}
      <div className="panel p-5">
        <ModelRegistrationForm
          form={form}
          updateField={updateField}
          onSubmit={handleSubmit}
          isSubmitting={isSubmitting}
          isBackendDown={isBackendDown}
          error={error}
          fieldErrors={fieldErrors}
          onClearError={clearError}
        />
      </div>

      {/* Security note */}
      <div className="flex items-start gap-3 rounded-xl bg-[rgba(34,211,238,0.05)] border border-[rgba(34,211,238,0.12)] p-3.5">
        <svg className="w-4 h-4 flex-shrink-0 mt-px text-cyan-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        </svg>
        <p className="text-[11px] text-slate-500 leading-relaxed">
          Your provider SDK secret is encrypted in transit and never stored by AppGate.
          The issued JWT is displayed once — copy it immediately to your secrets manager.
        </p>
      </div>

      {/* Ephemeral credential modal */}
      {credentials && (
        <SecretDisplayModal credentials={credentials} onClose={clearCredentials} />
      )}
    </div>
  )
}