import React from 'react';
import { Link } from 'react-router-dom';
import { ShieldCheck, Clock, LockKeyhole } from 'lucide-react';
import {
  ShadcnCard,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';

export const SupervisorDashboardPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Branch Supervisor Console</h1>
          <p className="text-xs font-mono text-zinc-400">
            Dual Authorization Control &amp; Four-Eyes Governance
          </p>
        </div>
        <Button asChild className="bg-amber-500 text-zinc-950 hover:bg-amber-400 font-semibold">
          <Link to="/branch/supervisor/approvals">Review Pending Queue (1)</Link>
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <ShadcnCard className="border-amber-500/25">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Pending Approvals
            </CardTitle>
            <Clock className="w-4 h-4 text-amber-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-amber-400">1 Item</div>
            <Badge variant="warning" className="mt-2">
              Action Required
            </Badge>
          </CardContent>
        </ShadcnCard>

        <ShadcnCard className="border-emerald-500/20">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Maker-Checker Compliance
            </CardTitle>
            <ShieldCheck className="w-4 h-4 text-[#00E676]" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-[#00E676]">100.0%</div>
            <CardDescription className="mt-1 font-mono">
              Zero self-approvals detected
            </CardDescription>
          </CardContent>
        </ShadcnCard>

        <ShadcnCard>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Branch Audit Alerts
            </CardTitle>
            <LockKeyhole className="w-4 h-4 text-[#00E676]" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-white">0 Anomalies</div>
            <Badge variant="success" className="mt-2">
              SHA-256 Chain Intact
            </Badge>
          </CardContent>
        </ShadcnCard>
      </div>
    </div>
  );
};
