const manifest = {
  name: 'mfe_audit_log',
  sections: [
    {
      key: 'auditLogs',
      title: 'sidebar.sections.auditLogs',
      items: [
        {
          icon: 'History',
          label: 'sidebar.navigation.auditLogs',
          href: '/dashboard/audit-logs',
        },
      ],
    },
  ],
  routes: [
    { path: '/dashboard/audit-logs', module: './AuditLogPage' },
  ],
};

export default manifest;
