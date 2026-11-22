import React, { useState } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Banknote,
  ShieldCheck,
  FileCheck2,
  LogOut,
  Menu,
  X,
  Activity,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Separator } from '@/components/ui/separator';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Skeleton } from '@/components/ui/skeleton';
import { useAuth } from '@/features/auth/context/AuthContext';

export const BranchLayout: React.FC = () => {
  const navigate = useNavigate();
  const { user, isInitializing, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const initials = (user?.full_name || user?.username || 'OP')
    .split(' ')
    .map((n) => n[0])
    .join('')
    .substring(0, 2)
    .toUpperCase();

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-3 px-3 py-2.5 rounded-md text-xs font-mono tracking-wide transition-all ${
      isActive
        ? 'bg-emerald-500/15 text-[#00E676] font-semibold border border-emerald-500/30 shadow-[0_0_12px_rgba(0,230,118,0.08)]'
        : 'text-zinc-400 hover:text-zinc-100 hover:bg-emerald-950/40 border border-transparent'
    }`;

  return (
    <div className="min-h-screen bg-[#050807] text-zinc-100 flex">
      {/* Mobile Sidebar Backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/70 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Left shadcn/ui Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-64 bg-[#080E0B] border-r border-emerald-500/20 flex flex-col transition-transform duration-200 lg:static lg:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        {/* Brand Header */}
        <div className="h-16 px-5 flex items-center justify-between border-b border-emerald-500/15">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-[#050807] border border-emerald-500/40 p-0.5 flex items-center justify-center shadow-[0_0_10px_rgba(0,230,118,0.2)]">
              <img src="/logo.jpg" alt="Elif Bank" className="w-full h-full object-contain rounded" />
            </div>
            <div>
              <div className="text-sm font-bold tracking-wider text-white">
                ELIF <span className="text-[#00E676]">BANK</span>
              </div>
              <div className="text-[10px] font-mono text-emerald-400/70 tracking-widest">
                CORE TERMINAL
              </div>
            </div>
          </div>
          <button
            onClick={() => setSidebarOpen(false)}
            className="lg:hidden text-zinc-400 hover:text-white"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Navigation Sections */}
        <nav className="flex-1 px-3 py-5 space-y-6 overflow-y-auto">
          {isInitializing ? (
            <div className="space-y-4 px-2">
              <Skeleton className="h-3 w-28" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <div className="pt-3 space-y-3">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-9 w-full" />
              </div>
            </div>
          ) : (
            <>
              {(user?.role === 'TELLER' || user?.role === 'ADMIN') && (
                <div className="space-y-1.5">
                  <div className="px-3 text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-500">
                    Teller Operations
                  </div>
                  <NavLink to="/branch/teller" end className={navLinkClass} onClick={() => setSidebarOpen(false)}>
                    <LayoutDashboard className="w-4 h-4" />
                    Teller Dashboard
                  </NavLink>
                  <NavLink to="/branch/teller/cash" className={navLinkClass} onClick={() => setSidebarOpen(false)}>
                    <Banknote className="w-4 h-4" />
                    Cash Counter
                  </NavLink>
                </div>
              )}

              {(user?.role === 'SUPERVISOR' || user?.role === 'ADMIN') && (
                <div className="space-y-1.5">
                  <div className="px-3 text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-500">
                    Supervisor Governance
                  </div>
                  <NavLink to="/branch/supervisor" end className={navLinkClass} onClick={() => setSidebarOpen(false)}>
                    <LayoutDashboard className="w-4 h-4" />
                    Supervisor Overview
                  </NavLink>
                  <NavLink
                    to="/branch/supervisor/approvals"
                    className={navLinkClass}
                    onClick={() => setSidebarOpen(false)}
                  >
                    <ShieldCheck className="w-4 h-4" />
                    Maker-Checker Queue
                  </NavLink>
                </div>
              )}

              <div className="space-y-1.5">
                <div className="px-3 text-[10px] font-mono font-semibold uppercase tracking-widest text-zinc-500">
                  Audit &amp; Integrity
                </div>
                <NavLink to="/branch/audit" className={navLinkClass} onClick={() => setSidebarOpen(false)}>
                  <FileCheck2 className="w-4 h-4" />
                  General Ledger Audit
                </NavLink>
              </div>
            </>
          )}
        </nav>

        {/* Bottom Operator Card */}
        <div className="p-3 border-t border-emerald-500/15 space-y-3">
          {isInitializing ? (
            <div className="flex items-center gap-3 px-2 py-1.5 rounded-lg bg-[#050807] border border-emerald-500/15">
              <Skeleton className="h-9 w-9 rounded-full" />
              <div className="space-y-1.5 flex-1">
                <Skeleton className="h-3.5 w-24" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-3 px-2 py-1.5 rounded-lg bg-[#050807] border border-emerald-500/15">
              <Avatar>
                <AvatarFallback>{initials}</AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <div className="text-xs font-semibold text-white truncate">
                  {user?.full_name || user?.username}
                </div>
                <div className="text-[10px] font-mono text-zinc-400 truncate">
                  {user?.branch_id || 'branch-hq'}
                </div>
              </div>
            </div>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleLogout}
            disabled={isInitializing}
            className="w-full justify-start gap-2 text-xs font-mono border-emerald-500/20 text-zinc-300 hover:text-rose-400 hover:border-rose-500/30 hover:bg-rose-500/10"
          >
            <LogOut className="w-3.5 h-3.5" />
            Sign Out
          </Button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top Status Header */}
        <header className="h-16 bg-[#080E0B]/80 backdrop-blur-md border-b border-emerald-500/15 px-4 sm:px-6 lg:px-8 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(true)}
              className="lg:hidden p-2 rounded-md text-zinc-400 hover:text-white hover:bg-emerald-950/50"
            >
              <Menu className="w-5 h-5" />
            </button>
            <div className="flex items-center gap-2 text-xs font-mono text-zinc-400">
              <Activity className="w-3.5 h-3.5 text-[#00E676]" />
              <span>SYSTEM:</span>
              <span className="text-[#00E676] font-semibold">TIDB CONNECTED</span>
            </div>
            <Separator orientation="vertical" className="h-4 hidden sm:block" />
            <span className="hidden sm:inline text-xs font-mono text-zinc-500">
              DOUBLE-ENTRY INVARIANT: <span className="text-zinc-300">ΣDR - ΣCR = 0</span>
            </span>
          </div>

          {isInitializing ? (
            <Skeleton className="h-6 w-28 rounded-full" />
          ) : (
            <Badge variant={user?.role === 'SUPERVISOR' ? 'warning' : 'success'}>
              ROLE: {user?.role || 'OPERATOR'}
            </Badge>
          )}
        </header>

        {/* Page Viewport */}
        <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl w-full mx-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
