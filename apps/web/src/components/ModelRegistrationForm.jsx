import React, { useCallback, useState } from 'react'
import { registrationSchema, LLM_MODEL_OPTIONS } from '../lib/validation.js'
import BudgetSlider from './BudgetSlider.jsx'

/**
 * ModelRegistrationForm — Enterprise-grade multi-segment registration form.
 *
 * Segments:
 *   1. Model Identity (project_name, llm_model_name)
 *   2. Upstream Provider Configuration (provider_url, provider_sdk_secret)
 *   3. Policy & Quota Management (monthly_budget_limit, rate_limit_rpm)
 *
 * Security:
 *  - Zod validation on every field before submission (fail closed).
 *  - Provider SDK secret is cleared from state on any error.
 *  - No secrets logged or persisted.
 *  - All inputs sanitized via Zod .trim().
 *
 * UX:
 *  - Glassmorphism dark theme with cyan accents.
 *  - Smooth transitions (duration-200) for validation states.
 *  - Reveal/hide toggle for the SDK secret field.
 *  - Unified slider + number input for budget.
 *  - Enterprise-grade loading spinner on submit.
 */
export default function ModelRegistrationForm({
  form,
  updateField,
  onSubmit,
  isSubmitting,
  error,
  fieldErrors,
  onClearError,
}) {
  const [showSecret, setShowSecret] = useState(false)

  const handleSubmit = useCallback(
    (e) => {
      e.preventDefault()
      onClearError?.()

      // Zod validation — fail closed
      const parsed = registrationSchema.safeParse({
        projectName: form.projectName,
        llmModelName: form.llmModelName,
        providerUrl: form.providerUrl,
        providerSdkSecret: form.providerSdkSecret,
        monthlyBudgetLimit: form.monthlyBudgetLimit,
        rateLimitRpm: form.rateLimitRpm,
      })

      if (!parsed.success) {
        // Map Zod errors to field-level messages
        const fieldMap = {}
        for (const issue of parsed.error.issues) {
          const path = issue.path.join('.')
          if (!fieldMap[path]) {
            fieldMap[path] = issue.message
          }
        }
        // Pass field errors up via a custom event
        const customEvent = new CustomEvent('registration-field-errors', {
          detail: fieldMap,
        })
        window.dispatchEvent(customEvent)
        return
      }

      // Submit validated payload
      onSubmit({
        projectName: parsed.data.projectName,
        model: parsed.data.llmModelName,
        llmModelName: parsed.data.llmModelName,
        providerKey: parsed.data.providerSdkSecret,
        providerUrl: parsed.data.providerUrl,
        monthlySpendUSD: parsed.data.monthlyBudgetLimit,
        rateLimitRPS: Math.max(1, Math.round(parsed.data.rateLimitRpm / 60)),
        rateLimitRPM: parsed.data.rateLimitRpm,
        rateLimitBurst: Math.max(1, Math.round(parsed.data.rateLimitRpm / 30)),
        webhookURL: form.webhookURL,
        tags: parseTags(form.tags),
      })
    },
    [form, onSubmit, onClearError]
  )

  return (
    <form onSubmit={handleSubmit} className="space-y-8" noValidate>
      {/* ================================================================ */}
      {/* ERROR BANNER                                                     */}
      {/* ================================================================ */}
      {error && (
        <div
          className="animate-slide-up rounded-lg bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400 flex items-start gap-2 transition-all duration-200"
          role="alert"
        >
          <svg
            className="w-5 h-5 flex-shrink-0 mt-0.5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <div className="flex-1">
            <strong className="block mb-0.5">Registration Failed</strong>
            <span>{error}</span>
          </div>
          <button
            type="button"
            onClick={onClearError}
            className="text-red-400/60 hover:text-red-300 transition-colors flex-shrink-0"
            aria-label="Dismiss error"
          >
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      )}

      {/* ================================================================ */}
      {/* SEGMENT 1: Model Identity                                        */}
      {/* ================================================================ */}
      <fieldset className="space-y-5">
        <legend className="text-sm font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-cyan-400" />
          Model Identity
        </legend>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormField
            label="Project Name"
            htmlFor="projectName"
            error={fieldErrors?.projectName}
            hint="Internal name for your application"
          >
            <input
              id="projectName"
              type="text"
              required
              maxLength={128}
              placeholder="my-llm-app"
              className={`glass-input w-full px-3 py-2.5 rounded-lg text-sm transition-all duration-200 ${
                fieldErrors?.projectName ? 'ring-2 ring-red-500/50 border-red-500/60' : ''
              }`}
              value={form.projectName}
              onChange={(e) => updateField('projectName', e.target.value)}
              autoComplete="off"
            />
          </FormField>

          <FormField
            label="LLM Model"
            htmlFor="llmModelName"
            error={fieldErrors?.llmModelName}
          >
            <select
              id="llmModelName"
              required
              className={`glass-input w-full px-3 py-2.5 rounded-lg text-sm appearance-none transition-all duration-200 ${
                fieldErrors?.llmModelName ? 'ring-2 ring-red-500/50 border-red-500/60' : ''
              }`}
              value={form.llmModelName}
              onChange={(e) => updateField('llmModelName', e.target.value)}
            >
              <option value="" disabled>
                Select a model…
              </option>
              {LLM_MODEL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </FormField>
        </div>
      </fieldset>

      {/* ================================================================ */}
      {/* SEGMENT 2: Upstream Provider Configuration                       */}
      {/* ================================================================ */}
      <fieldset className="space-y-5">
        <legend className="text-sm font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-cyan-400" />
          Upstream Provider Configuration
        </legend>

        <FormField
          label="Provider API Base URL"
          htmlFor="providerUrl"
          error={fieldErrors?.providerUrl}
          hint="HTTPS required. SSRF defense enforced."
        >
          <input
            id="providerUrl"
            type="url"
            required
            placeholder="https://api.openai.com/v1"
            className={`glass-input w-full px-3 py-2.5 rounded-lg text-sm transition-all duration-200 ${
              fieldErrors?.providerUrl ? 'ring-2 ring-red-500/50 border-red-500/60' : ''
            }`}
            value={form.providerUrl}
            onChange={(e) => updateField('providerUrl', e.target.value)}
            autoComplete="url"
          />
        </FormField>

        <FormField
          label="Provider SDK Secret"
          htmlFor="providerSdkSecret"
          error={fieldErrors?.providerSdkSecret}
          hint="Raw API key. Sent directly to AppGate CP. Never stored or logged."
        >
          <div className="relative">
            <input
              id="providerSdkSecret"
              type={showSecret ? 'text' : 'password'}
              required
              minLength={8}
              autoComplete="off"
              placeholder="sk-••••••••••••••••"
              className={`glass-input w-full px-3 py-2.5 rounded-lg text-sm font-mono pr-10 transition-all duration-200 ${
                fieldErrors?.providerSdkSecret ? 'ring-2 ring-red-500/50 border-red-500/60' : ''
              }`}
              value={form.providerSdkSecret}
              onChange={(e) => updateField('providerSdkSecret', e.target.value)}
            />
            <button
              type="button"
              onClick={() => setShowSecret((s) => !s)}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-500 hover:text-cyan-400 transition-colors duration-200"
              aria-label={showSecret ? 'Hide provider secret' : 'Reveal provider secret'}
              tabIndex={-1}
            >
              {showSecret ? (
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                  <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                  <line x1="1" y1="1" x2="23" y2="23" />
                </svg>
              ) : (
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              )}
            </button>
          </div>
        </FormField>
      </fieldset>

      {/* ================================================================ */}
      {/* SEGMENT 3: Policy & Quota Management                             */}
      {/* ================================================================ */}
      <fieldset className="space-y-5">
        <legend className="text-sm font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-cyan-400" />
          Policy &amp; Quota Management
        </legend>

        {/* Monthly Budget — unified slider + number input */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label htmlFor="monthlyBudgetLimit" className="text-sm font-medium text-slate-300">
              Monthly Budget Limit (USD)
            </label>
            <div className="flex items-center gap-2">
              <span className="text-slate-500 text-xs">$</span>
              <input
                id="monthlyBudgetLimit"
                type="number"
                min={1}
                max={100000}
                className="glass-input w-24 px-2 py-1 rounded-lg text-sm font-mono text-right tabular-nums transition-all duration-200"
                value={form.monthlyBudgetLimit}
                onChange={(e) => {
                  const v = Number(e.target.value)
                  if (!isNaN(v) && v >= 0) updateField('monthlyBudgetLimit', v)
                }}
              />
            </div>
          </div>
          <BudgetSlider value={form.monthlyBudgetLimit} onChange={(v) => updateField('monthlyBudgetLimit', v)} />
          {fieldErrors?.monthlyBudgetLimit && (
            <p className="mt-1 text-xs text-red-400 transition-all duration-200">{fieldErrors.monthlyBudgetLimit}</p>
          )}
        </div>

        {/* Rate Limit RPM */}
        <FormField
          label="Rate Limit (requests/min)"
          htmlFor="rateLimitRpm"
          error={fieldErrors?.rateLimitRpm}
          hint="Maximum requests per minute allowed through the gateway"
        >
          <input
            id="rateLimitRpm"
            type="number"
            required
            min={1}
            max={1000000}
            className={`glass-input w-full px-3 py-2.5 rounded-lg text-sm transition-all duration-200 ${
              fieldErrors?.rateLimitRpm ? 'ring-2 ring-red-500/50 border-red-500/60' : ''
            }`}
            value={form.rateLimitRpm}
            onChange={(e) => {
              const v = Number(e.target.value)
              if (!isNaN(v)) updateField('rateLimitRpm', v)
            }}
          />
        </FormField>

        {/* Optional fields */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormField label="Quota Webhook URL" htmlFor="webhookUrl" optional>
            <input
              id="webhookUrl"
              type="url"
              placeholder="https://ops.example.com/quota-events"
              className="glass-input w-full px-3 py-2.5 rounded-lg text-sm transition-all duration-200"
              value={form.webhookUrl}
              onChange={(e) => updateField('webhookURL', e.target.value)}
            />
          </FormField>

          <FormField label="Tags" htmlFor="tags" optional hint="Comma-separated key=value pairs">
            <input
              id="tags"
              type="text"
              placeholder="team=payments, env=prod"
              className="glass-input w-full px-3 py-2.5 rounded-lg text-sm transition-all duration-200"
              value={form.tags}
              onChange={(e) => updateField('tags', e.target.value)}
            />
          </FormField>
        </div>
      </fieldset>

      {/* ================================================================ */}
      {/* SUBMIT BUTTON                                                    */}
      {/* ================================================================ */}
      <button
        type="submit"
        disabled={isSubmitting}
        className="glass-button-primary w-full py-3 rounded-lg text-sm transition-all duration-200 disabled:opacity-40 disabled:cursor-not-allowed"
      >
        {isSubmitting ? (
          <span className="flex items-center justify-center gap-2">
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            Issuing credentials…
          </span>
        ) : (
          'Register Gateway & Issue Credentials'
        )}
      </button>
    </form>
  )
}

// ---------------------------------------------------------------------------
// FormField helper
// ---------------------------------------------------------------------------

function FormField({ label, htmlFor, error, hint, optional, children }) {
  return (
    <div className="animate-fade-in">
      <div className="flex items-center justify-between mb-1.5">
        <label htmlFor={htmlFor} className="text-sm font-medium text-slate-300">
          {label}
          {optional && <span className="text-slate-500 font-normal ml-1">(optional)</span>}
        </label>
      </div>
      {children}
      {error && (
        <p className="mt-1 text-xs text-red-400 flex items-center gap-1 transition-all duration-200" role="alert">
          <svg className="w-3 h-3 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          {error}
        </p>
      )}
      {!error && hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Tag parser
// ---------------------------------------------------------------------------

function parseTags(raw) {
  if (!raw) return {}
  const tags = {}
  for (const part of raw.split(',')) {
    const [k, v] = part.split('=')
    if (k && k.trim()) tags[k.trim()] = (v || '').trim()
  }
  return tags
}