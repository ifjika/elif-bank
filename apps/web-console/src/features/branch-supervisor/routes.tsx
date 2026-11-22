import { lazy, Suspense } from 'react';
import { RouteObject } from 'react-router-dom';
import {
  SupervisorDashboardSkeleton,
  ApprovalQueueSkeleton,
} from '@/components/ui/PageSkeletons';

const SupervisorDashboardPage = lazy(() =>
  import('./pages/SupervisorDashboardPage').then((m) => ({ default: m.SupervisorDashboardPage }))
);
const ApprovalQueuePage = lazy(() =>
  import('./pages/ApprovalQueuePage').then((m) => ({ default: m.ApprovalQueuePage }))
);

export const supervisorRoutes: RouteObject[] = [
  {
    path: 'supervisor',
    children: [
      {
        index: true,
        element: (
          <Suspense fallback={<SupervisorDashboardSkeleton />}>
            <SupervisorDashboardPage />
          </Suspense>
        ),
      },
      {
        path: 'approvals',
        element: (
          <Suspense fallback={<ApprovalQueueSkeleton />}>
            <ApprovalQueuePage />
          </Suspense>
        ),
      },
    ],
  },
];
