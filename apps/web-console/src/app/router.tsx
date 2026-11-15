import { createBrowserRouter, Navigate } from 'react-router-dom';
import { BranchLayout } from '@/layouts/BranchLayout';
import { ProtectedRoute } from '@/routes/ProtectedRoute';
import { authRoutes } from '@/features/auth/routes';
import { tellerRoutes } from '@/features/branch-teller/routes';
import { supervisorRoutes } from '@/features/branch-supervisor/routes';
import { auditRoutes } from '@/features/ledger-audit/routes';

export const router = createBrowserRouter([
  // Default root redirects to login
  {
    path: '/',
    element: <Navigate to="/login" replace />,
  },

  // Auth feature routes
  ...authRoutes,

  // Protected Branch Operational Console
  {
    path: '/branch',
    element: <BranchLayout />,
    children: [
      {
        index: true,
        element: <Navigate to="/branch/teller" replace />,
      },
      // Teller Operations Guard
      {
        element: <ProtectedRoute allowedRoles={['TELLER', 'ADMIN']} />,
        children: tellerRoutes,
      },
      // Supervisor Operations Guard
      {
        element: <ProtectedRoute allowedRoles={['SUPERVISOR', 'ADMIN']} />,
        children: supervisorRoutes,
      },
      // Audit & General Ledger Guard
      {
        element: <ProtectedRoute allowedRoles={['SUPERVISOR', 'ADMIN', 'TELLER']} />,
        children: auditRoutes,
      },
    ],
  },

  // Catch-all fallback
  {
    path: '*',
    element: <Navigate to="/login" replace />,
  },
]);
