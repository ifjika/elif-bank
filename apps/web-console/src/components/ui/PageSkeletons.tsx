import React from 'react';
import { Skeleton } from './skeleton';
import { ShadcnCard, CardHeader, CardContent } from './Card';

/**
 * Matches TellerDashboardPage:
 * Header + CTA Button | 3 KPI Cards | Recent Counter Mutations Table (6 cols)
 */
export const TellerDashboardSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="space-y-2">
          <Skeleton className="h-7 w-44" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <Skeleton className="h-9 w-48" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {[1, 2, 3].map((idx) => (
          <ShadcnCard key={idx}>
            <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
              <Skeleton className="h-3.5 w-36" />
              <Skeleton className="h-4 w-4 rounded-full" />
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-8 w-44" />
              <Skeleton className="h-3 w-32" />
            </CardContent>
          </ShadcnCard>
        ))}
      </div>

      <ShadcnCard>
        <CardHeader className="border-b border-border/60 space-y-2">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-3 w-56" />
        </CardHeader>
        <CardContent className="p-4 space-y-3">
          {[1, 2].map((row) => (
            <div key={row} className="grid grid-cols-6 gap-4 py-2 border-b border-border/40 last:border-0">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-5 w-20 rounded-full" />
              <Skeleton className="h-4 w-16 ml-auto" />
            </div>
          ))}
        </CardContent>
      </ShadcnCard>
    </div>
  );
};

/**
 * Matches CashTransactionPage:
 * Centered max-w-2xl | Header | Form Card (2 toggles, 2 inputs, textarea, submit)
 */
export const CashTransactionSkeleton: React.FC = () => {
  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="space-y-2">
        <Skeleton className="h-7 w-72" />
        <Skeleton className="h-3.5 w-56" />
      </div>

      <ShadcnCard className="p-5 space-y-5">
        <div className="space-y-2">
          <Skeleton className="h-3.5 w-36" />
          <div className="grid grid-cols-2 gap-3">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
          </div>
        </div>

        <div className="space-y-2">
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-9 w-full" />
        </div>

        <div className="space-y-2">
          <div className="flex justify-between">
            <Skeleton className="h-3.5 w-44" />
            <Skeleton className="h-5 w-40 rounded-full" />
          </div>
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-3 w-36" />
        </div>

        <div className="space-y-2">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-16 w-full" />
        </div>

        <Skeleton className="h-9 w-full" />
      </ShadcnCard>
    </div>
  );
};

/**
 * Matches SupervisorDashboardPage:
 * Header + Amber Queue Button | 3 KPI Cards (No table below)
 */
export const SupervisorDashboardSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="space-y-2">
          <Skeleton className="h-7 w-64" />
          <Skeleton className="h-3.5 w-72" />
        </div>
        <Skeleton className="h-9 w-48" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {[1, 2, 3].map((idx) => (
          <ShadcnCard key={idx}>
            <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
              <Skeleton className="h-3.5 w-36" />
              <Skeleton className="h-4 w-4 rounded-full" />
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-8 w-28" />
              <Skeleton className="h-5 w-32 rounded-full" />
            </CardContent>
          </ShadcnCard>
        ))}
      </div>
    </div>
  );
};

/**
 * Matches ApprovalQueuePage:
 * Header (no button) | Full-width 7-column Maker-Checker Table
 */
export const ApprovalQueueSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Skeleton className="h-7 w-80" />
        <Skeleton className="h-3.5 w-96 max-w-full" />
      </div>

      <ShadcnCard className="p-4 space-y-3">
        <div className="grid grid-cols-7 gap-4 pb-3 border-b border-border/60">
          {[1, 2, 3, 4, 5, 6, 7].map((col) => (
            <Skeleton key={col} className="h-3.5 w-full" />
          ))}
        </div>
        {[1, 2, 3].map((row) => (
          <div key={row} className="grid grid-cols-7 gap-4 py-2.5 items-center border-b border-border/40 last:border-0">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-5 w-24 rounded-full" />
            <Skeleton className="h-8 w-16 ml-auto" />
          </div>
        ))}
      </ShadcnCard>
    </div>
  );
};

/**
 * Matches AuditLedgerPage:
 * Header + Integrity Badge | Full-width 8-column Ledger Entries Table
 */
export const AuditLedgerSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-2">
          <Skeleton className="h-7 w-96 max-w-full" />
          <Skeleton className="h-3.5 w-72" />
        </div>
        <Skeleton className="h-6 w-48 rounded-full" />
      </div>

      <ShadcnCard className="p-4 space-y-3">
        <div className="grid grid-cols-8 gap-3 pb-3 border-b border-border/60">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((col) => (
            <Skeleton key={col} className="h-3.5 w-full" />
          ))}
        </div>
        {[1, 2, 3, 4].map((row) => (
          <div key={row} className="grid grid-cols-8 gap-3 py-2.5 items-center border-b border-border/40 last:border-0">
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-5 w-10 rounded-full" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        ))}
      </ShadcnCard>
    </div>
  );
};

export const SessionSkeleton: React.FC = () => {
  return (
    <div className="min-h-screen bg-[#050807] flex items-center justify-center p-6">
      <div className="w-full max-w-md space-y-4">
        <div className="flex items-center gap-3 justify-center mb-6">
          <Skeleton className="h-10 w-10 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
        <ShadcnCard className="p-6 space-y-4">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </ShadcnCard>
      </div>
    </div>
  );
};
