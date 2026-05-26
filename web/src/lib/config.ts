declare global {
  interface Window {
    AUDIT_LOG_PUBLIC_API?: string;
  }
}

export function getAuditLogApiBase(): string {
  const runtime = window.AUDIT_LOG_PUBLIC_API?.trim().replace(/\/+$/, '');
  if (runtime && !runtime.startsWith('__')) return runtime;
  // Fallback: use Vite proxy path (works in standalone and when federated inside the host)
  return '/api/audit-log';
}
