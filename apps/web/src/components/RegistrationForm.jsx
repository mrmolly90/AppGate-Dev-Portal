import React from 'react'
import BudgetSlider from './BudgetSlider.jsx'

/**
 * RegistrationForm — collects provider credentials and spend limits.
 * Glassmorphism inputs, fail-closed validation, cyan accent theme.
 * Provider keys are held in local state only — never logged or persisted.
 */
export default function RegistrationForm({ form, updateField, onSubmit, isSubmitting, error }) {
  const handleSubmit = (e) => {
    e.preventDefault()
    onSubmit({
      model: form.model,
      providerKey: form.providerKey,
      providerUrl: form.providerUrl,
      monthlySpendUSD: form.monthlySpendUSD,
      rateLimitRPS: form.rateLimitRPS,
      rateLimitBurst: form.rateLimitBurst,
      webhookURL: form.webhookURL,
      tags: parseTags(form.tags),
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {/* Error banner */}
      {error && (
        <div className="animate-slide-up rounded-lg bg-red-500/10 border border-red-500/30 px-4 py-3 text-sm text-red-400 flex items-start gap-2">
          <span className="mt-0.5 flex-shrink-0">⚠</span>
          <span>{error}</span>
        </div>
      )}

      {/* Model */}
      <Field label="Model Identifier" htmlFor="model" hint="e.g. gpt-4o, claude-3-5-sonnet, llama-3.1-70b">
        <input
          id="model"
          type="text"
          required
          maxLength={128}
          placeholder="gpt-4o, claude-3-5-sonnet, …"
          className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
          value={form.model}
          onChange={(e) => updateField('model', e.target.value)}
        />
      </Field>

      {/* Provider Key */}
      <Field label="Upstream Provider Key" htmlFor="providerKey" hint="Sent directly to AppGate CP. Never stored or logged.">
        <input
          id="providerKey"
          type="password"
          required
          minLength={8}
          autoComplete="off"
          placeholder="sk-••••••••••••••••"
          className="glass-input w-full px-3 py-2.5 rounded-lg text-sm font-mono"
          value={form.providerKey}
          onChange={(e) => updateField('providerKey', e.target.value)}
        />
      </Field>

      {/* Provider URL */}
      <Field label="Upstream Provider Base URL" htmlFor="providerUrl" hint="HTTPS required.">
        <input
          id="providerUrl"
          type="url"
          required
          placeholder="https://api.openai.com/v1"
          className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
          value={form.providerUrl}
          onChange={(e) => updateField('providerUrl', e.target.value)}
        />
      </Field>

      {/* Budget */}
      <BudgetSlider value={form.monthlySpendUSD} onChange={(v) => updateField('monthlySpendUSD', v)} />

      {/* Rate limits */}
      <div className="grid grid-cols-2 gap-4">
        <Field label="Rate Limit (req/s)" htmlFor="rateLimitRPS">
          <input
            id="rateLimitRPS"
            type="number"
            min={1}
            className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
            value={form.rateLimitRPS}
            onChange={(e) => updateField('rateLimitRPS', Number(e.target.value))}
          />
        </Field>
        <Field label="Burst Allowance" htmlFor="rateLimitBurst">
          <input
            id="rateLimitBurst"
            type="number"
            min={1}
            className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
            value={form.rateLimitBurst}
            onChange={(e) => updateField('rateLimitBurst', Number(e.target.value))}
          />
        </Field>
      </div>

      {/* Webhook */}
      <Field label="Quota Webhook URL" htmlFor="webhookUrl" optional>
        <input
          id="webhookUrl"
          type="url"
          placeholder="https://ops.example.com/quota-events"
          className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
          value={form.webhookUrl}
          onChange={(e) => updateField('webhookURL', e.target.value)}
        />
      </Field>

      {/* Tags */}
      <Field label="Tags" htmlFor="tags" optional hint="Comma-separated key=value pairs">
        <input
          id="tags"
          type="text"
          placeholder="team=payments, env=prod"
          className="glass-input w-full px-3 py-2.5 rounded-lg text-sm"
          value={form.tags}
          onChange={(e) => updateField('tags', e.target.value)}
        />
      </Field>

      {/* Submit */}
      <button
        type="submit"
        disabled={isSubmitting}
        className="glass-button-primary w-full py-3 rounded-lg text-sm"
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
// Field helper
// ---------------------------------------------------------------------------

function Field({ label, htmlFor, hint, optional, children }) {
  return (
    <div className="animate-fade-in">
      <div className="flex items-center justify-between mb-1.5">
        <label htmlFor={htmlFor} className="text-sm font-medium text-slate-300">
          {label}
          {optional && <span className="text-slate-500 font-normal ml-1">(optional)</span>}
        </label>
      </div>
      {children}
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
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