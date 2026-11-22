import React from 'react';
import { Outlet } from 'react-router-dom';

export const AuthLayout: React.FC = () => {
  return (
    <div className="min-h-screen bg-[#050807] flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative overflow-hidden">
      {/* Subtle Matrix/Trading terminal radial accent */}
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top,rgba(0,230,118,0.12),transparent_60%)]" />

      <div className="relative text-center sm:mx-auto sm:w-full sm:max-w-md">
        <div className="mx-auto w-16 h-16 rounded-xl bg-[#050807] p-1 border border-emerald-500/40 shadow-[0_0_24px_rgba(0,230,118,0.2)] flex items-center justify-center mb-4">
          <img src="/logo.jpg" alt="Elif Bank Logo" className="object-contain w-full h-full rounded-lg" />
        </div>
        <h1 className="text-3xl font-extrabold tracking-widest text-white">
          ELIF <span className="text-[#00E676]">BANK</span>
        </h1>
        <p className="mt-1.5 text-xs font-mono uppercase tracking-widest text-emerald-400/70">
          Core Banking System
        </p>
      </div>
      <div className="relative px-4 mt-8 sm:mx-auto sm:w-full sm:max-w-md sm:px-0">
        <Outlet />
      </div>
    </div>
  );
};
