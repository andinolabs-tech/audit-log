import { describe, expect, it } from 'vitest'

import { parseSearchQuery } from './searchQuery'

describe('parseSearchQuery', () => {
  it('parses valid key:value pairs into exact-match query params', () => {
    const result = parseSearchQuery('actor_id:user-1 action:LOGIN outcome:SUCCESS')

    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.params).toEqual([
      ['actor_id', 'user-1'],
      ['action', 'LOGIN'],
      ['outcome', 'SUCCESS'],
    ])
  })

  it('rejects unknown keys before building search params', () => {
    const result = parseSearchQuery('actor:user-1 action:LOGIN')

    expect(result.ok).toBe(false)
    if (result.ok) return
    expect(result.message).toContain('actor')
  })
})
