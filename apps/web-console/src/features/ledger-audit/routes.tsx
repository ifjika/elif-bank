import { lazy, Suspense } from 'react';
import { RouteObject } from 'react-router-dom';
import { AuditLedgerSkeleton } from '@/components/ui/PageSkeletons';

const AuditLedgerPage = lazy(() =>
  import('./pages/AuditLedgerPage').then((m) => ({ default: m.AuditLedgerPage }))
);

export const auditRoutes: RouteObject[] = [
  {
    path: 'audit',
    element: (
      <Suspense fallback={<AuditLedgerSkeleton />}>
        <AuditLedgerPage />
      </Suspense>
    ),
  },
];
