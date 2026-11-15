import { RouteObject } from 'react-router-dom';
import { TellerDashboardPage } from './pages/TellerDashboardPage';
import { CashTransactionPage } from './pages/CashTransactionPage';

export const tellerRoutes: RouteObject[] = [
  {
    path: 'teller',
    children: [
      {
        index: true,
        element: <TellerDashboardPage />,
      },
      {
        path: 'cash',
        element: <CashTransactionPage />,
      },
    ],
  },
];
