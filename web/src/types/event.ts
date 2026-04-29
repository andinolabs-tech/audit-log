export interface AuditEvent {
  id: string
  tenant_id: string
  namespace: string
  actor_id: string
  actor_type: string
  entity_type: string
  entity_id: string
  action: string
  outcome: string
  service_name: string
  source_ip?: string
  session_id?: string
  correlation_id?: string
  trace_id?: string
  timestamp: string
  occurred_at?: string
  compensates_id?: string
  reason?: string
  tags: string[]
  before?: Record<string, unknown>
  after?: Record<string, unknown>
  diff?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface QueryEventsResponse {
  events: AuditEvent[]
  next_page_token: string
}

export interface NamespacesResponse {
  namespaces: string[]
}
