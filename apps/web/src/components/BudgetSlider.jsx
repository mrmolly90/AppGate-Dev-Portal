import React from 'react'

/**
 * BudgetSlider — interactive monthly spend limit control with glassmorphism.
 * Features: range slider, preset chips, live dollar display, cyan accent.
 */
export default function BudgetSlider({ value, onChange }) {
  const presets = [100, 500, 1000, 5000, 10000]

  return (
    <div className="animate-fade-in">
      <div className="flex items-center justify-between mb-2">
        <label className="text-sm font-medium text-slate-300">
          Monthly Spend Limit (USD)
        </label>
        <span className="font-mono text-sm text-cyan-400 tabular-nums font-semibold">
          ${value.toLocaleString()}
        </span>
      </div>
      <input
        type="range"
        min={100}
        max={10000}
        step={100}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full"
        aria-label="Monthly spend limit"
      />
      <div className="flex gap-2 mt-3">
        {presets.map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => onChange(p)}
            className={`flex-1 px-2 py-1.5 rounded-lg text-xs font-medium border transition-all duration-200 ${
              value === p
                ? 'bg-cyan-500/15 border-cyan-500/40 text-cyan-400 shadow-sm shadow-cyan-500/10'
                : 'bg-slate-800/40 border-slate-700/50 text-slate-400 hover:border-cyan-500/30 hover:text-cyan-400/80'
            }`}
          >
            ${p.toLocaleString()}
          </button>
        ))}
      </div>
    </div>
  )
}