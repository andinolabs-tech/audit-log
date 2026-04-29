import { useEffect, useState } from 'react'
import type { AuditEvent, NamespacesResponse, QueryEventsResponse } from '../types/event'

export default function QueryPage() {
  const [namespaces, setNamespaces] = useState<string[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [pageSize, setPageSize] = useState(20)
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [nextPageToken, setNextPageToken] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)

  useEffect(() => {
    fetch('/api/namespaces')
      .then((response) => {
        if (!response.ok) throw new Error('Unable to load namespaces')
        return response.json() as Promise<NamespacesResponse>
      })
      .then((data) => setNamespaces(data.namespaces))
      .catch(() => setNamespaces([]))
  }, [])

  const buildUrl = (pageToken?: string) => {
    const params = new URLSearchParams()
    selected.forEach((namespace) => params.append('namespace', namespace))
    params.set('page_size', String(pageSize))
    if (pageToken) params.set('page_token', pageToken)
    return `/api/events?${params.toString()}`
  }

  const search = async (append = false, pageToken?: string) => {
    setLoading(true)
    setError('')
    try {
      const response = await fetch(buildUrl(pageToken))
      if (!response.ok) {
        const body = (await response.json()) as { error?: string }
        throw new Error(body.error ?? 'Request failed')
      }
      const data = (await response.json()) as QueryEventsResponse
      setEvents((previous) => (append ? [...previous, ...data.events] : data.events))
      setNextPageToken(data.next_page_token)
      setSearched(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  const toggleNamespace = (namespace: string) => {
    setSelected((previous) =>
      previous.includes(namespace)
        ? previous.filter((item) => item !== namespace)
        : [...previous, namespace],
    )
  }

  return (
    <main className="min-h-dvh bg-slate-50 px-4 py-6 text-slate-900 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mb-6">
          <p className="text-sm font-medium uppercase tracking-wide text-blue-700">Audit Log</p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight">Query events</h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-600">
            Filter audit events by namespace and inspect the latest recorded activity.
          </p>
        </div>

        <section className="mb-6 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="grid gap-4 lg:grid-cols-[1fr_auto_auto] lg:items-end">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-700">Namespaces</label>
              {namespaces.length === 0 ? (
                <p className="rounded-lg border border-dashed border-slate-300 px-3 py-2 text-sm text-slate-500">
                  No namespaces available yet. Searches will include all namespaces.
                </p>
              ) : (
                <div className="flex max-h-36 flex-wrap gap-2 overflow-y-auto rounded-lg border border-slate-300 p-2">
                  {namespaces.map((namespace) => (
                    <label
                      key={namespace}
                      className="flex min-h-11 cursor-pointer items-center gap-2 rounded-md px-2 text-sm hover:bg-slate-50"
                    >
                      <input
                        type="checkbox"
                        checked={selected.includes(namespace)}
                        onChange={() => toggleNamespace(namespace)}
                        className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-600"
                      />
                      <span>{namespace}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div>
              <label htmlFor="page-size" className="mb-2 block text-sm font-medium text-slate-700">
                Page size
              </label>
              <input
                id="page-size"
                type="number"
                min={1}
                max={500}
                value={pageSize}
                onChange={(event) => setPageSize(Number(event.target.value))}
                className="min-h-11 w-28 rounded-lg border border-slate-300 px-3 text-sm focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
              />
            </div>

            <button
              type="button"
              onClick={() => search(false)}
              disabled={loading}
              className="min-h-11 rounded-lg bg-blue-600 px-5 text-sm font-semibold text-white transition hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? 'Loading...' : 'Search'}
            </button>
          </div>
        </section>

        {error && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {searched && events.length === 0 && !loading && (
          <div className="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-slate-500">
            No events found.
          </div>
        )}

        {events.length > 0 && (
          <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200 text-sm">
                <thead className="bg-slate-100">
                  <tr>
                    {['Timestamp', 'Namespace', 'Action', 'Actor', 'Entity', 'Outcome'].map(
                      (heading) => (
                        <th
                          key={heading}
                          scope="col"
                          className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600"
                        >
                          {heading}
                        </th>
                      ),
                    )}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {events.map((event) => (
                    <tr key={event.id} className="hover:bg-slate-50">
                      <td className="whitespace-nowrap px-4 py-3 text-xs text-slate-600">
                        {new Date(event.timestamp).toLocaleString()}
                      </td>
                      <td className="px-4 py-3 font-medium text-slate-900">{event.namespace}</td>
                      <td className="px-4 py-3">
                        <span className="inline-flex rounded-full bg-blue-100 px-2 py-1 text-xs font-semibold text-blue-800">
                          {event.action}
                        </span>
                      </td>
                      <td className="max-w-40 truncate px-4 py-3 font-mono text-xs text-slate-600">
                        {event.actor_id}
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-600">
                        {event.entity_type}/{event.entity_id}
                      </td>
                      <td className="px-4 py-3">
                        <span className={outcomeClassName(event.outcome)}>{event.outcome}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {nextPageToken && (
              <div className="border-t border-slate-200 px-4 py-3 text-center">
                <button
                  type="button"
                  onClick={() => search(true, nextPageToken)}
                  disabled={loading}
                  className="min-h-11 rounded-lg px-4 text-sm font-semibold text-blue-700 hover:bg-blue-50 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {loading ? 'Loading...' : 'Load more'}
                </button>
              </div>
            )}
          </section>
        )}
      </div>
    </main>
  )
}

function outcomeClassName(outcome: string) {
  const base = 'inline-flex rounded-full px-2 py-1 text-xs font-semibold'
  if (outcome === 'SUCCESS') return `${base} bg-green-100 text-green-800`
  if (outcome === 'FAILURE') return `${base} bg-red-100 text-red-800`
  return `${base} bg-amber-100 text-amber-800`
}
