import { useCallback, useEffect, useState } from 'react'
import { registerTenant, fetchSecretStatus, fetchHealth, fetchGateways } from '../api/devPortalApi.js'

/**
 * useRegistration — manages the registration form state and submission.
 * Credentials live only in component state — never persisted.
 */
export function useRegistration() {
  const [form, setForm] = useState({
    model: '',
    providerKey: '',
    providerUrl: '',
    monthlySpendUSD: 500,
    rateLimitRPS: 1000,
    rateLimitBurst: 2000,
    webhookURL: '',
    tags: '',
  })

  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState(null)
  const [credentials, setCredentials] = useState(null)

  const updateField = useCallback((field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }, [])

  const submit = useCallback(async (payload = form) => {
    setIsSubmitting(true)
    setError(null)
    setCredentials(null)
    try {
      const creds = await registerTenant(payload)
      setCredentials(creds)
      return creds
    } catch (e) {
      setError(e.message || 'Registration failed')
      return null
    } finally {
      setIsSubmitting(false)
    }
  }, [form])

  const clearCredentials = useCallback(() => {
    setCredentials(null)
    setForm((prev) => ({ ...prev, providerKey: '' }))
  }, [])

  return {
    form,
    setForm,
    updateField,
    submit,
    isSubmitting,
    error,
    setError,
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