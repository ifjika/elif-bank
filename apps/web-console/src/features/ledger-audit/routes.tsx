import { RouteObject } from 'react-router-dom';
import { AuditLedgerPage } from './pages/AuditLedgerPage';

export const auditRoutes: RouteObject[] = [
  {
    path: 'audit',
    element: <AuditLedgerPage />,
  },
];
