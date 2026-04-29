export const SEARCH_KEYS = [
  'tenant_id',
  'actor_id',
  'actor_type',
  'entity_type',
  'entity_id',
  'action',
  'outcome',
  'service_name',
  'source_ip',
  'session_id',
  'correlation_id',
  'trace_id',
  'namespace',
] as const

type SearchKey = (typeof SEARCH_KEYS)[number]

export type ParsedSearchQuery =
  | { ok: true; params: Array<[SearchKey, string]> }
  | { ok: false; message: string }

const searchKeySet = new Set<string>(SEARCH_KEYS)

export function parseSearchQuery(query: string): ParsedSearchQuery {
  const tokens = query.trim().split(/\s+/).filter(Boolean)
  const params: Array<[SearchKey, string]> = []

  for (const token of tokens) {
    const separator = token.indexOf(':')
    if (separator < 1 || separator === token.length - 1) {
      return {
        ok: false,
        message: `Use key:value pairs only. Valid keys: ${SEARCH_KEYS.join(', ')}`,
      }
    }

    const key = token.slice(0, separator)
    const value = token.slice(separator + 1)
    if (!searchKeySet.has(key)) {
      return {
        ok: false,
        message: `Unknown search key "${key}". Valid keys: ${SEARCH_KEYS.join(', ')}`,
      }
    }

    params.push([key as SearchKey, value])
  }

  return { ok: true, params }
}
