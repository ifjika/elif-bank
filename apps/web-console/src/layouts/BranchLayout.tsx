import React from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { Badge } from '@/components/ui/Badge';
import { useAuth } from '@/features/auth/context/AuthContext';

export const BranchLayout: React.FC = () => {
  const navigate = useNavigate();
  const { user, logout } = useAuth();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-brand-surface flex flex-col">
      {/* Top Navigation Bar */}
      <header className="bg-brand-primary text-white border-b border-slate-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-6">
            <span className="text-xl font-bold tracking-wider text-blue-400">ELIF BANK</span>
            <span className="text-xs uppercase tracking-widest px-2 py-1 bg-slate-800 rounded text-slate-300 font-mono">
              Branch Console
            </span>
          </div>

          <div className="flex items-center space-x-4">
            <div className="text-right">
              <div className="text-sm font-medium">{user?.full_name || user?.username || 'Operator'}</div>
              <Badge variant={user?.role === 'SUPERVISOR' ? 'warning' : 'neutral'} className="text-[10px]">
                {user?.role || 'OPERATOR'}
              </Badge>
            </div>
            <button
              onClick={handleLogout}
              className="text-xs text-slate-400 hover:text-white bg-slate-800 hover:bg-slate-700 px-3 py-1.5 rounded transition-colors"
            >
              Sign Out
            </button>
          </div>
        </div>

        {/* Sub-nav Links */}
        <div className="bg-slate-800/60 border-t border-slate-700/50">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex space-x-6">
            {user?.role === 'TELLER' && (
              <>
                <NavLink
                  to="/branch/teller"
                  end
                  className={({ isActive }) =>
                    `py-3 text-xs font-medium border-b-2 transition-colors ${
                      isActive
                        ? 'border-brand-accent text-white font-semibold'
                        : 'border-transparent text-slate-400 hover:text-slate-200'
                    }`
                  }
                >
                  Teller Dashboard
                </NavLink>
                <NavLink
                  to="/branch/teller/cash"
                  className={({ isActive }) =>
                    `py-3 text-xs font-medium border-b-2 transition-colors ${
                      isActive
                        ? 'border-brand-accent text-white font-semibold'
                        : 'border-transparent text-slate-400 hover:text-slate-200'
                    }`
                  }
                >
                  Over-the-Counter Cash
                </NavLink>
              </>
            )}

            {user?.role === 'SUPERVISOR' && (
              <>
                <NavLink
                  to="/branch/supervisor"
                  end
                  className={({ isActive }) =>
                    `py-3 text-xs font-medium border-b-2 transition-colors ${
                      isActive
                        ? 'border-brand-accent text-white font-semibold'
                        : 'border-transparent text-slate-400 hover:text-slate-200'
                    }`
                  }
                >
                  Supervisor Overview
                </NavLink>
                <NavLink
                  to="/branch/supervisor/approvals"
                  className={({ isActive }) =>
                    `py-3 text-xs font-medium border-b-2 transition-colors ${
                      isActive
                        ? 'border-brand-accent text-white font-semibold'
                        : 'border-transparent text-slate-400 hover:text-slate-200'
                    }`
                  }
                >
                  Maker-Checker Approvals
                </NavLink>
              </>
            )}

            <NavLink
              to="/branch/audit"
              className={({ isActive }) =>
                `py-3 text-xs font-medium border-b-2 transition-colors ${
                  isActive
                    ? 'border-brand-accent text-white font-semibold'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`
              }
            >
              Ledger Audit Trail
            </NavLink>
          </div>
        </div>
      </header>

      {/* Main Content Viewport */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 lg:p-8">
        <Outlet />
      </main>
    </div>
  );
};
