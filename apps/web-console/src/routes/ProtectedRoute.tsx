import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { UserRole } from '@/types/auth';

interface ProtectedRouteProps {
  allowedRoles?: UserRole[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ allowedRoles }) => {
  const token = localStorage.getItem('access_token');
  const userJson = localStorage.getItem('user_info');

  if (!token || !userJson) {
    return <Navigate to="/login" replace />;
  }

  try {
    const user = JSON.parse(userJson);
    if (allowedRoles && allowedRoles.length > 0 && !allowedRoles.includes(user.role)) {
      return (
        <div className="flex flex-col items-center justify-center min-h-[60vh] text-center p-6">
          <h2 className="text-2xl font-bold text-slate-800">403 - Forbidden</h2>
          <p className="text-slate-600 mt-2">
            Your role (<span className="font-semibold">{user.role}</span>) does not have permission to access this module.
          </p>
        </div>
      );
    }
  } catch {
    localStorage.removeItem('access_token');
    localStorage.removeItem('user_info');
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
};
