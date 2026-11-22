import {
  ApprovalQueueSkeleton,
  AuditLedgerSkeleton,
  CashTransactionSkeleton,
  SessionSkeleton,
  SupervisorDashboardSkeleton,
  TellerDashboardSkeleton,
} from '@/components/ui/PageSkeletons';
import { useAuth } from '@/features/auth/context/AuthContext';
import { UserRole } from '@/types/auth';
import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';

interface ProtectedRouteProps {
  allowedRoles?: UserRole[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ allowedRoles }) => {
  const { user, isAuthenticated, isInitializing } = useAuth();
  const { pathname } = useLocation();

  if (isInitializing) {
    if (pathname.startsWith('/branch/teller/cash')) {
      return <CashTransactionSkeleton />;
    }
    if (pathname.startsWith('/branch/teller')) {
      return <TellerDashboardSkeleton />;
    }
    if (pathname.startsWith('/branch/supervisor/approvals')) {
      return <ApprovalQueueSkeleton />;
    }
    if (pathname.startsWith('/branch/supervisor')) {
      return <SupervisorDashboardSkeleton />;
    }
    if (pathname.startsWith('/branch/audit')) {
      return <AuditLedgerSkeleton />;
    }
    return <SessionSkeleton />;
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" replace />;
  }

  if (allowedRoles && allowedRoles.length > 0 && !allowedRoles.includes(user.role)) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] text-center p-6">
        <h2 className="text-2xl font-bold text-white">403 - Forbidden</h2>
        <p className="text-zinc-400 mt-2 text-sm font-mono">
          Your role (<span className="font-semibold text-[#00E676]">{user.role}</span>) does not have permission to access this module.
        </p>
      </div>
    );
  }

  return <Outlet />;
};
