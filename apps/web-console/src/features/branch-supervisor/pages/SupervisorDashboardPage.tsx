import React from 'react';
import { Card } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Link } from 'react-router-dom';

export const SupervisorDashboardPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Branch Supervisor Console</h1>
          <p className="text-xs text-slate-500">Dual Authorization Control & Four-Eyes Governance</p>
        </div>
        <Link
          to="/branch/supervisor/approvals"
          className="bg-amber-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-amber-700 transition-colors"
        >
          Review Pending Queue (1)
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <Card title="Pending Approvals" subtitle="Awaiting Checker authorization">
          <div className="text-2xl font-bold font-mono text-amber-600 mt-2">1 Item</div>
          <Badge variant="warning" className="mt-2">
            Action Required
          </Badge>
        </Card>

        <Card title="Maker-Checker Compliance" subtitle="Four-Eyes verification rate">
          <div className="text-2xl font-bold font-mono text-emerald-600 mt-2">100.0%</div>
          <p className="text-xs text-slate-400 mt-1">Zero self-approvals detected</p>
        </Card>

        <Card title="Branch Audit Alerts" subtitle="SHA-256 chain integrity">
          <div className="text-2xl font-bold font-mono text-slate-900 mt-2">0 Anomalies</div>
          <Badge variant="success" className="mt-2">
            Chain Intact
          </Badge>
        </Card>
      </div>
    </div>
  );
};
