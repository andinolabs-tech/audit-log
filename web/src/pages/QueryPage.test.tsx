import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import QueryPage from './QueryPage'

describe('QueryPage namespace filter', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps the namespace menu open after selecting an option and closes it on outside click', async () => {
    const user = userEvent.setup()
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ namespaces: ['local'] }),
    } as Response)

    render(<QueryPage />)

    await user.click(await screen.findByRole('button', { name: /all namespaces/i }))
    await user.click(screen.getByLabelText('local'))

    expect(screen.getByLabelText('local')).toBeTruthy()

    await user.click(document.body)

    await waitFor(() => {
      expect(screen.queryByLabelText('local')).toBeNull()
    })
  })
})

describe('QueryPage event details', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('expands details below a clicked event row', async () => {
    const user = userEvent.setup()
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.startsWith('/api/namespaces')) {
        return {
          ok: true,
          json: async () => ({ namespaces: ['local'] }),
        } as Response
      }

      return {
        ok: true,
        json: async () => ({
          events: [
            {
              id: '019d5136-8409-735e-acf7-3ddfc238419d',
              timestamp: '2026-04-29T15:19:23.000Z',
              namespace: 'local',
              action: 'CREATED',
              actor_id: 'system',
              entity_type: 'Order',
              entity_id: 'ord-456',
              outcome: 'SUCCESS',
            },
          ],
          next_page_token: '',
        }),
      } as Response
    })

    render(<QueryPage />)

    const searchButtons = screen.getAllByRole('button', { name: 'Search' })
    await user.click(searchButtons[searchButtons.length - 1])
    const eventRow = await screen.findByRole('row', { name: /created/i })
    await user.click(eventRow)

    expect(await screen.findByText('Event details')).toBeTruthy()
    expect(screen.getByText('Event ID')).toBeTruthy()
    expect(screen.getByText('019d5136-8409-735e-acf7-3ddfc238419d')).toBeTruthy()
    expect(screen.getByText('Service')).toBeTruthy()
  })
})
