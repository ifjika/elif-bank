import { RouteObject } from 'react-router-dom';
import { LoginPage } from './pages/LoginPage';
import { AuthLayout } from '@/layouts/AuthLayout';

export const authRoutes: RouteObject[] = [
  {
    element: <AuthLayout />,
    children: [
      {
        path: '/login',
        element: <LoginPage />,
      },
    ],
  },
];
