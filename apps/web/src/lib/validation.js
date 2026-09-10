/**
 * Zod validation schema for the gateway registration form.
 * Enforces strict client-side validation before submission (fail closed).
 * All fields are sanitized and type-checked.
 */
import { z } from 'zod'

const httpsUrl = z
  .string()
  .trim()
  .url('Must be a valid URL')
  .refine((v) => v.startsWith('https://'), {
    message: 'Only HTTPS URLs are allowed (SSRF defense)',
  })

export const registrationSchema = z
  .object({
    projectName: z
      .string()
      .trim()
      .min(1, 'Project name is required')
      .max(128, 'Project name must be 128 characters or less'),
    llmModelName: z
      .string()
      .trim()
      .min(1, 'LLM model selection is required'),
    providerUrl: httpsUrl,
    providerSdkSecret: z
      .string()
      .trim()
      .min(8, 'Provider SDK secret must be at least 8 characters'),
    monthlyBudgetLimit: z
      .number()
      .positive('Monthly budget must be greater than zero')
      .max(100000, 'Monthly budget cannot exceed $100,000'),
    rateLimitRpm: z
      .number()
      .int('Rate limit must be a whole number')
      .positive('Rate limit must be greater than zero')
      .max(1000000, 'Rate limit cannot exceed 1,000,000 RPM'),
  })
  .strict()

export const LLM_MODEL_OPTIONS = [
  { value: 'gpt-4', label: 'GPT-4' },
  { value: 'gpt-4o', label: 'GPT-4o' },
  { value: 'gpt-4o-mini', label: 'GPT-4o Mini' },
  { value: 'claude-3-opus', label: 'Claude 3 Opus' },
  { value: 'claude-3-sonnet', label: 'Claude 3 Sonnet' },
  { value: 'claude-3-haiku', label: 'Claude 3 Haiku' },
  { value: 'llama-3', label: 'Llama 3' },
  { value: 'llama-3.1-70b', label: 'Llama 3.1 70B' },
  { value: 'llama-3.1-405b', label: 'Llama 3.1 405B' },
  { value: 'mistral-large', label: 'Mistral Large' },
  { value: 'mistral-medium', label: 'Mistral Medium' },
  { value: 'gemini-pro', label: 'Gemini Pro' },
  { value: 'gemini-ultra', label: 'Gemini Ultra' },
  { value: 'custom', label: 'Custom Model' },
]