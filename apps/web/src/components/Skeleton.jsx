import React from 'react'

/**
 * Skeleton — a shimmer loading placeholder for content that hasn't loaded yet.
 * Uses the skeleton-shimmer CSS class for the animated gradient.
 */
export default function Skeleton({ className = '', lines = 1 }) {
  if (lines > 1) {
    return (
      <div className={`space-y-2 ${className}`}>
        {Array.from({ length: lines }).map((_, i) => (
          <div
            key={i}
            className="skeleton-shimmer rounded-lg"
            style={{
              height: '1rem',
              width: `${100 - i * 15}%`,
            }}
          />
        ))}
      </div>
    )
  }

  return (
    <div
      className={`skeleton-shimmer rounded-lg ${className}`}
      style={{ minHeight: '1rem' }}
    />
  )
}