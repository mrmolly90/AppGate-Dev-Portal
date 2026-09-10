import { useCallback, useEffect, useState } from 'react'
import { registerTenant, fetchSecretStatus, fetchHealth, fetchGateways } from '../api/devPortalApi.js'

/**
 * useRegistration — manages the registration form state and submission.
 * Credentials live only in component state — never persisted.
 * Provider SDK secret is wiped on any error (fail closed).
 */
export function useRegistration() {
  const [form, setForm] = useState({
    projectName: '',
    llmModelName: '',
    providerUrl: '',
    providerSdkSecret: '',
    monthlyBudgetLimit: 500,
    rateLimitRpm: 60000,
    webhookURL: '',
    tags: '',
  })

  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState(null)
  const [fieldErrors, setFieldErrors] = useState({})
  const [credentials, setCredentials] = useState(null)

  // Listen for Zod field-level validation errors
  useEffect(() => {
    const handler = (e) => {
      setFieldErrors(e.detail || {})
    }
    window.addEventListener('registration-field-errors', handler)
    return () => window.removeEventListener('registration-field-errors', handler)
  }, [])

  const updateField = useCallback((field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    // Clear field-level error when user corrects the field
    setFieldErrors((prev) => {
      if (prev[field]) {
        const next = { ...prev }
        delete next[field]
        return next
      }
      return prev
    })
  }, [])

  const submit = useCallback(async (payload) => {
    setIsSubmitting(true)
    setError(null)
    setCredentials(null)
    setFieldErrors({})
    try {
      const creds = await registerTenant(payload)
      setCredentials(creds)
      return creds
    } catch (e) {
      setError(e.message || 'Registration failed')
      // Fail closed: wipe provider secret on any error
      setForm((prev) => ({ ...prev, providerSdkSecret: '' }))
      return null
    } finally {
      setIsSubmitting(false)
    }
  }, [])

  const clearCredentials = useCallback(() => {
    // Zero trust: actively wipe all credential state
    setCredentials(null)
    setFieldErrors({})
  }, [])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  return {
    form,
    setForm,
    updateField,
    submit,
    isSubmitting,
    error,
    setError,
    clearError,
    fieldErrors,
    credentials,
    clearCredentials,
  }
}

/**
 * useSecretStatus — polls the control plane for a secret's lifecycle state.
 */
export function useSecretStatus(clientId, enabled = true) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const refresh = useCallback(async () => {
    if (!clientId || !enabled) return
    setLoading(true)
    setError(null)
    try {
      const result = await fetchSecretStatus(clientId)
      setData(result)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [clientId, enabled])

  useEffect(() => {
    refresh()
    const timer = setInterval(refresh, 30000)
    return () => clearInterval(timer)
  }, [refresh])

  return { data, loading, error, refresh }
}

/**
 * useBffHealth — monitors BFF and control plane health.
 */
export function useBffHealth() {
  const [health, setHealth] = useState({ status: 'loading' })
  const [error, setError] = useState(null)

  const check = useCallback(async () => {
    try {
      const h = await fetchHealth()
      setHealth(h)
      setError(null)
    } catch (e) {
      setHealth({ status: 'unreachable' })
      setError(e.message)
    }
  }, [])

  useEffect(() => {
    check()
    const timer = setInterval(check, 30000)
    return () => clearInterval(timer)
  }, [check])

  return { health, error, refresh: check }
}

/**
 * useGateways — fetches the list of registered gateways.
 */
export function useGateways() {
  const [gateways, setGateways] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await fetchGateways()
      setGateways(result)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { gateways, loading, error, refresh }
}