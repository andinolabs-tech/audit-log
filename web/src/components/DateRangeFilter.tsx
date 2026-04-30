import { useEffect, useRef, useState } from 'react'

import type { DatePreset, DateRangeValue } from '../dateRange'

interface DateRangeFilterProps {
  value: DateRangeValue
  onChange: (value: DateRangeValue) => void
}

const presetLabels: Record<DatePreset, string> = {
  today: 'Today',
  yesterday: 'Yesterday',
  this_week: 'This week',
  last_week: 'Last week',
  this_month: 'This month',
  last_month: 'Last month',
  custom: 'Custom',
}

const presetOptions = Object.entries(presetLabels) as Array<[DatePreset, string]>

export default function DateRangeFilter({ value, onChange }: DateRangeFilterProps) {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const handleDocumentClick = (event: MouseEvent) => {
      if (!containerRef.current) return
      if (containerRef.current.contains(event.target as Node)) return
      setOpen(false)
    }

    document.addEventListener('mousedown', handleDocumentClick)
    return () => document.removeEventListener('mousedown', handleDocumentClick)
  }, [])

  const selectedLabel = presetLabels[value.preset]

  const selectPreset = (preset: DatePreset) => {
    setOpen(false)
    if (preset === 'custom') {
      onChange({ preset: 'custom', from: '', to: '' })
      return
    }
    onChange({ preset })
  }

  const setCustomDate = (key: 'from' | 'to', date: string) => {
    onChange({
      preset: 'custom',
      from: value?.preset === 'custom' ? value.from : '',
      to: value?.preset === 'custom' ? value.to : '',
      [key]: date,
    })
  }

  return (
    <div ref={containerRef} className="relative lg:self-start">
      <label className="mb-2 block text-sm font-medium text-slate-700">Date range</label>
      <button
        type="button"
        onClick={() => setOpen((previous) => !previous)}
        aria-expanded={open}
        className="flex min-h-11 w-full cursor-pointer items-center justify-between rounded-lg border border-slate-300 px-3 text-sm text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
      >
        <span>{selectedLabel}</span>
        <span aria-hidden="true" className="text-slate-400">
          v
        </span>
      </button>

      {open && (
        <div className="absolute z-10 mt-1 w-full rounded-lg border border-slate-300 bg-white p-2 shadow-lg">
          {presetOptions.map(([preset, label]) => (
            <button
              key={preset}
              type="button"
              role="menuitem"
              onClick={() => selectPreset(preset)}
              className="flex min-h-10 w-full cursor-pointer items-center justify-between rounded-md px-2 text-left text-sm hover:bg-slate-50"
            >
              <span>{label}</span>
              {value.preset === preset && <span aria-hidden="true">✓</span>}
            </button>
          ))}
        </div>
      )}

      {value.preset === 'custom' && (
        <div className="mt-3 grid gap-3">
          <div>
            <label htmlFor="date-range-from" className="mb-1 block text-xs font-medium text-slate-600">
              From
            </label>
            <input
              id="date-range-from"
              type="date"
              value={value.from}
              onChange={(event) => setCustomDate('from', event.target.value)}
              className="min-h-11 w-full rounded-lg border border-slate-300 px-3 text-sm focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
            />
          </div>
          <div>
            <label htmlFor="date-range-to" className="mb-1 block text-xs font-medium text-slate-600">
              To
            </label>
            <input
              id="date-range-to"
              type="date"
              value={value.to}
              onChange={(event) => setCustomDate('to', event.target.value)}
              className="min-h-11 w-full rounded-lg border border-slate-300 px-3 text-sm focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
            />
          </div>
        </div>
      )}
    </div>
  )
}
