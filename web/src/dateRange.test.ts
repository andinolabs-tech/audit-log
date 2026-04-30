import { afterEach, describe, expect, it, vi } from 'vitest'

import { resolveDateRange } from './dateRange'

describe('resolveDateRange', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('resolves today from local midnight through local end of day', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 3, 30, 15, 30, 0))

    const range = resolveDateRange({ preset: 'today' })

    expect(range.from.toISOString()).toBe(new Date(2026, 3, 30, 0, 0, 0, 0).toISOString())
    expect(range.to.toISOString()).toBe(new Date(2026, 3, 30, 23, 59, 59, 999).toISOString())
  })

  it('resolves last week from Monday through Sunday in local time', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 3, 30, 15, 30, 0))

    const range = resolveDateRange({ preset: 'last_week' })

    expect(range.from.toISOString()).toBe(new Date(2026, 3, 20, 0, 0, 0, 0).toISOString())
    expect(range.to.toISOString()).toBe(new Date(2026, 3, 26, 23, 59, 59, 999).toISOString())
  })

  it('resolves custom date strings to local day boundaries', () => {
    const range = resolveDateRange({ preset: 'custom', from: '2026-04-01', to: '2026-04-15' })

    expect(range.from.toISOString()).toBe(new Date(2026, 3, 1, 0, 0, 0, 0).toISOString())
    expect(range.to.toISOString()).toBe(new Date(2026, 3, 15, 23, 59, 59, 999).toISOString())
  })
})
