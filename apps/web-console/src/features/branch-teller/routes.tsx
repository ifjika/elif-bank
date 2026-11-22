import { lazy, Suspense } from 'react';
import { RouteObject } from 'react-router-dom';
import {
  TellerDashboardSkeleton,
  CashTransactionSkeleton,
} from '@/components/ui/PageSkeletons';

const TellerDashboardPage = lazy(() =>
  import('./pages/TellerDashboardPage').then((m) => ({ default: m.TellerDashboardPage }))
);
const CashTransactionPage = lazy(() =>
  import('./pages/CashTransactionPage').then((m) => ({ default: m.CashTransactionPage }))
);

export const tellerRoutes: RouteObject[] = [
  {
    path: 'teller',
    children: [
      {
        index: true,
        element: (
          <Suspense fallback={<TellerDashboardSkeleton />}>
            <TellerDashboardPage />
          </Suspense>
        ),
      },
      {
        path: 'cash',
        element: (
          <Suspense fallback={<CashTransactionSkeleton />}>
            <CashTransactionPage />
          </Suspense>
        ),
      },
    ],
  },
];
