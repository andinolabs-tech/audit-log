import QueryPage from './pages/QueryPage';

interface AuditLogPageProps {
  companyId?: string;
  profileId?: string;
}

export default function AuditLogPage(_props: AuditLogPageProps) {
  return <QueryPage />;
}
