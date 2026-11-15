import { RouteObject } from 'react-router-dom';
import { SupervisorDashboardPage } from './pages/SupervisorDashboardPage';
import { ApprovalQueuePage } from './pages/ApprovalQueuePage';

export const supervisorRoutes: RouteObject[] = [
  {
    path: 'supervisor',
    children: [
      {
        index: true,
        element: <SupervisorDashboardPage />,
      },
      {
        path: 'approvals',
        element: <ApprovalQueuePage />,
      },
    ],
  },
];
